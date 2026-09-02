package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordershippingdocumentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordershippingdocument"
)

type orderShippingDocumentRepo struct {
	data *Data
}

func NewOrderShippingDocumentRepo(data *Data) biz.OrderShippingDocumentRepo {
	return &orderShippingDocumentRepo{data: data}
}

func (r *orderShippingDocumentRepo) order(ctx context.Context, organizationID, orderID uuid.UUID) (*ent.Order, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderShippingDocumentNotFound, nil)
	}
	if item.BusinessType == orderent.BusinessTypeSE {
		return nil, biz.ErrSeaShippingDocumentsDeprecated
	}
	return item, nil
}

func (r *orderShippingDocumentRepo) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderShippingDocument, error) {
	if _, err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	items, err := client.OrderShippingDocument.Query().
		Where(ordershippingdocumentent.OrderIDEQ(orderID)).
		Order(ordershippingdocumentent.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.OrderShippingDocument, 0, len(items))
	for _, item := range items {
		result = append(result, orderShippingDocumentToBiz(item))
	}
	return result, nil
}

func (r *orderShippingDocumentRepo) Add(ctx context.Context, organizationID, orderID uuid.UUID, input *biz.OrderShippingDocument, audit *biz.AuditEvent) (*biz.OrderShippingDocument, error) {
	if _, err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	var created *ent.OrderShippingDocument
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		builder := tx.OrderShippingDocument.Create().
			SetID(input.ID).
			SetOrderID(orderID).
			SetHouseNo(input.HouseNo).
			SetStatus(ordershippingdocumentent.StatusDRAFT)
		if input.ReleaseType != nil {
			builder.SetReleaseType(*input.ReleaseType)
		}
		if input.Note != nil {
			builder.SetNote(*input.Note)
		}
		var saveErr error
		created, saveErr = builder.Save(ctx)
		if saveErr != nil {
			return mapEntConstraint(saveErr, "ordershippingdocument_order_id_house_no", biz.ErrOrderShippingDocumentExists)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return orderShippingDocumentToBiz(created), nil
}

func (r *orderShippingDocumentRepo) Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *biz.OrderShippingDocument, audit *biz.AuditEvent) (*biz.OrderShippingDocument, error) {
	if _, err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	var updated *ent.OrderShippingDocument
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, queryErr := tx.OrderShippingDocument.Query().
			Where(
				ordershippingdocumentent.IDEQ(id),
				ordershippingdocumentent.OrderIDEQ(orderID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderShippingDocumentNotFound, nil)
		}
		if item.Status != ordershippingdocumentent.StatusDRAFT && item.Status != ordershippingdocumentent.StatusCONFIRMED {
			return biz.ErrOrderShippingDocumentInvalidStatus
		}
		builder := tx.OrderShippingDocument.UpdateOne(item).
			SetHouseNo(input.HouseNo)
		if input.ReleaseType != nil {
			builder.SetReleaseType(*input.ReleaseType)
		} else {
			builder.ClearReleaseType()
		}
		if input.Note != nil {
			builder.SetNote(*input.Note)
		} else {
			builder.ClearNote()
		}
		var saveErr error
		updated, saveErr = builder.Save(ctx)
		if saveErr != nil {
			return mapEntConstraint(saveErr, "ordershippingdocument_order_id_house_no", biz.ErrOrderShippingDocumentExists)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return orderShippingDocumentToBiz(updated), nil
}

func (r *orderShippingDocumentRepo) Transition(ctx context.Context, organizationID, orderID, id uuid.UUID, from, to biz.OrderShippingDocumentStatus, audit *biz.AuditEvent) (*biz.OrderShippingDocument, error) {
	if _, err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	var updated *ent.OrderShippingDocument
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, queryErr := tx.OrderShippingDocument.Query().
			Where(
				ordershippingdocumentent.IDEQ(id),
				ordershippingdocumentent.OrderIDEQ(orderID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderShippingDocumentNotFound, nil)
		}
		if item.Status != ordershippingdocumentent.Status(from) {
			return biz.ErrOrderShippingDocumentStatusConflict
		}
		if item.Status == ordershippingdocumentent.StatusRELEASED ||
			(item.Status == ordershippingdocumentent.StatusDRAFT && to != biz.OrderShippingDocumentStatusConfirmed) ||
			(item.Status == ordershippingdocumentent.StatusCONFIRMED && to != biz.OrderShippingDocumentStatusReleased) {
			return biz.ErrOrderShippingDocumentInvalidStatus
		}
		var saveErr error
		updated, saveErr = tx.OrderShippingDocument.UpdateOne(item).
			SetStatus(ordershippingdocumentent.Status(to)).
			Save(ctx)
		if saveErr != nil {
			return saveErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return orderShippingDocumentToBiz(updated), nil
}

func (r *orderShippingDocumentRepo) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, audit *biz.AuditEvent) error {
	if _, err := r.order(ctx, organizationID, orderID); err != nil {
		return err
	}
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, queryErr := tx.OrderShippingDocument.Query().
			Where(
				ordershippingdocumentent.IDEQ(id),
				ordershippingdocumentent.OrderIDEQ(orderID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderShippingDocumentNotFound, nil)
		}
		if item.Status == ordershippingdocumentent.StatusRELEASED {
			return biz.ErrOrderShippingDocumentInvalidStatus
		}
		if err := tx.OrderShippingDocument.DeleteOneID(id).Exec(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

func orderShippingDocumentToBiz(item *ent.OrderShippingDocument) *biz.OrderShippingDocument {
	if item == nil {
		return nil
	}
	result := &biz.OrderShippingDocument{
		ID:        item.ID,
		OrderID:   item.OrderID,
		HouseNo:   item.HouseNo,
		Status:    biz.OrderShippingDocumentStatus(item.Status),
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
	if item.ReleaseType != "" {
		v := item.ReleaseType
		result.ReleaseType = &v
	}
	if item.Note != "" {
		v := item.Note
		result.Note = &v
	}
	return result
}
