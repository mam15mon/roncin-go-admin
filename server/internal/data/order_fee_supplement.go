package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
)

type orderFeeSupplementRepo struct {
	data *Data
}

func NewOrderFeeSupplementRepo(data *Data) biz.OrderFeeSupplementRequestRepo {
	return &orderFeeSupplementRepo{data: data}
}

func (r *orderFeeSupplementRepo) GetOrderRef(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.OrderFeeSupplementOrderRef, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	order, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFeeSupplementNotFound, nil)
	}
	return &biz.OrderFeeSupplementOrderRef{ID: order.ID, OrganizationID: order.OrganizationID, OrderNo: order.OrderNo, BusinessType: string(order.BusinessType)}, nil
}

var _ biz.OrderFeeSupplementRequestRepo = (*orderFeeSupplementRepo)(nil)
