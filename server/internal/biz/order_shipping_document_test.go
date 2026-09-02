package biz

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type orderShippingDocumentRepoStub struct {
	added             *OrderShippingDocument
	addedAudit        *AuditEvent
	updated           *OrderShippingDocument
	updatedAudit      *AuditEvent
	transitioned      *OrderShippingDocument
	transitionedAudit *AuditEvent
	removed           bool
	removedAudit      *AuditEvent
}

func (s *orderShippingDocumentRepoStub) List(_ context.Context, _ uuid.UUID, orderID uuid.UUID) ([]*OrderShippingDocument, error) {
	return []*OrderShippingDocument{
		{
			ID:        uuid.New(),
			OrderID:   orderID,
			HouseNo:   "HBL123456",
			Status:    OrderShippingDocumentStatusDraft,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}, nil
}

func (s *orderShippingDocumentRepoStub) Add(_ context.Context, _ uuid.UUID, orderID uuid.UUID, input *OrderShippingDocument, audit *AuditEvent) (*OrderShippingDocument, error) {
	s.addedAudit = audit
	s.added = &OrderShippingDocument{
		ID:          input.ID,
		OrderID:     orderID,
		HouseNo:     input.HouseNo,
		ReleaseType: input.ReleaseType,
		Status:      input.Status,
		Note:        input.Note,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	return s.added, nil
}

func (s *orderShippingDocumentRepoStub) Update(_ context.Context, _ uuid.UUID, orderID, id uuid.UUID, input *OrderShippingDocument, audit *AuditEvent) (*OrderShippingDocument, error) {
	s.updatedAudit = audit
	s.updated = &OrderShippingDocument{
		ID:          id,
		OrderID:     orderID,
		HouseNo:     input.HouseNo,
		ReleaseType: input.ReleaseType,
		Status:      input.Status,
		Note:        input.Note,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	return s.updated, nil
}

func (s *orderShippingDocumentRepoStub) Transition(_ context.Context, _ uuid.UUID, orderID, id uuid.UUID, _, to OrderShippingDocumentStatus, audit *AuditEvent) (*OrderShippingDocument, error) {
	s.transitionedAudit = audit
	s.transitioned = &OrderShippingDocument{
		ID:        id,
		OrderID:   orderID,
		HouseNo:   "HBL123456",
		Status:    to,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return s.transitioned, nil
}

func (s *orderShippingDocumentRepoStub) Remove(_ context.Context, _ uuid.UUID, _, _ uuid.UUID, audit *AuditEvent) error {
	s.removed = true
	s.removedAudit = audit
	return nil
}

var _ OrderShippingDocumentRepo = (*orderShippingDocumentRepoStub)(nil)

func TestOrderShippingDocumentStatusValid(t *testing.T) {
	validStatuses := []OrderShippingDocumentStatus{
		OrderShippingDocumentStatusDraft,
		OrderShippingDocumentStatusConfirmed,
		OrderShippingDocumentStatusReleased,
	}
	for _, status := range validStatuses {
		if !status.Valid() {
			t.Fatalf("expected status %q to be valid", status)
		}
	}

	invalidStatuses := []OrderShippingDocumentStatus{
		"",
		"UNKNOWN",
		"draft",
		"PENDING",
	}
	for _, status := range invalidStatuses {
		if status.Valid() {
			t.Fatalf("expected status %q to be invalid", status)
		}
	}
}

func TestOrderShippingDocumentListValidates(t *testing.T) {
	repo := &orderShippingDocumentRepoStub{}
	usecase := NewOrderShippingDocumentUsecase(repo)

	if _, err := usecase.List(context.Background(), uuid.Nil, uuid.New()); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for nil orgID, got %v", err)
	}
	if _, err := usecase.List(context.Background(), uuid.New(), uuid.Nil); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for nil orderID, got %v", err)
	}

	items, err := usecase.List(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func TestOrderShippingDocumentAddValidatesAndAudits(t *testing.T) {
	repo := &orderShippingDocumentRepoStub{}
	usecase := NewOrderShippingDocumentUsecase(repo)
	orgID, actorID, orderID := uuid.New(), uuid.New(), uuid.New()

	baseValidInput := func() *OrderShippingDocument {
		return &OrderShippingDocument{
			HouseNo:     "HBL123456",
			ReleaseType: stringPtr("ORIGINAL"),
			Note:        stringPtr("test note"),
		}
	}

	// Nil IDs
	if _, err := usecase.Add(context.Background(), uuid.Nil, actorID, orderID, baseValidInput()); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for nil orgID, got %v", err)
	}
	if _, err := usecase.Add(context.Background(), orgID, uuid.Nil, orderID, baseValidInput()); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for nil actorID, got %v", err)
	}
	if _, err := usecase.Add(context.Background(), orgID, actorID, uuid.Nil, baseValidInput()); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for nil orderID, got %v", err)
	}

	// Nil input
	if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, nil); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for nil input, got %v", err)
	}

	// HouseNo validation
	invalidHouseNos := []string{"", "   ", strings.Repeat("B", 65)}
	for _, no := range invalidHouseNos {
		in := baseValidInput()
		in.HouseNo = no
		if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, in); err != ErrOrderShippingDocumentInvalidArgument {
			t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for houseNo %q, got %v", no, err)
		}
	}

	// ReleaseType too long
	{
		in := baseValidInput()
		in.ReleaseType = stringPtr(strings.Repeat("R", 65))
		if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, in); err != ErrOrderShippingDocumentInvalidArgument {
			t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for long releaseType, got %v", err)
		}
	}

	// Note too long
	{
		in := baseValidInput()
		in.Note = stringPtr(strings.Repeat("N", 501))
		if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, in); err != ErrOrderShippingDocumentInvalidArgument {
			t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for long note, got %v", err)
		}
	}

	// Blank optional fields normalized to nil and trimmed fields
	{
		in := baseValidInput()
		in.HouseNo = "  HBL999  "
		in.ReleaseType = stringPtr("   ")
		in.Note = stringPtr("   ")
		created, err := usecase.Add(context.Background(), orgID, actorID, orderID, in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created.HouseNo != "HBL999" {
			t.Fatalf("expected trimmed houseNo, got %q", created.HouseNo)
		}
		if created.ReleaseType != nil {
			t.Fatalf("expected nil ReleaseType for whitespace string, got %v", created.ReleaseType)
		}
		if created.Note != nil {
			t.Fatalf("expected nil Note for whitespace string, got %v", created.Note)
		}
	}

	// Success path with audit check
	repo.addedAudit = nil
	created, err := usecase.Add(context.Background(), orgID, actorID, orderID, baseValidInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created == nil || created.HouseNo != "HBL123456" {
		t.Fatalf("unexpected created shipping document: %#v", created)
	}
	if created.Status != OrderShippingDocumentStatusDraft {
		t.Fatalf("expected status %q, got %q", OrderShippingDocumentStatusDraft, created.Status)
	}
	if repo.addedAudit == nil {
		t.Fatalf("expected non-nil addedAudit passed to repo")
	}
	event := repo.addedAudit
	if event.Action != "order.shipping_document.add" {
		t.Fatalf("expected action 'order.shipping_document.add', got %q", event.Action)
	}
	if event.Details["shipping_document.id"] != created.ID.String() ||
		event.Details["order.id"] != orderID.String() ||
		event.Details["house_no"] != "HBL123456" {
		t.Fatalf("unexpected audit details: %#v", event.Details)
	}
}

func TestOrderShippingDocumentUpdateValidatesAndAudits(t *testing.T) {
	repo := &orderShippingDocumentRepoStub{}
	usecase := NewOrderShippingDocumentUsecase(repo)
	orgID, actorID, orderID, id := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	input := &OrderShippingDocument{
		HouseNo:     "HBL789",
		ReleaseType: stringPtr("SEAWAY_BILL"),
		Note:        stringPtr("updated note"),
	}

	// Nil IDs
	if _, err := usecase.Update(context.Background(), uuid.Nil, actorID, orderID, id, input); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for nil orgID, got %v", err)
	}
	if _, err := usecase.Update(context.Background(), orgID, uuid.Nil, orderID, id, input); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for nil actorID, got %v", err)
	}
	if _, err := usecase.Update(context.Background(), orgID, actorID, uuid.Nil, id, input); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for nil orderID, got %v", err)
	}
	if _, err := usecase.Update(context.Background(), orgID, actorID, orderID, uuid.Nil, input); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for nil id, got %v", err)
	}

	// Invalid input
	invalidIn := *input
	invalidIn.HouseNo = ""
	if _, err := usecase.Update(context.Background(), orgID, actorID, orderID, id, &invalidIn); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for empty houseNo, got %v", err)
	}

	invalidIn = *input
	invalidIn.HouseNo = strings.Repeat("H", 65)
	if _, err := usecase.Update(context.Background(), orgID, actorID, orderID, id, &invalidIn); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for long houseNo, got %v", err)
	}

	invalidIn = *input
	invalidIn.ReleaseType = stringPtr(strings.Repeat("R", 65))
	if _, err := usecase.Update(context.Background(), orgID, actorID, orderID, id, &invalidIn); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for long releaseType, got %v", err)
	}

	invalidIn = *input
	invalidIn.Note = stringPtr(strings.Repeat("N", 501))
	if _, err := usecase.Update(context.Background(), orgID, actorID, orderID, id, &invalidIn); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for long note, got %v", err)
	}

	// Blank optional normalized to nil
	{
		blankIn := *input
		blankIn.ReleaseType = stringPtr("   ")
		blankIn.Note = stringPtr("   ")
		updated, err := usecase.Update(context.Background(), orgID, actorID, orderID, id, &blankIn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.ReleaseType != nil {
			t.Fatalf("expected nil ReleaseType, got %v", updated.ReleaseType)
		}
		if updated.Note != nil {
			t.Fatalf("expected nil Note, got %v", updated.Note)
		}
	}

	// Success path
	repo.updatedAudit = nil
	updated, err := usecase.Update(context.Background(), orgID, actorID, orderID, id, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.ID != id || updated.HouseNo != "HBL789" {
		t.Fatalf("unexpected updated shipping document: %#v", updated)
	}
	if repo.updatedAudit == nil {
		t.Fatalf("expected non-nil updatedAudit passed to repo")
	}
	event := repo.updatedAudit
	if event.Action != "order.shipping_document.update" {
		t.Fatalf("expected action 'order.shipping_document.update', got %q", event.Action)
	}
	if event.Details["shipping_document.id"] != id.String() ||
		event.Details["order.id"] != orderID.String() ||
		event.Details["house_no"] != "HBL789" {
		t.Fatalf("unexpected audit details: %#v", event.Details)
	}
}

func TestOrderShippingDocumentTransitionValidatesAndAudits(t *testing.T) {
	repo := &orderShippingDocumentRepoStub{}
	usecase := NewOrderShippingDocumentUsecase(repo)
	orgID, actorID, orderID, id := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	// Nil IDs
	if _, err := usecase.Transition(context.Background(), uuid.Nil, actorID, orderID, id, OrderShippingDocumentStatusDraft, OrderShippingDocumentStatusConfirmed); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for nil orgID, got %v", err)
	}
	if _, err := usecase.Transition(context.Background(), orgID, uuid.Nil, orderID, id, OrderShippingDocumentStatusDraft, OrderShippingDocumentStatusConfirmed); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for nil actorID, got %v", err)
	}
	if _, err := usecase.Transition(context.Background(), orgID, actorID, uuid.Nil, id, OrderShippingDocumentStatusDraft, OrderShippingDocumentStatusConfirmed); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for nil orderID, got %v", err)
	}
	if _, err := usecase.Transition(context.Background(), orgID, actorID, orderID, uuid.Nil, OrderShippingDocumentStatusDraft, OrderShippingDocumentStatusConfirmed); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for nil id, got %v", err)
	}

	// Invalid transitions
	invalidTransitions := [][2]OrderShippingDocumentStatus{
		{OrderShippingDocumentStatusDraft, OrderShippingDocumentStatusReleased},
		{OrderShippingDocumentStatusReleased, OrderShippingDocumentStatusDraft},
		{OrderShippingDocumentStatusDraft, OrderShippingDocumentStatusDraft},
		{OrderShippingDocumentStatusConfirmed, OrderShippingDocumentStatusConfirmed},
		{OrderShippingDocumentStatusReleased, OrderShippingDocumentStatusReleased},
		{OrderShippingDocumentStatusReleased, OrderShippingDocumentStatusConfirmed},
		{"", OrderShippingDocumentStatusConfirmed},
		{OrderShippingDocumentStatusDraft, ""},
		{"INVALID", "INVALID"},
		{OrderShippingDocumentStatusDraft, "INVALID"},
	}
	for _, pair := range invalidTransitions {
		if _, err := usecase.Transition(context.Background(), orgID, actorID, orderID, id, pair[0], pair[1]); err != ErrOrderShippingDocumentInvalidStatus {
			t.Fatalf("expected ErrOrderShippingDocumentInvalidStatus for transition %q -> %q, got %v", pair[0], pair[1], err)
		}
	}

	// Valid transition: DRAFT -> CONFIRMED
	repo.transitionedAudit = nil
	transitioned, err := usecase.Transition(context.Background(), orgID, actorID, orderID, id, OrderShippingDocumentStatusDraft, OrderShippingDocumentStatusConfirmed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transitioned.Status != OrderShippingDocumentStatusConfirmed {
		t.Fatalf("expected status %q, got %q", OrderShippingDocumentStatusConfirmed, transitioned.Status)
	}
	if repo.transitionedAudit == nil {
		t.Fatalf("expected non-nil transitionedAudit passed to repo")
	}
	event := repo.transitionedAudit
	if event.Action != "order.shipping_document.transition" {
		t.Fatalf("expected action 'order.shipping_document.transition', got %q", event.Action)
	}
	if event.Details["shipping_document.id"] != id.String() ||
		event.Details["order.id"] != orderID.String() ||
		event.Details["from_status"] != string(OrderShippingDocumentStatusDraft) ||
		event.Details["to_status"] != string(OrderShippingDocumentStatusConfirmed) {
		t.Fatalf("unexpected audit details: %#v", event.Details)
	}

	// Valid transition: CONFIRMED -> RELEASED
	repo.transitionedAudit = nil
	transitioned, err = usecase.Transition(context.Background(), orgID, actorID, orderID, id, OrderShippingDocumentStatusConfirmed, OrderShippingDocumentStatusReleased)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transitioned.Status != OrderShippingDocumentStatusReleased {
		t.Fatalf("expected status %q, got %q", OrderShippingDocumentStatusReleased, transitioned.Status)
	}
	if repo.transitionedAudit == nil {
		t.Fatalf("expected non-nil transitionedAudit passed to repo")
	}
	event = repo.transitionedAudit
	if event.Details["from_status"] != string(OrderShippingDocumentStatusConfirmed) ||
		event.Details["to_status"] != string(OrderShippingDocumentStatusReleased) {
		t.Fatalf("unexpected audit details: %#v", event.Details)
	}
}

func TestOrderShippingDocumentRemoveValidatesAndAudits(t *testing.T) {
	repo := &orderShippingDocumentRepoStub{}
	usecase := NewOrderShippingDocumentUsecase(repo)
	orgID, actorID, orderID, id := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	// Nil IDs
	if err := usecase.Remove(context.Background(), uuid.Nil, actorID, orderID, id); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for nil orgID, got %v", err)
	}
	if err := usecase.Remove(context.Background(), orgID, uuid.Nil, orderID, id); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for nil actorID, got %v", err)
	}
	if err := usecase.Remove(context.Background(), orgID, actorID, uuid.Nil, id); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for nil orderID, got %v", err)
	}
	if err := usecase.Remove(context.Background(), orgID, actorID, orderID, uuid.Nil); err != ErrOrderShippingDocumentInvalidArgument {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidArgument for nil id, got %v", err)
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
	if event.Action != "order.shipping_document.remove" {
		t.Fatalf("expected action 'order.shipping_document.remove', got %q", event.Action)
	}
	if event.Details["shipping_document.id"] != id.String() ||
		event.Details["order.id"] != orderID.String() {
		t.Fatalf("unexpected audit details: %#v", event.Details)
	}
}
