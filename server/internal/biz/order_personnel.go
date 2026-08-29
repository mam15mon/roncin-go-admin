package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrOrderPersonnelNotFound        = errors.NotFound("ORDER_PERSONNEL_NOT_FOUND", "订单协作人员不存在")
	ErrOrderPersonnelExists          = errors.Conflict("ORDER_PERSONNEL_EXISTS", "订单协作人员已存在")
	ErrOrderPersonnelInvalidArgument = errors.BadRequest("ORDER_PERSONNEL_INVALID_ARGUMENT", "订单协作人员字段不合法")
	ErrOrderPersonnelUserInvalid     = errors.BadRequest("ORDER_PERSONNEL_USER_INVALID", "协作人员必须是订单组织范围内启用的组织成员")
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
	ID             uuid.UUID
	OrderID        uuid.UUID
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	Role           OrderPersonnelRole
	AssignedAt     time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Notification   *NotificationIntent
}

type OrderPersonnelRepo interface {
	List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderPersonnel, error)
	Assign(ctx context.Context, organizationID, orderID, userID, memberOrganizationID uuid.UUID, role OrderPersonnelRole, notification *NotificationIntent, audit *AuditEvent) (*OrderPersonnel, error)
	Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, audit *AuditEvent) error
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

func (uc *OrderPersonnelUsecase) Assign(ctx context.Context, organizationID, actorID, orderID, userID, memberOrganizationID uuid.UUID, role OrderPersonnelRole) (*OrderPersonnel, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || userID == uuid.Nil || memberOrganizationID == uuid.Nil {
		return nil, ErrOrderPersonnelInvalidArgument
	}
	if !role.Valid() {
		return nil, ErrOrderPersonnelInvalidArgument
	}
	var notification *NotificationIntent
	if role != OrderPersonnelRoleCreator {
		notification = NewOrderPersonnelNotification(userID)
	}
	audit := &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.personnel.assign",
		Result:         "success",
		Details: map[string]string{
			"order.id":        orderID.String(),
			"user.id":         userID.String(),
			"organization.id": memberOrganizationID.String(),
			"role":            string(role),
		},
	}
	created, err := uc.repo.Assign(ctx, organizationID, orderID, userID, memberOrganizationID, role, notification, audit)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (uc *OrderPersonnelUsecase) Remove(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return ErrOrderPersonnelInvalidArgument
	}
	audit := &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.personnel.remove",
		Result:         "success",
		Details: map[string]string{
			"personnel.id": id.String(),
			"order.id":     orderID.String(),
		},
	}
	if err := uc.repo.Remove(ctx, organizationID, orderID, id, audit); err != nil {
		return err
	}
	return nil
}
