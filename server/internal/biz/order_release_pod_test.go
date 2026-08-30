package biz

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type orderReleasePodRepoStub struct {
	added        *OrderReleasePod
	updated      *OrderReleasePod
	transitioned *OrderReleasePod
	removed      bool
	audits       []*AuditEvent
}

func (s *orderReleasePodRepoStub) List(_ context.Context, _ uuid.UUID, orderID uuid.UUID) ([]*OrderReleasePod, error) {
	return []*OrderReleasePod{
		{
			ID:        uuid.New(),
			OrderID:   orderID,
			Status:    OrderReleasePodStatusPending,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}, nil
}

func (s *orderReleasePodRepoStub) Add(_ context.Context, _ uuid.UUID, orderID uuid.UUID, input *OrderReleasePod, audit *AuditEvent) (*OrderReleasePod, error) {
	s.added = &OrderReleasePod{
		ID:                 input.ID,
		OrderID:            orderID,
		ShippingDocumentID: input.ShippingDocumentID,
		ReleaseNo:          input.ReleaseNo,
		PodNo:              input.PodNo,
		Status:             input.Status,
		SignedAt:           input.SignedAt,
		SignedBy:           input.SignedBy,
		Note:               input.Note,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	s.audits = append(s.audits, audit)
	return s.added, nil
}

func (s *orderReleasePodRepoStub) Update(_ context.Context, _ uuid.UUID, orderID, id uuid.UUID, input *OrderReleasePod, audit *AuditEvent) (*OrderReleasePod, error) {
	s.updated = &OrderReleasePod{
		ID:                 id,
		OrderID:            orderID,
		ShippingDocumentID: input.ShippingDocumentID,
		ReleaseNo:          input.ReleaseNo,
		PodNo:              input.PodNo,
		Status:             input.Status,
		SignedAt:           input.SignedAt,
		SignedBy:           input.SignedBy,
		Note:               input.Note,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	s.audits = append(s.audits, audit)
	return s.updated, nil
}

func (s *orderReleasePodRepoStub) Transition(_ context.Context, _ uuid.UUID, orderID, id uuid.UUID, _, to OrderReleasePodStatus, actorID uuid.UUID, audit *AuditEvent) (*OrderReleasePod, error) {
	now := time.Now()
	s.transitioned = &OrderReleasePod{
		ID:        id,
		OrderID:   orderID,
		Status:    to,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if to == OrderReleasePodStatusSigned {
		s.transitioned.SignedAt = &now
		s.transitioned.SignedBy = &actorID
	}
	s.audits = append(s.audits, audit)
	return s.transitioned, nil
}

func (s *orderReleasePodRepoStub) Remove(_ context.Context, _ uuid.UUID, _, _ uuid.UUID, audit *AuditEvent) error {
	s.removed = true
	s.audits = append(s.audits, audit)
	return nil
}

var _ OrderReleasePodRepo = (*orderReleasePodRepoStub)(nil)

func TestOrderReleasePodStatusValid(t *testing.T) {
	validStatuses := []OrderReleasePodStatus{
		OrderReleasePodStatusPending,
		OrderReleasePodStatusSigned,
		OrderReleasePodStatusReturned,
	}
	for _, status := range validStatuses {
		if !status.Valid() {
			t.Fatalf("expected status %q to be valid", status)
		}
	}

	invalidStatuses := []OrderReleasePodStatus{
		"",
		"UNKNOWN",
		"pending",
		"signed",
		"returned",
		"DRAFT",
		"CONFIRMED",
		"RELEASED",
	}
	for _, status := range invalidStatuses {
		if status.Valid() {
			t.Fatalf("expected status %q to be invalid", status)
		}
	}
}

func TestOrderReleasePodAllowedTargetStatuses(t *testing.T) {
	tests := []struct {
		status OrderReleasePodStatus
		want   OrderReleasePodStatus
	}{
		{status: OrderReleasePodStatusPending, want: OrderReleasePodStatusSigned},
		{status: OrderReleasePodStatusSigned, want: OrderReleasePodStatusReturned},
		{status: OrderReleasePodStatusReturned},
	}
	for _, tt := range tests {
		targets := (&OrderReleasePod{Status: tt.status}).AllowedTargetStatuses()
		if tt.want == "" && len(targets) != 0 {
			t.Fatalf("状态 %q 不应允许流转: %v", tt.status, targets)
		}
		if tt.want != "" && (len(targets) != 1 || targets[0] != tt.want) {
			t.Fatalf("状态 %q 允许流转 = %v，期望 %q", tt.status, targets, tt.want)
		}
	}
}

func TestOrderReleasePodListValidates(t *testing.T) {
	repo := &orderReleasePodRepoStub{}
	usecase := NewOrderReleasePodUsecase(repo)

	if _, err := usecase.List(context.Background(), uuid.Nil, uuid.New()); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for nil orgID, got %v", err)
	}
	if _, err := usecase.List(context.Background(), uuid.New(), uuid.Nil); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for nil orderID, got %v", err)
	}

	items, err := usecase.List(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func TestOrderReleasePodAddValidatesAndAudits(t *testing.T) {
	repo := &orderReleasePodRepoStub{}
	usecase := NewOrderReleasePodUsecase(repo)
	orgID, actorID, orderID := uuid.New(), uuid.New(), uuid.New()
	docID := uuid.New()

	baseValidInput := func() *OrderReleasePod {
		return &OrderReleasePod{
			ShippingDocumentID: &docID,
			ReleaseNo:          stringPtr("REL123456"),
			PodNo:              stringPtr("POD123456"),
			Note:               stringPtr("test note"),
		}
	}

	// Nil IDs
	if _, err := usecase.Add(context.Background(), uuid.Nil, actorID, orderID, baseValidInput()); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for nil orgID, got %v", err)
	}
	if _, err := usecase.Add(context.Background(), orgID, uuid.Nil, orderID, baseValidInput()); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for nil actorID, got %v", err)
	}
	if _, err := usecase.Add(context.Background(), orgID, actorID, uuid.Nil, baseValidInput()); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for nil orderID, got %v", err)
	}

	// Nil input
	if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, nil); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for nil input, got %v", err)
	}

	// ReleaseNo validation
	{
		in := baseValidInput()
		in.ReleaseNo = stringPtr(strings.Repeat("R", 65))
		if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, in); err != ErrOrderReleasePodInvalidArgument {
			t.Fatalf("expected ErrOrderReleasePodInvalidArgument for long releaseNo, got %v", err)
		}
	}

	// PodNo validation
	{
		in := baseValidInput()
		in.PodNo = stringPtr(strings.Repeat("P", 65))
		if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, in); err != ErrOrderReleasePodInvalidArgument {
			t.Fatalf("expected ErrOrderReleasePodInvalidArgument for long podNo, got %v", err)
		}
	}

	// Note too long
	{
		in := baseValidInput()
		in.Note = stringPtr(strings.Repeat("N", 501))
		if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, in); err != ErrOrderReleasePodInvalidArgument {
			t.Fatalf("expected ErrOrderReleasePodInvalidArgument for long note, got %v", err)
		}
	}

	// Blank optional fields normalized to nil and trimmed fields
	{
		in := baseValidInput()
		in.ReleaseNo = stringPtr("  REL999  ")
		in.PodNo = stringPtr("   ")
		in.Note = stringPtr("   ")
		created, err := usecase.Add(context.Background(), orgID, actorID, orderID, in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created.ReleaseNo == nil || *created.ReleaseNo != "REL999" {
			t.Fatalf("expected trimmed releaseNo, got %v", created.ReleaseNo)
		}
		if created.PodNo != nil {
			t.Fatalf("expected nil PodNo for whitespace string, got %v", created.PodNo)
		}
		if created.Note != nil {
			t.Fatalf("expected nil Note for whitespace string, got %v", created.Note)
		}
	}

	// Success path with audit check
	repo.audits = nil
	created, err := usecase.Add(context.Background(), orgID, actorID, orderID, baseValidInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created == nil || created.ReleaseNo == nil || *created.ReleaseNo != "REL123456" {
		t.Fatalf("unexpected created release pod: %#v", created)
	}
	if created.Status != OrderReleasePodStatusPending {
		t.Fatalf("expected status %q, got %q", OrderReleasePodStatusPending, created.Status)
	}
	if len(repo.audits) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(repo.audits))
	}
	event := repo.audits[0]
	if event.Action != "order.release_pod.add" {
		t.Fatalf("expected action 'order.release_pod.add', got %q", event.Action)
	}
	if event.Details["release_pod.id"] != created.ID.String() ||
		event.Details["order.id"] != orderID.String() {
		t.Fatalf("unexpected audit details: %#v", event.Details)
	}
}

