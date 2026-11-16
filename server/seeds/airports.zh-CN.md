# 机场中文对照数据编制说明

编制日期：2026-11-16。配套底表为仓库 `airports.csv`，来自本地已归档的
OurAirports 2026-10-27 快照，原缓存文件名为
`ourairports-airports-2026-10-27-3c27fedb907e.csv`。

## 覆盖范围与身份

本表覆盖当前解析口径下全部 **9,053** 个三字 IATA 地点，中文覆盖率 **100%**。
包含大型、小型机场、直升机场、水上飞机基地等底表类型；少量底表明确收录的联运
地点保留原身份，例如 `ZGN` 的原名是 `Zhongshan Ferry Port`，中文为“中山客运港”，
不为凑齐机场名称而改成机场。不同机场不因同城或中文名近似而合并。

| 字段 | 含义 |
| --- | --- |
| `iata_code` | 当前底表解析后的 IATA 三字码，一码一行 |
| `icao_code` | 底表解析得到的有效 ICAO；无有效四字码时为空 |
| `country_code` | 底表国家或地区码，用于防止同名、旧码复用串名 |
| `name_en` | 与当前底表逐字对应的机场英文身份 |
| `city_name_en` | 与当前底表对应的城市英文身份；原始为空时保持空 |
| `name_zh` | 已有中文名称或本次编制的中文翻译、音译 |
| `name_kind` | `existing` 为有可核验中文来源，`translated` 为本次编制或修订 |
| `source_ref` | Wikidata 实体链接或编制依据 |

本版 **4,133** 条 `existing`、**4,920** 条 `translated`。中文城市不是本次交付范围，
城市英文列只用于核对身份；同步不据此覆盖管理员维护的中文城市名称。

## 来源与编制方式

