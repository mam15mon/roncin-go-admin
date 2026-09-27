# 四项失败执行计划

- [x] 审阅规划并激活本子任务；读取上下文规范与四项测试，确认专用库目标、初始化和清理边界。
- [x] 通过安全环境变量注入 RONCIN_INTEGRATION_DATABASE_SOURCE，以下命令不得在未注入变量时当作有效集成验证；完整 migration 还需 RONCIN_POSTGRES_MIGRATION_TEST=1，子进程 DATABASE_SOURCE 指向相同隔离库。不得只按测试名后缀筛选而漏掉迁移用例。

```bash
go -C server test -count=1 -p 32 -timeout 20m ./internal/data/ -run '^(TestRoleWorkspaceAnchorBackfillPostgres|TestFeeCatalogCompanyTemplatesPostgres)$'
go -C server test -count=1 -p 32 -timeout 20m ./internal/platform/migration/ -run '^(TestFeeCatalogSeedSeedsTemplatesAndCompanyCopies|TestCompanyBoundaryMigrationRewritesFeeReferencesWithoutSnapshots)$'
```

- [x] 先记录失败；完成实现后按风险补验证，禁止 TDD。逐组修正根因，确保执行到目标断言，运行对应定向及相邻用例。
- [x] 分别、顺序执行两包真实库全量，保留脱敏输出，不同时占用同一测试目标：

```bash
go -C server test -json -count=1 -p 32 -timeout 40m ./internal/data/
go -C server test -json -count=1 -p 32 -timeout 40m ./internal/platform/migration/
go -C server vet ./internal/data/ ./internal/platform/migration/
git diff --check
```

- [x] 检查完整 -v 日志及独立 JSON 事件中的 PASS / FAIL / SKIP、panic 和超时；四项均实际执行，检查是否有先前提前失败掩盖的新错误。
- [x] Trellis 检查代理审阅契约与断言强度；生成 research/verification.md，完成组内提交，交接父任务与实弹验收子任务。
- [ ] 最终全栈门禁由父任务统筹；产品变更导致此前结果失效时重跑受影响检查。

当前交接：四项目标与相邻真实库定向均 PASS、无 SKIP，vet 与 diff 检查通过；主会话安排真实验收 Stage A 顺序执行两包完整集成并共享日志，本实施代理不重复启动全量。详见 `research/verification.md`。

完整迁移开关补查交接：首次 Stage A data 完整包 291 顶层与 453 子项 PASS、无 FAIL/SKIP（390.182s）；migration 因缺少 `RONCIN_POSTGRES_MIGRATION_TEST=1` 跳过 7 项，被编排拒绝。独立显式开启开关后定位额外四项历史迁移测试时点漂移，限制到各被测正式目标且保留原断言，完整 migration JSON 重跑 27 顶层与 9 子项 PASS、无 FAIL/SKIP（38.374s）。代码已冻结，父任务 runner 最终统一 Stage A 正在纳入本次修复；不重复完整 data。

最终交接：第二轮 Stage A 采用最终六文件代码，两包按顺序完整执行，data 291 顶层 + 453 子项 PASS（401.146s）、migration 27 顶层 + 9 子项 PASS（37.967s），0 FAIL / SKIP。六文件完整审阅无遗留。父任务继续全栈门禁与财务 HTTP / 浏览器验收，此子任务的测试基础设施交付已满足验收。
