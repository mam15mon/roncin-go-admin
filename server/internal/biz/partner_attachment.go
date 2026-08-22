package biz

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

const maxPartnerAttachmentSize int64 = 100 << 20

var (
	ErrPartnerAttachmentNotFound        = errors.NotFound("PARTNER_ATTACHMENT_NOT_FOUND", "附件不存在")
	ErrPartnerAttachmentExists          = errors.Conflict("PARTNER_ATTACHMENT_EXISTS", "附件幂等键已存在")
	ErrPartnerAttachmentInvalidArgument = errors.BadRequest("PARTNER_ATTACHMENT_INVALID_ARGUMENT", "附件字段不合法")
)

type PartnerAttachment struct {
	ID             uuid.UUID
	PartnerID      uuid.UUID
	IdempotencyKey string
	FileName       string
	MIMEType       string
	FileSize       int64
	ObjectKey      string
	Checksum       string
	UploadedBy     *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type PartnerAttachmentRepo interface {
	List(context.Context, uuid.UUID, uuid.UUID) ([]*PartnerAttachment, error)
	Create(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *PartnerAttachment) (*PartnerAttachment, error)
}

type PartnerAttachmentUsecase struct {
	repo  PartnerAttachmentRepo
	audit AuditRepo
}

func NewPartnerAttachmentUsecase(repo PartnerAttachmentRepo, audit AuditRepo) *PartnerAttachmentUsecase {
	return &PartnerAttachmentUsecase{repo: repo, audit: audit}
}

func (uc *PartnerAttachmentUsecase) List(ctx context.Context, organizationID, partnerID uuid.UUID) ([]*PartnerAttachment, error) {
	if organizationID == uuid.Nil || partnerID == uuid.Nil {
		return nil, ErrPartnerAttachmentInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, partnerID)
}

func (uc *PartnerAttachmentUsecase) Register(ctx context.Context, organizationID, actorID, partnerID uuid.UUID, input *PartnerAttachment) (*PartnerAttachment, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || partnerID == uuid.Nil {
		return nil, ErrPartnerAttachmentInvalidArgument
	}
	normalized, err := normalizePartnerAttachment(input, actorID)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.Create(ctx, organizationID, actorID, partnerID, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "partner.attachment.register", ResourceType: "partner", ResourceID: partnerID.String(), Result: "success", Details: map[string]string{"attachment.id": created.ID.String(), "partner.id": partnerID.String()}}); err != nil {
		return nil, fmt.Errorf("write partner attachment register audit: %w", err)
	}
	return created, nil
}

func normalizePartnerAttachment(input *PartnerAttachment, actorID uuid.UUID) (*PartnerAttachment, error) {
	if input == nil {
		return nil, ErrPartnerAttachmentInvalidArgument
	}
	if containsControl(input.IdempotencyKey) || containsControl(input.FileName) || containsControl(input.MIMEType) || containsControl(input.ObjectKey) || containsControl(input.Checksum) {
		return nil, ErrPartnerAttachmentInvalidArgument
	}
	output := *input
	output.IdempotencyKey = strings.TrimSpace(output.IdempotencyKey)
	output.FileName = strings.TrimSpace(output.FileName)
	output.MIMEType = strings.TrimSpace(output.MIMEType)
	output.ObjectKey = strings.TrimSpace(output.ObjectKey)
	output.Checksum = strings.TrimSpace(output.Checksum)
	if output.IdempotencyKey == "" || output.FileName == "" || output.MIMEType == "" || output.ObjectKey == "" || output.FileSize <= 0 || output.FileSize > maxPartnerAttachmentSize {
		return nil, ErrPartnerAttachmentInvalidArgument
	}
	if utf8.RuneCountInString(output.FileName) > 255 || utf8.RuneCountInString(output.MIMEType) > 127 || utf8.RuneCountInString(output.ObjectKey) > 1024 || utf8.RuneCountInString(output.IdempotencyKey) > 128 || utf8.RuneCountInString(output.Checksum) > 128 {
		return nil, ErrPartnerAttachmentInvalidArgument
	}
	if output.FileName == "." || output.FileName == ".." || strings.ContainsAny(output.FileName, `/\\`) || containsControl(output.FileName) || containsControl(output.MIMEType) || containsControl(output.ObjectKey) || containsControl(output.IdempotencyKey) {
		return nil, ErrPartnerAttachmentInvalidArgument
	}
	output.UploadedBy = &actorID
	return &output, nil
}

func containsControl(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}
