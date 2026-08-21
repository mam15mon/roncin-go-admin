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
	ErrOrderAbnormalCaseNotFound        = errors.NotFound(v1.ErrorReason_ORDER_ABNORMAL_CASE_NOT_FOUND.String(), "订单异常标记不存在")
	ErrOrderAbnormalCaseExists          = errors.Conflict(v1.ErrorReason_ORDER_ABNORMAL_CASE_EXISTS.String(), "该异常已在订单上标记且未解决")
	ErrOrderAbnormalCaseInvalidArgument = errors.BadRequest(v1.ErrorReason_ORDER_ABNORMAL_CASE_INVALID_ARGUMENT.String(), "订单异常标记字段不合法")
	ErrOrderAbnormalCaseKindInvalid     = errors.BadRequest(v1.ErrorReason_ORDER_ABNORMAL_CASE_KIND_INVALID.String(), "异常类型必须是当前组织启用的异常类型主数据")
	ErrOrderAbnormalCaseStatusConflict  = errors.Conflict(v1.ErrorReason_ORDER_ABNORMAL_CASE_STATUS_CONFLICT.String(), "异常状态已被并发修改")
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
	Mark(ctx context.Context, organizationID, orderID, actorID, abnormalCaseID uuid.UUID) (*OrderAbnormalCase, error)
	Resolve(ctx context.Context, organizationID, orderID, actorID, id uuid.UUID) (*OrderAbnormalCase, error)
	Remove(ctx context.Context, organizationID, orderID, id uuid.UUID) error
}

type OrderAbnormalCaseUsecase struct {
	repo  OrderAbnormalCaseRepo
	audit AuditRepo
}

func NewOrderAbnormalCaseUsecase(repo OrderAbnormalCaseRepo, audit AuditRepo) *OrderAbnormalCaseUsecase {
	return &OrderAbnormalCaseUsecase{repo: repo, audit: audit}
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
	marked, err := uc.repo.Mark(ctx, organizationID, orderID, actorID, abnormalCaseID)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.abnormal_case.mark",
		Result:         "success",
		Details: map[string]string{
			"abnormal_case_record.id": marked.ID.String(),
			"order.id":                orderID.String(),
			"abnormal_case.id":        abnormalCaseID.String(),
			"status":                  string(marked.Status),
		},
	}); err != nil {
		return nil, fmt.Errorf("write order abnormal case mark audit: %w", err)
	}
	return marked, nil
}

func (uc *OrderAbnormalCaseUsecase) Resolve(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID) (*OrderAbnormalCase, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return nil, ErrOrderAbnormalCaseInvalidArgument
	}
	resolved, err := uc.repo.Resolve(ctx, organizationID, orderID, actorID, id)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.abnormal_case.resolve",
		Result:         "success",
		Details: map[string]string{
			"abnormal_case_record.id": resolved.ID.String(),
			"order.id":                orderID.String(),
		},
	}); err != nil {
		return nil, fmt.Errorf("write order abnormal case resolve audit: %w", err)
	}
	return resolved, nil
}

func (uc *OrderAbnormalCaseUsecase) Remove(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return ErrOrderAbnormalCaseInvalidArgument
	}
	if err := uc.repo.Remove(ctx, organizationID, orderID, id); err != nil {
		return err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.abnormal_case.remove",
		Result:         "success",
		Details: map[string]string{
			"abnormal_case_record.id": id.String(),
			"order.id":                orderID.String(),
		},
	}); err != nil {
		return fmt.Errorf("write order abnormal case remove audit: %w", err)
	}
	return nil
}
