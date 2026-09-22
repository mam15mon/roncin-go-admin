package data

import (
	"context"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	"testing"
)

func TestSystemCompanyAdministratorAuthorizationBoundaries(t *testing.T) {
	f := newAuthPrincipalFixture(t)
	ctx := context.Background()
	if err := initializeCompanyAdministrator(ctx, f.data.db, f.branch); err != nil {
		t.Fatal(err)
	}
	companyRole, err := f.data.db.Role.Query().Where(role.OrganizationIDEQ(f.branch), role.CodeEQ("administrator")).WithPermissions().Only(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range companyRole.Edges.Permissions {
		if access.IsSystemManagementPermission(item.Key) {
			t.Fatalf("公司默认角色含系统权限: %s", item.Key)
		}
	}
	if count, err := f.data.db.Membership.Query().Where(membership.OrganizationIDEQ(f.branch)).Count(ctx); err != nil || count != 0 {
		t.Fatalf("初始化角色不得自动授予成员: count=%d err=%v", count, err)
	}
	repo := NewAdminRepo(f.data)
	audit := &biz.AuditEvent{OrganizationID: &f.branch, UserID: &f.bootstrapUserID, Action: "admin.role.create", Result: "success", Details: map[string]string{}}
	if _, err := repo.CreateRole(ctx, f.branch, &biz.AdminRole{Code: "bad", Name: "越权", DataScope: biz.DataScopeAll, Enabled: true}, []string{access.MasterDataPortUpdate}, audit); err != biz.ErrAdminPrivilegeEscalation {
		t.Fatalf("公司角色不得授系统权限: %v", err)
	}
	principal := &biz.Principal{UserID: f.bootstrapUserID, Organization: biz.Organization{ID: f.systemWorkspace, Kind: biz.OrganizationKindSystem}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: f.systemWorkspace, Kind: biz.OrganizationKindSystem}, {ID: f.branch, Kind: biz.OrganizationKindCompany}}, RoleGrants: []biz.RoleGrant{{RoleCode: "administrator", DataScope: biz.DataScopeAll, Permissions: map[string]struct{}{access.UserCreate: {}, access.UserUpdate: {}}}}}
	systemRole, err := f.data.db.Role.Create().SetOrganizationID(f.systemWorkspace).SetCode("administrator").SetName("系统管理员").SetDataScope(role.DataScopeAll).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	uc := biz.NewAdminUsecase(repo)
	if _, err := uc.CreateUserMembership(biz.WithPrincipal(ctx, principal), f.systemWorkspace, f.bootstrapUserID, f.normalUserID, f.systemWorkspace, false, []uuid.UUID{systemRole.ID}); err != biz.ErrAdminPrivilegeEscalation {
		t.Fatalf("有限系统用户管理者不得授予系统管理员: %v", err)
	}
	_, err = uc.CreateUserMembership(biz.WithPrincipal(ctx, principal), f.systemWorkspace, f.bootstrapUserID, f.normalUserID, f.branch, false, []uuid.UUID{companyRole.ID})
	if err != nil {
		t.Fatalf("系统管理应能显式分配公司管理员: %v", err)
	}
	companyPrincipal := &biz.Principal{UserID: f.normalUserID, Organization: biz.Organization{ID: f.branch, Kind: biz.OrganizationKindCompany}, OrganizationNodes: principal.OrganizationNodes, RoleGrants: []biz.RoleGrant{{RoleCode: "administrator", DataScope: biz.DataScopeAll, Permissions: map[string]struct{}{access.UserUpdate: {}}}}}
	companyCtx := biz.WithPrincipal(ctx, companyPrincipal)
	if _, err := uc.AuthorizeWeComUser(companyCtx, f.branch, f.branch, f.normalUserID, &biz.AdminUser{ID: f.bootstrapUserID, DisplayName: "不能全局授权"}, []uuid.UUID{companyRole.ID}); err != biz.ErrPermissionDenied {
		t.Fatalf("公司不得全局重启企业微信账号: %v", err)
	}
	if _, err := uc.AuthorizeDingTalkUser(companyCtx, f.branch, f.branch, f.normalUserID, &biz.AdminUser{ID: f.bootstrapUserID, DisplayName: "不能全局授权"}, []uuid.UUID{companyRole.ID}); err != biz.ErrPermissionDenied {
		t.Fatalf("公司不得全局重启钉钉账号: %v", err)
	}
	if err := uc.ResetUserPassword(companyCtx, f.branch, f.normalUserID, f.bootstrapUserID, "long-enough-password", nil); err != biz.ErrPermissionDenied {
		t.Fatalf("公司不得重置跨公司全局凭据: %v", err)
	}
	if err := uc.TerminateUser(companyCtx, f.branch, f.normalUserID, f.bootstrapUserID); err != biz.ErrPermissionDenied {
		t.Fatalf("公司不得停用全局账号: %v", err)
	}

	if _, err := uc.CreateUserMembership(companyCtx, f.branch, f.normalUserID, f.normalUserID, f.systemWorkspace, false, nil); err != biz.ErrAdminPrivilegeEscalation {
		t.Fatalf("空角色也不得加入系统工作台: %v", err)
	}
	memberships, err := repo.ListUserMemberships(companyCtx, f.normalUserID)
	if err != nil || len(memberships) != 1 || memberships[0].OrganizationID != f.branch {
		t.Fatalf("公司不得列出其他工作台成员关系: %#v %v", memberships, err)
	}
	systemMemberships, err := repo.ListUserMemberships(biz.WithPrincipal(ctx, principal), f.normalUserID)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range systemMemberships {
		if item.OrganizationID == f.systemWorkspace {
			if _, err := repo.GetUserMembership(companyCtx, f.normalUserID, item.ID); err != biz.ErrAdminUserMembershipNotFound {
				t.Fatalf("公司不得点查系统成员关系: %v", err)
			}
			if err := repo.DeleteUserMembership(companyCtx, f.normalUserID, item.ID, audit); err != biz.ErrAdminUserMembershipNotFound {
				t.Fatalf("公司不得删除系统成员关系: %v", err)
			}
		}
	}
	resolved, err := f.repo.ResolvePrincipal(ctx, f.normalUserID, f.branch)
	if err != nil || len(resolved.RoleGrants) != 1 {
		t.Fatalf("显式授权后应能进入公司: %v %#v", err, resolved)
	}
}

func TestSystemRegistrationCannotGrantSystemIdentity(t *testing.T) {
	f := newDingTalkRegistrationFixture(t)
	identity := &biz.DingTalkIdentity{UnionID: "pending-system-" + f.suffix, UserID: "pending-system-" + f.suffix, Name: "待审批"}
	credential, _, err := f.authRepo.RegisterDingTalkCredential(f.ctx, identity, nil, nil, &biz.AuditEvent{Action: "auth.dingtalk.register", Result: "success"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.repo.ApproveRegistration(f.ctx, &biz.DingTalkRegistrationDecision{UserID: credential.UserID, ActorID: f.inviter.ID, OrganizationIDs: []uuid.UUID{f.systemWorkspace.ID}, RoleIDs: []uuid.UUID{f.staffRole.ID}, Audit: &biz.AuditEvent{Action: "admin.dingtalk.registration.approve", Result: "success"}})
	if err != biz.ErrDingTalkRegistrationCompanyRequired {
		t.Fatalf("系统收口不得直接授角色: %v", err)
	}
	account, err := f.data.db.User.Get(f.ctx, credential.UserID)
	if err != nil || account.Enabled {
		t.Fatalf("失败审批应保持待审批账号停用: %v", err)
	}
}
