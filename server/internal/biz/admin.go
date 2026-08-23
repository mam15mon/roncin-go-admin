package biz

import (
	"context"
	"fmt"
	"github.com/roncin/roncin-go-admin/server/internal/security/password"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrAdminOrganizationNotFound       = errors.NotFound("ADMIN_ORGANIZATION_NOT_FOUND", "组织不存在")
	ErrAdminOrganizationCodeExists     = errors.Conflict("ADMIN_ORGANIZATION_CODE_EXISTS", "组织编码已存在")
	ErrAdminOrganizationParentRequired = errors.BadRequest("ADMIN_ORGANIZATION_PARENT_REQUIRED", "新建组织必须指定上级组织")
	ErrAdminOrganizationHierarchy      = errors.BadRequest("ADMIN_ORGANIZATION_HIERARCHY_INVALID", "组织层级不合法")
	ErrAdminUserNotFound               = errors.NotFound("ADMIN_USER_NOT_FOUND", "用户不存在")
	ErrAdminUsernameExists             = errors.Conflict("ADMIN_USERNAME_EXISTS", "用户名已存在")
	ErrAdminRoleNotFound               = errors.NotFound("ADMIN_ROLE_NOT_FOUND", "角色不存在")
	ErrAdminRoleCodeExists             = errors.Conflict("ADMIN_ROLE_CODE_EXISTS", "角色编码已存在")
	ErrAdminPermissionInvalid          = errors.BadRequest("ADMIN_PERMISSION_INVALID", "权限不存在或不属于当前请求")
	ErrAdminInvalidArgument            = errors.BadRequest("ADMIN_INVALID_ARGUMENT", "管理参数不合法")
)

type OrganizationKind string

const (
	OrganizationKindHeadquarters OrganizationKind = "headquarters"
	OrganizationKindCompany      OrganizationKind = "company"
	OrganizationKindDepartment   OrganizationKind = "department"
	OrganizationKindTeam         OrganizationKind = "team"
)

func (kind OrganizationKind) Valid() bool {
	return kind == OrganizationKindHeadquarters || kind == OrganizationKindCompany || kind == OrganizationKindDepartment || kind == OrganizationKindTeam
}

type AdminOrganization struct {
	ID       uuid.UUID
	Code     string
	Name     string
	Kind     OrganizationKind
	ParentID *uuid.UUID
	Enabled  bool
}

