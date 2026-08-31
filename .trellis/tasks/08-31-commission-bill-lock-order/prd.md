# 修复提成账单多行加锁缺排序

## Goal

修复提成生成在读取多个应收账单并执行 `FOR UPDATE` 时没有固定锁顺序的问题，
避免不同事务以不同账单顺序取得行锁而形成死锁；在进入生产前用 SQL 形态断言和
真实 PostgreSQL 并发场景共同验证。

## Background

- 风险位置：`server/internal/data/finance_commission.go:483` 的账单批量查询。
- 当前查询按核销分摊边的返回顺序构造 `billIDs`，随后直接 `ForUpdate().All(ctx)`；
  SQL 没有显式 `ORDER BY finance_bills.id`。
- 根规范要求“多行加锁必须先按主键排序再 `ForUpdate()`”。数据库默认返回顺序、
  UUID 切片顺序和 Ent 内部行为都不能作为锁顺序保证。
- 阶段 1 已把同一核销单的首次读取改为 `FOR UPDATE`，解决同一核销单内部的锁升级
  竞争；本任务处理不同核销单可能引用重叠账单集合时的多行锁顺序，不能依赖前一项
  修复替代。

## Requirements

1. 账单批量查询必须在 `ForUpdate()` 前显式按 `finance_bills.id` 升序排序。
2. 不改变账单筛选条件、关联行预加载、提成计算、指纹、错误映射或事务边界。
3. 增加 data 层 SQL 形态测试，直接证明锁定查询同时包含主键升序排序和
   `FOR UPDATE`，防止以后删掉排序仍有测试通过。
4. 增加真实 PostgreSQL 集成测试：两个事务以相反的账单 ID 输入顺序锁定同一组
   账单，断言两者均在超时前完成且没有 PostgreSQL deadlock 错误；测试必须执行与
   生产相同的查询构造路径，不复制一份仅供测试的 SQL。
5. 集成测试沿用 `RONCIN_INTEGRATION_DATABASE_SOURCE`：未配置时明确跳过，不读取
   默认开发库，也不新增兜底连接串。

## Acceptance Criteria

- [ ] 生产查询生成 `ORDER BY "finance_bills"."id" FOR UPDATE`，排序位于加锁子句前。
- [ ] SQL 形态测试在排序缺失、字段错误或排序位于错误查询时会失败。
- [ ] PostgreSQL 集成测试以相反输入顺序并发执行生产锁定路径，两事务均正常提交，
      没有死锁或超时。
- [ ] 预览等 `lock=false` 路径保持无 `FOR UPDATE`，业务计算与错误语义不变。
- [ ] `go -C server test ./internal/data`、`go -C server test ./...` 与
      `go -C server vet ./...` 全部通过；未配置集成数据库时仅对应测试明确跳过。
- [ ] 变更只涉及该查询、直接测试和必要的 Trellis 记录，不改契约、迁移或生成物。

## Out of Scope

- 不调整核销单、规则、员工、订单、费用或归属关系的既有锁顺序。
- 不增加死锁自动重试、超时兜底、隔离级别变更或新的事务封装。
- 不改提成幂等键、唯一索引、计算口径、CNY 快照或导出功能。

## Risks and Deferred Items

- 普通 CI 未提供 `RONCIN_INTEGRATION_DATABASE_SOURCE` 时，真实 PostgreSQL 用例会
  按仓库既有约定跳过；SQL 形态测试仍是默认质量门的一部分。
- 若集成测试暴露其他实体的锁顺序问题，只记录独立缺陷，不在本任务扩大范围。
