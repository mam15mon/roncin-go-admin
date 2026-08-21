package biz

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
)

var (
	ErrOrderShippingDocumentNotFound        = errors.NotFound(v1.ErrorReason_ORDER_SHIPPING_DOCUMENT_NOT_FOUND.String(), "订单提单不存在")
	ErrOrderShippingDocumentExists          = errors.Conflict(v1.ErrorReason_ORDER_SHIPPING_DOCUMENT_EXISTS.String(), "订单提单已存在")
	ErrOrderShippingDocumentInvalidArgument = errors.BadRequest(v1.ErrorReason_ORDER_SHIPPING_DOCUMENT_INVALID_ARGUMENT.String(), "订单提单字段不合法")
	ErrOrderShippingDocumentInvalidStatus   = errors.BadRequest(v1.ErrorReason_ORDER_SHIPPING_DOCUMENT_INVALID_STATUS.String(), "提单当前状态不允许该操作")
	ErrOrderShippingDocumentStatusConflict  = errors.Conflict(v1.ErrorReason_ORDER_SHIPPING_DOCUMENT_STATUS_CONFLICT.String(), "提单状态已被并发修改")
)

type OrderShippingDocumentStatus string

const (
	OrderShippingDocumentStatusDraft     OrderShippingDocumentStatus = "DRAFT"
	OrderShippingDocumentStatusConfirmed OrderShippingDocumentStatus = "CONFIRMED"
	OrderShippingDocumentStatusReleased  OrderShippingDocumentStatus = "RELEASED"
)

func (s OrderShippingDocumentStatus) Valid() bool {
	switch s {
	case OrderShippingDocumentStatusDraft,
		OrderShippingDocumentStatusConfirmed,
		OrderShippingDocumentStatusReleased:
		return true
	default:
		return false
	}
}

func validShippingDocumentTransition(from, to OrderShippingDocumentStatus) bool {
	switch from {
	case OrderShippingDocumentStatusDraft:
		return to == OrderShippingDocumentStatusConfirmed
	case OrderShippingDocumentStatusConfirmed:
		return to == OrderShippingDocumentStatusReleased
	default:
		return false
	}
}

type OrderShippingDocument struct {
	ID          uuid.UUID
	OrderID     uuid.UUID
	MasterNo    string
	HouseNo     string
	ReleaseType *string
	Status      OrderShippingDocumentStatus
	Note        *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type OrderShippingDocumentRepo interface {
	List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderShippingDocument, error)
	Add(ctx context.Context, organizationID, orderID uuid.UUID, input *OrderShippingDocument) (*OrderShippingDocument, error)
	Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *OrderShippingDocument) (*OrderShippingDocument, error)
	Transition(ctx context.Context, organizationID, orderID, id uuid.UUID, from, to OrderShippingDocumentStatus) (*OrderShippingDocument, error)
	Remove(ctx context.Context, organizationID, orderID, id uuid.UUID) error
}

type OrderShippingDocumentUsecase struct {
	repo  OrderShippingDocumentRepo
	audit AuditRepo
}

func NewOrderShippingDocumentUsecase(repo OrderShippingDocumentRepo, audit AuditRepo) *OrderShippingDocumentUsecase {
	return &OrderShippingDocumentUsecase{repo: repo, audit: audit}
}

func (uc *OrderShippingDocumentUsecase) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderShippingDocument, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderShippingDocumentInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, orderID)
}

func (uc *OrderShippingDocumentUsecase) Add(ctx context.Context, organizationID, actorID, orderID uuid.UUID, input *OrderShippingDocument) (*OrderShippingDocument, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderShippingDocumentInvalidArgument
	}
	normalized, err := normalizeOrderShippingDocument(input)
	if err != nil {
		return nil, err
	}
	normalized.Status = OrderShippingDocumentStatusDraft
	created, err := uc.repo.Add(ctx, organizationID, orderID, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.shipping_document.add",
		Result:         "success",
		Details: map[string]string{
			"shipping_document.id": created.ID.String(),
			"order.id":             orderID.String(),
			"master_no":            created.MasterNo,
		},
	}); err != nil {
		return nil, fmt.Errorf("write order shipping document add audit: %w", err)
	}
	return created, nil
}

func (uc *OrderShippingDocumentUsecase) Update(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID, input *OrderShippingDocument) (*OrderShippingDocument, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return nil, ErrOrderShippingDocumentInvalidArgument
	}
	normalized, err := normalizeOrderShippingDocument(input)
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
		Action:         "order.shipping_document.update",
		Result:         "success",
		Details: map[string]string{
			"shipping_document.id": updated.ID.String(),
			"order.id":             orderID.String(),
			"master_no":            updated.MasterNo,
		},
	}); err != nil {
		return nil, fmt.Errorf("write order shipping document update audit: %w", err)
	}
	return updated, nil
}

func (uc *OrderShippingDocumentUsecase) Transition(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID, from, to OrderShippingDocumentStatus) (*OrderShippingDocument, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return nil, ErrOrderShippingDocumentInvalidArgument
	}
	if !from.Valid() || !to.Valid() || !validShippingDocumentTransition(from, to) {
		return nil, ErrOrderShippingDocumentInvalidStatus
	}
	transitioned, err := uc.repo.Transition(ctx, organizationID, orderID, id, from, to)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.shipping_document.transition",
		Result:         "success",
		Details: map[string]string{
			"shipping_document.id": transitioned.ID.String(),
			"order.id":             orderID.String(),
			"from_status":          string(from),
			"to_status":            string(to),
		},
	}); err != nil {
		return nil, fmt.Errorf("write order shipping document transition audit: %w", err)
	}
	return transitioned, nil
}

func (uc *OrderShippingDocumentUsecase) Remove(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return ErrOrderShippingDocumentInvalidArgument
	}
	if err := uc.repo.Remove(ctx, organizationID, orderID, id); err != nil {
		return err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.shipping_document.remove",
		Result:         "success",
		Details: map[string]string{
			"shipping_document.id": id.String(),
			"order.id":             orderID.String(),
		},
	}); err != nil {
		return fmt.Errorf("write order shipping document remove audit: %w", err)
	}
	return nil
}

func normalizeOrderShippingDocument(input *OrderShippingDocument) (*OrderShippingDocument, error) {
	if input == nil {
		return nil, ErrOrderShippingDocumentInvalidArgument
	}
	masterNo := strings.TrimSpace(input.MasterNo)
	if masterNo == "" || utf8.RuneCountInString(masterNo) > 64 {
		return nil, ErrOrderShippingDocumentInvalidArgument
	}
	houseNo := strings.TrimSpace(input.HouseNo)
	if houseNo == "" || utf8.RuneCountInString(houseNo) > 64 {
		return nil, ErrOrderShippingDocumentInvalidArgument
	}
	var releaseType *string
	if input.ReleaseType != nil {
		v := strings.TrimSpace(*input.ReleaseType)
		if v != "" {
			if utf8.RuneCountInString(v) > 64 {
				return nil, ErrOrderShippingDocumentInvalidArgument
			}
			releaseType = &v
		}
	}
	var note *string
	if input.Note != nil {
		v := strings.TrimSpace(*input.Note)
		if v != "" {
			if utf8.RuneCountInString(v) > 500 {
				return nil, ErrOrderShippingDocumentInvalidArgument
			}
			note = &v
		}
	}
	output := *input
	output.MasterNo = masterNo
	output.HouseNo = houseNo
	output.ReleaseType = releaseType
	output.Note = note
	return &output, nil
}
