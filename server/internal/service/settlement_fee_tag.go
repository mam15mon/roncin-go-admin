package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
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
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrBusinessTagInvalidArgument)
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForRequestedOrganization(principal, access.FinanceFeeRead, false, request.OrganizationId)
	if scopeErr != nil {
		return nil, scopeErr
	}
	if len(organizationIDs) != 1 {
		return nil, biz.ErrFinanceLedgerInvalidArgument
	}
	items, total, err := s.tagUsecase.ListTagOptions(ctx, organizationIDs[0], request.GetKeyword(), page, pageSize)
	if err != nil {
		return nil, err
	}
	return &v1.ListFinanceFeeTagOptionsResponse{Tags: businessTagSummariesToFinanceAPI(items), Total: total, TraceId: requestmeta.TraceID(ctx)}, nil
}

// ListFinanceFeeTagAssignmentOptions 只为费用标签写入提供候选。读取筛选继续复用
// ListFinanceFeeTagOptions 的 fee.read 范围，不能借此接口扩大写入范围。
func (s *SettlementService) ListFinanceFeeTagAssignmentOptions(ctx context.Context, request *v1.ListFinanceFeeTagAssignmentOptionsRequest) (*v1.ListFinanceFeeTagAssignmentOptionsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrBusinessTagInvalidArgument)
	if err != nil {
		return nil, err
	}
	rawOrganizationID := request.GetOrganizationId()
	if strings.TrimSpace(rawOrganizationID) == "" {
		return nil, biz.ErrBusinessTagInvalidArgument
	}
	organizationIDs, scopeErr := organizationIDsForRequestedOrganization(principal, access.FinanceFeeTag, true, &rawOrganizationID)
	if scopeErr != nil {
		return nil, scopeErr
	}
	if len(organizationIDs) != 1 {
		return nil, biz.ErrBusinessTagInvalidArgument
	}
	items, total, err := s.tagUsecase.ListTagOptions(ctx, organizationIDs[0], request.GetKeyword(), page, pageSize)
	if err != nil {
		return nil, err
	}
	return &v1.ListFinanceFeeTagAssignmentOptionsResponse{Tags: businessTagSummariesToFinanceAPI(items), Total: total, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) BatchAssignFinanceFeeTags(ctx context.Context, request *v1.BatchAssignFinanceFeeTagsRequest) (*v1.BatchAssignFinanceFeeTagsResponse, error) {
	principal, feeIDs, tagIDs, organizationID, err := s.financeFeeTagRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	affected, err := s.tagUsecase.AssignOrderFeesInLedger(ctx, organizationID, principal.UserID, feeIDs, tagIDs)
	if err != nil {
		return nil, err
	}
	return &v1.BatchAssignFinanceFeeTagsResponse{AssignedCount: int32(affected), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) BatchRemoveFinanceFeeTags(ctx context.Context, request *v1.BatchRemoveFinanceFeeTagsRequest) (*v1.BatchRemoveFinanceFeeTagsResponse, error) {
	principal, feeIDs, tagIDs, organizationID, err := s.financeFeeTagRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	affected, err := s.tagUsecase.RemoveOrderFeesInLedger(ctx, organizationID, principal.UserID, feeIDs, tagIDs)
	if err != nil {
		return nil, err
	}
	return &v1.BatchRemoveFinanceFeeTagsResponse{RemovedCount: int32(affected), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) financeFeeTagRequest(ctx context.Context, request interface {
	GetFeeIds() []string
	GetTagIds() []string
	GetOrganizationId() string
}) (*biz.Principal, []uuid.UUID, []uuid.UUID, uuid.UUID, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, nil, nil, uuid.Nil, principalErr
	}
	feeIDs, tagIDs, err := orderTagBatchIDs(request.GetFeeIds(), request.GetTagIds())
	if err != nil {
		return nil, nil, nil, uuid.Nil, err
	}
	organizationID, parseErr := uuid.Parse(strings.TrimSpace(request.GetOrganizationId()))
	if parseErr != nil {
		return nil, nil, nil, uuid.Nil, biz.ErrFinanceLedgerInvalidArgument
	}
	organizationIDs, scopeErr := organizationIDsForPermission(principal, access.FinanceFeeTag, true)
	if scopeErr != nil {
		return nil, nil, nil, uuid.Nil, scopeErr
	}
	if !uuidIn(organizationID, organizationIDs) {
		return nil, nil, nil, uuid.Nil, biz.ErrPermissionDenied
	}
	resolvedID, resolveErr := s.usecase.ResolveFeeLedgerOrganization(ctx, organizationIDs, feeIDs)
	if resolveErr != nil {
		return nil, nil, nil, uuid.Nil, resolveErr
	}
	if resolvedID != organizationID {
		return nil, nil, nil, uuid.Nil, biz.ErrPermissionDenied
	}
	return principal, feeIDs, tagIDs, organizationID, nil
}
