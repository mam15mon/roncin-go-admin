# 账单创建与费用维护流程：技术设计（R8–R11）

对应 [prd.md](./prd.md)。本文只写设计，执行顺序见 implement.md；方案经用户审阅前不启动实施。
父任务规则见 [父 PRD](../11-13-order-fees-finance-workflow/prd.md)，已完成的相邻设计见
[费用台账设计](../11-13-finance-fee-entry-ledger/design.md)。
文中「核实」均为静态读码结论（引用 `文件:行号`），未运行测试或查询数据库；凡会影响实施的核实结论，
实施第 1 步以集成测试复核。

## 0. 已确认决策与工程默认

### 0.1 用户已确认（写入 PRD）

1. 零金额账单无需开票、核销，创建即视为已完成；负数更正账单按有效核销加有效对冲的带符号净额等于账单应收净额
   结清后完成，发票进度仍按需票规则。
2. 建账与同次提交中的发票登记／核销为整体原子，任一失败全部回滚并返回逐项错误。
3. 批量建账按「公司＋方向＋币种＋结算对象」自动分组，每组一张账单，各组独立事务，返回每组成功／失败，不跨组混合。
4. 此前已确认（PRD 既有章节）：免人工确认；锁定由有效发票、有效核销、有效对冲、有效无需发票标记任一触发；
   全部解除后恢复；添加仅限未建账、同公司、同方向、同币种、同结算对象费用；移除只解除关联；部分开票按费用分配。

### 0.2 工程默认（评审时可推翻）

- 账单状态枚举保留 `DRAFT/CONFIRMED/CANCELLED`，新建账单直接写 `CONFIRMED`（见第 2 节）；不重命名枚举。
- 账单费用增删用一个 `UpdateBillFees`（同一请求可同时增加与移除），而不是两个独立接口。
- 费用锁定、费用增删复用并扩展现有谓词 `protectedFinanceFactPredicate`，不另写第二套判定。
- 完成状态、开票进度、结算进度为读取时推导的投影，不落库（见第 5 节）。
- 批量响应一律 HTTP 200，成败看逐组结果；仅请求本身非法（鉴权、参数、公司范围）才整体失败。
- 单张 `CreateBill` 不加组合入口（前端建账只走批量，核实：`web/src` 无 `settlementServiceCreateBill` 业务调用）。

## 1. 现状盘点（含行为核实结果）

### 1.1 账单状态与确认

| 事实 | 位置 |
|---|---|
| 新账单一律 `DRAFT`；`Confirm`、`ConfirmBatch` 写 `CONFIRMED` 并记 `confirmed_at/by` | `biz/finance_bill.go:408`、`:346`；`data/finance_bill_write.go:213`；`data/finance_bill_batch.go:60` |
| 账单更新只允许 `DRAFT`，且费用集合取自原账单行，不支持增删 | `biz/finance_bill.go:254`、`:263-266`；`data/finance_bill_write.go:135`、`:180`（行数与入参不等即 `ErrFinanceBillBatchMismatch`） |
| 更新账户候选也只允许 `DRAFT` | `service/settlement_bill.go:132` |
| 取消允许 `DRAFT/CONFIRMED`，依赖发票/核销/对冲有效关联时拒绝，要求先逐项撤销 | `data/finance_bill_write.go:251`、`:379-430` |
| DB 约束：`confirmation_consistency` 允许 `CONFIRMED` 且 `confirmed_at/by` 同时有值 | `migrations/20260826160000_finance_bills.sql:46-49` |

### 1.2 依赖 `CONFIRMED` 的下游（取消人工确认必须同组处理）

| 下游 | 位置 |
|---|---|
| 发票创建、构建、作废释放 | `biz/finance_invoice.go:342`、`:358`；`data/finance_invoice.go:168`、`:220`、`:287`、`:419` |
| 核销（候选、创建） | `data/finance_verification.go:162`、`:253` |
| 对冲创建与确认、候选 | `biz/finance_netting.go:317`、`:405`；`data/finance_netting.go:194` |
| 提成来源 | `data/finance_commission_calculation.go:57` |
| 账单列表汇总、未结清默认状态、信用额度、逾期 | `data/finance_bill.go:76-80`、`:94`；`data/finance_bill_credit.go:128`、`:199`；`biz/finance_bill.go:538` |
| 工作台、费用更正提成投影 | `data/workbench.go:751`；`data/fee_correction_commission_projection.go:43`（`DRAFT,CONFIRMED`） |
| 开发数据种子 | `cmd/sync-dev/main.go:1414`、`:1596`、`:1718`（DRAFT）、`:1764`、`:1809` |
| 权限与规则 | `access/manifest.go:54`、`:228`；`server/auth.go:261`；`server/access_rules_gen.go:108-109`（生成物） |
| 前端 | `features/finance/bill-status/index.ts:10`；`pages/finance/bills/components/billColumns.tsx:258-274`；`pages/finance/bills/index.tsx:292`；`features/finance/bill-creation/BillCreationResultTable.tsx:114`、`BillCreationWorkbench.tsx:529`；`web/src/access.ts:246` |

结论：把新建账单直接写成 `CONFIRMED`，上述下游判定全部保持有效，不需要逐处改状态判断；删除的只是确认入口。

### 1.3 受保护财务事实

`server/internal/data/fee_protected_fact.go:33-62`：`feeHasProtectedFinanceFact` 为「费用→有效账单」的账单级判定，
有效事实为：发票链接 active 且发票 `DRAFT/ISSUED`；核销分摊 active 且核销 `ACTIVE`；对冲分摊 active 且对冲 `CONFIRMED`。
`protectedFinanceFactPredicate()` 是可复用谓词，文件内已注明「无需发票」标记的扩展点。当前唯一调用方是费用汇率联动
（`data/fee_rate_refresh.go:94`）。**草稿对冲不算保护**，但取消账单时草稿对冲算依赖（`finance_bill_write.go:410`），
两处口径不同。

