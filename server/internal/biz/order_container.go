package biz

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

type OrderContainer struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	OrderID         uuid.UUID
	ContainerNo     string
	ContainerSpecID uuid.UUID
	PackageCount    int32
	SealNo          *string
	GrossWeightKg   float64
	VolumeCbm       float64
	Note            *string
	Version         uint64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type OrderContainerRepo interface {
	List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderContainer, error)
	Add(ctx context.Context, organizationID, orderID uuid.UUID, input *OrderContainer, audit *AuditEvent) (*OrderContainer, error)
	Update(ctx context.Context, organizationID, orderID, id uuid.UUID, expectedVersion uint64, input *OrderContainer, audit *AuditEvent) (*OrderContainer, error)
	Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, expectedVersion uint64, audit *AuditEvent) error
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
	normalized.OrganizationID = organizationID
	normalized.OrderID = orderID
	normalized.Version = 1
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

func (uc *OrderContainerUsecase) Update(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID, expectedVersion uint64, input *OrderContainer) (*OrderContainer, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 {
		return nil, ErrOrderContainerInvalidArgument
	}
	normalized, err := normalizeOrderContainer(input)
	if err != nil {
		return nil, err
	}
	return uc.repo.Update(ctx, organizationID, orderID, id, expectedVersion, normalized, &AuditEvent{
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

func (uc *OrderContainerUsecase) Remove(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID, expectedVersion uint64) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 {
		return ErrOrderContainerInvalidArgument
	}
	return uc.repo.Remove(ctx, organizationID, orderID, id, expectedVersion, &AuditEvent{
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
	if input.PackageCount <= 0 {
		return nil, ErrOrderContainerInvalidArgument
	}
	if _, err := ValidateFloatWeight(input.GrossWeightKg, "毛重"); err != nil {
		return nil, ErrOrderContainerInvalidArgument
	}
	if _, err := ValidateFloatVolume(input.VolumeCbm, "体积"); err != nil {
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
