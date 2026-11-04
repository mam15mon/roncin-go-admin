# 验证记录

## 新增真实 PostgreSQL 回归

使用根协调者提供的一次性独立 PostgreSQL 实例及正式 SQL 迁移隔离 Schema，不访问开发业务库。

- `TestFinanceBillEconomicReconcilePostgres`：单张/批量 × 已确认/已付四组合通过。原提成 30，手工奖励 +5；取消成本账单后净权益 45，重建回到 35，改账单日至汇率 8 后为 25，再取消恢复 45。原计提、已支付金额与手工奖励保留；建账幂等重放无重复差额；审计失败回滚账单和全部差额。账单头/行按账单日固化 700，改日期后为 800，不与费用快照 600 强行相等。
- `TestFinanceBillCancelSerializesCommissionPaymentPostgres`：取消账单持有订单锁，旧预览支付受阻；取消提交后旧支付返回现有 409 领域错误，无支付事实，正确权益 40。
- `TestFinanceNettingRefreshDraftRatesPostgres`：真实两段预览及批量创建草稿，修改应收账单日期、确认账单后确认对冲；头/行应收 800、应付 700、汇差 -100。失败审计回滚草稿金额与分摊生效状态；重复确认被拒绝，已确认金额/版本不变。

上述三父测试首轮合计 9.867 秒通过；保留已有账单范围错误映射后最终复跑 12.797 秒再次通过。

## 相邻验证

`TestFinanceBillCreateSharedTransactionPostgres`、`TestFinanceBillBatchCreatePostgres`、`TestFinanceBillBatchNettingCreatePostgres`、`TestFinanceNettingTransactionPostgres`、`TestFinanceEconomicOrderSetExpansionPostgres`、`TestCommissionBillLockOrderConcurrentPostgres` 与 `TestFinanceNettingPersistsBothBaseAmountsAndGainLoss` 全部通过（14.373 秒）。保留原占用、幂等、并发快照与闭包变化门禁。

`git diff --check` 通过。对两个 Ent Schema 的四个金额字段移除 Immutable 后正式运行 `go generate ./internal/data/ent`；仅四份 netting/allocation create/update 生成物变化。第二次正式生成全目录指纹无差异；`go mod tidy` 后无 go.mod/go.sum 变更。数据库列形/约束及 API 均不变。

## 待办

- 根任务安排独立 review、整体验收与提交，本子代理未提交。
- 实际漏洞扫描按用户要求不运行。
