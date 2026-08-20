package biz

import (
	"context"
	"fmt"
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
	Set(context.Context, uuid.UUID, uuid.UUID, string, string, *time.Time, *string, bool, uuid.UUID) (*OrderMilestone, error)
}

type OrderMilestoneUsecase struct {
	repo  OrderMilestoneRepo
	audit AuditRepo
}

func NewOrderMilestoneUsecase(repo OrderMilestoneRepo, audit AuditRepo) *OrderMilestoneUsecase {
	return &OrderMilestoneUsecase{repo: repo, audit: audit}
}

func (uc *OrderMilestoneUsecase) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderMilestone, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderNotFound
	}
	return uc.repo.List(ctx, organizationID, orderID)
}

func (uc *OrderMilestoneUsecase) Set(ctx context.Context, organizationID, actorID, orderID uuid.UUID, milestoneType, expectedStatus string, occurredAt *time.Time, note *string, clearOccurredAt bool) (*OrderMilestone, error) {
	milestoneType = strings.ToUpper(strings.TrimSpace(milestoneType))
	expectedStatus = strings.ToUpper(strings.TrimSpace(expectedStatus))
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || milestoneType == "" || utf8.RuneCountInString(milestoneType) > 64 || utf8.RuneCountInString(derefString(note)) > 500 {
		return nil, ErrOrderInvalidArgument
	}
	if expectedStatus == "" {
		return nil, ErrOrderStatusConflict
	}
	if note != nil {
		value := strings.TrimSpace(*note)
		note = &value
	}
	updated, err := uc.repo.Set(ctx, organizationID, orderID, milestoneType, expectedStatus, occurredAt, note, clearOccurredAt, actorID)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "order.milestone.set", Result: "success", Details: map[string]string{"order.id": orderID.String(), "milestone.type": milestoneType, "occurred": fmt.Sprintf("%t", updated.OccurredAt != nil)}}); err != nil {
		return nil, fmt.Errorf("write order milestone audit: %w", err)
	}
	return updated, nil
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
