# 实施清单 (implement.md)

> 本任务为全链路删除型重构，按层推进，每层完成后验证编译再进下一层。

## Phase 1: 数据与契约
- [ ] **Step 1.1**: 删除 `server/internal/data/ent/schema/role_organization_access.go` 及 `role.go` 中 organization_accesses 边；`go -C server generate`；
- [ ] **Step 1.2**: 新增迁移 SQL：`DROP TABLE IF EXISTS role_organization_accesses;`（与既有迁移命名/风格一致）；
- [ ] **Step 1.3**: `server/api` 下删除 proto 中 Role 的 organization_accesses 字段；`make -C server api`；`pnpm run generate:web-client`。

## Phase 2: 服务端逻辑
- [ ] **Step 2.1**: biz：删除 `OrganizationAccess` 类型、`RoleGrant.OrganizationAccesses`、`ResolvePermissionOrganizationScope` 中 OrganizationAccesses 合并分支（`server/internal/biz/auth.go`）；
- [ ] **Step 2.2**: data：`admin_role.go`（WithOrganizationAccesses、replaceRoleOrganizationAccesses、toBiz 转换）、`auth.go` ResolvePrincipal（边加载与装配）；
- [ ] **Step 2.3**: service：角色 DTO 字段与校验删除；
- [ ] **Step 2.4**: 测试同步：`auth_principal_test.go`、`auth_test.go`、admin 角色用例、集成测试（如有引用）。

## Phase 3: 前端
- [ ] **Step 3.1**: `RoleFormModal.tsx` 删除可访问组织选择器与提交字段；
- [ ] **Step 3.2**: `roles.tsx` 删除列展示；
- [ ] **Step 3.3**: 相关前端测试同步。

## Phase 4: 验证
- [ ] **Step 4.1**: 全仓残留 grep（`OrganizationAccesses|role_organization_accesses|可访问组织`，迁移 DROP 语句除外）；
- [ ] **Step 4.2**: `go -C server build ./... && go -C server vet ./...`；`go -C server test ./internal/...`；
- [ ] **Step 4.3**: 真实库：注入 `RONCIN_INTEGRATION_DATABASE_SOURCE`（取 `.env.local`）跑 data 相关集成测试 + `pnpm run migrate:dev`；
- [ ] **Step 4.4**: 前端：定向 vitest（roles 相关测试文件）、`pnpm --dir web tsc`、biome 定向检查；
- [ ] **Step 4.5**: 生成物幂等（重跑生成命令无 diff）；`gofmt -l`、`git diff --check` 干净。
