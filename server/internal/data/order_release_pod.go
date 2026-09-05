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
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

type orderReleasePodRepo struct{ data *Data }

func NewOrderReleasePodRepo(data *Data) biz.OrderReleasePodRepo {
	return &orderReleasePodRepo{data: data}
}

func ensureReleasePodOrderEditable(ctx context.Context, client *ent.Client, existing *ent.Order) error {
	if existing == nil {
		return nil
	}
	if _, err := orderAccessBusinessType(existing.BusinessType); err != nil {
		return err
	}
	if existing.LockedAt == nil {
		return nil
	}
	var lockedByName string
	if existing.LockedBy != nil {
		user, err := client.User.Get(ctx, *existing.LockedBy)
		if err == nil && user != nil {
			lockedByName = user.DisplayName
		}
	}
	return biz.NewErrOrderBusinessLocked(existing.ID, existing.OrderNo, existing.LockGeneration, *existing.LockedAt, lockedByName)
}

func (r *orderReleasePodRepo) order(ctx context.Context, organizationID, orderID uuid.UUID) (*ent.Order, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderReleasePodNotFound, nil)
	}
	return item, nil
}

func validateReleasePodDocumentReference(ctx context.Context, client *ent.Client, order *ent.Order, input *biz.OrderReleasePod) error {
	if order.BusinessType != orderent.BusinessTypeSE {
		if input.SeaDocumentType != "" || input.SeaDocumentID != nil {
			return biz.ErrOrderReleasePodDocumentInvalid
		}
		if input.ShippingDocumentID == nil {
			return nil
		}
		exists, err := client.OrderShippingDocument.Query().
			Where(ordershippingdocumentent.IDEQ(*input.ShippingDocumentID), ordershippingdocumentent.OrderIDEQ(order.ID)).
			ForShare().Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return biz.ErrOrderReleasePodDocumentInvalid
		}
		return nil
	}

	if input.ShippingDocumentID != nil {
		return biz.ErrOrderReleasePodDocumentInvalid
	}
	if input.SeaDocumentType == "" && input.SeaDocumentID == nil {
		return nil
	}
	if input.SeaDocumentID == nil || !input.SeaDocumentType.Valid() {
		return biz.ErrOrderReleasePodDocumentInvalid
	}

	candidate, err := client.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlinkent.OrganizationIDEQ(order.OrganizationID),
			seamasterbillorderlinkent.OrderIDEQ(order.ID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return biz.ErrOrderReleasePodDocumentInvalid
		}
		return err
	}
	mbl, err := client.SeaMasterBill.Query().
		Where(seamasterbillent.IDEQ(candidate.MasterBillID), seamasterbillent.OrganizationIDEQ(order.OrganizationID)).
		ForUpdate().Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return biz.ErrOrderReleasePodDocumentInvalid
		}
		return err
	}
	if mbl.Status == seamasterbillent.StatusVOIDED {
		return biz.ErrOrderReleasePodDocumentInvalid
	}
	link, err := client.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlinkent.IDEQ(candidate.ID)).ForUpdate().Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return biz.ErrOrderReleasePodDocumentInvalid
		}
		return err
	}
	if link.Status != seamasterbillorderlinkent.StatusACTIVE || link.OrderID != order.ID || link.MasterBillID != mbl.ID || link.OrganizationID != order.OrganizationID {
		return biz.ErrOrderReleasePodDocumentInvalid
	}

	switch input.SeaDocumentType {
	case biz.SeaDocumentTypeMasterBill:
		if *input.SeaDocumentID != mbl.ID {
			return biz.ErrOrderReleasePodDocumentInvalid
		}
	case biz.SeaDocumentTypeHouseBill:
		hb, queryErr := client.SeaHouseBill.Query().
			Where(
				seahousebillent.IDEQ(*input.SeaDocumentID),
				seahousebillent.OrganizationIDEQ(order.OrganizationID),
				seahousebillent.OrderIDEQ(order.ID),
				seahousebillent.MasterBillIDEQ(mbl.ID),
			).ForUpdate().Only(ctx)
		if queryErr != nil {
			if ent.IsNotFound(queryErr) {
				return biz.ErrOrderReleasePodDocumentInvalid
			}
			return queryErr
		}
		if hb.Status == seahousebillent.StatusVOIDED || hb.Status == seahousebillent.StatusREPLACED {
			return biz.ErrOrderReleasePodDocumentInvalid
		}
	default:
		return biz.ErrOrderReleasePodDocumentInvalid
	}
	return nil
}

