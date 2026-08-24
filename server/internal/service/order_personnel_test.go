package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestProtoRoleToBizAndBack(t *testing.T) {
	pairs := []struct {
		proto v1.OrderPersonnelRole
		biz   biz.OrderPersonnelRole
	}{
		{v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_CREATOR, biz.OrderPersonnelRoleCreator},
		{v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_OPERATOR, biz.OrderPersonnelRoleOperator},
		{v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_SALES, biz.OrderPersonnelRoleSales},
		{v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_CUSTOMER_SERVICE, biz.OrderPersonnelRoleCustomerService},
		{v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_DOCUMENT, biz.OrderPersonnelRoleDocument},
		{v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_COMMERCIAL, biz.OrderPersonnelRoleCommercial},
		{v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_ASSOCIATE, biz.OrderPersonnelRoleAssociate},
		{v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_ASSOCIATE2, biz.OrderPersonnelRoleAssociate2},
	}

	for _, p := range pairs {
		b, err := protoRoleToBiz(p.proto)
		if err != nil {
			t.Fatalf("unexpected error converting %v to biz: %v", p.proto, err)
		}
		if b != p.biz {
			t.Fatalf("expected %v, got %v", p.biz, b)
		}
		back := bizRoleToProto(b)
		if back != p.proto {
			t.Fatalf("expected %v, got %v", p.proto, back)
		}
	}

	// Unspecified or invalid
	if _, err := protoRoleToBiz(v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_UNSPECIFIED); err != biz.ErrOrderPersonnelInvalidArgument {
		t.Fatalf("expected ErrOrderPersonnelInvalidArgument for unspecified role, got %v", err)
	}
	if _, err := protoRoleToBiz(v1.OrderPersonnelRole(999)); err != biz.ErrOrderPersonnelInvalidArgument {
		t.Fatalf("expected ErrOrderPersonnelInvalidArgument for unknown role, got %v", err)
	}
	if bizRoleToProto(biz.OrderPersonnelRole("UNKNOWN")) != v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_UNSPECIFIED {
		t.Fatalf("expected unspecified for unknown biz role")
	}
}

func TestOrderPersonnelToAPI(t *testing.T) {
	id := uuid.New()
	orderID := uuid.New()
	userID := uuid.New()
	organizationID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	item := &biz.OrderPersonnel{
		ID:             id,
		OrderID:        orderID,
		UserID:         userID,
		OrganizationID: organizationID,
		Role:           biz.OrderPersonnelRoleOperator,
		AssignedAt:     now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	api := orderPersonnelToAPI(item)
	if api.Id != id.String() || api.OrderId != orderID.String() || api.UserId != userID.String() || api.OrganizationId != organizationID.String() {
		t.Fatalf("unexpected IDs in api struct: %#v", api)
	}
	if api.Role != v1.OrderPersonnelRole_ORDER_PERSONNEL_ROLE_OPERATOR {
		t.Fatalf("unexpected role in api struct: %v", api.Role)
	}
	if api.AssignedAt != now.Format(time.RFC3339) || api.CreatedAt != now.Format(time.RFC3339) || api.UpdatedAt != now.Format(time.RFC3339) {
		t.Fatalf("unexpected timestamp formats: %#v", api)
	}
}
