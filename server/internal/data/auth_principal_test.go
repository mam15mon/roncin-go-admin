package data

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
)

func TestResolvePrincipalBuildsRoleGrantOnlyForEnabledRoles(t *testing.T) {
	roleID := uuid.New()
	role := &ent.Role{
		ID:        roleID,
		Code:      "operator",
		DataScope: roleent.DataScopeOrganizationTree,
		Enabled:   true,
		Edges: ent.RoleEdges{
			Permissions: []*ent.Permission{{Key: "business.order.se.read"}},
		},
	}

	grant, enabled := roleGrantFromEntRole(role)
	if !enabled {
		t.Fatal("启用角色应进入 ResolvePrincipal 的 RoleGrants")
	}
	if grant.RoleID != roleID || grant.RoleCode != "operator" || grant.DataScope != biz.DataScopeOrganizationTree {
		t.Fatalf("角色来源信息 = %#v", grant)
	}
	if _, ok := grant.Permissions["business.order.se.read"]; !ok {
		t.Fatalf("角色权限未保留: %#v", grant.Permissions)
	}

	role.Enabled = false
	if _, enabled := roleGrantFromEntRole(role); enabled {
		t.Fatal("停用角色不应进入 ResolvePrincipal 的 RoleGrants")
	}
}

func TestResolvePrincipalOrganizationBaseCurrencyRetainsDisabledAncestorLookup(t *testing.T) {
	parentID := uuid.New()
	childID := uuid.New()
	currency := "CNY"
	parent := &ent.Organization{ID: parentID, Enabled: false, BaseCurrency: &currency}
	child := &ent.Organization{ID: childID, Enabled: true, ParentID: &parentID}

	actual, err := resolvePrincipalOrganizationBaseCurrency(child, map[uuid.UUID]*ent.Organization{
		parentID: parent,
		childID:  child,
	})
	if err != nil {
		t.Fatalf("停用父组织的本币继承不应改变: %v", err)
	}
	if actual != currency {
		t.Fatalf("本币 = %q，期望 %q", actual, currency)
	}
}

// authPrincipalFixture 构造 bootstrap 管理员全组织穿透解析的隔离数据：
// bootstrap 用户仅持总部成员关系（无角色），另有其无成员关系的分公司与一个停用组织；
// 普通用户同样仅持总部成员关系，用于普通用户路径回归。
type authPrincipalFixture struct {
	t               *testing.T
	repo            *authRepo
	data            *Data
	bootstrapUserID uuid.UUID
	normalUserID    uuid.UUID
	headquarters    uuid.UUID
	branch          uuid.UUID
	disabledOrg     uuid.UUID
	permissionKeys  []string
	sessionExpiry   time.Time
}

func newAuthPrincipalFixture(t *testing.T) *authPrincipalFixture {
	t.Helper()
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	suffix := uuid.NewString()[:12]
	headquarters, err := data.db.Organization.Create().
		SetCode("HQ-" + suffix).
		SetName("总部集团-" + suffix).
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建总部组织失败: %v", err)
	}
	branch, err := data.db.Organization.Create().
		SetCode("BJ-" + suffix).
		SetName("北京财务-" + suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建分公司组织失败: %v", err)
	}
	disabledOrg, err := data.db.Organization.Create().
		SetCode("OFF-" + suffix).
		SetName("停用组织-" + suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		SetEnabled(false).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建停用组织失败: %v", err)
	}
	permissionKeys := []string{"test.bootstrap.read." + suffix, "test.bootstrap.manage." + suffix}
	for _, key := range permissionKeys {
		if _, err := data.db.Permission.Create().
			SetKey(key).
			SetName("穿透测试权限-" + key).
			SetGroup("test").
			SetDescription("bootstrap 穿透集成测试权限").
			Save(ctx); err != nil {
			t.Fatalf("创建权限失败: %v", err)
		}
	}
	bootstrapUser, err := data.db.User.Create().
		SetUsername("bootstrap-" + suffix).
		SetDisplayName("穿透测试管理员-" + suffix).
		SetIsBootstrapAdmin(true).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 bootstrap 管理员失败: %v", err)
	}
	if _, err := data.db.Membership.Create().
		SetUserID(bootstrapUser.ID).
		SetOrganizationID(headquarters.ID).
		SetPrimary(true).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建 bootstrap 管理员成员关系失败: %v", err)
	}
	normalUser, err := data.db.User.Create().
		SetUsername("member-" + suffix).
		SetDisplayName("普通用户-" + suffix).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建普通用户失败: %v", err)
	}
	if _, err := data.db.Membership.Create().
		SetUserID(normalUser.ID).
		SetOrganizationID(headquarters.ID).
		SetPrimary(true).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建普通用户成员关系失败: %v", err)
	}
	return &authPrincipalFixture{
		t:               t,
		repo:            NewAuthRepo(data).(*authRepo),
		data:            data,
		bootstrapUserID: bootstrapUser.ID,
		normalUserID:    normalUser.ID,
		headquarters:    headquarters.ID,
		branch:          branch.ID,
		disabledOrg:     disabledOrg.ID,
		permissionKeys:  permissionKeys,
		sessionExpiry:   time.Now().Add(time.Hour),
	}
}

