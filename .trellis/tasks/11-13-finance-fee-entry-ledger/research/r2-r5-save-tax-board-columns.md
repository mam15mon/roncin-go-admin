# Research: R2 整表保存 / R3 税率权限 / R4 双口径看板 / R5 列宽

- **Query**: R2-R5 现有代码与规范盘点（只读，不含改进建议；风险与待决问题单列）
- **Scope**: internal
- **Date**: 2026-11-14

## 1. R2 整表保存现状

### 前端（均为现状事实）
- 费用页 `web/src/pages/orders/fees.tsx`（913 行）：应收/应付两张表由 `components/fees/OrderFeeTableTabs.tsx`（1896 行）渲染，各自一个 `EditableProTable`（应收 :1507，应付 :1655），`key=receivable:${orderId}`，`pagination={false}`、`scroll={{x:'max-content'}}`。
- 行内编辑为 `editable.type:'single'`（:1517 起），`editableKeys` 各表独立（receivableEditableKeys / payable...），只能同时编辑一行；行内「编辑」按钮 :1437 调 `action.startEditable(id)`；`onSave` -> `handleSaveFee(key,row,originRow,newLine)`（:1552）。
- `handleSaveFee`（约 :800-905）逐行调用 `orderFeeServiceAddFee`（新增，`idempotencyKey: generateUUID()` 每次点击新生成，:867）或 `orderFeeServiceUpdateFee`（带 `expectedVersion: originRow.version`，:880）；成功后 `receivableActionRef/payableActionRef.reload()`（:891-892，两表都 reload）。请求体硬编码 `taxInclusive: true`，不发送 `taxRate`（:859）。
- 草稿状态：`receivableGuard / payableGuard`（`markDirty`/`resetDirty`，:1527-1532 起，来源为 formProps.onFieldsChange），两侧各自一个脏标记；`rowSelection` 排除无 id/`new_` 前缀/无 version 的行（:1542）。
- 另有弹窗式新增/编辑 `FeeFormModal.tsx`（`fees.tsx:405 handleModalSubmit`，用 `createIdempotencyKeyRef` 保持幂等键，成功后换新键 :459）。
- 版本检测轮询：`fees.tsx`、`components/fees/*.tsx`、`hooks` 中 grep `refetchInterval|15000` 无命中（不含测试）。15 秒外部变动提示现见于 `pages/orders/detail.tsx`、`use-sea-order-save.ts`、`split.tsx`（订单详情/拆单），费用页未发现同类实现。
- 费用请求：`request` 内 `orderFeeServiceListFees({orderId})` 一次取整单，再按方向过滤（:1560-1600），并通过 `setAllReceivableItems/setReceivableSummary` 写父级汇总。

### 后端写接口（`server/api/order/v1/order_fee.proto`）
| RPC | 行 | 说明 |
|---|---|---|
| AddFee | :74 POST /orders/{id}/fees | `AddFeeRequest`(:271) 必带 `idempotency_key`(16)；无 tax_rate/expected_version |
| UpdateFee | :84 PUT /orders/{id}/fees/{id} | `UpdateFeeRequest`(:290) 带 `expected_version`(17)、`tax_rate`(19)、`fee_name`(20) |
| RemoveFee | :94 | `expected_version` |
| BulkUpdateOrderFees | :102 | 仅「统一改结算单位」，targets=[fee_id+expected_version]，单事务全成败（:373-393） |
| BulkRemoveOrderFees | :113 | 批量物理删除，同上 |
- 权限规则：Add=`fee.create`、Update/BulkUpdate=`fee.update`、Remove/BulkRemove=`fee.delete`（`ACCESS_MODE_ORDER_PERMISSION`，按业务线）。
- **不存在**「整表新增+修改混合」接口；批量接口也不支持逐行字段值。
- 用例 `server/internal/biz/order_fee.go`：`Add`:242（幂等：`GetByIdempotencyKey`:268，`sameOrderFeeCreateIntent`:290，冲突 `ErrOrderFeeIdempotencyConflict`；唯一索引 `orderfee_order_id_idempotency_key` 兜底，`data/order_fee.go` Add）；`Update`:316（已建账直接 `ErrOrderFeeBillOccupied`:324-326；税率/费名请求非空直接 `ErrOrderFeeInvalidArgument`:353-355）；`BulkUpdate`:608；`BulkOrderFeeError`:195 把费用 ID 拼进 message（"费用 %s：..."），是现有唯一的「行级定位」做法（无结构化 details）。
- 数据层 `server/internal/data/order_fee.go`：均走 `WithTx`；`lockOrderForFeeMutation`:245（锁订单行 + `LockedAt`!=nil 返回 `ErrOrderAlreadyLocked` + `ensureOrderBusinessEditable`）；Update:416，`ForUpdate`:427、`ErrOrderFeeBillOccupied`:432、`Version!=` -> `ErrOrderFeeVersionConflict`:438、`SetTaxRate/ClearTaxRate`:511/513；批量 `lockOrderFeesForBulk`:639 按 `ByID()` 排序 `ForUpdate`（:643-644）后逐行比对版本（:665）。
- `reconcileOrdinaryFeeEconomics`（定义 `data/fee_correction_commission.go:344`）调用点：Add :400、Update :540、Remove :618、BulkUpdate :745、BulkRemove :833，均在同一事务内；每次写还执行 `AddFeeContentVersion(1)`（订单级费用内容版本，Add :395）。
- 汇率手工覆盖依赖整单费用集合：`resolveExchangeRate`:522 -> `orderFeeCurrencySetsMatch(existingFees, fee)`:493（`existingFees` 来自 `repo.List`，事务外读取，Add:258 / Update:342）。整表保存时应收与应付待提交值未入库，此条件需看「提交后集合」还是「已落库集合」——现有代码按已落库集合。
- `CanEdit` 能力由 data 层推导：`data/order_fee_review.go:528`（含 `LockedAt==nil`、ACTIVE、`!HasActiveBill`、`IsCurrentFee`）。
- 整表最小契约扩展位置（事实层面）：proto 加入 `order_fee.proto` 的 service（紧邻 :102 BulkUpdate 前后）；可复用元素：`BulkOrderFeeTarget`(:373)、`UpdateFeeRequest` 字段集、`AddFeeRequest.idempotency_key`、`BulkOrderFeeError` 行级错误文案、`WithTx`+`lockOrderFeesForBulk` 按 ID 排序加锁。其余（批次幂等键、结构化行级错误）现无先例。

