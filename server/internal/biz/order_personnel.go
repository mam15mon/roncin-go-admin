package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrOrderPersonnelNotFound        = errors.NotFound("ORDER_PERSONNEL_NOT_FOUND", "订单协作人员不存在")
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
}

type OrderPersonnelUsecase struct {
	repo OrderPersonnelRepo
}

func NewOrderPersonnelUsecase(repo OrderPersonnelRepo) *OrderPersonnelUsecase {
	return &OrderPersonnelUsecase{repo: repo}
}

func (uc *OrderPersonnelUsecase) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderPersonnel, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderPersonnelInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, orderID)
}
