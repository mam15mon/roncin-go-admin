# 四项失败执行计划

- [x] 审阅规划并激活本子任务；读取上下文规范与四项测试，确认专用库目标、初始化和清理边界。
- [x] 通过安全环境变量注入 RONCIN_INTEGRATION_DATABASE_SOURCE，以下命令不得在未注入变量时当作有效集成验证；也不得只按测试名后缀筛选而漏掉迁移用例。

```bash
go -C server test -count=1 -p 32 -timeout 20m ./internal/data/ -run '^(TestRoleWorkspaceAnchorBackfillPostgres|TestFeeCatalogCompanyTemplatesPostgres)$'
go -C server test -count=1 -p 32 -timeout 20m ./internal/platform/migration/ -run '^(TestFeeCatalogSeedSeedsTemplatesAndCompanyCopies|TestCompanyBoundaryMigrationRewritesFeeReferencesWithoutSnapshots)$'
```

- [x] 先记录失败；完成实现后按风险补验证，禁止 TDD。逐组修正根因，确保执行到目标断言，运行对应定向及相邻用例。
- [ ] 分别、顺序执行两包真实库全量，保留脱敏输出，不同时占用同一测试目标：

```bash
go -C server test -json -count=1 -p 32 -timeout 40m ./internal/data/
go -C server test -json -count=1 -p 32 -timeout 40m ./internal/platform/migration/
go -C server vet ./internal/data/ ./internal/platform/migration/
git diff --check
```

- [ ] 检查 JSON 事件中的 PASS / FAIL / SKIP、panic 和超时；四项均实际执行，检查是否有先前提前失败掩盖的新错误。
- [ ] Trellis 检查代理审阅契约与断言强度；生成 research/verification.md，完成组内提交，交接父任务与实弹验收子任务。
- [ ] 最终全栈门禁由父任务统筹；产品变更导致此前结果失效时重跑受影响检查。

当前交接：四项目标与相邻真实库定向均 PASS、无 SKIP，vet 与 diff 检查通过；主会话安排真实验收 Stage A 顺序执行两包完整集成并共享日志，本实施代理不重复启动全量。详见 `research/verification.md`。
