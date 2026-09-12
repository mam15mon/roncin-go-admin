# 对冲提成通道与遗留修补 Technical Design

## 1. Architecture & Design Principles

1. **来源二选一，不做多态抽象**：`finance_commissions` 同时持有可空 `verification_id`
   与可空 `netting_id`（各带快照单号列），服务层校验恰好一个非空。两个显式 FK 保留
   参照完整性，查询与冲减路由按来源字段直连，不引入 `source_type/source_id` 通用模型。
2. **计算管线复用**：已实现收入聚合入口从「核销分摊」推广为「结算分摊」——核销
   (ACTIVE, RECEIVABLE) 与对冲 (CONFIRMED, RECEIVABLE) 同构地按
   `分摊金额/账单总额 × 账单行本位币` 摊入 orderRealized；成本分摊、规则、CNY 换算、
   版本指纹全部复用既有实现，不为对冲另起一套。
3. **Clawback 对称**：对冲 REVERSE 复用核销 REVERSE 的提成冲减计划
   （`PlanCommissionReversal`：无支付敞口取消、有则 CONFIRMED 冲减），仅来源路由不同；
   费用财务锁的净额判定（09-11 规范）来源无关，自动生效。

## 2. Core Modules & Data Models

### 2.1 Schema 与迁移（Ent 真相源 + 手写迁移）

`server/internal/data/ent/schema/finance_commission.go`：
- `verification_id` → `Optional().Nillable()`，`verification_no` → `Optional().MaxLen(64)`；
  edge `verification` 去掉 `Required()`；
- 新增 `netting_id uuid Optional().Nillable().Immutable()`、`netting_no Optional().MaxLen(64).Immutable()`；
  edge `netting`（FinanceNetting.Ref("commissions")，Unique，非 Required）；
- 既有索引调整：
  - `(organization_id, verification_id, employee_id, personnel_role)` 部分索引加
    `WHERE verification_id IS NOT NULL`（防 NULL 重复），新增对冲侧同构部分索引
    `(organization_id, netting_id, employee_id, personnel_role) WHERE netting_id IS NOT NULL`；
  - 活跃去重唯一约束（20260910150000 引入的 target active unique）按来源拆分为两个
    部分唯一索引，语义：同来源同员工同角色仅一条非终态提成。

迁移（`server/migrations/<ts>_commission_netting_source.sql`）：
`ALTER verification_id/no DROP NOT NULL` → 加 `netting_id/netting_no` 列与 FK → 重建
上述部分索引/唯一约束 → 存量核销行不受影响。

### 2.2 biz 层（finance_commission.go / finance_netting.go）

- `CreateCommissionInput` / 预览输入：`VerificationID` 与 `NettingID` 均可空，用例层校验
  恰好一个非空，两个都空/都填返回 `ErrCommissionInvalidArgument`（新增或复用）；
- 计算入口：既有按 verification 加载分摊的仓储方法旁增加按 netting 加载
  （CONFIRMED 对冲单 → RECEIVABLE allocations → 关联 bill），进入同一
  `commissionCalculationStore`；`orderRealized` 聚合对两种来源统一：
  `billLine.BaseCurrencyAmount × (allocation.Amount / bill.TotalAmount)`；
- 指纹（fingerprint）加入来源标识段（`src=netting|<id>`），陈旧检测不受影响；
- `finance_netting.go` Reverse：事务内对来源为该 netting 的提成调用与核销 Reverse 相同的
  冲减计划落库逻辑（抽公共助手，两处复用）。

### 2.3 data 层（finance_commission.go / finance_netting.go）

- 读写映射补两新字段（空来源 NULL 化）；
- 候选：新增 `ListNettingCandidates`（CONFIRMED、存在 RECEIVABLE 分摊、按创建时间倒序
  分页），复用分页公共校验；
- 创建事务：锁序沿用既有（netting 行 `ForShare`→`ForUpdate` 依既有模式，多行按主键序），
  幂等键唯一冲突走 `ent.IsConstraintError` 映射。

### 2.4 API（server/api/finance/v1/settlement.proto）

- `PreviewCommissionRequest` / `CreateCommissionRequest`：`verification_id` 改
  `optional`，新增 `optional string netting_id`；service 层二选一校验；