- [OurAirports 数据下载](https://ourairports.com/data/)及其
  [官方机场 CSV](https://davidmegginson.github.io/ourairports-data/airports.csv)
  提供原始代码、英文名称、国别、地点和机场类型。OurAirports 明确将数据置于公有领域。
  本仓库固定归档版本，日常同步不重新下载。
- [Wikidata](https://www.wikidata.org/) 提供已有中文标签；按 IATA（P238）、ICAO
  （P239）、国家码（P17/P297）及英文名称核对。ICAO 不同、国家不符、同码旧身份等
  候选不直接套用。结构化数据采用
  [CC0](https://www.wikidata.org/wiki/Wikidata:Licensing)，逐行保留实体链接。
- 中文标签优先使用 `zh-cn`、`zh-hans`；其他中文标签通过
  [OpenCC](https://github.com/BYVoid/OpenCC) 转为简体。仅有英文专名加“机场”的标签
  不算完整中文来源，改为编制译名。
- 长尾名称由 [Argos Translate](https://github.com/argosopentech/argos-translate)
  的 en→zh 1.9 离线模型辅助生成，结合原始英文和设施类型整理。模型来自
  [Argos 官方模型仓库](https://data.argosopentech.com/argospm/v1/translate-en_zh-1_9.argosmodel)，
  其 OPUS-MT 衍生模型由 Jörg Tiedemann、Santhosh Thottingal 等提供，模型包署名为
  CC BY 4.0，参见 [OPUS-MT](https://github.com/Helsinki-NLP/Opus-MT) 及同仓库
  [航空公司编制说明](./airlines.zh-CN.md) 的模型署名、许可与包摘要。模型、Python 库与采集缓存均不是
  同步命令的运行依赖，也不提交到仓库。

已有中文标签不等于当地机场官方中文标准；编制译名也不宣称已由机场经营方或业务
专家逐条审定。常用机场、中国地名、人物专名、军用与民用设施、模型元数据残留和
普通词误译作为重点校订项。后续业务修订维护此表及其来源说明。

## 编制中的身份与语义处理

- `PVG`、`SHA`、`PEK`、`PKX`、`CAN` 等保留真实城市及机场专名；`LAX`、`JFK`、
  `LHR`、`NRT`、`HND`、`ICN` 等常用机场单独核对，避免丢失人名、地名或主机场区别。
- 机场三字码可能被历史实体复用。例如本底表 `HAE` 是美国的 `Lava Falls Heliport`，
  不能因 Wikidata 同码候选是朝鲜海州机场就套用“海州机场”。
- `Airfield/Field` 在机场语境中不译成球场、战地或字段；`Seaplane Base` 保留水上飞机
  基地含义，`Heliport` 保留直升机场含义，军用基地与普通民用机场不混同。
- 地名、人名中的普通词不应机械按日常词义翻译；原文简称、旧名、纪念人物及设施区分
  信息结合原始名称整理。中国和东亚机场优先采用可核对的汉字名称，未找到完整中文
  来源的项目保守标为 `translated`。
- 不用“英文原名 + 机场”“三字码 + 机场”“未知机场”等占位方式补齐覆盖率。

## 同步与维护

```bash
# 仓库根目录：离线核对全部底表与中文集合，不写库
go -C server run ./cmd/sync-airports

# 配置 DATABASE_SOURCE 后显式写入；根命令包含 -apply
pnpm run sync:airports

# 自定义配套快照：先预览，确认后加 -apply
go -C server run ./cmd/sync-airports -source /path/to/airports.csv -chinese-source /path/to/airports.zh-CN.csv -release 2026-11-16
```

默认读取仓库两份配套文件；自定义 `-source` 未指定中文路径时，读取底表同目录的
`airports.zh-CN.csv`。自定义底表未指定 `-release` 时以底表摘要作为稳定版本。
没有数据库配置的预览完整校验源文件，并明确提示未检查数据库冲突；写入必须有
`DATABASE_SOURCE` 并显式指定 `-apply`。

中文 CSV 必须是 UTF-8，表头固定。集合缺失、多余、重复，代码、国别或英文身份不符，
空名、超长名、占位名、非法类别或空来源均在写库前失败。升级快照时同步维护中文表的
新增、撤销和更名，不使用另一版本的旧名称自动兜底，不在导入过程中在线补译。

底表 SHA-256：`3c27fedb907e02dedd0c1df7fd223cc31f847fce5f15eb722d871db480041299`。
中文 CSV 使用 LF 换行，SHA-256：`efa3c9a7db03e4152b190944a6ef446302791029f73849e1b14a119d5bed70c9`。
命令同时打印底表、中文表各自的摘要；落库 `source_hash` 为
`SHA256("ourairports-zh-v1\n" + 底表摘要 + "\n" + 中文表摘要 + "\n")`，仅修订中文也会
改变联合指纹。已有 ID、订单引用与中文城市保持原状态；同来源更新显式写入机场中文名，
刷新拼音与首字母检索键。

## 验证入口

- `cmd/sync-airports` 测试核对全量 9,053 机场的身份、中文覆盖、常用机场名称，覆盖
  缺行、多行、重复、错版、错国别、英文城市错配、非法中文、类别及来源失败矩阵。
- 指纹测试核对底表与中文表任一变化都会改变联合摘要；离线预览及自定义配对路径测试
  验证无数据库连接也可验源，显式写入缺配置时拒绝执行。
- 数据层真实 PostgreSQL 专项验证由 `TestAirReferenceChineseSyncPostgres` 执行，覆盖
  旧行补中文、已存在检索键更新、中文/全拼/首字母查询、ID 和订单引用、重复同步、
  来源冲突及整批失败回滚；真实库测试跳过不计为通过。
- 本版实际命令在正式迁移的独立 PostgreSQL Schema 首次新增 9,053 条，第二次新增
  0 条、更新 9,053 条；全部代码、中英文名与 CSV 逐条一致，记录 ID 保持不变，中文
  及检索键均非空。验收后已清理该临时 Schema，未向开发业务库执行 apply。
