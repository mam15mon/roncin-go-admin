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

func (r *orderAttachmentRepo) Create(ctx context.Context, organizationID, actorID, orderID uuid.UUID, input *biz.OrderAttachment, audit *biz.AuditEvent) (*biz.OrderAttachment, error) {
	var created *ent.OrderAttachment
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if _, queryErr := tx.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx); queryErr != nil {
			if ent.IsNotFound(queryErr) {
				return biz.ErrOrderAttachmentNotFound
			}
			return queryErr
		}
		create := tx.OrderAttachment.Create().
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
		var saveErr error
		created, saveErr = create.Save(ctx)
		if saveErr != nil {
			if ent.IsConstraintError(saveErr) && strings.Contains(saveErr.Error(), "order_attachment_idempotency_key") {
				return biz.ErrOrderAttachmentExists
			}
			return saveErr
		}
		audit.Details["attachment.id"] = created.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
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
