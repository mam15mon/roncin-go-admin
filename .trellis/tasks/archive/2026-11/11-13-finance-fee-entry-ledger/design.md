# 费用录入与费用台账：技术设计（R1–R7）

对应 [prd.md](./prd.md)，调研见 `research/r1-*.md`、`r2-r5-*.md`、`r6-r7-*.md`。
本文只写设计，执行顺序见 implement.md。

## 0. 已确认决策

用户在规划阶段确认：

1. **日期权威源**：共享航程／主单改期时，在同一事务内把新日期同步写回各成员订单的 `orders.etd/eta`，订单、费用、报表读同一值。
2. **税率权限范围**：新增和修改均可改税率，仅限未建账费用；改后的税率不被费用设置目录税率覆盖；已建账费用仍只能走费用更正。
3. **筛选设置接入范围**：本任务只交付公共能力并接入费用台账、账单、提成三页；收付、发票、对冲、核销与报表页随各自任务接入。

以下为我据调研取的工程默认，评审时可推翻：

- 历史来源 `DERIVED`、`INHERITED_LAST_WEEK` 与同币种 `SYSTEM` 视为系统汇率，只有 `MANUAL` 受保护。
- 「有效」财务关联：发票状态 `DRAFT`/`ISSUED`（排除 `CANCELLED`/`RED_FLUSHED`）；核销 `ACTIVE`；对冲 `CONFIRMED`。均为账单级事实，账单含该费用即视为关联。
- 审核锁定订单不绕过：共享改期本来就要求全部成员未锁，费用写入口也拒审核锁订单，联动不新增绕过。
- 整表保存只含新增与修改，删除沿用单独的删除接口。
- 本币未税由后端出，复用 `biz.CommissionNetBase`。

## 1. R1 汇率按 ETD／ETA 取值并联动

### 1.1 取值日期

- 新增纯函数 `OrderRateDay(order)`（放 `biz`）：`trade_direction=export` 取 ETD，`import` 取 ETA，转业务时区自然日；字段缺失返回新专用错误（见 1.4），不回退另一日期、不回退创建时间。
- `resolveExchangeRate`（`biz/order_fee.go:522`）改调该函数；`repo.OrderETDDate`（`data/order_fee.go:149`）改为读取订单方向与 ETD/ETA 两列。
- 内贸、陆运、铁路是否一律按 `trade_direction` 判定，调研未确认。实施第 1 步先核对这些品类的 `trade_direction` 是否有值；若无值则缺日期显式报错，不静默默认出口。

### 1.2 共享改期同步订单列

- 共享海运航程更新（`data/sea_order_change_transport_update.go:105-246`）与空运主单更新（`data/air_export.go:385-427`）在已持有成员订单升序锁的前提下，给每个成员订单写 `SetEtd/SetEta`，再进入联动。
- 单票订单保存（`UpdateDraft`，`data/order_write_draft.go:267-268`）与单票 SE 内联改 MBL/TE（`data/order_write_sea_master_bill.go:443-451、546-554`）已写订单列，只需在日期实际变化后触发联动。
- 改配、拆票、恢复（`sea_order_change_reassignment.go`、`sea_order_change_split_execute.go`）也写订单日期列，纳入同一联动入口，作为单独实施步骤并配回归。
- 只有**与品类取值基准相关**的日期变化才触发：出口看 ETD，进口看 ETA；改另一个日期不重取费用汇率。

### 1.3 联动用例

新增 `data` 内部函数 `refreshFeeRatesForOrders(ctx, orderIDs)`，在日期写入的同一事务里调用，输入是已按主键升序锁定的订单集合：

1. 对每个订单（升序）读取其费用，按主键升序 `ForUpdate`。
2. 筛选可更新费用：
   - 来源不是 `MANUAL`；
   - 是当前有效费用（`biz.IsCurrentFee`，未被更正替代）；
   - 在锁内最新事实下没有有效发票、核销、对冲关联；
   - 没有「无需发票」标记（标记由账单任务引入，见第 6 节）。
   未建账与已建账但无有效财务关联的费用都可更新；不得借此修改已建账费用的其他受限字段。
3. 对所有可更新费用先批量解析目标周汇率（`ForShare` 读汇率设置）。**任一缺失即整体返回专用错误，日期与费用都不写。** 不需更新的费用不要求其汇率存在。
4. 全部可解析后再写：汇率、取值日期、来源、折本币金额；原币金额不变；费用 `version+1`、订单 `fee_content_version+1`、调用 `reconcileOrdinaryFeeEconomics`（`data/fee_correction_commission.go:344`）、写审计。
5. 账单行汇率与折本币快照不动（PRD 已定），费用汇率更新不覆盖账单汇率。

