package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderpersonnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
)

type orderPersonnelRepo struct {
	data *Data
}

func NewOrderPersonnelRepo(data *Data) biz.OrderPersonnelRepo {
	return &orderPersonnelRepo{data: data}
}

func (r *orderPersonnelRepo) order(ctx context.Context, organizationID, orderID uuid.UUID) error {
	if _, err := r.data.db.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx); err != nil {
		if ent.IsNotFound(err) {
			return biz.ErrOrderPersonnelNotFound
		}
		return err
	}
	return nil
}

func (r *orderPersonnelRepo) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderPersonnel, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	items, err := r.data.db.OrderPersonnel.Query().
		Where(orderpersonnelent.OrderIDEQ(orderID)).
		Order(orderpersonnelent.ByAssignedAt(), orderpersonnelent.ByRole()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.OrderPersonnel, 0, len(items))
	for _, item := range items {
		result = append(result, orderPersonnelToBiz(item))
	}
	return result, nil
}

func (r *orderPersonnelRepo) Assign(ctx context.Context, organizationID, orderID, userID uuid.UUID, role biz.OrderPersonnelRole) (*biz.OrderPersonnel, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	m, err := r.data.db.Membership.Query().
		Where(
			membership.OrganizationIDEQ(organizationID),
			membership.UserIDEQ(userID),
			membership.EnabledEQ(true),
		).
		WithUser().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderPersonnelUserInvalid
		}
		return nil, err
	}
	user, err := m.Edges.UserOrErr()
	if err != nil || !user.Enabled {
		return nil, biz.ErrOrderPersonnelUserInvalid
	}
	created, err := r.data.db.OrderPersonnel.Create().
		SetOrderID(orderID).
		SetUserID(userID).
		SetRole(orderpersonnelent.Role(role)).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, biz.ErrOrderPersonnelExists
		}
		return nil, err
	}
	return orderPersonnelToBiz(created), nil
}

func (r *orderPersonnelRepo) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID) error {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return err
	}
	n, err := r.data.db.OrderPersonnel.Delete().
		Where(
			orderpersonnelent.IDEQ(id),
			orderpersonnelent.OrderIDEQ(orderID),
		).
		Exec(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return biz.ErrOrderPersonnelNotFound
	}
	return nil
}

func orderPersonnelToBiz(item *ent.OrderPersonnel) *biz.OrderPersonnel {
	return &biz.OrderPersonnel{
		ID:         item.ID,
		OrderID:    item.OrderID,
		UserID:     item.UserID,
		Role:       biz.OrderPersonnelRole(item.Role),
		AssignedAt: item.AssignedAt,
		CreatedAt:  item.CreatedAt,
		UpdatedAt:  item.UpdatedAt,
	}
}

var _ biz.OrderPersonnelRepo = (*orderPersonnelRepo)(nil)