func TestAuthRepoResolvePrincipalBootstrapAdminWithoutMembershipSynthesizesAdministratorGrant(t *testing.T) {
	fixture := newAuthPrincipalFixture(t)

	principal, err := fixture.repo.ResolvePrincipal(context.Background(), fixture.bootstrapUserID, fixture.branch)
	if err != nil {
		t.Fatalf("bootstrap 管理员解析无成员关系组织失败: %v", err)
	}
	if principal.Organization.ID != fixture.branch {
		t.Fatalf("当前组织应为目标组织: %#v", principal.Organization)
	}
	if !principal.IsBootstrapAdmin {
		t.Fatal("主体应保留 bootstrap 管理员标记")
	}
	if len(principal.RoleGrants) != 1 {
		t.Fatalf("合成授权数量 = %d，期望 1: %#v", len(principal.RoleGrants), principal.RoleGrants)
	}
	grant := principal.RoleGrants[0]
	if grant.RoleCode != "administrator" || grant.DataScope != biz.DataScopeAll {
		t.Fatalf("合成授权 = %#v，期望 administrator + 最宽数据范围", grant)
	}
	for _, key := range fixture.permissionKeys {
		if !principal.HasPermission(key) {
			t.Fatalf("合成授权缺少权限键 %s: %#v", key, grant.Permissions)
		}
	}
	organizationIDs := make(map[uuid.UUID]biz.Organization, len(principal.Organizations))
	for _, organization := range principal.Organizations {
		organizationIDs[organization.ID] = organization
	}
	if _, ok := organizationIDs[fixture.headquarters]; !ok {
		t.Fatalf("候选组织应包含总部: %#v", principal.Organizations)
	}
	if _, ok := organizationIDs[fixture.branch]; !ok {
		t.Fatalf("候选组织应包含无成员关系的分公司: %#v", principal.Organizations)
	}
	if _, ok := organizationIDs[fixture.disabledOrg]; ok {
		t.Fatalf("候选组织不应包含停用组织: %#v", principal.Organizations)
	}
}

func TestAuthRepoResolvePrincipalBootstrapAdminRejectsDisabledAndUnknownOrganization(t *testing.T) {
	fixture := newAuthPrincipalFixture(t)
	ctx := context.Background()

	tests := []struct {
		name           string
		organizationID uuid.UUID
	}{
		{name: "停用组织", organizationID: fixture.disabledOrg},
		{name: "未知组织", organizationID: uuid.New()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := fixture.repo.ResolvePrincipal(ctx, fixture.bootstrapUserID, test.organizationID); err != biz.ErrOrganizationForbidden {
				t.Fatalf("解析错误 = %v，期望 ErrOrganizationForbidden", err)
			}
		})
	}
}

func TestAuthRepoResolvePrincipalNormalUserWithoutMembershipStillForbidden(t *testing.T) {
	fixture := newAuthPrincipalFixture(t)
	ctx := context.Background()

	// 普通用户路径零变化：无成员关系的组织依旧不可解析。
	if _, err := fixture.repo.ResolvePrincipal(ctx, fixture.normalUserID, fixture.branch); err != biz.ErrOrganizationForbidden {
		t.Fatalf("普通用户解析无成员关系组织错误 = %v，期望 ErrOrganizationForbidden", err)
	}
	principal, err := fixture.repo.ResolvePrincipal(ctx, fixture.normalUserID, fixture.headquarters)
	if err != nil {
		t.Fatalf("普通用户解析成员组织失败: %v", err)
	}
	if principal.Organization.ID != fixture.headquarters {
		t.Fatalf("普通用户当前组织 = %v，期望总部", principal.Organization.ID)
	}
	for _, organization := range principal.Organizations {
		if organization.ID == fixture.branch {
			t.Fatalf("普通用户候选组织不应包含无成员关系组织: %#v", principal.Organizations)
		}
	}
}

