# 数据库规范（Ent / PostgreSQL / 事务 / 并发）

## Schema 与迁移

- PostgreSQL Schema 以 Ent Schema（`server/internal/data/ent/schema/`）为真相源。
- 生产变更必须生成迁移并随代码提交（`pnpm run migrate:server` 同时幂等同步权限目录）。
- 禁止在业务代码散落手写 SQL；确有必要时集中封装在 `internal/data` 并说明原因。
- 唯一性靠数据库唯一索引兜底，不用先查后插。

## Scenario：正式 CHECK 与 Ent Schema 同源

### 1. Scope / Trigger

- 正式 PostgreSQL 迁移新增、删除或修改 `CHECK` 约束时适用。
- `field.Enum(...)`、`field.Int64(...).Positive()` 等 Go 侧声明不能替代数据库
  `CHECK` 的真相源声明；凡迁移中要求数据库强制执行的条件，都必须在 Ent Schema
  同步声明。

### 2. Signatures

- Ent 表级约束：
  `Annotations() []schema.Annotation { return []schema.Annotation{entsql.Checks(...)}}`。
- 正式迁移：`ALTER TABLE ... ADD CONSTRAINT <stable_name> CHECK (...)` 或建表内同名
  `CONSTRAINT`。
- 生成验证对象：`server/internal/data/ent/migrate.<Table>.Annotation.Checks`。

### 3. Contracts

- Ent 注解、生成的 `ent/migrate/schema.go` 与正式 SQL 迁移必须使用相同的稳定约束名
  和等价表达式。
- 生产迁移仍以 `pnpm run migrate:server` 执行；Ent 注解用于保持 Schema 真相源完整，
  防止开发期 `Schema.Create` 或后续差异生成把正式约束识别为漂移。
- 生成文件只能通过 `go -C server generate ./...` 更新，不得手改。

### 4. Validation & Error Matrix

| 条件 | 行为 |
|------|------|
| SQL 有 CHECK，Ent Schema 没有 | 阻断提交；补 Ent 注解和生成元数据测试 |
| Ent 与 SQL 约束名不同 | 阻断提交；统一稳定名称，避免被识别成删除再重建 |
| Ent 与 SQL 表达式语义不同 | 阻断提交；以已批准的业务约束修正两处 |
| 生成后 `Annotation.Checks` 缺项 | 生成/测试失败，不得手改 `ent/migrate/schema.go` |
| 开发库迁移后 `pg_constraint` 缺项 | 数据库验收失败，不得只凭 Go 测试宣告完成 |

### 5. Good / Base / Bad Cases

- Good：迁移与 `entsql.Checks` 都声明 `file_size > 0`，测试断言生成表注解中存在同名
  约束，并在空库迁移后查询 `pg_constraint`。
- Base：纯应用层格式校验无需数据库强制时，不额外添加推测性的 CHECK。
- Bad：只在 SQL 迁移写 CHECK，认为 `field.Enum` 会自动让后续 Schema 同步保留它。

### 6. Tests Required

- 生成元数据测试：断言表名、约束名和完整表达式。
- 空 PostgreSQL Schema 迁移测试：断言 `pg_constraint` 中约束存在且迁移 revision/checksum
  正确。
- 生成幂等：重跑 `go -C server generate ./...` 后 tracked/untracked 内容指纹不变。

### 7. Wrong vs Correct

#### Wrong

```go
// 正式迁移有 CHECK，但 Ent 真相源没有对应声明。
field.Enum("result_role").Values("ORIGINAL", "CREATED")
```

#### Correct

```go
func (SeaOrderSplitResult) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Checks(map[string]string{
		"sea_order_split_results_result_role_check":
			"result_role IN ('ORIGINAL', 'CREATED')",
	})}
}
```

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
