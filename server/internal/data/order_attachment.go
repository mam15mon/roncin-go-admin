package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	objectdeletionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/objectstoragedeletion"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderattachmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderattachment"
	orderattachmentassetent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderattachmentasset"
)

type orderAttachmentRepo struct {
	data *Data
}

func NewOrderAttachmentRepo(data *Data) biz.OrderAttachmentRepo {
	return &orderAttachmentRepo{data: data}
}

func (r *orderAttachmentRepo) order(ctx context.Context, client *ent.Client, organizationID, orderID uuid.UUID) error {
	if _, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx); err != nil {
		return mapEntError(err, biz.ErrOrderAttachmentNotFound, nil)
	}
	return nil
}

// CheckEditable 预读沿用提交时的同一业务门禁，最终写入仍在订单锁内重验。
func (r *orderAttachmentRepo) CheckEditable(ctx context.Context, organizationID, orderID uuid.UUID) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	order, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return mapEntError(err, biz.ErrOrderAttachmentNotFound, nil)
	}
	return ensureOrderBusinessContentEditable(ctx, client.User, order)
}

func (r *orderAttachmentRepo) Get(ctx context.Context, organizationID, orderID, attachmentID uuid.UUID) (*biz.OrderAttachment, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	if err := r.order(ctx, client, organizationID, orderID); err != nil {
		return nil, err
	}
	item, err := client.OrderAttachment.Query().Where(orderattachmentent.IDEQ(attachmentID), orderattachmentent.OrderIDEQ(orderID)).WithAsset().Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderAttachmentNotFound, nil)
	}
	return orderAttachmentToBiz(item), nil
}

func (r *orderAttachmentRepo) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderAttachment, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	if err := r.order(ctx, client, organizationID, orderID); err != nil {
		return nil, err
	}
	items, err := client.OrderAttachment.Query().
		Where(orderattachmentent.OrderIDEQ(orderID)).
		WithAsset().
		Order(orderattachmentent.ByCreatedAt(), orderattachmentent.ByID()).
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
		order, queryErr := tx.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderAttachmentNotFound, nil)
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}
		// 同一授权重试只返回完全一致的既有引用，不创建重复附件。
		previous, queryErr := tx.OrderAttachment.Query().Where(orderattachmentent.OrderIDEQ(orderID), orderattachmentent.IdempotencyKeyEQ(input.IdempotencyKey)).WithAsset().Only(ctx)
		if queryErr == nil {
			current := orderAttachmentToBiz(previous)
			if current.ObjectKey != input.ObjectKey || current.DocType != input.DocType || current.FileName != input.FileName || current.MIMEType != input.MIMEType || current.FileSize != input.FileSize || current.Checksum != input.Checksum || current.UploadedBy == nil || *current.UploadedBy != actorID {
				return biz.ErrOrderAttachmentExists
			}
			created = previous
			return nil
		}
		if !ent.IsNotFound(queryErr) {
			return queryErr
		}
		// 1. 查找或创建资产
		asset, queryErr := tx.OrderAttachmentAsset.Query().
			Where(
				orderattachmentassetent.OrganizationIDEQ(organizationID),
				orderattachmentassetent.ObjectKeyEQ(input.ObjectKey),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			if ent.IsNotFound(queryErr) {
				// 必须在资产锁查询之后检查：跨票解除最后引用时，该查询可能等待
				// 删除事务提交后才返回不存在，旧上传授权不能让待删除对象复活。
				deleting, err := tx.ObjectStorageDeletion.Query().Where(objectdeletionent.ObjectKeyEQ(input.ObjectKey)).Exist(ctx)
				if err != nil {
					return err
				}
				if deleting {
					return biz.ErrOrderAttachmentUploadInvalid
				}
				createAsset := tx.OrderAttachmentAsset.Create().
					SetOrganizationID(organizationID).
					SetObjectKey(input.ObjectKey).
					SetFileName(input.FileName).
					SetMimeType(input.MIMEType).
					SetFileSize(input.FileSize)
				if input.Checksum != "" {
					createAsset.SetChecksum(input.Checksum)
				}
				createAsset.SetUploadedBy(actorID)
				var saveErr error
				asset, saveErr = createAsset.Save(ctx)
				if saveErr != nil {
					return saveErr
				}
			} else {
				return queryErr
			}
		}

		// 2. 创建订单引用
		create := tx.OrderAttachment.Create().
			SetOrderID(orderID).
			SetAssetID(asset.ID).
			SetDocType(input.DocType).
			SetIdempotencyKey(input.IdempotencyKey).
			SetCreatedBy(actorID)

		var saveErr error
		created, saveErr = create.Save(ctx)
		if saveErr != nil {
			return mapEntConstraint(saveErr, "order_attachment_idempotency_key", biz.ErrOrderAttachmentExists)
		}
		created.Edges.Asset = asset
		audit.Details["attachment.id"] = created.ID.String()
		audit.Details["asset.id"] = asset.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return orderAttachmentToBiz(created), nil
}

