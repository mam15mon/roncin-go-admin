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
	ErrOrderCargoItemNotFound        = errors.NotFound("ORDER_CARGO_ITEM_NOT_FOUND", "订单货物明细不存在")
	ErrOrderCargoItemInvalidArgument = errors.BadRequest("ORDER_CARGO_ITEM_INVALID_ARGUMENT", "订单货物明细字段不合法")
)

type OrderCargoItem struct {
	ID            uuid.UUID
	OrderID       uuid.UUID
	CargoName     string
	PackageCount  int
	GrossWeightKg float64
	VolumeCbm     float64
	NetWeightKg   *float64
	Note          *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type OrderCargoItemRepo interface {
	List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderCargoItem, error)
	Add(ctx context.Context, organizationID, orderID uuid.UUID, input *OrderCargoItem, audit *AuditEvent) (*OrderCargoItem, error)
	Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *OrderCargoItem, audit *AuditEvent) (*OrderCargoItem, error)
	Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, audit *AuditEvent) error
}

type OrderCargoItemUsecase struct {
	repo OrderCargoItemRepo
}

func NewOrderCargoItemUsecase(repo OrderCargoItemRepo) *OrderCargoItemUsecase {
	return &OrderCargoItemUsecase{repo: repo}
}

func (uc *OrderCargoItemUsecase) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderCargoItem, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderCargoItemInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, orderID)
}

func (uc *OrderCargoItemUsecase) Add(ctx context.Context, organizationID, actorID, orderID uuid.UUID, input *OrderCargoItem) (*OrderCargoItem, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderCargoItemInvalidArgument
	}
	normalized, err := normalizeOrderCargoItem(input)
	if err != nil {
		return nil, err
	}
	normalized.ID = uuid.Must(uuid.NewV7())
	return uc.repo.Add(ctx, organizationID, orderID, normalized, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.cargo_item.add",
		Result:         "success",
		Details: map[string]string{
			"cargo_item.id":   normalized.ID.String(),
			"order.id":        orderID.String(),
			"cargo_item.name": normalized.CargoName,
		},
	})
}

func (uc *OrderCargoItemUsecase) Update(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID, input *OrderCargoItem) (*OrderCargoItem, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return nil, ErrOrderCargoItemInvalidArgument
	}
	normalized, err := normalizeOrderCargoItem(input)
	if err != nil {
		return nil, err
	}
	return uc.repo.Update(ctx, organizationID, orderID, id, normalized, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.cargo_item.update",
		Result:         "success",
		Details: map[string]string{
			"cargo_item.id":   id.String(),
			"order.id":        orderID.String(),
			"cargo_item.name": normalized.CargoName,
		},
	})
}

func (uc *OrderCargoItemUsecase) Remove(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return ErrOrderCargoItemInvalidArgument
	}
	return uc.repo.Remove(ctx, organizationID, orderID, id, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.cargo_item.remove",
		Result:         "success",
		Details: map[string]string{
			"cargo_item.id": id.String(),
			"order.id":      orderID.String(),
		},
	})
}

func normalizeOrderCargoItem(input *OrderCargoItem) (*OrderCargoItem, error) {
	if input == nil {
		return nil, ErrOrderCargoItemInvalidArgument
	}
	cargoName := strings.TrimSpace(input.CargoName)
	if cargoName == "" || utf8.RuneCountInString(cargoName) > 200 {
		return nil, ErrOrderCargoItemInvalidArgument
	}
	if input.PackageCount < 1 {
		return nil, ErrOrderCargoItemInvalidArgument
	}
	if input.GrossWeightKg <= 0 || input.VolumeCbm <= 0 {
		return nil, ErrOrderCargoItemInvalidArgument
	}
	var netWeight *float64
	if input.NetWeightKg != nil {
		if *input.NetWeightKg <= 0 {
			return nil, ErrOrderCargoItemInvalidArgument
		}
		v := *input.NetWeightKg
		netWeight = &v
	}
	var note *string
	if input.Note != nil {
		v := strings.TrimSpace(*input.Note)
		if v != "" {
			if utf8.RuneCountInString(v) > 500 {
				return nil, ErrOrderCargoItemInvalidArgument
			}
			note = &v
		}
	}
	output := *input
	output.CargoName = cargoName
	output.NetWeightKg = netWeight
	output.Note = note
	return &output, nil
}
