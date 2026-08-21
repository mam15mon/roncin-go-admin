package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordercargoitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargoitem"
)

type orderCargoItemRepo struct {
	data *Data
}

func NewOrderCargoItemRepo(data *Data) biz.OrderCargoItemRepo {
	return &orderCargoItemRepo{data: data}
}

func (r *orderCargoItemRepo) order(ctx context.Context, organizationID, orderID uuid.UUID) error {
	if _, err := r.data.db.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx); err != nil {
		if ent.IsNotFound(err) {
			return biz.ErrOrderCargoItemNotFound
		}
		return err
	}
	return nil
}

func (r *orderCargoItemRepo) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderCargoItem, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	items, err := r.data.db.OrderCargoItem.Query().
		Where(ordercargoitement.OrderIDEQ(orderID)).
		Order(ordercargoitement.ByCreatedAt(), ordercargoitement.ByCargoName()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.OrderCargoItem, 0, len(items))
	for _, item := range items {
		result = append(result, orderCargoItemToBiz(item))
	}
	return result, nil
}

func (r *orderCargoItemRepo) Add(ctx context.Context, organizationID, orderID uuid.UUID, input *biz.OrderCargoItem) (*biz.OrderCargoItem, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	builder := r.data.db.OrderCargoItem.Create().
		SetOrderID(orderID).
		SetCargoName(input.CargoName).
		SetPackageCount(input.PackageCount).
		SetGrossWeightKg(input.GrossWeightKg).
		SetVolumeCbm(input.VolumeCbm)
	if input.NetWeightKg != nil {
		builder.SetNetWeightKg(*input.NetWeightKg)
	}
	if input.Note != nil {
		builder.SetNote(*input.Note)
	}
	created, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}
	return orderCargoItemToBiz(created), nil
}

func (r *orderCargoItemRepo) Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *biz.OrderCargoItem) (*biz.OrderCargoItem, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	item, err := r.data.db.OrderCargoItem.Query().
		Where(
			ordercargoitement.IDEQ(id),
			ordercargoitement.OrderIDEQ(orderID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderCargoItemNotFound
		}
		return nil, err
	}
	builder := item.Update().
		SetCargoName(input.CargoName).
		SetPackageCount(input.PackageCount).
		SetGrossWeightKg(input.GrossWeightKg).
		SetVolumeCbm(input.VolumeCbm)
	if input.NetWeightKg != nil {
		builder.SetNetWeightKg(*input.NetWeightKg)
	} else {
		builder.ClearNetWeightKg()
	}
	if input.Note != nil {
		builder.SetNote(*input.Note)
	} else {
		builder.ClearNote()
	}
	updated, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}
	return orderCargoItemToBiz(updated), nil
}

func (r *orderCargoItemRepo) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID) error {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return err
	}
	n, err := r.data.db.OrderCargoItem.Delete().
		Where(
			ordercargoitement.IDEQ(id),
			ordercargoitement.OrderIDEQ(orderID),
		).
		Exec(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return biz.ErrOrderCargoItemNotFound
	}
	return nil
}

func orderCargoItemToBiz(item *ent.OrderCargoItem) *biz.OrderCargoItem {
	result := &biz.OrderCargoItem{
		ID:            item.ID,
		OrderID:       item.OrderID,
		CargoName:     item.CargoName,
		PackageCount:  item.PackageCount,
		GrossWeightKg: item.GrossWeightKg,
		VolumeCbm:     item.VolumeCbm,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
	if item.NetWeightKg > 0 {
		v := item.NetWeightKg
		result.NetWeightKg = &v
	}
	if item.Note != "" {
		v := item.Note
		result.Note = &v
	}
	return result
}

var _ biz.OrderCargoItemRepo = (*orderCargoItemRepo)(nil)
