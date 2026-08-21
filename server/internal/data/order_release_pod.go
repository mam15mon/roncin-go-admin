package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderreleasepodent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderreleasepod"
	ordershippingdocumentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordershippingdocument"
)

type orderReleasePodRepo struct {
	data *Data
}

func NewOrderReleasePodRepo(data *Data) biz.OrderReleasePodRepo {
	return &orderReleasePodRepo{data: data}
}

func (r *orderReleasePodRepo) order(ctx context.Context, organizationID, orderID uuid.UUID) error {
	if _, err := r.data.db.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx); err != nil {
		if ent.IsNotFound(err) {
			return biz.ErrOrderReleasePodNotFound
		}
		return err
	}
	return nil
}

func (r *orderReleasePodRepo) validateShippingDocument(ctx context.Context, orderID uuid.UUID, documentID *uuid.UUID) error {
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
		return biz.ErrOrderReleasePodDocumentInvalid
	}
	return nil
}

func (r *orderReleasePodRepo) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderReleasePod, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	items, err := r.data.db.OrderReleasePod.Query().
		Where(orderreleasepodent.OrderIDEQ(orderID)).
		Order(orderreleasepodent.ByCreatedAt(), orderreleasepodent.ByReleaseNo()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.OrderReleasePod, 0, len(items))
	for _, item := range items {
		result = append(result, orderReleasePodToBiz(item))
	}
	return result, nil
}

func (r *orderReleasePodRepo) Add(ctx context.Context, organizationID, orderID uuid.UUID, input *biz.OrderReleasePod) (*biz.OrderReleasePod, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	if err := r.validateShippingDocument(ctx, orderID, input.ShippingDocumentID); err != nil {
		return nil, err
	}
	builder := r.data.db.OrderReleasePod.Create().
		SetOrderID(orderID).
		SetStatus(orderreleasepodent.StatusPENDING)
	if input.ShippingDocumentID != nil {
		builder.SetShippingDocumentID(*input.ShippingDocumentID)
	}
	if input.ReleaseNo != nil {
		builder.SetReleaseNo(*input.ReleaseNo)
	}
	if input.PodNo != nil {
		builder.SetPodNo(*input.PodNo)
	}
	if input.Note != nil {
		builder.SetNote(*input.Note)
	}
	created, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}
	return orderReleasePodToBiz(created), nil
}

func (r *orderReleasePodRepo) Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *biz.OrderReleasePod) (*biz.OrderReleasePod, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	if err := r.validateShippingDocument(ctx, orderID, input.ShippingDocumentID); err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	item, err := tx.OrderReleasePod.Query().
		Where(
			orderreleasepodent.IDEQ(id),
			orderreleasepodent.OrderIDEQ(orderID),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderReleasePodNotFound
		}
		return nil, err
	}
	if item.Status == orderreleasepodent.StatusRETURNED {
		_ = tx.Rollback()
		return nil, biz.ErrOrderReleasePodInvalidStatus
	}
	builder := tx.OrderReleasePod.UpdateOne(item)
	if input.ShippingDocumentID != nil {
		builder.SetShippingDocumentID(*input.ShippingDocumentID)
	} else {
		builder.ClearShippingDocumentID()
	}
	if input.ReleaseNo != nil {
		builder.SetReleaseNo(*input.ReleaseNo)
	} else {
		builder.ClearReleaseNo()
	}
	if input.PodNo != nil {
		builder.SetPodNo(*input.PodNo)
	} else {
		builder.ClearPodNo()
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
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return orderReleasePodToBiz(updated), nil
}

func (r *orderReleasePodRepo) Transition(ctx context.Context, organizationID, orderID, id uuid.UUID, from, to biz.OrderReleasePodStatus, actorID uuid.UUID) (*biz.OrderReleasePod, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	item, err := tx.OrderReleasePod.Query().
		Where(
			orderreleasepodent.IDEQ(id),
			orderreleasepodent.OrderIDEQ(orderID),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderReleasePodNotFound
		}
		return nil, err
	}
	if item.Status != orderreleasepodent.Status(from) {
		_ = tx.Rollback()
		return nil, biz.ErrOrderReleasePodStatusConflict
	}
	if item.Status == orderreleasepodent.StatusRETURNED ||
		(item.Status == orderreleasepodent.StatusPENDING && to != biz.OrderReleasePodStatusSigned) ||
		(item.Status == orderreleasepodent.StatusSIGNED && to != biz.OrderReleasePodStatusReturned) {
		_ = tx.Rollback()
		return nil, biz.ErrOrderReleasePodInvalidStatus
	}
	builder := tx.OrderReleasePod.UpdateOne(item).
		SetStatus(orderreleasepodent.Status(to))
	if to == biz.OrderReleasePodStatusSigned {
		builder.SetSignedAt(time.Now()).SetSignedBy(actorID)
	}
	updated, err := builder.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return orderReleasePodToBiz(updated), nil
}

func (r *orderReleasePodRepo) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID) error {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return err
	}
	item, err := r.data.db.OrderReleasePod.Query().
		Where(
			orderreleasepodent.IDEQ(id),
			orderreleasepodent.OrderIDEQ(orderID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return biz.ErrOrderReleasePodNotFound
		}
		return err
	}
	if item.Status == orderreleasepodent.StatusRETURNED {
		return biz.ErrOrderReleasePodInvalidStatus
	}
	n, err := r.data.db.OrderReleasePod.Delete().
		Where(
			orderreleasepodent.IDEQ(id),
			orderreleasepodent.OrderIDEQ(orderID),
		).
		Exec(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return biz.ErrOrderReleasePodNotFound
	}
	return nil
}

func orderReleasePodToBiz(item *ent.OrderReleasePod) *biz.OrderReleasePod {
	if item == nil {
		return nil
	}
	result := &biz.OrderReleasePod{
		ID:                 item.ID,
		OrderID:            item.OrderID,
		ShippingDocumentID: item.ShippingDocumentID,
		Status:             biz.OrderReleasePodStatus(item.Status),
		SignedAt:           item.SignedAt,
		SignedBy:           item.SignedBy,
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	}
	if item.ReleaseNo != "" {
		v := item.ReleaseNo
		result.ReleaseNo = &v
	}
	if item.PodNo != "" {
		v := item.PodNo
		result.PodNo = &v
	}
	if item.Note != "" {
		v := item.Note
		result.Note = &v
	}
	return result
}

var _ biz.OrderReleasePodRepo = (*orderReleasePodRepo)(nil)
