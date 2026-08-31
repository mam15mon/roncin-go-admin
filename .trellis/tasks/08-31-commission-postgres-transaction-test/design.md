# 设计：提成完整 PostgreSQL 事务集成测试

## 变更边界

主要交付物是 `server/internal/data` 下的提成创建集成测试及其测试夹具。预计不修改
生产代码；文档只更新四个已归档子任务 PRD 的验收勾选状态。若真实数据库测试发现
生产事务违约，必须暂停并重新规划生产修复。

## 被测生产链路

```text
CommissionUsecase.Create
  ├─ 事务外：参数校验、幂等预查、提成编号生成
  ├─ Data.WithinTransaction
  │    ├─ CommissionRepo.GetGenerationContext → 核销单 FOR UPDATE
  │    ├─ ExchangeRateUsecase.Resolve → 真实组织/时间标准/汇率查询
  │    ├─ ResolveCommissionCNYRate
  │    └─ CommissionRepo.Create
  │         ├─ 真实来源重算与锁定
  │         ├─ INSERT finance_commissions
  │         ├─ INSERT finance_commission_lines
  │         └─ INSERT audit_logs
  └─ 提交后 CommissionRepo.Get
```

测试不得绕开上述链路直接调用 Ent 创建提成结果；Ent 只用于准备来源夹具和验证最终
数据库状态。

## 夹具设计

优先复用现有 `newFinanceBillPostgresFixture`、`newVerificationPostgresFixture` 或其
底层辅助方法，补齐提成计算缺少的最小来源：

- commission 编号规则；
- 可用员工；
- 订单提成归属关系；
- 已确认、包含活动账单明细的应收账单；
- 已核销分摊；
- 已启用且生效的提成规则；
- CNY 本位币成功路径所需的 WRITE_OFF 时间标准。

夹具清理使用 `t.Cleanup` LIFO：先注册数据库关闭，再注册按外键逆序删除数据的清理
函数，使业务数据先清理、连接最后关闭。每个删除步骤独立报告错误。

## 失败注入

### 汇率解析失败

删除或不创建 WRITE_OFF 时间标准，让真实 `ExchangeRateUsecase.Resolve` 返回日期缺失
领域错误。此时尚未调用仓储创建，但编号可能已按既有事务外设计分配；验收只检查
事务拥有的提成主单、明细和创建审计。

### 保存失败

定义仅存在于 `_test.go` 的 `CommissionRepo` 装饰器，重写 `Create`：把快照或待保存
对象改为能够确定触发 Ent/PostgreSQL 持久化校验错误的值，然后调用内层真实
`CommissionRepo.Create`。装饰器不得直接返回自造错误。测试断言错误外传且没有
事务内产物。

### 审计失败

定义测试装饰器，将 `audit.Result` 改为数据库不接受的枚举值，再调用真实仓储。
由于 `writeAudit` 位于主单和明细插入之后，该错误证明外层共享事务会回滚前序写入。

## 状态断言

成功路径按组织、提成 ID 和 action 精确查询：

- 主单数量、状态、版本、来源指纹、归属日期和 CNY 字段；
- 明细数量、订单/员工/规则口径与金额；
- `finance.commission.create` 审计及其 JSON details；
- 用例返回 ID 与持久化主单一致。

失败路径按本次幂等键/组织/action 断言主单、明细、创建审计均为零。来源核销、账单、
订单和费用应保持可读，证明回滚没有破坏来源数据。

## 文档整理

只更新以下归档 PRD 中已经由归档报告和质量门证明完成的 Acceptance Criteria：

- `08-31-commission-cny-snapshot/prd.md`
- `08-31-commission-export/prd.md`
- `08-31-commission-cny-ui/prd.md`
- `08-31-commission-static-route-order/prd.md`

不修改历史 `implement.md`，因为它记录的是执行计划而不是最终勾选真相源。
