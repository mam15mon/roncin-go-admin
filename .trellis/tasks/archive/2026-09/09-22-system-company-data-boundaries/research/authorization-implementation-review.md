# Research: 系统工作台实现中的授权复核

- Query: 当前实施中的公司准入、权限授予、公司初始管理员及经营读取隔离是否闭环。
- Scope: internal
- Date: 2026-09-22

## Findings

本报告为并行实现期间的代码快照审查；已实时将问题发送主会话和 system_workspace_impl，修复状态必须由最终检查重新确认。

### 已确认实现方向正确

- `data/auth.go:ResolvePrincipal` 已取消 bootstrap 合成，候选取真实 membership；RotateSession 事务内复核目标工作台和成员资格。
- `biz/auth.go:permissionAvailableInWorkspace` 过滤 system 的经营读写，workspacePermissionScope 将公司权限范围截到本公司。
- `access/manifest.go:PermissionAllowedInWorkspace` 被运行时、角色写入、manifest 同步共同使用；公司角色无法新写公共字典治理权限。
- `data/company_administrator.go` 初始化公司专属管理员角色但不加任何用户成员，系统随后可显式授予。
- `data/admin_role.go:ListRoles` 和 `data/admin_organization.go:ListOrganizations/UpdateOrganization` 已加工作台组织范围。

### 发现 1：成员关系 CRUD 直接跨组织（高危，实施代理已开始修）

`biz/admin_user_membership.go` 的 Create/Update/Delete/List 原来缺少目标组织scope校验，空 roleIDs 还会跳过 validateRolesPrivilege。HTTP规则 `server/internal/server/access_rules_gen.go` 仅 UserUpdate+ALL；公司 administrator 持 ALL 可添加自己到system，或带已知system roleID借 IsSuperAdmin直接授予系统角色。List任意user、Update/Delete任意membership同样缺口。

实施代理已告知并可见新增：biz validateRolesPrivilege在空roles前检查company subtree；data membership target复核及查询predicate。需要PG验证空roles、已知system角色、foreign company、list/update/delete全部拒绝，system正常为公司建立首个成员成功。

### 发现 2：公司管理员可接管跨公司账号（高危）

`data/admin_user.go:ResetUserPassword` 仅要求目标在本公司有启用成员资格，就可改全局密码/用户名并撤销所有会话。同一目标还在B公司或system时，A管理员可登录被重置账号获得B/system身份。

`TerminateUser` 确认A成员后删除目标所有角色、停用所有公司membership与全局账号；违反公司独立且能停用系统管理员。

`UpdateUser` 修改全局 displayName/email/enabled，影响其他公司的同一账号。建议全局密码、启停、资料修改限system；公司管理仅本公司成员/角色，离职走本公司membership停用。系统授权人仍应有明确全局账号管理权限。不能仅用前端隐藏。

### 发现 3：有限系统管理者自动变全权系统管理员（高危）

`biz/admin_role.go:getActorPrivilegeProfile` 对system身份且持任意 RoleCreate/RoleUpdate/UserCreate/UserUpdate直接返回 IsSuperAdmin。该豁免未限定目标公司：有限系统人员管理员可把自己授system administrator；有限角色编辑者可创建全权system角色。

建议将“可为公司授初始角色”限定目标company，与target system的真实角色提权校验分开。不能借缺少公司经营权限为理由，给所有system目标的授予都开豁免。

### 发现 4：公司用户列表泄露其他公司成员摘要（中等）

`data/admin_user.go:adminUserRow` candidates筛scope，但active收集全部组织，随后`membershipToUser`返回全部Organizations。`service/admin.go:ListUsers` 通过 UserRead+ALL 控制是否展示，公司ALL仍成立。应按工作台scope过滤摘要，不让公司A看到B/system成员情况。

### 邻接路径与闭环判断

