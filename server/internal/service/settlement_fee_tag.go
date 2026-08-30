package service

import (
	"context"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
)

func businessTagSummariesToFinanceAPI(items []*biz.BusinessTagSummary) []*v1.BusinessTagSummary {
	if len(items) == 0 {
		return nil
	}
	result := make([]*v1.BusinessTagSummary, 0, len(items))
	for _, item := range items {
		result = append(result, &v1.BusinessTagSummary{Id: item.ID.String(), Name: item.Name, GroupId: item.GroupID.String(), GroupName: item.GroupName, GroupColor: item.GroupColor, Enabled: item.Enabled})
	}
	return result
}

func (s *SettlementService) ListFinanceFeeTagOptions(ctx context.Context, request *v1.ListFinanceFeeTagOptionsRequest) (*v1.ListFinanceFeeTagOptionsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := orderTagPageValues(request.GetPage(), request.GetPageSize())
	if err != nil {
		return nil, err
	}
	items, total, err := s.tagUsecase.ListTagOptions(ctx, principal.Organization.ID, request.GetKeyword(), page, pageSize)
	if err != nil {
		return nil, err
	}
	return &v1.ListFinanceFeeTagOptionsResponse{Tags: businessTagSummariesToFinanceAPI(items), Total: total, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) BatchAssignFinanceFeeTags(ctx context.Context, request *v1.BatchAssignFinanceFeeTagsRequest) (*v1.BatchAssignFinanceFeeTagsResponse, error) {
	principal, feeIDs, tagIDs, err := financeFeeTagRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	affected, err := s.tagUsecase.AssignOrderFeesInLedger(ctx, principal.Organization.ID, principal.UserID, feeIDs, tagIDs)
	if err != nil {
		return nil, err
	}
	return &v1.BatchAssignFinanceFeeTagsResponse{AssignedCount: int32(affected), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) BatchRemoveFinanceFeeTags(ctx context.Context, request *v1.BatchRemoveFinanceFeeTagsRequest) (*v1.BatchRemoveFinanceFeeTagsResponse, error) {
	principal, feeIDs, tagIDs, err := financeFeeTagRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	affected, err := s.tagUsecase.RemoveOrderFeesInLedger(ctx, principal.Organization.ID, principal.UserID, feeIDs, tagIDs)
	if err != nil {
		return nil, err
	}
	return &v1.BatchRemoveFinanceFeeTagsResponse{RemovedCount: int32(affected), TraceId: requestmeta.TraceID(ctx)}, nil
}

func financeFeeTagRequest[Req interface {
	GetFeeIds() []string
	GetTagIds() []string
}](ctx context.Context, request Req) (*biz.Principal, []uuid.UUID, []uuid.UUID, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, nil, nil, principalErr
	}
	feeIDs, tagIDs, err := orderTagBatchIDs(request.GetFeeIds(), request.GetTagIds())
	if err != nil {
		return nil, nil, nil, err
	}
	return principal, feeIDs, tagIDs, nil
}
