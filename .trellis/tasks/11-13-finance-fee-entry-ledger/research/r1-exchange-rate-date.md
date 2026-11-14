# Research: R1 费用汇率按出口 ETD／进口 ETA 取值并在日期保存时联动

- **Query**: R1 现有代码与规范调研（调用链、日期入口、财务谓词、锁序、规范、风险）
- **Scope**: internal（只读）
- **Date**: 2026-11-14
- 路径前缀：`server/internal/`；规范前缀：`.trellis/spec/server/backend/`

## 1. 取汇率调用链
- 新增：`biz/order_fee.go` Create 在 :255-265 先 `resolveCatalog`→`repo.List`→`resolveExchangeRate`→`calculateAmounts`，再 `repo.Add`（data/order_fee.go:385 写来源）。
- 更新：`Update` :316-358。`current.HasActiveBill` 直接 `ErrOrderFeeBillOccupied`（:324）；未改币种/方向且无手工覆盖时原样沿用旧快照（:339-340，不重取）；否则 `resolveExchangeRate`。
- `resolveExchangeRate` :522-554：日期恒取 `orderETDDay`（:483）→`repo.OrderETDDate`（data/order_fee.go:149-162，只读 `orders.etd` 字符串）→`orderEtdBusinessDay`（:473，RFC3339 转业务时区自然日；失败 `ErrOrderFeeEtdMissing` ORDER_ETD_MISSING，:27）。**当前完全不读 ETA，也不看 trade_direction**。
- 手工覆盖（:527-543）：需 `canOverrideExchangeRate` 权限+`orderFeeCurrencySetsMatch`（:493）+本币必须为 1，写 `ExchangeRateSource=MANUAL`、`SettingID=nil`、日期=ETD 日。
- 系统取值：`ExchangeRateUsecase.ResolveRate`（biz/exchange_rate.go:301-313），只查目标自然周公司行；本币=费用币种直接 1/`SYSTEM`（:312）；否则 `repo.ResolveRate`，来源 `WEEKLY`/`BOC_SYNC`，方向 RECEIVABLE→ar_rate、PAYABLE→ap_rate。
- 来源枚举 biz/exchange_rate.go:54-59；注释称 INHERITED_LAST_WEEK 与 DERIVED 已退役（:51）。ent 枚举六值保留（ent/schema/order_fee.go:49）。
- 区分系统/手工：只有 `MANUAL` 是手工；但同币种恒等取值是 `SYSTEM`（无 setting_id），`WEEKLY`/`BOC_SYNC` 带 `exchange_rate_setting_id`。费用无独立「是否手工」字段，判定只能看 source。历史 `DERIVED`/`INHERITED_LAST_WEEK` 快照归类需用户定（见 7）。
- 订单字段：`orders.etd/eta`（字符串，data/order_convert.go:60）；方向 `orders.trade_direction`（`biz/order_types.go:112-116` export/import）；品类 `business_type` SE/SI/AE/AI/LAND/RAIL。全仓无按方向选 ETD/ETA 的现成函数。
- 视图层覆盖：AE 读取时 `o.ETD/ETA` 被主单覆盖（data/air_export.go:180-181）；SE 摘要取 TE（data/order_convert.go:154、sea_master_bill.go:250）。费用取值读的是 `orders` 表列，不是这些覆盖值。