func TestOrderReleasePodUpdateValidatesAndAudits(t *testing.T) {
	repo := &orderReleasePodRepoStub{}
	usecase := NewOrderReleasePodUsecase(repo)
	orgID, actorID, orderID, id := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	docID := uuid.New()

	input := &OrderReleasePod{
		ShippingDocumentID: &docID,
		ReleaseNo:          stringPtr("REL789"),
		PodNo:              stringPtr("POD789"),
		Note:               stringPtr("updated note"),
	}

	// Nil IDs
	if _, err := usecase.Update(context.Background(), uuid.Nil, actorID, orderID, id, input); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for nil orgID, got %v", err)
	}
	if _, err := usecase.Update(context.Background(), orgID, uuid.Nil, orderID, id, input); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for nil actorID, got %v", err)
	}
	if _, err := usecase.Update(context.Background(), orgID, actorID, uuid.Nil, id, input); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for nil orderID, got %v", err)
	}
	if _, err := usecase.Update(context.Background(), orgID, actorID, orderID, uuid.Nil, input); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for nil id, got %v", err)
	}

	// Nil input
	if _, err := usecase.Update(context.Background(), orgID, actorID, orderID, id, nil); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for nil input, got %v", err)
	}

	// Invalid input
	invalidIn := *input
	invalidIn.ReleaseNo = stringPtr(strings.Repeat("R", 65))
	if _, err := usecase.Update(context.Background(), orgID, actorID, orderID, id, &invalidIn); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for long releaseNo, got %v", err)
	}

	invalidIn = *input
	invalidIn.PodNo = stringPtr(strings.Repeat("P", 65))
	if _, err := usecase.Update(context.Background(), orgID, actorID, orderID, id, &invalidIn); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for long podNo, got %v", err)
	}

	invalidIn = *input
	invalidIn.Note = stringPtr(strings.Repeat("N", 501))
	if _, err := usecase.Update(context.Background(), orgID, actorID, orderID, id, &invalidIn); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for long note, got %v", err)
	}

	// Blank optional normalized to nil
	{
		blankIn := *input
		blankIn.ReleaseNo = stringPtr("   ")
		blankIn.PodNo = stringPtr("   ")
		blankIn.Note = stringPtr("   ")
		updated, err := usecase.Update(context.Background(), orgID, actorID, orderID, id, &blankIn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.ReleaseNo != nil {
			t.Fatalf("expected nil ReleaseNo, got %v", updated.ReleaseNo)
		}
		if updated.PodNo != nil {
			t.Fatalf("expected nil PodNo, got %v", updated.PodNo)
		}
		if updated.Note != nil {
			t.Fatalf("expected nil Note, got %v", updated.Note)
		}
	}

	// Success path
	repo.audits = nil
	updated, err := usecase.Update(context.Background(), orgID, actorID, orderID, id, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.ID != id || updated.ReleaseNo == nil || *updated.ReleaseNo != "REL789" {
		t.Fatalf("unexpected updated release pod: %#v", updated)
	}
	if len(repo.audits) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(repo.audits))
	}
	event := repo.audits[0]
	if event.Action != "order.release_pod.update" {
		t.Fatalf("expected action 'order.release_pod.update', got %q", event.Action)
	}
	if event.Details["release_pod.id"] != id.String() ||
		event.Details["order.id"] != orderID.String() {
		t.Fatalf("unexpected audit details: %#v", event.Details)
	}
}