- 新增 `ListCommissionNettingCandidates`（GET
  `/api/v1/finance/commissions/netting-candidates`，权限
  `system.finance.commission.manage` + `DATA_SCOPE_ORGANIZATION`），响应为 netting 形态；
- `CommissionCalculation` / `FinanceCommission` 消息：`verification_id/no` 改 optional，
  新增 `netting_id/netting_no`；`make api` 重生成，`generate:web-client` 同步。

## 2.5 拆票与操作门禁（sea_order_change.go）

- `PreviewSplit` 在 `GetSplitContext` 取得来源订单后，执行
  `ensureOrderBusinessContentEditable`（复用 `orderBusinessEditBlockReason` 提取文案）；
  未通过 → 向 `preview.ValidationErrors` 追加
  `SeaOrderSplitValidationError{Reason: "ORDER_GATE", Message: "订单 <no> <原因>"}`，
  预览照常返回（不 500）；Execute 既有 409 语义不变。
- `GetChangeActions` 在检查生命周期之后，执行 `ensureOrderBusinessContentEditable`
  （或直接检查 `order.BusinessLockStatus == LOCKED`）；若被锁定，将 `CanSplit = false`
  与 `CanReassign = false`，并在 `SplitBlockedReasons` 与 `ReassignBlockedReasons`
  中记录 `"订单 " + order.OrderNo + " " + orderBusinessEditBlockReason(err)`，使前端按钮
  即时禁用并提示锁定原因。

## 2.6 非 CNY 本币提成 CNY 汇率解析与换算修正（finance_commission.go）

- **汇率解析路由**：提成预览与创建时，当 `generation.BaseCurrency != cnyCurrency` 时，
  调用 `u.exchangeRate.ResolveRate(ctx, org, generation.BaseCurrency, generation.CommissionDate)`；
  底层 `ResolveContext` 取得总部上下文（BaseCurrency=CNY），在总部查询
  `From: baseCurrency, To: CNY` 的基准汇率（如 `USD → CNY` = 7.2），彻底解决去向总部反查
  不存在的 `CNY → USD` 报 `ErrExchangeRateMissing` 的问题。
- **金额换算函数**：`ResolveCommissionCNYRate(baseCurrency, date, resolvedRate)`
  在 `baseCurrency != cnyCurrency` 时，直接采用 `resolvedRate` 作为快照汇率
  （不再做 `1 / resolvedRate` 倒数），`ApplyCommissionAmount` 保持 `commissionAmount * exchangeRate`
  相乘，得到符合财务定义的正确 CNY 提成总额。

## 2.7 admin 集成测试迁移

- `TestAdminEmployeeLifecyclePostgres`（及其私有夹具）从 `newIntegrationData` +
  `AutoMigrate` 直连 public schema 模式迁到 `getIntegrationData(t)` 隔离 schema 模式；
  断言行为不变；夹具组织/根组织构造适配隔离 schema 语义（单一根组织）。

## 3. Frontend

- 提成页（`web/src/pages/finance/commissions*`）：来源 Tab「核销提成 / 对冲提成」；
  对冲 Tab 列 CONFIRMED 对冲单（单号/单位/币种/金额/确认时间），选中后走既有
  员工/规则候选 → 预览 → 创建流程（请求体带 `nettingId`）；
- 提成列表/详情：来源列显示核销单号或对冲单号（二选一非空）；
- 生成客户端后消费新字段，旧 `verificationId` 必填引用全部收敛。

## 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| Preview/Create 同时缺 verification_id 与 netting_id，或同时提供 | 400 参数错误 |
| netting 非 CONFIRMED / 无 RECEIVABLE 分摊 | 400 候选无效 |
| 同 (netting, employee, personnel_role) 已有非终态提成 | 409 重复 |
| 对冲 REVERSE 时其提成未支付 | 取消（CANCELLED） |
| 对冲 REVERSE 时其提成有 PAID/CONFIRMED 敞口 | 生成 CONFIRMED 冲减调整 |
| 拆票预览遇锁定/终态订单 | ValidationErrors 含原因，正常返回 |
| change-actions 遇业务锁定订单 | CanSplit=false, CanReassign=false，含锁定原因 |
| 非 CNY 本币组织计提提成 | 按 baseCurrency→CNY 正向汇率换算，不报错、不颠倒 |
