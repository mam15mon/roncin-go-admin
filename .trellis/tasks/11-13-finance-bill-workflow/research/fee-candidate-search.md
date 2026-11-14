# 账单费用候选拼音检索窄研究

日期：2026-11-14。仅依据当前代码检查，没有修改检索 Schema 或查询。

## 结论

单独关联公司费目的检索键不能完整覆盖费用候选。普通费用强制选公司费目，但受控更正/补录允许自定义费用名称；账单使用的是费用快照名称，费目后续也可改名。应给 `OrderFee` 维护自身中英文名称的检索键，并复用已有生成与回填范式。

## 来源及名称边界

- `biz/order_fee.go:828` 要求非空 `FeeSettingID`、`BillingUnitID`；`resolveCatalog`（同文件约 475 行）从所属公司启用费目覆盖 `FeeCode`、`FeeName`、`FeeNameEN`。普通单笔更新会拒绝 `FeeNameOverride`（约 387 行），整表保存也调用同一目录解析。
- `biz/order_fee_correction_usecase.go:95` 解析目录后，约 98–99 行明确接受 `FeeNameOverride` 并赋值 `after.FeeName`。`data/order_fee_correction_apply.go:createCorrectionFee` 把该名称写入正式 `OrderFee`。因此不能把费用名称恒等于当前费目名称作为检索前提。
- `ent/schema/fee_setting.go` 已有 `search_keywords` 和 `searchKeywordsHook("name_zh", "name_en", "alias_name")`；`data/fee_catalog.go` 的费目查询覆盖代码、中文、英文、别名及拼音，且费目有公司归属。它可用于额外支持费目别名，但不足以替代费用自身快照名称的键。
- 费用记录自身有 `fee_code`、`fee_name`、可选 `fee_name_en`，没有别名字段。当前 `ListBillFeeCandidates` 只匹配费用代码、中文名和订单号，缺英文和拼音；初始建账候选 `finance_bill.go:ListCreationCandidates` 同样缺费用英文与拼音，但已有结算对象拼音/别名分支。

## 最小实施点

1. `ent/schema/order_fee.go` 复用 `searchKeywordsField()` 及 `searchKeywordsHook("fee_name", "fee_name_en")`；已有 `platform/searchtext.Build` 生成无声调连续/分词全拼及首字母，不新增另一套算法。
2. 正式迁移新增 `search_keywords text NOT NULL DEFAULT ''` 与 GIN `gin_trgm_ops` 索引；更新 Ent 生成物。
3. `data/search_keywords_backfill.go:selectorSearchKeywordBackfills` 加 `order_fees` 的 `fee_name/fee_name_en`。正式 `cmd/migrate` 在 SQL 迁移后已调用 `BackfillSelectorSearchKeywords`，可复用只补空键、参数化批量写入的范式。
4. 现有两个费用选择器增加 `FeeNameEnContainsFold` 与 `SearchKeywordsContainsFold`；保持同公司、方向、币种、对象、未占用和待审更正等谓词不变，不先加载全量费用做内存过滤。若需命中所属费目别名，用现有 `fee_setting_id` 的受限关联附加查询，不能跨公司按同代码模糊匹配。

## 所有真实写入口与 Hook 覆盖

| 入口 | 当前实现 | 需要的处理 |
| --- | --- | --- |
| 普通新增、整表保存新增 | `data/order_fee.go:newOrderFeeCreate` | 创建设置中英文名，通用 Schema hook 自动生成键 |
| 普通更新、整表保存更新 | `data/order_fee.go:newOrderFeeUpdate` | UpdateOne 设置中文名及设置/清除英文名，hook 读取完整新旧字段重建键 |
| 受控补录/更正生效 | `data/order_fee_correction_apply.go:createCorrectionFee` | 创建包含人工名称及可选英文名，hook 自动覆盖 |
| 海运拆票克隆未建账费用 | `data/sea_order_change_split_execute.go` 约 1894 行的 `OrderFee.Create` | 已复制中英文名，hook 自动生成目标新行键 |
| 开发业务数据 | `cmd/sync-dev/main.go` 约 1339 行创建费用 | 设置中文名，hook 自动覆盖；不手写另一套种子拼音 |
| 汇率联动、建账/取消/增删版本推进、更正源费失效 | `fee_rate_refresh.go`、`finance_bill*.go`、`order_fee_correction_apply.go` 的其他更新 | 不改名称，应保留现有键；通用 hook 在名称未变时不写 |

已用全仓 `OrderFee.Create/Update/UpdateOne` 与 `SetFeeName` 检索核对以上生产写入口；测试中的直接 Ent 创建同样会经过 hook。通用 hook 对 `ent.OpUpdate` 批量更新刻意跳过源字段重建，目前生产批量费用更新仅推进版本等非名称字段；不得新增批量改名后依赖该 hook 自动补齐。

## 针对性验证

- 中文、英文、代码、全拼、首字母分别命中；同公司同范围过滤不因 OR 分支扩大。
- 自定义更正名称、普通 UpdateOne 改费目导致名称/英文变化、清除英文名均重建键。
- 费目改名后旧费用快照名称的全拼仍可命中；克隆及更正新行自动有键。
- 真实正式迁移前态建立旧费用，升级后执行公共回填两次，验证检索键非空、幂等且费用金额/名称/版本不变；实际索引存在。
- 不使用前端循环分页、数据库内自造拼音函数或开发库清空。