func (r *orderReleasePodRepo) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderReleasePod, error) {
	if _, err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	items, err := client.OrderReleasePod.Query().Where(orderreleasepodent.OrderIDEQ(orderID)).
		Order(orderreleasepodent.ByCreatedAt(), orderreleasepodent.ByReleaseNo()).All(ctx)
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
	if _, err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	var created *ent.OrderReleasePod
	err := r.data.WithinTransaction(ctx, func(txCtx context.Context) error {
		client, err := r.data.client(txCtx)
		if err != nil {
			return err
		}
		order, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(txCtx)
		if err != nil {
			return mapEntError(err, biz.ErrOrderReleasePodNotFound, nil)
		}
		if err := ensureReleasePodOrderEditable(txCtx, client, order); err != nil {
			return err
		}
		if err := validateReleasePodDocumentReference(txCtx, client, order, input); err != nil {
			return err
		}
		builder := client.OrderReleasePod.Create().SetID(input.ID).SetOrderID(orderID).
			SetStatus(orderreleasepodent.StatusPENDING).SetNillableShippingDocumentID(input.ShippingDocumentID)
		setSeaReleasePodReferenceOnCreate(builder, input)
		if input.ReleaseNo != nil {
			builder.SetReleaseNo(*input.ReleaseNo)
		}
		if input.PodNo != nil {
			builder.SetPodNo(*input.PodNo)
		}
		if input.Note != nil {
			builder.SetNote(*input.Note)
		}
		created, err = builder.Save(txCtx)
		if err != nil {
			return err
		}
		return writeAudit(txCtx, client.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return orderReleasePodToBiz(created), nil
}

func setSeaReleasePodReferenceOnCreate(builder *ent.OrderReleasePodCreate, input *biz.OrderReleasePod) {
	if input.SeaDocumentID == nil {
		return
	}
	if input.SeaDocumentType == biz.SeaDocumentTypeMasterBill {
		builder.SetSeaMasterBillID(*input.SeaDocumentID)
	} else if input.SeaDocumentType == biz.SeaDocumentTypeHouseBill {
		builder.SetSeaHouseBillID(*input.SeaDocumentID)
	}
}

func setReleasePodReferencesOnUpdate(builder *ent.OrderReleasePodUpdateOne, input *biz.OrderReleasePod) {
	builder.ClearShippingDocumentID().ClearSeaMasterBillID().ClearSeaHouseBillID()
	if input.ShippingDocumentID != nil {
		builder.SetShippingDocumentID(*input.ShippingDocumentID)
	}
	if input.SeaDocumentID == nil {
		return
	}
	if input.SeaDocumentType == biz.SeaDocumentTypeMasterBill {
		builder.SetSeaMasterBillID(*input.SeaDocumentID)
	} else if input.SeaDocumentType == biz.SeaDocumentTypeHouseBill {
		builder.SetSeaHouseBillID(*input.SeaDocumentID)
	}
}

func (r *orderReleasePodRepo) Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *biz.OrderReleasePod, audit *biz.AuditEvent) (*biz.OrderReleasePod, error) {
	if _, err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	var updated *ent.OrderReleasePod
	err := r.data.WithinTransaction(ctx, func(txCtx context.Context) error {
		client, err := r.data.client(txCtx)
		if err != nil {
			return err
		}
		order, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(txCtx)
		if err != nil {
			return mapEntError(err, biz.ErrOrderReleasePodNotFound, nil)
		}
		if err := ensureReleasePodOrderEditable(txCtx, client, order); err != nil {
			return err
		}
		if err := validateReleasePodDocumentReference(txCtx, client, order, input); err != nil {
			return err
		}
		item, err := client.OrderReleasePod.Query().Where(orderreleasepodent.IDEQ(id), orderreleasepodent.OrderIDEQ(orderID)).ForUpdate().Only(txCtx)
		if err != nil {
			return mapEntError(err, biz.ErrOrderReleasePodNotFound, nil)
		}
		if item.Status == orderreleasepodent.StatusRETURNED {
			return biz.ErrOrderReleasePodInvalidStatus
		}
		builder := client.OrderReleasePod.UpdateOne(item)
		setReleasePodReferencesOnUpdate(builder, input)
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
		updated, err = builder.Save(txCtx)
		if err != nil {
			return err
		}
		return writeAudit(txCtx, client.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return orderReleasePodToBiz(updated), nil
}

func (r *orderReleasePodRepo) Transition(ctx context.Context, organizationID, orderID, id uuid.UUID, from, to biz.OrderReleasePodStatus, actorID uuid.UUID, audit *biz.AuditEvent) (*biz.OrderReleasePod, error) {
	if _, err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	var updated *ent.OrderReleasePod
	err := r.data.WithinTransaction(ctx, func(txCtx context.Context) error {
		client, err := r.data.client(txCtx)
		if err != nil {
			return err
		}
		order, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(txCtx)
		if err != nil {
			return mapEntError(err, biz.ErrOrderReleasePodNotFound, nil)
		}
		if err := ensureReleasePodOrderEditable(txCtx, client, order); err != nil {
			return err
		}
		item, err := client.OrderReleasePod.Query().Where(orderreleasepodent.IDEQ(id), orderreleasepodent.OrderIDEQ(orderID)).ForUpdate().Only(txCtx)
		if err != nil {
			return mapEntError(err, biz.ErrOrderReleasePodNotFound, nil)
		}
		if item.Status != orderreleasepodent.Status(from) {
			return biz.ErrOrderReleasePodStatusConflict
		}
		if item.Status == orderreleasepodent.StatusRETURNED ||
			(item.Status == orderreleasepodent.StatusPENDING && to != biz.OrderReleasePodStatusSigned) ||
			(item.Status == orderreleasepodent.StatusSIGNED && to != biz.OrderReleasePodStatusReturned) {
			return biz.ErrOrderReleasePodInvalidStatus
		}
		builder := client.OrderReleasePod.UpdateOne(item).SetStatus(orderreleasepodent.Status(to))
		if to == biz.OrderReleasePodStatusSigned {
			builder.SetSignedAt(time.Now()).SetSignedBy(actorID)
		}
		updated, err = builder.Save(txCtx)
		if err != nil {
			return err
		}
		return writeAudit(txCtx, client.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return orderReleasePodToBiz(updated), nil
}

func (r *orderReleasePodRepo) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, audit *biz.AuditEvent) error {
	if _, err := r.order(ctx, organizationID, orderID); err != nil {
		return err
	}
	return r.data.WithinTransaction(ctx, func(txCtx context.Context) error {
		client, err := r.data.client(txCtx)
		if err != nil {
			return err
		}
		order, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(txCtx)
		if err != nil {
			return mapEntError(err, biz.ErrOrderReleasePodNotFound, nil)
		}
		if err := ensureReleasePodOrderEditable(txCtx, client, order); err != nil {
			return err
		}
		item, err := client.OrderReleasePod.Query().Where(orderreleasepodent.IDEQ(id), orderreleasepodent.OrderIDEQ(orderID)).ForUpdate().Only(txCtx)
		if err != nil {
			return mapEntError(err, biz.ErrOrderReleasePodNotFound, nil)
		}
		if item.Status == orderreleasepodent.StatusRETURNED {
			return biz.ErrOrderReleasePodInvalidStatus
		}
		n, err := client.OrderReleasePod.Delete().Where(orderreleasepodent.IDEQ(id), orderreleasepodent.OrderIDEQ(orderID)).Exec(txCtx)
		if err != nil {
			return err
		}
		if n == 0 {
			return biz.ErrOrderReleasePodNotFound
		}
		return writeAudit(txCtx, client.AuditLog, audit)
	})
}

func orderReleasePodToBiz(item *ent.OrderReleasePod) *biz.OrderReleasePod {
	if item == nil {
		return nil
	}
	result := &biz.OrderReleasePod{
		ID: item.ID, OrderID: item.OrderID, ShippingDocumentID: item.ShippingDocumentID,
		Status: biz.OrderReleasePodStatus(item.Status), SignedAt: item.SignedAt, SignedBy: item.SignedBy,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
	if item.SeaMasterBillID != nil {
		result.SeaDocumentType = biz.SeaDocumentTypeMasterBill
		result.SeaDocumentID = item.SeaMasterBillID
	} else if item.SeaHouseBillID != nil {
		result.SeaDocumentType = biz.SeaDocumentTypeHouseBill
		result.SeaDocumentID = item.SeaHouseBillID
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
