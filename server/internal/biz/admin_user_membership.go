package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrAdminUserMembershipNotFound = errors.NotFound("ADMIN_USER_MEMBERSHIP_NOT_FOUND", "用户组织成员关系不存在")
	ErrAdminUserMembershipExists   = errors.Conflict("ADMIN_USER_MEMBERSHIP_EXISTS", "用户已属于该组织")
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

func (uc *AdminUsecase) ListUserMemberships(ctx context.Context, userID uuid.UUID) ([]*AdminUserMembership, error) {
	if userID == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	return uc.repo.ListUserMemberships(ctx, userID)
}

func (uc *AdminUsecase) CreateUserMembership(ctx context.Context, actorOrganizationID, actorID, userID, organizationID uuid.UUID, primary bool, roleIDs []uuid.UUID) (*AdminUserMembership, error) {
	if actorOrganizationID == uuid.Nil || actorID == uuid.Nil || userID == uuid.Nil || organizationID == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	if err := uc.validateRolesPrivilege(ctx, actorOrganizationID, actorID, organizationID, roleIDs); err != nil {
		return nil, err
	}
	created, err := uc.repo.CreateUserMembership(ctx, &AdminUserMembership{
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

func (uc *AdminUsecase) UpdateUserMembership(ctx context.Context, actorOrganizationID, actorID, userID, membershipID uuid.UUID, enabled, primary bool, roleIDs []uuid.UUID) (*AdminUserMembership, error) {
	if actorOrganizationID == uuid.Nil || actorID == uuid.Nil || userID == uuid.Nil || membershipID == uuid.Nil || (primary && !enabled) {
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
