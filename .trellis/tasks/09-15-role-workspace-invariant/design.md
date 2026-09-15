# 技术设计：角色库工作台归属不变量固化

## 目标不变量

`roles.organization_id` 只允许指向工作台节点（kind = `headquarters` | `company`）。
部门/团队节点的角色读取、分配与提权校验一律解析到 `workspaceAncestorID`（复用
`internal/data/auth.go` 既有纯函数，与登录/切换工作台同一口径）。

## 现状与差距

- 读路径已收敛：`ListRoles` / `GetRole` / `rolesPrivilegeProfiles` / `rolesForOrganization`
  均经 `resolveRoleOrganizationID`；`actorRolesPrivilegeProfiles` 按工作台子树收集成员关系角色。
- 差距：
  1. `CreateRole` / `UpdateRole` / `DeleteRole` 直接使用原始 organizationID，安全性仅靠
     「会话组织恒为工作台」这一隐式保证；
  2. `resolveRoleOrganizationID` 对树异常（未知节点/断链/成环 → `workspaceAncestorID`
     返回 `uuid.Nil`）静默回退为输入值继续查询；
  3. 存量角色可能存在部门/团队锚定行（待探查确认）；
  4. 前端 `userConstants.ts` 的 `formatOrganizationHierarchyName` 以数字字面量 +
     `String(kind).includes('COMPANY')` 死分支复制了工作台判定。

## 决策

### D1 写路径拒绝，而非静默归一

`CreateRole` / `UpdateRole` / `DeleteRole` 在 data 层先解析锚点：`resolved != 入参`
即锚点不是工作台节点，返回新增业务错误 `biz.ErrAdminRoleAnchorInvalid`；服务层映射
中文提示「角色只能在公司/总部维护」（遵循 `error-handling.md` 既有映射模式，HTTP 4xx）。

- 拒绝优于归一：在部门上下文管理角色不是合法操作，静默归一只是另一种纠错掩盖。
- `UpdateRole` / `DeleteRole` 的目标行过滤同步改用 resolved 口径，与读路径对称
  （入参必为工作台时行为不变；传入部门锚点时被 D1 拒绝，不会走到行过滤）。

### D2 读路径显式失败

`resolveRoleOrganizationID` 移除「`uuid.Nil` → 返回原值」回退：解析失败直接返回
`biz.ErrAdminOrganizationNotFound`。`actorRolesPrivilegeProfiles` 内现有的
Nil 兜底分支同步收紧为显式错误。

- 影响评估：`ListOrganizationRoles`（前端拉取成员关系角色选项）从「未知组织返回空列表」
  变为显式组织不存在错误；组织无删除路径（仅 enabled 开关），且 `workspaceAncestorID`
  不关注启用态，部门停用仍可解析到公司，无行为回归。

### D3 存量归一随迁移执行

在 `cmd/migrate` 的数据同步段新增幂等回填（参照拼音检索键回填先例，集中封装、
不散落业务 SQL）：

```sql
-- 伪代码：按父链解析工作台祖先
UPDATE roles SET organization_id = <workspace 祖先>
WHERE organization_id IN (kind = 'department' | 'team' 的节点);
```

- 迁移完成后自检断言：不存在 department/team 锚定行，违反则迁移报错终止（对应 A4）。
- `(organization_id, code)` 唯一索引冲突：实施前先探查存量（开发库预期为空）；若存在
  真实冲突，默认策略为「保留公司现有同名角色；部门角色重命名为 `<code>_dept_<短id>`
  并 `enabled=false`」，避免静默丢弃配置；执行前向用户展示冲突清单（对应 PRD Q1）。
- 不新增 schema 字段、不改 Ent 结构，迁移仅为数据归一，可重复执行。

### D4 前端工作台判定单一出口

`userConstants.ts` 新增导出 `isWorkspaceKindValue(kind: number): boolean`
（1=总部、2=公司，注释标明与后端 `organization.Enum` / proto 序号的对应关系）；
`formatOrganizationHierarchyName` 复用之；删除字符串猜测死分支。前端「工作台 kind」
事实点收敛为 `organizationKindLabels` + `isWorkspaceKindValue`。

## 影响面与兼容

- 无 proto/OpenAPI 契约变更、无权限码变更、无 Ent schema 变更 → 不需要
  `generate:web-client` / `generate:permission-keys`。
- 新增业务错误仅影响服务层错误映射与前端提示文案。
- 回滚：代码 `git revert` 即可；迁移为幂等数据归一，回滚代码后重跑迁移不会产生新副作用
  （无部门锚定行时跳过）。

## 测试设计

集成测试（`RONCIN_INTEGRATION_DATABASE_SOURCE`，隔离 Schema 自动建删）：

- T1 部门锚点 `CreateRole` / `UpdateRole` / `DeleteRole` → `ErrAdminRoleAnchorInvalid`
  （A2）。
- T2 未知组织 ID 调用 `ListRoles` → `ErrAdminOrganizationNotFound`（A3）。
- T3 现有 `TestAdminDepartmentInheritsCompanyRolesPostgres` 保持通过（A1）。

单测/前端：

- `resolveRoleOrganizationID` 显式失败路径按既有测试基建补充（若纯函数层可测则不引 DB）。
- `formatOrganizationHierarchyName` 定向 vitest：部门拼「公司 / 部门」、多级部门中间层、
  总部/公司直接返回自身名（A5）。