func TestAuthRepoListEnabledOrganizationsForBootstrapAdminCandidates(t *testing.T) {
	fixture := newAuthPrincipalFixture(t)

	choices, err := fixture.repo.ListEnabledOrganizations(context.Background(), fixture.bootstrapUserID)
	if err != nil {
		t.Fatalf("查询全部启用组织候选失败: %v", err)
	}
	byID := make(map[uuid.UUID]biz.OrganizationChoice, len(choices))
	for _, choice := range choices {
		byID[choice.OrganizationID] = choice
	}
	headquartersChoice, ok := byID[fixture.headquarters]
	if !ok || !headquartersChoice.IsDefault {
		t.Fatalf("总部候选应存在并标记默认: %#v", headquartersChoice)
	}
	branchChoice, ok := byID[fixture.branch]
	if !ok || branchChoice.IsDefault {
		t.Fatalf("分公司候选应存在且不标记默认: %#v", branchChoice)
	}
	if _, filtered := byID[fixture.disabledOrg]; filtered {
		t.Fatalf("停用组织不应进入候选列表: %#v", choices)
	}
}

func TestAuthRepoRotateSessionBootstrapAdminEntersNonMembershipOrganization(t *testing.T) {
	fixture := newAuthPrincipalFixture(t)
	ctx := context.Background()
	currentHash := "bootstrap-rotate-current-" + uuid.NewString()
	nextHash := "bootstrap-rotate-next-" + uuid.NewString()
	if err := fixture.repo.CreateSession(ctx, &biz.Session{
		TokenHash:      currentHash,
		UserID:         fixture.bootstrapUserID,
		OrganizationID: fixture.headquarters,
		ExpiresAt:      fixture.sessionExpiry,
		UserAgent:      "integration-test",
	}, "", &biz.AuditEvent{OrganizationID: &fixture.headquarters, UserID: &fixture.bootstrapUserID, Action: "auth.login", Result: "success"}); err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}

	// bootstrap 管理员可轮转进入无成员关系的启用中组织。
	err := fixture.repo.RotateSession(ctx, currentHash, &biz.Session{
		TokenHash:      nextHash,
		UserID:         fixture.bootstrapUserID,
		OrganizationID: fixture.branch,
		ExpiresAt:      fixture.sessionExpiry,
	}, time.Now().UTC(), &biz.AuditEvent{OrganizationID: &fixture.branch, UserID: &fixture.bootstrapUserID, Action: "auth.organization.switch", Result: "success"})
	if err != nil {
		t.Fatalf("bootstrap 管理员轮转进入非成员组织失败: %v", err)
	}
	switched, err := fixture.repo.FindSession(ctx, nextHash, time.Now().UTC())
	if err != nil {
		t.Fatalf("新令牌应立即可用: %v", err)
	}
	if switched.OrganizationID != fixture.branch {
		t.Fatalf("新会话组织 = %v，期望分公司", switched.OrganizationID)
	}
	if _, err := fixture.repo.FindSession(ctx, currentHash, time.Now().UTC()); err != biz.ErrSessionExpired {
		t.Fatalf("旧令牌应失效，错误 = %v", err)
	}

	// 停用组织依旧不可进入，且被拒绝的轮转不影响当前会话。
	rejectedHash := "bootstrap-rotate-rejected-" + uuid.NewString()
	if err := fixture.repo.RotateSession(ctx, nextHash, &biz.Session{
		TokenHash:      rejectedHash,
		UserID:         fixture.bootstrapUserID,
		OrganizationID: fixture.disabledOrg,
		ExpiresAt:      fixture.sessionExpiry,
	}, time.Now().UTC(), &biz.AuditEvent{OrganizationID: &fixture.disabledOrg, UserID: &fixture.bootstrapUserID, Action: "auth.organization.switch", Result: "success"}); err != biz.ErrAuthOrganizationForbidden {
		t.Fatalf("轮转进入停用组织错误 = %v，期望 ErrAuthOrganizationForbidden", err)
	}
	if _, err := fixture.repo.FindSession(ctx, rejectedHash, time.Now().UTC()); err != biz.ErrSessionExpired {
		t.Fatalf("被拒绝的轮转不应留下新会话: %v", err)
	}
	if _, err := fixture.repo.FindSession(ctx, nextHash, time.Now().UTC()); err != nil {
		t.Fatalf("被拒绝的轮转不应影响当前会话: %v", err)
	}
}

