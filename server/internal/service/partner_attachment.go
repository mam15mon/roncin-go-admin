package service

import (
	"context"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"

	"github.com/google/uuid"
)

func (s *PartnerService) ListPartnerAttachments(ctx context.Context, request *v1.ListPartnerAttachmentsRequest) (*v1.ListPartnerAttachmentsResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	if err != nil {
		return nil, biz.ErrPartnerAttachmentInvalidArgument
	}
	items, err := s.attachmentUsecase.List(ctx, principal.Organization.ID, partnerID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.PartnerAttachment, 0, len(items))
	for _, item := range items {
		data = append(data, partnerAttachmentToAPI(item))
	}
	return &v1.ListPartnerAttachmentsResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) RegisterPartnerAttachment(ctx context.Context, request *v1.RegisterPartnerAttachmentRequest) (*v1.RegisterPartnerAttachmentResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	if err != nil {
		return nil, biz.ErrPartnerAttachmentInvalidArgument
	}
	created, err := s.attachmentUsecase.Register(ctx, principal.Organization.ID, principal.UserID, partnerID, &biz.PartnerAttachment{
		IdempotencyKey: request.GetIdempotencyKey(), FileName: request.GetFileName(), MIMEType: request.GetMimeType(), FileSize: request.GetFileSize(), ObjectKey: request.GetObjectKey(), Checksum: request.GetChecksum(),
	})
	if err != nil {
		return nil, err
	}
	return partnerAttachmentResponse(ctx, created), nil
}

func partnerAttachmentToAPI(value *biz.PartnerAttachment) *v1.PartnerAttachment {
	result := &v1.PartnerAttachment{
		Id: value.ID.String(), PartnerId: value.PartnerID.String(), IdempotencyKey: value.IdempotencyKey, FileName: value.FileName,
		MimeType: value.MIMEType, FileSize: value.FileSize, ObjectKey: value.ObjectKey, Checksum: value.Checksum,
		CreatedAt: value.CreatedAt.Format(time.RFC3339), UpdatedAt: value.UpdatedAt.Format(time.RFC3339),
	}
	if value.UploadedBy != nil {
		result.UploadedBy = value.UploadedBy.String()
	}
	return result
}

func partnerAttachmentResponse(ctx context.Context, value *biz.PartnerAttachment) *v1.RegisterPartnerAttachmentResponse {
	return &v1.RegisterPartnerAttachmentResponse{Success: true, Code: 0, Message: "OK", Data: partnerAttachmentToAPI(value), TraceId: requestmeta.TraceID(ctx)}
}
