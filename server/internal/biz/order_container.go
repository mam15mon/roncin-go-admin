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
	ErrOrderContainerNotFound        = errors.NotFound(v1.ErrorReason_ORDER_CONTAINER_NOT_FOUND.String(), "订单集装箱不存在")
	ErrOrderContainerExists          = errors.Conflict(v1.ErrorReason_ORDER_CONTAINER_EXISTS.String(), "订单集装箱已存在")
	ErrOrderContainerInvalidArgument = errors.BadRequest(v1.ErrorReason_ORDER_CONTAINER_INVALID_ARGUMENT.String(), "订单集装箱字段不合法")
	ErrOrderContainerSpecInvalid     = errors.BadRequest(v1.ErrorReason_ORDER_CONTAINER_SPEC_INVALID.String(), "箱型必须是当前组织启用的箱型主数据")
)

type OrderContainer struct {
	ID              uuid.UUID
	OrderID         uuid.UUID
	ContainerNo     string
	ContainerSpecID uuid.UUID
	SealNo          *string
	GrossWeightKg   float64
	VolumeCbm       float64
	Note            *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type OrderContainerRepo interface {
	List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderContainer, error)
	Add(ctx context.Context, organizationID, orderID uuid.UUID, input *OrderContainer) (*OrderContainer, error)
	Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *OrderContainer) (*OrderContainer, error)
	Remove(ctx context.Context, organizationID, orderID, id uuid.UUID) error
}

type OrderContainerUsecase struct {
	repo  OrderContainerRepo
	audit AuditRepo
}

func NewOrderContainerUsecase(repo OrderContainerRepo, audit AuditRepo) *OrderContainerUsecase {
	return &OrderContainerUsecase{repo: repo, audit: audit}
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
	created, err := uc.repo.Add(ctx, organizationID, orderID, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.container.add",
		Result:         "success",
		Details: map[string]string{
			"container.id":      created.ID.String(),
			"order.id":          orderID.String(),
			"container.no":      created.ContainerNo,
			"container.spec_id": created.ContainerSpecID.String(),
		},
	}); err != nil {
		return nil, fmt.Errorf("write order container add audit: %w", err)
	}
	return created, nil
}

func (uc *OrderContainerUsecase) Update(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID, input *OrderContainer) (*OrderContainer, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return nil, ErrOrderContainerInvalidArgument
	}
	normalized, err := normalizeOrderContainer(input)
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
		Action:         "order.container.update",
		Result:         "success",
		Details: map[string]string{
			"container.id":      updated.ID.String(),
			"order.id":          orderID.String(),
			"container.no":      updated.ContainerNo,
			"container.spec_id": updated.ContainerSpecID.String(),
		},
	}); err != nil {
		return nil, fmt.Errorf("write order container update audit: %w", err)
	}
	return updated, nil
}

func (uc *OrderContainerUsecase) Remove(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return ErrOrderContainerInvalidArgument
	}
	if err := uc.repo.Remove(ctx, organizationID, orderID, id); err != nil {
		return err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.container.remove",
		Result:         "success",
		Details: map[string]string{
			"container.id": id.String(),
			"order.id":     orderID.String(),
		},
	}); err != nil {
		return fmt.Errorf("write order container remove audit: %w", err)
	}
	return nil
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
