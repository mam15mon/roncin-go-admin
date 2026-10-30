package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
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
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	if _, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx); err != nil {
		return mapEntError(err, biz.ErrOrderPersonnelNotFound, nil)
	}
	return nil
}

func (r *orderPersonnelRepo) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderPersonnel, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	items, err := client.OrderPersonnel.Query().
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

func orderPersonnelToBiz(item *ent.OrderPersonnel) *biz.OrderPersonnel {
	return &biz.OrderPersonnel{
		ID:             item.ID,
		OrderID:        item.OrderID,
		UserID:         item.UserID,
		OrganizationID: item.OrganizationID,
		Role:           biz.OrderPersonnelRole(item.Role),
		AssignedAt:     item.AssignedAt,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

var _ biz.OrderPersonnelRepo = (*orderPersonnelRepo)(nil)
