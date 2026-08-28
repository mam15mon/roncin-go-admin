package biz

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type adminRepoStub struct {
	organization      *AdminOrganization
	organizationID    uuid.UUID
	organizationInput *AdminOrganization
	userListOptions   AdminUserListOptions
	roleKeys          []string
	userPassword      string
	userInput         *AdminUser
	resetPassword     string
	deletedUserID     uuid.UUID
	deleteOrgID       uuid.UUID
	memberships       []*AdminUserMembership
	membershipInput   *AdminUserMembership
	membershipRoleIDs []uuid.UUID
	deletedMembership uuid.UUID
}

func (s *adminRepoStub) ListOrganizations(context.Context) ([]*AdminOrganization, error) {
	return nil, nil
}

func (s *adminRepoStub) GetOrganization(_ context.Context, id uuid.UUID) (*AdminOrganization, error) {
	s.organizationID = id
	if s.organization == nil || s.organization.ID != id {
		return nil, ErrAdminOrganizationNotFound
	}
	return s.organization, nil
}

func (s *adminRepoStub) CreateOrganization(_ context.Context, input *AdminOrganization) (*AdminOrganization, error) {
	s.organizationInput = input
	return input, nil
}

func (s *adminRepoStub) UpdateOrganization(_ context.Context, _ uuid.UUID, input *AdminOrganization) (*AdminOrganization, error) {
	return input, nil
}

func (s *adminRepoStub) ListUsers(_ context.Context, _ uuid.UUID, options AdminUserListOptions) (*AdminUserList, error) {
	s.userListOptions = options
	return &AdminUserList{Page: options.Page, PageSize: options.PageSize}, nil
}

func (s *adminRepoStub) CreateUser(_ context.Context, _ uuid.UUID, input *AdminUser, passwordHash string, _ []uuid.UUID) (*AdminUser, error) {
	s.userInput = input
	s.userPassword = passwordHash
	return input, nil
}

func (s *adminRepoStub) UpdateUser(_ context.Context, _ uuid.UUID, _ uuid.UUID, input *AdminUser, _ []uuid.UUID) (*AdminUser, error) {
	return input, nil
}

func (s *adminRepoStub) ListUserMemberships(_ context.Context, _ uuid.UUID) ([]*AdminUserMembership, error) {
	return s.memberships, nil
}

func (s *adminRepoStub) CreateUserMembership(_ context.Context, input *AdminUserMembership, roleIDs []uuid.UUID) (*AdminUserMembership, error) {
	s.membershipInput = input
	s.membershipRoleIDs = roleIDs
	input.ID = uuid.New()
	return input, nil
}

func (s *adminRepoStub) UpdateUserMembership(_ context.Context, input *AdminUserMembership, roleIDs []uuid.UUID) (*AdminUserMembership, error) {
	s.membershipInput = input
	s.membershipRoleIDs = roleIDs
	input.OrganizationID = uuid.New()
	return input, nil
}

func (s *adminRepoStub) DeleteUserMembership(_ context.Context, _ uuid.UUID, membershipID uuid.UUID) error {
	s.deletedMembership = membershipID
	return nil
}

func (s *adminRepoStub) TerminateUser(_ context.Context, organizationID, id uuid.UUID) error {
	s.deleteOrgID = organizationID
	s.deletedUserID = id
	return nil
}

func (s *adminRepoStub) AuthorizeWeComUser(_ context.Context, _, targetOrganizationID uuid.UUID, input *AdminUser, _ []uuid.UUID) (*AdminUser, error) {
	input.Enabled = true
	s.organizationID = targetOrganizationID
	return input, nil
}

func (s *adminRepoStub) AuthorizeDingTalkUser(_ context.Context, _, targetOrganizationID uuid.UUID, input *AdminUser, _ []uuid.UUID) (*AdminUser, error) {
	input.Enabled = true
	s.organizationID = targetOrganizationID
	return input, nil
}

func (s *adminRepoStub) ResetUserPassword(_ context.Context, _ uuid.UUID, _ uuid.UUID, passwordHash string) error {
	s.resetPassword = passwordHash
	return nil
}

func (s *adminRepoStub) ListRoles(context.Context, uuid.UUID) ([]*AdminRole, error) {
	return nil, nil
}

