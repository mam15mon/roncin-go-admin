# 实施计划：提成账单固定加锁顺序

## 1. 修正生产查询

- 在 `server/internal/data/finance_commission.go` 的账单批量查询增加
  `Order(bill.ByID())`。
- 保持原有谓词、`WithLines()`、`ForUpdate()` 条件和后续计算代码不变。
- 验证：检查生成 SQL 的排序与锁子句相对顺序。

## 2. 补充针对性验证

- 在 `server/internal/data/finance_commission_test.go` 增加锁定 SQL 形态断言。
- 增加 `finance_commission_transaction_integration_test.go` 或复用同域现有集成测试
  文件，构造相反输入顺序的双事务真实 PostgreSQL 场景。
- 集成测试只从 `RONCIN_INTEGRATION_DATABASE_SOURCE` 取连接；未配置时 `Skip`。
- 禁止先写测试再实现；先完成查询修正，再补验证。

## 3. 验证与独立检查

依次运行：

```bash
go -C server test ./internal/data -run 'Test.*Commission.*Lock' -count=1
go -C server test ./internal/data -count=1
go -C server test ./...
go -C server vet ./...
```

如果配置了 `RONCIN_INTEGRATION_DATABASE_SOURCE`，单独以 `-count=1` 执行新增
PostgreSQL 并发测试并确认没有跳过。随后由独立 `trellis-check` 复核 SQL 是否确实
来自生产路径、测试是否可能在删除排序后仍通过、是否有超出任务范围的修改。

## 4. 提交与回滚点

- 实施代理禁止 Git 操作；Codex 主会话复核和验证后提交：
  `fix: 固定提成账单多行加锁顺序`。
- 若真实并发夹具需要扩大到 API/biz 契约或 Schema，停止实施并返回规划，不扩大范围。
