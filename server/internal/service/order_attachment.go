package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type OrderAttachmentService struct {
	v1.UnimplementedOrderAttachmentServiceServer
	usecase *biz.OrderAttachmentUsecase
}

func NewOrderAttachmentService(usecase *biz.OrderAttachmentUsecase) *OrderAttachmentService {
	return &OrderAttachmentService{usecase: usecase}
}

func (s *OrderAttachmentService) ListAttachments(ctx context.Context, request *v1.ListAttachmentsRequest) (*v1.ListAttachmentsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderAttachmentInvalidArgument
	}
	items, err := s.usecase.List(ctx, principal.Organization.ID, orderID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.OrderAttachment, 0, len(items))
	for _, item := range items {
		data = append(data, orderAttachmentToAPI(item))
	}
	return okList(ctx, &v1.ListAttachmentsResponse{
		Data: data,
	}), nil
}

func (s *OrderAttachmentService) RegisterAttachment(ctx context.Context, request *v1.RegisterAttachmentRequest) (*v1.RegisterAttachmentResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderAttachmentInvalidArgument
	}
	created, err := s.usecase.Register(ctx, principal.Organization.ID, principal.UserID, orderID, request.GetUploadToken())
	if err != nil {
		return nil, err
	}
	return orderAttachmentResponse(ctx, created), nil
}

func (s *OrderAttachmentService) RemoveAttachmentReference(ctx context.Context, request *v1.RemoveAttachmentReferenceRequest) (*v1.RemoveAttachmentReferenceResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderAttachmentInvalidArgument
	}
	attachmentID, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderAttachmentInvalidArgument
	}
	if err := s.usecase.RemoveReference(ctx, principal.Organization.ID, principal.UserID, orderID, attachmentID); err != nil {
		return nil, err
	}
	return ok(ctx, &v1.RemoveAttachmentReferenceResponse{}), nil
}

func orderAttachmentResponse(ctx context.Context, value *biz.OrderAttachment) *v1.RegisterAttachmentResponse {
	return ok(ctx, &v1.RegisterAttachmentResponse{
		Data: orderAttachmentToAPI(value),
	})

}

func orderAttachmentToAPI(value *biz.OrderAttachment) *v1.OrderAttachment {
	result := &v1.OrderAttachment{
		Id:             value.ID.String(),
		OrderId:        value.OrderID.String(),
		DocType:        value.DocType,
		IdempotencyKey: value.IdempotencyKey,
		FileName:       value.FileName,
		MimeType:       value.MIMEType,
		FileSize:       value.FileSize,
		ObjectKey:      value.ObjectKey,
		CreatedAt:      value.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      value.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if value.Checksum != "" {
		checksum := value.Checksum
		result.Checksum = &checksum
	}
	if value.UploadedBy != nil {
		uploadedBy := value.UploadedBy.String()
		result.UploadedBy = &uploadedBy
	}
	if value.AssetID != uuid.Nil {
		assetID := value.AssetID.String()
		result.AssetId = &assetID
	}
	return result
}

var _ v1.OrderAttachmentServiceServer = (*OrderAttachmentService)(nil)

func (s *OrderAttachmentService) PrepareAttachmentUpload(ctx context.Context, request *v1.PrepareAttachmentUploadRequest) (*v1.PrepareAttachmentUploadResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderAttachmentInvalidArgument
	}
	upload, err := s.usecase.PrepareUpload(ctx, principal.Organization.ID, principal.UserID, orderID, &biz.OrderAttachment{
		DocType: request.GetDocType(), FileName: request.GetFileName(), MIMEType: request.GetMimeType(), FileSize: request.GetFileSize(), Checksum: request.GetChecksum(),
	})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.PrepareAttachmentUploadResponse{Data: &v1.AttachmentUpload{UploadUrl: upload.UploadURL, UploadToken: upload.UploadToken, Headers: upload.Headers, ExpiresAt: upload.ExpiresAt.UTC().Format(time.RFC3339)}}), nil
}

func (s *OrderAttachmentService) GetAttachmentAccess(ctx context.Context, request *v1.GetAttachmentAccessRequest) (*v1.GetAttachmentAccessResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderAttachmentInvalidArgument
	}
	attachmentID, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderAttachmentInvalidArgument
	}
	access, err := s.usecase.GetAccess(ctx, principal.Organization.ID, orderID, attachmentID, request.GetDownload())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.GetAttachmentAccessResponse{Data: &v1.AttachmentAccess{Url: access.URL, ExpiresAt: access.ExpiresAt.UTC().Format(time.RFC3339)}}), nil
}