func (s *adminRepoStub) CreateRole(_ context.Context, organizationID uuid.UUID, input *AdminRole, keys []string) (*AdminRole, error) {
	s.roleKeys = keys
	input.OrganizationID = organizationID
	return input, nil
}

func (s *adminRepoStub) UpdateRole(_ context.Context, _ uuid.UUID, _ uuid.UUID, input *AdminRole, _ []string) (*AdminRole, error) {
	return input, nil
}

func (s *adminRepoStub) ListPermissions(context.Context) ([]*AdminPermission, error) {
	return nil, nil
}

func (s *adminRepoStub) ListAuditLogs(_ context.Context, _ uuid.UUID, options AdminAuditLogListOptions) (*AdminAuditLogList, error) {
	return &AdminAuditLogList{Page: options.Page, PageSize: options.PageSize}, nil
}

type auditRepoStub struct {
	events []*AuditEvent
}

func (s *auditRepoStub) WriteAudit(_ context.Context, event *AuditEvent) error {
	s.events = append(s.events, event)
	return nil
}

func TestAdminUsecaseListUsersNormalizesOptions(t *testing.T) {
	repo := &adminRepoStub{}
	usecase := NewAdminUsecase(repo, &auditRepoStub{})
	organizationID := uuid.New()

	_, err := usecase.ListUsers(context.Background(), organizationID, AdminUserListOptions{Page: 2, PageSize: 20, Keyword: "  Alice  "})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if repo.userListOptions.Keyword != "Alice" {
		t.Fatalf("keyword = %q, want Alice", repo.userListOptions.Keyword)
	}

	if _, err := usecase.ListUsers(context.Background(), organizationID, AdminUserListOptions{Page: 0, PageSize: 20}); err != ErrAdminInvalidArgument {
		t.Fatalf("invalid page error = %v, want ErrAdminInvalidArgument", err)
	}
	if _, err := usecase.ListUsers(context.Background(), organizationID, AdminUserListOptions{Page: 1, PageSize: MaxListPageSize}); err != nil {
		t.Fatalf("maximum page size error = %v, want nil", err)
	}
	if _, err := usecase.ListUsers(context.Background(), organizationID, AdminUserListOptions{Page: 1, PageSize: MaxListPageSize + 1}); err != ErrAdminInvalidArgument {
		t.Fatalf("invalid page size error = %v, want ErrAdminInvalidArgument", err)
	}
}

func TestAdminUsecaseAuthorizeWeComUserRequiresTargetAndRole(t *testing.T) {
	usecase := NewAdminUsecase(&adminRepoStub{}, &auditRepoStub{})
	input := &AdminUser{ID: uuid.New(), DisplayName: "张三"}
	if _, err := usecase.AuthorizeWeComUser(context.Background(), uuid.New(), uuid.Nil, uuid.New(), input, []uuid.UUID{uuid.New()}); err != ErrAdminInvalidArgument {
		t.Fatalf("missing target organization error = %v", err)
	}
	if _, err := usecase.AuthorizeWeComUser(context.Background(), uuid.New(), uuid.New(), uuid.New(), input, nil); err != ErrAdminInvalidArgument {
		t.Fatalf("missing roles error = %v", err)
	}
}

func TestAdminUsecaseAuthorizeDingTalkUserRequiresTargetAndRole(t *testing.T) {
	usecase := NewAdminUsecase(&adminRepoStub{}, &auditRepoStub{})
	input := &AdminUser{ID: uuid.New(), DisplayName: "张三"}
	if _, err := usecase.AuthorizeDingTalkUser(context.Background(), uuid.New(), uuid.Nil, uuid.New(), input, []uuid.UUID{uuid.New()}); err != ErrAdminInvalidArgument {
		t.Fatalf("missing target organization error = %v", err)
	}
	if _, err := usecase.AuthorizeDingTalkUser(context.Background(), uuid.New(), uuid.New(), uuid.New(), input, nil); err != ErrAdminInvalidArgument {
		t.Fatalf("missing roles error = %v", err)
	}
}

func TestAdminUsecaseListOrganizationsRequiresOrganizationScope(t *testing.T) {
	usecase := NewAdminUsecase(&adminRepoStub{}, &auditRepoStub{})
	if _, err := usecase.ListOrganizations(context.Background(), uuid.Nil); err != ErrAdminInvalidArgument {
		t.Fatalf("ListOrganizations() error = %v, want ErrAdminInvalidArgument", err)
	}
}