type AdminUser struct {
	ID          uuid.UUID
	Username    string
	DisplayName string
	Email       *string
	WeComUserID *string
	WeComName   *string
	Enabled     bool
	RoleIDs     []uuid.UUID
	RoleCodes   []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AdminRole struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Code           string
	Name           string
	DataScope      DataScope
	Enabled        bool
	PermissionKeys []string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type AdminPermission struct {
	Key         string
	Name        string
	Group       string
	Description string
}

type AdminUserListOptions struct {
	Page     int
	PageSize int
	Keyword  string
}

type AdminUserList struct {
	Items    []*AdminUser
	Total    int
	Page     int
	PageSize int
}

type AdminAuditLogListOptions struct {
	Page         int
	PageSize     int
	Action       string
	UserID       *uuid.UUID
	StartTime    *time.Time
	EndTime      *time.Time
	ResourceType string
	ResourceID   string
}

type AdminAuditLogList struct {
	Items    []*AuditLog
	Total    int
	Page     int
	PageSize int
}

type AdminRepo interface {
	ListOrganizations(context.Context) ([]*AdminOrganization, error)
	GetOrganization(context.Context, uuid.UUID) (*AdminOrganization, error)
	CreateOrganization(context.Context, *AdminOrganization) (*AdminOrganization, error)
	UpdateOrganization(context.Context, uuid.UUID, uuid.UUID, string, bool) (*AdminOrganization, error)
	ListUsers(context.Context, uuid.UUID, AdminUserListOptions) (*AdminUserList, error)
	CreateUser(context.Context, uuid.UUID, *AdminUser, string, []uuid.UUID) (*AdminUser, error)
	UpdateUser(context.Context, uuid.UUID, uuid.UUID, *AdminUser, []uuid.UUID) (*AdminUser, error)
	ResetUserPassword(context.Context, uuid.UUID, uuid.UUID, string) error
	ListRoles(context.Context, uuid.UUID) ([]*AdminRole, error)
	CreateRole(context.Context, uuid.UUID, *AdminRole, []string) (*AdminRole, error)
	UpdateRole(context.Context, uuid.UUID, uuid.UUID, *AdminRole, []string) (*AdminRole, error)
	ListPermissions(context.Context) ([]*AdminPermission, error)
	ListAuditLogs(context.Context, uuid.UUID, AdminAuditLogListOptions) (*AdminAuditLogList, error)
}

type AdminUsecase struct {
	repo  AdminRepo
	audit AuditRepo
}

func NewAdminUsecase(repo AdminRepo, audit AuditRepo) *AdminUsecase {
	return &AdminUsecase{repo: repo, audit: audit}
}

func (uc *AdminUsecase) ListOrganizations(ctx context.Context, organizationID uuid.UUID) ([]*AdminOrganization, error) {
	if organizationID == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	return uc.repo.ListOrganizations(ctx)
}

func (uc *AdminUsecase) CreateOrganization(ctx context.Context, userID uuid.UUID, input *AdminOrganization) (*AdminOrganization, error) {
	normalized, err := normalizeOrganization(input)
	if err != nil {
		return nil, err
	}
	if normalized.ParentID == nil {
		return nil, ErrAdminOrganizationParentRequired
	}
	if *normalized.ParentID == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	parent, err := uc.repo.GetOrganization(ctx, *normalized.ParentID)
	if err != nil {
		return nil, err
	}
	if parent.Kind == OrganizationKindHeadquarters && normalized.Kind != OrganizationKindCompany ||
		parent.Kind == OrganizationKindCompany && normalized.Kind != OrganizationKindDepartment ||
		parent.Kind == OrganizationKindDepartment && normalized.Kind != OrganizationKindTeam ||
		parent.Kind == OrganizationKindTeam {
		return nil, ErrAdminOrganizationHierarchy
	}
	created, err := uc.repo.CreateOrganization(ctx, normalized)
	if err != nil {
		return nil, err
	}
	return created, uc.writeAudit(ctx, userID, nil, "admin.organization.create", created.ID.String())
}

func (uc *AdminUsecase) UpdateOrganization(ctx context.Context, userID, organizationID, id uuid.UUID, name string, enabled bool) (*AdminOrganization, error) {
	name = strings.TrimSpace(name)
	if organizationID == uuid.Nil || id == uuid.Nil || name == "" {
		return nil, ErrAdminInvalidArgument
	}
	updated, err := uc.repo.UpdateOrganization(ctx, organizationID, id, name, enabled)
	if err != nil {
		return nil, err
	}
	return updated, uc.writeAudit(ctx, userID, &id, "admin.organization.update", updated.Code)
}

func (uc *AdminUsecase) ListUsers(ctx context.Context, organizationID uuid.UUID, options AdminUserListOptions) (*AdminUserList, error) {
	if organizationID == uuid.Nil || options.Page < 1 || options.PageSize < 1 || options.PageSize > 100 {
		return nil, ErrAdminInvalidArgument
	}
	options.Keyword = strings.TrimSpace(options.Keyword)
	return uc.repo.ListUsers(ctx, organizationID, options)
}

func (uc *AdminUsecase) CreateUser(ctx context.Context, organizationID, actorID uuid.UUID, input *AdminUser, plainPassword string, roleIDs []uuid.UUID) (*AdminUser, error) {
	normalized, err := normalizeUser(input)
	if err != nil || len(strings.TrimSpace(plainPassword)) < 12 {
		return nil, ErrAdminInvalidArgument
	}
	hash, err := password.Hash(plainPassword)
	if err != nil {
		return nil, fmt.Errorf("hash admin password: %w", err)
	}
	created, err := uc.repo.CreateUser(ctx, organizationID, normalized, hash, roleIDs)
	if err != nil {
		return nil, err
	}
	return created, uc.writeAudit(ctx, actorID, &created.ID, "admin.user.create", created.Username)
}

func (uc *AdminUsecase) UpdateUser(ctx context.Context, organizationID, actorID, id uuid.UUID, input *AdminUser, roleIDs []uuid.UUID) (*AdminUser, error) {
	if id == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	normalized, err := normalizeUser(input)
	if err != nil {
		return nil, err
	}
	updated, err := uc.repo.UpdateUser(ctx, organizationID, id, normalized, roleIDs)
	if err != nil {
		return nil, err
	}
	return updated, uc.writeAudit(ctx, actorID, &id, "admin.user.update", updated.Username)
}

func (uc *AdminUsecase) ResetUserPassword(ctx context.Context, organizationID, actorID, id uuid.UUID, plainPassword string) error {
	if organizationID == uuid.Nil || id == uuid.Nil || len(strings.TrimSpace(plainPassword)) < 12 {
		return ErrAdminInvalidArgument
	}
	hash, err := password.Hash(plainPassword)
	if err != nil {
		return fmt.Errorf("hash reset password: %w", err)
	}
	if err := uc.repo.ResetUserPassword(ctx, organizationID, id, hash); err != nil {
		return err
	}
	return uc.writeAudit(ctx, actorID, &id, "admin.user.password.reset", "")
}

func (uc *AdminUsecase) ListRoles(ctx context.Context, organizationID uuid.UUID) ([]*AdminRole, error) {
	if organizationID == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	return uc.repo.ListRoles(ctx, organizationID)
}

func (uc *AdminUsecase) CreateRole(ctx context.Context, organizationID, actorID uuid.UUID, input *AdminRole, permissionKeys []string) (*AdminRole, error) {
	normalized, err := normalizeRole(input)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.CreateRole(ctx, organizationID, normalized, normalizeKeys(permissionKeys))
	if err != nil {
		return nil, err
	}
	return created, uc.writeAudit(ctx, actorID, &created.ID, "admin.role.create", created.Code)
}

func (uc *AdminUsecase) UpdateRole(ctx context.Context, organizationID, actorID, id uuid.UUID, input *AdminRole, permissionKeys []string) (*AdminRole, error) {
	if id == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	normalized, err := normalizeRole(input)
	if err != nil {
		return nil, err
	}
	updated, err := uc.repo.UpdateRole(ctx, organizationID, id, normalized, normalizeKeys(permissionKeys))
	if err != nil {
		return nil, err
	}
	return updated, uc.writeAudit(ctx, actorID, &id, "admin.role.update", updated.Code)
}

func (uc *AdminUsecase) ListPermissions(ctx context.Context) ([]*AdminPermission, error) {
	return uc.repo.ListPermissions(ctx)
}

func (uc *AdminUsecase) ListAuditLogs(ctx context.Context, organizationID uuid.UUID, options AdminAuditLogListOptions) (*AdminAuditLogList, error) {
	if organizationID == uuid.Nil || options.Page < 1 || options.PageSize < 1 || options.PageSize > 100 {
		return nil, ErrAdminInvalidArgument
	}
	options.Action = strings.TrimSpace(options.Action)
	options.ResourceType = strings.TrimSpace(options.ResourceType)
	options.ResourceID = strings.TrimSpace(options.ResourceID)
	if utf8.RuneCountInString(options.ResourceType) > 100 || utf8.RuneCountInString(options.ResourceID) > 160 {
		return nil, ErrAdminInvalidArgument
	}
	if options.StartTime != nil && options.EndTime != nil && options.StartTime.After(*options.EndTime) {
		return nil, ErrAdminInvalidArgument
	}
	return uc.repo.ListAuditLogs(ctx, organizationID, options)
}

func (uc *AdminUsecase) writeAudit(ctx context.Context, userID uuid.UUID, resourceID *uuid.UUID, action, value string) error {
	organizationID, _ := PrincipalFromContext(ctx)
	var orgID *uuid.UUID
	if organizationID != nil {
		orgID = &organizationID.Organization.ID
	}
	return uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: orgID, UserID: &userID, Action: action, Result: "success", Details: map[string]string{"value": value, "resource_id": resourceIDString(resourceID)}})
}

