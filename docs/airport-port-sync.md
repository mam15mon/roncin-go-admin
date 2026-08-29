# 机场与港口主数据网络同步方案

本方案定义机场（OurAirports）和海运港口（UNECE UN/LOCODE）两条官方数据链路
在新系统中的同步方式。数据来源、同步语义和组织隔离决策以本文为准；实施结果
回填到本文「实施状态」一节。

背景是对旧 `roncin` 项目数据链路的审计结论：旧项目从 OurAirports 每日数据导
入机场、从 UN/LOCODE 年度发布导入全球地点，航司与船司为代码内硬编码种子。
旧实现的「下载/导入分离、按标准码幂等 upsert、保留数据来源」值得沿用；全量
地点导入、中文名 COALESCE 覆盖、SCAC 混入 Carrier 表等问题不迁移。

## 数据源决策

| 主数据 | 数据源 | 决策 | 理由 |
|---|---|---|---|
| 机场 | OurAirports `airports.csv`（每日更新） | 接入 | 公共领域数据，含 IATA/ICAO、英文名、城市、国家、类型 |
| 海运港口 | UNECE UN/LOCODE（每年约两版） | 接入，只导海港 | 官方权威地点库；但它是全球地点库而非港口库，必须按 Function 首位筛选 |
| 航司 | 无开放权威源 | 不接入 | `awb_prefix`（AWB 单号前缀）无公开权威数据，字段本身决定只能人工维护 |
| 船司 | 无开放权威源 | 不接入 | SCAC 权威数据的开放性与授权不如 OurAirports、UN/LOCODE 明确 |

航司和船司继续走 `master_data` 人工维护与既有种子，不伪装成网络同步。

## 架构决策

### D1：只同步到指定组织，不引入全局主数据层

Airport、Port 均为组织级实体（唯一键含 `organization_id`），而网络数据是全球
公共数据。本方案不新建全局共享主数据层，同步命令通过 `-org-code` 指定唯一
目标组织（缺省读 `BOOTSTRAP_ORGANIZATION_CODE` 环境变量，与 bootstrap-admin
的创建参数一致），数据落在该组织下。

当前 MVP 只有初始化组织使用订单主数据，因此第一版只同步指定组织。新增其他
可录单组织前，必须先决定改为全局目录还是对全部业务组织执行同步；不允许让
各组织手工重复维护同一套全球公共数据。

### D2：命令形态沿用 `cmd/sync-regions` 模式

命令交互对照 `server/cmd/sync-regions/main.go`：独立 `main` 包命令，默认预览、
`-apply` 写库。下载、解析和预览属于 `cmd`；组织查询、事务、停用和 upsert
集中在 `internal/data`，通过 Ent 完成，不在新命令中复制手写 SQL。

### D3：同步语义

- **幂等 upsert**：机场按 `(organization_id, iata_code)`，港口按
  `(organization_id, un_locode)` 冲突更新。
- **停用本版缺失**：事务开始时先把目标组织内同 `source` 的记录全部
  `enabled = false`，再按本版数据逐条 upsert 并写入本版计算的 `enabled`。
  「上一版存在、本版缺失」的记录因此保持停用；人工维护记录（`source` 不同）
  不受影响。UN/LOCODE 的改码行（change code `#`）不做专门处理，由该规则处理。
- **人工中文名保护**：`name_zh`、`city_name_zh` 改为可空；网络首次导入不写
  中文字段，后续同步也不覆盖。人工维护接口仍允许补充和修正中文名。
- **英文官方名随源更新**：`name_en`、`city_name_en`、`country_code`、
  `icao_code`（机场）、`transport_modes`（港口）、`enabled` 每次同步按本版
  数据覆盖。
- **来源由系统管理**：人工维护接口不接受客户端指定网络来源；手工新建记录的
  `source` 固定为 `manual`。同步命令只管理其自身来源记录。目标组织存在同标准
  码但来源不同的记录时，预览报告冲突，`-apply` 拒绝写入，不抢占人工数据。

### D4：溯源字段

`Airport`、`Port` 增加 `source_version`、`source_hash` 可空字段（命名与
`administrative_region.source_version` 先例一致）。`source` 取常量
`OURAIRPORTS`、`UNECE_UNLOCODE`（对照 `MCA_DMFW` 风格）。