## 2. R3 权限与税率校验

- 权限清单 `server/internal/access/manifest.go`：
  - 常量区 :43 `FinanceOrderFeeExchangeRateOverride = "system.finance.order_fee.exchange_rate.override"`；:44-47 费用设置/总台账键。
  - 条目 :215 `{Name:"调整订单费用汇率", Group:"费用管理 · 汇率", Requires:[FinanceExchangeRateRead]}`——最接近的先例（系统级键，不分业务线）。费用管理分组现有：汇率、费用设置、费用总台账、账单、收付、开票、核销、对冲、提成。
  - 订单费用 CRUD 按业务线动态生成 `订单管理 · {业务线} · 费用`（:309 定义 `fee.read/create/update/delete`，Requires 推导 :408-420：操作依赖同资源读+订单读）。
- 前端对齐：`web/src/access.ts:32-33` 键名映射、:219-222 `canOverrideFeeExchangeRate = canOperateBusiness && has(permissions.financeOrderFeeExchangeRateOverride)`；生成物 `web/src/permissions.generated.ts:32`；改 manifest 后执行 `pnpm run generate:permission-keys`（`package.json:17`，`check:permission-keys` 在 `check:web` 中校验）。`cmd/migrate` 自动同步 permissions 表并给 administrator 补挂（AGENTS.md）。
- 服务端汇率覆盖的传参与拒绝方式（可对照）：`service/order_fee.go` AddFee/UpdateFee 把 `principal.HasPermissionInScope(access.FinanceOrderFeeExchangeRateOverride, DataScopeOrganization)` 作为 bool 传入用例（:130 / :152 附近）；用例在 `resolveExchangeRate`:527-529 返回 `ErrOrderFeeExchangeRateOverrideForbidden`（403，`order_fee.go:25`，"无权手工覆盖费用汇率"）。
- 税率现状：
  - 费用税率来源是费用设置目录快照：`resolveCatalog`:429 设置 `fee.TaxRate=&snapshot.TaxRate`（:441）；`calculateAmounts`:404 用 `fee.TaxRate`（nil 则报错）。
  - 普通 `UpdateFee` 的 `tax_rate`（proto :288-290 注释「审核中及已批准费用均禁止直接更新」）：service `order_fee.go:~146-152` 解析为 `input.TaxRateOverride`，biz `Update`:327,353 只要非空即 `ErrOrderFeeInvalidArgument`，且 `TaxRateOverride` 在 normalize :785 仅校验 0-100 范围与格式，**普通 Update 路径从未应用它**。
  - 仅「费用更正」路径应用：`biz/order_fee_correction_usecase.go:101-103`（`after.TaxRate = after.TaxRateOverride`），service 解析 `service/order_fee_correction.go:121-127`；补录/补审 `order_fee_review.go:73` 将其置空。
  - 已建账字段策略里有 `BilledFeeFieldTaxRate`（`biz/finance_custom_setting.go:28`，校验于 `order_fee.go:394`，映射 `service/settlement_fee_ledger.go:248,266`）——属财务自定义「已建账可改字段」设置，与本次权限不同。
  - 前端费用页无税率输入：税率为只读可选列（`orderFeeOptionalColumns.tsx:222-243`，来自 `record.taxRate`或编辑预览的费用设置默认税率）。
