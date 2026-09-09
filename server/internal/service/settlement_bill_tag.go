package service

import (
	"context"

	"github.com/google/uuid"

	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
)

func (s *SettlementService) ListFinanceBillTagOptions(ctx context.Context, request *v1.ListFinanceBillTagOptionsRequest) (*v1.ListFinanceBillTagOptionsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrBusinessTagInvalidArgument)
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForRequestedOrganization(principal, access.FinanceBillRead, false, request.OrganizationId)
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
	return &v1.ListFinanceBillTagOptionsResponse{Tags: businessTagSummariesToFinanceAPI(items), Total: total, TraceId: requestmeta.TraceID(ctx)}, nil
}

// ListFinanceBillTagAssignmentOptions 只为账单标签写入提供候选。读取筛选继续复用
// ListFinanceBillTagOptions 的 bill.read 范围，不能借此接口扩大写入范围。
func (s *SettlementService) ListFinanceBillTagAssignmentOptions(ctx context.Context, request *v1.ListFinanceBillTagAssignmentOptionsRequest) (*v1.ListFinanceBillTagAssignmentOptionsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrBusinessTagInvalidArgument)
	if err != nil {
		return nil, err
	}
	rawOrganizationID := request.GetOrganizationId()
	if rawOrganizationID == "" {
		return nil, biz.ErrBusinessTagInvalidArgument
	}
	organizationIDs, scopeErr := organizationIDsForRequestedOrganization(principal, access.FinanceBillUpdate, true, &rawOrganizationID)
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
	return &v1.ListFinanceBillTagAssignmentOptionsResponse{Tags: businessTagSummariesToFinanceAPI(items), Total: total, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) BatchAssignFinanceBillTags(ctx context.Context, request *v1.BatchAssignFinanceBillTagsRequest) (*v1.BatchAssignFinanceBillTagsResponse, error) {
	principal, billIDs, tagIDs, err := financeBillTagRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	organizationID, err := s.financeBillTagOwnerOrganization(ctx, principal, billIDs)
	if err != nil {
		return nil, err
	}
	affected, err := s.tagUsecase.AssignFinanceBills(ctx, organizationID, principal.UserID, billIDs, tagIDs)
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
	organizationID, err := s.financeBillTagOwnerOrganization(ctx, principal, billIDs)
	if err != nil {
		return nil, err
	}
	affected, err := s.tagUsecase.RemoveFinanceBills(ctx, organizationID, principal.UserID, billIDs, tagIDs)
	if err != nil {
		return nil, err
	}
	return &v1.BatchRemoveFinanceBillTagsResponse{RemovedCount: int32(affected), TraceId: requestmeta.TraceID(ctx)}, nil
}

// financeBillTagOwnerOrganization 先以 bill.update 的可写组织范围定位每张账单，
// 再要求批量对象属于同一组织。实际标签写入仍在 Data 事务中以该组织和全部账单
// ID 重查，避免跨组织批量关联。
func (s *SettlementService) financeBillTagOwnerOrganization(ctx context.Context, principal *biz.Principal, billIDs []uuid.UUID) (uuid.UUID, error) {
	organizationIDs, err := organizationIDsForPermission(principal, access.FinanceBillUpdate, true)
	if err != nil {
		return uuid.Nil, err
	}
	ownerID := uuid.Nil
	for _, billID := range billIDs {
		bill, getErr := s.billUsecase.Get(ctx, organizationIDs, billID)
		if getErr != nil {
			return uuid.Nil, getErr
		}
		if ownerID == uuid.Nil {
			ownerID = bill.OrganizationID
			continue
		}
		if ownerID != bill.OrganizationID {
			return uuid.Nil, biz.ErrBusinessTagInvalidArgument
		}
	}
	if ownerID == uuid.Nil {
		return uuid.Nil, biz.ErrBusinessTagInvalidArgument
	}
	return ownerID, nil
}

func financeBillTagRequest[Req interface {
	GetBillIds() []string
	GetTagIds() []string
}](ctx context.Context, request Req) (*biz.Principal, []uuid.UUID, []uuid.UUID, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, nil, nil, principalErr
	}
	billIDs, tagIDs, err := orderTagBatchIDs(request.GetBillIds(), request.GetTagIds())
	if err != nil {
		return nil, nil, nil, err
	}
	return principal, billIDs, tagIDs, nil
}