func TestAdminUsecaseCreateOrganizationValidatesParent(t *testing.T) {
	parentID := uuid.New()
	repo := &adminRepoStub{organization: &AdminOrganization{ID: parentID, Kind: OrganizationKindHeadquarters, BaseCurrency: "CNY"}}
	usecase := NewAdminUsecase(repo, &auditRepoStub{})

	created, err := usecase.CreateOrganization(context.Background(), uuid.New(), &AdminOrganization{Code: " branch ", Name: " 分公司 ", Kind: OrganizationKindCompany, ParentID: &parentID, BaseCurrency: "usd"})
	if err != nil {
		t.Fatalf("CreateOrganization() error = %v", err)
	}
	if repo.organizationID != parentID || created.ParentID == nil || *created.ParentID != parentID {
		t.Fatalf("created organization = %#v, looked up parent = %s", created, repo.organizationID)
	}
	if created.BaseCurrency != "USD" {
		t.Fatalf("company base currency = %s, want USD", created.BaseCurrency)
	}

	missingParentID := uuid.New()
	if _, err := usecase.CreateOrganization(context.Background(), uuid.New(), &AdminOrganization{Code: "missing", Name: "缺失父组织", Kind: OrganizationKindCompany, ParentID: &missingParentID}); err != ErrAdminOrganizationNotFound {
		t.Fatalf("missing parent error = %v, want ErrAdminOrganizationNotFound", err)
	}
	if repo.organizationInput.Code != "BRANCH" {
		t.Fatalf("unexpected organization created after missing parent: %#v", repo.organizationInput)
	}
}

func TestAdminUsecaseCreateOrganizationRejectsRoot(t *testing.T) {
	repo := &adminRepoStub{}
	usecase := NewAdminUsecase(repo, &auditRepoStub{})

	if _, err := usecase.CreateOrganization(context.Background(), uuid.New(), &AdminOrganization{Code: " root ", Name: " 根组织 ", Kind: OrganizationKindCompany}); err != ErrAdminOrganizationParentRequired {
		t.Fatalf("root organization error = %v, want ErrAdminOrganizationParentRequired", err)
	}
}

func TestAdminUsecaseCreateOrganizationValidatesHierarchy(t *testing.T) {
	parentID := uuid.New()
	repo := &adminRepoStub{organization: &AdminOrganization{ID: parentID, Kind: OrganizationKindCompany, BaseCurrency: "CNY"}}
	usecase := NewAdminUsecase(repo, &auditRepoStub{})

	created, err := usecase.CreateOrganization(context.Background(), uuid.New(), &AdminOrganization{
		Code: "finance", Name: "财务部", Kind: OrganizationKindDepartment, ParentID: &parentID,
	})
	if err != nil {
		t.Fatalf("CreateOrganization() error = %v", err)
	}
	if created.Kind != OrganizationKindDepartment || created.BaseCurrency != "CNY" {
		t.Fatalf("created department = %#v", created)
	}

	_, err = usecase.CreateOrganization(context.Background(), uuid.New(), &AdminOrganization{
		Code: "sub_company", Name: "错误下级公司", Kind: OrganizationKindCompany, ParentID: &parentID,
	})
	if err != ErrAdminOrganizationHierarchy {
		t.Fatalf("invalid hierarchy error = %v, want ErrAdminOrganizationHierarchy", err)
	}

	repo.organization = &AdminOrganization{ID: parentID, Kind: OrganizationKindDepartment, BaseCurrency: "CNY"}
	created, err = usecase.CreateOrganization(context.Background(), uuid.New(), &AdminOrganization{
		Code: "customer_service", Name: "客服组", Kind: OrganizationKindTeam, ParentID: &parentID,
	})
	if err != nil {
		t.Fatalf("create team error = %v", err)
	}
	if created.Kind != OrganizationKindTeam || created.BaseCurrency != "CNY" {
		t.Fatalf("created team = %#v", created)
	}
}

