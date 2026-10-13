# 提成净额财务锁契约

## 1. Scope / Trigger

适用于订单费用的财务锁判定（`ORDER_FEE_FINANCE_LOCKED`）、费用台账 `finance_locked` 投影
与提成来源模型。修改提成/调整单状态机、核销或对冲冲减链路、提成来源字段或费用写入口
锁判定时必须遵循。

## 2. Signatures

```go
// internal/data/settlement.go —— SQL 谓词与 Go 口径必须同步维护（注释互相锚定）
financeLockedOrderPredicate() predicate.Order      // 净额 > 0 才锁（SQL 子查询）
financeCommissionLockNetAmount(lines, adjustments) // 同一公式的 Go 投影口径
```

```text
net(order) = Σ 提成线金额(所属提成单 status ∈ {CONFIRMED, PAID})
           + Σ 符号化调整单金额(status ∈ {CONFIRMED, PAID}，DECREASE 记负)
锁定条件: net(order) > 0

提成来源（finance_commissions）：verification_id 与 netting_id 恰好一个非空
（服务/用例层 XOR 校验）；活跃去重 = 两个部分唯一索引按来源各自生效
（WHERE 来源非空 AND status <> 'CANCELLED'）
```

## 3. Contracts

- **来源二选一**：提成单来源为核销单或对冲单之一，不做多态来源抽象；同幂等键换来源必返
  重复错误（幂等意图比对含两个来源字段）；计算指纹含 `src=` 来源段。对冲来源计算口径与
  核销同构：普通 CONFIRMED 对冲单的真实收入贡献经统一账单行清偿投影计算；费用更正内部对冲不新增收入。详细根链、退款及配额规则见 [费用更正契约](./order-fee-correction.md)。
- 需要重算员工快照的创建、确认与月度批准，先以 `NO KEY UPDATE` 锁员工 User（仅稳定非主键快照，同时允许方案分配外键 `KEY SHARE`），再锁申请头（存在时）及订单；月申提交/重提先 User→Membership，再申请头/订单。`GetGenerationContext` 接收 employeeID，在事务入口即遵守此序，不能在外层已锁 Order 后才靠内核补锁员工。
- 单笔确认以完整来源订单集合预锁；整批确认与月申提交/重提须先合并所有来源订单，再按主键一次锁定联合集合。不得先锁员工归属的提成行子集或逐来源追加锁，否则不同员工/来源之间仍会形成反序。
- 对冲计提预锁订单只从活跃 `RECEIVABLE` 分摊收集，与实际计提集合一致；应付在其他订单时不能误报 `FINANCE_COMMISSION_SOURCE_CHANGED`。预览、创建、确认及月度申请使用相同经济来源口径。
- **来源反转与经济重算共用计划**：核销/对冲撤销后在同事务执行经济重算，未支付父单取消，已支付父单不动，以实际支付及已执行调整净额生成追回；未执行调整取消并留历史。禁止先走旧来源反转助手再做经济重算导致重复冲减。系统来源不允许通用确认/取消，支付仍检查服务端执行能力。
- PAID 是提成终态，永不改状态、不删除；已支付历史通过冲减调整追回，即 Clawback 模型。
- 净额 ≤ 0（全额冲减或超额冲减）自动释放提成财务锁；仍须取消有效账单、撤销费用批准，
  并满足业务锁与订单生命周期限制，才能修正费用后重新审核/结算。部分冲减（净额仍 > 0）保持锁定。此为普通编辑路径；受控费用更正不要求先解锁、取消账单或撤销原批准。
- DRAFT/CANCELLED 的提成与调整不参与净额。
- 已批准费用的正向成本补录只需费用审核权限；批准事务创建费用、刷新整票批准快照并生成
  CONFIRMED 系统净额调整；已执行的历史调整保持，必要时差额可为增提。待审补录/更正阻止新增建账及受影响提成确认/发放。
- 未支付父单发放金额为原提成加所有 CONFIRMED 调整的有符号净额，锁内复核预览；父单和
  调整同时落支付事实，调整记 OFFSET_WITH_PARENT，不另付一次。零净额可抵扣结清，负净额拒绝。
- PAID 父单及已执行调整的实际支付证据不可猜测补齐；来源整体反转统一取消未执行调整并按实际支付敞口重建追回，普通费用更正则保留手工调整。
- **双落点同一语义**：费用写入拦截（`internal/data/order_fee.go` 的
  `lockOrderForFeeMutation`）必须复用 `financeLockedOrderPredicate`；台账列表/详情的
  `finance_locked` 投影与 `financeLockCommissionNos` 走 `financeCommissionLockNetAmount`
  同口径。禁止出现"列表显示已解锁、写入仍被拒"的漂移。

## 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| net(order) > 0 时修改费用 | `ORDER_FEE_FINANCE_LOCKED`，提示走提成调整/冲减 |
| net(order) ≤ 0 时修改费用 | 财务锁放行，仍检查费用审核状态、业务锁、生命周期和账单占用 |
| 台账投影 finance_locked | 与写入拦截一致（同口径） |

## 5. Good / Base / Bad Cases

- Good：PAID 100 → 反核销全额冲减 -100 → 净额 0 → 取消有效账单、撤销费用批准且其他限制允许 → 修正费用并重新审核、核销生成新提成。
- Base：CONFIRMED 50 无冲减 → 净额 50 → 费用锁定。
- Bad：谓词改成"存在 CONFIRMED/PAID 提成即锁"（旧语义）——全额冲减后订单永久锁死。
- Bad：只改写入拦截不改台账投影（或反之）——两侧漂移。

## 6. Tests Required

- Data/PostgreSQL（`TestFeeLedgerCommissionNetLockReleasePostgres`）：全额冲减释放
  （列表投影+筛选谓词+写入拦截三处一致）、部分冲减仍锁、DRAFT 调整不参与、负净额释放。
- 反核销链路：反核销只生成 CONFIRMED 冲减调整，不动 PAID 父单状态。

## 7. Wrong vs Correct

### Wrong

```go
order.HasFinanceCommissionLinesWith(commission.StatusIn(CONFIRMED, PAID)) // EXISTS 即锁
```

### Correct

```go
// 净额聚合子查询：Σ提成线 + Σ符号化调整 > 0 才锁
financeLockedOrderPredicate() // settlement.go，与 Go 口径函数成对维护
```
