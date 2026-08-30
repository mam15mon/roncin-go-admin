package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	masterdataitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordercontainerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainer"
	ordershippingdocumentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordershippingdocument"
)

type orderContainerRepo struct {
	data *Data
}

func NewOrderContainerRepo(data *Data) biz.OrderContainerRepo {
	return &orderContainerRepo{data: data}
}

func (r *orderContainerRepo) order(ctx context.Context, organizationID, orderID uuid.UUID) (*ent.Order, error) {
	item, err := r.data.db.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderContainerNotFound, nil)
	}
	return item, nil
}

func validateOrderSupportsContainers(item *ent.Order) error {
	if item.ShipmentType == nil || *item.ShipmentType != orderent.ShipmentTypeFCL {
		return biz.ErrOrderContainerShipmentType
	}
	return nil
}

func (r *orderContainerRepo) validateContainerSpec(ctx context.Context, organizationID, specID uuid.UUID) error {
	count, err := r.data.db.MasterDataItem.Query().
		Where(
			masterdataitement.IDEQ(specID),
			masterdataitement.OrganizationIDEQ(organizationID),
			masterdataitement.KindEQ(masterdataitement.KindContainerSpec),
			masterdataitement.EnabledEQ(true),
		).
		Count(ctx)
	if err != nil {
		return err
	}
	if count != 1 {
		return biz.ErrOrderContainerSpecInvalid
	}
	return nil
}

func (r *orderContainerRepo) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderContainer, error) {
	if _, err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	items, err := r.data.db.OrderContainer.Query().
		Where(ordercontainerent.OrderIDEQ(orderID)).
		Order(ordercontainerent.ByContainerNo()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.OrderContainer, 0, len(items))
	for _, item := range items {
		result = append(result, orderContainerToBiz(item))
	}
	return result, nil
}

func (r *orderContainerRepo) Add(ctx context.Context, organizationID, orderID uuid.UUID, input *biz.OrderContainer, audit *biz.AuditEvent) (*biz.OrderContainer, error) {
	item, err := r.order(ctx, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	if err := validateOrderSupportsContainers(item); err != nil {
		return nil, err
	}
	if err := r.validateContainerSpec(ctx, organizationID, input.ContainerSpecID); err != nil {
		return nil, err
	}
	if err := r.validateShippingDocument(ctx, orderID, input.ShippingDocumentID); err != nil {
		return nil, err
	}
	var created *ent.OrderContainer
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		builder := tx.OrderContainer.Create().
			SetID(input.ID).
			SetOrderID(orderID).
			SetContainerNo(input.ContainerNo).
			SetContainerSpecID(input.ContainerSpecID).
			SetGrossWeightKg(input.GrossWeightKg).
			SetVolumeCbm(input.VolumeCbm)
		if input.ShippingDocumentID != nil {
			builder.SetShippingDocumentID(*input.ShippingDocumentID)
		}
		if input.SealNo != nil {
			builder.SetSealNo(*input.SealNo)
		}
		if input.Note != nil {
			builder.SetNote(*input.Note)
		}
		var saveErr error
		created, saveErr = builder.Save(ctx)
		if saveErr != nil {
			return mapEntConstraint(saveErr, "ordercontainer_order_id_container_no", biz.ErrOrderContainerExists)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return orderContainerToBiz(created), nil
}

// validateShippingDocument 校验可选的提单引用必须挂在同一订单下。
func (r *orderContainerRepo) validateShippingDocument(ctx context.Context, orderID uuid.UUID, documentID *uuid.UUID) error {
	if documentID == nil {
		return nil
	}
	exists, err := r.data.db.OrderShippingDocument.Query().
		Where(
			ordershippingdocumentent.IDEQ(*documentID),
			ordershippingdocumentent.OrderIDEQ(orderID),
		).
		Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return biz.ErrOrderShippingDocumentNotFound
	}
	return nil
}

func (r *orderContainerRepo) Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *biz.OrderContainer, audit *biz.AuditEvent) (*biz.OrderContainer, error) {
	order, err := r.order(ctx, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	if err := validateOrderSupportsContainers(order); err != nil {
		return nil, err
	}
	if err := r.validateContainerSpec(ctx, organizationID, input.ContainerSpecID); err != nil {
		return nil, err
	}
	if err := r.validateShippingDocument(ctx, orderID, input.ShippingDocumentID); err != nil {
		return nil, err
	}
	var updated *ent.OrderContainer
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, queryErr := tx.OrderContainer.Query().
			Where(
				ordercontainerent.IDEQ(id),
				ordercontainerent.OrderIDEQ(orderID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderContainerNotFound, nil)
		}
		builder := tx.OrderContainer.UpdateOne(item).
			SetContainerNo(input.ContainerNo).
			SetContainerSpecID(input.ContainerSpecID).
			SetGrossWeightKg(input.GrossWeightKg).
			SetVolumeCbm(input.VolumeCbm)
		if input.ShippingDocumentID != nil {
			builder.SetShippingDocumentID(*input.ShippingDocumentID)
		} else {
			builder.ClearShippingDocumentID()
		}
		if input.SealNo != nil {
			builder.SetSealNo(*input.SealNo)
		} else {
			builder.ClearSealNo()
		}
		if input.Note != nil {
			builder.SetNote(*input.Note)
		} else {
			builder.ClearNote()
		}
		var saveErr error
		updated, saveErr = builder.Save(ctx)
		if saveErr != nil {
			return mapEntConstraint(saveErr, "ordercontainer_order_id_container_no", biz.ErrOrderContainerExists)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return orderContainerToBiz(updated), nil
}

func (r *orderContainerRepo) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, audit *biz.AuditEvent) error {
	if _, err := r.order(ctx, organizationID, orderID); err != nil {
		return err
	}
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		n, deleteErr := tx.OrderContainer.Delete().
			Where(
				ordercontainerent.IDEQ(id),
				ordercontainerent.OrderIDEQ(orderID),
			).
			Exec(ctx)
		if deleteErr != nil {
			return deleteErr
		}
		if n == 0 {
			return biz.ErrOrderContainerNotFound
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

func orderContainerToBiz(item *ent.OrderContainer) *biz.OrderContainer {
	result := &biz.OrderContainer{
		ID:                 item.ID,
		OrderID:            item.OrderID,
		ContainerNo:        item.ContainerNo,
		ContainerSpecID:    item.ContainerSpecID,
		ShippingDocumentID: item.ShippingDocumentID,
		GrossWeightKg:      item.GrossWeightKg,
		VolumeCbm:          item.VolumeCbm,
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	}
	if item.SealNo != "" {
		v := item.SealNo
		result.SealNo = &v
	}
	if item.Note != "" {
		v := item.Note
		result.Note = &v
	}
	return result
}

var _ biz.OrderContainerRepo = (*orderContainerRepo)(nil)