// authCompanyGranularityFixture 构造公司粒度工作台口径的隔离数据：总部 → 公司 → 部门
// 三级树。双职用户在总部与部门各持一条启用成员关系并各挂一个角色，用于断言总部工作台
// 不聚合公司子树角色、公司工作台聚合子树角色；仅部门成员用户用于断言部门成员关系向上
// 取整出公司候选与 RotateSession 的子树复核。
type authCompanyGranularityFixture struct {
	t             *testing.T
	repo          *authRepo
	data          *Data
	headquarters  uuid.UUID
	company       uuid.UUID
	department    uuid.UUID
	hqRoleID      uuid.UUID
	deptRoleID    uuid.UUID
	dualUserID    uuid.UUID
	deptOnlyUser  uuid.UUID
	sessionExpiry time.Time
}

func newAuthCompanyGranularityFixture(t *testing.T) *authCompanyGranularityFixture {
	t.Helper()
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	suffix := uuid.NewString()[:12]
	headquarters, err := data.db.Organization.Create().
		SetCode("GH-" + suffix).
		SetName("集团总部-" + suffix).
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建总部组织失败: %v", err)
	}
	company, err := data.db.Organization.Create().
		SetCode("CO-" + suffix).
		SetName("融迅公司-" + suffix).
		SetKind("company").
		SetParentID(headquarters.ID).
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建公司组织失败: %v", err)
	}
	department, err := data.db.Organization.Create().
		SetCode("DP-" + suffix).
		SetName("北京财务-" + suffix).
		SetKind("department").
		SetParentID(company.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建部门组织失败: %v", err)
	}
	hqRole, err := data.db.Role.Create().
		SetOrganizationID(headquarters.ID).
		SetCode("hq_role_" + suffix).
		SetName("总部角色-" + suffix).
		SetDataScope(roleent.DataScopeOrganization).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建总部角色失败: %v", err)
	}
	deptRole, err := data.db.Role.Create().
		SetOrganizationID(company.ID).
		SetCode("dept_role_" + suffix).
		SetName("部门角色-" + suffix).
		SetDataScope(roleent.DataScopeOrganization).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建部门角色失败: %v", err)
	}
	dualUser, err := data.db.User.Create().
		SetUsername("dual-" + suffix).
		SetDisplayName("双职用户-" + suffix).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建双职用户失败: %v", err)
	}
	hqMembership, err := data.db.Membership.Create().
		SetUserID(dualUser.ID).
		SetOrganizationID(headquarters.ID).
		SetPrimary(true).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建总部成员关系失败: %v", err)
	}
	deptMembership, err := data.db.Membership.Create().
		SetUserID(dualUser.ID).
		SetOrganizationID(department.ID).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建部门成员关系失败: %v", err)
	}
	if _, err := data.db.RoleAssignment.Create().SetMembershipID(hqMembership.ID).SetRoleID(hqRole.ID).Save(ctx); err != nil {
		t.Fatalf("挂载总部角色失败: %v", err)
	}
	if _, err := data.db.RoleAssignment.Create().SetMembershipID(deptMembership.ID).SetRoleID(deptRole.ID).Save(ctx); err != nil {
		t.Fatalf("挂载部门角色失败: %v", err)
	}
	deptOnlyUser, err := data.db.User.Create().
		SetUsername("deponly-" + suffix).
		SetDisplayName("部门用户-" + suffix).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建部门用户失败: %v", err)
	}
	if _, err := data.db.Membership.Create().
		SetUserID(deptOnlyUser.ID).
		SetOrganizationID(department.ID).
		SetPrimary(true).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建仅部门成员关系失败: %v", err)
	}
	return &authCompanyGranularityFixture{
		t:             t,
		repo:          NewAuthRepo(data).(*authRepo),
		data:          data,
		headquarters:  headquarters.ID,
		company:       company.ID,
		department:    department.ID,
		hqRoleID:      hqRole.ID,
		deptRoleID:    deptRole.ID,
		dualUserID:    dualUser.ID,
		deptOnlyUser:  deptOnlyUser.ID,
		sessionExpiry: time.Now().Add(time.Hour),
	}
}

