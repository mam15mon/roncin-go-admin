package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrOrderAbnormalCaseNotFound        = errors.NotFound("ORDER_ABNORMAL_CASE_NOT_FOUND", "订单异常标记不存在")
	ErrOrderAbnormalCaseExists          = errors.Conflict("ORDER_ABNORMAL_CASE_EXISTS", "该异常已在订单上标记且未解决")
	ErrOrderAbnormalCaseInvalidArgument = errors.BadRequest("ORDER_ABNORMAL_CASE_INVALID_ARGUMENT", "订单异常标记字段不合法")
	ErrOrderAbnormalCaseKindInvalid     = errors.BadRequest("ORDER_ABNORMAL_CASE_KIND_INVALID", "异常类型必须是当前组织启用的异常类型主数据")
	ErrOrderAbnormalCaseStatusConflict  = errors.Conflict("ORDER_ABNORMAL_CASE_STATUS_CONFLICT", "异常状态已被并发修改")
)

type OrderAbnormalCaseStatus string

const (
	OrderAbnormalCaseStatusActive   OrderAbnormalCaseStatus = "ACTIVE"
	OrderAbnormalCaseStatusResolved OrderAbnormalCaseStatus = "RESOLVED"
)

func (s OrderAbnormalCaseStatus) Valid() bool {
	switch s {
	case OrderAbnormalCaseStatusActive, OrderAbnormalCaseStatusResolved:
		return true
	default:
		return false
	}
}

type OrderAbnormalCase struct {
	ID             uuid.UUID
	OrderID        uuid.UUID
	AbnormalCaseID uuid.UUID
	Status         OrderAbnormalCaseStatus
	MarkedAt       time.Time
	MarkedBy       uuid.UUID
	ResolvedAt     *time.Time
	ResolvedBy     *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type OrderAbnormalCaseRepo interface {
	List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderAbnormalCase, error)
	Mark(ctx context.Context, organizationID, orderID, actorID, abnormalCaseID uuid.UUID, newID uuid.UUID, audit *AuditEvent) (*OrderAbnormalCase, error)
	Resolve(ctx context.Context, organizationID, orderID, actorID, id uuid.UUID, audit *AuditEvent) (*OrderAbnormalCase, error)
	Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, audit *AuditEvent) error
}

type OrderAbnormalCaseUsecase struct {
	repo OrderAbnormalCaseRepo
}

func NewOrderAbnormalCaseUsecase(repo OrderAbnormalCaseRepo) *OrderAbnormalCaseUsecase {
	return &OrderAbnormalCaseUsecase{repo: repo}
}

func (uc *OrderAbnormalCaseUsecase) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderAbnormalCase, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderAbnormalCaseInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, orderID)
}

func (uc *OrderAbnormalCaseUsecase) Mark(ctx context.Context, organizationID, actorID, orderID, abnormalCaseID uuid.UUID) (*OrderAbnormalCase, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || abnormalCaseID == uuid.Nil {
		return nil, ErrOrderAbnormalCaseInvalidArgument
	}
	newID := uuid.Must(uuid.NewV7())
	return uc.repo.Mark(ctx, organizationID, orderID, actorID, abnormalCaseID, newID, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.abnormal_case.mark",
		Result:         "success",
		Details: map[string]string{
			"order.id":         orderID.String(),
			"abnormal_case.id": abnormalCaseID.String(),
			"status":           string(OrderAbnormalCaseStatusActive),
		},
	})
}

func (uc *OrderAbnormalCaseUsecase) Resolve(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID) (*OrderAbnormalCase, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return nil, ErrOrderAbnormalCaseInvalidArgument
	}
	return uc.repo.Resolve(ctx, organizationID, orderID, actorID, id, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.abnormal_case.resolve",
		Result:         "success",
		Details: map[string]string{
			"abnormal_case_record.id": id.String(),
			"order.id":                orderID.String(),
		},
	})
}

func (uc *OrderAbnormalCaseUsecase) Remove(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return ErrOrderAbnormalCaseInvalidArgument
	}
	return uc.repo.Remove(ctx, organizationID, orderID, id, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.abnormal_case.remove",
		Result:         "success",
		Details: map[string]string{
			"abnormal_case_record.id": id.String(),
			"order.id":                orderID.String(),
		},
	})
}