「有效账单关联」谓词：`data/fee_bill_association.go:13`（活动行且账单未取消）；`finance_bill_lines_active_fee_unique`
（`active=true` 部分唯一索引，`ent/schema/finance_bill_line.go:54`）保证一条费用至多一条有效行。

### 1.4 锁序与并发（核实）

- 账单类写路径统一「订单（含提成父订单闭包，升序）→ 账单行(升序) → 费用行」：`lockBillEconomicOrders`
  （`data/finance_bill_economics.go:16-63`，加锁前展开闭包、锁后复核）；已有账单用 `lockExistingBillOrders`
  （`data/order_fee_review.go:428`，锁后复核行集合未漂移，否则 `ErrFinanceBillVersionConflict`）；新建用
  `lockBillFeeOrders`（`:250`）。
- 发票创建先 `lockExistingBillOrders` 再按主键升序 `ForUpdate` 账单（`data/finance_invoice.go:265`、`:279`）；
  核销按 `lockOrdersForFinanceReversal` 升序锁订单，再锁流水、账单（`data/finance_verification.go:200-213`）；
  对冲确认锁对冲单与账单（`data/finance_netting.go:401-415`，`LockNettingBills`）。
  这些写路径的订单集合都包含账单行所属订单，因此与「增删费用」至少在同一订单行上互斥，再由账单行锁复核。
- 账单更新现状为「订单→账单→费用→账单行」（biz 事务内先 `Get(txCtx)` 加锁，再 `LoadBillableFees`）；
  取消为「订单→账单→账单行→费用」。同一账单被 `FOR UPDATE` 串行，二者不构成环。
- 单张建账不在 biz 事务外读费用后写：`FinanceBillUsecase.Create` 全程在 `WithinTransaction` 内
  （`biz/finance_bill.go:148-196`）。

### 1.5 事务封装可组合性（核实）

发票、核销、流水、对冲仓储均使用 `r.data.WithTx` / `WithinTransaction`
（`data/finance_invoice.go:260`、`data/finance_verification.go:188`、`data/finance_cashflow.go:233`、
`data/finance_netting.go:401`），未发现 `withSQLTx`，因此可在同一 `Transactor.WithinTransaction` 回调内与建账共享事务。
核销用例已有范本 `biz/finance_verification.go:176`。

### 1.6 批量建账现状

- 分组：`group_key = sha256(方向、purpose、结算对象、币种、本位币[、更正请求][、税率][、订单])`
  （`biz/finance_bill_batch.go:90-100`）。组织在 `organization_id` 入参里，一次请求只对一个公司。
- 普通模式一次请求只允许单一收付方向，混入即 `ErrFinanceBillMixedDirection`（`biz/finance_bill_batch.go:83`）。
  与本次「方向是分组维度」的新决策冲突，需放开。
- 一个 `WithinTransaction` 内创建整批，预览令牌覆盖全部叶子，整批成功或整批失败（`biz/finance_bill_batch.go:546-631`；
  `data/finance_bill_batch.go:110-252`）。各账单幂等键已由批次键与 group_key 派生
  （`biz/finance_bill_batch.go:585`、`:718`），可直接支撑「逐组重试」。
- `NETTING` 模式在同一事务生成双方账单和草稿对冲（`data/finance_bill_batch.go:200-238`），本质依赖整批原子。
- 批次头 `finance_bill_batches`：`fee_count/bill_count/total_base_amount` 不可变且 `>0`
  （`ent/schema/finance_bill_batch.go:27-31`、`migrations/20260826205000_finance_bill_batches.sql:16`）。

### 1.7 金额与结清口径（核实）

- 费用总额必须为正（`biz/order_fee.go:846`）；DB 约束 `finance_bills_amount_positive`、`finance_bill_lines_amount_positive`
  要求 `total_amount > 0 AND base_currency_amount > 0`，`tax/net >= 0`
  （`migrations/20260826160000_finance_bills.sql:40-41`、`:81-82`）；核销、对冲要求账单总额为正
  （`biz/finance_verification.go:219`、`biz/finance_netting.go:323`、`:587`）。
  **因此当前系统不可能产生零金额或负数账单**；新决策要求放开 DB 约束与相关校验，且需要明确「负数从哪里来」
  （见第 5.4 节与待确认 1）。
- 结清口径：未核销余额 = 总额 − 有效核销 − 有效对冲，展示侧负值钳零（`data/finance_bill.go:312-322`）；
  SQL 谓词 `billUnsettledPredicate` 为 `total > 核销 + 对冲`（`data/finance_bill_credit.go:23-37`）。
  对零额与负数账单：展示钳零、谓词判「已结清」，与新决策的「带符号净额相等才结清」不一致，需同步改。
- 费用 `CORRECTION_REVERSAL` 用途在代码中已不被费用产生（`FinanceBillPurposeForFee` 恒返回 `NORMAL`，
  `biz/finance_bill_types.go:26`），发票构建仍拒绝该用途账单（`biz/finance_invoice.go:358`）。

### 1.8 前端现状

- 建账工作台 `features/finance/bill-creation`：步骤 0..3，创建走 `settlementServiceCreateBillBatch`，结果页「确认本批」
  走 `settlementServiceConfirmBillBatch`（`BillCreationWorkbench.tsx:529`）。
- 结算账户选择 `BillGroupCard.tsx:53 SettlementAccountSelect`：按叶子 identity 拉 `ListBillSettlementAccountCandidates`，
  无默认账户时只给空选择；表单值在 antd `groups.<groupKey>.*`，账户列表加载 effect 依赖 `identity`。
- 订单页已有「下拉底部快速添加」范式 `pages/orders/components/PartnerQuickAddSelect.tsx` 与公共
  `components/ui/quick-create-modal/QuickCreateModal.tsx`；账户创建接口 `partnerServiceCreatePartnerAccount`
  （`business.partner.account.create`，`server/api/partner/v1/partner.proto:84`）已存在。
- 账单页 `pages/finance/bills`：编辑弹窗 `BillEditModal`（草稿才可编辑，账户候选走
  `ListBillSettlementAccountUpdateCandidates`）、详情抽屉 `BillDetailDrawer` 只读展示行。