- 税率修改的现有校验点汇总：biz Update:353（一律拒）、normalize:785（范围）、correction usecase:101（应用）。没有按权限校验税率修改的代码。

## 3. R4 看板数据来源

- 页面上有两处汇总：
  1. `components/fees/OrderFeeHeader.tsx:101-153`：三张 `Statistic`（应收总计/应付总计/预计毛利），值来自 `receivableSummary.totalAmount` 与 `payableSummary.totalAmount`，硬编码前缀 `¥`；毛利与毛利率在 `fees.tsx:614-618` 计算：`profitCny = receivable - payable`。
  2. 页面底部 `FinanceSummaryBoard`（`fees.tsx:785-803`，组件 `components/ui/finance-ledger-template/FinanceSummaryBoard.tsx`）：按方向、原币分组，另显「折合 本币」「N 条未折算」；毛利项 `config.profit`（应收-应付）。
- OrderFeeHeader 的合计口径：`OrderFeeTableTabs.tsx:1582-1587` 用 `cur.baseCurrencyAmount ? Number(...) : 0` 累加（含税本币；缺失按 0；`Number` 浮点），再 `setReceivableSummary({totalAmount,count})`。
- FinanceSummaryBoard 口径：`features/finance/ledger-summary/index.ts:121-136 orderFeeSummaryConfig.projectRow` 用 `row.totalAmount`(含税原币)+`baseCurrencyAmount`；`aggregateLedgerAmounts`（`ui/finance-ledger-template/summary.ts:8-38`）按 `[direction,currency,baseCurrency]` 分桶，Decimal 精度 48；`sumAmountsByCurrency`:41 分原币/本币；缺本币的行 `unconvertedCount` 计数，不补 0。该桶只含 `amount`/`baseAmount` 两列，无未税列。类型 `FinanceLedgerAmount`（`types.ts:44`）。
- 字段：`order_fee.proto` `OrderFee` 同时有 `total_amount`(11,含税)、`net_amount`(28)、`tax_amount`(29)、`base_currency`(30)、`base_currency_amount`(31)、`exchange_rate`(13)；biz `OrderFee` 同字段（`order_fee.go:84-95`）。`calculateAmounts`:404-427：含税输入 `net=total/(1+r)` RoundBank(8)、`tax=total-net`；`BaseCurrencyAmount = TotalAmount × ExchangeRate` RoundBank(8)。**无本位币未税/税额持久化字段**。
- 折算先例：`biz.CommissionNetBase(base, net, total)`=`base×net÷total` RoundBank(8)，total 为 0 返回 0（`biz/finance_commission_lifecycle.go:196-201`），用于 `finance_commission_economics.go:92,252`、`data/finance_commission_calculation.go:440,543`、`fee_correction_commission_projection.go:71`；沿用费用自身汇率快照。
- 验收例（含税106/未税100，应付含税53/未税50）：同币种下可由 `total_amount`/`net_amount` 直接按币种求和；本币侧未税需 `CommissionNetBase` 同公式（前端当前无等价实现，grep 未见 net 本币汇总）。
- 前端编辑预览：`orderFeeOptionalColumns.tsx:78-115 computeTaxBreakdown/computeNetUnitPrice`（Decimal，两位 ROUND_HALF_UP，与后端 8 位 RoundBank 口径不同）；编辑行预览（rowPreviews）不进入看板汇总，看板汇总来自 `request` 已落库数据（`:1560-1600`）。
- 多币种现有做法：FinanceSummaryBoard 原币分组并列展示（用 ` / ` 连接）；OrderFeeHeader 只显示本币单值，不做原币分组。

