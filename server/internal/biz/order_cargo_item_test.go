package biz

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type orderCargoItemRepoStub struct {
	added        *OrderCargoItem
	addedAudit   *AuditEvent
	updated      *OrderCargoItem
	updatedAudit *AuditEvent
	removed      bool
	removedAudit *AuditEvent
}

func (s *orderCargoItemRepoStub) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderCargoItem, error) {
	return []*OrderCargoItem{
		{
			ID:            uuid.New(),
			OrderID:       orderID,
			CargoName:     "Electronics",
			PackageCount:  100,
			GrossWeightKg: 1500.5,
			VolumeCbm:     12.3,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
	}, nil
}

func (s *orderCargoItemRepoStub) Add(ctx context.Context, organizationID, orderID uuid.UUID, input *OrderCargoItem, audit *AuditEvent) (*OrderCargoItem, error) {
	s.addedAudit = audit
	s.added = &OrderCargoItem{
		ID:            input.ID,
		OrderID:       orderID,
		CargoName:     input.CargoName,
		PackageCount:  input.PackageCount,
		GrossWeightKg: input.GrossWeightKg,
		VolumeCbm:     input.VolumeCbm,
		NetWeightKg:   input.NetWeightKg,
		Note:          input.Note,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	return s.added, nil
}

func (s *orderCargoItemRepoStub) Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *OrderCargoItem, audit *AuditEvent) (*OrderCargoItem, error) {
	s.updatedAudit = audit
	s.updated = &OrderCargoItem{
		ID:            id,
		OrderID:       orderID,
		CargoName:     input.CargoName,
		PackageCount:  input.PackageCount,
		GrossWeightKg: input.GrossWeightKg,
		VolumeCbm:     input.VolumeCbm,
		NetWeightKg:   input.NetWeightKg,
		Note:          input.Note,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	return s.updated, nil
}

func (s *orderCargoItemRepoStub) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, audit *AuditEvent) error {
	s.removed = true
	s.removedAudit = audit
	return nil
}

var _ OrderCargoItemRepo = (*orderCargoItemRepoStub)(nil)

func floatPtr(v float64) *float64 {
	return &v
}

func TestOrderCargoItemListValidates(t *testing.T) {
	repo := &orderCargoItemRepoStub{}
	usecase := NewOrderCargoItemUsecase(repo)

	if _, err := usecase.List(context.Background(), uuid.Nil, uuid.New()); err != ErrOrderCargoItemInvalidArgument {
		t.Fatalf("expected ErrOrderCargoItemInvalidArgument for nil orgID, got %v", err)
	}
	if _, err := usecase.List(context.Background(), uuid.New(), uuid.Nil); err != ErrOrderCargoItemInvalidArgument {
		t.Fatalf("expected ErrOrderCargoItemInvalidArgument for nil orderID, got %v", err)
	}

	items, err := usecase.List(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func TestOrderCargoItemAddValidatesAndAudits(t *testing.T) {
	repo := &orderCargoItemRepoStub{}
	usecase := NewOrderCargoItemUsecase(repo)
	orgID, actorID, orderID := uuid.New(), uuid.New(), uuid.New()

	baseValidInput := func() *OrderCargoItem {
		return &OrderCargoItem{
			CargoName:     "Auto Parts",
			PackageCount:  50,
			GrossWeightKg: 1200.5,
			VolumeCbm:     8.5,
			NetWeightKg:   floatPtr(1100.0),
			Note:          stringPtr("fragile components"),
		}
	}

	// Nil IDs
	if _, err := usecase.Add(context.Background(), uuid.Nil, actorID, orderID, baseValidInput()); err != ErrOrderCargoItemInvalidArgument {
		t.Fatalf("expected ErrOrderCargoItemInvalidArgument for nil orgID, got %v", err)
	}
	if _, err := usecase.Add(context.Background(), orgID, uuid.Nil, orderID, baseValidInput()); err != ErrOrderCargoItemInvalidArgument {
		t.Fatalf("expected ErrOrderCargoItemInvalidArgument for nil actorID, got %v", err)
	}
	if _, err := usecase.Add(context.Background(), orgID, actorID, uuid.Nil, baseValidInput()); err != ErrOrderCargoItemInvalidArgument {
		t.Fatalf("expected ErrOrderCargoItemInvalidArgument for nil orderID, got %v", err)
	}

	// Nil input
	if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, nil); err != ErrOrderCargoItemInvalidArgument {
		t.Fatalf("expected ErrOrderCargoItemInvalidArgument for nil input, got %v", err)
	}

	// CargoName validation
	invalidNames := []string{"", "   ", strings.Repeat("C", 201)}
	for _, name := range invalidNames {
		in := baseValidInput()
		in.CargoName = name
		if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, in); err != ErrOrderCargoItemInvalidArgument {
			t.Fatalf("expected ErrOrderCargoItemInvalidArgument for cargoName %q, got %v", name, err)
		}
	}

	// PackageCount validation
	for _, pc := range []int{0, -1, -50} {
		in := baseValidInput()
		in.PackageCount = pc
		if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, in); err != ErrOrderCargoItemInvalidArgument {
			t.Fatalf("expected ErrOrderCargoItemInvalidArgument for packageCount %d, got %v", pc, err)
		}
	}

	// GrossWeightKg validation
	for _, gw := range []float64{0, -1, -10.5} {
		in := baseValidInput()
		in.GrossWeightKg = gw
		if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, in); err != ErrOrderCargoItemInvalidArgument {
			t.Fatalf("expected ErrOrderCargoItemInvalidArgument for GrossWeightKg %f, got %v", gw, err)
		}
	}

	// VolumeCbm validation
	for _, v := range []float64{0, -1, -5.2} {
		in := baseValidInput()
		in.VolumeCbm = v
		if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, in); err != ErrOrderCargoItemInvalidArgument {
			t.Fatalf("expected ErrOrderCargoItemInvalidArgument for VolumeCbm %f, got %v", v, err)
		}
	}

	// NetWeightKg validation (> 0 when provided)
	for _, nw := range []float64{0, -1, -100} {
		in := baseValidInput()
		in.NetWeightKg = floatPtr(nw)
		if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, in); err != ErrOrderCargoItemInvalidArgument {
			t.Fatalf("expected ErrOrderCargoItemInvalidArgument for NetWeightKg %f, got %v", nw, err)
		}
	}

	// Note too long
	{
		in := baseValidInput()
		in.Note = stringPtr(strings.Repeat("N", 501))
		if _, err := usecase.Add(context.Background(), orgID, actorID, orderID, in); err != ErrOrderCargoItemInvalidArgument {
			t.Fatalf("expected ErrOrderCargoItemInvalidArgument for long note, got %v", err)
		}
	}

	// Blank optional Note normalized to nil
	{
		in := baseValidInput()
		in.CargoName = "  Machine Parts  "
		in.Note = stringPtr("   ")
		created, err := usecase.Add(context.Background(), orgID, actorID, orderID, in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created.CargoName != "Machine Parts" {
			t.Fatalf("expected trimmed cargoName, got %q", created.CargoName)
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
	if created == nil || created.CargoName != "Auto Parts" {
		t.Fatalf("unexpected created cargo item: %#v", created)
	}
	if repo.addedAudit == nil {
		t.Fatalf("expected non-nil addedAudit passed to repo")
	}
	event := repo.addedAudit
	if event.Action != "order.cargo_item.add" {
		t.Fatalf("expected action 'order.cargo_item.add', got %q", event.Action)
	}
	if event.Details["cargo_item.id"] != created.ID.String() ||
		event.Details["order.id"] != orderID.String() ||
		event.Details["cargo_item.name"] != "Auto Parts" {
		t.Fatalf("unexpected audit details: %#v", event.Details)
	}
}

func TestOrderCargoItemUpdateValidatesAndAudits(t *testing.T) {
	repo := &orderCargoItemRepoStub{}
	usecase := NewOrderCargoItemUsecase(repo)
	orgID, actorID, orderID, id := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	input := &OrderCargoItem{
		CargoName:     "Medical Supplies",
		PackageCount:  20,
		GrossWeightKg: 500,
		VolumeCbm:     3.0,
		NetWeightKg:   floatPtr(450),
		Note:          stringPtr("keep cool"),
	}

	// Nil IDs
	if _, err := usecase.Update(context.Background(), uuid.Nil, actorID, orderID, id, input); err != ErrOrderCargoItemInvalidArgument {
		t.Fatalf("expected ErrOrderCargoItemInvalidArgument for nil orgID, got %v", err)
	}
	if _, err := usecase.Update(context.Background(), orgID, uuid.Nil, orderID, id, input); err != ErrOrderCargoItemInvalidArgument {
		t.Fatalf("expected ErrOrderCargoItemInvalidArgument for nil actorID, got %v", err)
	}
	if _, err := usecase.Update(context.Background(), orgID, actorID, uuid.Nil, id, input); err != ErrOrderCargoItemInvalidArgument {
		t.Fatalf("expected ErrOrderCargoItemInvalidArgument for nil orderID, got %v", err)
	}
	if _, err := usecase.Update(context.Background(), orgID, actorID, orderID, uuid.Nil, input); err != ErrOrderCargoItemInvalidArgument {
		t.Fatalf("expected ErrOrderCargoItemInvalidArgument for nil id, got %v", err)
	}

	// Invalid input
	invalidIn := *input
	invalidIn.PackageCount = 0
	if _, err := usecase.Update(context.Background(), orgID, actorID, orderID, id, &invalidIn); err != ErrOrderCargoItemInvalidArgument {
		t.Fatalf("expected ErrOrderCargoItemInvalidArgument for invalid packageCount, got %v", err)
	}

	// Success path
	repo.updatedAudit = nil
	updated, err := usecase.Update(context.Background(), orgID, actorID, orderID, id, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.ID != id || updated.CargoName != "Medical Supplies" {
		t.Fatalf("unexpected updated cargo item: %#v", updated)
	}
	if repo.updatedAudit == nil {
		t.Fatalf("expected non-nil updatedAudit passed to repo")
	}
	event := repo.updatedAudit
	if event.Action != "order.cargo_item.update" {
		t.Fatalf("expected action 'order.cargo_item.update', got %q", event.Action)
	}
	if event.Details["cargo_item.id"] != id.String() ||
		event.Details["order.id"] != orderID.String() ||
		event.Details["cargo_item.name"] != "Medical Supplies" {
		t.Fatalf("unexpected audit details: %#v", event.Details)
	}
}

func TestOrderCargoItemRemoveValidatesAndAudits(t *testing.T) {
	repo := &orderCargoItemRepoStub{}
	usecase := NewOrderCargoItemUsecase(repo)
	orgID, actorID, orderID, id := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	// Nil IDs
	if err := usecase.Remove(context.Background(), uuid.Nil, actorID, orderID, id); err != ErrOrderCargoItemInvalidArgument {
		t.Fatalf("expected ErrOrderCargoItemInvalidArgument for nil orgID, got %v", err)
	}
	if err := usecase.Remove(context.Background(), orgID, uuid.Nil, orderID, id); err != ErrOrderCargoItemInvalidArgument {
		t.Fatalf("expected ErrOrderCargoItemInvalidArgument for nil actorID, got %v", err)
	}
	if err := usecase.Remove(context.Background(), orgID, actorID, uuid.Nil, id); err != ErrOrderCargoItemInvalidArgument {
		t.Fatalf("expected ErrOrderCargoItemInvalidArgument for nil orderID, got %v", err)
	}
	if err := usecase.Remove(context.Background(), orgID, actorID, orderID, uuid.Nil); err != ErrOrderCargoItemInvalidArgument {
		t.Fatalf("expected ErrOrderCargoItemInvalidArgument for nil id, got %v", err)
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
	if event.Action != "order.cargo_item.remove" {
		t.Fatalf("expected action 'order.cargo_item.remove', got %q", event.Action)
	}
	if event.Details["cargo_item.id"] != id.String() ||
		event.Details["order.id"] != orderID.String() {
		t.Fatalf("unexpected audit details: %#v", event.Details)
	}
}
