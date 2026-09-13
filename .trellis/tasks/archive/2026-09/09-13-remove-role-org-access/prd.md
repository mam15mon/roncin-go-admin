# 移除角色「可访问组织」功能 PRD

## 背景

- 角色「可访问组织」（`role_organization_accesses`）为角色额外授予一组组织的数据读写范围，独立于当前组织与组织树解析，带独立「可写」标志。
- 机制上有三个独特能力（非子树组织白名单、跨组织联合视图、读写分离），但**当前零使用**：开发库 `role_organization_accesses` 为 0 行，全系统唯一角色为 `administrator`（data_scope=all）。
- 跨组织访问的现实口径已由**多组织成员资格 + DataScope** 覆盖（用户加入多个组织并切换，管理角色用全部/组织树范围）。
- 用户决策（2026-09-13）：确认该功能无现实用例，全链路移除，简化权限模型。

## 目标

角色「可访问组织」功能全链路删除，不留兼容分支、死代码或空转 UI。

## Requirements（范围：全链路一次性删除）

1. **Ent Schema**：删除 `schema/role_organization_access.go` 与 `role` 上的 organization_accesses 边；`go -C server generate` 重新生成；新增迁移 `DROP TABLE role_organization_accesses`（冷启动与现有库均干净应用）。
2. **proto**：删除 Role 消息中 organization_accesses 相关字段，`make -C server api` 重新生成。
3. **biz**：删除 `RoleGrant.OrganizationAccesses`、`OrganizationAccess` 类型、`Principal.ResolvePermissionOrganizationScope` 中的 OrganizationAccesses 合并分支；DataScope（all/organization_tree/organization/self）语义不变。
4. **data**：`admin_role.go` 的 `WithOrganizationAccesses`/`replaceRoleOrganizationAccesses`/转换逻辑；`auth.go` `ResolvePrincipal` 的边加载与 grant 装配。
5. **service**：角色 DTO 中可访问组织字段与校验。
6. **web**：`RoleFormModal` 可访问组织选择器与提交字段、`roles.tsx` 列展示、生成客户端重生成。
7. **测试**：`auth_principal_test`、`auth_test`、admin_role 相关用例同步删除或改写。

## 非目标

- DataScope 机制、成员资格与组织切换、权限 Manifest 与权限码均不动。
- 不做历史兼容：现网无数据，直接 DROP TABLE。

## Acceptance Criteria

- [ ] 全仓 grep 无 `OrganizationAccesses` / `role_organization_accesses` / 「可访问组织」残留（迁移 SQL 中的 DROP TABLE 语句除外）。
- [ ] 迁移在隔离 Schema 冷启动与现有 dev 库均干净应用，`migrate:server` 严格校验可用。
- [ ] 收尾验证：go build/vet/test 全绿；真实库集成用例通过；前端 tsc/biome/定向 vitest 绿；生成物幂等；gofmt 干净。
- [ ] 角色创建/更新/查询的其余行为（DataScope、权限码）不变。