## 2. 影响 ETD/ETA 的真实写入口（均未触碰费用）
| 入口 | 位置 | 事务/锁 |
|---|---|---|
| 订单资料保存 UpdateDraft（SetEtd/SetEta） | data/order_write_draft.go:267-268；锁 :38-83、:171 起 | `WithTx`；默认仅 `Order FOR UPDATE` 单行；若 `orderUpdateNeedsSharedMasterBillMemberLocks`（涉及 MBL 身份/航程/正文）则锁全部活动成员（成员 ID 升序）+MBL+Link，锁后重验成员集合（:140-165）；集合变化 `SEA_DOCUMENT_STRUCTURE_CONFLICT`。该函数全程不碰费用。 |
| 订单创建 | data/order_write.go:86-87 | 新单无费用，不涉及联动 |
| SE 订单保存内改 MBL/TE（单票 activeCount==1） | data/order_write_sea_master_bill.go:443-451、546-554 | 写 TE.etd/eta 版本+1；多票共享（activeCount>1）仅做冲突检测（:462-483），不写 TE |
| 共享航程更新 ExecuteTransportExecutionUpdate | data/sea_order_change_transport_update.go:105-246（写 :207/:212） | 顺序：成员 Order 升序 FOR UPDATE（:121）→逐单 `ensureOrderBusinessEditable`（:128，任一成员锁定/终止即整体回滚）→MBL 升序→Link→TE→重验 Link 集合→写 TE+版本+审计。**只写 TE，没有 SetEtd 到成员 `orders`** |
| AE 主单更新 airExportRepo.Update | data/air_export.go:385-427（SetEtd/SetEta :406） | `lockAirMasterMembers`（:34-70：成员 Order 升序 ForUpdate→Master→重验 Link）+`ensureAirMembersEditable`。**不写成员 `orders.etd/eta`**；AE 订单保存时才由 air_order_sync.go:59-60 从主单回灌 input.ETD/ETA |
| 改配/拆票/恢复 | sea_order_change_reassignment.go:794-800；sea_order_change_split_execute.go:1206-1212、1270-1283、1517-1523 | 改配/拆票把目标 TE 的 etd/eta 写入订单列；各自既有事务。是否在本期联动范围内需决定 |
- 事实要点：现有代码里「共享航程/主单」改日期不会同步 `orders.etd/eta`，而费用读的是 `orders.etd`。若 R1 要求共享改期联动多票费用，需先确定权威日期源与同步（见 7-Q2）。

## 3. 财务状态谓词
- 有效账单关联：`data/fee_bill_association.go:14-26` `effectiveBillLinePredicate`（line.active 且 bill.status≠CANCELLED，含 DRAFT）与预加载 `effectiveBillLineFilter`；映射成 `OrderFee.HasActiveBill`（data/order_fee.go:887，biz/order_fee.go:60-62）。
- `HasActiveBill` 使用点：biz/order_fee.go:324（Update 拒绝）；data/order_fee.go:429（锁内 `FinanceBillLines>0`→`ErrOrderFeeBillOccupied`）、:551；Remove 锁内再查（data/order_fee.go:575-590）；建账 biz/finance_bill.go:339/398/417、finance_bill_batch.go:69；更正 order_fee_correction_usecase.go:70/172；review.go:62。
- `ValidateBilledFeeUpdate`（biz/order_fee.go:362-396）**在 biz/data/service 非测试代码中无调用点**（grep 仅定义；`BilledFeeEditPolicy` 在 finance_custom_setting.go:31-102 存取）。当前 Update 路径对已建账一律 `ErrOrderFeeBillOccupied`（biz :324，data :429），该策略暂未接入写路径。注释说「数据层会在锁内再次调用」与现状不符。
- 发票/核销/对冲是**账单级**事实：`FinanceInvoiceBill`（invoice↔bill，invoice 状态 DRAFT/ISSUED/CANCELLED/RED_FLUSHED）、`FinanceVerification`（ACTIVE/REVERSED）+allocations、`FinanceNetting`（DRAFT/CONFIRMED/CANCELLED/REVERSED）+`FinanceNettingAllocation`（bill_id、source_fee_id）。**未找到现成的「费用是否有任何有效发票/核销/对冲」谓词**；财务进度七状态为只读投影（order-fee-review.md:5「分批建账与七状态」，口径见 finance-bill-currency.md:146 起），代码中未检索到单一可复用函数（仅按名称 grep 未命中 BillProgress/PaymentProgress）。
- 「无需发票」标记：全仓（Go/proto/迁移/spec）未检索到任何实现或字段，属新建；PRD 要求操作者与历史记录。
- 其他已存在保护：`item.SupersededByRequestID != nil || EconomicEffect==DECREASE`→`ErrFeeCorrectionProtected`（data/order_fee.go:433；`biz.IsCurrentFee` order_fee_correction.go:37）。

