# 执行计划：角色库工作台归属不变量固化

## 前置确认

- [x] `git status` 干净；HEAD 在 `91df65f2` 之后。（已完成：HEAD `d6336a66`，仅
  `.trellis/` 任务目录变化，无代码改动。）

- [x] 探查存量部门/团队锚定角色（开发库与集成库）——**已完成，两库均无违规行，
  PRD Q1 自然消解**：

```sql
SELECT r.id, r.code, r.organization_id, o.kind
FROM roles r JOIN organizations o ON o.id = r.organization_id
WHERE o.kind IN ('department', 'team');
```

实测结果（2026-09-15）：

- `roncin_go_admin`：organizations 为 headquarters 1 / company 1 / department 2；
  roles 仅挂在 company(1) 与 headquarters(2)，**部门锚定 0 行**。
- `roncin_go_admin_integration`：organizations 与 roles 均为空（按测试建删 Schema）。

结论：存量归一无冲突行，design D3 的「重命名加后缀并停用」分支**不需要实现**；
迁移仍按 D3 实现幂等归一 + 迁移后自检断言（A4），因为断言是长期不变量守门，
不依赖当前存量恰好干净。唯一索引 `role_organization_id_code (organization_id, code)`
已在两库存在，归一写入需保持其约束可满足。

- [x] 探查 PRD Q2（前端是否存在部门上下文角色写入口）——**已完成，不存在，无需扩大前端改动**：

  - `web/src/pages/admin/roles.tsx` 全文无 `organizationId`，`adminServiceListRoles`
    / `adminServiceCreateRole` 均不传组织参数；
  - 服务端 `internal/service/admin.go` 的 `CreateRole` / `UpdateRole` / `DeleteRole`
    一律取 `principal.Organization.ID`；
  - 会话组织只可能落在工作台：`SwitchOrganization` 要求目标在
    `listOrganizationChoices` 的候选集内，而该候选集按 `isWorkspaceKind` 过滤
    （`internal/data/auth.go:662`）。

结论：D1 属纵深防御而非当前可达缺陷——现有安全性确实只靠「会话组织恒为工作台」
这一隐式保证，与 PRD 描述一致。前端仅需按 R4/D4 收敛判定出口，不加写入口管控。

## 步骤

1. [ ] 服务端错误定义：`internal/biz` 新增 `ErrAdminRoleAnchorInvalid`；服务层按
   `error-handling.md` 既有模式映射中文提示「角色只能在公司/总部维护」。
2. [ ] data 层改造（`internal/data/admin_role.go`）：
   - `resolveRoleOrganizationID`：解析失败返回 `ErrAdminOrganizationNotFound`，删除静默回退；
   - `CreateRole` / `UpdateRole` / `DeleteRole`：锚点解析后 `resolved != 入参` 即拒绝；
   - `actorRolesPrivilegeProfiles` 内 Nil 兜底分支同步收紧。
3. [ ] 迁移回填：`cmd/migrate` 新增幂等归一步骤 + 迁移后自检断言（无 department/team 锚定行）。
4. [ ] 服务端验证：

```bash
go -C server vet ./...
go -C server test ./internal/data/ -run 'TestAdmin'
RONCIN_INTEGRATION_DATABASE_SOURCE='postgresql://roncin@127.0.0.1:5432/roncin_go_admin_integration?sslmode=disable' \
  go -C server test ./internal/data/ -run 'TestAdminDepartmentInheritsCompanyRolesPostgres|TestAdminRoleAnchor'
```

   新增集成测试 T1（部门锚点建/改/删角色被拒）、T2（未知组织显式报错）一并落在本步。

5. [ ] 前端改造（`web/src/pages/admin/components/users/userConstants.ts`）：
   - 新增 `isWorkspaceKindValue`，`formatOrganizationHierarchyName` 复用，删死分支；
   - 新增 `formatOrganizationHierarchyName` 定向测试；
   - 验证：`pnpm --dir web exec vitest run <新增测试文件>` + 修改文件
     `pnpm --dir web exec biome check`。
6. [ ] spec 固化（trellis-update-spec）：把「角色库归属工作台、部门共享、写路径显式拒绝、
   读路径显式失败」写入 `.trellis/spec/server/backend/`（建议并入
   `auth-session-org-switch.md` 或新增 `role-workspace-ownership.md`）。

## 收尾核验（全任务完成时执行一次）

```bash
go -C server test ./internal/data/
pnpm --dir web tsc
pnpm --dir web exec vitest run src/pages/admin/components/users/
git diff --check
```

## 提交

- 单提交：`refactor(admin): 角色库锚定收敛到工作台并显式失败`，包含服务端、迁移、
  前端口径与测试（无生成物变更）；若 review 需要可拆「服务端+迁移」与「前端」两笔。

## 回滚点

- 步骤 1–2、4–5 均为纯代码，`git checkout --` 可回退；
- 步骤 3 迁移上线前先在集成库演练一次（跑两遍验证幂等）。

## 风险与备注

- 本任务无新增事务需求，沿用仓储既有 `WithTx` 封装，不手写事务。
- 迁移不得在 `txCtx` 回调中调用走原生事务的仓储方法。
- 若 Q2 探查发现前端存在部门上下文角色写入口，先报告用户再决定是否扩大前端改动。
