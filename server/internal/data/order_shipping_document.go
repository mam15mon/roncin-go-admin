package data

import (
	"context"
	"strings"

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

func (r *orderShippingDocumentRepo) order(ctx context.Context, organizationID, orderID uuid.UUID) error {
	if _, err := r.data.db.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx); err != nil {
		if ent.IsNotFound(err) {
			return biz.ErrOrderShippingDocumentNotFound
		}
		return err
	}
	return nil
}

func (r *orderShippingDocumentRepo) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderShippingDocument, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	items, err := r.data.db.OrderShippingDocument.Query().
		Where(ordershippingdocumentent.OrderIDEQ(orderID)).
		Order(ordershippingdocumentent.ByCreatedAt(), ordershippingdocumentent.ByMasterNo()).
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
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	builder := tx.OrderShippingDocument.Create().
		SetID(input.ID).
		SetOrderID(orderID).
		SetMasterNo(input.MasterNo).
		SetHouseNo(input.HouseNo).
		SetStatus(ordershippingdocumentent.StatusDRAFT)
	if input.ReleaseType != nil {
		builder.SetReleaseType(*input.ReleaseType)
	}
	if input.Note != nil {
		builder.SetNote(*input.Note)
	}
	created, err := builder.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) && strings.Contains(err.Error(), "ordershippingdocument_order_id_master_no") {
			return nil, biz.ErrOrderShippingDocumentExists
		}
		return nil, err
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return orderShippingDocumentToBiz(created), nil
}

func (r *orderShippingDocumentRepo) Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *biz.OrderShippingDocument, audit *biz.AuditEvent) (*biz.OrderShippingDocument, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	item, err := tx.OrderShippingDocument.Query().
		Where(
			ordershippingdocumentent.IDEQ(id),
			ordershippingdocumentent.OrderIDEQ(orderID),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderShippingDocumentNotFound
		}
		return nil, err
	}
	if item.Status != ordershippingdocumentent.StatusDRAFT && item.Status != ordershippingdocumentent.StatusCONFIRMED {
		_ = tx.Rollback()
		return nil, biz.ErrOrderShippingDocumentInvalidStatus
	}
	builder := tx.OrderShippingDocument.UpdateOne(item).
		SetMasterNo(input.MasterNo).
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
	updated, err := builder.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) && strings.Contains(err.Error(), "ordershippingdocument_order_id_master_no") {
			return nil, biz.ErrOrderShippingDocumentExists
		}
		return nil, err
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return orderShippingDocumentToBiz(updated), nil
}

func (r *orderShippingDocumentRepo) Transition(ctx context.Context, organizationID, orderID, id uuid.UUID, from, to biz.OrderShippingDocumentStatus, audit *biz.AuditEvent) (*biz.OrderShippingDocument, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	item, err := tx.OrderShippingDocument.Query().
		Where(
			ordershippingdocumentent.IDEQ(id),
			ordershippingdocumentent.OrderIDEQ(orderID),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderShippingDocumentNotFound
		}
		return nil, err
	}
	if item.Status != ordershippingdocumentent.Status(from) {
		_ = tx.Rollback()
		return nil, biz.ErrOrderShippingDocumentStatusConflict
	}
	if item.Status == ordershippingdocumentent.StatusRELEASED ||
		(item.Status == ordershippingdocumentent.StatusDRAFT && to != biz.OrderShippingDocumentStatusConfirmed) ||
		(item.Status == ordershippingdocumentent.StatusCONFIRMED && to != biz.OrderShippingDocumentStatusReleased) {
		_ = tx.Rollback()
		return nil, biz.ErrOrderShippingDocumentInvalidStatus
	}
	updated, err := tx.OrderShippingDocument.UpdateOne(item).
		SetStatus(ordershippingdocumentent.Status(to)).
		Save(ctx)
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
	return orderShippingDocumentToBiz(updated), nil
}

func (r *orderShippingDocumentRepo) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, audit *biz.AuditEvent) error {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	item, err := tx.OrderShippingDocument.Query().
		Where(
			ordershippingdocumentent.IDEQ(id),
			ordershippingdocumentent.OrderIDEQ(orderID),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return biz.ErrOrderShippingDocumentNotFound
		}
		return err
	}
	if item.Status == ordershippingdocumentent.StatusRELEASED {
		_ = tx.Rollback()
		return biz.ErrOrderShippingDocumentInvalidStatus
	}
	n, err := tx.OrderShippingDocument.Delete().
		Where(
			ordershippingdocumentent.IDEQ(id),
			ordershippingdocumentent.OrderIDEQ(orderID),
		).
		Exec(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if n == 0 {
		_ = tx.Rollback()
		return biz.ErrOrderShippingDocumentNotFound
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func orderShippingDocumentToBiz(item *ent.OrderShippingDocument) *biz.OrderShippingDocument {
	if item == nil {
		return nil
	}
	result := &biz.OrderShippingDocument{
		ID:        item.ID,
		OrderID:   item.OrderID,
		MasterNo:  item.MasterNo,
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

var _ biz.OrderShippingDocumentRepo = (*orderShippingDocumentRepo)(nil)
