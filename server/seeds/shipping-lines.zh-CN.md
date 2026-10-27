# 船公司中文对照与 seed 命令

编制日期：2026-10-27。底表：`SMDG_Liner-codes-list-20260903.xlsx`。

`shipping-lines.zh-CN.csv` 包含底表全部 **360** 家船公司，按 SMDG 三位代码排序，
中文覆盖 **100%**。SMDG 提供代码与公司原文身份，中文对照由本项目单独维护，
不是 SMDG 官方中文名录。

## 字段与编制口径

| 字段 | 说明 |
| --- | --- |
| `smdg_code` | 底表三位代码，含数字的 `G2O`、`TI2` 同样有效 |
| `name_en` | 与底表英文身份一致，比较时忽略排版换行及连续空白，仍保留全部单词、大小写和标点 |
| `name_zh` | 已有中文或本次翻译、音译；允许保留品牌缩写，例如 TI2 |
| `name_kind` | `existing` 为已有中文，`translated` 为本次编制或修订 |
| `source_ref` | 已有名称出处，或编制方式及身份核对说明 |

本版 **36** 条 `existing`、**324** 条 `translated`。已有名称主要来自仓库原有的
中文对照，逐行注明；东进商船依据[招商港口合作伙伴名单](https://en.portshekou.com/hzzgg/)，
古德航运依据[展会主办方中英文参观指南](https://www.wbx-sh.com/uploads/Show%20guide-%202024.pdf)。
其余按底表公司原文翻译或音译，未逐条取得行业通用中文出处，均保守标为编译名；
不能把编译名视为公司法定中文全称。品牌仅有缩写时保留缩写并译出业务类型。

本次按底表身份修正了容易混淆的旧映射：

- `CNC` 为 `DONGSUE Shipping`，编译为「东穗航运」；正利航业对应 `CLN`。
- `DJS` 为 `Dongjin Shipping`，中文为「东进商船」；锦江对应 `SJJ`。
- `PAN` 为 `Pan Continental Shipping`，修订为「泛洲海运」；不套用 Pan Ocean。
- `SML` 为 `Seaboard Marine`，编译为「海滨海运」；主体见[公司介绍](https://www.seaboardmarine.com/about/)，森罗商船对应 `SLC`。
- 长锦商船对应 `SMM`，中国外运集运对应 `SNL`，中谷海运新加坡公司对应 `ZGL`。
- `COE` 为 Diamond Line，编译为「钻石航运」；`HTM` 为长荣海运英国公司。

上述名称调整不更改底表代码或英文身份，既有国家和联盟字典沿用。
实际入库验证另发现旧箱号字典混用了不符合数据库约束的承运人代码：
移除长荣的 `EGLV`，保留 BIC 登记的 [EMCU](https://www.bic-code.org/bic-codes/emcu/)；
将 `ONEY` 改为 [ONEU](https://www.bic-code.org/bic-codes/oneu/)，
`KMTC` 改为 [KMTU](https://www.bic-code.org/bic-codes/kmtu/)，
`MATS` 改为 [MATU](https://www.bic-code.org/bic-codes/matu/)。
同步前校验箱号前缀为三位字母加 `U/J/Z`，避免整批在写入时才因格式错误回滚。

## 命令与更新行为

仓库根目录执行：

```bash
# 本次新增含数字 SMDG 代码的数据库约束迁移，首次使用前执行
pnpm run migrate:server

# 完整离线校验，不要求数据库连接
go -C server run ./cmd/sync-shipping-lines

# 新环境导入或已有同来源记录补中文，同时刷新全拼、首字母检索键
pnpm run sync:shipping-lines

# 统一主数据同步已包含上面的船公司命令
pnpm run sync:all

# 自定义底表必须同时提供与之匹配的完整中文表；预览后加 -apply 写库
go -C server run ./cmd/sync-shipping-lines -source /path/to/liners.xlsx -chinese-source /path/to/shipping-lines.zh-CN.csv
```

中文表默认取 Excel 同目录的 `shipping-lines.zh-CN.csv`，不需要手工导入中文或额外参数。
新建及已有同来源记录都写入中文，保留船公司 ID 和订单引用，重复执行不会新增重复船司。
既有同步规则仍会更新同来源的名称、启用状态、国家、联盟、网址及箱号前缀，并停用底表已撤销的
同来源船司；其他来源占用同码时显式拒绝，管理员修改同来源记录会由 seed 覆盖。

同步只读取归档文件，不联网翻译。缺行、多行、重复代码、英文身份不符、无效中文、
非法名称类别或来源为空都会在连接数据库前失败，不再把英文复制到中文字段。
预览显示中文覆盖、两类数量、Excel 与 CSV 各自的 SHA-256；船公司 Schema 目前不持久化文件指纹。
无数据库配置时只验源并明确提示未检查数据库冲突；`-apply` 必须提供数据库连接。

升级底表时必须同步维护中文表的新增、撤销和英文更名；业务人员修订编译名时修改 CSV 并保留依据，
再运行原 seed 命令即可；后续仅修订中文表无需新增 Schema 迁移或生成前端客户端。

## 本版验证

- 底表 YQN 公司名包含排版换行，CSV 去除换行前的行尾空格；匹配仅忽略空白差异，
  数据库仍保存底表原文，不会因排版空白把同一公司误判为错版。
- 全量配套文件测试覆盖 360 条中文、代码集合及英文身份；失败矩阵覆盖缺行、多行、
  重复、身份错配、无效中文、类别及来源，并验证错误发生在数据库连接之前。
- 独立 PostgreSQL 测试库先执行正式迁移，从 1 条英文占位旧记录开始实际运行 seed：
  首次新增 359、更新 1，再次新增 0、更新 360。数据库中文与 CSV 逐项一致，英文身份一致并保留底表排版原文，
  所有记录启用且检索键非空，旧记录 ID 保持。
- `TestShippingLineChineseSyncPostgres` 经正式迁移的隔离 Schema 验证新旧中文写入、
  代码/中英文/拼音检索、订单引用及箱号前缀保持、来源冲突，以及含数字代码入库。
  `TestShippingLineCodeSchemaMetadata` 检查 Ent 生成约束与正式迁移一致。
- 本版实际写入验证仅使用一次性测试库。