### 1.9 相邻事实

- 已建账费用编辑策略 `ValidateBilledFeeUpdate` 仅被测试引用，写路径未调用（全仓 grep），本任务不触碰。
- 权限清单移除键由 `SyncPermissionManifest` 自动删除清单外权限（`data/permission_manifest_sync.go:82`），
  删除 `FinanceBillConfirm` 不需要手写权限迁移。
- 账单币种固定边界规范 `.trellis/spec/server/backend/finance-bill-currency.md` 描述「确认」「草稿更新」，实施后需同步改写。

### 1.10 阶段 0 核对结论（2026-11-14，实施第 1 步）

核对方式：逐路径读码复核锁序，并对开发库做只读查询；未修改产品代码与数据。结论：**3.2 锁序与 7.3 前提均成立，
阶段 A 可继续。**

1. **发票、核销、对冲确认均先锁订单再锁账单**（3.2、3.3 前提成立）：
   - 发票创建/开具：`lockExistingBillOrders`（预读有效账单行所属订单 → `lockBillEconomicOrders` 闭包升序加锁 →
     锁后重读有效行，行数或行→订单映射变化即 `ErrFinanceBillVersionConflict`）→ 开票资料 → 账单 `ForUpdate`
     （`data/finance_invoice.go:265-279`、`:339-352`）。
   - 核销创建：`commissionEconomicAffectedOrderIDs` 预读 → `lockOrdersForFinanceReversal` 升序锁订单 →
     `validateFinanceEconomicOrderSet` 锁后复核订单集合（变化即 `ErrOrderStatusConflict`）→ 流水 → 账单
     （`data/finance_verification.go:196-213`）。
   - 对冲确认：`LockNetting` 在事务内先预读分摊账单的订单闭包并加锁、复核，再锁对冲单；`LockNettingBills` 再次锁订单
     （同事务可重入）后锁账单（`data/finance_netting.go:139-166`、`:292-309`）。
   - 推论：增删费用的种子订单含账单现有有效行订单，与上述路径至少在同一订单行互斥；若增删改变了账单行集合或订单集合，
     后获锁者的锁后复核会以冲突失败（fail-closed），不会在旧集合上写入。
2. **共事务可组合**（1.5 成立）：发票 `Create`、核销 `Create`、流水 `Create`/`transition`、对冲 `Create`/`Confirm` 均走
   `r.data.WithTx` / `WithinTransaction`，`withSQLTx` 仅 `data/auth.go`、`data/password_reset.go` 使用，与财务无关。
3. **核销流水锁序与新建账单不冲突**（7.3 成立）：组事务锁序为「订单闭包 → 费用 → 账户 → 汇率 →（同事务重入订单）→
   流水 → 新账单」；其他核销事务为「订单 → 流水 → 自身账单」。新账单提交前对外不可见，无人竞争其行锁；
   流水均在订单之后加锁，且流水 `transition` 只锁流水行，不构成环。
4. **存量 DRAFT 与回填性**（开发库只读查询）：账单 CONFIRMED 10、DRAFT 2、CANCELLED 1；2 张 DRAFT 均带 `batch_id`，
   可由批次 `created_by` 回填，无需审计兜底，无不可回填项。已执行 `pnpm run migrate:dev` 应用阶段 A 迁移后为
   CONFIRMED 12（`confirmed_by/at` 全部有值）、CANCELLED 1。
5. **补充发现**：DB 约束 `finance_bills_confirmation_consistency` 允许 CONFIRMED 且 `confirmed_at/by` 同时为空，
   但新建与迁移一律成对写入；`finance_bills.status` 列默认值仍为 `DRAFT`，写路径均显式赋值，本期不改列默认值。

## 2. 免确认：状态契约

方案：状态枚举与 CHECK 不变；**创建时直接写 `CONFIRMED`**，`confirmed_at=创建时刻`、`confirmed_by=创建人`
（满足 `finance_bills_confirmation_consistency`）。`DRAFT` 不再产生。

- 删除：`Confirm`、`ConfirmBatch` 用例与仓储方法；`ConfirmBill`、`ConfirmBillBatch` RPC 及其消息、`BillExpectedVersion`；
  `FinanceBillConfirm` 权限；前端确认按钮与「确认本批」。`access_rules_gen.go` 由 `make -C server api` 重新生成。
- 迁移：一条正式 SQL，把存量 `DRAFT` 账单置 `CONFIRMED` 并回填 `confirmed_at = created_at`、`confirmed_by = 最近创建人`
  （账单无 `created_by` 列，取其批次 `created_by`，单张账单取审计记录创建人；取不到时迁移显式失败，不静默填充，
  失败时由人工处理，不清库）。草稿对冲不受影响。这是数据迁移，不是清库，不需要清库授权，但迁移内容需用户在审阅时确认
  （见待确认 3）。
- `UpdateBill` 的前置条件由「必须 `DRAFT`」改为「`CONFIRMED` 且无受保护财务事实」（第 3 节谓词）；
  受保护时只允许不改变金额与汇率快照的字段（见待确认 4）。`ListBillSettlementAccountUpdateCandidates` 同步放开。
- 展示：状态列文案「已确认」改为「正常」（`billStatusMeta` 的 `CONFIRMED`），「草稿」选项从筛选移除；
  对用户暴露的是第 5 节的「开票进度／结算进度／完成」，不再暴露确认概念。
- 开发数据：`cmd/sync-dev` 中创建 `DRAFT` 的种子（`:1718`）改为 `CONFIRMED`，随同组提交。

## 3. 锁定与并发

### 3.1 单一锁定谓词

在 `data/fee_protected_fact.go` 增加账单级函数，供增删费用、账单更新、费用汇率联动共同使用：

```text
billProtectedFinanceFacts(ctx, client, billID) -> []ProtectedFact   // 含类型，用于错误提示
billHasProtectedFinanceFact(ctx, client, billID) -> bool
feeHasProtectedFinanceFact = 费用 -> 有效账单 -> billHasProtectedFinanceFact   // 保留，行为不变
```