func (r *orderAttachmentRepo) RemoveReference(ctx context.Context, organizationID, actorID, orderID, attachmentID uuid.UUID, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 1. 验证订单存在且属于该组织并加锁
		order, err := tx.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrOrderAttachmentNotFound, nil)
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}
		// 2. 查询目标引用
		ref, err := tx.OrderAttachment.Query().Where(orderattachmentent.IDEQ(attachmentID), orderattachmentent.OrderIDEQ(orderID)).Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrOrderAttachmentNotFound, nil)
		}
		// 3. 按固定锁序锁定资产
		asset, err := tx.OrderAttachmentAsset.Query().Where(orderattachmentassetent.IDEQ(ref.AssetID)).ForUpdate().Only(ctx)
		if err != nil {
			return err
		}
		// 4. 删除当前订单引用
		if err := tx.OrderAttachment.DeleteOneID(attachmentID).Exec(ctx); err != nil {
			return err
		}
		// 5. 检查是否仍有其他订单引用该资产
		otherRefsCount, err := tx.OrderAttachment.Query().Where(orderattachmentent.AssetIDEQ(asset.ID)).Count(ctx)
		if err != nil {
			return err
		}
		// 若为最后一条引用，在同一事务登记对象删除任务并删除资产记录
		if otherRefsCount == 0 {
			if err := enqueueObjectStorageDeletion(ctx, tx, organizationID, asset.ObjectKey); err != nil {
				return err
			}
			if err := tx.OrderAttachmentAsset.DeleteOneID(asset.ID).Exec(ctx); err != nil {
				return err
			}
			audit.Details["asset.deleted"] = "true"
		}
		audit.Details["attachment.id"] = attachmentID.String()
		audit.Details["asset.id"] = asset.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

func orderAttachmentToBiz(item *ent.OrderAttachment) *biz.OrderAttachment {
	result := &biz.OrderAttachment{
		ID:             item.ID,
		OrderID:        item.OrderID,
		AssetID:        item.AssetID,
		DocType:        item.DocType,
		IdempotencyKey: item.IdempotencyKey,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
	if item.CreatedBy != nil {
		val := *item.CreatedBy
		result.CreatedBy = &val
	}
	if asset, err := item.Edges.AssetOrErr(); err == nil && asset != nil {
		result.FileName = asset.FileName
		result.MIMEType = asset.MimeType
		result.FileSize = asset.FileSize
		result.ObjectKey = asset.ObjectKey
		if asset.Checksum != nil {
			result.Checksum = *asset.Checksum
		}
		if asset.UploadedBy != nil {
			val := *asset.UploadedBy
			result.UploadedBy = &val
		}
	}
	return result
}

var _ biz.OrderAttachmentRepo = (*orderAttachmentRepo)(nil)
