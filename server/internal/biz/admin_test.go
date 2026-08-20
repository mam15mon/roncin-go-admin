package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type adminRepoStub struct {
	userListOptions AdminUserListOptions
	roleKeys        []string
	userPassword    string
	userInput       *AdminUser
}

func (s *adminRepoStub) ListOrganizations(context.Context, uuid.UUID) ([]*AdminOrganization, error) {
	return nil, nil
}

func (s *adminRepoStub) CreateOrganization(_ context.Context, input *AdminOrganization) (*AdminOrganization, error) {
	return input, nil
}

func (s *adminRepoStub) UpdateOrganization(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string, _ bool) (*AdminOrganization, error) {
	return &AdminOrganization{}, nil
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
	if _, err := usecase.ListUsers(context.Background(), organizationID, AdminUserListOptions{Page: 1, PageSize: 101}); err != ErrAdminInvalidArgument {
		t.Fatalf("invalid page size error = %v, want ErrAdminInvalidArgument", err)
	}
}

func TestAdminUsecaseListOrganizationsRequiresOrganizationScope(t *testing.T) {
	usecase := NewAdminUsecase(&adminRepoStub{}, &auditRepoStub{})
	if _, err := usecase.ListOrganizations(context.Background(), uuid.Nil); err != ErrAdminInvalidArgument {
		t.Fatalf("ListOrganizations() error = %v, want ErrAdminInvalidArgument", err)
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

func stringPtr(value string) *string { return &value }

var _ AdminRepo = (*adminRepoStub)(nil)
var _ AuditRepo = (*auditRepoStub)(nil)