`protectedFinanceFactPredicate()` 追加第四项「该账单存在有效无需发票标记」；标记表由发票任务建立（第 9 节）。
提示类型枚举：`INVOICE / VERIFICATION / NETTING / INVOICE_EXEMPTION`。草稿对冲是否算保护见待确认 5。

### 3.2 `UpdateBillFees` 加锁顺序

在 `biz` 的 `WithinTransaction` 内，仓储实现按固定顺序：

1. 无锁预读：账单有效行所属订单、待添加／待移除费用所属订单，取并集 `seeds`（组织范围内）。
2. `lockBillEconomicOrders(seeds)`：展开提成父订单闭包、升序 `ForUpdate`、锁后复核闭包不变（沿用现有函数）。
3. 账单行 `ForUpdate`（组织范围），校验 `status=CONFIRMED`、`version == expected_version`，否则
   `ErrFinanceBillVersionConflict`（409，「已被更新，请刷新后重试」）。
4. 受保护事实复核（3.1）：在账单行锁内、事实查询之后判断，命中即拒绝并返回事实类型。
5. 费用行按主键升序 `ForUpdate`（待添加、待移除、不含未变化的现有行），再取账单行 `ForUpdate`（升序）。
6. 复核（见 4.2）后写入：账单行 `active` 切换、新行插入、账单金额与汇率快照重算、`version+1`、费用 `version+1`。
7. `reconcileBillEconomics`（提成经济重算，沿用现有函数，触发标识 `BILL_FEE_UPDATE` + 新版本号），写审计。

这与账单更新现状「订单→账单→费用→账单行」一致，不引入反向路径。

### 3.3 并发矩阵（实施时逐项用集成测试覆盖）

| 并发 A | 并发 B | 保证 |
|---|---|---|
| 增删费用 | 发票登记 / 核销 / 对冲确认 / 标记无需发票 | 二者都先锁账单行所属订单、再锁账单行；后获锁者在锁内重读，增删看到新关联即拒绝，登记看到费用集合已变则以最新行为准 |
| 增删费用 | 增删费用 | 靠 `version` 与账单行锁串行，后者 409 |
| 账单 X 添加费用 f | 账单 Y 添加同一费用 f | 费用行 `ForUpdate` 后复核「无有效账单行」，后者 `ErrFinanceBillFeeInvalid` |
| 增删费用 | 取消账单 | 同一账单行锁串行；取消成功后增删看到 `CANCELLED` 拒绝 |
| 增删费用 | 费用汇率联动（ETD/ETA） | 联动用同一谓词，锁内复核；未锁定账单上的费用被联动更新后，增删重算以账单自身汇率快照为准，不读费用汇率 |

### 3.4 版本字段

账单已有 `version`（乐观锁）。费用变更成功 `+1`，前端在详情抽屉里据此做外部变动检测（沿用
`.trellis/spec/web/frontend/state-management.md` 的 15 秒轮询与「这个账单已经发生变动，请重新刷新。」）。
不新增版本列。

### 3.5 恢复可编辑

不存储锁定状态；每次判定实时查询有效事实。发票全部作废、核销全部反核销、对冲全部撤销、标记撤销后，谓词自然为假，
恢复增删，历史记录不删除。无需新增「解锁」动作。

## 4. 账单费用增删（R9）

### 4.1 接口

```text
ListBillFeeCandidates  GET  /api/v1/finance/bills/{bill_id}/fee-candidates?keyword&page&page_size
UpdateBillFees         POST /api/v1/finance/bills/{id}/fees:update
    { expected_version, add_fee_ids[], remove_fee_ids[] }   // 至少一项非空，各不超过 200，无重复
```

- 候选服务端过滤并分页：`page>=1`、`1<=page_size<=200`（`biz.ValidListPagination`），关键字覆盖费用代码、费用名称、订单号
  （结算对象已固定，不需要拼音检索伙伴）。条件：同组织、`AccountingRequired`、未被替代、无有效账单行、
  结算方向 = 账单方向（沿用 `entFeeSettlementDirection`）、`SettlementPartyID` 相同、币种与本位币相同、
  `purpose=NORMAL`，并排除待审更正源费用（`pendingCorrectionSourceFees`）。
- 提交校验与候选过滤共用同一个纯函数（`validateBillFeeForAttach`，由 `buildFinanceBill` 的逐项校验提取），保证二者一致。
- 响应返回更新后的完整账单（事务提交后用普通上下文重读）。

### 4.2 校验与重算

- 添加：每条费用在锁内复核归属、方向、结算对象、币种、金额口径、无有效占用、无待审更正；失败整体回滚，返回
  `fee_id` 级别错误元数据。不信任前端候选。
- 移除：必须是本账单当前有效行；仅 `active=false`（不改订单费用，不改历史行）；移除后至少保留一条有效行，
  不允许移空（待确认 2），要释放全部费用请使用取消账单。
- 同一费用在同一请求中既添加又移除，拒绝。
- 重算：账单头与各行保持账单日汇率快照与币种；总额／不含税／税额为有效行之和；本位币金额
  `bill.total × bill.exchange_rate` 按 `RoundBank(8)`，行本位币金额逐行乘汇率，最后一行吸收尾差
  （沿用 `applyBillExchangeRate` 口径，`biz/finance_bill.go:226-237`）；`fee_count` 为有效行数；
  预计开票金额按账单上已存预计币种与汇率重算（`financeBillEstimatedInvoiceSnapshot`）。
  行的 `exchange_rate` 保持账单汇率，不读费用汇率。
- 增加的新行取费用当前金额快照；费用行 `version+1`（与建账一致，使费用页看到变动）。
- 订单审核锁不阻断（与建账一致：`finance-bill-currency.md` 明确锁定后正式费用仍可建账）；不调用解除审核锁的逻辑。
- 重新添加曾移除的费用：生成新的账单行（旧行保留为历史），符合 `active` 部分唯一索引。

