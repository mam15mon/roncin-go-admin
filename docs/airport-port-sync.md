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

其他组织需要机场/港口时走人工维护或后续单独决策（例如按需复制或全局层），
不在本方案范围内。

### D2：命令形态沿用 `cmd/sync-regions` 模式

对照 `server/cmd/sync-regions/main.go`：独立 `main` 包命令，默认预览、
`-apply` 写库，单事务内「先停用同源记录、再逐条 upsert」。下载与导入合为
一体，预览模式即 dry-run。

与 `AGENTS.md` 手写 SQL 约束的关系：同步命令属于 `cmd` 一次性工具，不属于
业务代码路径；批量幂等 upsert 使用 `ON CONFLICT` 是 `sync-regions` 已确立的
先例，本方案沿用，不在 `internal/data` 中新增仓储方法。

### D3：同步语义

- **幂等 upsert**：机场按 `(organization_id, iata_code)`，港口按
  `(organization_id, un_locode)` 冲突更新。
- **停用本版缺失**：事务开始时先把目标组织内同 `source` 的记录全部
  `enabled = false`，再按本版数据逐条 upsert 并写入本版计算的 `enabled`。
  「上一版存在、本版缺失」的记录因此保持停用；人工维护记录（`source` 不同）
  不受影响。UN/LOCODE 的改码行（change code `#`）不做专门处理，由该规则兜底。
- **人工中文名保护**：upsert 不覆盖已非空的 `name_zh`、`city_name_zh`；为空
  时用英文名兜底写入。首次导入因字段非空约束，中文名一律以英文名初始化，
  之后靠人工修正。
- **英文官方名随源更新**：`name_en`、`city_name_en`、`country_code`、
  `icao_code`（机场）、`transport_modes`（港口）、`enabled` 每次同步按本版
  数据覆盖。

### D4：溯源字段

`Airport`、`Port` 增加 `source_version` 可空字段（命名与
`administrative_region.source_version` 先例一致）。`source` 取常量
`OURAIRPORTS`、`UNECE_UNLOCODE`（对照 `MCA_DMFW` 风格）。

本方案不建导入批次表：`BackgroundTask` 模型明确不保存业务 payload，导入统计
先以命令行输出与结构化日志为准；需要可查询的批次历史时（对照旧项目
`UnlocodeImportBatch`）再单独设计。

### D5：数据获取方式

- **OurAirports**：默认从 `https://ourairports.com/data/airports.csv` 下载并
  缓存到本地；`-source` 可指定本地文件跳过下载。
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

1. `iata_code` 非空且匹配 `^[A-Z0-9]{3}$`（全量约 7.5 万行，有合法 IATA 码
   的约 6 千余条；无 IATA 码的小机场不导入）。
2. 同一 IATA 码多行时保留首行（源数据偶发重复）。

### 字段映射

| OurAirports 列 | Airport 字段 | 规则 |
|---|---|---|
| `iata_code` | `iata_code` | 大写化，过滤见上 |
| `gps_code` | `icao_code` | 匹配 `^[A-Z0-9]{4}$` 才写入；空或非法置 NULL；同一组织内 ICAO 已被占用时后来者置 NULL，避免触发 `(organization_id, icao_code)` 唯一索引；不使用 `local_code` 兜底，保持 ICAO 语义 |
| `name` | `name_en` | 原样写入 |
| `municipality` | `city_name_en` | 可空 |
| `iso_country` | `country_code` | 大写两位 |
| `type` | `enabled` | `closed` → false，其余 → true |
| —（首次导入） | `name_zh` | 用 `name` 兜底；后续同步不覆盖非空值 |
| —（首次导入） | `city_name_zh` | 用 `municipality` 兜底，为空时用 `name` |
| — | `source` | `OURAIRPORTS` |
| — | `source_version` | `-release` 值 |
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
| Change Indicator 为 `+`、`#`、`|` 或空 | 正常导入 |
| Country 或 Location 为空 | 跳过并计数 |

### 筛选与映射

- **只导入海港**：`Function` 首位（index 0）为 `1` 的行才导入；不重蹈旧项目
  全量 10 万条地点入库、查询时再过滤的路径。
- `un_locode` = Country + Location（5 字符，大写）。
- `name_en` = 官方 Name 列；`name_zh` 首次导入用 `name_en` 兜底，后续不覆盖。
- `transport_modes` 由 Function 各位置生成，取值约定为
  `SEA`（位置 1）、`RAIL`（位置 2）、`ROAD`（位置 3），与订单线路类型的大写
  风格一致。位置 4（机场）、5（邮政）、6（多式联运）、7（固定设施）、
  `B`（边境口岸）第一版忽略，作为已知限制记录。
- `Status` 为 `XX`（撤销）的行导入但 `enabled = false`，保留可检索历史记录。
- Subdivision、IATA、Coordinates、Remarks、Date 列不导入（模型无对应字段；
  坐标待地图需求出现后加字段重导，数据源幂等）。

## 已知限制

1. 机场与港口无权威中文名，`name_zh` 初始为英文名，需人工修正；旧项目自维
   护的约 415 条机场中文映射可作为一次性人工种子单独迁入（只写空值），不参
   与后续同步。
2. UN/LOCODE 别名行被跳过、去变音名列未存储、Function 位置 4-7 与 `B` 未映
   射、坐标未存储。
3. 只同步单一目标组织；多组织共享主数据未决策。
4. 无导入批次历史表，统计仅在命令输出与日志中。
5. UN/LOCODE 需人工下载新包；OurAirports 每日更新但同步频率由人工触发决定。

## 验证

- 解析与清洗逻辑（CSV 解析、latin1 样本、Function 映射、标题/别名行、
  ICAO 冲突清洗）以表驱动单测覆盖，样本数据内嵌测试文件，不依赖网络。
- `go -C server test ./...`、`go -C server vet ./...`。
- 真实数据验收：预览统计数量级核对（机场约 6 千、海港约 1 万）；`-apply` 后
  SQL 抽查新增/更新/停用计数、人工中文名未被覆盖、重复执行结果幂等。

## 实施步骤

1. `Airport`、`Port` schema 增加 `source_version`，生成迁移并提交。
2. 实现 `cmd/sync-airports` 与单测，真实数据预览验收后提交。
3. 实现 `cmd/sync-unlocode` 与单测，真实数据预览验收后提交。
4. （可选）迁移旧项目机场中文映射作为一次性种子脚本。
5. 每步完成后回填本文「实施状态」。

## 实施状态

未开始。本方案对应 `PLAN.md` D11/D14 中 AirportDictionary、Unlocode 系列模
型「外部文件解析」的迁移要求；落地后同步更新迁移矩阵对应行。
