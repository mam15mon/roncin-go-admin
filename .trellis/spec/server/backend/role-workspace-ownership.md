# 角色库工作台归属不变量

## 1. Scope / Trigger

适用于全部角色库读写路径与组织树解析：`roles.organization_id` 的锚定口径、角色 CRUD、
成员关系角色分配、提权画像校验，以及前端「是否工作台」判定。**新增或修改角色相关接口、
组织树解析、组织 kind 判定时必须遵循**。（2026-09 角色库归属重构确立；取代此前「靠
各查询点自觉沿组织树解析、写路径直接用会话组织」的调用约定。）

## 2. Signatures

```go
// internal/data/auth.go —— 工作台模型的唯一口径（登录/切换工作台同一套）
isWorkspaceKind(kind string) bool                     // 工作台 = headquarters | company
workspaceAncestorID(nodes, organizationID) uuid.UUID  // 自身或最近工作台祖先；失败返回 uuid.Nil

// internal/data/admin_role.go —— 角色锚点解析（读路径）
resolveRoleOrganizationID(ctx, client, organizationID) (uuid.UUID, error)
//   解析失败（组织不存在/断链/成环）→ biz.ErrAdminOrganizationNotFound，不回退入参

// internal/data/admin_role.go —— 角色锚点守门（写路径）
resolveRoleAnchorOrganizationID(ctx, client, organizationID) (uuid.UUID, error)
//   resolved != 入参 → biz.ErrAdminRoleAnchorInvalid

// internal/data/role_workspace_backfill.go —— 迁移期存量归一（幂等）
BackfillRoleWorkspaceAnchors(ctx, database transactionStarter) error

// internal/biz/admin_role.go
ErrAdminRoleAnchorInvalid = errors.BadRequest("ADMIN_ROLE_ANCHOR_INVALID", "角色只能在公司/总部维护")

// web/src/pages/admin/components/users/userConstants.ts —— 前端唯一出口
isWorkspaceKindValue(kind?: number): boolean  // 取自 @/enums.generated，禁止裸数字
```

## 3. Contracts

- **归属**：角色库只归属工作台节点（总部/公司）；部门与团队**共享所属工作台的角色库**，
  不自建角色库。公司之间不共享（既有隔离设计）。
- **读路径**：`ListRoles` / `GetRole` / `rolesPrivilegeProfiles` / `actorRolesPrivilegeProfiles`
  一律先把入参组织解析为工作台锚点再查；解析不出工作台即显式报错。
- **写路径**：`CreateRole` / `UpdateRole` / `DeleteRole` 在事务内先校验锚点，**非工作台锚点
  一律拒绝**，不做静默归一（把部门上下文的写入悄悄改挂到公司，只是另一种纠错掩盖）。
  行过滤使用解析后的锚点，与读路径对称。
- **仓储取客户端**：必须经 `Data.client(ctx)`，禁止直连 `r.data.db`——直连会静默落到
  共享事务之外。
- **前端**：工作台判定收敛为 `isWorkspaceKindValue` 单一出口，层级名展示复用它；
  前端不复制第二套 kind 规则，也不对字符串形态的 kind 做兜底猜测。
- **不变量**：`roles.organization_id` 只允许指向 `kind ∈ {headquarters, company}` 的节点，
  由迁移自检断言长期守门。

## 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| 写路径传入部门/团队锚点（`resolved != 入参`） | `ErrAdminRoleAnchorInvalid`（400「角色只能在公司/总部维护」） |
| 读路径组织不存在 / 父链断链 / 父链成环（`workspaceAncestorID` 返回 `uuid.Nil`） | `ErrAdminOrganizationNotFound`（404），不回退入参继续查询 |
| `actorRolesPrivilegeProfiles` 解析不出工作台，或工作台子树成员范围为空 | `ErrAdminOrganizationNotFound`，不再回退为原始组织 |
| 迁移归一后仍存在部门/团队锚定角色 | 迁移自检断言失败，报错终止发版 |
| 归一与既有 `(organization_id, code)` 唯一索引冲突 | 迁移显式报错终止，交人工确认，不自动改名/停用/丢弃 |
| 角色仍被成员关系引用而删除 | `ErrAdminRoleAssigned`（行为不变） |

