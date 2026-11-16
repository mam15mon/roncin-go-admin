package biz

import (
	"context"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var ErrImportHandlingInvalid = errors.BadRequest("IMPORT_HANDLING_INVALID", "进口服务资料不合法")
var importHandlingKinds = []string{"ARRIVAL", "EXCHANGE", "CUSTOMS", "PICKUP", "DELIVERY"}

type ImportHandlingStep struct {
	ResponsibleUserName string      `json:"-"`
	Kind                string      `json:"kind"`
	Undertaken          bool        `json:"undertaken"`
	OccurredAt          *time.Time  `json:"occurredAt,omitempty"`
	ResponsibleUserID   *uuid.UUID  `json:"responsibleUserId,omitempty"`
	ReferenceNo         string      `json:"referenceNo"`
	Notes               string      `json:"notes"`
	AttachmentIDs       []uuid.UUID `json:"attachmentIds"`
}
type OrderImportHandling struct {
	OrderID           uuid.UUID            `json:"orderId"`
	OrderVersion      uint64               `json:"orderVersion"`
	ExchangeAgentID   *uuid.UUID           `json:"exchangeAgentId,omitempty"`
	ExchangeAgentName string               `json:"exchangeAgentName"`
	Steps             []ImportHandlingStep `json:"steps"`
}
type ImportHandlingEvent struct {
	ActorName  string
	ID         uuid.UUID
	OrderID    uuid.UUID
	ActorID    uuid.UUID
	OccurredAt time.Time
	Reason     string
	Before     *OrderImportHandling
	After      *OrderImportHandling
}

func EmptyImportHandling(orderID uuid.UUID, expectedVersion uint64) *OrderImportHandling {
	out := &OrderImportHandling{OrderID: orderID, OrderVersion: expectedVersion}
	for _, kind := range importHandlingKinds {
		out.Steps = append(out.Steps, ImportHandlingStep{Kind: kind, AttachmentIDs: []uuid.UUID{}})
	}
	return out
}
func ImportHandlingFlow(value *OrderImportHandling) OrderFlowStatus {
	undertaken, completed := 0, 0
	for _, step := range value.Steps {
		if step.Undertaken {
			undertaken++
			if step.OccurredAt != nil {
				completed++
			}
		}
	}
	if undertaken == 0 {
		return OrderFlowImportNotApplicable
	}
	if completed == 0 {
		return OrderFlowImportNotStarted
	}
	if completed == undertaken {
		return OrderFlowImportCompleted
	}
	return OrderFlowImportInProgress
}

// ValidateImportHandling 规范化五项完整快照；既有完成事实的改动必须说明原因。
func ValidateImportHandling(input, before *OrderImportHandling, reason string) error {
	if input == nil || len(input.Steps) != 5 || utf8.RuneCountInString(reason) > 500 {
		return ErrImportHandlingInvalid
	}
	byKind := map[string]ImportHandlingStep{}
	for _, step := range input.Steps {
		if _, ok := byKind[step.Kind]; ok {
			return ErrImportHandlingInvalid
		}
		step.ReferenceNo = strings.TrimSpace(step.ReferenceNo)
		step.Notes = strings.TrimSpace(step.Notes)
		if utf8.RuneCountInString(step.ReferenceNo) > 200 || utf8.RuneCountInString(step.Notes) > 2000 || len(step.AttachmentIDs) > 50 || !step.Undertaken && step.OccurredAt != nil || step.ResponsibleUserID != nil && *step.ResponsibleUserID == uuid.Nil {
			return ErrImportHandlingInvalid
		}
		if err := validateUUIDSet(step.AttachmentIDs); err != nil {
			return ErrImportHandlingInvalid
		}
		if step.OccurredAt != nil && step.OccurredAt.IsZero() {
			return ErrImportHandlingInvalid
		}
		if step.AttachmentIDs == nil {
			step.AttachmentIDs = []uuid.UUID{}
		}
		byKind[step.Kind] = step
	}
	if input.ExchangeAgentID != nil && *input.ExchangeAgentID == uuid.Nil {
		return ErrImportHandlingInvalid
	}
	if before != nil && before.Steps[1].OccurredAt != nil && !reflect.DeepEqual(before.ExchangeAgentID, input.ExchangeAgentID) && strings.TrimSpace(reason) == "" {
		return errors.BadRequest("IMPORT_HANDLING_REASON_REQUIRED", "纠正已完成换单的代理必须填写原因")
	}
	input.Steps = nil
	for i, kind := range importHandlingKinds {
		step, ok := byKind[kind]
		if !ok {
			return ErrImportHandlingInvalid
		}
		input.Steps = append(input.Steps, step)
		previous := ImportHandlingStep{}
		if before != nil {
			previous = before.Steps[i]
			previous.ResponsibleUserName = ""
		}
		step.ResponsibleUserName = ""
		if before != nil && previous.OccurredAt != nil && !reflect.DeepEqual(previous, step) && strings.TrimSpace(reason) == "" {
			return errors.BadRequest("IMPORT_HANDLING_REASON_REQUIRED", "撤回、取消或纠正已完成服务必须填写原因")
		}
	}
	return nil
}
func (uc *OrderUsecase) GetImportHandling(ctx context.Context, organizationID, orderID uuid.UUID) (*OrderImportHandling, error) {
	return uc.repo.GetImportHandling(ctx, organizationID, orderID)
}
func (uc *OrderUsecase) SaveImportHandling(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedVersion uint64, input *OrderImportHandling, reason string) (*OrderImportHandling, error) {
	reason = strings.TrimSpace(reason)
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || expectedVersion == 0 {
		return nil, ErrImportHandlingInvalid
	}
	if err := ValidateImportHandling(input, nil, reason); err != nil {
		return nil, err
	}
	return uc.repo.SaveImportHandling(ctx, organizationID, actorID, orderID, expectedVersion, input, reason)
}
func (uc *OrderUsecase) ListImportHandlingEvents(ctx context.Context, organizationID, orderID uuid.UUID, page, size int) ([]*ImportHandlingEvent, int, error) {
	if !ValidListPagination(page, size) {
		return nil, 0, ErrImportHandlingInvalid
	}
	return uc.repo.ListImportHandlingEvents(ctx, organizationID, orderID, page, size)
}
