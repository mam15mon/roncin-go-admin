# 规划证据（2026-09-27）

## 历史结果来源

- `.trellis/tasks/archive/2026-09/09-27-fee-status-review-fixes/research/verification.md:82`：真实库 data 与 migration 两包合计四项失败。
- `.trellis/tasks/archive/2026-09/09-26-fee-hard-delete/research/baseline-failure-registry.md`：原登记 #13—#16；§6 记录公司边界迁移包对照结论。该文档保留了更早的五项失败历史，不以其旧计数覆盖后续四项结论。

| 测试与定位 | 历史失败线索 | 实施需要证明的事实 |
| --- | --- | --- |
| `server/internal/data/admin_role_anchor_integration_test.go:159` / `TestRoleWorkspaceAnchorBackfillPostgres` | organizations_workspace_parent_check 先拒绝夹具 | 按目标迁移时点合法构造夹具，真正验证锚点回填及组织约束 |
| `server/internal/data/fee_catalog_integration_test.go:149` / `TestFeeCatalogCompanyTemplatesPostgres` | 费用目录断言漂移，旧日志包含结构指针输出 | 先输出可定位差异，再根据现行系统模板 / 公司副本契约修复 |
| `server/internal/platform/migration/fee_catalog_seed_integration_test.go:87` / `TestFeeCatalogSeedSeedsTemplatesAndCompanyCopies` | master_data_items 实际 18，期望 5 | 区分全局种子与目标集合，证明模板、副本、幂等性正确 |
| `server/internal/platform/migration/company_boundaries_integration_test.go:173` / `TestCompanyBoundaryMigrationRewritesFeeReferencesWithoutSnapshots` | 费用快照或其他字段被修改 | 列级对照引用重写和必须保持的快照，明确夹具还是产品缺陷 |

以上为历史证据，规划阶段未重新复现，不提前断言根因已彻底查明。

## 当前可复用入口

- `package.json` 提供 acceptance:finance-bill-batch、acceptance:finance-payable、acceptance:finance-ui、acceptance:finance:foreign-currency、acceptance:finance:disposable。
- `scripts/acceptance-finance-bill-batch.mjs:115` 与 payable 脚本 `:105`：默认只检查前置条件，真实闭环需 `--apply`。
- `scripts/acceptance-finance-foreign-currency.mjs:703` 起包含提成预览、生成和读取断言；执行模式与应收/应付脚本不同，运行前单独核验。
- `scripts/run-acceptance-finance-disposable.mjs:17` 起要求显式数据库管理员与 bootstrap 配置；`:44` 起使用 8010 / 8001 端口；`:452` 起创建一次性角色和库，`:495` 起销毁。不能直接对现有开发库运行清理流程。
- `web/playwright.config.ts`：真实浏览器单 worker，失败保留 trace / screenshot，默认前端地址 8001。
- `web/tests/e2e/finance-bill-batch.e2e.ts:16`：已有财务 UI 用例；`:41` 起 API 查找既有 ACC-FIN 订单，结尾从列表选取已有提成查看，并未完整从页面创建费用后追踪到同单提成。

## 技术未知与执行期检查

当前专用库权限、验收账号、端口、浏览器二进制以及编排与现行契约的适配程度均待执行期检查。这些是运行条件，不改变验收目标；缺失时记录阻塞且不宣称通过。
