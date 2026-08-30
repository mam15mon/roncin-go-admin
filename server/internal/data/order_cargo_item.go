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
		return mapEntError(err, biz.ErrOrderCargoItemNotFound, nil)
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
	var created *ent.OrderCargoItem
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
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
		var saveErr error
		created, saveErr = builder.Save(ctx)
		if saveErr != nil {
			return saveErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return orderCargoItemToBiz(created), nil
}

func (r *orderCargoItemRepo) Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *biz.OrderCargoItem, audit *biz.AuditEvent) (*biz.OrderCargoItem, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	var updated *ent.OrderCargoItem
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, queryErr := tx.OrderCargoItem.Query().
			Where(
				ordercargoitement.IDEQ(id),
				ordercargoitement.OrderIDEQ(orderID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderCargoItemNotFound, nil)
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
		var saveErr error
		updated, saveErr = builder.Save(ctx)
		if saveErr != nil {
			return saveErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return orderCargoItemToBiz(updated), nil
}

func (r *orderCargoItemRepo) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, audit *biz.AuditEvent) error {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return err
	}
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		n, deleteErr := tx.OrderCargoItem.Delete().
			Where(
				ordercargoitement.IDEQ(id),
				ordercargoitement.OrderIDEQ(orderID),
			).
			Exec(ctx)
		if deleteErr != nil {
			return deleteErr
		}
		if n == 0 {
			return biz.ErrOrderCargoItemNotFound
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
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
