package biz

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrOrderShippingDocumentNotFound        = errors.NotFound("ORDER_SHIPPING_DOCUMENT_NOT_FOUND", "订单提单不存在")
	ErrOrderShippingDocumentExists          = errors.Conflict("ORDER_SHIPPING_DOCUMENT_EXISTS", "订单提单已存在")
	ErrOrderShippingDocumentInvalidArgument = errors.BadRequest("ORDER_SHIPPING_DOCUMENT_INVALID_ARGUMENT", "订单提单字段不合法")
	ErrOrderShippingDocumentInvalidStatus   = errors.BadRequest("ORDER_SHIPPING_DOCUMENT_INVALID_STATUS", "提单当前状态不允许该操作")
	ErrOrderShippingDocumentStatusConflict  = errors.Conflict("ORDER_SHIPPING_DOCUMENT_STATUS_CONFLICT", "提单状态已被并发修改")
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
	HouseNo     string
	ReleaseType *string
	Status      OrderShippingDocumentStatus
	Note        *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type OrderShippingDocumentRepo interface {
	List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderShippingDocument, error)
	Add(ctx context.Context, organizationID, orderID uuid.UUID, input *OrderShippingDocument, audit *AuditEvent) (*OrderShippingDocument, error)
	Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *OrderShippingDocument, audit *AuditEvent) (*OrderShippingDocument, error)
	Transition(ctx context.Context, organizationID, orderID, id uuid.UUID, from, to OrderShippingDocumentStatus, audit *AuditEvent) (*OrderShippingDocument, error)
	Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, audit *AuditEvent) error
}

type OrderShippingDocumentUsecase struct {
	repo OrderShippingDocumentRepo
}

func NewOrderShippingDocumentUsecase(repo OrderShippingDocumentRepo) *OrderShippingDocumentUsecase {
	return &OrderShippingDocumentUsecase{repo: repo}
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
	normalized.ID = uuid.Must(uuid.NewV7())
	normalized.Status = OrderShippingDocumentStatusDraft
	return uc.repo.Add(ctx, organizationID, orderID, normalized, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.shipping_document.add",
		Result:         "success",
		Details: map[string]string{
			"shipping_document.id": normalized.ID.String(),
			"order.id":             orderID.String(),
			"house_no":             normalized.HouseNo,
		},
	})
}

func (uc *OrderShippingDocumentUsecase) Update(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID, input *OrderShippingDocument) (*OrderShippingDocument, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return nil, ErrOrderShippingDocumentInvalidArgument
	}
	normalized, err := normalizeOrderShippingDocument(input)
	if err != nil {
		return nil, err
	}
	return uc.repo.Update(ctx, organizationID, orderID, id, normalized, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.shipping_document.update",
		Result:         "success",
		Details: map[string]string{
			"shipping_document.id": id.String(),
			"order.id":             orderID.String(),
			"house_no":             normalized.HouseNo,
		},
	})
}

func (uc *OrderShippingDocumentUsecase) Transition(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID, from, to OrderShippingDocumentStatus) (*OrderShippingDocument, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return nil, ErrOrderShippingDocumentInvalidArgument
	}
	if !from.Valid() || !to.Valid() || !validShippingDocumentTransition(from, to) {
		return nil, ErrOrderShippingDocumentInvalidStatus
	}
	return uc.repo.Transition(ctx, organizationID, orderID, id, from, to, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.shipping_document.transition",
		Result:         "success",
		Details: map[string]string{
			"shipping_document.id": id.String(),
			"order.id":             orderID.String(),
			"from_status":          string(from),
			"to_status":            string(to),
		},
	})
}

func (uc *OrderShippingDocumentUsecase) Remove(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return ErrOrderShippingDocumentInvalidArgument
	}
	return uc.repo.Remove(ctx, organizationID, orderID, id, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.shipping_document.remove",
		Result:         "success",
		Details: map[string]string{
			"shipping_document.id": id.String(),
			"order.id":             orderID.String(),
		},
	})
}

func normalizeOrderShippingDocument(input *OrderShippingDocument) (*OrderShippingDocument, error) {
	if input == nil {
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
	return &OrderShippingDocument{
		ID:          input.ID,
		OrderID:     input.OrderID,
		HouseNo:     houseNo,
		ReleaseType: releaseType,
		Status:      input.Status,
		Note:        note,
		CreatedAt:   input.CreatedAt,
		UpdatedAt:   input.UpdatedAt,
	}, nil
}