## 4. 人工审核锁与规范约束
- 费用写门禁 `lockOrderForFeeMutation`（data/order_fee.go:245-254）：`lockOrdersForFeeEconomics`（order_fee_supplement_lock.go:89-125：按提成经济集合扩展订单 ID，升序 ForUpdate，锁后复核集合）→`order.LockedAt!=nil`→`ErrOrderAlreadyLocked`→`ensureOrderBusinessEditable`。即普通费用写在审核锁定订单上被拒，只能走补录补审。
- order-lock-and-document-version.md：L46-52/L70 人工统一审核锁后「普通订单资料与费用修改均拒绝」，内容门禁为 `ensureOrderBusinessContentEditable`（先生命周期后业务锁，禁第二套门禁）；L84、L100 共享 MBL 写入任一活动成员已锁→`SEA_MASTER_BILL_MEMBER_ORDER_LOCKED`；L103-104/L170 成员 Order 升序一次性锁，再 MBL→Link→TE，锁后重验成员集合，禁锁后补锁。AE 固定 Order→MAWB→Link→HAWB（L64）。
- 因此 spec 现状：共享航程更新要求**所有成员均未锁**才可执行；已审核锁订单的费用不被日期联动改写不会成为问题（整体先被门禁拒绝）。「自动更新是否可绕过审核锁」在 spec 中未规定，PRD 明说「不默认绕过」。
- order-fee-review.md:L29-35/L66：改费要求 ACTIVE、未锁、无有效账单关联；锁后改费一律补录补审；锁事务序 User→Membership→Role→排序完整 Order 集合→子资源。结清/建账/核销/对冲不自动锁。
- order-fee-correction.md:L9/L17/L48：REPLACE/VOID 源费须无有效账单；锁序 授权父行→排序后完整经济关联 Order 集合→申请/费用及提成子资源，锁内重验集合变化→409；账单创建/申请、批准/回退等需在共同权威锁内串行。
- finance-bill-currency.md:L53-58、L98：建账序「订单（升序）→费用→账户→汇率设置；需要费用行锁时最后锁费用」，草稿/确认/取消「订单→账单→账单行→费用」，禁止费用→订单反向。
- exchange-rate-single-rate.md:L33-37：订单费用统一按订单 **ETD** 所在自然周（该规范仍写 ETD，与 R1 的 ETA 进口口径冲突需随任务更新）；手工覆盖权限 `system.finance.order_fee.exchange_rate.override` 与币种集合一致性；已固化快照不随配置重算（L69 矩阵）；事务内汇率读取走 `Data.client(ctx)`+`ForShare()`（L18）。

## 5. 缺汇率与同币种
- 缺目标周：`ErrExchangeRateMissing`（biz/exchange_rate.go:21，`FEE_EXCHANGE_RATE_MISSING`，400，文案「汇率日期未命中生效汇率，请先维护汇率」）；同周多行 `ErrExchangeRateConflict`。错误不含币种/周/方向元数据，PRD 要求「提示缺失币种、目标周和方向」，现有类型没有 metadata。
- 缺 ETD：`ORDER_ETD_MISSING`（biz/order_fee.go:27，文案写「ETD」，进口需 ETA 版本）。
- 同币种：本币直接 1/SYSTEM（biz/exchange_rate.go:311-313，不查周表，故不会缺失）；手工本币需为 1（`ErrExchangeRateSameCurrencyInvalid`，order_fee.go:536）。
- 多币种约束：手工覆盖要求应收币种集合=应付币种集合且非空（`orderFeeCurrencySetsMatch` order_fee.go:493-520，输入含待提交费用）。日期联动只重取系统汇率，应不触发该集合检查，但这是否算「同币种等值汇率」口径需确认（Q6）。
- 汇率精度 4 位小数，存储 numeric(18,8)（exchange-rate-single-rate.md:L16）；折本币 8 位 RoundBank（calculateAmounts order_fee.go:403 起）。

