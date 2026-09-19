# ADR 0004: 统一事务封装 WithTx / WithinTransaction

- 状态：已采纳
- 日期：2026-08-30（此前各仓储手写事务，当日统一收敛）
- 主题：事务与并发

## 背景

事务最初以 `db.Tx(ctx)` + 逐点 `tx.Rollback()` / `tx.Commit()` 的模板散落
在各仓储里。该写法有三个问题：错误路径漏写回滚导致事务悬挂；`panic` 时不
回滚；「订单取号与创建必须原子提交」（2026-08-30 `c6877ac4`）这类跨步骤
原子性只能靠每个调用点自觉。同时出现了跨仓储共享事务的需求（如账单创建
需要同时写账单、费用、提成多个聚合），手写模板无法表达。

## 决策

- 单仓储事务统一走
  [../../../server/internal/data/transaction.go](../../../server/internal/data/transaction.go)
  的 `Data.WithTx(ctx, fn)`：封装内部负责 Begin、出错自动 Rollback、
  `panic` 安全回滚、成功 Commit。
- 跨仓储共享事务由 `biz.Transactor.WithinTransaction(ctx, fn)` 建立：
  回调内取得 `txCtx`，多个仓储在同一事务中读写；嵌套 `WithinTransaction`
  与 `WithTx` 自动合并到外层事务；事务结束后再使用 `txCtx` 返回业务错误，
  不会静默开新事务。
- 仓储取得 Ent 客户端必须经由 `Data.client(ctx)`，禁止直连 `d.db`——
  直连查询会静默落到共享事务之外。事务内参与的读取按并发要求加 `ForShare`。
- `withSQLTx` 原生 SQL 事务不加入共享事务：`txCtx` 回调内禁止调用走原生
  事务的仓储方法。
- 共享事务回调只做校验与写入，完整业务响应在事务提交后用普通上下文重读
  （范本：`internal/biz/finance_bill.go` 的 `Create`）；回调错误必须原样
  外传，不得吞掉后继续提交。

## 理由

- 事务正确性是样板代码，不应依赖每个调用点的自觉；集中封装后 Begin /
  Rollback / panic 语义只实现并测试一次。
- `txCtx` 隐式传播让多仓储原子性对用例层透明，同时「事务结束后报错而非
  静默开新事务」保证不会产生半提交假象。
- 备选的显式传递 `tx` 对象方案侵入所有仓储签名，改造成本高于隐式上下文。

## 后果

- 存量手写事务已全部迁移；触碰旧写法时必须直接改写为封装调用。
- `WithTx` 回调收到的 `ctx` 不携带事务标识，回调内辅助方法须经由 `tx`
  查询或显式传参（见 database-guidelines 的 Warning）。
- 锁互斥行为无法用 sqlmock 覆盖，依赖
  `RONCIN_INTEGRATION_DATABASE_SOURCE` 集成环境回归。
- 规范全文见
  [../server/backend/database-guidelines.md](../server/backend/database-guidelines.md)。
