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

1. [x] 服务端错误定义：`internal/biz` 新增 `ErrAdminRoleAnchorInvalid`；服务层按
   `error-handling.md` 既有模式映射中文提示「角色只能在公司/总部维护」。
   （已完成：`internal/biz/admin_role.go` 定义
   `errors.BadRequest("ADMIN_ROLE_ANCHOR_INVALID", "角色只能在公司/总部维护")`；
   kratos 错误自带 HTTP 语义，`service` 层按既有惯例不做二次翻译，故无 service 改动。）
2. [x] data 层改造（`internal/data/admin_role.go`）：
   - `resolveRoleOrganizationID`：解析失败返回 `ErrAdminOrganizationNotFound`，删除静默回退；
   - `CreateRole` / `UpdateRole` / `DeleteRole`：锚点解析后 `resolved != 入参` 即拒绝；
   - `actorRolesPrivilegeProfiles` 内 Nil 兜底分支同步收紧。
   （已完成，另新增 `resolveRoleAnchorOrganizationID` 承载写路径守门；顺带按 AGENTS.md
   「仓储必须经 `Data.client(ctx)`」把本文件 `r.data.db` 直连全部收敛。）
3. [x] 迁移回填：`cmd/migrate` 新增幂等归一步骤 + 迁移后自检断言（无 department/team 锚定行）。
   （已完成：新增 `internal/data/role_workspace_backfill.go` 的
   `BackfillRoleWorkspaceAnchors`，在 `cmd/migrate` PostStep 内调用；断链/成环与唯一索引
   冲突均显式报错终止，无重命名/停用兜底。）
4. [x] 服务端验证：

```bash
go -C server vet ./...
go -C server test ./internal/data/ -run 'TestAdmin'
RONCIN_INTEGRATION_DATABASE_SOURCE='postgresql://roncin@127.0.0.1:5432/roncin_go_admin_integration?sslmode=disable' \
  go -C server test ./internal/data/ -run 'TestAdminDepartmentInheritsCompanyRolesPostgres|TestAdminRoleAnchor'
```

   新增集成测试 T1（部门锚点建/改/删角色被拒）、T2（未知组织显式报错）一并落在本步。
   （已完成：`vet` 通过；集成测试以显式 DSN 复跑 3/3 PASS 且 `-v` 确认无 SKIP——
   注意 `.env.local` 内**没有** `RONCIN_INTEGRATION_DATABASE_SOURCE`，不显式传 DSN 时
   用例会静默 SKIP 造成假绿。新增用例落在
   `internal/data/admin_role_anchor_integration_test.go`：T1/T2 为
   `TestAdminRoleAnchorPostgres`，另补 `TestRoleWorkspaceAnchorBackfillPostgres` 覆盖 A4。）

5. [x] 前端改造（`web/src/pages/admin/components/users/userConstants.ts`）：
   - 新增 `isWorkspaceKindValue`，`formatOrganizationHierarchyName` 复用，删死分支；
   - 新增 `formatOrganizationHierarchyName` 定向测试；
   - 验证：`pnpm --dir web exec vitest run <新增测试文件>` + 修改文件
     `pnpm --dir web exec biome check`。
   （已完成：判定值取自 `@/enums.generated` 的 `AdminOrganizationKind` 而非裸数字；
   测试文件 `userConstants.test.ts` 8 用例。`organizationKindLabels` 的裸数字键未一并收敛
   ——不在 R4/D4 范围且有其他消费点。）
6. [x] spec 固化（trellis-update-spec）：把「角色库归属工作台、部门共享、写路径显式拒绝、
   读路径显式失败」写入 `.trellis/spec/server/backend/`（建议并入
   `auth-session-org-switch.md` 或新增 `role-workspace-ownership.md`）。
   （已完成：新增 `role-workspace-ownership.md`，并在 `index.md` 登记索引与前置检查第 11 条。）

## 收尾核验（全任务完成时执行一次）

```bash
go -C server test ./internal/data/
pnpm --dir web tsc
pnpm --dir web exec vitest run src/pages/admin/components/users/
git diff --check
```

实测结果：

| 检查 | 结果 |
| --- | --- |
| `go -C server vet ./...` | PASS |
| 集成测试（显式 DSN，`-v` 确认无 SKIP） | 3/3 PASS：既有部门继承公司角色 5.41s、`TestAdminRoleAnchorPostgres` 7.36s、`TestRoleWorkspaceAnchorBackfillPostgres` 7.63s |
| `pnpm --dir web exec vitest run src/pages/admin/components/users/` | 3 文件 19 用例 PASS（新文件 8 用例） |
| `pnpm --dir web tsc` | PASS（exit 0） |
| 修改文件 `biome check` | PASS（无 issue） |
| `git diff --check` | clean |

对应验收标准：A1–A6 全部覆盖。A4 的迁移断言由
`TestRoleWorkspaceAnchorBackfillPostgres` 与实施期 `cmd/migrate` 临时 schema 端到端演练
（空库通过 / 植入部门锚定角色后归一 / 同码冲突与断链均 exit 1 且整体回滚）共同验证。

## 提交

- 单提交：`refactor(admin): 角色库锚定收敛到工作台并显式失败`，包含服务端、迁移、
  前端口径与测试（无生成物变更）；若 review 需要可拆「服务端+迁移」与「前端」两笔。
- 已提交：`ab819246`（服务端 + 迁移 + 前端 + 测试 + 任务文档）。
  本任务未做 `trellis-check` 独立复核（用户要求直接提交），改由主会话核对真实差异后
  自行复跑全部定向验证，结果见上表。

## 回滚点

- 步骤 1–2、4–5 均为纯代码，`git checkout --` 可回退；
- 步骤 3 迁移上线前先在集成库演练一次（跑两遍验证幂等）。

## 风险与备注

- 本任务无新增事务需求，沿用仓储既有 `WithTx` 封装，不手写事务。
- 迁移不得在 `txCtx` 回调中调用走原生事务的仓储方法。
- 若 Q2 探查发现前端存在部门上下文角色写入口，先报告用户再决定是否扩大前端改动。
