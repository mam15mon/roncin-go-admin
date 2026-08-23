package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
)

// OrderReleasePodService 订单放货凭证服务，只做 DTO 转换、边界校验和用例调用。
type OrderReleasePodService struct {
	v1.UnimplementedOrderReleasePodServiceServer
	usecase *biz.OrderReleasePodUsecase
}

func NewOrderReleasePodService(usecase *biz.OrderReleasePodUsecase) *OrderReleasePodService {
	return &OrderReleasePodService{usecase: usecase}
}

func (s *OrderReleasePodService) ListReleasePods(ctx context.Context, request *v1.ListReleasePodsRequest) (*v1.ListReleasePodsResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderReleasePodInvalidArgument
	}
	items, err := s.usecase.List(ctx, principal.Organization.ID, orderID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.OrderReleasePod, 0, len(items))
	for _, item := range items {
		data = append(data, orderReleasePodToAPI(item))
	}
	return &v1.ListReleasePodsResponse{
		Success: true,
		Code:    0,
		Message: "OK",
		Data:    data,
		TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func (s *OrderReleasePodService) AddReleasePod(ctx context.Context, request *v1.AddReleasePodRequest) (*v1.AddReleasePodResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	orderID, input, err := orderReleasePodInputFromAPI(request.GetOrderId(), request.GetShippingDocumentId(), request.GetReleaseNo(), request.GetPodNo(), request.GetNote())
	if err != nil {
		return nil, err
	}
	created, err := s.usecase.Add(ctx, principal.Organization.ID, principal.UserID, orderID, input)
	if err != nil {
		return nil, err
	}
	return &v1.AddReleasePodResponse{Success: true, Code: 0, Message: "OK", Data: orderReleasePodToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderReleasePodService) UpdateReleasePod(ctx context.Context, request *v1.UpdateReleasePodRequest) (*v1.UpdateReleasePodResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderReleasePodInvalidArgument
	}
	orderID, input, err := orderReleasePodInputFromAPI(request.GetOrderId(), request.GetShippingDocumentId(), request.GetReleaseNo(), request.GetPodNo(), request.GetNote())
	if err != nil {
		return nil, err
	}
	updated, err := s.usecase.Update(ctx, principal.Organization.ID, principal.UserID, orderID, id, input)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateReleasePodResponse{Success: true, Code: 0, Message: "OK", Data: orderReleasePodToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderReleasePodService) TransitionReleasePodStatus(ctx context.Context, request *v1.TransitionReleasePodStatusRequest) (*v1.TransitionReleasePodStatusResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderReleasePodInvalidArgument
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderReleasePodInvalidArgument
	}
	from, err := orderReleasePodStatusFromAPI(request.GetExpectedStatus())
	if err != nil {
		return nil, err
	}
	to, err := orderReleasePodStatusFromAPI(request.GetToStatus())
	if err != nil {
		return nil, err
	}
	updated, err := s.usecase.Transition(ctx, principal.Organization.ID, principal.UserID, orderID, id, from, to)
	if err != nil {
		return nil, err
	}
	return &v1.TransitionReleasePodStatusResponse{Success: true, Code: 0, Message: "OK", Data: orderReleasePodToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderReleasePodService) RemoveReleasePod(ctx context.Context, request *v1.RemoveReleasePodRequest) (*v1.RemoveReleasePodResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderReleasePodInvalidArgument
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderReleasePodInvalidArgument
	}
	if err := s.usecase.Remove(ctx, principal.Organization.ID, principal.UserID, orderID, id); err != nil {
		return nil, err
	}
	return &v1.RemoveReleasePodResponse{
		Success: true,
		Code:    0,
		Message: "OK",
		TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func orderReleasePodToAPI(value *biz.OrderReleasePod) *v1.OrderReleasePod {
	item := &v1.OrderReleasePod{
		Id:        value.ID.String(),
		OrderId:   value.OrderID.String(),
		Status:    orderReleasePodStatusToAPI(value.Status),
		CreatedAt: value.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: value.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if value.ShippingDocumentID != nil {
		documentID := value.ShippingDocumentID.String()
		item.ShippingDocumentId = &documentID
	}
	if value.ReleaseNo != nil {
		item.ReleaseNo = value.ReleaseNo
	}
	if value.PodNo != nil {
		item.PodNo = value.PodNo
	}
	if value.Note != nil {
		item.Note = value.Note
	}
	if value.SignedAt != nil {
		signedAt := value.SignedAt.UTC().Format(time.RFC3339)
		item.SignedAt = &signedAt
	}
	if value.SignedBy != nil {
		signedBy := value.SignedBy.String()
		item.SignedBy = &signedBy
	}
	return item
}

func orderReleasePodInputFromAPI(orderIDText, documentIDText, releaseNo, podNo, note string) (uuid.UUID, *biz.OrderReleasePod, error) {
	orderID, err := uuid.Parse(orderIDText)
	if err != nil {
		return uuid.Nil, nil, biz.ErrOrderReleasePodInvalidArgument
	}
	input := &biz.OrderReleasePod{}
	if documentIDText != "" {
		documentID, err := uuid.Parse(documentIDText)
		if err != nil {
			return uuid.Nil, nil, biz.ErrOrderReleasePodInvalidArgument
		}
		input.ShippingDocumentID = &documentID
	}
	if releaseNo != "" {
		input.ReleaseNo = &releaseNo
	}
	if podNo != "" {
		input.PodNo = &podNo
	}
	if note != "" {
		input.Note = &note
	}
	return orderID, input, nil
}

func orderReleasePodStatusFromAPI(status v1.OrderReleasePodStatus) (biz.OrderReleasePodStatus, error) {
	switch status {
	case v1.OrderReleasePodStatus_ORDER_RELEASE_POD_STATUS_PENDING:
		return biz.OrderReleasePodStatusPending, nil
	case v1.OrderReleasePodStatus_ORDER_RELEASE_POD_STATUS_SIGNED:
		return biz.OrderReleasePodStatusSigned, nil
	case v1.OrderReleasePodStatus_ORDER_RELEASE_POD_STATUS_RETURNED:
		return biz.OrderReleasePodStatusReturned, nil
	default:
		return "", biz.ErrOrderReleasePodInvalidArgument
	}
}

func orderReleasePodStatusToAPI(status biz.OrderReleasePodStatus) v1.OrderReleasePodStatus {
	switch status {
	case biz.OrderReleasePodStatusPending:
		return v1.OrderReleasePodStatus_ORDER_RELEASE_POD_STATUS_PENDING
	case biz.OrderReleasePodStatusSigned:
		return v1.OrderReleasePodStatus_ORDER_RELEASE_POD_STATUS_SIGNED
	case biz.OrderReleasePodStatusReturned:
		return v1.OrderReleasePodStatus_ORDER_RELEASE_POD_STATUS_RETURNED
	default:
		return v1.OrderReleasePodStatus_ORDER_RELEASE_POD_STATUS_UNSPECIFIED
	}
}

var _ v1.OrderReleasePodServiceServer = (*OrderReleasePodService)(nil)