func TestOrderReleasePodTransitionValidatesAndAudits(t *testing.T) {
	repo := &orderReleasePodRepoStub{}
	usecase := NewOrderReleasePodUsecase(repo)
	orgID, actorID, orderID, id := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	// Nil IDs
	if _, err := usecase.Transition(context.Background(), uuid.Nil, actorID, orderID, id, OrderReleasePodStatusPending, OrderReleasePodStatusSigned); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for nil orgID, got %v", err)
	}
	if _, err := usecase.Transition(context.Background(), orgID, uuid.Nil, orderID, id, OrderReleasePodStatusPending, OrderReleasePodStatusSigned); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for nil actorID, got %v", err)
	}
	if _, err := usecase.Transition(context.Background(), orgID, actorID, uuid.Nil, id, OrderReleasePodStatusPending, OrderReleasePodStatusSigned); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for nil orderID, got %v", err)
	}
	if _, err := usecase.Transition(context.Background(), orgID, actorID, orderID, uuid.Nil, OrderReleasePodStatusPending, OrderReleasePodStatusSigned); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for nil id, got %v", err)
	}

	// Invalid transitions
	invalidTransitions := [][2]OrderReleasePodStatus{
		{OrderReleasePodStatusPending, OrderReleasePodStatusReturned},
		{OrderReleasePodStatusReturned, OrderReleasePodStatusPending},
		{OrderReleasePodStatusPending, OrderReleasePodStatusPending},
		{OrderReleasePodStatusSigned, OrderReleasePodStatusSigned},
		{OrderReleasePodStatusReturned, OrderReleasePodStatusReturned},
		{OrderReleasePodStatusReturned, OrderReleasePodStatusSigned},
		{"", OrderReleasePodStatusSigned},
		{OrderReleasePodStatusPending, ""},
		{"INVALID", "INVALID"},
		{OrderReleasePodStatusPending, "INVALID"},
	}
	for _, pair := range invalidTransitions {
		if _, err := usecase.Transition(context.Background(), orgID, actorID, orderID, id, pair[0], pair[1]); err != ErrOrderReleasePodInvalidStatus {
			t.Fatalf("expected ErrOrderReleasePodInvalidStatus for transition %q -> %q, got %v", pair[0], pair[1], err)
		}
	}

	// Valid transition: PENDING -> SIGNED
	repo.audits = nil
	transitioned, err := usecase.Transition(context.Background(), orgID, actorID, orderID, id, OrderReleasePodStatusPending, OrderReleasePodStatusSigned)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transitioned.Status != OrderReleasePodStatusSigned {
		t.Fatalf("expected status %q, got %q", OrderReleasePodStatusSigned, transitioned.Status)
	}
	if transitioned.SignedAt == nil || transitioned.SignedBy == nil || *transitioned.SignedBy != actorID {
		t.Fatalf("expected SignedAt and SignedBy to be populated")
	}
	if len(repo.audits) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(repo.audits))
	}
	event := repo.audits[0]
	if event.Action != "order.release_pod.transition" {
		t.Fatalf("expected action 'order.release_pod.transition', got %q", event.Action)
	}
	if event.Details["release_pod.id"] != id.String() ||
		event.Details["order.id"] != orderID.String() ||
		event.Details["from_status"] != string(OrderReleasePodStatusPending) ||
		event.Details["to_status"] != string(OrderReleasePodStatusSigned) {
		t.Fatalf("unexpected audit details: %#v", event.Details)
	}

	// Valid transition: SIGNED -> RETURNED
	repo.audits = nil
	transitioned, err = usecase.Transition(context.Background(), orgID, actorID, orderID, id, OrderReleasePodStatusSigned, OrderReleasePodStatusReturned)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transitioned.Status != OrderReleasePodStatusReturned {
		t.Fatalf("expected status %q, got %q", OrderReleasePodStatusReturned, transitioned.Status)
	}
	if len(repo.audits) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(repo.audits))
	}
	event = repo.audits[0]
	if event.Action != "order.release_pod.transition" {
		t.Fatalf("expected action 'order.release_pod.transition', got %q", event.Action)
	}
	if event.Details["release_pod.id"] != id.String() ||
		event.Details["order.id"] != orderID.String() ||
		event.Details["from_status"] != string(OrderReleasePodStatusSigned) ||
		event.Details["to_status"] != string(OrderReleasePodStatusReturned) {
		t.Fatalf("unexpected audit details: %#v", event.Details)
	}
}

