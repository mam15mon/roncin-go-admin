package service

import (
	"context"

	workbenchv1 "github.com/roncin/roncin-go-admin/server/api/workbench/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func (s *WorkbenchService) ListMyUnbilledFees(ctx context.Context, request *workbenchv1.ListMyUnbilledFeesRequest) (*workbenchv1.ListMyUnbilledFeesResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	page, size, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrWorkbenchInvalid)
	if err != nil {
		return nil, err
	}
	result, err := s.usecase.ListMyUnbilledFees(ctx, workbenchScopeFromPrincipal(principal), biz.WorkbenchTodoFilter{Page: page, PageSize: size})
	if err != nil {
		return nil, err
	}
	items := make([]*workbenchv1.WorkbenchUnbilledFee, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, &workbenchv1.WorkbenchUnbilledFee{
			FeeId:               item.FeeID.String(),
			OrderId:             item.OrderID.String(),
			OrderNo:             item.OrderNo,
			BusinessType:        item.BusinessType,
			CustomerName:        item.CustomerName,
			OrderLocked:         item.OrderLocked,
			TerminationStatus:   item.TerminationStatus,
			ClosureStatus:       item.ClosureStatus,
			Direction:           string(item.Direction),
			FeeName:             item.FeeName,
			SettlementPartyName: item.SettlementPartyName,
			Currency:            item.Currency,
			TotalAmount:         item.TotalAmount.StringFixed(8),
			CreatedAt:           workbenchTime(item.CreatedAt),
			HasPendingChange:    item.HasPendingChange,
			BlockedReasons:      item.BlockedReasons,
		})
	}
	return okList(ctx, &workbenchv1.ListMyUnbilledFeesResponse{Data: items, Total: int64(result.Total), Page: int32(result.Page), PageSize: int32(result.PageSize)}), nil
}

func (s *WorkbenchService) ListMyOpenAbnormalCases(ctx context.Context, request *workbenchv1.ListMyOpenAbnormalCasesRequest) (*workbenchv1.ListMyOpenAbnormalCasesResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	page, size, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrWorkbenchInvalid)
	if err != nil {
		return nil, err
	}
	result, err := s.usecase.ListMyOpenAbnormalCases(ctx, workbenchScopeFromPrincipal(principal), biz.WorkbenchTodoFilter{Page: page, PageSize: size})
	if err != nil {
		return nil, err
	}
	items := make([]*workbenchv1.WorkbenchOpenAbnormalCase, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, &workbenchv1.WorkbenchOpenAbnormalCase{
			Id:                         item.ID.String(),
			OrderId:                    item.OrderID.String(),
			OrderNo:                    item.OrderNo,
			BusinessType:               item.BusinessType,
			CustomerName:               item.CustomerName,
			OrderLocked:                item.OrderLocked,
			TerminationStatus:          item.TerminationStatus,
			ClosureStatus:              item.ClosureStatus,
			AbnormalCaseId:             item.AbnormalCaseID.String(),
			AbnormalCaseName:           item.AbnormalCaseName,
			MarkedAt:                   workbenchTime(item.MarkedAt),
			BusinessWriteBlockedReason: item.BusinessWriteBlockedReason,
		})
	}
	return okList(ctx, &workbenchv1.ListMyOpenAbnormalCasesResponse{Data: items, Total: int64(result.Total), Page: int32(result.Page), PageSize: int32(result.PageSize)}), nil
}
