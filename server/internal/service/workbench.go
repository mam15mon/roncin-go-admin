package service

import (
	"context"
	"time"

	workbenchv1 "github.com/roncin/roncin-go-admin/server/api/workbench/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// WorkbenchService 工作台读接口：只做 DTO 转换、从会话解析当前主体并调用用例，
// 不写业务规则；主体与组织一律取自会话，不接收客户端改写参数。
type WorkbenchService struct {
	workbenchv1.UnimplementedWorkbenchServiceServer
	usecase *biz.WorkbenchUsecase
}

func NewWorkbenchService(usecase *biz.WorkbenchUsecase) *WorkbenchService {
	return &WorkbenchService{usecase: usecase}
}

// workbenchScopeFromPrincipal 把当前登录主体解析为工作台读范围：当前工作区组织、
// 当前用户、服务端财务业务日期与当前组织上的财务权限标记。权限缺失按 false
// 处理而不是报错——个人工作台接口对无财务权限的员工仍然可用。
func workbenchScopeFromPrincipal(p *biz.Principal) biz.WorkbenchScope {
	now := time.Now()
	return biz.WorkbenchScope{
		OrganizationID:       p.Organization.ID,
		UserID:               p.UserID,
		Today:                biz.FinanceBusinessDate(now),
		Now:                  now,
		CommissionReadable:   p.CanAccessOrganizationForPermission(access.FinanceCommissionRead, p.Organization.ID, false),
		CommissionManageable: p.CanAccessOrganizationForPermission(access.FinanceCommissionManage, p.Organization.ID, false),
		IsBootstrapAdmin:     p.IsBootstrapAdmin,
	}
}

func workbenchStatusFromAPI(value workbenchv1.WorkbenchCommissionStatus) biz.CommissionStatus {
	switch value {
	case workbenchv1.WorkbenchCommissionStatus_WORKBENCH_COMMISSION_STATUS_DRAFT:
		return biz.CommissionDraft
	case workbenchv1.WorkbenchCommissionStatus_WORKBENCH_COMMISSION_STATUS_CONFIRMED:
		return biz.CommissionConfirmed
	case workbenchv1.WorkbenchCommissionStatus_WORKBENCH_COMMISSION_STATUS_PAID:
		return biz.CommissionPaid
	case workbenchv1.WorkbenchCommissionStatus_WORKBENCH_COMMISSION_STATUS_CANCELLED:
		return biz.CommissionCancelled
	default:
		return ""
	}
}

func workbenchStatusToAPI(value biz.CommissionStatus) workbenchv1.WorkbenchCommissionStatus {
	switch value {
	case biz.CommissionDraft:
		return workbenchv1.WorkbenchCommissionStatus_WORKBENCH_COMMISSION_STATUS_DRAFT
	case biz.CommissionConfirmed:
		return workbenchv1.WorkbenchCommissionStatus_WORKBENCH_COMMISSION_STATUS_CONFIRMED
	case biz.CommissionPaid:
		return workbenchv1.WorkbenchCommissionStatus_WORKBENCH_COMMISSION_STATUS_PAID
	case biz.CommissionCancelled:
		return workbenchv1.WorkbenchCommissionStatus_WORKBENCH_COMMISSION_STATUS_CANCELLED
	default:
		return workbenchv1.WorkbenchCommissionStatus_WORKBENCH_COMMISSION_STATUS_UNSPECIFIED
	}
}

func workbenchOptionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func workbenchTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func (s *WorkbenchService) GetWorkbenchOverview(ctx context.Context, _ *workbenchv1.GetWorkbenchOverviewRequest) (*workbenchv1.GetWorkbenchOverviewResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	overview, err := s.usecase.GetOverview(ctx, workbenchScopeFromPrincipal(p))
	if err != nil {
		return nil, err
	}
	return ok(ctx, &workbenchv1.GetWorkbenchOverviewResponse{Data: workbenchOverviewToAPI(overview)}), nil
}

func workbenchOverviewToAPI(overview *biz.WorkbenchOverview) *workbenchv1.GetWorkbenchOverviewData {
	// has_commission_eligibility 为 optional bool 且服务端始终赋值，避免前端把
	// 缺省 false 误判为「尚未加载」。
	data := &workbenchv1.GetWorkbenchOverviewData{
		HasCommissionEligibility: &overview.Eligible,
		NextEffectiveDate:        workbenchOptionalString(overview.NextEffectiveDate),
		BaseCurrency:             overview.BaseCurrency,
		RecentOrders:             make([]*workbenchv1.WorkbenchRecentOrder, 0, len(overview.RecentOrders)),
		Todos: &workbenchv1.WorkbenchTodoSummary{
			DraftFeeCount:     int32(overview.Todos.DraftFeeCount),
			OpenAbnormalCount: int32(overview.Todos.OpenAbnormalCount),
		},
	}
	if overview.CommissionSummary != nil {
		data.CommissionSummary = workbenchCommissionSummaryToAPI(overview.CommissionSummary)
	}
	for _, item := range overview.RecentOrders {
		data.RecentOrders = append(data.RecentOrders, workbenchRecentOrderToAPI(item))
	}
	if overview.Finance != nil {
		data.Finance = workbenchFinanceSummaryToAPI(overview.Finance)
	}
	return data
}

func workbenchCommissionSummaryToAPI(summary *biz.WorkbenchCommissionSummary) *workbenchv1.WorkbenchCommissionSummary {
	result := &workbenchv1.WorkbenchCommissionSummary{
		BaseCurrency:            summary.BaseCurrency,
		DraftCount:              int32(summary.DraftCount),
		DraftAmount:             summary.DraftAmount.StringFixed(8),
		ConfirmedCount:          int32(summary.ConfirmedCount),
		ConfirmedAmount:         summary.ConfirmedAmount.StringFixed(8),
		PaidCount:               int32(summary.PaidCount),
		PaidAmount:              summary.PaidAmount.StringFixed(8),
		PaidAmountThisYear:      summary.PaidThisYear.StringFixed(8),
		PaidAmountThisMonth:     summary.PaidThisMonth.StringFixed(8),
		DecreaseDraftCount:      int32(summary.DecreaseDraftCount),
		DecreaseDraftAmount:     summary.DecreaseDraftAmount.StringFixed(8),
		DecreaseConfirmedCount:  int32(summary.DecreaseConfirmedCount),
		DecreaseConfirmedAmount: summary.DecreaseConfirmedAmount.StringFixed(8),
		DecreasePaidCount:       int32(summary.DecreasePaidCount),
		DecreasePaidAmount:      summary.DecreasePaidAmount.StringFixed(8),
	}
	if summary.Estimated != nil {
		result.Estimated = &workbenchv1.WorkbenchEstimatedOpportunity{
			OpportunityCount: int32(summary.Estimated.OpportunityCount),
			EstimatedAmount:  summary.Estimated.EstimatedAmount.StringFixed(8),
			HasMore:          summary.Estimated.HasMore,
		}
	}
	return result
}

func workbenchRecentOrderToAPI(item *biz.WorkbenchRecentOrder) *workbenchv1.WorkbenchRecentOrder {
	return &workbenchv1.WorkbenchRecentOrder{
		OrderId:           item.OrderID.String(),
		OrderNo:           item.OrderNo,
		BusinessType:      item.BusinessType,
		CustomerName:      item.CustomerName,
		FlowStatus:        item.FlowStatus,
		TerminationStatus: item.TerminationStatus,
		OrderDate:         item.OrderDate,
		CreatedAt:         workbenchTime(item.CreatedAt),
	}
}

func workbenchFinanceSummaryToAPI(summary *biz.WorkbenchFinanceSummary) *workbenchv1.WorkbenchFinanceSummary {
	result := &workbenchv1.WorkbenchFinanceSummary{
		CanReadCommission:              summary.CanReadCommission,
		CanManageCommission:            summary.CanManageCommission,
		ConfirmedCommissionCount:       int32(summary.ConfirmedCommissionCount),
		PendingDecreaseCount:           int32(summary.PendingDecreaseCount),
		PendingSupplementApprovalCount: int32(summary.PendingSupplementApprovalCount),
		PendingSupplementApprovals:     make([]*workbenchv1.WorkbenchSupplementApprovalItem, 0, len(summary.PendingSupplementApprovals)),
		SupplementApprovalsTruncated:   summary.SupplementApprovalsTruncated,
	}
	for _, item := range summary.PendingSupplementApprovals {
		result.PendingSupplementApprovals = append(result.PendingSupplementApprovals, &workbenchv1.WorkbenchSupplementApprovalItem{
			RequestId:       item.RequestID.String(),
			OrderId:         item.OrderID.String(),
			OrderNo:         item.OrderNo,
			FeeCode:         item.FeeCode,
			FeeName:         item.FeeName,
			Currency:        item.Currency,
			Amount:          item.Amount.StringFixed(8),
			Reason:          item.Reason,
			RequestedByName: item.RequestedByName,
			RequestedAt:     workbenchTime(item.RequestedAt),
		})
	}
	return result
}

func (s *WorkbenchService) ListMyCommissions(ctx context.Context, r *workbenchv1.ListMyCommissionsRequest) (*workbenchv1.ListMyCommissionsResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(r.GetPage(), r.GetPageSize(), biz.ErrWorkbenchInvalid)
	if err != nil {
		return nil, err
	}
	result, err := s.usecase.ListMyCommissions(ctx, workbenchScopeFromPrincipal(p), biz.WorkbenchCommissionFilter{
		Page:               page,
		PageSize:           pageSize,
		Status:             workbenchStatusFromAPI(r.GetStatus()),
		CommissionDateFrom: financeOptionalString(r.CommissionDateFrom),
		CommissionDateTo:   financeOptionalString(r.CommissionDateTo),
	})
	if err != nil {
		return nil, err
	}
	data := make([]*workbenchv1.WorkbenchMyCommission, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, workbenchMyCommissionToAPI(item))
	}
	return okList(ctx, &workbenchv1.ListMyCommissionsResponse{
		Total:    int64(result.Total),
		Page:     int32(result.Page),
		PageSize: int32(result.PageSize),
		Data:     data,
	}), nil
}