### 4.3 错误

沿用 `ErrFinanceBillFeeInvalid`、`ErrFinanceBillFeeMismatch`、`ErrFinanceBillVersionConflict`；新增
`ErrFinanceBillFeesLocked`（409，`FINANCE_BILL_FEES_LOCKED`，元数据 `facts=INVOICE,VERIFICATION,...`）、
`ErrFinanceBillLastFee`（409，禁止移空）。新增 proto 错误原因枚举并同步前端 `errorReasons.generated.ts`。

### 4.4 数据模型影响

- `finance_bills.fee_count`：Ent 字段由 `Immutable()` 改为可变（DB 列与 `fee_count > 0` 检查不变）。
- `finance_bill_lines` 无需改结构；行 `order_fee_id` 现为 `Immutable()` 且可空，不变。
- 无新表。

## 5. 完成状态（R10 展示契约，兼顾零金额与负数更正）

### 5.1 投影字段（`FinanceBill` 新增，推导而非落库）

| 字段 | 取值 |
|---|---|
| `settlement_progress` | `UNSETTLED / PARTIAL / SETTLED` |
| `invoice_progress` | `NOT_REQUIRED / NONE / PARTIAL / FULL` |
| `completed` | bool：`settlement_progress = SETTLED` 且 `invoice_progress ∈ {FULL, NOT_REQUIRED}` |
| `fees_locked` / `fee_lock_facts[]` | 第 3.1 谓词结果与类型，供前端按钮置灰与提示 |
| `invoiced_amount` | 有效发票对本账单的分配额合计（来自发票任务的分配事实） |

### 5.2 计算（均以账单币种、8 位小数、带符号）

```text
net_settled = Σ 有效核销分摊.amount + Σ 有效对冲分摊.amount
settlement_progress:
    total == 0                    -> SETTLED            // 零金额：无需核销
    net_settled == 0              -> UNSETTLED
    net_settled == total          -> SETTLED            // 正数、负数同口径（带符号相等）
    其他                          -> PARTIAL
invoice_progress:
    total == 0                    -> NOT_REQUIRED       // 零金额：无需开票，不依赖标记
    存在有效无需发票标记          -> NOT_REQUIRED
    invoiced == 0                 -> NONE
    invoiced == invoiceable_total -> FULL
    其他                          -> PARTIAL
```

- `invoiceable_total` 取账单需开票总额（现有 `InvoiceableAmount` 或账单总额，口径由发票任务给出；零金额与负数账单的
  需开票额也在发票任务中确定，见 9.2）。
- 「有效」沿用 3.1：发票 `DRAFT/ISSUED` 且链接 active；核销 `ACTIVE`；对冲 `CONFIRMED`。
- 与现有口径的差异必须同组修改：展示侧不再对负数钳零；`billUnsettledPredicate` 改为带符号比较
  `total <> 核销 + 对冲`（零额账单判已结清）；`CalculateOverdueDays` 对零额、负数返回 0；信用额度未结清汇总
  （`data/finance_bill_credit.go:128`）按带符号净额参与。

### 5.3 列表筛选

「开票进度」「结算进度」「是否已完成」作为 `ListBills` 新筛选字段，在 SQL 侧用同一套聚合子查询，与 5.2 同源；
汇总与导出沿用 `onLoad` 快照口径（`finance-ledger-snapshots.md`）。

### 5.4 零金额与负数更正的承载（需要决定的缺口，见待确认 1）

- 放开 DB 约束：`finance_bills_amount_positive`、`finance_bill_lines_amount_positive` 与费用相关校验需调整为允许 `= 0`
  或负数；这是正式 Schema 迁移（先确认再改）。
- 目前没有任何代码路径会产生零额或负数账单（1.7）。零额／负数的**来源**（负数费用、更正对冲、冲销费用、
  组合账单净额）尚未定义；来源未定之前，只做字段与计算的承载设计，不放开创建入口。
- 核销与对冲当前要求账单总额为正（`biz/finance_verification.go:219`、`biz/finance_netting.go:323`），
  负数账单的「有效核销＋对冲净额等于应收净额」需要这两个任务定义带符号分摊（9.3），本任务只消费其结果。

## 6. 批量建账：分组与独立事务（R11）

### 6.1 分组

- 分组键：`公司＋方向＋币种＋结算对象`（组织是当前公司工作区，恒为请求公司）。`group_key` 沿用现有构成
  （方向、purpose、结算对象、币种、本位币，`biz/finance_bill_batch.go:90`），**放开「普通模式单一方向」限制**：
  应收与应付落在不同组，同一请求内可并存。删除 `ErrFinanceBillMixedDirection` 的普通模式分支。
- 税率、订单两个可选拆分维度是否保留见待确认 6；保留时它们是同一分组内的进一步拆分，不改变「不跨组混合」。

### 6.2 预览与令牌

- `PreviewBillBatch` 保留，返回每组 `group_token`（对该组费用 ID＋费用版本＋金额＋叶子配置的摘要）；
  不再返回覆盖整批的单一 `preview_token`。
- 创建时每组带自己的 `group_token`，该组事务内重算并比对；陈旧只让该组失败（`FINANCE_BILL_PREVIEW_STALE`）。

### 6.3 创建流程

```text
CreateBillBatch(org, request):
  1. 请求级校验（权限、公司、参数、重复费用、分页式上限：费用 <= 500、分组 <= 费用数）
  2. 请求哈希与批次幂等键：同键同哈希 -> 返回已有结果（见 6.5）；同键不同哈希 -> 批次冲突
  3. 批次头写入（独立短事务，幂等）
  4. 按 group_key 升序，逐组：
       WithinTransaction {
         锁 -> 重算 -> 校验 group_token -> 建账（直接 CONFIRMED）
         [组合：发票登记、核销，见第 7 节]
         写审计与提成经济重算
       }
       成功 -> 记录账单；失败 -> 记录该组错误，继续下一组
  5. 返回逐组结果
```