func resourceIDString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func normalizeOrganization(input *AdminOrganization) (*AdminOrganization, error) {
	if input == nil {
		return nil, ErrAdminInvalidArgument
	}
	output := *input
	output.Code = strings.ToUpper(strings.TrimSpace(output.Code))
	output.Name = strings.TrimSpace(output.Name)
	if output.Code == "" || output.Name == "" || !output.Kind.Valid() || output.Kind == OrganizationKindHeadquarters {
		return nil, ErrAdminInvalidArgument
	}
	return &output, nil
}

func normalizeUser(input *AdminUser) (*AdminUser, error) {
	if input == nil || input.ID == uuid.Nil && strings.TrimSpace(input.Username) == "" {
		return nil, ErrAdminInvalidArgument
	}
	output := *input
	output.Username = strings.ToLower(strings.TrimSpace(output.Username))
	output.DisplayName = strings.TrimSpace(output.DisplayName)
	if output.DisplayName == "" || (output.ID == uuid.Nil && output.Username == "") {
		return nil, ErrAdminInvalidArgument
	}
	if output.Email != nil {
		value := strings.TrimSpace(*output.Email)
		output.Email = &value
	}
	return &output, nil
}

func normalizeRole(input *AdminRole) (*AdminRole, error) {
	if input == nil {
		return nil, ErrAdminInvalidArgument
	}
	output := *input
	output.Code = strings.ToLower(strings.TrimSpace(output.Code))
	output.Name = strings.TrimSpace(output.Name)
	if output.Name == "" || (output.ID == uuid.Nil && output.Code == "") || !output.DataScope.Valid() {
		return nil, ErrAdminInvalidArgument
	}
	return &output, nil
}

func (scope DataScope) Valid() bool {
	return scope == DataScopeAll || scope == DataScopeOrganization || scope == DataScopeOrganizationTree || scope == DataScopeSelf
}

func normalizeKeys(keys []string) []string {
	seen := make(map[string]struct{}, len(keys))
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	return result
}

var _ AdminRepo = (AdminRepo)(nil)
