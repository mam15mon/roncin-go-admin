package data

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	"testing"
	"time"
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
	// 批量入口不再信任传入的权限快照，有限管理员必须在真实授权夹具中建立。
	userUpdate, err := f.data.db.Permission.Create().SetKey(access.UserUpdate).SetName("成员管理").SetGroup("系统").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	managerRole, err := f.data.db.Role.Create().SetOrganizationID(f.systemWorkspace).SetCode("user-manager").SetName("成员管理员").SetDataScope(role.DataScopeOrganization).AddPermissionIDs(userUpdate.ID).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	managerMembership, err := f.data.db.Membership.Query().Where(membership.UserIDEQ(f.bootstrapUserID), membership.OrganizationIDEQ(f.systemWorkspace)).Only(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.data.db.RoleAssignment.Create().SetMembershipID(managerMembership.ID).SetRoleID(managerRole.ID).Save(ctx); err != nil {
		t.Fatal(err)
	}
	uc := biz.NewAdminUsecase(repo, f.data)
	if _, err := uc.CreateUserMemberships(biz.WithPrincipal(ctx, principal), f.systemWorkspace, f.bootstrapUserID, f.normalUserID, []biz.AdminUserMembershipCreation{{OrganizationID: f.systemWorkspace, RoleIDs: []uuid.UUID{systemRole.ID}}}); !errors.Is(err, biz.ErrAdminPrivilegeEscalation) {
		t.Fatalf("有限系统用户管理者不得授予系统管理员: %v", err)
	}
	_, err = uc.CreateUserMemberships(biz.WithPrincipal(ctx, principal), f.systemWorkspace, f.bootstrapUserID, f.normalUserID, []biz.AdminUserMembershipCreation{{OrganizationID: f.branch, RoleIDs: []uuid.UUID{companyRole.ID}}})
	if err != nil {
		t.Fatalf("系统管理应能显式分配公司管理员: %v", err)
	}
	companyPrincipal := &biz.Principal{UserID: f.normalUserID, Organization: biz.Organization{ID: f.branch, Kind: biz.OrganizationKindCompany}, OrganizationNodes: principal.OrganizationNodes, RoleGrants: []biz.RoleGrant{{RoleCode: "administrator", DataScope: biz.DataScopeAll, Permissions: map[string]struct{}{access.UserUpdate: {}}}}}
	companyCtx := biz.WithPrincipal(ctx, companyPrincipal)
	if _, err := uc.AuthorizeDingTalkUser(companyCtx, f.branch, f.branch, f.normalUserID, &biz.AdminUser{ID: f.bootstrapUserID, DisplayName: "不能全局授权"}, []uuid.UUID{companyRole.ID}); err != biz.ErrPermissionDenied {
		t.Fatalf("公司不得全局重启钉钉账号: %v", err)
	}
	if err := uc.ResetUserPassword(companyCtx, f.branch, f.normalUserID, f.bootstrapUserID, "long-enough-password", nil); err != biz.ErrPermissionDenied {
		t.Fatalf("公司不得重置跨公司全局凭据: %v", err)
	}
	if err := uc.TerminateUser(companyCtx, f.branch, f.normalUserID, f.bootstrapUserID); err != biz.ErrPermissionDenied {
		t.Fatalf("公司不得停用全局账号: %v", err)
	}

	if _, err := uc.CreateUserMemberships(companyCtx, f.branch, f.normalUserID, f.normalUserID, []biz.AdminUserMembershipCreation{{OrganizationID: f.systemWorkspace}}); !errors.Is(err, biz.ErrAdminPrivilegeEscalation) {
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

func TestSystemDirectoryAssignmentCannotGrantSystemIdentity(t *testing.T) {
	f := newAuthPrincipalFixture(t)
	ctx := context.Background()
	account, err := f.data.db.User.Create().SetDisplayName("待分配").SetDingtalkUnionid("pending-union").SetDingtalkUserid("pending-user").SetPendingAssignment(true).SetEnabled(false).SetDingtalkLastSyncedAt(time.Now()).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ctx, actor := directoryAdministrator(t, f.data, f.systemWorkspace)
	_, err = biz.NewAdminUsecase(&adminRepo{data: f.data}, f.data).AuthorizeDingTalkUser(ctx, f.systemWorkspace, f.systemWorkspace, actor, &biz.AdminUser{ID: account.ID}, []uuid.UUID{uuid.New()})
	if err != biz.ErrAdminOrganizationNotFound {
		t.Fatalf("通讯录员工不得被分配至系统工作台: %v", err)
	}
	reloaded, err := f.data.db.User.Get(ctx, account.ID)
	if err != nil || reloaded.Enabled || !reloaded.PendingAssignment {
		t.Fatalf("失败分配必须保持待分配: %v", err)
	}
}