锁序沿用既有「成员订单升序 → 主单／Link／TE → 费用行升序」，不出现费用先于订单。

**并发校验点**：调研未验证发票、核销、对冲提交时是否先锁 Order。实施第 2 步先确认；若不先锁，则联动在费用行锁内重读关联事实，并让这些写路径在写关联前对相关费用行加锁，不能只靠前端状态。

### 1.4 缺汇率专用错误

新增 `ErrExchangeRateMissingForDate`，错误 metadata 带缺失币种、目标周、方向；`ORDER_ETD_MISSING` 改为带「ETD/ETA」字段名的专用文案。前端日期保存失败时在表单上展示该信息，不显示成功后再异步报错。

### 1.5 规范同步

`exchange-rate-single-rate.md:33` 的「统一按订单 ETD」改为出口 ETD、进口 ETA，并补联动与手工保护规则。

## 2. R2 应收／应付独立整表保存

### 2.1 后端契约

在 `order_fee.proto` 新增 `SaveOrderFees`（`POST /orders/{id}/fees:save`）：

- 请求：订单、方向（RECEIVABLE／PAYABLE）、`request_key`、`rows[]`；每行含 `client_row_id`、可选 `fee_id`+`expected_version`、可编辑字段（含 `tax_rate`，受 R3 权限约束）。
- 权限：新增行要求 `fee.create`，修改行要求 `fee.update`，两者在服务端分别校验；税率字段另受 R3 权限。
- 单事务：锁订单（含 `LockedAt`/业务门禁，复用 `lockOrderForFeeMutation`）→ 目标费用按主键升序 `ForUpdate` → **逐行完整校验后才写**：版本、已建账、受保护、手工汇率覆盖条件、币种集合。全部通过才落库，任一行失败整体回滚。
- 手工汇率覆盖的币种集合条件按「本次提交后的整单费用集合」在事务内重读判定，而不是事务外已落库集合；另一侧未保存数据不算服务端事实。
- 一次提交后只调用一次 `reconcileOrdinaryFeeEconomics`，只递增一次 `fee_content_version`。
- 幂等：新增行的幂等键由 `sha256(request_key + client_row_id)` 派生，复用 `orderfee_order_id_idempotency_key` 唯一索引兜底，**不新增表**。响应丢失重试：若所有新增行已存在且意图一致、所有修改行已处于目标内容，返回当前状态视为成功；其他半应用情形返回 409，不重复新增。
- 行级错误：错误 metadata 带 `client_row_id`、字段、原因，前端据此定位，不再把费用 ID 拼进文案。

### 2.2 前端

- `OrderFeeTableTabs.tsx` 的两个 `EditableProTable` 改为多行常开编辑（`editable.type:'multiple'`），取消行内「编辑」按钮；各表标题右侧「保存」调用 `SaveOrderFees`，只提交本侧。
- 保存成功只刷新本侧数据；另一侧的草稿与提交基线保留，不被新查询结果覆盖。当前 `:891-892` 的两表同时 reload 需拆开。
- 保存失败保留输入并按行展示错误；响应丢失后重试沿用同一 `request_key`，成功后更换。
- 费用页没有外部变动检测（只在订单详情、拆单页有），**新建**：前台页签每 15 秒查询轻量变动标识（`fee_content_version`），隐藏页签不轮询，发现变化持续提示「这个订单已经发生变动，请重新刷新。」，保留草稿，覆盖前确认。

## 3. R3 订单费用税率更改权限

- `access/manifest.go` 新增系统级权限 `FinanceOrderFeeTaxRateEdit`（`system.finance.order_fee.tax_rate.edit`），名称「订单费用税率更改权限」，分组「费用管理 · 税率」，`Requires` 取费用设置读取与订单费用读取中已存在的键（实施时按 manifest 现有键名选定，不新造依赖）。修改后执行 `pnpm run generate:permission-keys`，并同步 `access.ts`。
- 服务层沿用汇率覆盖的传参方式：`HasPermissionInScope` 的结果作为 bool 传入用例，无权且请求带税率变化返回 403 `ErrOrderFeeTaxRateForbidden`。
- 业务规则：新增、修改未建账费用时允许设置税率；已建账费用带税率变化被拒（仍走费用更正）。
- 存储：费用行已有 `tax_rate`，不新增列。`resolveCatalog` 仅在**费用设置变更且请求未显式给税率**时取目录税率；显式税率或未变更费用设置时保持已存税率，不被目录覆盖。
- 审计：税率变化写入费用审计，记录前后值与操作者。

## 4. R4 含税／未税双口径看板

