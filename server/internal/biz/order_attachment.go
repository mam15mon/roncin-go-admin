package biz

import (
	"context"
	"encoding/hex"
	"mime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

const maxOrderAttachmentSize int64 = 100 << 20

var (
	ErrOrderAttachmentStorageUnavailable = errors.ServiceUnavailable("ORDER_ATTACHMENT_STORAGE_UNAVAILABLE", "订单附件 OSS 未配置或暂不可用")
	ErrOrderAttachmentUploadInvalid      = errors.BadRequest("ORDER_ATTACHMENT_UPLOAD_INVALID", "附件上传授权已失效或文件校验未通过，请重新上传")
	ErrOrderAttachmentVersioning         = errors.Conflict("ORDER_ATTACHMENT_BUCKET_VERSIONING", "附件 Bucket 必须保持未开启版本控制，请检查 OSS 配置")
	ErrOrderAttachmentNotFound           = errors.NotFound("ORDER_ATTACHMENT_NOT_FOUND", "订单附件不存在")
	ErrOrderAttachmentExists             = errors.Conflict("ORDER_ATTACHMENT_EXISTS", "订单附件幂等键已存在")
	ErrOrderAttachmentInvalidArgument    = errors.BadRequest("ORDER_ATTACHMENT_INVALID_ARGUMENT", "订单附件字段不合法")
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
	CheckEditable(context.Context, uuid.UUID, uuid.UUID) error
	Get(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*OrderAttachment, error)
	List(context.Context, uuid.UUID, uuid.UUID) ([]*OrderAttachment, error)
	Create(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *OrderAttachment, *AuditEvent) (*OrderAttachment, error)
	RemoveReference(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, *AuditEvent) error
}

type OrderAttachmentUsecase struct {
	repo    OrderAttachmentRepo
	storage OrderAttachmentStorage
}

func NewOrderAttachmentUsecase(repo OrderAttachmentRepo, storage OrderAttachmentStorage) *OrderAttachmentUsecase {
	return &OrderAttachmentUsecase{repo: repo, storage: storage}
}

func (uc *OrderAttachmentUsecase) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderAttachment, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderAttachmentInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, orderID)
}

func (uc *OrderAttachmentUsecase) Register(ctx context.Context, organizationID, actorID, orderID uuid.UUID, token string) (*OrderAttachment, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderAttachmentInvalidArgument
	}
	if uc.storage == nil || !uc.storage.Enabled() {
		return nil, ErrOrderAttachmentStorageUnavailable
	}
	normalized, err := uc.storage.VerifyUpload(ctx, organizationID, actorID, orderID, token)
	if err != nil {
		return nil, err
	}
	normalized, err = normalizeOrderAttachment(normalized, actorID)
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

// OrderAttachmentUpload 是浏览器直传所需的最小授权，不包含云端访问密钥。
type OrderAttachmentUpload struct {
	UploadURL   string
	UploadToken string
	Headers     map[string]string
	ExpiresAt   time.Time
}

type OrderAttachmentAccess struct {
	URL       string
	ExpiresAt time.Time
}

// OrderAttachmentStorage 隔离 OSS 协议、上传授权与实际对象校验。
type OrderAttachmentStorage interface {
	Enabled() bool
	PrepareUpload(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *OrderAttachment) (*OrderAttachmentUpload, error)
	VerifyUpload(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (*OrderAttachment, error)
	Access(context.Context, uuid.UUID, *OrderAttachment, bool) (*OrderAttachmentAccess, error)
	Delete(context.Context, uuid.UUID, string) error
}

func (uc *OrderAttachmentUsecase) PrepareUpload(ctx context.Context, organizationID, actorID, orderID uuid.UUID, input *OrderAttachment) (*OrderAttachmentUpload, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || input == nil {
		return nil, ErrOrderAttachmentInvalidArgument
	}
	candidate := *input
	candidate.IdempotencyKey, candidate.ObjectKey = "prepare", "prepare"
	normalized, err := normalizeOrderAttachment(&candidate, actorID)
	if err != nil {
		return nil, err
	}
	checksum, err := hex.DecodeString(normalized.Checksum)
	if err != nil || len(checksum) != 32 || normalized.Checksum != strings.ToLower(normalized.Checksum) {
		return nil, ErrOrderAttachmentInvalidArgument
	}
	mediaType, params, err := mime.ParseMediaType(normalized.MIMEType)
	if err != nil || !strings.Contains(mediaType, "/") || len(params) > 0 || mediaType != normalized.MIMEType {
		return nil, ErrOrderAttachmentInvalidArgument
	}
	if uc.storage == nil || !uc.storage.Enabled() {
		return nil, ErrOrderAttachmentStorageUnavailable
	}
	if err := uc.repo.CheckEditable(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	return uc.storage.PrepareUpload(ctx, organizationID, actorID, orderID, normalized)
}

func (uc *OrderAttachmentUsecase) GetAccess(ctx context.Context, organizationID, orderID, attachmentID uuid.UUID, download bool) (*OrderAttachmentAccess, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil || attachmentID == uuid.Nil {
		return nil, ErrOrderAttachmentInvalidArgument
	}
	attachment, err := uc.repo.Get(ctx, organizationID, orderID, attachmentID)
	if err != nil {
		return nil, err
	}
	if uc.storage == nil || !uc.storage.Enabled() {
		return nil, ErrOrderAttachmentStorageUnavailable
	}
	return uc.storage.Access(ctx, organizationID, attachment, download)
}
