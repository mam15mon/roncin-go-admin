package biz

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type orderAbnormalCaseRepoStub struct {
	items         []*OrderAbnormalCase
	marked        *OrderAbnormalCase
	markedAudit   *AuditEvent
	resolved      *OrderAbnormalCase
	resolvedAudit *AuditEvent
	removed       bool
	removedAudit  *AuditEvent
	err           error
}

func (s *orderAbnormalCaseRepoStub) List(_ context.Context, _, orderID uuid.UUID) ([]*OrderAbnormalCase, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.items, nil
}

func (s *orderAbnormalCaseRepoStub) Mark(_ context.Context, _, orderID, actorID, abnormalCaseID uuid.UUID, newID uuid.UUID, audit *AuditEvent) (*OrderAbnormalCase, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.markedAudit = audit
	s.marked = &OrderAbnormalCase{
		ID:             newID,
		OrderID:        orderID,
		AbnormalCaseID: abnormalCaseID,
		Status:         OrderAbnormalCaseStatusActive,
		MarkedAt:       time.Now(),
		MarkedBy:       actorID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	return s.marked, nil
}

func (s *orderAbnormalCaseRepoStub) Resolve(_ context.Context, _, orderID, actorID, id uuid.UUID, audit *AuditEvent) (*OrderAbnormalCase, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.resolvedAudit = audit
	now := time.Now()
	s.resolved = &OrderAbnormalCase{
		ID:             id,
		OrderID:        orderID,
		AbnormalCaseID: uuid.New(),
		Status:         OrderAbnormalCaseStatusResolved,
		MarkedAt:       now.Add(-time.Hour),
		MarkedBy:       uuid.New(),
		ResolvedAt:     &now,
		ResolvedBy:     &actorID,
		CreatedAt:      now.Add(-time.Hour),
		UpdatedAt:      now,
	}
	return s.resolved, nil
}

func (s *orderAbnormalCaseRepoStub) Remove(_ context.Context, _, _, id uuid.UUID, audit *AuditEvent) error {
	if s.err != nil {
		return s.err
	}
	s.removed = true
	s.removedAudit = audit
	return nil
}

var _ OrderAbnormalCaseRepo = (*orderAbnormalCaseRepoStub)(nil)

func TestOrderAbnormalCaseStatusValid(t *testing.T) {
	validStatuses := []OrderAbnormalCaseStatus{
		OrderAbnormalCaseStatusActive,
		OrderAbnormalCaseStatusResolved,
		OrderAbnormalCaseStatusActive,
		OrderAbnormalCaseStatusResolved,
	}
	for _, status := range validStatuses {
		if !status.Valid() {
			t.Fatalf("expected status %q to be valid", status)
		}
	}

	invalidStatuses := []OrderAbnormalCaseStatus{
		"",
		"UNKNOWN",
		"active",
		"resolved",
		"PENDING",
	}
	for _, status := range invalidStatuses {
		if status.Valid() {
			t.Fatalf("expected status %q to be invalid", status)
		}
	}
}

func TestOrderAbnormalCaseListValidates(t *testing.T) {
	repo := &orderAbnormalCaseRepoStub{
		items: []*OrderAbnormalCase{
			{
				ID:             uuid.New(),
				OrderID:        uuid.New(),
				AbnormalCaseID: uuid.New(),
				Status:         OrderAbnormalCaseStatusActive,
				MarkedAt:       time.Now(),
				MarkedBy:       uuid.New(),
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			},
		},
	}
	usecase := NewOrderAbnormalCaseUsecase(repo)

	if _, err := usecase.List(context.Background(), uuid.Nil, uuid.New()); err != ErrOrderAbnormalCaseInvalidArgument {
		t.Fatalf("expected ErrOrderAbnormalCaseInvalidArgument for nil orgID, got %v", err)
	}
	if _, err := usecase.List(context.Background(), uuid.New(), uuid.Nil); err != ErrOrderAbnormalCaseInvalidArgument {
		t.Fatalf("expected ErrOrderAbnormalCaseInvalidArgument for nil orderID, got %v", err)
	}

	items, err := usecase.List(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func TestOrderAbnormalCaseMarkValidatesAndAudits(t *testing.T) {
	repo := &orderAbnormalCaseRepoStub{}
	usecase := NewOrderAbnormalCaseUsecase(repo)
	orgID, actorID, orderID, abnormalCaseID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	// Nil IDs
	if _, err := usecase.Mark(context.Background(), uuid.Nil, actorID, orderID, abnormalCaseID); err != ErrOrderAbnormalCaseInvalidArgument {
		t.Fatalf("expected ErrOrderAbnormalCaseInvalidArgument for nil orgID, got %v", err)
	}
	if _, err := usecase.Mark(context.Background(), orgID, uuid.Nil, orderID, abnormalCaseID); err != ErrOrderAbnormalCaseInvalidArgument {
		t.Fatalf("expected ErrOrderAbnormalCaseInvalidArgument for nil actorID, got %v", err)
	}
	if _, err := usecase.Mark(context.Background(), orgID, actorID, uuid.Nil, abnormalCaseID); err != ErrOrderAbnormalCaseInvalidArgument {
		t.Fatalf("expected ErrOrderAbnormalCaseInvalidArgument for nil orderID, got %v", err)
	}
	if _, err := usecase.Mark(context.Background(), orgID, actorID, orderID, uuid.Nil); err != ErrOrderAbnormalCaseInvalidArgument {
		t.Fatalf("expected ErrOrderAbnormalCaseInvalidArgument for nil abnormalCaseID, got %v", err)
	}

	// Success path with audit check
	repo.markedAudit = nil
	created, err := usecase.Mark(context.Background(), orgID, actorID, orderID, abnormalCaseID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created == nil || created.OrderID != orderID || created.AbnormalCaseID != abnormalCaseID {
		t.Fatalf("unexpected created abnormal case: %#v", created)
	}
	if created.Status != OrderAbnormalCaseStatusActive {
		t.Fatalf("expected status %q, got %q", OrderAbnormalCaseStatusActive, created.Status)
	}
	if repo.markedAudit == nil {
		t.Fatalf("expected non-nil markedAudit passed to repo")
	}
	event := repo.markedAudit
	if event.Action != "order.abnormal_case.mark" {
		t.Fatalf("expected action 'order.abnormal_case.mark', got %q", event.Action)
	}
	if event.Details["order.id"] != orderID.String() ||
		event.Details["abnormal_case.id"] != abnormalCaseID.String() ||
		event.Details["status"] != string(OrderAbnormalCaseStatusActive) {
		t.Fatalf("unexpected audit details: %#v", event.Details)
	}
}

func TestOrderAbnormalCaseResolveValidatesAndAudits(t *testing.T) {
	repo := &orderAbnormalCaseRepoStub{}
	usecase := NewOrderAbnormalCaseUsecase(repo)
	orgID, actorID, orderID, id := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	// Nil IDs
	if _, err := usecase.Resolve(context.Background(), uuid.Nil, actorID, orderID, id); err != ErrOrderAbnormalCaseInvalidArgument {
		t.Fatalf("expected ErrOrderAbnormalCaseInvalidArgument for nil orgID, got %v", err)
	}
	if _, err := usecase.Resolve(context.Background(), orgID, uuid.Nil, orderID, id); err != ErrOrderAbnormalCaseInvalidArgument {
		t.Fatalf("expected ErrOrderAbnormalCaseInvalidArgument for nil actorID, got %v", err)
	}
	if _, err := usecase.Resolve(context.Background(), orgID, actorID, uuid.Nil, id); err != ErrOrderAbnormalCaseInvalidArgument {
		t.Fatalf("expected ErrOrderAbnormalCaseInvalidArgument for nil orderID, got %v", err)
	}
	if _, err := usecase.Resolve(context.Background(), orgID, actorID, orderID, uuid.Nil); err != ErrOrderAbnormalCaseInvalidArgument {
		t.Fatalf("expected ErrOrderAbnormalCaseInvalidArgument for nil id, got %v", err)
	}

	// Success path with audit check
	repo.resolvedAudit = nil
	resolved, err := usecase.Resolve(context.Background(), orgID, actorID, orderID, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved == nil || resolved.ID != id || resolved.OrderID != orderID {
		t.Fatalf("unexpected resolved abnormal case: %#v", resolved)
	}
	if resolved.Status != OrderAbnormalCaseStatusResolved {
		t.Fatalf("expected status %q, got %q", OrderAbnormalCaseStatusResolved, resolved.Status)
	}
	if repo.resolvedAudit == nil {
		t.Fatalf("expected non-nil resolvedAudit passed to repo")
	}
	event := repo.resolvedAudit
	if event.Action != "order.abnormal_case.resolve" {
		t.Fatalf("expected action 'order.abnormal_case.resolve', got %q", event.Action)
	}
	if event.Details["abnormal_case_record.id"] != id.String() ||
		event.Details["order.id"] != orderID.String() {
		t.Fatalf("unexpected audit details: %#v", event.Details)
	}
}

func TestOrderAbnormalCaseRemoveValidatesAndAudits(t *testing.T) {
	repo := &orderAbnormalCaseRepoStub{}
	usecase := NewOrderAbnormalCaseUsecase(repo)
	orgID, actorID, orderID, id := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	// Nil IDs
	if err := usecase.Remove(context.Background(), uuid.Nil, actorID, orderID, id); err != ErrOrderAbnormalCaseInvalidArgument {
		t.Fatalf("expected ErrOrderAbnormalCaseInvalidArgument for nil orgID, got %v", err)
	}
	if err := usecase.Remove(context.Background(), orgID, uuid.Nil, orderID, id); err != ErrOrderAbnormalCaseInvalidArgument {
		t.Fatalf("expected ErrOrderAbnormalCaseInvalidArgument for nil actorID, got %v", err)
	}
	if err := usecase.Remove(context.Background(), orgID, actorID, uuid.Nil, id); err != ErrOrderAbnormalCaseInvalidArgument {
		t.Fatalf("expected ErrOrderAbnormalCaseInvalidArgument for nil orderID, got %v", err)
	}
	if err := usecase.Remove(context.Background(), orgID, actorID, orderID, uuid.Nil); err != ErrOrderAbnormalCaseInvalidArgument {
		t.Fatalf("expected ErrOrderAbnormalCaseInvalidArgument for nil id, got %v", err)
	}

	// Success
	repo.removedAudit = nil
	if err := usecase.Remove(context.Background(), orgID, actorID, orderID, id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.removed {
		t.Fatalf("expected repo.Remove to have been called")
	}
	if repo.removedAudit == nil {
		t.Fatalf("expected non-nil removedAudit passed to repo")
	}
	event := repo.removedAudit
	if event.Action != "order.abnormal_case.remove" {
		t.Fatalf("expected action 'order.abnormal_case.remove', got %q", event.Action)
	}
	if event.Details["abnormal_case_record.id"] != id.String() ||
		event.Details["order.id"] != orderID.String() {
		t.Fatalf("unexpected audit details: %#v", event.Details)
	}
}
