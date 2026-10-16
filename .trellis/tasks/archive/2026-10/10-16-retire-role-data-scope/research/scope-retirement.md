# Research: 角色数据访问范围退役

- Query: 在已有公司隔离模型下完整退役可配置角色数据范围，保留内部治理门禁；核对迁移、授权、防提权与生成验证边界。
- Scope: internal
- Date: 2026-10-16

## Findings

### 推荐

删除角色创建/编辑/列表契约与界面中的 dataScope，并从 Ent Role 删除 data_scope，正式迁移删除数据库列。保留 biz.DataScope、auth PermissionCapability 等内部有效能力模型，由角色所属工作台决定：company → organization，system → all。这一有效能力不是可配置角色属性，不能保留隐藏的请求字段或仓储 input.DataScope 写入路径。角色授权仍按权限来源、工作台归属与已启用状态检查。

仅隐藏界面并保留数据库可配置字段不完整：service/admin.go:321、337 仍接收 DataScope；admin_role.go:289、336 仍写入该值；normalizeRole 在 biz/admin_role.go:252-259 拒绝 self，造成旧角色无法编辑。故推荐完整删列，避免继续存在两个真相源。

### 数据迁移语义

1. 当前 biz/auth.go:385-386 的 active() 不包含 self。旧 self 角色整体处于无效授权状态，迁移应先 UPDATE roles SET enabled=false WHERE data_scope='self'，再删列，保留角色、成员关系与权限关联。不能直接去列后把 self 自动变为 organization/all，否则无意激活旧权限。
2. 公司原 all/tree 角色固定 organization，现有 workspacePermissionScope（biz/auth.go:795 起）已经把公司可读写范围封闭在当前公司，符合既有隔离。
3. 系统原 organization/tree 固定 all 会扩大其已持有治理权限的适用范围。这是移除配置后的明确契约变更，应在设计中写明并测试；不能声称完全无行为变化。系统工作台的业务权限仍由 permissionAvailableInWorkspace 拦截，公司业务操作不能因 all 激活。
4. 原 enabled=false 必须保持；self 迁移后管理员可通过正常角色编辑显式启用，不能由种子/管理员补挂自动恢复。bootstrap_administrator.go:94 当前有独立 self 保护，删列后 enabled=false 分支继续承担保护。
5. 迁移依据 Ent Schema 差异正式生成 DROP COLUMN（参考 migrations/20261014100000_remove_membership_primary.sql），数据停用 DML 在同迁移中位于 DROP 前。不修改历史迁移，不重置数据库。

### 无 N+1 的范围推导

- data/auth.go:237 collectWorkspaceRoleGrants 已有 nodes 和 workspaceID，直接 nodes[workspaceID].Kind 决定有效范围，再传给 roleGrantFromEntRole（当前 :426），不需要逐角色查询。
- data/admin_role.go:185 actorRolesPrivilegeProfiles 已加载组织树和解析 workspaceID，同样推导后传入 roleProfileToBiz。
- data/admin_role.go:238 rolesPrivilegeProfiles 目前先 resolveRoleOrganizationID（它内部已读树但只返回 ID），可在该函数一次读取组织树并解析 ID+kind，或抽出一次返回工作台节点的内部辅助；避免重复加载整棵树。
- 新建/更新输入 AdminRole 应不再有 DataScope；权限校验用当前实际工作台固定范围（不可简单以 actor profile 最大 scope 代替目标 scope）。工作台 kind 可从已验证 principal 推导；无 principal 的仓储/集成调用需要通过角色组织查询确定，不能默认 system/all。
- 无效 kind / uuid.Nil 显式拒绝，禁止默认公司或 all。

### 防提权与审批/锁定关键路径

- biz/admin_role.go:64 validateRolesPrivilege 约束目标组织范围；:121 checkPrivilegeEscalation 保留权限子集、administrator 保护、按同一来源角色验证权限与能力，不可把额外角色的宽 scope 拼到另一个角色权限上。
- data/admin_role.go 创建/更新事务已有 PermissionAllowedInWorkspace 检查（:282、:329），须保留，防公司发系统权限、系统发公司业务权限。
- data/order_lock_qualification.go:105-135 不是简单展示：SQL 用 role.data_scope 构建 all/org/tree 三分支，影响锁定人员和费用审核授权。删列后必须按订单所属公司解析角色与成员关系范围：启用用户+成员关系+角色+对应业务权限，并限制角色归属目标公司、成员关系属于该公司工作台子树；不能保留原 all 允许任意组织的分支。系统角色不能成为公司业务审核人。
- data/exchange_rate_reminder.go:89 使用 DataScopeIn 决定通知接收人，应按固定工作台归属与原有启用/权限条件查询，不能因删除条件让其他公司或系统角色收到业务通知。
- data/order_fee_review.go:465 与 cmd/sync-dev/fee_review.go:92 的授权快照读取 Ent Role.DataScope，需要使用推导的有效范围或明确删掉快照字段；旧审计快照不需要回写。

