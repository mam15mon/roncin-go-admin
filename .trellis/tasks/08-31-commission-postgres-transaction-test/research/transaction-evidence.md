# 提成事务测试代码证据

## 当前缺口

- `server/internal/biz/finance_commission_test.go` 使用 stub 验证 CNY 创建编排，不能证明
  PostgreSQL 提交、回滚和提交后重读。
- `server/internal/data/finance_commission_test.go` 使用 sqlmock 验证 SQL 形态。
- `server/internal/data/finance_commission_transaction_integration_test.go` 只验证账单多行
  锁顺序，不执行完整提成创建。

## 可复用模式

- `finance_bill_transaction_integration_test.go`：共享 `Data`、按子测试创建独立组织、
  非法审计装饰器、成功/回滚状态断言和逆序清理。
- `finance_verification_transaction_integration_test.go`：在账单夹具上扩展核销来源、
  `newIntegrationData`、真实汇率仓储和共享事务用例。

## 生产边界

- `CommissionUsecase.Create` 的编号生成位于 `WithinTransaction` 之前；阶段 1 的
  `design.md` 明确记录为批准设计。本测试不把编号序列纳入事务回滚断言。
- `CommissionRepo.Create` 内部顺序为来源重算、主单写入、明细批量写入、审计写入，
  其中非法审计结果可以在最晚点触发真实回滚。
- 提成响应在 `WithinTransaction` 成功返回后通过普通 `ctx` 重读，成功测试应断言
  返回对象与数据库已提交对象一致。
