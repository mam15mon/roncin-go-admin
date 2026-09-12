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
  核销同构：CONFIRMED 对冲单的 RECEIVABLE 分摊按 `分摊金额/账单总额` 摊入账单行本位币。
- **冲减联动对称**：核销 REVERSE 与对冲 REVERSE 共用冲减助手
  （`reconcileCommissionsForSourceReversal`）：未支付取消、有支付敞口生成 CONFIRMED
  DECREASE 调整（`source_type` 分别为 VERIFICATION_REVERSAL / NETTING_REVERSAL，
  幂等键前缀 vr:/nt:）。前端对两种系统冲减同款 gating：不暴露取消/确认操作，
  PAID 文案为「追回」族。
- PAID 是提成终态，永不改状态、不删除；已支付历史通过冲减调整追回，即 Clawback 模型。
- 净额 ≤ 0（全额冲减或超额冲减）自动释放该订单的费用编辑锁，打通"反核销/反对冲 → 修正
  费用/账单 → 重新结算"的纠错通道；部分冲减（净额仍 > 0）保持锁定。
- DRAFT/CANCELLED 的提成与调整不参与净额。
- **双落点同一语义**：费用写入拦截（`internal/data/order_fee.go` 的
  `lockOrderForFeeMutation`）必须复用 `financeLockedOrderPredicate`；台账列表/详情的
  `finance_locked` 投影与 `financeLockCommissionNos` 走 `financeCommissionLockNetAmount`
  同口径。禁止出现"列表显示已解锁、写入仍被拒"的漂移。

## 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| net(order) > 0 时修改费用 | `ORDER_FEE_FINANCE_LOCKED`，提示走提成调整/冲减 |
| net(order) ≤ 0 时修改费用 | 放行 |
| 台账投影 finance_locked | 与写入拦截一致（同口径） |

## 5. Good / Base / Bad Cases

- Good：PAID 100 → 反核销全额冲减 -100 → 净额 0 → 费用可改 → 修正后重新核销生成新提成。
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