- 组与组串行处理，不同时持有跨组锁，因此组间不会产生锁环；每组内部沿用既有锁序
  （订单闭包→费用→账户→汇率设置，`biz/finance_bill.go:184`）。
- 每组失败的原因不影响其他组；失败组的费用保持未建账，可单独重试。

### 6.4 批次头模型

- 批次头只保存批次身份：`id/organization_id/batch_no/idempotency_key/request_hash/grouping/created_by/created_at`。
  去掉 `fee_count/bill_count/total_base_amount/base_currency` 四列（它们在部分成功下没有稳定含义），读取时由
  `finance_bills.batch_id` 聚合得出成功账单数、费用数、本位币合计（按本位币分组，不跨币种相加）。
  这是正式迁移，实施时随 Ent Schema 同步（见 8 节）；工程默认，评审可改为「保留列、成功后更新」。
- 批次幂等：批次键不变；账单幂等键沿用 `financeBillBatchBillKey(batchKey, groupKey)`。

### 6.5 响应与重试

```text
CreateBillBatchResponse {
  success, code, message, trace_id
  data: FinanceBillBatch      // 批次头 + 已成功账单
  results: [BillBatchGroupResult]   // 按 group_key 升序，覆盖请求内每一组
  succeeded_count, failed_count
}
BillBatchGroupResult { group_key, success, bill?, error? }
BillBatchGroupError  { reason, message, metadata, items: [BillBatchItemError] }
BillBatchItemError   { step: BILL|INVOICE|VERIFICATION, ref (费用ID/发票序号/核销序号), reason, message }
```

- 同批次键＋同请求再发：已成功的组按派生幂等键原样返回；失败的组重新尝试（费用若已变化会再次失败并给出新原因）。
- 前端不能把 `200` 当作全部成功，必须读取 `results`；类型安全按 `.trellis/spec/web/frontend/type-safety.md`
  的 wire 形态要求（逐组可缺 `bill` 或 `error`）。
- `NETTING` 模式：整批原子的语义与「各组独立」冲突，见待确认 7。

## 7. 创建时组合发票登记／核销（R11，整体原子）

### 7.1 边界

每个分组可带可选的 `invoice` 与 `verifications` 块；有则与该组建账在**同一个组事务**内完成，任一失败整组回滚、
返回逐项错误（`BillBatchItemError.step`）。不带则只建账。载荷类型由相邻任务定义，本任务只定义信封与编排：

- `invoice`：由发票任务定义的登记输入（号码、日期、金额、费用分配）。因账单 ID 尚不存在，载荷用 **费用 ID**
  标识分配对象（账单行 ID 在事务内才生成）。
- `verifications`：由收付核销任务定义的核销输入（既有流水分配、差额流水）。同样以费用 ID／本组账单为目标。

### 7.2 编排与权限

- 在 `biz` 的 `WithinTransaction` 回调内先建账，再用返回的账单 ID 调发票、核销用例；仓储均走 `WithTx`，自动并入外层事务
  （1.5）。回调错误原样外传，不吞错继续提交；完整响应在事务提交后用普通上下文重读（范本 `biz/finance_bill.go:197`）。
- 逐步鉴权：建账要求 `system.finance.bill.create`；带发票要求 `system.finance.invoice.create`；带核销要求
  `system.finance.verification.create`，带差额流水另要求 `system.finance.cashflow.create`。缺任一权限，整组拒绝，
  不降级为「只建账」。
- 幂等：组合各步的幂等键由 `batch_key + group_key + 步骤序号` 派生，响应丢失后重试回放原结果，同键不同内容拒绝
  （范本 `cashflow-create-idempotency.md`）。

### 7.3 锁序

组事务内：订单闭包→费用→账户→汇率设置（建账，沿用）→ 发票链路（账单行锁在本事务新建，无他人竞争）→ 核销链路
（流水行升序）。新账单在提交前对其他事务不可见，因此组合步骤不引入环；流水行锁序以收付核销任务设计为准
（现状为「订单→流水→账单」，`data/finance_verification.go:200-213`，需与新建账单行兼容，实施第一步以集成测试核实）。

### 7.4 本期不做

不在本任务实现发票分配、税额精度、核销分摊算法；这些由发票、收付核销任务落地，本任务只验收组合原子性与错误结构。

## 8. 结算账户快建（R8）

- 位置：`BillGroupCard.SettlementAccountSelect` 的下拉底部「新增结算账户」，复用
  `components/ui/quick-create-modal` 与 `PartnerQuickAddSelect` 的交互约定；创建调用既有
  `partnerServiceCreatePartnerAccount`，**不新增后端接口**。
- 预填与锁定：结算对象＝当前叶子 `settlementPartyId`；币种＝叶子币种；用途按叶子方向（应收→`RECEIVABLE`，
  应付→`PAYABLE`），均只读；其余字段（名称、户名、开户行、账号、SWIFT）由用户填写；是否设为该方向默认账户由用户勾选。
- 权限：入口仅在 `access.canCreatePartnerAccounts`（`business.partner.account.create`）且当前公司可操作时显示；
  服务端仍以账户创建接口的权限与组织范围为准，账户必须属于当前公司结算对象（`hydrateFinanceBillSettlementAccounts`
  已复核，`data/finance_bill_write.go:316-362`）。
- 回填不丢配置：创建成功后只对**当前叶子**重载候选（递增该叶子的刷新序号，作为 `identity` 的一部分），并只写入该叶子的
  `settlementAccountId`；不重置 `billDate/paymentTermsDays/note/预计开票` 等其他表单值，不触发整批预览重取以外的重置；
  通过 `onConfigurationChange` 轮转预览令牌（账户变化本就属于配置变化）。迟到响应沿用现有 `requestSequenceRef`
  守卫。取消弹窗不创建任何数据。
- 失败：保留弹窗与输入，不选用；账户重名等冲突沿用既有错误提示。
- 抽取：多处复用时放 `features/finance/...` 或 `features/partners`，经 `index.ts` 公开最小导出面，并更新
  `.trellis/spec/web/frontend/capability-navigation.md`；`pnpm run check:architecture` 通过。

