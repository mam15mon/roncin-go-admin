# 提成完整 PostgreSQL 事务集成测试

## Goal

补齐提成创建从 biz 用例到真实 PostgreSQL 仓储的端到端事务验证，证明成功时完整
固化主单、明细、CNY 快照和审计，失败时不留下事务内半成品；同时修正本轮提成
归档 PRD 中“实际已验收但仍未勾选”的记账漂移。

## Confirmed Facts

- 现有单元测试已覆盖 CNY 纯计算、用例编排、错误外传、锁 SQL 和列表/导出口径，
  但没有通过 `CommissionUsecase.Create` 跑完整真实数据库链路。
- 共享事务范围是生成上下文、汇率解析和 `CommissionRepo.Create`；编号生成按阶段 1
  已批准设计位于事务外，本任务不改变或伪装该编号语义。
- `CommissionRepo.Create` 在同一事务内重新计算、写主单、批量写明细并写审计；审计
  是最晚失败点，适合证明前序写入会随外层事务回滚。
- 集成测试约定只读取 `RONCIN_INTEGRATION_DATABASE_SOURCE`；未配置时明确跳过，
  不读取默认开发库配置。

## Requirements

1. 新增提成创建 PostgreSQL 集成测试，必须使用真实 `CommissionUsecase`、
   `CommissionRepo`、`ExchangeRateRepo`、`OrderConfigRepo` 和 `Data` 共享事务，禁止用
   sqlmock 或复制一份测试专用业务 SQL代替生产路径。
2. 成功路径至少构造组织、用户、订单、费用、账单/明细、核销/分摊、提成规则与
   订单归属关系，并断言：
   - 提成草稿和一条以上提成明细已提交；
   - `commission_date`、CNY 汇率来源/日期/金额与核销上下文一致；
   - 创建审计存在且详情包含折算依据；
   - 用例返回值来自事务提交后的普通上下文重读。
3. 汇率解析失败路径使用真实汇率用例与真实数据库配置缺口触发领域错误，断言没有
   提成主单、明细或创建审计。
4. 保存失败路径只允许通过测试仓储装饰器把合法用例输入变为可确定触发持久化错误
   的值，再委托真实仓储执行；不得在生产代码添加测试开关。断言没有主单、明细或
   创建审计。
5. 审计失败路径沿用仓库既有装饰器模式，把审计结果改为非法枚举后委托真实仓储，
   使失败发生在主单和明细写入之后；断言三类事务内产物全部回滚。
6. 测试夹具使用随机业务键隔离，清理顺序覆盖提成审计、明细、主单及全部来源数据；
   所有清理错误必须报告，数据库连接最后关闭。
7. 只把已有实现报告、独立检查和质量门证据能够证明完成的归档子任务 PRD 验收框
   改为 `[x]`；不修改 `implement.md` 历史执行清单，不补写不存在的测试证据。

## Acceptance Criteria

- [ ] 成功用例通过真实 PostgreSQL 完成创建、提交后重读，并精确验证主单、明细、
      CNY 快照及创建审计。
- [ ] 汇率失败、保存失败、审计失败三条路径均返回非空错误，且数据库中不存在本次
      提成主单、明细和创建审计。
- [ ] 审计失败确实发生在真实仓储完成主单和明细写入之后，不是装饰器提前返回。
- [ ] 默认环境未设置 `RONCIN_INTEGRATION_DATABASE_SOURCE` 时只有新集成测试明确
      跳过；显式注入隔离临时库后全部子用例通过且清理无残留。
- [ ] `go -C server test ./internal/data -count=1`、`go -C server test ./...`、
      `go -C server vet ./...` 和 `git diff --check` 全部通过。
- [ ] 不新增生产兜底、测试开关、API、迁移或生成物；若测试证明生产事务契约违约，
      停止并回到规划，不在测试任务中静默扩展生产修复。
- [ ] 阶段 1、阶段 2、阶段 3和静态路由归档 PRD 的已完成验收框与现有证据一致。

## Out of Scope

- 不改变失败是否消耗提成编号的既有语义；编号生成位于共享事务外。
- 不增加并发幂等压力测试、死锁重试、事务隔离级别或锁顺序调整；这些已有独立测试。
- 不改变导出 Count/分批读取的一致性边界，不新增 CSV 浏览器 E2E 或依赖升级。
- 不修改阶段 1/2/3 的历史实施清单，只整理 PRD 完成定义的勾选状态。

## Risks and Deferred Items

- 测试夹具跨越多类来源实体，优先复用现有账单/核销 PostgreSQL 夹具并做最小扩展；
  不为减少测试代码而抽象生产模块。
- 保存失败注入必须由独立检查确认确实进入真实仓储持久化路径；如果只能提前返回，
  该用例不算通过。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
