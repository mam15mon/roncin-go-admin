# 数据库规范（Ent / PostgreSQL / 事务 / 并发）

## Schema 与迁移

- PostgreSQL Schema 以 Ent Schema（`server/internal/data/ent/schema/`）为真相源。
- 生产变更必须生成迁移并随代码提交（`pnpm run migrate:server` 同时幂等同步权限目录）。
- 禁止在业务代码散落手写 SQL；确有必要时集中封装在 `internal/data` 并说明原因。
- 唯一性靠数据库唯一索引兜底，不用先查后插。

## 事务统一封装（`internal/data/transaction.go`）

所有事务必须走统一封装，禁止手写 `db.Tx` + `Rollback`/`Commit` 模板，禁止
混用 `sqlDB.BeginTx` 原生事务：

```go
// 单仓储事务：封装内部负责 Begin / 出错自动 Rollback / panic 安全回滚 / 成功 Commit
err := r.data.WithTx(ctx, func(tx *ent.Tx) error { ... })

// 跨仓储共享事务：biz.Transactor 建立，回调内取得 txCtx，多仓储同事务读写
err := r.transactor.WithinTransaction(ctx, func(txCtx context.Context) error { ... })
```

- 嵌套 `WithinTransaction` / `WithTx` 自动合并到外层事务。
- 事务结束后再用 `txCtx` 会返回业务错误，不会静默开新事务。
- **仓储取得 Ent 客户端必须经由 `Data.client(ctx)`**，直连 `d.db` 会静默落到
  共享事务之外。事务内参与的读取按并发要求加 `ForShare`。
- `withSQLTx` 原生 SQL 事务不加入共享事务：`txCtx` 回调内禁止调用走原生事务
  的仓储方法。共享事务回调只做校验与写入，完整业务响应在提交后用普通上下文
  重读（范本：`internal/biz/finance_bill.go` 的 `Create`）；回调错误必须原样
  外传，不得吞掉后继续提交。

## 并发修改防护：「悲观锁 + 乐观锁」双层模式

范本：`internal/data/order_write.go` 的 `UpdateDraft`
（`server/internal/data/order_write.go:164`）：

```go
err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
    existing, queryErr := tx.Order.Query().
        Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID)).
        ForUpdate().Only(ctx)                       // 1. 悲观锁锁定目标行
    if queryErr != nil {
        return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
    }
    if existing.Version != expectedVersion {         // 2. 乐观锁版本比对
        return biz.ErrOrderStatusConflict           //    → HTTP 409「已被更新，请刷新后重试」
    }
    if existing.FlowStatus != orderent.FlowStatusDRAFT { // 3. 状态机检查后才允许变更
        return biz.ErrOrderStatusConflict
    }
    // ...校验引用...
    update := existing.Update().
        SetVersion(existing.Version + 1).           // 4. 版本号 +1
        ...
})
```

按实体分层取舍：

| 场景 | 防护 |
|------|------|
| 多人协作编辑（订单、账单、结算等） | `version` 字段 + `expectedVersion` 校验 + `ForUpdate` |
| 状态流转、号码分配等原子操作 | `ForUpdate` + 状态检查 |
| 低频单人维护的主数据 | 可不加版本检测 |

新增实体判定不清时，先向用户确认。

- 多行加锁必须先按主键排序再 `ForUpdate()`，固定加锁顺序防死锁。
- 同一事务内后续需要对某行 `FOR UPDATE` 时，首次读取禁止先 `FOR SHARE` 再升级：
  两个并发事务可同持共享锁互等升级形成死锁；应从入口即 `FOR UPDATE` 串行化
  （范本：`GetGenerationContext` 在事务上下文对核销单直接 `ForUpdate`）。
- 驱动错误用 `ent.IsConstraintError` 统一判断并映射为业务错误。

## 列表分页约定

- 常规列表 `pageSize` 上限统一 **200**，默认值按接口交互需要设置；禁止出现
  `100`、`500` 等不一致上限。
- 复用 `internal/biz` 的公共常量与校验函数：`page >= 1` 且
  `1 <= pageSize <= 200`，并补边界测试。
- 选择器类查询支持服务端 `keyword` 过滤；keyword 必须覆盖业务代码、中英文
  名称、别名（若有）、无声调全拼及拼音首字母；新增可搜索中文实体时同步维护
  拼音检索键、索引与历史回填。