- `OrderFee` 响应新增计算字段 `base_currency_net_amount`（不持久化）：由 `CommissionNetBase(base_currency_amount, net_amount, total_amount)` 得出，沿用费用自身汇率快照，含税与未税同一汇率。
- 页头三张卡（应收、应付、预计毛利）各自同时显示「含税／不含税」两组金额；毛利两组分别为应收减应付。
- 汇总用 `Decimal` 累加，替换现有 `Number` 浮点（`OrderFeeTableTabs.tsx:1582`）；缺本币金额的费用计入「未折算」并显示 `-`，不再按 0。多币种在原币层分组展示，不把不同币种直接相加。
- 底部 `FinanceSummaryBoard` 保持现状，不在本期扩展。编辑中未保存的预览不进入看板，仍只汇总已落库数据。
- 验收样例：应收含税 106／未税 100，应付含税 53／未税 50 → 毛利含税 53／未税 50，六个数同时可见。

## 5. R5 列宽

- 金额类（单价、总金额、税金、不含税总额、折本币）统一 120，右对齐；税率 90；币种 90；汇率 110；数量 190；费用名称、结算单位 200；备注 160 省略显示。
- 整表保存后取消「编辑」按钮，操作列只剩「删除」，宽度约 80，保持右侧固定。
- `scroll x:'max-content'` 与全宽容器保持，不加 `maxWidth`。
- 列显隐/排序/冻结偏好存的是列名，不含宽度，用户已存偏好不需重置。

## 6. 与账单任务的依赖：无需发票标记

「无需发票」标记是账单任务（R9、R10）新建的财务事实，此处不创建。本任务把「有效财务关联」收敛为单一判定函数 `feeHasProtectedFinanceFact(ctx, client, feeID)`：现在判定发票、核销、对冲；账单任务引入标记后只在该函数追加一项条件。涉及标记的验收（标记保护、撤销后恢复）在账单任务里验证，本任务不假装已覆盖。

## 7. R6 主单号

- 主单号投影已在提交 `904343f6` 补齐（`data/fee_ledger_master_no.go:64-107`，列 `feeLedgerColumns.tsx:108`），不再改根因。
- 本期补：台账导出（`ledgerExport.ts`）加入主单号列；补海运有主单、空运有主单、无主单、终止订单的验收用例；导出与列表值一致。

## 8. R7 统一筛选设置（公共能力）

- 新增 `web/src/components/ui/filter-settings/`：`preference.ts`、`useFilterSettings`、`FilterSettingsEntry`，位置符合架构检查（`components/ui` 不导入 `pages`／`features`，字段选项由页面传入）。
- 存储 key：`roncin:filter-settings:v1:<pageKey>:<userId>:<orgId>`，值仅保存可见字段名；未知字段忽略，新字段按页面声明的默认可见补入；存储失败明确提示一次，当次仍生效；身份未就绪不读不写。
- 扩展 `SearchFilterTemplate`：字段由可见变不可见时复用 `deleteChip` 的清值逻辑——清除表单值与已提交条件，并用剩余条件刷新、回到第一页、清空勾选；重新加入不恢复旧值；折叠只改变展示，不改变查询。
- 公司工作区与权限范围不是筛选字段，不可通过设置移除。页签恢复（`useListQueryState`）取回筛选值时剔除已隐藏字段，避免隐藏条件继续生效。
- 列表、顶部指标、筛选合计、导出沿用既有 `onLoad` 采纳快照，因值已在移除时清除而自然一致。
- 接入页：费用台账、账单、提成；其余页面不在本任务内。不支持调整筛选项顺序。

## 9. 风险与取舍

| 风险 | 处理 |
|---|---|
| 联动事务内费用与发票、核销、对冲并发 | 锁内重读；实施第 2 步先验证现有写路径锁序，必要时补费用行锁 |
| 共享改期写成员订单列扩大了改动面 | 复用已持有的升序成员锁，同一事务写入；加多票回归 |
| 已确认或已支付提成受费用汇率联动影响 | 走既有 `reconcileOrdinaryFeeEconomics`，已付事实与手工调整保护沿用 finance-commission-lock |
| 整表保存回放判定复杂 | 仅「全部已应用」才视为成功，半应用返回 409，不猜测 |
| LAND／RAIL 品类方向值可能缺失 | 实施第 1 步核对，缺失显式报错 |
| 新增本币未税字段与提成口径分叉 | 统一复用 `CommissionNetBase` |
| 账单任务晚于本任务 | 判定函数留扩展点，标记相关验收放账单任务 |

## 10. 不做的事

不加汇率日期优先级配置；不自动回退创建时间或前后周汇率；不改手工覆盖权限，不加「恢复自动汇率」；不新增表；不同步筛选偏好到服务端；不接入收付、发票、对冲、核销与报表页的筛选设置；不恢复业务标签。