## 5. Good / Base / Bad Cases

- Good：部门上下文查询角色，沿父链解析到所属公司角色库，返回公司角色集合。
- Good：在部门上下文调用角色管理接口，收到「角色只能在公司/总部维护」而非静默改挂。
- Base：组织停用不影响解析——`workspaceAncestorID` 不关注启用态，停用部门仍可解析到公司。
- Bad：`resolveRoleOrganizationID` 解析失败返回入参继续查询——未知组织凭空获得「自己的
  角色库」，读路径静默纠错。
- Bad：`CreateRole` 直接用 `organizationID` 写入，安全性只靠「会话组织恒为工作台」这一
  隐式保证（该保证由 `SwitchOrganization` 的候选集过滤提供，属别处契约，不是角色契约）。
- Bad：前端用 `String(kind).includes('COMPANY')` 猜测工作台——字符串形态在本契约中属
  不匹配，不做兼容。

## 6. Tests Required

- 真实 PostgreSQL 集成（带 `RONCIN_INTEGRATION_DATABASE_SOURCE`；**未配置会全部静默
  skip 造成假绿，必须用 `-v` 确认无 SKIP**）：
  - 部门锚点 `CreateRole` / `UpdateRole` / `DeleteRole` → `ErrAdminRoleAnchorInvalid`；
  - 未知组织调 `ListRoles` → `ErrAdminOrganizationNotFound`；
  - 既有「部门继承公司角色」用例保持通过（读写对称的回归底线）；
  - 存量归一回填：幂等（跑两遍结果一致）、自检断言、断链与唯一冲突显式失败。
- 前端定向 vitest：`isWorkspaceKindValue`（总部/公司 true，部门/团队/未提供 false）、
  `formatOrganizationHierarchyName`（部门拼「公司 / 部门」、多级中间层、工作台返回自身名、
  断链/成环不追加不可达上级）。

## 7. Wrong vs Correct

### Wrong

```go
func resolveRoleOrganizationID(ctx context.Context, client *ent.Client, organizationID uuid.UUID) (uuid.UUID, error) {
	nodes, _ := loadOrganizationTree(ctx, client)
	effectiveID := workspaceAncestorID(authOrganizationNodes(nodes), organizationID)
	if effectiveID == uuid.Nil {
		return organizationID, nil // 静默回退：未知组织也有「自己的」角色库
	}
	return effectiveID, nil
}

// 写路径直接用原始入参，锚点正确性靠调用方的隐式约定
created, saveErr = tx.Role.Create().SetOrganizationID(organizationID)...
```

### Correct

```go
// 读路径：解析失败显式失败
effectiveID := workspaceAncestorID(authOrganizationNodes(nodes), organizationID)
if effectiveID == uuid.Nil {
	return uuid.Nil, biz.ErrAdminOrganizationNotFound
}

// 写路径：非工作台锚点一律拒绝
roleOrgID, resolveErr := resolveRoleAnchorOrganizationID(ctx, tx.Client(), organizationID)
if resolveErr != nil {
	return resolveErr
}
created, saveErr = tx.Role.Create().SetOrganizationID(roleOrgID)...
```

## 8. Common Mistakes

- **迁移里 `SELECT "parent_id"::text` 扫到 NULL 会炸**：PostgreSQL 对 NULL 拼接 `::text`
  仍返回 NULL，`rows.Scan(&rawParentID)` 到 `string` 报
  `converting NULL to string is unsupported`。必须用 `sql.NullString` 并判 `Valid`。
  该分支只在存在违规行时触发，没有对应测试会一路藏到上线。
- **集成测试假绿**：未设置 `RONCIN_INTEGRATION_DATABASE_SOURCE` 时相关用例静默 SKIP，
  加 `-v` 看 SKIP 才能确认真的跑过。
- **角色 CRUD 的 service 层不做二次错误翻译**：kratos 错误自带 HTTP 语义，中文提示在
  `internal/biz` 定义处一次写定，不要在 `service` 再包一层。
- **`organizationKindLabels` 仍是裸数字键**：它与 `isWorkspaceKindValue` 不属同一收敛批次，
  改动前先确认所有消费点，不要只改一处造成两套口径。
