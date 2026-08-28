package biz

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/roncin/roncin-go-admin/server/internal/security/password"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrAdminOrganizationNotFound       = errors.NotFound("ADMIN_ORGANIZATION_NOT_FOUND", "组织不存在")
	ErrAdminOrganizationCodeExists     = errors.Conflict("ADMIN_ORGANIZATION_CODE_EXISTS", "组织编码已存在")
	ErrAdminOrganizationParentRequired = errors.BadRequest("ADMIN_ORGANIZATION_PARENT_REQUIRED", "新建组织必须指定上级组织")
	ErrAdminOrganizationHierarchy      = errors.BadRequest("ADMIN_ORGANIZATION_HIERARCHY_INVALID", "组织层级不合法")
	ErrAdminOrganizationCurrency       = errors.BadRequest("ADMIN_ORGANIZATION_CURRENCY_INVALID", "组织本币必须是启用的 ISO 币种")
	ErrAdminUserNotFound               = errors.NotFound("ADMIN_USER_NOT_FOUND", "用户不存在")
	ErrAdminUsernameExists             = errors.Conflict("ADMIN_USERNAME_EXISTS", "用户名已存在")
	ErrAdminUserSelfDelete             = errors.BadRequest("ADMIN_USER_SELF_DELETE", "不能移除当前登录账号或为其办理离职")
	ErrAdminUserLastMembership         = errors.BadRequest("ADMIN_USER_LAST_MEMBERSHIP", "在职用户必须保留至少一个有效组织；请先加入新组织或办理离职")
	ErrAdminUserTerminationRequired    = errors.BadRequest("ADMIN_USER_TERMINATION_REQUIRED", "停用员工请使用办理离职")
	ErrAdminUserAuthorizationRequired  = errors.BadRequest("ADMIN_USER_AUTHORIZATION_REQUIRED", "外部身份账号必须通过身份授权流程启用")
	ErrAdminUserPasswordUnavailable    = errors.BadRequest("ADMIN_USER_PASSWORD_UNAVAILABLE", "该用户未启用密码登录，不能重置密码")
	ErrAdminUserMembershipNotFound     = errors.NotFound("ADMIN_USER_MEMBERSHIP_NOT_FOUND", "用户组织成员关系不存在")
	ErrAdminUserMembershipExists       = errors.Conflict("ADMIN_USER_MEMBERSHIP_EXISTS", "用户已属于该组织")
	ErrAdminRoleNotFound               = errors.NotFound("ADMIN_ROLE_NOT_FOUND", "角色不存在")
	ErrAdminRoleCodeExists             = errors.Conflict("ADMIN_ROLE_CODE_EXISTS", "角色编码已存在")
	ErrAdminPermissionInvalid          = errors.BadRequest("ADMIN_PERMISSION_INVALID", "权限不存在或不属于当前请求")
	ErrAdminPrivilegeEscalation        = errors.Forbidden("ADMIN_PRIVILEGE_ESCALATION_DENIED", "不能分配超出自身权限范围的角色")
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
	ID           uuid.UUID
	Code         string
	Name         string
	Kind         OrganizationKind
	ParentID     *uuid.UUID
	Enabled      bool
	BaseCurrency string
}