func grantRoleIDs(principal *biz.Principal) map[uuid.UUID]struct{} {
	result := make(map[uuid.UUID]struct{}, len(principal.RoleGrants))
	for _, grant := range principal.RoleGrants {
		result[grant.RoleID] = struct{}{}
	}
	return result
}

// TestAuthRepoResolvePrincipalWorkspaceRoleScopeFollowsWorkspaceKind 锁定两条容易回退的
// 口径：总部工作台只收集总部节点本身的成员关系角色（不跨公司聚合）；公司工作台聚合
// 公司子树内全部启用成员关系角色（部门角色在公司工作区生效），但不回收总部角色。
func TestAuthRepoResolvePrincipalWorkspaceRoleScopeFollowsWorkspaceKind(t *testing.T) {
	fixture := newAuthCompanyGranularityFixture(t)
	ctx := context.Background()

	headquartersPrincipal, err := fixture.repo.ResolvePrincipal(ctx, fixture.dualUserID, fixture.headquarters)
	if err != nil {
		t.Fatalf("解析总部工作台失败: %v", err)
	}
	hqGrants := grantRoleIDs(headquartersPrincipal)
	if _, ok := hqGrants[fixture.hqRoleID]; !ok {
		t.Fatalf("总部工作台应包含总部成员关系角色: %#v", headquartersPrincipal.RoleGrants)
	}
	if _, ok := hqGrants[fixture.deptRoleID]; ok {
		t.Fatalf("总部工作台不应聚合公司子树角色: %#v", headquartersPrincipal.RoleGrants)
	}
	organizationIDs := make(map[uuid.UUID]struct{}, len(headquartersPrincipal.Organizations))
	for _, organization := range headquartersPrincipal.Organizations {
		organizationIDs[organization.ID] = struct{}{}
	}
	if _, ok := organizationIDs[fixture.company]; !ok {
		t.Fatalf("部门成员关系应让所属公司进入候选: %#v", headquartersPrincipal.Organizations)
	}
	if _, ok := organizationIDs[fixture.department]; ok {
		t.Fatalf("部门不是工作台，不应进入候选: %#v", headquartersPrincipal.Organizations)
	}

	companyPrincipal, err := fixture.repo.ResolvePrincipal(ctx, fixture.dualUserID, fixture.company)
	if err != nil {
		t.Fatalf("解析公司工作台失败: %v", err)
	}
	companyGrants := grantRoleIDs(companyPrincipal)
	if _, ok := companyGrants[fixture.deptRoleID]; !ok {
		t.Fatalf("公司工作台应聚合子树内部门成员关系角色: %#v", companyPrincipal.RoleGrants)
	}
	if _, ok := companyGrants[fixture.hqRoleID]; ok {
		t.Fatalf("公司工作台不应包含总部成员关系角色: %#v", companyPrincipal.RoleGrants)
	}
}

