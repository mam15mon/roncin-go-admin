package biz

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrOrderReleasePodNotFound        = errors.NotFound("ORDER_RELEASE_POD_NOT_FOUND", "放货凭证不存在")
	ErrOrderReleasePodInvalidArgument = errors.BadRequest("ORDER_RELEASE_POD_INVALID_ARGUMENT", "放货凭证字段不合法")
	ErrOrderReleasePodInvalidStatus   = errors.BadRequest("ORDER_RELEASE_POD_INVALID_STATUS", "放货凭证当前状态不允许该操作")
	ErrOrderReleasePodStatusConflict  = errors.Conflict("ORDER_RELEASE_POD_STATUS_CONFLICT", "放货凭证状态已被并发修改")
	ErrOrderReleasePodDocumentInvalid = errors.BadRequest("ORDER_RELEASE_POD_DOCUMENT_INVALID", "关联提单必须是同一订单下的提单")
)

type OrderReleasePodStatus string

const (
	OrderReleasePodStatusPending  OrderReleasePodStatus = "PENDING"
	OrderReleasePodStatusSigned   OrderReleasePodStatus = "SIGNED"
	OrderReleasePodStatusReturned OrderReleasePodStatus = "RETURNED"
)

func (s OrderReleasePodStatus) Valid() bool {
	switch s {
	case OrderReleasePodStatusPending,
		OrderReleasePodStatusSigned,
		OrderReleasePodStatusReturned:
		return true
	default:
		return false
	}
}

func validReleasePodTransition(from, to OrderReleasePodStatus) bool {
	switch from {
	case OrderReleasePodStatusPending:
		return to == OrderReleasePodStatusSigned
	case OrderReleasePodStatusSigned:
		return to == OrderReleasePodStatusReturned
	default:
		return false
	}
}

type OrderReleasePod struct {
	ID                 uuid.UUID
	OrderID            uuid.UUID
	ShippingDocumentID *uuid.UUID
	ReleaseNo          *string
	PodNo              *string
	Status             OrderReleasePodStatus
	SignedAt           *time.Time
	SignedBy           *uuid.UUID
	Note               *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type OrderReleasePodRepo interface {
	List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderReleasePod, error)
	Add(ctx context.Context, organizationID, orderID uuid.UUID, input *OrderReleasePod) (*OrderReleasePod, error)
	Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *OrderReleasePod) (*OrderReleasePod, error)
	Transition(ctx context.Context, organizationID, orderID, id uuid.UUID, from, to OrderReleasePodStatus, actorID uuid.UUID) (*OrderReleasePod, error)
	Remove(ctx context.Context, organizationID, orderID, id uuid.UUID) error
}

type OrderReleasePodUsecase struct {
	repo  OrderReleasePodRepo
	audit AuditRepo
}

func NewOrderReleasePodUsecase(repo OrderReleasePodRepo, audit AuditRepo) *OrderReleasePodUsecase {
	return &OrderReleasePodUsecase{repo: repo, audit: audit}
}

func (uc *OrderReleasePodUsecase) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderReleasePod, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderReleasePodInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, orderID)
}

func (uc *OrderReleasePodUsecase) Add(ctx context.Context, organizationID, actorID, orderID uuid.UUID, input *OrderReleasePod) (*OrderReleasePod, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderReleasePodInvalidArgument
	}
	normalized, err := normalizeOrderReleasePod(input)
	if err != nil {
		return nil, err
	}
	normalized.Status = OrderReleasePodStatusPending
	created, err := uc.repo.Add(ctx, organizationID, orderID, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.release_pod.add",
		Result:         "success",
		Details: map[string]string{
			"release_pod.id": created.ID.String(),
			"order.id":       orderID.String(),
		},
	}); err != nil {
		return nil, fmt.Errorf("write order release pod add audit: %w", err)
	}
	return created, nil
}

func (uc *OrderReleasePodUsecase) Update(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID, input *OrderReleasePod) (*OrderReleasePod, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return nil, ErrOrderReleasePodInvalidArgument
	}
	normalized, err := normalizeOrderReleasePod(input)
	if err != nil {
		return nil, err
	}
	updated, err := uc.repo.Update(ctx, organizationID, orderID, id, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.release_pod.update",
		Result:         "success",
		Details: map[string]string{
			"release_pod.id": updated.ID.String(),
			"order.id":       orderID.String(),
		},
	}); err != nil {
		return nil, fmt.Errorf("write order release pod update audit: %w", err)
	}
	return updated, nil
}

func (uc *OrderReleasePodUsecase) Transition(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID, from, to OrderReleasePodStatus) (*OrderReleasePod, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return nil, ErrOrderReleasePodInvalidArgument
	}
	if !from.Valid() || !to.Valid() || !validReleasePodTransition(from, to) {
		return nil, ErrOrderReleasePodInvalidStatus
	}
	transitioned, err := uc.repo.Transition(ctx, organizationID, orderID, id, from, to, actorID)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.release_pod.transition",
		Result:         "success",
		Details: map[string]string{
			"release_pod.id": transitioned.ID.String(),
			"order.id":       orderID.String(),
			"from_status":    string(from),
			"to_status":      string(to),
		},
	}); err != nil {
		return nil, fmt.Errorf("write order release pod transition audit: %w", err)
	}
	return transitioned, nil
}

func (uc *OrderReleasePodUsecase) Remove(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return ErrOrderReleasePodInvalidArgument
	}
	if err := uc.repo.Remove(ctx, organizationID, orderID, id); err != nil {
		return err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.release_pod.remove",
		Result:         "success",
		Details: map[string]string{
			"release_pod.id": id.String(),
			"order.id":       orderID.String(),
		},
	}); err != nil {
		return fmt.Errorf("write order release pod remove audit: %w", err)
	}
	return nil
}

func normalizeOrderReleasePod(input *OrderReleasePod) (*OrderReleasePod, error) {
	if input == nil {
		return nil, ErrOrderReleasePodInvalidArgument
	}
	var releaseNo *string
	if input.ReleaseNo != nil {
		v := strings.TrimSpace(*input.ReleaseNo)
		if v != "" {
			if utf8.RuneCountInString(v) > 64 {
				return nil, ErrOrderReleasePodInvalidArgument
			}
			releaseNo = &v
		}
	}
	var podNo *string
	if input.PodNo != nil {
		v := strings.TrimSpace(*input.PodNo)
		if v != "" {
			if utf8.RuneCountInString(v) > 64 {
				return nil, ErrOrderReleasePodInvalidArgument
			}
			podNo = &v
		}
	}
	var note *string
	if input.Note != nil {
		v := strings.TrimSpace(*input.Note)
		if v != "" {
			if utf8.RuneCountInString(v) > 500 {
				return nil, ErrOrderReleasePodInvalidArgument
			}
			note = &v
		}
	}
	output := *input
	output.ReleaseNo = releaseNo
	output.PodNo = podNo
	output.Note = note
	return &output, nil
}