type AdminUser struct {
	ID                       uuid.UUID
	Username                 string
	DisplayName              string
	Email                    *string
	AvatarURL                *string
	WeComUserID              *string
	WeComName                *string
	DingTalkUnionID          *string
	DingTalkUserID           *string
	DingTalkName             *string
	Enabled                  bool
	Status                   AdminUserStatus
	CurrentMembershipEnabled bool
	HasPassword              bool
	RoleIDs                  []uuid.UUID
	RoleCodes                []string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type AdminUserStatus string

const (
	AdminUserStatusActive                  AdminUserStatus = "ACTIVE"
	AdminUserStatusPendingAuthorization    AdminUserStatus = "PENDING_AUTHORIZATION"
	AdminUserStatusTerminated              AdminUserStatus = "TERMINATED"
	AdminUserStatusRemovedFromOrganization AdminUserStatus = "REMOVED_FROM_ORGANIZATION"
	AdminUserStatusDisabled                AdminUserStatus = "DISABLED"
)

type AdminUserMembership struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	OrganizationID   uuid.UUID
	OrganizationCode string
	OrganizationName string
	OrganizationKind OrganizationKind
	Primary          bool
	Enabled          bool
	RoleIDs          []uuid.UUID
	RoleCodes        []string
	RoleNames        []string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type AdminRole struct {
	ID                        uuid.UUID
	OrganizationID            uuid.UUID
	Code                      string
	Name                      string
	DataScope                 DataScope
	Enabled                   bool
	PermissionKeys            []string
	OrderOrganizationAccesses []OrderOrganizationAccess
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

type OrderOrganizationAccess struct {
	OrganizationID uuid.UUID
	Writable       bool
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
	UpdateOrganization(context.Context, uuid.UUID, *AdminOrganization) (*AdminOrganization, error)
	ListUsers(context.Context, uuid.UUID, AdminUserListOptions) (*AdminUserList, error)
	CreateUser(context.Context, uuid.UUID, uuid.UUID, *AdminUser, string, []uuid.UUID) (*AdminUser, error)
	UpdateUser(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *AdminUser, []uuid.UUID) (*AdminUser, error)
	ListUserMemberships(context.Context, uuid.UUID) ([]*AdminUserMembership, error)
	CreateUserMembership(context.Context, uuid.UUID, *AdminUserMembership, []uuid.UUID) (*AdminUserMembership, error)
	UpdateUserMembership(context.Context, uuid.UUID, *AdminUserMembership, []uuid.UUID) (*AdminUserMembership, error)
	DeleteUserMembership(context.Context, uuid.UUID, uuid.UUID) error
	TerminateUser(context.Context, uuid.UUID, uuid.UUID) error
	AuthorizeWeComUser(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *AdminUser, []uuid.UUID) (*AdminUser, error)
	AuthorizeDingTalkUser(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *AdminUser, []uuid.UUID) (*AdminUser, error)
	ResetUserPassword(context.Context, uuid.UUID, uuid.UUID, string, *string) error
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
	if normalized.Kind == OrganizationKindCompany {
		if !validOrganizationCurrency(normalized.BaseCurrency) {
			return nil, ErrAdminOrganizationCurrency
		}
	} else {
		if normalized.BaseCurrency != "" {
			return nil, ErrAdminOrganizationCurrency
		}
		normalized.BaseCurrency = parent.BaseCurrency
	}
	created, err := uc.repo.CreateOrganization(ctx, normalized)
	if err != nil {
		return nil, err
	}
	return created, uc.writeAudit(ctx, userID, nil, "admin.organization.create", created.ID.String())
}

func (uc *AdminUsecase) UpdateOrganization(ctx context.Context, userID, organizationID, id uuid.UUID, name string, enabled bool, baseCurrency string) (*AdminOrganization, error) {
	name = strings.TrimSpace(name)
	if organizationID == uuid.Nil || id == uuid.Nil || name == "" {
		return nil, ErrAdminInvalidArgument
	}
	current, err := uc.repo.GetOrganization(ctx, id)
	if err != nil {
		return nil, err
	}
	baseCurrency = strings.ToUpper(strings.TrimSpace(baseCurrency))
	if current.Kind == OrganizationKindHeadquarters || current.Kind == OrganizationKindCompany {
		if !validOrganizationCurrency(baseCurrency) {
			return nil, ErrAdminOrganizationCurrency
		}
	} else if baseCurrency != "" {
		return nil, ErrAdminOrganizationCurrency
	}
	updated, err := uc.repo.UpdateOrganization(ctx, organizationID, &AdminOrganization{ID: id, Name: name, Enabled: enabled, Kind: current.Kind, BaseCurrency: baseCurrency})
	if err != nil {
		return nil, err
	}
	return updated, uc.writeAudit(ctx, userID, &id, "admin.organization.update", updated.Code)
}

func (uc *AdminUsecase) ListUsers(ctx context.Context, organizationID uuid.UUID, options AdminUserListOptions) (*AdminUserList, error) {
	if organizationID == uuid.Nil || !ValidListPagination(options.Page, options.PageSize) {
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
	created, err := uc.repo.CreateUser(ctx, organizationID, actorID, normalized, hash, roleIDs)
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
	updated, err := uc.repo.UpdateUser(ctx, organizationID, actorID, id, normalized, roleIDs)
	if err != nil {
		return nil, err
	}
	return updated, uc.writeAudit(ctx, actorID, &id, "admin.user.update", updated.Username)
}

func (uc *AdminUsecase) ListUserMemberships(ctx context.Context, userID uuid.UUID) ([]*AdminUserMembership, error) {
	if userID == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	return uc.repo.ListUserMemberships(ctx, userID)
}

func (uc *AdminUsecase) CreateUserMembership(ctx context.Context, actorID, userID, organizationID uuid.UUID, primary bool, roleIDs []uuid.UUID) (*AdminUserMembership, error) {
	if actorID == uuid.Nil || userID == uuid.Nil || organizationID == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	created, err := uc.repo.CreateUserMembership(ctx, actorID, &AdminUserMembership{
		UserID:         userID,
		OrganizationID: organizationID,
		Primary:        primary,
		Enabled:        true,
	}, roleIDs)
	if err != nil {
		return nil, err
	}
	return created, uc.writeAudit(ctx, actorID, &userID, "admin.user.membership.create", created.OrganizationID.String())
}

func (uc *AdminUsecase) UpdateUserMembership(ctx context.Context, actorID, userID, membershipID uuid.UUID, enabled, primary bool, roleIDs []uuid.UUID) (*AdminUserMembership, error) {
	if actorID == uuid.Nil || userID == uuid.Nil || membershipID == uuid.Nil || primary && !enabled {
		return nil, ErrAdminInvalidArgument
	}
	if actorID == userID && !enabled {
		return nil, ErrAdminUserSelfDelete
	}
	updated, err := uc.repo.UpdateUserMembership(ctx, actorID, &AdminUserMembership{
		ID:      membershipID,
		UserID:  userID,
		Enabled: enabled,
		Primary: primary,
	}, roleIDs)
	if err != nil {
		return nil, err
	}
	return updated, uc.writeAudit(ctx, actorID, &userID, "admin.user.membership.update", updated.OrganizationID.String())
}

func (uc *AdminUsecase) DeleteUserMembership(ctx context.Context, actorID, userID, membershipID uuid.UUID) error {
	if actorID == uuid.Nil || userID == uuid.Nil || membershipID == uuid.Nil {
		return ErrAdminInvalidArgument
	}
	if actorID == userID {
		return ErrAdminUserSelfDelete
	}
	if err := uc.repo.DeleteUserMembership(ctx, userID, membershipID); err != nil {
		return err
	}
	return uc.writeAudit(ctx, actorID, &userID, "admin.user.membership.delete", membershipID.String())
}

func (uc *AdminUsecase) TerminateUser(ctx context.Context, organizationID, actorID, id uuid.UUID) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil {
		return ErrAdminInvalidArgument
	}
	if actorID == id {
		return ErrAdminUserSelfDelete
	}
	if err := uc.repo.TerminateUser(ctx, organizationID, id); err != nil {
		return err
	}
	return uc.writeAudit(ctx, actorID, &id, "admin.user.terminate", "")
}

func (uc *AdminUsecase) AuthorizeWeComUser(ctx context.Context, sourceOrganizationID, targetOrganizationID, actorID uuid.UUID, input *AdminUser, roleIDs []uuid.UUID) (*AdminUser, error) {
	if sourceOrganizationID == uuid.Nil || targetOrganizationID == uuid.Nil || len(roleIDs) == 0 {
		return nil, ErrAdminInvalidArgument
	}
	normalized, err := normalizeUser(input)
	if err != nil {
		return nil, err
	}
	authorized, err := uc.repo.AuthorizeWeComUser(ctx, sourceOrganizationID, targetOrganizationID, actorID, normalized, roleIDs)
	if err != nil {
		return nil, err
	}
	return authorized, uc.writeAudit(ctx, actorID, &authorized.ID, "admin.user.wecom.authorize", authorized.Username)
}

func (uc *AdminUsecase) AuthorizeDingTalkUser(ctx context.Context, sourceOrganizationID, targetOrganizationID, actorID uuid.UUID, input *AdminUser, roleIDs []uuid.UUID) (*AdminUser, error) {
	if sourceOrganizationID == uuid.Nil || targetOrganizationID == uuid.Nil || len(roleIDs) == 0 {
		return nil, ErrAdminInvalidArgument
	}
	normalized, err := normalizeUser(input)
	if err != nil {
		return nil, err
	}
	authorized, err := uc.repo.AuthorizeDingTalkUser(ctx, sourceOrganizationID, targetOrganizationID, actorID, normalized, roleIDs)
	if err != nil {
		return nil, err
	}
	return authorized, uc.writeAudit(ctx, actorID, &authorized.ID, "admin.user.dingtalk.authorize", authorized.Username)
}

func (uc *AdminUsecase) ResetUserPassword(ctx context.Context, organizationID, actorID, id uuid.UUID, plainPassword string, username *string) error {
	if organizationID == uuid.Nil || id == uuid.Nil || len(strings.TrimSpace(plainPassword)) < 12 {
		return ErrAdminInvalidArgument
	}
	if username != nil {
		trimmed := strings.TrimSpace(*username)
		if !validUsername(trimmed) {
			return ErrAdminInvalidArgument
		}
	}
	hash, err := password.Hash(plainPassword)
	if err != nil {
		return fmt.Errorf("hash reset password: %w", err)
	}
	if err := uc.repo.ResetUserPassword(ctx, organizationID, id, hash, username); err != nil {
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
	if err := uc.validateOrderOrganizationAccesses(ctx, organizationID, normalized.OrderOrganizationAccesses); err != nil {
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
	if err := uc.validateOrderOrganizationAccesses(ctx, organizationID, normalized.OrderOrganizationAccesses); err != nil {
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
	if organizationID == uuid.Nil || !ValidListPagination(options.Page, options.PageSize) {
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
	output.BaseCurrency = strings.ToUpper(strings.TrimSpace(output.BaseCurrency))
	if output.Code == "" || output.Name == "" || !output.Kind.Valid() || output.Kind == OrganizationKindHeadquarters {
		return nil, ErrAdminInvalidArgument
	}
	return &output, nil
}

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func validOrganizationCurrency(value string) bool {
	return currencyPattern.MatchString(value)
}

func validUsername(value string) bool {
	return len(value) >= 3 && len(value) <= 64 && usernamePattern.MatchString(value)
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

func (uc *AdminUsecase) validateOrderOrganizationAccesses(ctx context.Context, sourceOrganizationID uuid.UUID, accesses []OrderOrganizationAccess) error {
	seen := make(map[uuid.UUID]struct{}, len(accesses))
	for _, access := range accesses {
		if access.OrganizationID == uuid.Nil || access.OrganizationID == sourceOrganizationID {
			return ErrAdminInvalidArgument
		}
		if _, ok := seen[access.OrganizationID]; ok {
			return ErrAdminInvalidArgument
		}
		seen[access.OrganizationID] = struct{}{}
		organization, err := uc.repo.GetOrganization(ctx, access.OrganizationID)
		if err != nil || !organization.Enabled || organization.Kind != OrganizationKindCompany {
			return ErrAdminInvalidArgument
		}
	}
	return nil
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