func workbenchMyCommissionToAPI(item *biz.WorkbenchMyCommission) *workbenchv1.WorkbenchMyCommission {
	result := &workbenchv1.WorkbenchMyCommission{
		Id:               item.ID.String(),
		CommissionNo:     item.CommissionNo,
		Status:           workbenchStatusToAPI(item.Status),
		PersonnelRole:    string(item.PersonnelRole),
		RuleName:         item.RuleName,
		CalculationBasis: item.CalculationBasis,
		BaseCurrency:     item.BaseCurrency,
		CommissionAmount: item.CommissionAmount.StringFixed(8),
		CommissionDate:   item.CommissionDate,
		VerificationNo:   item.VerificationNo,
		NettingNo:        item.NettingNo,
		CreatedAt:        workbenchTime(item.CreatedAt),
		Adjustments:      make([]*workbenchv1.WorkbenchMyCommissionAdjustment, 0, len(item.Adjustments)),
	}
	for _, adjustmentItem := range item.Adjustments {
		result.Adjustments = append(result.Adjustments, &workbenchv1.WorkbenchMyCommissionAdjustment{
			Id:           adjustmentItem.ID.String(),
			AdjustmentNo: adjustmentItem.AdjustmentNo,
			Direction:    string(adjustmentItem.Direction),
			Status:       workbenchStatusToAPI(adjustmentItem.Status),
			Amount:       adjustmentItem.Amount.StringFixed(8),
			Reason:       adjustmentItem.Reason,
			CreatedAt:    workbenchTime(adjustmentItem.CreatedAt),
		})
	}
	return result
}