## 6. 锁序与原子性（事实与现有约束推出的约束，非实现）
- 现有统一入口：`Data.WithTx`、嵌套自动合并；跨仓储用 `Transactor.WithinTransaction`（AGENTS.md 事务规范）。禁手写 Begin/Rollback。
- 费用更新锁序已有范式：`lockOrdersForFeeEconomics`（订单集合升序）→费用行（data/order_fee.go:245/427）。订单保存与共享航程写路径均已先锁订单集合（升序）。若联动放进这些事务，应沿用「成员 Order 升序→MBL/Master→Link→TE→（新增）受影响费用按 ID 升序 ForUpdate」，不得出现费用→订单。
- 联动事务内需做（供设计参考）：锁后重读 ETD/ETA 与方向；重读费用有效账单/发票/核销/对冲/无需发票/来源（锁内最新事实，PRD 明示）；先收集全部需更新费用与其目标周汇率，缺任一即整体返回错误（日期与费用都不写）；费用行 version+1、`orders.fee_content_version` 递增（data/order_fee.go:520 做法）、`reconcileOrdinaryFeeEconomics`（fee_correction_commission.go:344）并写审计；`ForShare` 读汇率设置。
- 账单级并发：开票/核销/对冲/建账路径锁序为订单→账单→账单行→费用；联动在持有成员订单锁后再锁费用行，与其一致。发票/核销/对冲是否在锁订单前就能提交（不先锁 Order）需在设计中确认，否则无法靠订单锁串行（未验证）。

## 7. 风险与待用户决定
1. 日期源：现有费用只读 `orders.etd`。进口取 `orders.eta` 后，旧 ETD 逻辑、ORDER_ETD_MISSING 文案、exchange-rate-single-rate.md L33 需同步；`trade_direction` 为 export/import，AE/AI/LAND/RAIL 的方向判定是否都用该字段？内贸水运/陆运/铁路是否也按 ETD/ETA（参考系统有 ETD 分支，未确认）？
2. 共享改期不同步成员 `orders.etd/eta`（见 §2）：联动的权威日期是 TE/Master 还是订单列？是否要在同一事务把成员订单列同步？（否则联动读到旧值）
3. 「有效发票/核销/对冲」判定为账单级事实，一票多账单、合并账单时按行/按单？部分发生、对冲 DRAFT 是否算「有效」？需明确状态集合（invoice ISSUED/DRAFT？netting DRAFT/CONFIRMED？verification ACTIVE）。
4. 「手工」判定：仅 `MANUAL`？历史 `DERIVED`/`INHERITED_LAST_WEEK` 快照（枚举保留，注释称退役）按系统可更新还是保留？同币种 `SYSTEM` 恒等值是否算系统且参与联动（联动无需取汇率）？
5. 联动入口范围：订单保存、共享 TE 更新、AE 主单更新、单票 SE 内联 MBL/TE 改期、改配/拆票/恢复/订单创建是否都算？
6. 「已建账但无关联」的系统费用被自动改汇率，而 `Update` 仍对已建账一律拒绝：是否沿用「仅汇率快照联动，其他字段仍受限」？账单行折本币/账单汇率不变（PRD 已定）；提成/利润重算（`reconcileOrdinaryFeeEconomics`）对已确认/已支付提成的处理需设计核实（finance-commission-lock.md）。
7. 审核锁：成员已业务锁定时共享改期现已整体被拒；单票订单保存也被拒。是否维持（即不新增绕过）？已锁订单费用不联动是否需明示？
8. 缺汇率错误需带「缺失币种/目标周/方向」元数据，现有 `ErrExchangeRateMissing` 无元数据，是否新增专用错误或扩展 metadata（契约/前端展示联动）？
9. 「无需发票」标记新建：存放层级（费用/账单）、权限码、历史表、撤销条件；与账单费用增删禁止（PRD L123）联动，范围较大。
10. 并发：联动与开票/核销/对冲/手工改价的序列化点（见 §6 末）及费用 `version` 变化对前端 15 秒变动检测的影响（预期通过 `fee_content_version` 递增提示）。

## Caveats
- `ValidateBilledFeeUpdate` 无调用点、无需发票字段不存在、费用级「财务关联」谓词不存在，均基于 grep 结果，未做全仓动态验证。
- 未逐一读取改配/拆票/恢复路径的事务细节；未读 finance_bill 并发建账实现与 proto。
