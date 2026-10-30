# Research: 订单里程碑 Schema、权限和生成物退役

- Query: 彻底删除独立订单履约里程碑，确定正式迁移、权限落库清理、生成物清理及隔离验证办法。
- Scope: internal
- Date: 2026-10-30

## Findings

### Schema 真相源和精确删除范围

- `server/internal/data/ent/schema/order_milestone.go:12` 定义唯一独立 `OrderMilestone` 实体；字段含订单 ID、类型、模板节点代码/名称、发生时间、备注、更新人和通用 ID/时间。
- `server/internal/data/ent/schema/order_milestone.go:28` 只有指向 Order 的必填唯一反向边；`server/internal/data/ent/schema/order.go:125` 是对应 `edge.To("milestones", OrderMilestone.Type)`。没有组织/User 反向边需要额外删。
- `server/migrations/20260820000000_baseline.sql:66` 创建 `order_milestones`，唯一外键为 `order_milestones_orders_milestones(order_id) -> orders(id) ON DELETE NO ACTION`，方向为里程碑引用订单，没有发现其他正式迁移建立指向该表的外键。
- 同文件 `:67-69` 定义三个索引：`ordermilestone_updated_at`、唯一 `ordermilestone_order_id_type`、`ordermilestone_order_id_occurred_at`。`DROP TABLE "order_milestones";` 会移除本表及这些专属索引/外键，无需 CASCADE，不应删除任何订单/业务状态表。
- `server/internal/data/ent/migrate/schema.go:4947` 等表结构定义是生成物，删 Schema/边后通过 Ent 生成更新，禁止手改。

### 正式迁移与部署行为

- `server/migrations/README.md` 指定正式迁移为 14 位时间戳 SQL，事务内执行并记录 SHA-256；新增迁移必须排链尾，不用 `IF EXISTS/IF NOT EXISTS` 掩盖漂移。
- 删除先例 `server/migrations/20260829103000_remove_milestone_templates.sql` 为两条直接 DROP TABLE；历史模板已退役，本次不要重写/删除该历史迁移或 baseline。
- `server/migrations/20261020120000_remove_house_bill_number_rule.sql` 展示按业务目标精确清理前置关联后删除的模式；本表没有被其他业务表引用，不需要类似的关联删除。
- 建议新增链尾迁移 `YYYYMMDDhhmmss_remove_order_milestones.sql`，审阅结果必须只有里程碑表 DROP，不使用 CASCADE，不改历史迁移，不重置现有库。该迁移执行后表内旧里程碑记录会被永久删除；现有开发库是否实际执行属于主会话的用户授权/交付边界，本研究未运行迁移。
- 仓库没有已配置的 Atlas HCL/独立 migration-diff 脚本；有现成 Ent/Atlas Go 依赖：Ent v0.14.6、Atlas v0.36.2-0.20250730182955-2c6300d0a3e1（`server/go.mod`）。若要求生成 SQL，应在一次性数据库建立旧正式迁移前态后，用 Ent `dialect/sql/schema.NewMigrate` + `WithDir` + `NamedDiff` 对新生成 `migrate.Tables` 做差异计划，输出临时目录，审阅并只纳入本次目标 DROP。
- `server/internal/data/ent/migrate/migrate.go` 的 `Schema.WriteTo` 虽不执行DDL，却不是仓库正式版本化迁移入口；不要以 `Schema.Create` 或运行时 AutoMigrate 替代正式 SQL。Ent/Atlas 的 `NamedDiff` 已在本地模块源码确认，`ModeInspect` 为默认；不建议为一次删表新增通用迁移平台。

### 权限无需另造清理 SQL

- `server/internal/access/manifest.go:155-156,309` 注册 `milestone.read` 与 `milestone.set` 操作；按 SE/SI/AE/AI/LAND/RAIL 展开为 12 个 `business.order.<kind>.milestone.<read|set>` 权限码。源清单应移除这两个操作及其注册/权限依赖引用。
- `server/internal/data/permission_manifest_sync.go:31` 明确在事务内同步清单并移除清单外权限，`:82` 执行 `Permission.Delete().Where(permission.KeyNotIn(manifestKeys...))`。
- `server/cmd/migrate/main.go:43-48` 在正式迁移 PostStep 内调用该同步，受迁移 advisory lock 保护；不需要用户另跑 bootstrap。
- `server/migrations/20260820000000_baseline.sql:122` 的 `role_permissions.permission_id` 外键 `ON DELETE CASCADE`，旧权限删除会自然移除角色关联，其他角色/权限不受影响。
- 没有独立权限依赖关系表：传递依赖来自 Manifest `Requires` 并由 `access.ResolveDependencies` 重算。因此不需要为本次再写 role_permissions 或 permission dependencies 清理 SQL；精确删除 Manifest 源定义后，运行正式迁移流程即可。
- 既有旧通用 `business.order.milestone.*` 与模板权限若仍意外留存，同一个“移除清单外”逻辑也会清掉；不要为仅清本次权限而改写历史权限迁移。

### 生成物清理机制与顺序

