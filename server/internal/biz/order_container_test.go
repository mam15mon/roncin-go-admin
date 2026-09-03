package biz

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type orderContainerRepoStub struct {
	added        *OrderContainer
	addedAudit   *AuditEvent
	updated      *OrderContainer
	updatedAudit *AuditEvent
	removed      bool
	removedAudit *AuditEvent
}

func (s *orderContainerRepoStub) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderContainer, error) {
	return []*OrderContainer{
		{
			ID:              uuid.New(),
			OrderID:         orderID,
			ContainerNo:     "MSCU1234567",
			ContainerSpecID: uuid.New(),
			GrossWeightKg:   2000.5,
			VolumeCbm:       30.2,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}, nil
}

func (s *orderContainerRepoStub) Add(ctx context.Context, organizationID, orderID uuid.UUID, input *OrderContainer, audit *AuditEvent) (*OrderContainer, error) {
	s.addedAudit = audit
	s.added = &OrderContainer{
		ID:              input.ID,
		OrderID:         orderID,
		ContainerNo:     input.ContainerNo,
		ContainerSpecID: input.ContainerSpecID,
		SealNo:          input.SealNo,
		GrossWeightKg:   input.GrossWeightKg,
		VolumeCbm:       input.VolumeCbm,
		Note:            input.Note,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	return s.added, nil
}

func (s *orderContainerRepoStub) Update(ctx context.Context, organizationID, orderID, id uuid.UUID, expectedVersion uint64, input *OrderContainer, audit *AuditEvent) (*OrderContainer, error) {
	s.updatedAudit = audit
	s.updated = &OrderContainer{
		ID:              id,
		OrderID:         orderID,
		ContainerNo:     input.ContainerNo,
		ContainerSpecID: input.ContainerSpecID,
		PackageCount:    input.PackageCount,
		SealNo:          input.SealNo,
		GrossWeightKg:   input.GrossWeightKg,
		VolumeCbm:       input.VolumeCbm,
		Note:            input.Note,
		Version:         expectedVersion + 1,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	return s.updated, nil
}

func (s *orderContainerRepoStub) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, expectedVersion uint64, audit *AuditEvent) error {
	s.removed = true
	s.removedAudit = audit
	return nil
}

var _ OrderContainerRepo = (*orderContainerRepoStub)(nil)

func TestOrderContainerListValidates(t *testing.T) {
	repo := &orderContainerRepoStub{}
	usecase := NewOrderContainerUsecase(repo)

	if _, err := usecase.List(context.Background(), uuid.Nil, uuid.New()); err != ErrOrderContainerInvalidArgument {
		t.Fatalf("expected ErrOrderContainerInvalidArgument for nil orgID, got %v", err)
	}
	if _, err := usecase.List(context.Background(), uuid.New(), uuid.Nil); err != ErrOrderContainerInvalidArgument {
		t.Fatalf("expected ErrOrderContainerInvalidArgument for nil orderID, got %v", err)
	}

	items, err := usecase.List(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func TestOrderContainerAddValidatesAndAudits(t *testing.T) {
	repo := &orderContainerRepoStub{}
	usecase := NewOrderContainerUsecase(repo)
	orgID, actorID, orderID, specID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	baseValidInput := func() *OrderContainer {
		return &OrderContainer{
			ContainerNo:     "MSCU1234567",
			ContainerSpecID: specID,
			PackageCount:    100,
			GrossWeightKg:   15000,
			VolumeCbm:       33.2,
			SealNo:          stringPtr("SEAL123456"),
			Note:            stringPtr("standard dry container"),
		}
	}

	// Nil IDs
	if _, err := usecase.Add(context.Background(), uuid.Nil, actorID, orderID, baseValidInput()); err != ErrOrderContainerInvalidArgument {
		t.Fatalf("expected ErrOrderContainerInvalidArgument for nil orgID, got %v", err)
	}
	if _, err := usecase.Add(context.Background(), orgID, uuid.Nil, orderID, baseValidInput()); err != ErrOrderContainerInvalidArgument {
		t.Fatalf("expected ErrOrderContainerInvalidArgument for nil actorID, got %v", err)
	}
	if _, err := usecase.Add(context.Background(), orgID, actorID, uuid.Nil, baseValidInput()); err != ErrOrderContainerInvalidArgument {
		t.Fatalf("expected ErrOrderContainerInvalidArgument for nil orderID, got %v", err)
	}

	// Nil input
	if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, nil); err != ErrOrderContainerInvalidArgument {
		t.Fatalf("expected ErrOrderContainerInvalidArgument for nil input, got %v", err)
	}

	// ContainerNo validation
	invalidNos := []string{"", "   ", strings.Repeat("A", 65)}
	for _, no := range invalidNos {
		in := baseValidInput()
		in.ContainerNo = no
		if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, in); err != ErrOrderContainerInvalidArgument {
			t.Fatalf("expected ErrOrderContainerInvalidArgument for containerNo %q, got %v", no, err)
		}
	}

	// SpecID validation
	{
		in := baseValidInput()
		in.ContainerSpecID = uuid.Nil
		if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, in); err != ErrOrderContainerInvalidArgument {
			t.Fatalf("expected ErrOrderContainerInvalidArgument for nil specID, got %v", err)
		}
	}

	// GrossWeightKg validation
	for _, gw := range []float64{0, -1, -100.5} {
		in := baseValidInput()
		in.GrossWeightKg = gw
		if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, in); err != ErrOrderContainerInvalidArgument {
			t.Fatalf("expected ErrOrderContainerInvalidArgument for GrossWeightKg %f, got %v", gw, err)
		}
	}

	// VolumeCbm validation
	for _, v := range []float64{0, -1, -50.2} {
		in := baseValidInput()
		in.VolumeCbm = v
		if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, in); err != ErrOrderContainerInvalidArgument {
			t.Fatalf("expected ErrOrderContainerInvalidArgument for VolumeCbm %f, got %v", v, err)
		}
	}

	// SealNo too long
	{
		in := baseValidInput()
		in.SealNo = stringPtr(strings.Repeat("S", 65))
		if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, in); err != ErrOrderContainerInvalidArgument {
			t.Fatalf("expected ErrOrderContainerInvalidArgument for long sealNo, got %v", err)
		}
	}

	// Note too long
	{
		in := baseValidInput()
		in.Note = stringPtr(strings.Repeat("N", 501))
		if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, in); err != ErrOrderContainerInvalidArgument {
			t.Fatalf("expected ErrOrderContainerInvalidArgument for long note, got %v", err)
		}
	}

	// Blank optional fields normalized to nil
	{
		in := baseValidInput()
		in.ContainerNo = "  MSCU7654321  "
		in.SealNo = stringPtr("   ")
		in.Note = stringPtr("   ")
		created, err := usecase.Add(context.Background(), orgID, actorID, orderID, in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created.ContainerNo != "MSCU7654321" {
			t.Fatalf("expected trimmed containerNo, got %q", created.ContainerNo)
		}
		if created.SealNo != nil {
			t.Fatalf("expected nil SealNo for whitespace string, got %v", created.SealNo)
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
	if created == nil || created.ContainerNo != "MSCU1234567" {
		t.Fatalf("unexpected created container: %#v", created)
	}
	if repo.addedAudit == nil {
		t.Fatalf("expected non-nil addedAudit passed to repo")
	}
	event := repo.addedAudit
	if event.Action != "order.container.add" {
		t.Fatalf("expected action 'order.container.add', got %q", event.Action)
	}
	if event.Details["container.id"] != created.ID.String() ||
		event.Details["order.id"] != orderID.String() ||
		event.Details["container.no"] != "MSCU1234567" ||
		event.Details["container.spec_id"] != specID.String() {
		t.Fatalf("unexpected audit details: %#v", event.Details)
	}
}

func TestOrderContainerUpdateValidatesAndAudits(t *testing.T) {
	repo := &orderContainerRepoStub{}
	usecase := NewOrderContainerUsecase(repo)
	orgID, actorID, orderID, id, specID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()

	input := &OrderContainer{
		ContainerNo:     "COSU9876543",
		ContainerSpecID: specID,
		PackageCount:    100,
		GrossWeightKg:   22000,
		VolumeCbm:       55.5,
		SealNo:          stringPtr("SL9999"),
		Note:            stringPtr("updated note"),
	}

	// Nil IDs
	if _, err := usecase.Update(context.Background(), uuid.Nil, actorID, orderID, id, 1, input); err != ErrOrderContainerInvalidArgument {
		t.Fatalf("expected ErrOrderContainerInvalidArgument for nil orgID, got %v", err)
	}
	if _, err := usecase.Update(context.Background(), orgID, uuid.Nil, orderID, id, 1, input); err != ErrOrderContainerInvalidArgument {
		t.Fatalf("expected ErrOrderContainerInvalidArgument for nil actorID, got %v", err)
	}
	if _, err := usecase.Update(context.Background(), orgID, actorID, uuid.Nil, id, 1, input); err != ErrOrderContainerInvalidArgument {
		t.Fatalf("expected ErrOrderContainerInvalidArgument for nil orderID, got %v", err)
	}
	if _, err := usecase.Update(context.Background(), orgID, actorID, orderID, uuid.Nil, 1, input); err != ErrOrderContainerInvalidArgument {
		t.Fatalf("expected ErrOrderContainerInvalidArgument for nil id, got %v", err)
	}
	if _, err := usecase.Update(context.Background(), orgID, actorID, orderID, id, 0, input); err != ErrOrderContainerInvalidArgument {
		t.Fatalf("expected ErrOrderContainerInvalidArgument for zero expectedVersion, got %v", err)
	}

	// Invalid input
	invalidIn := *input
	invalidIn.ContainerNo = ""
	if _, err := usecase.Update(context.Background(), orgID, actorID, orderID, id, 1, &invalidIn); err != ErrOrderContainerInvalidArgument {
		t.Fatalf("expected ErrOrderContainerInvalidArgument for empty containerNo, got %v", err)
	}

	// Success path
	repo.updatedAudit = nil
	updated, err := usecase.Update(context.Background(), orgID, actorID, orderID, id, 1, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.ID != id || updated.ContainerNo != "COSU9876543" {
		t.Fatalf("unexpected updated container: %#v", updated)
	}
	if repo.updatedAudit == nil {
		t.Fatalf("expected non-nil updatedAudit passed to repo")
	}
	event := repo.updatedAudit
	if event.Action != "order.container.update" {
		t.Fatalf("expected action 'order.container.update', got %q", event.Action)
	}
	if event.Details["container.id"] != id.String() ||
		event.Details["order.id"] != orderID.String() ||
		event.Details["container.no"] != "COSU9876543" ||
		event.Details["container.spec_id"] != specID.String() {
		t.Fatalf("unexpected audit details: %#v", event.Details)
	}
}

func TestOrderContainerRemoveValidatesAndAudits(t *testing.T) {
	repo := &orderContainerRepoStub{}
	usecase := NewOrderContainerUsecase(repo)
	orgID, actorID, orderID, id := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	// Nil IDs
	if err := usecase.Remove(context.Background(), uuid.Nil, actorID, orderID, id, 1); err != ErrOrderContainerInvalidArgument {
		t.Fatalf("expected ErrOrderContainerInvalidArgument for nil orgID, got %v", err)
	}
	if err := usecase.Remove(context.Background(), orgID, uuid.Nil, orderID, id, 1); err != ErrOrderContainerInvalidArgument {
		t.Fatalf("expected ErrOrderContainerInvalidArgument for nil actorID, got %v", err)
	}
	if err := usecase.Remove(context.Background(), orgID, actorID, uuid.Nil, id, 1); err != ErrOrderContainerInvalidArgument {
		t.Fatalf("expected ErrOrderContainerInvalidArgument for nil orderID, got %v", err)
	}
	if err := usecase.Remove(context.Background(), orgID, actorID, orderID, uuid.Nil, 1); err != ErrOrderContainerInvalidArgument {
		t.Fatalf("expected ErrOrderContainerInvalidArgument for nil id, got %v", err)
	}
	if err := usecase.Remove(context.Background(), orgID, actorID, orderID, id, 0); err != ErrOrderContainerInvalidArgument {
		t.Fatalf("expected ErrOrderContainerInvalidArgument for zero expectedVersion, got %v", err)
	}

	// Success
	repo.removedAudit = nil
	if err := usecase.Remove(context.Background(), orgID, actorID, orderID, id, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.removed {
		t.Fatalf("expected repo.Remove to have been called")
	}
	if repo.removedAudit == nil {
		t.Fatalf("expected non-nil removedAudit passed to repo")
	}
	event := repo.removedAudit
	if event.Action != "order.container.remove" {
		t.Fatalf("expected action 'order.container.remove', got %q", event.Action)
	}
	if event.Details["container.id"] != id.String() ||
		event.Details["order.id"] != orderID.String() {
		t.Fatalf("unexpected audit details: %#v", event.Details)
	}
}
