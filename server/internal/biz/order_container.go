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
	ErrOrderContainerNotFound        = errors.NotFound("ORDER_CONTAINER_NOT_FOUND", "订单集装箱不存在")
	ErrOrderContainerExists          = errors.Conflict("ORDER_CONTAINER_EXISTS", "订单集装箱已存在")
	ErrOrderContainerInvalidArgument = errors.BadRequest("ORDER_CONTAINER_INVALID_ARGUMENT", "订单集装箱字段不合法")
	ErrOrderContainerSpecInvalid     = errors.BadRequest("ORDER_CONTAINER_SPEC_INVALID", "箱型必须是当前组织启用的箱型主数据")
	ErrOrderContainerShipmentType    = errors.BadRequest("ORDER_CONTAINER_SHIPMENT_TYPE_INVALID", "仅整箱订单可维护订单级集装箱，拼箱集装箱应由自拼批次维护")
)

type OrderContainer struct {
	ID                 uuid.UUID
	OrderID            uuid.UUID
	ContainerNo        string
	ContainerSpecID    uuid.UUID
	ShippingDocumentID *uuid.UUID
	SealNo             *string
	GrossWeightKg      float64
	VolumeCbm          float64
	Note               *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type OrderContainerRepo interface {
	List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderContainer, error)
	Add(ctx context.Context, organizationID, orderID uuid.UUID, input *OrderContainer, audit *AuditEvent) (*OrderContainer, error)
	Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *OrderContainer, audit *AuditEvent) (*OrderContainer, error)
	Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, audit *AuditEvent) error
}

type OrderContainerUsecase struct {
	repo OrderContainerRepo
}

func NewOrderContainerUsecase(repo OrderContainerRepo) *OrderContainerUsecase {
	return &OrderContainerUsecase{repo: repo}
}

func (uc *OrderContainerUsecase) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderContainer, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderContainerInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, orderID)
}

func (uc *OrderContainerUsecase) Add(ctx context.Context, organizationID, actorID, orderID uuid.UUID, input *OrderContainer) (*OrderContainer, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderContainerInvalidArgument
	}
	normalized, err := normalizeOrderContainer(input)
	if err != nil {
		return nil, err
	}
	normalized.ID = uuid.Must(uuid.NewV7())
	return uc.repo.Add(ctx, organizationID, orderID, normalized, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.container.add",
		Result:         "success",
		Details: map[string]string{
			"container.id":      normalized.ID.String(),
			"order.id":          orderID.String(),
			"container.no":      normalized.ContainerNo,
			"container.spec_id": normalized.ContainerSpecID.String(),
		},
	})
}

func (uc *OrderContainerUsecase) Update(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID, input *OrderContainer) (*OrderContainer, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return nil, ErrOrderContainerInvalidArgument
	}
	normalized, err := normalizeOrderContainer(input)
	if err != nil {
		return nil, err
	}
	return uc.repo.Update(ctx, organizationID, orderID, id, normalized, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.container.update",
		Result:         "success",
		Details: map[string]string{
			"container.id":      id.String(),
			"order.id":          orderID.String(),
			"container.no":      normalized.ContainerNo,
			"container.spec_id": normalized.ContainerSpecID.String(),
		},
	})
}

func (uc *OrderContainerUsecase) Remove(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return ErrOrderContainerInvalidArgument
	}
	return uc.repo.Remove(ctx, organizationID, orderID, id, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.container.remove",
		Result:         "success",
		Details: map[string]string{
			"container.id": id.String(),
			"order.id":     orderID.String(),
		},
	})
}

func normalizeOrderContainer(input *OrderContainer) (*OrderContainer, error) {
	if input == nil {
		return nil, ErrOrderContainerInvalidArgument
	}
	containerNo := strings.TrimSpace(input.ContainerNo)
	if containerNo == "" || utf8.RuneCountInString(containerNo) > 64 {
		return nil, ErrOrderContainerInvalidArgument
	}
	if input.ContainerSpecID == uuid.Nil {
		return nil, ErrOrderContainerInvalidArgument
	}
	if input.GrossWeightKg <= 0 || input.VolumeCbm <= 0 {
		return nil, ErrOrderContainerInvalidArgument
	}
	var sealNo *string
	if input.SealNo != nil {
		v := strings.TrimSpace(*input.SealNo)
		if v != "" {
			if utf8.RuneCountInString(v) > 64 {
				return nil, ErrOrderContainerInvalidArgument
			}
			sealNo = &v
		}
	}
	var note *string
	if input.Note != nil {
		v := strings.TrimSpace(*input.Note)
		if v != "" {
			if utf8.RuneCountInString(v) > 500 {
				return nil, ErrOrderContainerInvalidArgument
			}
			note = &v
		}
	}
	output := *input
	output.ContainerNo = containerNo
	output.SealNo = sealNo
	output.Note = note
	return &output, nil
}
