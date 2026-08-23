package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
)

// OrderShippingDocumentService 订单提单服务，只做 DTO 转换、边界校验和用例调用。
type OrderShippingDocumentService struct {
	v1.UnimplementedOrderShippingDocumentServiceServer
	usecase *biz.OrderShippingDocumentUsecase
}

func NewOrderShippingDocumentService(usecase *biz.OrderShippingDocumentUsecase) *OrderShippingDocumentService {
	return &OrderShippingDocumentService{usecase: usecase}
}

func (s *OrderShippingDocumentService) ListShippingDocuments(ctx context.Context, request *v1.ListShippingDocumentsRequest) (*v1.ListShippingDocumentsResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderShippingDocumentInvalidArgument
	}
	items, err := s.usecase.List(ctx, principal.Organization.ID, orderID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.OrderShippingDocument, 0, len(items))
	for _, item := range items {
		data = append(data, orderShippingDocumentToAPI(item))
	}
	return &v1.ListShippingDocumentsResponse{
		Success: true,
		Code:    0,
		Message: "OK",
		Data:    data,
		TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func (s *OrderShippingDocumentService) AddShippingDocument(ctx context.Context, request *v1.AddShippingDocumentRequest) (*v1.AddShippingDocumentResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	orderID, input, err := orderShippingDocumentInputFromAPI(request.GetOrderId(), request.GetMasterNo(), request.GetHouseNo(), request.GetReleaseType(), request.GetNote())
	if err != nil {
		return nil, err
	}
	created, err := s.usecase.Add(ctx, principal.Organization.ID, principal.UserID, orderID, input)
	if err != nil {
		return nil, err
	}
	return &v1.AddShippingDocumentResponse{Success: true, Code: 0, Message: "OK", Data: orderShippingDocumentToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderShippingDocumentService) UpdateShippingDocument(ctx context.Context, request *v1.UpdateShippingDocumentRequest) (*v1.UpdateShippingDocumentResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderShippingDocumentInvalidArgument
	}
	orderID, input, err := orderShippingDocumentInputFromAPI(request.GetOrderId(), request.GetMasterNo(), request.GetHouseNo(), request.GetReleaseType(), request.GetNote())
	if err != nil {
		return nil, err
	}
	updated, err := s.usecase.Update(ctx, principal.Organization.ID, principal.UserID, orderID, id, input)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateShippingDocumentResponse{Success: true, Code: 0, Message: "OK", Data: orderShippingDocumentToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderShippingDocumentService) TransitionShippingDocumentStatus(ctx context.Context, request *v1.TransitionShippingDocumentStatusRequest) (*v1.TransitionShippingDocumentStatusResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderShippingDocumentInvalidArgument
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderShippingDocumentInvalidArgument
	}
	from, err := orderShippingDocumentStatusFromAPI(request.GetExpectedStatus())
	if err != nil {
		return nil, err
	}
	to, err := orderShippingDocumentStatusFromAPI(request.GetToStatus())
	if err != nil {
		return nil, err
	}
	updated, err := s.usecase.Transition(ctx, principal.Organization.ID, principal.UserID, orderID, id, from, to)
	if err != nil {
		return nil, err
	}
	return &v1.TransitionShippingDocumentStatusResponse{Success: true, Code: 0, Message: "OK", Data: orderShippingDocumentToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderShippingDocumentService) RemoveShippingDocument(ctx context.Context, request *v1.RemoveShippingDocumentRequest) (*v1.RemoveShippingDocumentResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderShippingDocumentInvalidArgument
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderShippingDocumentInvalidArgument
	}
	if err := s.usecase.Remove(ctx, principal.Organization.ID, principal.UserID, orderID, id); err != nil {
		return nil, err
	}
	return &v1.RemoveShippingDocumentResponse{
		Success: true,
		Code:    0,
		Message: "OK",
		TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func orderShippingDocumentToAPI(value *biz.OrderShippingDocument) *v1.OrderShippingDocument {
	return &v1.OrderShippingDocument{
		Id:          value.ID.String(),
		OrderId:     value.OrderID.String(),
		MasterNo:    value.MasterNo,
		HouseNo:     value.HouseNo,
		ReleaseType: value.ReleaseType,
		Status:      orderShippingDocumentStatusToAPI(value.Status),
		Note:        value.Note,
		CreatedAt:   value.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   value.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func orderShippingDocumentInputFromAPI(orderIDText, masterNo, houseNo, releaseType, note string) (uuid.UUID, *biz.OrderShippingDocument, error) {
	orderID, err := uuid.Parse(orderIDText)
	if err != nil {
		return uuid.Nil, nil, biz.ErrOrderShippingDocumentInvalidArgument
	}
	input := &biz.OrderShippingDocument{
		MasterNo: masterNo,
		HouseNo:  houseNo,
	}
	if releaseType != "" {
		input.ReleaseType = &releaseType
	}
	if note != "" {
		input.Note = &note
	}
	return orderID, input, nil
}

func orderShippingDocumentStatusFromAPI(status v1.OrderShippingDocumentStatus) (biz.OrderShippingDocumentStatus, error) {
	switch status {
	case v1.OrderShippingDocumentStatus_ORDER_SHIPPING_DOCUMENT_STATUS_DRAFT:
		return biz.OrderShippingDocumentStatusDraft, nil
	case v1.OrderShippingDocumentStatus_ORDER_SHIPPING_DOCUMENT_STATUS_CONFIRMED:
		return biz.OrderShippingDocumentStatusConfirmed, nil
	case v1.OrderShippingDocumentStatus_ORDER_SHIPPING_DOCUMENT_STATUS_RELEASED:
		return biz.OrderShippingDocumentStatusReleased, nil
	default:
		return "", biz.ErrOrderShippingDocumentInvalidArgument
	}
}

func orderShippingDocumentStatusToAPI(status biz.OrderShippingDocumentStatus) v1.OrderShippingDocumentStatus {
	switch status {
	case biz.OrderShippingDocumentStatusDraft:
		return v1.OrderShippingDocumentStatus_ORDER_SHIPPING_DOCUMENT_STATUS_DRAFT
	case biz.OrderShippingDocumentStatusConfirmed:
		return v1.OrderShippingDocumentStatus_ORDER_SHIPPING_DOCUMENT_STATUS_CONFIRMED
	case biz.OrderShippingDocumentStatusReleased:
		return v1.OrderShippingDocumentStatus_ORDER_SHIPPING_DOCUMENT_STATUS_RELEASED
	default:
		return v1.OrderShippingDocumentStatus_ORDER_SHIPPING_DOCUMENT_STATUS_UNSPECIFIED
	}
}

var _ v1.OrderShippingDocumentServiceServer = (*OrderShippingDocumentService)(nil)
