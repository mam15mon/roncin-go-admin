package data

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderattachmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderattachment"
)

type orderAttachmentRepo struct {
	data *Data
}

func NewOrderAttachmentRepo(data *Data) biz.OrderAttachmentRepo {
	return &orderAttachmentRepo{data: data}
}

func (r *orderAttachmentRepo) order(ctx context.Context, organizationID, orderID uuid.UUID) error {
	if _, err := r.data.db.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx); err != nil {
		if ent.IsNotFound(err) {
			return biz.ErrOrderAttachmentNotFound
		}
		return err
	}
	return nil
}

func (r *orderAttachmentRepo) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderAttachment, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	items, err := r.data.db.OrderAttachment.Query().
		Where(orderattachmentent.OrderIDEQ(orderID)).
		Order(orderattachmentent.ByOrderID(), orderattachmentent.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.OrderAttachment, 0, len(items))
	for _, item := range items {
		result = append(result, orderAttachmentToBiz(item))
	}
	return result, nil
}

func (r *orderAttachmentRepo) Create(ctx context.Context, organizationID, actorID, orderID uuid.UUID, input *biz.OrderAttachment) (*biz.OrderAttachment, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	create := r.data.db.OrderAttachment.Create().
		SetOrderID(orderID).
		SetDocType(input.DocType).
		SetIdempotencyKey(input.IdempotencyKey).
		SetFileName(input.FileName).
		SetMimeType(input.MIMEType).
		SetFileSize(input.FileSize).
		SetObjectKey(input.ObjectKey).
		SetUploadedBy(actorID)
	if input.Checksum != "" {
		create.SetChecksum(input.Checksum)
	}
	created, err := create.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) && strings.Contains(err.Error(), "order_attachment_idempotency_key") {
			return nil, biz.ErrOrderAttachmentExists
		}
		return nil, err
	}
	return orderAttachmentToBiz(created), nil
}

func orderAttachmentToBiz(item *ent.OrderAttachment) *biz.OrderAttachment {
	result := &biz.OrderAttachment{
		ID:             item.ID,
		OrderID:        item.OrderID,
		DocType:        item.DocType,
		IdempotencyKey: item.IdempotencyKey,
		FileName:       item.FileName,
		MIMEType:       item.MimeType,
		FileSize:       item.FileSize,
		ObjectKey:      item.ObjectKey,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
	if item.Checksum != nil {
		result.Checksum = *item.Checksum
	}
	if item.UploadedBy != nil {
		value := *item.UploadedBy
		result.UploadedBy = &value
	}
	return result
}

var _ biz.OrderAttachmentRepo = (*orderAttachmentRepo)(nil)
