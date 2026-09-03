package biz

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

const maxOrderAttachmentSize int64 = 100 << 20

var (
	ErrOrderAttachmentNotFound        = errors.NotFound("ORDER_ATTACHMENT_NOT_FOUND", "订单附件不存在")
	ErrOrderAttachmentExists          = errors.Conflict("ORDER_ATTACHMENT_EXISTS", "订单附件幂等键已存在")
	ErrOrderAttachmentInvalidArgument = errors.BadRequest("ORDER_ATTACHMENT_INVALID_ARGUMENT", "订单附件字段不合法")
)

type OrderAttachment struct {
	ID             uuid.UUID
	OrderID        uuid.UUID
	AssetID        uuid.UUID
	DocType        string
	IdempotencyKey string
	FileName       string
	MIMEType       string
	FileSize       int64
	ObjectKey      string
	Checksum       string
	UploadedBy     *uuid.UUID
	CreatedBy      *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type OrderAttachmentRepo interface {
	List(context.Context, uuid.UUID, uuid.UUID) ([]*OrderAttachment, error)
	Create(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *OrderAttachment, *AuditEvent) (*OrderAttachment, error)
	RemoveReference(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, *AuditEvent) error
}

type OrderAttachmentUsecase struct {
	repo OrderAttachmentRepo
}

func NewOrderAttachmentUsecase(repo OrderAttachmentRepo) *OrderAttachmentUsecase {
	return &OrderAttachmentUsecase{repo: repo}
}

func (uc *OrderAttachmentUsecase) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderAttachment, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderAttachmentInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, orderID)
}

func (uc *OrderAttachmentUsecase) Register(ctx context.Context, organizationID, actorID, orderID uuid.UUID, input *OrderAttachment) (*OrderAttachment, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderAttachmentInvalidArgument
	}
	normalized, err := normalizeOrderAttachment(input, actorID)
	if err != nil {
		return nil, err
	}
	audit := &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.attachment.register",
		Result:         "success",
		Details:        map[string]string{"order.id": orderID.String()},
	}
	created, err := uc.repo.Create(ctx, organizationID, actorID, orderID, normalized, audit)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (uc *OrderAttachmentUsecase) RemoveReference(ctx context.Context, organizationID, actorID, orderID, attachmentID uuid.UUID) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || attachmentID == uuid.Nil {
		return ErrOrderAttachmentInvalidArgument
	}
	audit := &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.attachment.remove_reference",
		Result:         "success",
		Details:        map[string]string{"order.id": orderID.String(), "attachment.id": attachmentID.String()},
	}
	return uc.repo.RemoveReference(ctx, organizationID, actorID, orderID, attachmentID, audit)
}

func normalizeOrderAttachment(input *OrderAttachment, actorID uuid.UUID) (*OrderAttachment, error) {
	if input == nil {
		return nil, ErrOrderAttachmentInvalidArgument
	}
	if containsControl(input.DocType) || containsControl(input.IdempotencyKey) || containsControl(input.FileName) || containsControl(input.MIMEType) || containsControl(input.ObjectKey) || containsControl(input.Checksum) {
		return nil, ErrOrderAttachmentInvalidArgument
	}
	output := *input
	output.DocType = strings.TrimSpace(output.DocType)
	output.IdempotencyKey = strings.TrimSpace(output.IdempotencyKey)
	output.FileName = strings.TrimSpace(output.FileName)
	output.MIMEType = strings.TrimSpace(output.MIMEType)
	output.ObjectKey = strings.TrimSpace(output.ObjectKey)
	output.Checksum = strings.TrimSpace(output.Checksum)
	if output.DocType == "" || output.IdempotencyKey == "" || output.FileName == "" || output.MIMEType == "" || output.ObjectKey == "" || output.FileSize <= 0 || output.FileSize > maxOrderAttachmentSize {
		return nil, ErrOrderAttachmentInvalidArgument
	}
	if utf8.RuneCountInString(output.DocType) > 64 || utf8.RuneCountInString(output.FileName) > 255 || utf8.RuneCountInString(output.MIMEType) > 127 || utf8.RuneCountInString(output.ObjectKey) > 1024 || utf8.RuneCountInString(output.IdempotencyKey) > 128 || utf8.RuneCountInString(output.Checksum) > 128 {
		return nil, ErrOrderAttachmentInvalidArgument
	}
	if output.FileName == "." || output.FileName == ".." || strings.ContainsAny(output.FileName, `/\\`) || containsControl(output.DocType) || containsControl(output.FileName) || containsControl(output.MIMEType) || containsControl(output.ObjectKey) || containsControl(output.IdempotencyKey) {
		return nil, ErrOrderAttachmentInvalidArgument
	}
	output.UploadedBy = &actorID
	return &output, nil
}
