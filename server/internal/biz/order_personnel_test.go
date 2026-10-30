package biz

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type orderPersonnelRepoStub struct{}

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

func TestOrderPersonnelListValidates(t *testing.T) {
	repo := &orderPersonnelRepoStub{}
	usecase := NewOrderPersonnelUsecase(repo)

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
