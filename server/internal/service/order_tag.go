package service

import (
	"context"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
)

// OrderTagService 负责订单标签候选与批量维护的请求转换。
type OrderTagService struct {
	v1.UnimplementedOrderTagServiceServer
	usecase *biz.BusinessTagUsecase
}

func NewOrderTagService(usecase *biz.BusinessTagUsecase) *OrderTagService {
	return &OrderTagService{usecase: usecase}
}

func (s *OrderTagService) ListOrderTagOptions(ctx context.Context, request *v1.ListOrderTagOptionsRequest) (*v1.ListOrderTagOptionsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := orderTagPageValues(request.GetPage(), request.GetPageSize())
	if err != nil {
		return nil, err
	}
	items, total, err := s.usecase.ListTagOptions(ctx, principal.Organization.ID, request.GetKeyword(), page, pageSize)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.BusinessTagSummary, 0, len(items))
	for _, item := range items {
		data = append(data, businessTagSummaryToAPI(item))
	}
	return &v1.ListOrderTagOptionsResponse{Tags: data, Total: total, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderTagService) BatchAssignOrderTags(ctx context.Context, request *v1.BatchAssignOrderTagsRequest) (*v1.BatchAssignOrderTagsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderIDs, tagIDs, err := orderTagBatchIDs(request.GetOrderIds(), request.GetTagIds())
	if err != nil {
		return nil, err
	}
	affected, err := s.usecase.AssignOrderTags(ctx, principal.Organization.ID, principal.UserID, orderBusinessTypeFromAPI(request.GetBusinessType()), orderIDs, tagIDs)
	if err != nil {
		return nil, err
	}
	return &v1.BatchAssignOrderTagsResponse{AssignedCount: int32(affected), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderTagService) BatchRemoveOrderTags(ctx context.Context, request *v1.BatchRemoveOrderTagsRequest) (*v1.BatchRemoveOrderTagsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderIDs, tagIDs, err := orderTagBatchIDs(request.GetOrderIds(), request.GetTagIds())
	if err != nil {
		return nil, err
	}
	affected, err := s.usecase.RemoveOrderTags(ctx, principal.Organization.ID, principal.UserID, orderBusinessTypeFromAPI(request.GetBusinessType()), orderIDs, tagIDs)
	if err != nil {
		return nil, err
	}
	return &v1.BatchRemoveOrderTagsResponse{RemovedCount: int32(affected), TraceId: requestmeta.TraceID(ctx)}, nil
}

func orderTagPageValues(page, pageSize int32) (int, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if !biz.ValidListPagination(int(page), int(pageSize)) {
		return 0, 0, biz.ErrBusinessTagInvalidArgument
	}
	return int(page), int(pageSize), nil
}

func orderTagBatchIDs(orderIDValues, tagIDValues []string) ([]uuid.UUID, []uuid.UUID, error) {
	orderIDs, err := parseUUIDList(orderIDValues)
	if err != nil {
		return nil, nil, err
	}
	tagIDs, err := parseUUIDList(tagIDValues)
	if err != nil {
		return nil, nil, err
	}
	return orderIDs, tagIDs, nil
}

func parseUUIDList(values []string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, biz.ErrBusinessTagInvalidArgument
		}
		result = append(result, id)
	}
	return result, nil
}

func businessTagSummaryToAPI(item *biz.BusinessTagSummary) *v1.BusinessTagSummary {
	if item == nil {
		return nil
	}
	return &v1.BusinessTagSummary{Id: item.ID.String(), Name: item.Name, GroupId: item.GroupID.String(), GroupName: item.GroupName, GroupColor: item.GroupColor, Enabled: item.Enabled}
}
