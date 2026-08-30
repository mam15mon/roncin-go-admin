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
	input := &biz.OrderAttachment{
		DocType:        request.GetDocType(),
		IdempotencyKey: request.GetIdempotencyKey(),
		FileName:       request.GetFileName(),
		MIMEType:       request.GetMimeType(),
		FileSize:       request.GetFileSize(),
		ObjectKey:      request.GetObjectKey(),
		Checksum:       request.GetChecksum(),
	}
	created, err := s.usecase.Register(ctx, principal.Organization.ID, principal.UserID, orderID, input)
	if err != nil {
		return nil, err
	}
	return orderAttachmentResponse(ctx, created), nil
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
	return result
}

var _ v1.OrderAttachmentServiceServer = (*OrderAttachmentService)(nil)
