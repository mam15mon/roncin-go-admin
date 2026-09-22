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
// bootstrap 用户仅持系统管理成员关系（无角色），另有其无成员关系的分公司与一个停用组织；
// 普通用户同样仅持系统管理成员关系，用于普通用户路径回归。
type authPrincipalFixture struct {
	t               *testing.T
	repo            *authRepo
	data            *Data
	bootstrapUserID uuid.UUID
	normalUserID    uuid.UUID
	systemWorkspace uuid.UUID
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
	systemWorkspace, err := data.db.Organization.Create().
		SetCode("HQ-" + suffix).
		SetName("系统管理-" + suffix).
		SetKind("system").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建系统管理组织失败: %v", err)
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
		SetOrganizationID(systemWorkspace.ID).
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
		SetOrganizationID(systemWorkspace.ID).
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
		systemWorkspace: systemWorkspace.ID,
		branch:          branch.ID,
		disabledOrg:     disabledOrg.ID,
		permissionKeys:  permissionKeys,
		sessionExpiry:   time.Now().Add(time.Hour),
	}
}

func TestAuthRepoResolvePrincipalBootstrapAdminRequiresMembershipAndDoesNotSynthesizePermissions(t *testing.T) {
	f := newAuthPrincipalFixture(t)
	if _, err := f.repo.ResolvePrincipal(context.Background(), f.bootstrapUserID, f.branch); err != biz.ErrOrganizationForbidden {
		t.Fatalf("无成员公司应拒绝: %v", err)
	}
	principal, err := f.repo.ResolvePrincipal(context.Background(), f.bootstrapUserID, f.systemWorkspace)
	if err != nil {
		t.Fatal(err)
	}
	if len(principal.RoleGrants) != 0 || len(principal.Organizations) != 1 {
		t.Fatalf("不应合成角色或公司候选: %#v", principal)
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
	principal, err := fixture.repo.ResolvePrincipal(ctx, fixture.normalUserID, fixture.systemWorkspace)
	if err != nil {
		t.Fatalf("普通用户解析成员组织失败: %v", err)
	}
	if principal.Organization.ID != fixture.systemWorkspace {
		t.Fatalf("普通用户当前组织 = %v，期望系统管理", principal.Organization.ID)
	}
	for _, organization := range principal.Organizations {
		if organization.ID == fixture.branch {
			t.Fatalf("普通用户候选组织不应包含无成员关系组织: %#v", principal.Organizations)
		}
	}
}

func TestAuthRepoBootstrapCandidatesRequireMembership(t *testing.T) {
	f := newAuthPrincipalFixture(t)
	choices, err := f.repo.ListEnabledMembershipOrganizations(context.Background(), f.bootstrapUserID)
	if err != nil {
		t.Fatal(err)
	}
	if len(choices) != 1 || choices[0].OrganizationID != f.systemWorkspace || !choices[0].IsDefault {
		t.Fatalf("候选仅来自成员资格: %#v", choices)
	}
}

func TestAuthRepoRotateSessionBootstrapAdminRejectsNonMembershipOrganization(t *testing.T) {
	f := newAuthPrincipalFixture(t)
	ctx := context.Background()
	currentHash, nextHash := "current-"+uuid.NewString(), "next-"+uuid.NewString()
	audit := &biz.AuditEvent{OrganizationID: &f.systemWorkspace, UserID: &f.bootstrapUserID, Action: "auth.login", Result: "success"}
	if err := f.repo.CreateSession(ctx, &biz.Session{TokenHash: currentHash, UserID: f.bootstrapUserID, OrganizationID: f.systemWorkspace, ExpiresAt: f.sessionExpiry}, "", audit); err != nil {
		t.Fatal(err)
	}
	err := f.repo.RotateSession(ctx, currentHash, &biz.Session{TokenHash: nextHash, UserID: f.bootstrapUserID, OrganizationID: f.branch, ExpiresAt: f.sessionExpiry}, time.Now().UTC(), audit)
	if err != biz.ErrAuthOrganizationForbidden {
		t.Fatalf("无成员公司应拒绝: %v", err)
	}
	if _, err := f.repo.FindSession(ctx, currentHash, time.Now().UTC()); err != nil {
		t.Fatalf("原会话须保留: %v", err)
	}
	if _, err := f.repo.FindSession(ctx, nextHash, time.Now().UTC()); err != biz.ErrSessionExpired {
		t.Fatalf("不应创建新会话: %v", err)
	}
}

// authCompanyGranularityFixture 构造公司粒度工作台口径的隔离数据：系统管理 → 公司 → 部门
// 三级树。双职用户在系统管理与部门各持一条启用成员关系并各挂一个角色，用于断言系统管理工作台
// 不聚合公司子树角色、公司工作台聚合子树角色；仅部门成员用户用于断言部门成员关系向上
// 取整出公司候选与 RotateSession 的子树复核。
type authCompanyGranularityFixture struct {
	t               *testing.T
	repo            *authRepo
	data            *Data
	systemWorkspace uuid.UUID
	company         uuid.UUID
	department      uuid.UUID
	hqRoleID        uuid.UUID
	deptRoleID      uuid.UUID
	dualUserID      uuid.UUID
	deptOnlyUser    uuid.UUID
	sessionExpiry   time.Time
}

func newAuthCompanyGranularityFixture(t *testing.T) *authCompanyGranularityFixture {
	t.Helper()
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	suffix := uuid.NewString()[:12]
	systemWorkspace, err := data.db.Organization.Create().
		SetCode("GH-" + suffix).
		SetName("集团系统管理-" + suffix).
		SetKind("system").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建系统管理组织失败: %v", err)
	}
	company, err := data.db.Organization.Create().
		SetCode("CO-" + suffix).
		SetName("融迅公司-" + suffix).
		SetKind("company").
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
		SetOrganizationID(systemWorkspace.ID).
		SetCode("hq_role_" + suffix).
		SetName("系统管理角色-" + suffix).
		SetDataScope(roleent.DataScopeOrganization).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建系统管理角色失败: %v", err)
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
		SetOrganizationID(systemWorkspace.ID).
		SetPrimary(true).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建系统管理成员关系失败: %v", err)
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
		t.Fatalf("挂载系统管理角色失败: %v", err)
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
		t:               t,
		repo:            NewAuthRepo(data).(*authRepo),
		data:            data,
		systemWorkspace: systemWorkspace.ID,
		company:         company.ID,
		department:      department.ID,
		hqRoleID:        hqRole.ID,
		deptRoleID:      deptRole.ID,
		dualUserID:      dualUser.ID,
		deptOnlyUser:    deptOnlyUser.ID,
		sessionExpiry:   time.Now().Add(time.Hour),
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
// 口径：系统管理工作台只收集系统管理节点本身的成员关系角色（不跨公司聚合）；公司工作台聚合
// 公司子树内全部启用成员关系角色（部门角色在公司工作区生效），但不回收系统管理角色。
func TestAuthRepoResolvePrincipalWorkspaceRoleScopeFollowsWorkspaceKind(t *testing.T) {
	fixture := newAuthCompanyGranularityFixture(t)
	ctx := context.Background()

	systemWorkspacePrincipal, err := fixture.repo.ResolvePrincipal(ctx, fixture.dualUserID, fixture.systemWorkspace)
	if err != nil {
		t.Fatalf("解析系统管理工作台失败: %v", err)
	}
	hqGrants := grantRoleIDs(systemWorkspacePrincipal)
	if _, ok := hqGrants[fixture.hqRoleID]; !ok {
		t.Fatalf("系统管理工作台应包含系统管理成员关系角色: %#v", systemWorkspacePrincipal.RoleGrants)
	}
	if _, ok := hqGrants[fixture.deptRoleID]; ok {
		t.Fatalf("系统管理工作台不应聚合公司子树角色: %#v", systemWorkspacePrincipal.RoleGrants)
	}
	organizationIDs := make(map[uuid.UUID]struct{}, len(systemWorkspacePrincipal.Organizations))
	for _, organization := range systemWorkspacePrincipal.Organizations {
		organizationIDs[organization.ID] = struct{}{}
	}
	if _, ok := organizationIDs[fixture.company]; !ok {
		t.Fatalf("部门成员关系应让所属公司进入候选: %#v", systemWorkspacePrincipal.Organizations)
	}
	if _, ok := organizationIDs[fixture.department]; ok {
		t.Fatalf("部门不是工作台，不应进入候选: %#v", systemWorkspacePrincipal.Organizations)
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
		t.Fatalf("公司工作台不应包含系统管理成员关系角色: %#v", companyPrincipal.RoleGrants)
	}
}

// TestAuthRepoListEnabledMembershipOrganizationsRoundsDepartmentUpToCompany 断言普通用户
// 候选口径：部门成员关系向上取整让所属公司成为候选，系统管理不因公司成员资格进入候选，
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
	if choice, ok := byID[fixture.systemWorkspace]; !ok || !choice.IsDefault {
		t.Fatalf("系统管理成员关系（primary）应映射为默认系统管理候选: %#v", choice)
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
// 公司节点本身成员关系）；系统管理范围外与部门目标一律拒绝。
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
		{name: "系统管理（子树外）", organizationID: fixture.systemWorkspace},
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
