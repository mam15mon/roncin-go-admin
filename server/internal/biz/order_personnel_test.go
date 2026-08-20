package biz

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type orderPersonnelRepoStub struct {
	assigned *OrderPersonnel
	removed  bool
}

func (s *orderPersonnelRepoStub) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderPersonnel, error) {
	return []*OrderPersonnel{
		{
			ID:         uuid.New(),
			OrderID:    orderID,
			UserID:     uuid.New(),
			Role:       OrderPersonnelRoleOperator,
			AssignedAt: time.Now(),
		},
	}, nil
}

func (s *orderPersonnelRepoStub) Assign(ctx context.Context, organizationID, orderID, userID uuid.UUID, role OrderPersonnelRole) (*OrderPersonnel, error) {
	s.assigned = &OrderPersonnel{
		ID:         uuid.New(),
		OrderID:    orderID,
		UserID:     userID,
		Role:       role,
		AssignedAt: time.Now(),
	}
	return s.assigned, nil
}

func (s *orderPersonnelRepoStub) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID) error {
	s.removed = true
	return nil
}

func TestOrderPersonnelRoleValid(t *testing.T) {
	validRoles := []OrderPersonnelRole{
		OrderPersonnelRoleCreator,
		OrderPersonnelRoleOperator,
		OrderPersonnelRoleSales,
		OrderPersonnelRoleCustomerService,
		OrderPersonnelRoleDocument,
		OrderPersonnelRoleCommercial,
		OrderPersonnelRoleAssociate,
		OrderPersonnelRoleAssociate2,
	}
	for _, r := range validRoles {
		if !r.Valid() {
			t.Fatalf("expected role %s to be valid", r)
		}
	}
	invalidRoles := []OrderPersonnelRole{
		"",
		"INVALID",
		"admin",
	}
	for _, r := range invalidRoles {
		if r.Valid() {
			t.Fatalf("expected role %s to be invalid", r)
		}
	}
}

func TestOrderPersonnelAssignValidatesAndAudits(t *testing.T) {
	repo := &orderPersonnelRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewOrderPersonnelUsecase(repo, audit)
	organizationID, actorID, orderID, userID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	// Empty IDs
	if _, err := usecase.Assign(context.Background(), uuid.Nil, actorID, orderID, userID, OrderPersonnelRoleSales); err != ErrOrderPersonnelInvalidArgument {
		t.Fatalf("expected ErrOrderPersonnelInvalidArgument for nil orgID, got %v", err)
	}
	if _, err := usecase.Assign(context.Background(), organizationID, uuid.Nil, orderID, userID, OrderPersonnelRoleSales); err != ErrOrderPersonnelInvalidArgument {
		t.Fatalf("expected ErrOrderPersonnelInvalidArgument for nil actorID, got %v", err)
	}
	if _, err := usecase.Assign(context.Background(), organizationID, actorID, uuid.Nil, userID, OrderPersonnelRoleSales); err != ErrOrderPersonnelInvalidArgument {
		t.Fatalf("expected ErrOrderPersonnelInvalidArgument for nil orderID, got %v", err)
	}
	if _, err := usecase.Assign(context.Background(), organizationID, actorID, orderID, uuid.Nil, OrderPersonnelRoleSales); err != ErrOrderPersonnelInvalidArgument {
		t.Fatalf("expected ErrOrderPersonnelInvalidArgument for nil userID, got %v", err)
	}

	// Invalid role
	if _, err := usecase.Assign(context.Background(), organizationID, actorID, orderID, userID, OrderPersonnelRole("UNKNOWN")); err != ErrOrderPersonnelInvalidArgument {
		t.Fatalf("expected ErrOrderPersonnelInvalidArgument for invalid role, got %v", err)
	}

	// Success
	created, err := usecase.Assign(context.Background(), organizationID, actorID, orderID, userID, OrderPersonnelRoleSales)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created == nil || created.Role != OrderPersonnelRoleSales || created.OrderID != orderID || created.UserID != userID {
		t.Fatalf("unexpected created personnel: %#v", created)
	}
	if len(audit.events) != 1 || audit.events[0].Action != "order.personnel.assign" {
		t.Fatalf("unexpected audit events: %#v", audit.events)
	}
}

func TestOrderPersonnelRemoveValidatesAndAudits(t *testing.T) {
	repo := &orderPersonnelRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewOrderPersonnelUsecase(repo, audit)
	organizationID, actorID, orderID, id := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	// Empty IDs
	if err := usecase.Remove(context.Background(), uuid.Nil, actorID, orderID, id); err != ErrOrderPersonnelInvalidArgument {
		t.Fatalf("expected ErrOrderPersonnelInvalidArgument, got %v", err)
	}
	if err := usecase.Remove(context.Background(), organizationID, uuid.Nil, orderID, id); err != ErrOrderPersonnelInvalidArgument {
		t.Fatalf("expected ErrOrderPersonnelInvalidArgument, got %v", err)
	}
	if err := usecase.Remove(context.Background(), organizationID, actorID, uuid.Nil, id); err != ErrOrderPersonnelInvalidArgument {
		t.Fatalf("expected ErrOrderPersonnelInvalidArgument, got %v", err)
	}
	if err := usecase.Remove(context.Background(), organizationID, actorID, orderID, uuid.Nil); err != ErrOrderPersonnelInvalidArgument {
		t.Fatalf("expected ErrOrderPersonnelInvalidArgument, got %v", err)
	}

	// Success
	if err := usecase.Remove(context.Background(), organizationID, actorID, orderID, id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.removed {
		t.Fatalf("expected repo.Remove to have been called")
	}
	if len(audit.events) != 1 || audit.events[0].Action != "order.personnel.remove" {
		t.Fatalf("unexpected audit events: %#v", audit.events)
	}
}

func TestOrderPersonnelListValidates(t *testing.T) {
	repo := &orderPersonnelRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewOrderPersonnelUsecase(repo, audit)

	if _, err := usecase.List(context.Background(), uuid.Nil, uuid.New()); err != ErrOrderPersonnelInvalidArgument {
		t.Fatalf("expected ErrOrderPersonnelInvalidArgument, got %v", err)
	}
	if _, err := usecase.List(context.Background(), uuid.New(), uuid.Nil); err != ErrOrderPersonnelInvalidArgument {
		t.Fatalf("expected ErrOrderPersonnelInvalidArgument, got %v", err)
	}

	items, err := usecase.List(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

var _ OrderPersonnelRepo = (*orderPersonnelRepoStub)(nil)
