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

func (r *orderCargoItemRepo) Add(ctx context.Context, organizationID, orderID uuid.UUID, input *biz.OrderCargoItem, audit *biz.AuditEvent) (*biz.OrderCargoItem, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	builder := tx.OrderCargoItem.Create().
		SetID(input.ID).
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
		_ = tx.Rollback()
		return nil, err
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return orderCargoItemToBiz(created), nil
}

func (r *orderCargoItemRepo) Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *biz.OrderCargoItem, audit *biz.AuditEvent) (*biz.OrderCargoItem, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	item, err := tx.OrderCargoItem.Query().
		Where(
			ordercargoitement.IDEQ(id),
			ordercargoitement.OrderIDEQ(orderID),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderCargoItemNotFound
		}
		return nil, err
	}
	builder := tx.OrderCargoItem.UpdateOne(item).
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
		_ = tx.Rollback()
		return nil, err
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return orderCargoItemToBiz(updated), nil
}

func (r *orderCargoItemRepo) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, audit *biz.AuditEvent) error {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	n, err := tx.OrderCargoItem.Delete().
		Where(
			ordercargoitement.IDEQ(id),
			ordercargoitement.OrderIDEQ(orderID),
		).
		Exec(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if n == 0 {
		_ = tx.Rollback()
		return biz.ErrOrderCargoItemNotFound
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
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
