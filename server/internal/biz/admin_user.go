package biz

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/security/password"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrAdminUserNotFound              = errors.NotFound("ADMIN_USER_NOT_FOUND", "用户不存在")
	ErrAdminUsernameExists            = errors.Conflict("ADMIN_USERNAME_EXISTS", "用户名已存在")
	ErrAdminUserSelfDelete            = errors.BadRequest("ADMIN_USER_SELF_DELETE", "不能移除当前登录账号或为其办理离职")
	ErrAdminUserLastMembership        = errors.BadRequest("ADMIN_USER_LAST_MEMBERSHIP", "在职用户必须保留至少一个有效组织；请先加入新组织或办理离职")
	ErrAdminUserTerminationRequired   = errors.BadRequest("ADMIN_USER_TERMINATION_REQUIRED", "停用员工请使用办理离职")
	ErrAdminUserAuthorizationRequired = errors.BadRequest("ADMIN_USER_AUTHORIZATION_REQUIRED", "外部身份账号必须通过身份授权流程启用")
)

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

func (uc *AdminUsecase) ListUsers(ctx context.Context, organizationID uuid.UUID, options AdminUserListOptions) (*AdminUserList, error) {
	if organizationID == uuid.Nil || !ValidListPagination(options.Page, options.PageSize) {
		return nil, ErrAdminInvalidArgument
	}
	options.Keyword = strings.TrimSpace(options.Keyword)
	return uc.repo.ListUsers(ctx, organizationID, options)
}
func (uc *AdminUsecase) CreateUser(ctx context.Context, organizationID, actorID uuid.UUID, input *AdminUser, plainPassword string, roleIDs []uuid.UUID) (*AdminUser, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	if err := uc.validateRolesPrivilege(ctx, organizationID, actorID, organizationID, roleIDs); err != nil {
		return nil, err
	}
	normalized, err := normalizeUser(input)
	if err != nil || len(strings.TrimSpace(plainPassword)) < 12 {
		return nil, ErrAdminInvalidArgument
	}
	hash, err := password.Hash(plainPassword)
	if err != nil {
		return nil, fmt.Errorf("hash admin password: %w", err)
	}
	return uc.repo.CreateUser(ctx, organizationID, normalized, hash, roleIDs, adminAuditEvent(ctx, actorID, nil, "admin.user.create", normalized.Username))
}

func (uc *AdminUsecase) UpdateUser(ctx context.Context, organizationID, actorID, id uuid.UUID, input *AdminUser, roleIDs []uuid.UUID) (*AdminUser, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	if err := uc.validateRolesPrivilege(ctx, organizationID, actorID, organizationID, roleIDs); err != nil {
		return nil, err
	}
	normalized, err := normalizeUser(input)
	if err != nil {
		return nil, err
	}
	return uc.repo.UpdateUser(ctx, organizationID, id, normalized, roleIDs, adminAuditEvent(ctx, actorID, &id, "admin.user.update", ""))
}
func (uc *AdminUsecase) TerminateUser(ctx context.Context, organizationID, actorID, id uuid.UUID) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil {
		return ErrAdminInvalidArgument
	}
	if actorID == id {
		return ErrAdminUserSelfDelete
	}
	return uc.repo.TerminateUser(ctx, organizationID, id, adminAuditEvent(ctx, actorID, &id, "admin.user.terminate", ""))
}
func (uc *AdminUsecase) AuthorizeWeComUser(ctx context.Context, sourceOrganizationID, targetOrganizationID, actorID uuid.UUID, input *AdminUser, roleIDs []uuid.UUID) (*AdminUser, error) {
	if sourceOrganizationID == uuid.Nil || targetOrganizationID == uuid.Nil || actorID == uuid.Nil || len(roleIDs) == 0 {
		return nil, ErrAdminInvalidArgument
	}
	if err := uc.validateRolesPrivilege(ctx, sourceOrganizationID, actorID, targetOrganizationID, roleIDs); err != nil {
		return nil, err
	}
	normalized, err := normalizeUser(input)
	if err != nil {
		return nil, err
	}
	return uc.repo.AuthorizeWeComUser(ctx, sourceOrganizationID, targetOrganizationID, normalized, roleIDs, adminAuditEvent(ctx, actorID, &normalized.ID, "admin.user.wecom.authorize", ""))
}

func (uc *AdminUsecase) AuthorizeDingTalkUser(ctx context.Context, sourceOrganizationID, targetOrganizationID, actorID uuid.UUID, input *AdminUser, roleIDs []uuid.UUID) (*AdminUser, error) {
	if sourceOrganizationID == uuid.Nil || targetOrganizationID == uuid.Nil || actorID == uuid.Nil || len(roleIDs) == 0 {
		return nil, ErrAdminInvalidArgument
	}
	if err := uc.validateRolesPrivilege(ctx, sourceOrganizationID, actorID, targetOrganizationID, roleIDs); err != nil {
		return nil, err
	}
	normalized, err := normalizeUser(input)
	if err != nil {
		return nil, err
	}
	notification := NewDingTalkUserAuthorizedNotification(normalized.ID)
	return uc.repo.AuthorizeDingTalkUser(ctx, sourceOrganizationID, targetOrganizationID, normalized, roleIDs, notification, adminAuditEvent(ctx, actorID, &normalized.ID, "admin.user.dingtalk.authorize", ""))
}
func (uc *AdminUsecase) ResetUserPassword(ctx context.Context, organizationID, actorID, id uuid.UUID, plainPassword string, username *string) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil || len(strings.TrimSpace(plainPassword)) < 12 {
		return ErrAdminInvalidArgument
	}
	var normalizedUsername *string
	if username != nil {
		trimmed := strings.ToLower(strings.TrimSpace(*username))
		if !validUsername(trimmed) {
			return ErrAdminInvalidArgument
		}
		normalizedUsername = &trimmed
	}
	hash, err := password.Hash(plainPassword)
	if err != nil {
		return fmt.Errorf("hash reset password: %w", err)
	}
	return uc.repo.ResetUserPassword(ctx, organizationID, id, hash, normalizedUsername, adminAuditEvent(ctx, actorID, &id, "admin.user.password.reset", ""))
}

var usernamePattern = regexp.MustCompile(`^[a-z0-9_.-]+$`)

func validUsername(value string) bool {
	return len(value) >= 3 && len(value) <= 64 && usernamePattern.MatchString(value)
}
func normalizeUser(input *AdminUser) (*AdminUser, error) {
	if input == nil || (input.ID == uuid.Nil && strings.TrimSpace(input.Username) == "") {
		return nil, ErrAdminInvalidArgument
	}
	output := *input
	output.Username = strings.ToLower(strings.TrimSpace(output.Username))
	output.DisplayName = strings.TrimSpace(output.DisplayName)
	if output.DisplayName == "" || (output.ID == uuid.Nil && !validUsername(output.Username)) {
		return nil, ErrAdminInvalidArgument
	}
	if output.Email != nil {
		value := strings.TrimSpace(*output.Email)
		output.Email = &value
	}
	return &output, nil
}