本方案不建导入批次表：`BackgroundTask` 模型明确不保存业务 payload，导入统计
先以命令行输出与结构化日志为准；每次同步计算原始文件 SHA-256 并写入记录，
使下载日期相同的数据仍可区分。需要可查询的批次历史时（对照旧项目
`UnlocodeImportBatch`）再单独设计。

### D5：数据获取方式

- **OurAirports**：默认从 `https://ourairports.com/data/airports.csv` 下载并
  缓存到本地 `.cache/master-data/`；`-source` 可指定本地文件跳过下载，缓存
  目录不提交 Git。
- **UN/LOCODE**：不自动下载。发布地址随版本变化，要求人工从 UNECE 官网下载
  zip 后通过 `-source` 指定本地路径。版本标识优先从 zip 内文件名前缀解析
  （如 `2025-1 UNLOCODE CodeListPart1.csv` → `2025-1`），解析不出时必须显式
  传 `-release`。
- OurAirports 无显式版本号，`-release` 缺省用下载当日日期（如 `2026-08-23`）。

## 命令设计：cmd/sync-airports

```text
用法：
  go run ./cmd/sync-airports [-apply] [-source <csv路径>] [-release <版本>]
                             [-org-code <组织代码>]

流程：
  读取/下载 CSV → 解析与清洗 → 输出统计（预览）→ [-apply] 单事务写库
```

### 过滤规则

1. `iata_code` 非空且匹配 `^[A-Z]{3}$`，与现有领域校验一致（全量约 7.5 万行，有合法 IATA 码
   的约 6 千余条；无 IATA 码的小机场不导入）。
2. 同一 IATA 或有效 ICAO 码出现多行时记录冲突；预览展示冲突明细，`-apply`
   拒绝写入，不按文件顺序静默保留首行。

### 字段映射

| OurAirports 列 | Airport 字段 | 规则 |
|---|---|---|
| `iata_code` | `iata_code` | 大写化，过滤见上 |
| `gps_code` | `icao_code` | 匹配 `^[A-Z0-9]{4}$` 才写入；空值不写入，非法值计数；ICAO 冲突时禁止 `-apply`，不使用 `local_code` 替代 |
| `name` | `name_en` | 原样写入 |
| `municipality` | `city_name_en` | 可空 |
| `iso_country` | `country_code` | 大写两位 |
| `type` | `enabled` | `closed` → false，其余 → true |
| — | `name_zh` | 网络同步不写入，保留人工值 |
| — | `city_name_zh` | 网络同步不写入，保留人工值 |
| — | `source` | `OURAIRPORTS` |
| — | `source_version` | `-release` 值 |
| — | `source_hash` | 原始 CSV 的 SHA-256 |
| — | `sort_order` | 保持默认 100，同步不修改 |

## 命令设计：cmd/sync-unlocode

```text
用法：
  go run ./cmd/sync-unlocode -source <zip路径> [-apply] [-release <版本>]
                             [-org-code <组织代码>]

流程：
  读取 zip（archive/zip）→ 解析 CodeListPart1-3.csv → 筛选海港 →
  输出统计（预览）→ [-apply] 单事务写库
```

### 解析规则

- zip 内按去版本前缀的文件名匹配 `UNLOCODE CodeListPart1/2/3.csv`
  （官方包条目可能带目录前缀或 `2025-1 ` 版本前缀）。
- CSV 编码兼容 UTF-8 与 cp1252，实现时用官方样本验证重音名称列；`Name w/o
  Diacritics` 列不单独存储（模型无对应字段，记录为已知限制）。
- 列序（官方）：Change Indicator、Country、Location、Name、Name w/o
  Diacritics、Subdivision、Function、Status、Date、IATA、Coordinates、Remarks。

### 行级处理

| 行特征 | 处理 |
|---|---|
| Location 为空且 Name 以 `.` 开头 | 国家标题行，跳过 |
| Change Indicator 为 `=` | 别名行，跳过（当前模型无别名表，已知限制） |
| Change Indicator 为 `+`、`#`、`|` 或空 | 正常候选行；同码时按 `#`、`+`、`|`、空的明确优先级选择新行，同优先级重复则阻止写入 |
| Country 或 Location 为空 | 跳过并计数 |