func TestOrderReleasePodRemoveValidatesAndAudits(t *testing.T) {
	repo := &orderReleasePodRepoStub{}
	usecase := NewOrderReleasePodUsecase(repo)
	orgID, actorID, orderID, id := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	// Nil IDs
	if err := usecase.Remove(context.Background(), uuid.Nil, actorID, orderID, id); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for nil orgID, got %v", err)
	}
	if err := usecase.Remove(context.Background(), orgID, uuid.Nil, orderID, id); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for nil actorID, got %v", err)
	}
	if err := usecase.Remove(context.Background(), orgID, actorID, uuid.Nil, id); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for nil orderID, got %v", err)
	}
	if err := usecase.Remove(context.Background(), orgID, actorID, orderID, uuid.Nil); err != ErrOrderReleasePodInvalidArgument {
		t.Fatalf("expected ErrOrderReleasePodInvalidArgument for nil id, got %v", err)
	}

	// Success
	repo.audits = nil
	if err := usecase.Remove(context.Background(), orgID, actorID, orderID, id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.removed {
		t.Fatalf("expected repo.Remove to have been called")
	}
	if len(repo.audits) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(repo.audits))
	}
	event := repo.audits[0]
	if event.Action != "order.release_pod.remove" {
		t.Fatalf("expected action 'order.release_pod.remove', got %q", event.Action)
	}
	if event.Details["release_pod.id"] != id.String() ||
		event.Details["order.id"] != orderID.String() {
		t.Fatalf("unexpected audit details: %#v", event.Details)
	}
}