func TestAdminUsecaseListAuditLogsRejectsReversedTimeRange(t *testing.T) {
	usecase := NewAdminUsecase(&adminRepoStub{}, &auditRepoStub{})
	start := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	end := start.Add(-time.Minute)
	if _, err := usecase.ListAuditLogs(context.Background(), uuid.New(), AdminAuditLogListOptions{Page: 1, PageSize: 20, StartTime: &start, EndTime: &end}); err != ErrAdminInvalidArgument {
		t.Fatalf("ListAuditLogs() error = %v, want ErrAdminInvalidArgument", err)
	}
}

func TestAdminUsecaseListAuditLogsRejectsLongResourceFilter(t *testing.T) {
	usecase := NewAdminUsecase(&adminRepoStub{}, &auditRepoStub{})
	if _, err := usecase.ListAuditLogs(context.Background(), uuid.New(), AdminAuditLogListOptions{
		Page: 1, PageSize: 20, ResourceType: strings.Repeat("x", 101),
	}); err != ErrAdminInvalidArgument {
		t.Fatalf("ListAuditLogs() error = %v, want ErrAdminInvalidArgument", err)
	}
}

func TestAdminUsecaseCreateRoleNormalizesPermissionKeysAndAudits(t *testing.T) {
	repo := &adminRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewAdminUsecase(repo, audit)
	organizationID := uuid.New()
	actorID := uuid.New()

	created, err := usecase.CreateRole(
		context.Background(),
		organizationID,
		actorID,
		&AdminRole{Code: "  OPERATIONS ", Name: "  操作员 ", DataScope: DataScopeOrganization},
		[]string{" system.user.manage ", "", "system.user.manage", "system.role.manage"},
	)
	if err != nil {
		t.Fatalf("CreateRole() error = %v", err)
	}
	if created.Code != "operations" || created.Name != "操作员" {
		t.Fatalf("normalized role = %#v", created)
	}
	wantKeys := []string{"system.user.manage", "system.role.manage"}
	if len(repo.roleKeys) != len(wantKeys) {
		t.Fatalf("permission key count = %d, want %d", len(repo.roleKeys), len(wantKeys))
	}
	for index, key := range wantKeys {
		if repo.roleKeys[index] != key {
			t.Fatalf("permission key %d = %q, want %q", index, repo.roleKeys[index], key)
		}
	}
	if len(audit.events) != 1 || audit.events[0].Action != "admin.role.create" || audit.events[0].UserID == nil || *audit.events[0].UserID != actorID {
		t.Fatalf("audit events = %#v", audit.events)
	}
}

func TestAdminUsecaseCreateUserRequiresStrongPasswordAndAudits(t *testing.T) {
	repo := &adminRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewAdminUsecase(repo, audit)
	organizationID := uuid.New()
	actorID := uuid.New()

	if _, err := usecase.CreateUser(context.Background(), organizationID, actorID, &AdminUser{Username: "new-user", DisplayName: "新用户"}, "short", nil); err != ErrAdminInvalidArgument {
		t.Fatalf("short password error = %v, want ErrAdminInvalidArgument", err)
	}
	if repo.userPassword != "" {
		t.Fatal("short password was sent to repository")
	}

	created, err := usecase.CreateUser(context.Background(), organizationID, actorID, &AdminUser{Username: " New.User ", DisplayName: " 新用户 ", Email: stringPtr(" user@example.com ")}, "a-strong-password", nil)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if created.Username != "new.user" || created.DisplayName != "新用户" || created.Email == nil || *created.Email != "user@example.com" {
		t.Fatalf("normalized user = %#v", created)
	}
	if repo.userPassword == "a-strong-password" || repo.userPassword == "" {
		t.Fatal("repository did not receive a password hash")
	}
	if len(audit.events) != 1 || audit.events[0].Action != "admin.user.create" {
		t.Fatalf("audit events = %#v", audit.events)
	}
}

func TestAdminUsecaseResetUserPasswordHashesAndAudits(t *testing.T) {
	repo := &adminRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewAdminUsecase(repo, audit)
	organizationID := uuid.New()
	actorID := uuid.New()
	userID := uuid.New()

	if err := usecase.ResetUserPassword(context.Background(), organizationID, actorID, userID, "short"); err != ErrAdminInvalidArgument {
		t.Fatalf("short reset password error = %v, want ErrAdminInvalidArgument", err)
	}
	if err := usecase.ResetUserPassword(context.Background(), organizationID, actorID, userID, "new-strong-password"); err != nil {
		t.Fatalf("ResetUserPassword() error = %v", err)
	}
	if repo.resetPassword == "" || repo.resetPassword == "new-strong-password" {
		t.Fatal("reset password was not hashed")
	}
	if len(audit.events) != 1 || audit.events[0].Action != "admin.user.password.reset" {
		t.Fatalf("audit events = %#v", audit.events)
	}
}

