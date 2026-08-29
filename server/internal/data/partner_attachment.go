package data

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partnerattachmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerattachment"
)

type partnerAttachmentRepo struct{ data *Data }

func NewPartnerAttachmentRepo(data *Data) biz.PartnerAttachmentRepo {
	return &partnerAttachmentRepo{data: data}
}

func (r *partnerAttachmentRepo) partner(ctx context.Context, organizationID, partnerID uuid.UUID) error {
	if _, err := r.data.db.Partner.Query().Where(partnerent.IDEQ(partnerID), partnerent.OrganizationIDEQ(organizationID)).Only(ctx); err != nil {
		if ent.IsNotFound(err) {
			return biz.ErrPartnerAttachmentInvalidArgument
		}
		return err
	}
	return nil
}

func (r *partnerAttachmentRepo) List(ctx context.Context, organizationID, partnerID uuid.UUID) ([]*biz.PartnerAttachment, error) {
	if err := r.partner(ctx, organizationID, partnerID); err != nil {
		return nil, err
	}
	items, err := r.data.db.PartnerAttachment.Query().Where(partnerattachmentent.PartnerIDEQ(partnerID)).Order(partnerattachmentent.ByCreatedAt()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.PartnerAttachment, 0, len(items))
	for _, item := range items {
		result = append(result, partnerAttachmentToBiz(item))
	}
	return result, nil
}

func (r *partnerAttachmentRepo) Create(ctx context.Context, organizationID, actorID, partnerID uuid.UUID, input *biz.PartnerAttachment, audit *biz.AuditEvent) (*biz.PartnerAttachment, error) {
	if err := r.partner(ctx, organizationID, partnerID); err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	create := tx.PartnerAttachment.Create().
		SetPartnerID(partnerID).
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
		_ = tx.Rollback()
		if ent.IsConstraintError(err) && strings.Contains(err.Error(), "partner_attachment_idempotency_key") {
			return nil, biz.ErrPartnerAttachmentExists
		}
		return nil, err
	}
	audit.Details["attachment.id"] = created.ID.String()
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return partnerAttachmentToBiz(created), nil
}

func partnerAttachmentToBiz(item *ent.PartnerAttachment) *biz.PartnerAttachment {
	result := &biz.PartnerAttachment{
		ID: item.ID, PartnerID: item.PartnerID, IdempotencyKey: item.IdempotencyKey, FileName: item.FileName,
		MIMEType: item.MimeType, FileSize: item.FileSize, ObjectKey: item.ObjectKey, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
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