// TestAuthRepoListEnabledMembershipOrganizationsRoundsDepartmentUpToCompany 断言普通用户
// 候选口径：部门成员关系向上取整让所属公司成为候选，总部不因公司成员资格进入候选，
// 部门/团队本身任何情况下不是候选。
func TestAuthRepoListEnabledMembershipOrganizationsRoundsDepartmentUpToCompany(t *testing.T) {
	fixture := newAuthCompanyGranularityFixture(t)
	ctx := context.Background()

	deptOnlyChoices, err := fixture.repo.ListEnabledMembershipOrganizations(ctx, fixture.deptOnlyUser)
	if err != nil {
		t.Fatalf("查询仅部门成员用户候选失败: %v", err)
	}
	if len(deptOnlyChoices) != 1 {
		t.Fatalf("仅部门成员用户候选数 = %d，期望 1: %#v", len(deptOnlyChoices), deptOnlyChoices)
	}
	if deptOnlyChoices[0].OrganizationID != fixture.company || !deptOnlyChoices[0].IsDefault {
		t.Fatalf("部门成员关系应取整为默认公司候选: %#v", deptOnlyChoices)
	}

	dualChoices, err := fixture.repo.ListEnabledMembershipOrganizations(ctx, fixture.dualUserID)
	if err != nil {
		t.Fatalf("查询双职用户候选失败: %v", err)
	}
	byID := make(map[uuid.UUID]biz.OrganizationChoice, len(dualChoices))
	for _, choice := range dualChoices {
		byID[choice.OrganizationID] = choice
	}
	if len(dualChoices) != 2 {
		t.Fatalf("双职用户候选数 = %d，期望 2: %#v", len(dualChoices), dualChoices)
	}
	if choice, ok := byID[fixture.headquarters]; !ok || !choice.IsDefault {
		t.Fatalf("总部成员关系（primary）应映射为默认总部候选: %#v", choice)
	}
	if choice, ok := byID[fixture.company]; !ok || choice.IsDefault {
		t.Fatalf("部门成员关系应映射出公司候选且不标记默认: %#v", choice)
	}
	if _, ok := byID[fixture.department]; ok {
		t.Fatalf("部门不是工作台，不应进入候选: %#v", dualChoices)
	}
}

// TestAuthRepoRotateSessionNormalUserRechecksSubtreeMembership 锁定 RotateSession 的
// TOCTOU 子树复核：普通用户目标为公司时，公司子树内任一启用成员关系即可通过（无需
// 公司节点本身成员关系）；总部范围外与部门目标一律拒绝。
func TestAuthRepoRotateSessionNormalUserRechecksSubtreeMembership(t *testing.T) {
	fixture := newAuthCompanyGranularityFixture(t)
	ctx := context.Background()
	currentHash := "dept-rotate-current-" + uuid.NewString()
	nextHash := "dept-rotate-next-" + uuid.NewString()
	if err := fixture.repo.CreateSession(ctx, &biz.Session{
		TokenHash:      currentHash,
		UserID:         fixture.deptOnlyUser,
		OrganizationID: fixture.company,
		ExpiresAt:      fixture.sessionExpiry,
		UserAgent:      "integration-test",
	}, "", &biz.AuditEvent{OrganizationID: &fixture.company, UserID: &fixture.deptOnlyUser, Action: "auth.login", Result: "success"}); err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}

	// 目标公司在用户子树（部门）内有启用成员关系：即使公司节点本身无成员关系也应通过。
	if err := fixture.repo.RotateSession(ctx, currentHash, &biz.Session{
		TokenHash:      nextHash,
		UserID:         fixture.deptOnlyUser,
		OrganizationID: fixture.company,
		ExpiresAt:      fixture.sessionExpiry,
	}, time.Now().UTC(), &biz.AuditEvent{OrganizationID: &fixture.company, UserID: &fixture.deptOnlyUser, Action: "auth.organization.switch", Result: "success"}); err != nil {
		t.Fatalf("公司子树成员关系复核应通过: %v", err)
	}

	rejected := []struct {
		name           string
		organizationID uuid.UUID
	}{
		{name: "总部（子树外）", organizationID: fixture.headquarters},
		{name: "部门（非工作台）", organizationID: fixture.department},
	}
	for _, test := range rejected {
		deniedHash := "dept-rotate-denied-" + uuid.NewString()
		if err := fixture.repo.RotateSession(ctx, nextHash, &biz.Session{
			TokenHash:      deniedHash,
			UserID:         fixture.deptOnlyUser,
			OrganizationID: test.organizationID,
			ExpiresAt:      fixture.sessionExpiry,
		}, time.Now().UTC(), &biz.AuditEvent{OrganizationID: &test.organizationID, UserID: &fixture.deptOnlyUser, Action: "auth.organization.switch", Result: "failure"}); err != biz.ErrAuthOrganizationForbidden {
			t.Fatalf("轮转进入%s错误 = %v，期望 ErrAuthOrganizationForbidden", test.name, err)
		}
		if _, err := fixture.repo.FindSession(ctx, deniedHash, time.Now().UTC()); err != biz.ErrSessionExpired {
			t.Fatalf("被拒绝的轮转不应留下新会话: %v", err)
		}
	}
	if _, err := fixture.repo.FindSession(ctx, nextHash, time.Now().UTC()); err != nil {
		t.Fatalf("被拒绝的轮转不应影响当前会话: %v", err)
	}
}