func (s *WorkbenchService) ListMyReceivables(ctx context.Context, r *workbenchv1.ListMyReceivablesRequest) (*workbenchv1.ListMyReceivablesResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(r.GetPage(), r.GetPageSize(), biz.ErrWorkbenchInvalid)
	if err != nil {
		return nil, err
	}
	result, err := s.usecase.ListMyReceivables(ctx, workbenchScopeFromPrincipal(p), biz.WorkbenchReceivableFilter{Page: page, PageSize: pageSize})
	if err != nil {
		return nil, err
	}
	data := make([]*workbenchv1.WorkbenchMyReceivable, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, &workbenchv1.WorkbenchMyReceivable{
			BillId:              item.BillID.String(),
			BillNo:              item.BillNo,
			SettlementPartyName: item.SettlementPartyName,
			Currency:            item.Currency,
			TotalAmount:         item.TotalAmount.StringFixed(8),
			UnverifiedAmount:    item.UnverifiedAmount.StringFixed(8),
			BillDate:            item.BillDate,
			DueDate:             item.DueDate,
			OverdueDays:         item.OverdueDays,
			ConfirmedAt:         workbenchTime(item.ConfirmedAt),
		})
	}
	return okList(ctx, &workbenchv1.ListMyReceivablesResponse{
		Total:    int64(result.Total),
		Page:     int32(result.Page),
		PageSize: int32(result.PageSize),
		Data:     data,
	}), nil
}

func (s *WorkbenchService) ListMyRecentOrders(ctx context.Context, r *workbenchv1.ListMyRecentOrdersRequest) (*workbenchv1.ListMyRecentOrdersResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(r.GetPage(), r.GetPageSize(), biz.ErrWorkbenchInvalid)
	if err != nil {
		return nil, err
	}
	result, err := s.usecase.ListMyRecentOrders(ctx, workbenchScopeFromPrincipal(p), biz.WorkbenchOrderFilter{Page: page, PageSize: pageSize})
	if err != nil {
		return nil, err
	}
	data := make([]*workbenchv1.WorkbenchRecentOrder, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, workbenchRecentOrderToAPI(item))
	}
	return okList(ctx, &workbenchv1.ListMyRecentOrdersResponse{
		Total:    int64(result.Total),
		Page:     int32(result.Page),
		PageSize: int32(result.PageSize),
		Data:     data,
	}), nil
}
