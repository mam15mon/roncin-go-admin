package biz

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

type OrderMilestone struct {
	ID                uuid.UUID
	OrderID           uuid.UUID
	Type              string
	TemplateNodeCode  *string
	TemplateNodeLabel *string
	OccurredAt        *time.Time
	Note              *string
	UpdatedBy         *uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type OrderMilestoneRepo interface {
	List(context.Context, uuid.UUID, uuid.UUID) ([]*OrderMilestone, error)
	Set(context.Context, uuid.UUID, uuid.UUID, string, uint64, *time.Time, *string, bool, uuid.UUID, *AuditEvent) (*OrderMilestone, error)
}

type OrderMilestoneUsecase struct {
	repo OrderMilestoneRepo
}

func NewOrderMilestoneUsecase(repo OrderMilestoneRepo) *OrderMilestoneUsecase {
	return &OrderMilestoneUsecase{repo: repo}
}

func (uc *OrderMilestoneUsecase) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderMilestone, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderNotFound
	}
	return uc.repo.List(ctx, organizationID, orderID)
}

func (uc *OrderMilestoneUsecase) Set(ctx context.Context, organizationID, actorID, orderID uuid.UUID, milestoneType string, expectedVersion uint64, occurredAt *time.Time, note *string, clearOccurredAt bool) (*OrderMilestone, error) {
	milestoneType = strings.ToUpper(strings.TrimSpace(milestoneType))
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || milestoneType == "" || utf8.RuneCountInString(milestoneType) > 64 || utf8.RuneCountInString(derefString(note)) > 500 {
		return nil, ErrOrderInvalidArgument
	}
	if expectedVersion == 0 {
		return nil, ErrOrderStatusConflict
	}
	if note != nil {
		value := strings.TrimSpace(*note)
		note = &value
	}
	audit := &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.milestone.set",
		Result:         "success",
		Details: map[string]string{
			"order.id":       orderID.String(),
			"milestone.type": milestoneType,
		},
	}
	updated, err := uc.repo.Set(ctx, organizationID, orderID, milestoneType, expectedVersion, occurredAt, note, clearOccurredAt, actorID, audit)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