## 9. 与发票、核销任务的契约边界

### 9.1 本任务负责

- 账单锁定谓词与 `UpdateBillFees`、完成状态投影的**计算框架**、免确认、批量与组合编排、账户快建。
- 对外提供：`billHasProtectedFinanceFact`、`protectedFinanceFactPredicate()` 的扩展位、`fees_locked` 投影字段。

### 9.2 发票任务负责（本任务依赖其数据契约）

- 「无需发票」标记的数据模型与写接口：标记表（账单、操作者、时间、原因、版本、撤销人／时间），标记与撤销审计，
  与有效发票互斥的并发复核，权限键。本任务只消费「是否存在有效标记」，并在 `protectedFinanceFactPredicate`
  中追加该条件。**标记表的归属待确认（待确认 8）。**
- 发票分配事实：逐账单、逐费用的分配额与有效性，用于 `invoiced_amount` 与 `invoice_progress`。
- 零金额、负数账单的需开票额与是否允许登记发票。

### 9.3 收付核销任务负责

- 核销、对冲分摊的带符号语义（负数账单如何被核销或对冲结清）与撤销事实。
- 组合提交中的差额流水与核销输入、幂等与失败语义。

### 9.4 依赖顺序

免确认＋锁定谓词（本任务）→ 发票／核销契约设计 → 组合编排与完成状态联调。实施中任一方契约变化需回到本文第 5、7 节同步。

## 10. 接口与生成物变更汇总

| 变更 | 文件 |
|---|---|
| 删除 `ConfirmBill`、`ConfirmBillBatch` 及消息、`BillExpectedVersion` | `server/api/finance/v1/settlement.proto` |
| 新增 `ListBillFeeCandidates`、`UpdateBillFees` 及消息 | 同上 |
| `FinanceBill` 增加 `settlement_progress`、`invoice_progress`、`completed`、`fees_locked`、`fee_lock_facts`、`invoiced_amount` | 同上 |
| `ListBillsRequest` 增加进度／完成筛选 | 同上 |
| `BillBatchPreviewGroup` 增加 `group_token`；`PreviewBillBatchResponse` 去掉整批 `preview_token` | 同上 |
| `CreateBillBatchRequest` 组内增加 `group_token`、`invoice`、`verifications`，去掉整批 `preview_token` | 同上 |
| `CreateBillBatchResponse` 增加 `results`、`succeeded_count`、`failed_count` | 同上 |
| 错误原因：`FEES_LOCKED`、`LAST_FEE`、批量组错误结构 | `error_reason.proto` |
| 生成：`make -C server api`、`pnpm run generate:web-client` | `*.pb.go`、`openapi.yaml`、`access_rules_gen.go`、`web/src/services/roncin/*`、`errorReasons.generated.ts`、`enums.generated.ts` |

迁移（新增文件，晚于 `20261109110000`；每项先审阅 SQL 再提交，不含 `IF [NOT] EXISTS`）：

1. 存量 `DRAFT` 账单置 `CONFIRMED`（2 节）。
2. 放开账单与账单行金额检查以承载零额／负数（5.4，来源确认后才执行）。
3. 批次头去掉汇总列（6.4）。
4. `fee_count` 可变只影响 Ent 生成代码，无 DDL。
5. 无需发票标记表由发票任务出迁移，本任务仅依赖其存在。

Ent 生成：`go -C server generate ./...`。迁移后用空库冷启动与已有版本升级两种方式验证。

## 11. 权限键

| 动作 | 权限 | 说明 |
|---|---|---|
| 查看账单、候选 | `system.finance.bill.read` | 不变 |
| 创建（含批量） | `system.finance.bill.create` | 不变 |
| 增删账单费用、修改账单 | `system.finance.bill.update` | 复用既有「编辑账单」，不新增键（待确认 9） |
| 确认账单 | 删除 `system.finance.bill.confirm` | 修改 `manifest.go` 后执行 `pnpm run generate:permission-keys`；清单外权限由迁移同步自动移除 |
| 快建结算账户 | `business.partner.account.create` | 既有键，不新增 |
| 组合登记发票／核销 | `invoice.create`、`verification.create`、`cashflow.create` | 逐步校验，见 7.2 |
| 无需发票标记与撤销 | 由发票任务定义 | 不默认赋予所有财务人员 |

`server/auth.go:261` 的写权限集合去掉 `FinanceBillConfirm`；前端 `access.ts:246 canConfirmFinanceBills` 删除。

## 12. 前端改动

1. **移除确认**：`billColumns.tsx` 确认按钮、`bills/index.tsx confirmBill`、`BillCreationResultTable` 确认入口、
   `BillCreationWorkbench.confirmBatch`、`access.ts` 对应能力；`billStatusMeta` 文案调整；筛选去「草稿」。
2. **账单列表**：新增「开票进度」「结算进度」「已完成」列与筛选；零额／无需发票显示明确标记，不伪装为已开票；
   金额按币种分组，汇总口径不变（`onLoad` 采纳快照）。
3. **账单详情抽屉**：增加「费用维护」区——有权限且未锁定时展示「添加费用」（`ListBillFeeCandidates` 服务端关键字
   检索选择器，支持多选）与每行「移除」；锁定时按钮置灰并显示事实类型（来自 `fee_lock_facts`），提示需先撤销发票／核销／
   对冲／标记；提交调用 `UpdateBillFees`，携带 `expectedVersion`，成功后用 `actionRef.reload()` 刷新列表并重载详情。
   外部变动检测按 `state-management.md`：前台页签 15 秒查询轻量版本，发现变化持续提示并保护输入。
4. **编辑弹窗**：账户候选与可编辑条件改为「未受保护」；受保护时按待确认 4 的结果限制字段。
5. **建账工作台**：
   - 分组配置卡片内「新增结算账户」快建（第 8 节）；
   - 创建后的结果页逐组展示成功／失败与逐项错误，失败组保留其配置并可单独重试；成功组不重复提交；
   - 去掉「确认本批」；批次摘要改为按本位币分组的成功账单合计；
   - 组合发票登记／核销的录入区块随发票／核销任务的表单提供，本任务提供按组的载入位与失败回显。