1. 删除 `server/api/order/v1/milestone.proto`。当前 `server/buf.gen.yaml:1-19` 没有 `clean`，只会生成仍存在的 proto；因此精确删除该已移除源对应的 `milestone.pb.go`、`milestone_grpc.pb.go`、`milestone_http.pb.go`，然后 `make -C server api`。这属于删除废生成物，不是手改生成内容。不要给 Buf 加 `clean:true`，其 out 含 `api` 和仓库 `.`，会扩大清理范围。
2. `make -C server api` 同时 `go generate ./internal/server`，由 `server/internal/server/access_rules.go:8` 重新扫描 proto 生成 `access_rules_gen.go`，自然移除两个路由规则；OpenAPI 总文件同样重生成。
3. 删除 Ent Schema/Order 边后执行 `go -C server generate ./internal/data/ent`（源入口 `server/internal/data/ent/generate.go:6`）。本地 Ent v0.14.6 `entc/gen/graph.go:1086` 的 `cleanOldNodes` 明确清理被删除节点的各模板生成文件与包目录；之后检查 `ordermilestone*` 不再残留，禁止手改其聚合生成物。
4. Service/Biz/Data ProviderSet 与 HTTP/gRPC 构造参数先移除后，重新运行 Wire 生成 `cmd/server/wire_gen.go`（具体 wire 入口见后端研究）；不要手改 wire_gen。
5. `pnpm run generate:web-client` 从更新后的 `server/openapi.yaml` 生成输入和客户端。`web/scripts/generate-services.mjs:14` 调用 `@umijs/openapi`，当前 v1.14.1 `dist/serviceGenerator.js:282` 起的 `genFile()` 会 glob+rimraf 清空目标服务目录再重建（仅含 `_deperated` 的路径被排除）；所以 `web/src/services/roncin/orderMilestoneService.ts` 与对应 typings/index 导出会自动消失，无需手删客户端。
6. `pnpm run generate:permission-keys` 从 Manifest 重生成 `web/src/permissions.generated.ts`；proto 相关枚举检查也随 generate:web-client 运行。

### 必须验证的邻近路径

- 新增隔离 PostgreSQL 迁移回归：通过正式迁移前缀构造删表前态，用原生 SQL 插入里程碑（因为新 Ent 不再含该实体），保留订单/lifecycle/其他子资源快照；应用新正式迁移并确认 `to_regclass('order_milestones')` 为空、订单及生命周期不变。
- 用 `SyncPermissionManifest` 验证里程碑旧权限及 role_permissions 关系消失，相邻订单 read/update/attachment 等保留；重复跑正式迁移/同步幂等。
- `server/internal/data/integration_test_helper_test.go:29` 有 `getIntegrationDataBeforeMigration(t, version)`，用正式 SQL 前缀+随机 Schema，需显式专用 `RONCIN_INTEGRATION_DATABASE_SOURCE`。不要对初始化 Schema 的测试使用 t.Parallel，也不要用运行期 Schema.Create。
- `server/internal/platform/migration/postgres_integration_test.go:50` 冷启动全链需同时 `RONCIN_POSTGRES_MIGRATION_TEST=1`。建议在一次性测试库运行：
  `RONCIN_INTEGRATION_DATABASE_SOURCE=<专用临时库> DATABASE_SOURCE=<同库> RONCIN_POSTGRES_MIGRATION_TEST=1 go -C server test -p 32 ./internal/platform/migration ./internal/data -run '<新删表回归>|TestPostgresColdStartMigration|TestSyncPermissionManifestPostgres'`。
  输出不能把 SKIP 当通过；测试库创建/销毁必须只针对本次命名且核验目标的临时库。
- 契约/生成完成后运行权限键、proto 常量、分层/架构检查和风险匹配完整门禁 `pnpm run check:fast`；搜索运行态源码与生成物确保没有 milestone API、模型、权限、前端服务残留。历史 SQL/历史任务/研究文档的文字不应要求全仓库零命中。

## Related Specs

- 根 `AGENTS.md`、`server/AGENTS.md`：边界、源先于生成物、正式迁移、秘密与数据保护。
- `.trellis/spec/server/backend/database-guidelines.md`：Ent 真相源、禁止运行时自动建表、事务封装与Schema并发禁忌。
- `.trellis/spec/server/backend/quality-guidelines.md`：真实库 PASS/SKIP 区分、迁移前态与被测时点。
- `.trellis/spec/web/frontend/column-settings.md`：不要把功能删除实现成列/入口默认隐藏。

## External References

未访问外网；版本和生成清理结论均来自仓库锁定依赖及本地模块源码（Ent v0.14.6、@umijs/openapi v1.14.1、Buf配置）。

## Caveats / Not Found

- 本研究仅阅读代码并写本文件，没有执行数据库语句、迁移或代码生成；也未创建/删除测试库。
- 删除表会删除原里程碑记录；本次目标不包含删除订单、重置开发库、修改历史审计记录或删除历史正式迁移。
- 自动生成清理结论需实施时用实际生成差异复核；Ent 子目录如有非生成自定义文件可能令目录移除失败，要精确检查而非盲目删整个 ent 目录。