func TestAdminUsecaseTerminateUserRejectsSelfAndAudits(t *testing.T) {
	repo := &adminRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewAdminUsecase(repo, audit)
	organizationID := uuid.New()
	actorID := uuid.New()

	if err := usecase.TerminateUser(context.Background(), organizationID, actorID, actorID); err != ErrAdminUserSelfDelete {
		t.Fatalf("TerminateUser() self error = %v, want ErrAdminUserSelfDelete", err)
	}
	if repo.deletedUserID != uuid.Nil {
		t.Fatal("self deletion was sent to repository")
	}

	userID := uuid.New()
	if err := usecase.TerminateUser(context.Background(), organizationID, actorID, userID); err != nil {
		t.Fatalf("TerminateUser() error = %v", err)
	}
	if repo.deleteOrgID != organizationID || repo.deletedUserID != userID {
		t.Fatalf("deleted organization/user = %s/%s", repo.deleteOrgID, repo.deletedUserID)
	}
	if len(audit.events) != 1 || audit.events[0].Action != "admin.user.terminate" || audit.events[0].Details["resource_id"] != userID.String() {
		t.Fatalf("audit events = %#v", audit.events)
	}
}

func TestNormalizeRoleRejectsUnspecifiedDataScope(t *testing.T) {
	if _, err := normalizeRole(&AdminRole{Code: "viewer", Name: "查看者"}); err != ErrAdminInvalidArgument {
		t.Fatalf("normalizeRole() error = %v, want ErrAdminInvalidArgument", err)
	}
}

func TestAdminUsecaseUpdateUserRejectsNilID(t *testing.T) {
	usecase := NewAdminUsecase(&adminRepoStub{}, &auditRepoStub{})
	if _, err := usecase.UpdateUser(context.Background(), uuid.New(), uuid.New(), uuid.Nil, &AdminUser{ID: uuid.Nil, DisplayName: "用户"}, nil); err != ErrAdminInvalidArgument {
		t.Fatalf("UpdateUser() error = %v, want ErrAdminInvalidArgument", err)
	}
}

func TestAdminUsecaseCreateUserMembershipValidatesAndAudits(t *testing.T) {
	repo := &adminRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewAdminUsecase(repo, audit)
	actorID := uuid.New()
	userID := uuid.New()
	organizationID := uuid.New()
	roleID := uuid.New()

	if _, err := usecase.CreateUserMembership(context.Background(), actorID, userID, uuid.Nil, false, nil); err != ErrAdminInvalidArgument {
		t.Fatalf("CreateUserMembership() missing organization error = %v", err)
	}
	created, err := usecase.CreateUserMembership(context.Background(), actorID, userID, organizationID, true, []uuid.UUID{roleID})
	if err != nil {
		t.Fatalf("CreateUserMembership() error = %v", err)
	}
	if created.OrganizationID != organizationID || !created.Enabled || !created.Primary {
		t.Fatalf("created membership = %#v", created)
	}
	if len(repo.membershipRoleIDs) != 1 || repo.membershipRoleIDs[0] != roleID {
		t.Fatalf("membership role IDs = %#v", repo.membershipRoleIDs)
	}
	if len(audit.events) != 1 || audit.events[0].Action != "admin.user.membership.create" {
		t.Fatalf("audit events = %#v", audit.events)
	}
}

func TestAdminUsecaseUpdateUserMembershipRejectsDisabledPrimary(t *testing.T) {
	usecase := NewAdminUsecase(&adminRepoStub{}, &auditRepoStub{})
	if _, err := usecase.UpdateUserMembership(context.Background(), uuid.New(), uuid.New(), uuid.New(), false, true, nil); err != ErrAdminInvalidArgument {
		t.Fatalf("UpdateUserMembership() error = %v, want ErrAdminInvalidArgument", err)
	}
	actorID := uuid.New()
	if _, err := usecase.UpdateUserMembership(context.Background(), actorID, actorID, uuid.New(), false, false, nil); err != ErrAdminUserSelfDelete {
		t.Fatalf("UpdateUserMembership() self-disable error = %v, want ErrAdminUserSelfDelete", err)
	}
}