6. 生成客户端更新后检查 `BillCreationWorkbench*.test.tsx`、`billColumns.test.tsx`、`bill-save.test.tsx`、
   `BillDetailDrawer.test.tsx` 并补新用例；一次编写多个测试文件时按功能拆文件，避免单文件超过 20 秒。

## 13. 风险与取舍

| 风险 | 处理 |
|---|---|
| 取消确认牵动发票／核销／对冲／提成等多处 `CONFIRMED` 判定 | 采用「新建即 `CONFIRMED`」，下游不改；存量 `DRAFT` 用迁移转换并显式失败于无法回填创建人 |
| 增删费用与发票／核销并发 | 同一订单行锁互斥＋账单行锁内复核谓词；矩阵 3.3 全部写成集成测试 |
| 零额／负数账单放开多处「必须为正」假设 | 来源未定前只做承载设计，不开放创建入口；改约束与改谓词同组并回归核销、对冲、信用、逾期 |
| 批量由整批原子改为逐组提交 | 组间不共享锁；幂等键逐组派生；响应逐组结果，前端不得以 200 判成功 |
| 批次头汇总列删除 | 读取时聚合，按本位币分组不跨币种相加；迁移与 Ent 同步 |
| `NETTING` 与逐组独立冲突 | 待确认 7，未决前不改动对冲模式 |
| 组合步骤依赖两个未完成任务的契约 | 本任务只做信封、编排、原子与错误结构；契约变化回到第 7 节 |
| 草稿对冲在增删费用后金额漂移 | 待确认 5 |

## 14. 不做的事

不恢复人工确认或「回退到草稿」；不做自动发送邮件；不让增删费用修改或删除订单费用；不复制参考系统的名称匹配或状态码；
不新增通用流程引擎；不在本任务实现发票分配、核销算法；不同步筛选偏好到服务端；不恢复业务标签。

## 15. 已确认结论（2026-11-14）

用户确认（第 1、7、8、9 项）：

1. **零额与负数账单**：本期只做判定逻辑（完成状态、未结清按带符号净额计算），不放开创建入口，不允许负数费用，不改 `finance_bills_amount_positive` 等约束；有真实更正场景时另起任务。
7. **对冲建账（NETTING）**：保留，对冲模式整批原子；普通模式按分组逐组独立事务。
8. **「无需发票」标记**：数据表、写接口、权限键、互斥校验与页面归发票任务；本任务只在受保护谓词与完成状态中保留扩展位。
9. **费用增删权限**：复用 `system.finance.bill.update`，不新增权限码。

主会话按设计建议定为默认（用户可在评审时推翻）：

2. 禁止移除最后一条费用，释放全部费用须取消账单。
3. 存量 `DRAFT` 账单迁移为 `CONFIRMED`，确认人/确认时间按创建人/创建时间回填（与免确认后「创建者即确认者」语义一致），迁移 SQL 中显式写明；不清库。
4. 未受保护：可改账单日期、账户、账期、预计开票、备注；受保护后只允许备注、对账标题、到期日，禁止改账户与任何触及金额/汇率的字段。
5. 草稿对冲视为阻止费用增删。
6. 「按税率拆分／按订单拆分」保留为同组内可选细分，默认关闭。
10. 完成状态读取时按当前金额推导，不冻结。

## 阶段 F 最终信封与编排（2026-11-14）

每组新增可选 `invoices[]` 与 `verifications[]`。发票记录使用号码、日期、类型、抬头ID、总额、备注和本组源费用分配（费用ID、金额、税额）；不接受尚未存在的账单ID或版本。核销记录使用核销日、备注和逐资金ID/金额分配，可附差额流水交易资料；空资金ID指本记录附带的新流水。流水方向、结算对象、币种及公司一律由本组新账单填充，不给用户猜ID或更改归属。

组事务在建账写入后先按ID升序预锁本原子单元全部已有资金：NORMAL为当前组，NETTING为整批，避免多条核销或跨组依次取得不同资金锁造成环。新流水在提交前不可见。随后发票整组一次 `RegisterWithinTransaction`（全部共享新账单起始version=1，由登记命令统一递增一次），再逐条核销。核销公开 `CreateWithinTransaction` / `CreateWithCashflowWithinTransaction` 只写或回放ID，不在外层事务内读取完整响应。原独立入口保留事务提交后重读。

组合依赖由Service将既有invoice/verification/cashflow usecase显式传给Bill用例，不新增构造器循环；权限范围由Service逐动作解析为本公司授权位，Biz在各组事务前执行整组门禁。带发票要求invoice.create；带核销要求verification.create；带差额要求cashflow.create及cashflow.update（确认）；显式差额汇率另要求exchange_rate.update。缺任一步权限整组拒绝，不退化为只建账；NORMAL其他组继续。

主请求摘要包括所有规范化附带意图；序号与其顺序有意义，费用/资金集合内部规范排序。步骤键由批次键、组键、步骤和序号派生，发票命令为整组一个键。已成功组按已存在账单直接回放，不重放附带写入；失败组无孤立账单、发票、核销、流水、审计或编号增量，下一次原键可重试。

NORMAL逐组错误新增 `items[]`，字段为step（BILL/INVOICE/VERIFICATION）、1起记录序号（0表示整步骤）、reason和message。发票用例提供的0起invoice_index转换为业务序号。NETTING维持整批原子及现行HTTP失败语义，错误公开metadata补step/item_index/group_key；只公开这三个定位键及既有白名单，不暴露内部错误。成功完整账单和批次响应均提交后按普通上下文重读。

请求级结构错误（UUID/decimal无法解析、重复组/费用等）仍整请求拒绝；事务内的登记/分配余额、权限、号码冲突等业务错误归对应组和步骤。金额/税额计算和发票表单复用发票任务共享能力；不复制其计算或引入新手续费算法。
