package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
)

// OrderAbnormalCaseService 订单异常标记服务，只做 DTO 转换、边界校验和用例调用。
type OrderAbnormalCaseService struct {
	v1.UnimplementedOrderAbnormalCaseServiceServer
	usecase *biz.OrderAbnormalCaseUsecase
}

func NewOrderAbnormalCaseService(usecase *biz.OrderAbnormalCaseUsecase) *OrderAbnormalCaseService {
	return &OrderAbnormalCaseService{usecase: usecase}
}

func (s *OrderAbnormalCaseService) ListAbnormalCases(ctx context.Context, request *v1.ListAbnormalCasesRequest) (*v1.ListAbnormalCasesResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderAbnormalCaseInvalidArgument
	}
	items, err := s.usecase.List(ctx, principal.Organization.ID, orderID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.OrderAbnormalCase, 0, len(items))
	for _, item := range items {
		data = append(data, orderAbnormalCaseToAPI(item))
	}
	return &v1.ListAbnormalCasesResponse{
		Success: true,
		Code:    0,
		Message: "OK",
		Data:    data,
		TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func (s *OrderAbnormalCaseService) MarkAbnormalCase(ctx context.Context, request *v1.MarkAbnormalCaseRequest) (*v1.MarkAbnormalCaseResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderAbnormalCaseInvalidArgument
	}
	abnormalCaseID, err := uuid.Parse(request.GetAbnormalCaseId())
	if err != nil {
		return nil, biz.ErrOrderAbnormalCaseInvalidArgument
	}
	marked, err := s.usecase.Mark(ctx, principal.Organization.ID, orderID, principal.UserID, abnormalCaseID)
	if err != nil {
		return nil, err
	}
	return &v1.MarkAbnormalCaseResponse{Success: true, Code: 0, Message: "OK", Data: orderAbnormalCaseToAPI(marked), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderAbnormalCaseService) ResolveAbnormalCase(ctx context.Context, request *v1.ResolveAbnormalCaseRequest) (*v1.ResolveAbnormalCaseResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderAbnormalCaseInvalidArgument
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderAbnormalCaseInvalidArgument
	}
	resolved, err := s.usecase.Resolve(ctx, principal.Organization.ID, orderID, principal.UserID, id)
	if err != nil {
		return nil, err
	}
	return &v1.ResolveAbnormalCaseResponse{Success: true, Code: 0, Message: "OK", Data: orderAbnormalCaseToAPI(resolved), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderAbnormalCaseService) RemoveAbnormalCase(ctx context.Context, request *v1.RemoveAbnormalCaseRequest) (*v1.RemoveAbnormalCaseResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderAbnormalCaseInvalidArgument
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderAbnormalCaseInvalidArgument
	}
	if err := s.usecase.Remove(ctx, principal.Organization.ID, principal.UserID, orderID, id); err != nil {
		return nil, err
	}
	return &v1.RemoveAbnormalCaseResponse{
		Success: true,
		Code:    0,
		Message: "OK",
		TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func orderAbnormalCaseToAPI(value *biz.OrderAbnormalCase) *v1.OrderAbnormalCase {
	item := &v1.OrderAbnormalCase{
		Id:             value.ID.String(),
		OrderId:        value.OrderID.String(),
		AbnormalCaseId: value.AbnormalCaseID.String(),
		Status:         orderAbnormalCaseStatusToAPI(value.Status),
		MarkedAt:       value.MarkedAt.UTC().Format(time.RFC3339),
		MarkedBy:       value.MarkedBy.String(),
		CreatedAt:      value.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      value.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if value.ResolvedAt != nil {
		resolvedAt := value.ResolvedAt.UTC().Format(time.RFC3339)
		item.ResolvedAt = &resolvedAt
	}
	if value.ResolvedBy != nil {
		resolvedBy := value.ResolvedBy.String()
		item.ResolvedBy = &resolvedBy
	}
	return item
}

func orderAbnormalCaseStatusToAPI(status biz.OrderAbnormalCaseStatus) v1.OrderAbnormalCaseStatus {
	switch status {
	case biz.OrderAbnormalCaseStatusActive:
		return v1.OrderAbnormalCaseStatus_ORDER_ABNORMAL_CASE_STATUS_ACTIVE
	case biz.OrderAbnormalCaseStatusResolved:
		return v1.OrderAbnormalCaseStatus_ORDER_ABNORMAL_CASE_STATUS_RESOLVED
	default:
		return v1.OrderAbnormalCaseStatus_ORDER_ABNORMAL_CASE_STATUS_UNSPECIFIED
	}
}

var _ v1.OrderAbnormalCaseServiceServer = (*OrderAbnormalCaseService)(nil)