### 文件范围

源契约与服务：
- server/api/admin/v1/admin.proto:168（角色专属枚举，若无消费删除）、:460/:467/:477（请求响应字段，建议 reserved 原字段号/名）；server/internal/service/admin.go:321/:337/:436/:451/:558。
- server/internal/biz/admin_role.go；server/internal/biz/admin_test.go（规范化、self、提权测试）。
- server/internal/data/admin_role.go、auth.go、ent/schema/role.go、bootstrap_administrator.go、company_administrator.go、order_lock_qualification.go、order_fee_review.go、exchange_rate_reminder.go；server/cmd/sync-dev/fee_review.go。

前端：
- web/src/pages/admin/roles.tsx（RoleRecord、范围列、搜索筛选）、components/roles/RoleFormModal.tsx（默认值/校验/载荷）、components/roles/components/RoleBasicFormFields.tsx（表单项）、components/roles/roleConstants.ts（枚举/类型）；相关 roles.test.tsx、RoleFormModal.test.tsx。
- web/src/pages/admin/index.tsx:51 的“数据范围权限”文案。
- 用户角色名称展示 AvatarDropdown.tsx 只读取 roleScopes 的 roleName/roleCode，无需删除该结构；内部 auth RoleScope.data_scope 若保留有效能力描述不应再呈现为配置能力。当前用户管理组件未找到额外数据范围展示。

直接受删列影响的测试/fixture（按编译与语义逐一同步）：
- data/admin_lifecycle_integration_test.go、admin_role_anchor_integration_test.go、admin_role_delete_integration_test.go、admin_user_workspace_scope_test.go、auth_principal_test.go。
- data/dingtalk_directory_integration_test.go、dingtalk_multi_organization_integration_test.go、fee_catalog_integration_test.go、finance_commission_order_summary_integration_test.go、order_fee_review*_test.go、order_fee_supplement_transaction_integration_test.go、order_lock*_integration_test.go、permission_manifest_sync_integration_test.go、retire_wecom_migration_integration_test.go、system_company_authorization_integration_test.go、workbench_integration_test.go。
- 新增迁移验证：self 停用、旧禁用不恢复、普通角色维持启用/分配/权限、列确实消失；SQL迁移链冷启动与本库升级都通过。

生成物：
- server/api/admin/v1/admin.pb.go、admin_grpc.pb.go、admin_http.pb.go、server/openapi.yaml（生成器决定实际变化）；server/internal/data/ent 下 role.go、role_create.go、role_update.go、mutation.go、role/{role,where}.go、migrate/schema.go 等。
- web/src/services/roncin/typings.d.ts、adminService.ts（生成器决定差异）、web/src/enums.generated.ts；其余 OpenAPI 输入随命令更新。

### 生成与最小验证

- go -C server generate ./internal/data/ent（Ent 注解当前入口 ent/generate.go:6，允许 sql/lock、sql/upsert）；正式迁移从新 schema 差异生成并审阅。
- make -C server api；pnpm run generate:web-client（包含 generate:proto-constants）。不手改生成物。
- 定向 Go biz/service/data 测试覆盖角色 CRUD、越权分配、工作台隔离、锁定/费用审核、汇率提醒；迁移/真实 PostgreSQL 测试用 -v 确认非 SKIP。
- pnpm --dir web exec vitest run src/pages/admin/roles.test.tsx src/pages/admin/components/roles/RoleFormModal.test.tsx；修改文件 Biome；pnpm --dir web tsc。
- 最终跨层高风险验收 pnpm run check:fast、git diff --check；按目标操作完成提交、主分支合并与安全分支删除的证据由主代理收集。

### Related specs / References

- .trellis/spec/server/backend/role-workspace-ownership.md：工作台锚点与部门/团队共用角色库；写入非工作台拒绝。
- .trellis/spec/server/backend/auth-session-org-switch.md：会话切换与工作台候选。
- .trellis/spec/adr/0003-ent-schema-and-sql-migrations.md；server/backend/database-guidelines.md：正式迁移唯一执行真相，禁止依赖 Schema.Create，并行初始化禁忌。
- .trellis/spec/domain/{glossary,architecture-map}.md：需同步角色 scope 退役后的术语。
- 仅仓库一手代码研究，未引入外部依赖或外部文档。

## Caveats / Not Found

- 未读取生产/本地数据库真实角色数据，因此系统 organization/tree 与 self 的存量规模未知；迁移必须对存在此数据的数据库有明确可验语义。
- “固定 system=all”是治理权限作用范围变化，不能用 UI 删除视作低风险修改。
- 用户明确退役配置；内部 DataScope 保留仅用于既有治理门禁与权限能力表达，不能成为第二个可配置入口。
- 所有文件定位基于研究时工作树，实施中行号会变化。没有执行生成/迁移/测试或任何 Git 操作。
