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

func (r *orderReleasePodRepo) Add(ctx context.Context, organizationID, orderID uuid.UUID, input *biz.OrderReleasePod, audit *biz.AuditEvent) (*biz.OrderReleasePod, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	if err := r.validateShippingDocument(ctx, orderID, input.ShippingDocumentID); err != nil {
		return nil, err
	}
	var created *ent.OrderReleasePod
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		builder := tx.OrderReleasePod.Create().
			SetID(input.ID).
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
	return orderReleasePodToBiz(created), nil
}

func (r *orderReleasePodRepo) Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *biz.OrderReleasePod, audit *biz.AuditEvent) (*biz.OrderReleasePod, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	if err := r.validateShippingDocument(ctx, orderID, input.ShippingDocumentID); err != nil {
		return nil, err
	}
	var updated *ent.OrderReleasePod
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, queryErr := tx.OrderReleasePod.Query().
			Where(
				orderreleasepodent.IDEQ(id),
				orderreleasepodent.OrderIDEQ(orderID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			if ent.IsNotFound(queryErr) {
				return biz.ErrOrderReleasePodNotFound
			}
			return queryErr
		}
		if item.Status == orderreleasepodent.StatusRETURNED {
			return biz.ErrOrderReleasePodInvalidStatus
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
	return orderReleasePodToBiz(updated), nil
}

func (r *orderReleasePodRepo) Transition(ctx context.Context, organizationID, orderID, id uuid.UUID, from, to biz.OrderReleasePodStatus, actorID uuid.UUID, audit *biz.AuditEvent) (*biz.OrderReleasePod, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	var updated *ent.OrderReleasePod
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, queryErr := tx.OrderReleasePod.Query().
			Where(
				orderreleasepodent.IDEQ(id),
				orderreleasepodent.OrderIDEQ(orderID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			if ent.IsNotFound(queryErr) {
				return biz.ErrOrderReleasePodNotFound
			}
			return queryErr
		}
		if item.Status != orderreleasepodent.Status(from) {
			return biz.ErrOrderReleasePodStatusConflict
		}
		if item.Status == orderreleasepodent.StatusRETURNED ||
			(item.Status == orderreleasepodent.StatusPENDING && to != biz.OrderReleasePodStatusSigned) ||
			(item.Status == orderreleasepodent.StatusSIGNED && to != biz.OrderReleasePodStatusReturned) {
			return biz.ErrOrderReleasePodInvalidStatus
		}
		builder := tx.OrderReleasePod.UpdateOne(item).
			SetStatus(orderreleasepodent.Status(to))
		if to == biz.OrderReleasePodStatusSigned {
			builder.SetSignedAt(time.Now()).SetSignedBy(actorID)
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
	return orderReleasePodToBiz(updated), nil
}

func (r *orderReleasePodRepo) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, audit *biz.AuditEvent) error {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return err
	}
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, queryErr := tx.OrderReleasePod.Query().
			Where(
				orderreleasepodent.IDEQ(id),
				orderreleasepodent.OrderIDEQ(orderID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			if ent.IsNotFound(queryErr) {
				return biz.ErrOrderReleasePodNotFound
			}
			return queryErr
		}
		if item.Status == orderreleasepodent.StatusRETURNED {
			return biz.ErrOrderReleasePodInvalidStatus
		}
		n, deleteErr := tx.OrderReleasePod.Delete().
			Where(
				orderreleasepodent.IDEQ(id),
				orderreleasepodent.OrderIDEQ(orderID),
			).
			Exec(ctx)
		if deleteErr != nil {
			return deleteErr
		}
		if n == 0 {
			return biz.ErrOrderReleasePodNotFound
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
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
