package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrAdminUserMembershipNotFound = errors.NotFound("ADMIN_USER_MEMBERSHIP_NOT_FOUND", "用户组织成员关系不存在")
	ErrAdminUserMembershipExists   = errors.Conflict("ADMIN_USER_MEMBERSHIP_EXISTS", "成员关系已存在（含停用记录），请通过已有成员关系编辑角色或恢复状态")
	ErrAdminUserInactive           = errors.Conflict("ADMIN_USER_INACTIVE", "用户账号已停用，请先通过开通流程恢复")
)

type AdminUserMembership struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	OrganizationID   uuid.UUID
	OrganizationCode string
	OrganizationName string
	OrganizationKind OrganizationKind
	Enabled          bool
	RoleIDs          []uuid.UUID
	RoleCodes        []string
	RoleNames        []string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (uc *AdminUsecase) ListUserMemberships(ctx context.Context, userID uuid.UUID) ([]*AdminUserMembership, error) {
	if userID == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	return uc.repo.ListUserMemberships(ctx, userID)
}

// AdminUserMembershipCreation 表示一个组织及其独立的选填角色。
type AdminUserMembershipCreation struct {
	OrganizationID uuid.UUID
	RoleIDs        []uuid.UUID
}

func (uc *AdminUsecase) CreateUserMemberships(ctx context.Context, actorOrganizationID, actorID, userID uuid.UUID, items []AdminUserMembershipCreation) ([]*AdminUserMembership, error) {
	if actorOrganizationID == uuid.Nil || actorID == uuid.Nil || userID == uuid.Nil || len(items) == 0 {
		return nil, ErrAdminInvalidArgument
	}
	seen := make(map[uuid.UUID]struct{}, len(items))
	audits := make([]*AuditEvent, 0, len(items))
	for _, item := range items {
		if item.OrganizationID == uuid.Nil {
			return nil, ErrAdminInvalidArgument
		}
		if _, exists := seen[item.OrganizationID]; exists {
			return nil, ErrAdminInvalidArgument
		}
		seen[item.OrganizationID] = struct{}{}
		for _, roleID := range item.RoleIDs {
			if roleID == uuid.Nil {
				return nil, ErrAdminInvalidArgument
			}
		}
		audits = append(audits, adminAuditEvent(ctx, actorID, &userID, "admin.user.membership.create", item.OrganizationID.String()))
	}
	var ids []uuid.UUID
	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		var err error
		ids, err = uc.repo.CreateUserMemberships(txCtx, actorOrganizationID, actorID, userID, items, audits)
		return err
	})
	if err != nil {
		return nil, err
	}
	created := make([]*AdminUserMembership, 0, len(ids))
	for _, id := range ids {
		item, err := uc.repo.GetUserMembership(ctx, userID, id)
		if err != nil {
			return nil, err
		}
		created = append(created, item)
	}
	return created, nil
}

// ValidateUserMembershipRolesPrivilege 供仓储取得授权依据的锁后复用领域提权规则。
func (uc *AdminUsecase) ValidateUserMembershipRolesPrivilege(ctx context.Context, actorOrganizationID, actorID, targetOrganizationID uuid.UUID, roleIDs []uuid.UUID) error {
	return uc.validateRolesPrivilege(ctx, actorOrganizationID, actorID, targetOrganizationID, roleIDs)
}

func (uc *AdminUsecase) UpdateUserMembership(ctx context.Context, actorOrganizationID, actorID, userID, membershipID uuid.UUID, enabled bool, roleIDs []uuid.UUID) (*AdminUserMembership, error) {
	if actorOrganizationID == uuid.Nil || actorID == uuid.Nil || userID == uuid.Nil || membershipID == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	if actorID == userID && !enabled {
		return nil, ErrAdminUserSelfDelete
	}
	currentMembership, err := uc.repo.GetUserMembership(ctx, userID, membershipID)
	if err != nil {
		return nil, err
	}
	if err := uc.validateRolesPrivilege(ctx, actorOrganizationID, actorID, currentMembership.OrganizationID, roleIDs); err != nil {
		return nil, err
	}
	updated, err := uc.repo.UpdateUserMembership(ctx, &AdminUserMembership{
		ID:      membershipID,
		UserID:  userID,
		Enabled: enabled,
	}, roleIDs, adminAuditEvent(ctx, actorID, &userID, "admin.user.membership.update", currentMembership.OrganizationID.String()))
	return updated, err
}

func (uc *AdminUsecase) DeleteUserMembership(ctx context.Context, actorID, userID, membershipID uuid.UUID) error {
	if actorID == uuid.Nil || userID == uuid.Nil || membershipID == uuid.Nil {
		return ErrAdminInvalidArgument
	}
	if actorID == userID {
		return ErrAdminUserSelfDelete
	}
	return uc.repo.DeleteUserMembership(ctx, userID, membershipID, adminAuditEvent(ctx, actorID, &userID, "admin.user.membership.delete", membershipID.String()))
}

// AdminUserMembershipItemError 只为已通过管理范围校验的组织补充可见名称。
func AdminUserMembershipItemError(organizationName string, err error) error {
	itemError := errors.FromError(err)
	if itemError.Code >= 500 {
		return err
	}
	return errors.New(int(itemError.Code), itemError.Reason, fmt.Sprintf("%s：%s", organizationName, itemError.Message))
}
