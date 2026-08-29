package biz

import (
	"context"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var ErrAdminInvalidArgument = errors.BadRequest("ADMIN_INVALID_ARGUMENT", "管理参数不合法")

// AdminRepo 按组织、用户、用户成员、角色权限、审计日志聚合组织实现；
// 各聚合的领域对象、错误与用例分散在 admin_organization.go、admin_user.go、
// admin_user_membership.go、admin_role.go、admin_audit.go 中。
type AdminRepo interface {
	ListOrganizations(context.Context) ([]*AdminOrganization, error)
	GetOrganization(context.Context, uuid.UUID) (*AdminOrganization, error)
	CreateOrganization(context.Context, *AdminOrganization, *AuditEvent) (*AdminOrganization, error)
	UpdateOrganization(context.Context, uuid.UUID, *AdminOrganization, *AuditEvent) (*AdminOrganization, error)
	ListUsers(context.Context, uuid.UUID, AdminUserListOptions) (*AdminUserList, error)
	CreateUser(context.Context, uuid.UUID, *AdminUser, string, []uuid.UUID, *AuditEvent) (*AdminUser, error)
	UpdateUser(context.Context, uuid.UUID, uuid.UUID, *AdminUser, []uuid.UUID, *AuditEvent) (*AdminUser, error)
	ListUserMemberships(context.Context, uuid.UUID) ([]*AdminUserMembership, error)
	GetUserMembership(context.Context, uuid.UUID, uuid.UUID) (*AdminUserMembership, error)
	CreateUserMembership(context.Context, *AdminUserMembership, []uuid.UUID, *AuditEvent) (*AdminUserMembership, error)
	UpdateUserMembership(context.Context, *AdminUserMembership, []uuid.UUID, *AuditEvent) (*AdminUserMembership, error)
	DeleteUserMembership(context.Context, uuid.UUID, uuid.UUID, *AuditEvent) error
	TerminateUser(context.Context, uuid.UUID, uuid.UUID, *AuditEvent) error
	AuthorizeWeComUser(context.Context, uuid.UUID, uuid.UUID, *AdminUser, []uuid.UUID, *AuditEvent) (*AdminUser, error)
	AuthorizeDingTalkUser(context.Context, uuid.UUID, uuid.UUID, *AdminUser, []uuid.UUID, *NotificationIntent, *AuditEvent) (*AdminUser, error)
	ResetUserPassword(context.Context, uuid.UUID, uuid.UUID, string, *string, *AuditEvent) error
	ListRoles(context.Context, uuid.UUID) ([]*AdminRole, error)
	GetRole(context.Context, uuid.UUID, uuid.UUID) (*AdminRole, error)
	CreateRole(context.Context, uuid.UUID, *AdminRole, []string, *AuditEvent) (*AdminRole, error)
	UpdateRole(context.Context, uuid.UUID, uuid.UUID, *AdminRole, []string, *AuditEvent) (*AdminRole, error)
	ListPermissions(context.Context) ([]*AdminPermission, error)
	ListAuditLogs(context.Context, uuid.UUID, AdminAuditLogListOptions) (*AdminAuditLogList, error)
	GetActorRolesPrivilegeProfiles(context.Context, uuid.UUID, uuid.UUID) ([]*AdminRoleProfile, error)
	GetRolesPrivilegeProfiles(context.Context, uuid.UUID, []uuid.UUID) ([]*AdminRoleProfile, error)
}

type AdminUsecase struct {
	repo AdminRepo
}

func NewAdminUsecase(repo AdminRepo) *AdminUsecase {
	return &AdminUsecase{repo: repo}
}

func adminAuditEvent(ctx context.Context, userID uuid.UUID, resourceID *uuid.UUID, action, value string) *AuditEvent {
	organizationID, _ := PrincipalFromContext(ctx)
	var orgID *uuid.UUID
	if organizationID != nil {
		orgID = &organizationID.Organization.ID
	}
	return &AuditEvent{OrganizationID: orgID, UserID: &userID, Action: action, Result: "success", Details: map[string]string{"value": value, "resource_id": resourceIDString(resourceID)}}
}

func resourceIDString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

var _ AdminRepo = (AdminRepo)(nil)
