# 机场与港口主数据同步

> 现状核对：2026-11-09，依据当前 CLI、Ent Schema 和同步仓储。文末保留早期验收记录；本次未联网下载、写库或重跑验收。

机场与海港属于全局公共目录，所有公司消费同一份标准资料，系统工作台按权限维护。
同步命令不再接受 `-org-code`，也不以 `BOOTSTRAP_ORGANIZATION_CODE` 决定目标公司。
公共目录与维护权见[共享主数据契约](../.trellis/spec/server/backend/organization-shared-masterdata.md)。

## 数据源与命令

| 主数据 | 当前数据源及入口 | 中文与维护边界 |
| --- | --- | --- |
| 机场 | OurAirports CSV，`cmd/sync-airports`；默认联网下载，`-source` 可读本地文件 | 英文资料随源更新，保留已维护的中文名及中文城市名 |
| 海港 | UNECE UN/LOCODE ZIP + 配套中文 CSV，`cmd/sync-unlocode`；默认读取仓库归档 | 只导海港；中文对照必须完整且匹配当前英文身份，更新会写入该中文值 |
| 航司 | OpenFlights 归档及补充资料，`cmd/sync-airlines` | 已有独立同步入口，字段与对照边界见 seeds 说明 |
| 船公司 | SMDG 归档及中文对照，`cmd/sync-shipping-lines` | 已有独立同步入口，不再是“只人工维护、不接入外部源” |

归档版本、授权与中文资料来源见[种子说明](../server/seeds/README.md)、
[港口中文对照](../server/seeds/ports.zh-CN.md)和[船公司对照](../server/seeds/shipping-lines.zh-CN.md)。
这些资料由显式命令同步，不代表后台实时更新或已实现定时作业。

在仓库根目录写入公共资料：

```bash
# 根目录同步脚本均已带 -apply，会写数据库
pnpm run sync:airports
pnpm run sync:unlocode

# 或一次同步船代、船公司、海港、航司、机场和行政区划
pnpm run sync:all
```

直接运行 Go CLI 时默认只预览，追加 `-apply` 才写库。下例在仓库根目录加载本地环境：

```bash
node --env-file-if-exists=.env.local scripts/run-with-env.mjs \
  go -C server run ./cmd/sync-unlocode

node --env-file-if-exists=.env.local scripts/run-with-env.mjs \
  go -C server run ./cmd/sync-airports -source /path/to/airports.csv -release 2026-11-09
```

## 同步一致性

- 机场以 `iata_code`、港口以 `un_locode` 全局匹配，不包含公司唯一键。
- 完整解析、字段校验和数据库异来源冲突预检通过后才写库；写事务内再次校验冲突。
- 在同一事务中停用本来源旧记录，再 upsert 当前版本；本版缺失记录保持停用。
  不同来源或人工记录不被抢占，同码异来源冲突显式拒绝。
- 来源由系统维护；记录保留 `source`、`source_version` 和 `source_hash`，人工接口
  不能伪造同步来源。统计由命令输出，不把后台任务表当任意导入 payload 存储。
- 事务统一使用 Data 封装，查询与持久化集中在 `internal/data/industry_reference_sync.go`。
  下载及解析在 CLI，数据库事务中不执行网络下载。

## 机场规则

CLI 参数为 `-apply`、`-source`、`-release`。默认下载地址由
`server/cmd/sync-airports/main.go` 的 `ourAirportsURL` 定义，当前为 OurAirports
发布的 GitHub Pages CSV；下载文件缓存在 `.cache/master-data/`，不提交 Git。
无显式版本号时以执行日期标识；只预览也可能下载和写本地缓存，且要连接数据库预检。

| OurAirports 列 | Airport 字段与校验 |
| --- | --- |
| `iata_code` | 大写化，须匹配三个英文字母；无合法 IATA 不导入 |
| `gps_code` | 仅四位字母数字写入 `icao_code`，不使用 `local_code` 替代 |
| `name`、`municipality` | 对应英文机场名、城市名 |
| `iso_country` | 大写两位国家码 |
| `type` | `closed` 为停用，其他为启用 |

同一 IATA 或有效 ICAO 多行冲突时拒绝写入。中文字段不由此 CSV 填写，重复同步保留人工值；
不应把机场规则推广为港口中文的写入规则。

## 海港规则

CLI 参数为 `-apply`、`-source`、`-chinese-source`、`-release`：

- 默认寻找 `server/seeds/loc251csv.zip` 归档包及同目录 `ports.zh-CN.csv`。
- ZIP 发布版本优先从分片文件名前缀识别；无法识别时必须显式提供 `-release`。
- 更换 ZIP 时同时提供匹配的中文对照，不能沿用错版 CSV，也不联网临时补译。
- 中文 CSV 的代码集合必须与当前海港集合完全一致，逐条核对英文身份、合法中文名、
  名称类别和来源；任一缺失、重复或不匹配均失败。`source_hash` 包含 ZIP 与中文 CSV 的摘要。
- 未配置 `DATABASE_SOURCE` 且未传 `-apply` 时可离线校验源文件；此时明确没有检查数据库冲突。

换用其他版本的写入示例（参数直接附在脚本名后，不额外插入 `--`）：

```bash
pnpm run sync:unlocode -source /path/to/release.zip \
  -chinese-source /path/to/ports.zh-CN.csv -release 2025-2
```

解析 ZIP 要求 `UNLOCODE CodeListPart1/2/3.csv` 三个分片齐全，可识别目录及版本前缀，
支持 UTF-8 与 cp1252。只有 Function 首位为 `1` 才进入海港目录：

| 行或字段 | 当前处理 |
| --- | --- |
| Location 空且 Name 以 `.` 开头 | 国家标题，跳过 |
| Change Indicator 为 `=` | 别名行，跳过 |
| 同码变更候选 | 按 `#`、`+`、`\|`、空的明确优先级选择，同优先级重复阻止写入 |
| Country 或 Location 空 | 跳过并计数 |
| `un_locode` | 国家 + 地点，共 5 位大写标准码 |
| `name_en`、`name_zh` | 官方 Name 与配套中文对照，写入时均更新 |
| `transport_modes` | 当前映射 SEA、RAIL、ROAD |
| Status 为 `XX` | 保留记录但停用 |

## 验证与限制

解析、冲突拒绝和同步规则由 CLI 与 data 层相关测试覆盖；真实数据验收仍须在专用环境
分别记录源文件版本、摘要、预览、新增/更新/停用数量、重复执行与中文维护结果。
未配置真实库而跳过测试不能计作通过。

本期没有可查询的导入批次历史表或自动定时同步。UN/LOCODE 的别名、去变音名、坐标，
以及 Function 位置 4—7 和 B 尚未作为独立字段导入。机场缺少中文时可人工补充，
不因没有中文而静默改用其他来源。

## 历史实施记录

2026-08-23 的原始验收记录：向当时 HQ 组织写入 9,054 条机场与 17,524 条海港，
海港重复执行新增为 0。这些数量与组织归属仅反映当时版本，不是当前库存统计或当前
组织级同步契约。其后的公共目录迁移、港口完整中文对照、航司和船公司同步均以当前
实现及上文说明为准；本次文档核查没有重跑这些写入。
