package service

import (
	"github.com/google/uuid"
	"context"

	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
)

func (s *SettlementService) ListFinanceBillTagOptions(ctx context.Context, request *v1.ListFinanceBillTagOptionsRequest) (*v1.ListFinanceBillTagOptionsResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	page, pageSize, err := orderTagPageValues(request.GetPage(), request.GetPageSize())
	if err != nil {
		return nil, err
	}
	items, total, err := s.tagUsecase.ListTagOptions(ctx, principal.Organization.ID, request.GetKeyword(), page, pageSize)
	if err != nil {
		return nil, err
	}
	return &v1.ListFinanceBillTagOptionsResponse{Tags: businessTagSummariesToFinanceAPI(items), Total: total, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) BatchAssignFinanceBillTags(ctx context.Context, request *v1.BatchAssignFinanceBillTagsRequest) (*v1.BatchAssignFinanceBillTagsResponse, error) {
	principal, billIDs, tagIDs, err := financeBillTagRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	affected, err := s.tagUsecase.AssignFinanceBills(ctx, principal.Organization.ID, principal.UserID, billIDs, tagIDs)
	if err != nil {
		return nil, err
	}
	return &v1.BatchAssignFinanceBillTagsResponse{AssignedCount: int32(affected), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) BatchRemoveFinanceBillTags(ctx context.Context, request *v1.BatchRemoveFinanceBillTagsRequest) (*v1.BatchRemoveFinanceBillTagsResponse, error) {
	principal, billIDs, tagIDs, err := financeBillTagRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	affected, err := s.tagUsecase.RemoveFinanceBills(ctx, principal.Organization.ID, principal.UserID, billIDs, tagIDs)
	if err != nil {
		return nil, err
	}
	return &v1.BatchRemoveFinanceBillTagsResponse{RemovedCount: int32(affected), TraceId: requestmeta.TraceID(ctx)}, nil
}

func financeBillTagRequest[Req interface {
	GetBillIds() []string
	GetTagIds() []string
}](ctx context.Context, request Req) (*biz.Principal, []uuid.UUID, []uuid.UUID, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, nil, nil, biz.ErrSessionRequired
	}
	billIDs, tagIDs, err := orderTagBatchIDs(request.GetBillIds(), request.GetTagIds())
	if err != nil {
		return nil, nil, nil, err
	}
	return principal, billIDs, tagIDs, nil
}