func TestAdminUsecaseDeleteUserMembershipRejectsSelfAndAudits(t *testing.T) {
	repo := &adminRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewAdminUsecase(repo, audit)
	actorID := uuid.New()
	membershipID := uuid.New()

	if err := usecase.DeleteUserMembership(context.Background(), actorID, actorID, membershipID); err != ErrAdminUserSelfDelete {
		t.Fatalf("DeleteUserMembership() self error = %v", err)
	}
	userID := uuid.New()
	if err := usecase.DeleteUserMembership(context.Background(), actorID, userID, membershipID); err != nil {
		t.Fatalf("DeleteUserMembership() error = %v", err)
	}
	if repo.deletedMembership != membershipID {
		t.Fatalf("deleted membership = %s, want %s", repo.deletedMembership, membershipID)
	}
	if len(audit.events) != 1 || audit.events[0].Action != "admin.user.membership.delete" {
		t.Fatalf("audit events = %#v", audit.events)
	}
}

func TestPrincipalPermissionRequiresDataScope(t *testing.T) {
	principal := &Principal{
		Permissions: []string{"system.user.manage"},
		RoleScopes:  []RoleScope{{RoleCode: "viewer", DataScope: DataScopeSelf}},
		RolePermissions: map[string]map[string]struct{}{
			"viewer": {"system.user.manage": {}},
		},
	}
	if principal.HasPermissionInScope("system.user.manage", DataScopeOrganization) {
		t.Fatal("self-scoped role unexpectedly passed organization authorization")
	}
	principal.RoleScopes = []RoleScope{{RoleCode: "manager", DataScope: DataScopeOrganization}}
	principal.RolePermissions = map[string]map[string]struct{}{
		"manager": {"system.user.manage": {}},
	}
	if !principal.HasPermissionInScope("system.user.manage", DataScopeOrganization) {
		t.Fatal("organization-scoped role was denied organization authorization")
	}
	if principal.HasPermissionInScope("system.user.manage", DataScopeAll) {
		t.Fatal("organization-scoped role unexpectedly passed global authorization")
	}

	principal.Permissions = []string{"system.platform.access", "system.user.manage"}
	principal.RoleScopes = []RoleScope{
		{RoleCode: "platform", DataScope: DataScopeAll},
		{RoleCode: "operator", DataScope: DataScopeSelf},
	}
	principal.RolePermissions = map[string]map[string]struct{}{
		"platform": {"system.platform.access": {}},
		"operator": {"system.user.manage": {}},
	}
	if principal.HasPermissionInScope("system.user.manage", DataScopeOrganization) {
		t.Fatal("permission from a self-scoped role was incorrectly widened by another role")
	}
}

func TestPrincipalOrderOrganizationAccess(t *testing.T) {
	currentOrganizationID := uuid.New()
	readOnlyOrganizationID := uuid.New()
	writableOrganizationID := uuid.New()
	principal := &Principal{
		Organization: Organization{ID: currentOrganizationID},
		OrderOrganizationAccesses: []OrderOrganizationAccess{
			{OrganizationID: readOnlyOrganizationID},
			{OrganizationID: writableOrganizationID, Writable: true},
		},
	}

	if !principal.CanAccessOrderOrganization(currentOrganizationID, true) {
		t.Fatal("current organization must retain write access")
	}
	if !principal.CanAccessOrderOrganization(readOnlyOrganizationID, false) || principal.CanAccessOrderOrganization(readOnlyOrganizationID, true) {
		t.Fatal("read-only organization access was not enforced")
	}
	if !principal.CanAccessOrderOrganization(writableOrganizationID, true) {
		t.Fatal("writable organization access was denied")
	}
	if principal.CanAccessOrderOrganization(uuid.New(), false) {
		t.Fatal("unassigned organization was accessible")
	}
	if got := principal.OrderOrganizationIDs(); len(got) != 3 || got[0] != currentOrganizationID {
		t.Fatalf("OrderOrganizationIDs() = %#v", got)
	}
}

func stringPtr(value string) *string { return &value }

var _ AdminRepo = (*adminRepoStub)(nil)
var _ AuditRepo = (*auditRepoStub)(nil)