## 4. R5 列宽现状（`OrderFeeTableTabs.tsx` `buildColumns` :925 起；偏好 `feeColumnPreference.ts:55-147`）
| 列 | width | 默认可见 | 备注 |
|---|---|---|---|
| 费用代码 feeCode | 100 | 是 | :937 |
| 费用名称 feeSettingId | 180 | 是 | :949 select |
| 结算单位 | 200 | 是 | :1057 |
| 币种 | 90 | 是 | :1108 |
| 计费单位 | 100 | 是 | :1289 |
| 数量 | 210 | 是 | :1163（含数量来源辅助）|
| 单价 | 110 | 是 | :1133 |
| 不含税单价 | 110 | 是 | optional :179 |
| 总金额 | 120 | 是 | :1326 right |
| 税率(%) | 100 | 是 | optional :224 |
| 税金 | 100 | 是 | optional :248 |
| 不含税总额 | 110 | 是 | optional :279 |
| 汇率 | 120 | 是 | :1375 |
| 折本币金额 | 140 | 是 | optional :310 |
| 录入日期 | 150 | 否 | :1414 |
| 备注 | 140 | 是 | :1422 ellipsis |
| 账单号/关联账单财务进度 | 150/150 | 否 | optional :156,:167，仅启用账单跟踪时存在 |
| 操作 | 120 | 是 | :1433，`fixed:'right'` |
- 默认可见列宽度合计约 2150px（不含 rowSelection 列）；两表均 `scroll={{x:'max-content'}}`，窄屏出现整表横向滚动；仅「操作」列固定右侧（:1434），列冻结偏好由 `orderFeeColumnFreeze` 管理（`feeColumnPreference.ts:13`，另有 `OrderFeeTableTabs.tsx:354` structuralFields 将 option 固定右）。
- 列显隐/排序/冻结偏好存浏览器 localStorage（`feeColumnPreference.ts`，`ui/column-settings/preference.ts:19`）。
- 视觉规范：AGENTS.md「页面容器与宽度规范」——禁止 maxWidth:1440，需 100% 满屏平铺。
- 列宽方案（供审阅，非定论；按目标同时可读、可操作）：名称类（费用名称 200、结算单位 200）；金额类（单价/总金额/税金/不含税总额/折本币统一 120，右对齐）；税率 90；币种 90；汇率 110；数量 190；备注 160（ellipsis）；操作列整表保存后只剩「删除」可缩到约 80，另可考虑把「编辑」入口移除；窄屏沿用 `x:'max-content'`，并让「费用名称」与「操作」保持固定。具体取值需用户/设计确认。

## 相关规范
- `.trellis/spec/server/backend/order-fee-review.md`（审核/补录写入口、整批原子）、`order-fee-correction.md`（费用更正含税率）、`order-lock-and-document-version.md`（人工审核锁与版本）、`finance-bill-currency.md`、`finance-commission-lock.md`；前端 `state-management.md`、`finance-ledger-snapshots.md`。

## 5. 风险与待用户决定（未拍板）
1. 整表保存契约：新增行幂等键放在整表请求级还是每行；响应丢失重试如何识别（现仅 Add 有行级唯一索引，Update 靠版本）；行级错误返回是 message 拼 ID（`BulkOrderFeeError` 现状）还是结构化 details/ proto 字段。
2. 整表内是否允许「删除」行：现有删除为单独 `RemoveFee` 并有确认弹窗，整表保存是否包含删除未在需求中明确。
3. 手工汇率覆盖条件（`orderFeeCurrencySetsMatch`）按「已落库集合」还是「本次提交后集合」判定；父 PRD 要求单侧提交不得绕过依赖完整费用集合的校验，现有函数读事务外数据，整表事务内需重读。
4. 保存一侧后 reload 目前两表都 reload（`:891-892`）；需明确如何保留另一侧草稿与提交基线（与 `editable single` 的单行模式、多行同时编辑的差异）。
5. R3：权限码名称/分组/Requires（系统级 vs 按业务线；是否依赖 `fee.update`）；越权拒绝的错误码（可仿 `ErrOrderFeeExchangeRateOverrideForbidden` 的 403）；普通 `UpdateFee` 现一律拒绝税率，范围是「仅普通 Update 放开」还是同时影响 Add、更正、补录路径；税率覆盖是否写入费用并随后续目录变化保持（`resolveCatalog` 每次保存会覆盖为目录税率）；已建账费用/受保护费用是否允许；与 `BilledFeeFieldTaxRate` 的关系；是否要审计记录税率变化。
6. R4：本币未税需用 `CommissionNetBase`（真实本币未税无持久化列）——是前端用同公式、还是后端补汇总字段/接口；OrderFeeHeader 现为 `Number` 浮点且缺失计 0，与「缺失不能伪装成零」验收冲突需决定改造边界；「看板」指 Header 三卡片还是底部 FinanceSummaryBoard 或两者；编辑中未保存预览是否进入看板。
7. R5：是否取消「编辑」按钮、操作列保留项、未税/税额列是否默认可见、列宽方案是否需随列偏好（用户已存偏好）重置。
8. 费用页没有 15 秒外部变动检测（grep 未命中）：R2 验收引用的「现有版本检测机制」在费用页是否存在需先核实，可能需新建。
