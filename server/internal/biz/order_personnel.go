package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
)

var (
	ErrOrderPersonnelNotFound        = errors.NotFound(v1.ErrorReason_ORDER_PERSONNEL_NOT_FOUND.String(), "订单协作人员不存在")
	ErrOrderPersonnelExists          = errors.Conflict(v1.ErrorReason_ORDER_PERSONNEL_EXISTS.String(), "订单协作人员已存在")
	ErrOrderPersonnelInvalidArgument = errors.BadRequest(v1.ErrorReason_ORDER_PERSONNEL_INVALID_ARGUMENT.String(), "订单协作人员字段不合法")
	ErrOrderPersonnelUserInvalid     = errors.BadRequest(v1.ErrorReason_ORDER_PERSONNEL_USER_INVALID.String(), "协作人员必须是当前组织启用的用户")
)

type OrderPersonnelRole string

const (
	OrderPersonnelRoleCreator         OrderPersonnelRole = "CREATOR"
	OrderPersonnelRoleOperator        OrderPersonnelRole = "OPERATOR"
	OrderPersonnelRoleSales           OrderPersonnelRole = "SALES"
	OrderPersonnelRoleCustomerService OrderPersonnelRole = "CUSTOMER_SERVICE"
	OrderPersonnelRoleDocument        OrderPersonnelRole = "DOCUMENT"
	OrderPersonnelRoleCommercial      OrderPersonnelRole = "COMMERCIAL"
	OrderPersonnelRoleAssociate       OrderPersonnelRole = "ASSOCIATE"
	OrderPersonnelRoleAssociate2      OrderPersonnelRole = "ASSOCIATE2"
)

func (r OrderPersonnelRole) Valid() bool {
	switch r {
	case OrderPersonnelRoleCreator,
		OrderPersonnelRoleOperator,
		OrderPersonnelRoleSales,
		OrderPersonnelRoleCustomerService,
		OrderPersonnelRoleDocument,
		OrderPersonnelRoleCommercial,
		OrderPersonnelRoleAssociate,
		OrderPersonnelRoleAssociate2:
		return true
	default:
		return false
	}
}

type OrderPersonnel struct {
	ID         uuid.UUID
	OrderID    uuid.UUID
	UserID     uuid.UUID
	Role       OrderPersonnelRole
	AssignedAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type OrderPersonnelRepo interface {
	List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderPersonnel, error)
	Assign(ctx context.Context, organizationID, orderID, userID uuid.UUID, role OrderPersonnelRole) (*OrderPersonnel, error)
	Remove(ctx context.Context, organizationID, orderID, id uuid.UUID) error
}

type OrderPersonnelUsecase struct {
	repo  OrderPersonnelRepo
	audit AuditRepo
}

func NewOrderPersonnelUsecase(repo OrderPersonnelRepo, audit AuditRepo) *OrderPersonnelUsecase {
	return &OrderPersonnelUsecase{repo: repo, audit: audit}
}

func (uc *OrderPersonnelUsecase) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderPersonnel, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderPersonnelInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, orderID)
}

func (uc *OrderPersonnelUsecase) Assign(ctx context.Context, organizationID, actorID, orderID, userID uuid.UUID, role OrderPersonnelRole) (*OrderPersonnel, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || userID == uuid.Nil {
		return nil, ErrOrderPersonnelInvalidArgument
	}
	if !role.Valid() {
		return nil, ErrOrderPersonnelInvalidArgument
	}
	created, err := uc.repo.Assign(ctx, organizationID, orderID, userID, role)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.personnel.assign",
		Result:         "success",
		Details: map[string]string{
			"personnel.id": created.ID.String(),
			"order.id":     orderID.String(),
			"user.id":      userID.String(),
			"role":         string(role),
		},
	}); err != nil {
		return nil, fmt.Errorf("write order personnel assign audit: %w", err)
	}
	return created, nil
}

func (uc *OrderPersonnelUsecase) Remove(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return ErrOrderPersonnelInvalidArgument
	}
	if err := uc.repo.Remove(ctx, organizationID, orderID, id); err != nil {
		return err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.personnel.remove",
		Result:         "success",
		Details: map[string]string{
			"personnel.id": id.String(),
			"order.id":     orderID.String(),
		},
	}); err != nil {
		return fmt.Errorf("write order personnel remove audit: %w", err)
	}
	return nil
}