- CreateUser：service入参organizationID来自principal，data不使用input.CurrentOrganizationID指定目标；未发现该路径跨组织创建。
- UpdateUser：锚定membership来自本公司scope，角色目标正确；风险在全局字段。
- 旧 AuthorizeWeComUser/AuthorizeDingTalkUser 使用validateRolesPrivilege，新增目标scope可阻断公司授system，但仍需禁止普通待审批用户把system作为入职目标。
- `DingTalkRegistrationUsecase.validateRolePrivilege` 系统注册管理者可授公司角色；目标必须company，角色归属继续由仓储复核。未指定公司收口要求先转公司再审批。
- 新公司administrator为organization scope，现会员CRUD HTTP要求ALL可能导致公司管理员不能管理本公司部门成员，需要同步审视入口最低scope。组织管理范围应本公司子树，经营实体才固定公司ID。

### 相关规范与引用

内部依据：任务prd/design/implement，server/backend/auth-session-org-switch.md、role-workspace-ownership.md、dingtalk-registration-approval.md，以及此前 system-workspace-authorization.md。
外部参考：无；本次只审查仓库实现。

## Caveats / Not Found

- 没有执行测试或写产品代码；上述高危问题均已通知实施代理及主会话。
- 并行编辑中行号可能移动，使用函数名定位更稳妥。
- 发现1已有修复在写，发现2/3/4已通知，不能将这份中途研究报告作为最终通过结论。

## 最终定向复核（三个已修问题）

2026-09-22 再次核查当前实现：

1. **成员范围：对应代码检查通过。** biz validateRolesPrivilege 在空角色分支前检查公司子树；仓储 Create 事务内校验目标，List/Get/Update/Delete使用 adminMembershipPredicate，不能凭 company ALL 越界。公司成员摘要也已过滤scope。
2. **全局账号：对应代码检查通过。** biz及data的 UpdateUser/TerminateUser/ResetUserPassword 均显式拒绝非system请求身份；公司不能通过共享账号改密码接管system/B身份。内部无principal路径保留给种子/测试，HTTP始终要求身份。
3. **system角色委派：对应代码检查通过。** getActorPrivilegeProfile已撤销“有任何管理权限即超级管理员”。validateRolesPrivilege仅在system操作者对非system目标授公司角色时开放委派；target system仍使用真实角色画像。创建角色遇 code=administrator 且非超级管理员时checkPrivilegeEscalation拒绝，普通名字副本也逐权限及scope校验；data CreateRole/UpdateRole 再按目标workspace过滤权限，公司管理员无法借副本注入系统权限。

### 测试证据及精确限制

- `data/system_company_authorization_integration_test.go:TestSystemCompanyAdministratorAuthorizationBoundaries` 覆盖：不自动生成成员、公司角色拒系统权限、有限system操作者拒授system administrator、允许显式授公司管理员、公司拒全局密码重置和停用、空roles不能加入system、成员List/Get/Delete拒system。
- `biz/admin_test.go:TestCheckPrivilegeEscalation` 覆盖管理员名称及额外权限/更大范围拒绝；`TestAdminUsecaseUpdateRoleRejectsAdministratorAndScopeEscalation` 覆盖修改administrator与扩大scope拒绝。
- 尚未看到新建任意名字system角色副本再自赋的完整PG用例，也未看到带非空system roleID的公司CreateMembership和跨组织UpdateMembership独立断言；对应防护逻辑存在，属于定向测试覆盖不完整，不是此次复核发现的绕过。
- 本次未自行执行测试，不替代主会话测试结果。三个指定问题未发现剩余代码漏洞。

## 最后专项：旧外部账号授权入口

仅检查 `AuthorizeWeComUser` / `AuthorizeDingTalkUser`：当前两biz入口未要求system，API规则仅专属authorize权限+ALL。共享仓储 `data/admin_user.go:authorizePendingUser` 虽要求目标已有外部身份且账号disabled、来源工作台含该账号membership，仍会：全局启用账号、改姓名/email、删除其全部公司角色、停用全部公司membership，再在目标公司建立角色。

因此公司ALL管理员可对被系统停用但仍保留本公司membership的外部账号重启，并破坏其其他公司或系统成员身份。**不应称为外部身份绑定直接接管**：此接口没有改unionID、userID或密码，且enabled账号会被拒绝；已证实的是绕过全局停用和跨公司身份修改。

已通知 system_workspace_impl 在两biz入口及共享data helper拒非system请求，并补公司调用两入口拒绝测试。公司正常注册申请继续走既有注册审批用例，不需开放这个旧全局账号授权入口。尚未对该最后修复作通过判断。