### 筛选与映射

- **只导入海港**：`Function` 首位（index 0）为 `1` 的行才导入；不重蹈旧项目
  全量 10 万条地点入库、查询时再过滤的路径。
- `un_locode` = Country + Location（5 字符，大写）。
- `name_en` = 官方 Name 列；网络同步不写入或覆盖 `name_zh`。
- `transport_modes` 由 Function 各位置生成，取值约定为
  `SEA`（位置 1）、`RAIL`（位置 2）、`ROAD`（位置 3），与订单线路类型的大写
  风格一致。位置 4（机场）、5（邮政）、6（多式联运）、7（固定设施）、
  `B`（边境口岸）第一版忽略，作为已知限制记录。
- `Status` 为 `XX`（撤销）的行导入但 `enabled = false`，保留可检索历史记录。
- Subdivision、IATA、Coordinates、Remarks、Date 列不导入（模型无对应字段；
  坐标待地图需求出现后加字段重导，数据源幂等）。

### 写入门禁

- 机场必须成功解析表头且至少得到一条有效记录；UN/LOCODE 必须同时找到三个
  CodeList 分片且至少得到一条有效海港记录。
- 除 UN/LOCODE 官方变更行覆盖旧行外，标准码同优先级重复、与目标组织异来源
  同码、字段超出数据库长度或国家码非法均列为致命冲突；预览输出统计和样例，
  `-apply` 直接失败。
- 所有解析、校验和冲突检查在开启写事务前完成，禁止用残缺数据执行全量停用。

## 已知限制

1. 机场与港口无权威中文名，网络导入后中文名为空；旧项目自维
   护的约 415 条机场中文映射可作为一次性人工种子单独迁入（只写空值），不参
   与后续同步。
2. UN/LOCODE 别名行被跳过、去变音名列未存储、Function 位置 4-7 与 `B` 未映
   射、坐标未存储。
3. 当前 MVP 只同步单一目标组织；启用其他录单组织前必须完成共享策略决策。
4. 无导入批次历史表，统计仅在命令输出与日志中。
5. UN/LOCODE 需人工下载新包；OurAirports 每日更新但同步频率由人工触发决定。

## 验证

- 解析与清洗逻辑（CSV 解析、cp1252 样本、Function 映射、标题/别名行、
  标准码冲突拒绝）以表驱动单测覆盖，样本数据内嵌测试文件，不依赖网络。
- `go -C server test ./...`、`go -C server vet ./...`。
- 真实数据验收：预览统计数量级核对（机场约 6 千、海港约 1 万）；`-apply` 后
  SQL 抽查新增/更新/停用计数、人工中文名未被覆盖、重复执行结果幂等。

## 实施步骤

1. 调整机场/港口中文字段与来源所有权，增加 `source_version`、`source_hash`，
   更新 Proto、Ent、前端生成客户端和迁移后提交。
2. 在 `internal/data` 增加同步仓储，实现 `cmd/sync-airports` 与单测，真实数据
   预览验收后提交。
3. 实现 `cmd/sync-unlocode` 与单测，真实数据预览验收后提交。
4. 增加根目录 `pnpm` 同步入口并完成全量验证。
5. （可选）迁移旧项目机场中文映射作为一次性种子脚本。
6. 每步完成后回填本文「实施状态」。

## 实施状态

已于 2026-08-23 完成：

- Airport、Port 已支持可空中文名、系统来源、来源版本和 SHA-256。
- `cmd/sync-airports` 已通过 OurAirports 真实数据验收并向 HQ 写入 9,054 条机场。
- `cmd/sync-unlocode` 已通过 UNECE 2025-1 真实数据验收并向 HQ 写入 17,524 条
  海港；重复执行新增为 0，验证幂等。
- 根目录提供 `pnpm run sync:airports`、`pnpm run sync:unlocode` 正式写入入口。

本方案对应 `docs/migration-matrix.md` D11/D14 中 AirportDictionary、Unlocode 系
列模型的外部文件解析要求。后台定时执行、导入批次查询和多组织共享策略仍按「已知
限制」处理，不在本次实现中伪装完成。
