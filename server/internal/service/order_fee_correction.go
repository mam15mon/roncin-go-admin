package service

import (
	"context"
	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"time"
)

func correctionOperationToAPI(value string) v1.OrderFeeCorrectionOperation {
	switch value {
	case "ADD":
		return v1.OrderFeeCorrectionOperation_ORDER_FEE_CORRECTION_OPERATION_ADD
	case "REPLACE":
		return v1.OrderFeeCorrectionOperation_ORDER_FEE_CORRECTION_OPERATION_REPLACE
	case "VOID":
		return v1.OrderFeeCorrectionOperation_ORDER_FEE_CORRECTION_OPERATION_VOID
	}
	return v1.OrderFeeCorrectionOperation_ORDER_FEE_CORRECTION_OPERATION_UNSPECIFIED
}
func correctionRequestToAPI(r *biz.OrderFeeCorrectionRequest) *v1.OrderFeeCorrectionRequestData {
	out := &v1.OrderFeeCorrectionRequestData{Id: r.ID.String(), OrderId: r.OrderID.String(), OrganizationId: r.OrganizationID.String(), Operation: correctionOperationToAPI(r.Operation), Status: r.Status, Version: r.Version, Reason: r.Reason, RequestedBy: r.RequestedBy.String(), RequestedByName: r.RequestedByName, RequestedAt: r.RequestedAt.UTC().Format(time.RFC3339), DecidedByName: r.DecidedByName, DecisionReason: r.DecisionReason, CanApprove: r.CanApprove, CanWithdraw: r.CanWithdraw, ApproverAvailable: r.CanApprove}
	if r.LockRecordID != uuid.Nil {
		id := r.LockRecordID.String()
		out.LockRecordId = &id
		generation := r.LockGeneration
		out.LockGeneration = &generation
	}
	if r.SourceFeeID != nil {
		value := r.SourceFeeID.String()
		out.SourceFeeId = &value
	}
	if r.Before != nil {
		out.BeforeFee = orderFeeToAPI(r.Before)
		version := r.Before.Version
		out.SourceFeeVersion = &version
	}
	if r.After != nil {
		out.AfterFee = orderFeeToAPI(r.After)
		if r.Status == "APPROVED" {
			value := r.After.ID.String()
			out.ReplacementFeeId = &value
		}
	}
	if r.DecidedBy != nil {
		value := r.DecidedBy.String()
		out.DecidedBy = &value
	}
	if r.DecidedAt != nil {
		value := r.DecidedAt.UTC().Format(time.RFC3339)
		out.DecidedAt = &value
	}
	if r.ReversalFeeID != nil {
		value := r.ReversalFeeID.String()
		out.ReversalFeeId = &value
	}
	out.FinancialStatus = r.FinancialStatus
	if r.ApprovalResult != nil {
		out.ApprovedCurrentSummary = feeReviewSummaryToAPI(&r.ApprovalResult.Current)
		out.ApprovedProjectedSummary = feeReviewSummaryToAPI(&r.ApprovalResult.Projected)
		if r.ApprovalResult.CommissionImpact != nil {
			for _, line := range r.ApprovalResult.CommissionImpact.Lines {
				out.CommissionImpacts = append(out.CommissionImpacts, correctionCommissionImpactToAPI(line))
			}
		}
	}
	return out
}
func correctionPreviewToAPI(p *biz.OrderFeeCorrectionPreview) *v1.OrderFeeCorrectionApprovalData {
	out := &v1.OrderFeeCorrectionApprovalData{Request: correctionRequestToAPI(p.Request), BaseCurrency: p.After.Summary.BaseCurrency, CurrentReceivable: p.Before.Summary.Receivable.StringFixed(8), CurrentPayable: p.Before.Summary.Payable.StringFixed(8), CurrentProfit: p.Before.Summary.Profit.StringFixed(8), ProjectedReceivable: p.After.Summary.Receivable.StringFixed(8), ProjectedPayable: p.After.Summary.Payable.StringFixed(8), ProjectedProfit: p.After.Summary.Profit.StringFixed(8), ProfitChange: p.After.Summary.Profit.Sub(p.Before.Summary.Profit).StringFixed(8), PreviewToken: p.PreviewToken}
	if p.After.Summary.ProfitRate != nil {
		value := p.After.Summary.ProfitRate.StringFixed(8)
		out.ProjectedProfitRate = &value
	}
	for _, line := range p.CommissionImpact.Lines {
		out.CommissionImpacts = append(out.CommissionImpacts, correctionCommissionImpactToAPI(line))
	}
	return out
}
func parseCorrectionTarget(t *v1.OrderFeeCorrectionTarget) (*biz.OrderFee, error) {
	if t == nil {
		return nil, nil
	}
	direction, valid := orderFeeDirectionFromAPI(t.Direction)
	if !valid {
		return nil, biz.ErrFeeCorrectionInvalid
	}
	setting, err := uuid.Parse(t.FeeSettingId)
	if err != nil {
		return nil, biz.ErrFeeCorrectionInvalid
	}
	party, err := uuid.Parse(t.SettlementPartyId)
	if err != nil {
		return nil, biz.ErrFeeCorrectionInvalid
	}
	unit, err := uuid.Parse(t.BillingUnitId)
	if err != nil {
		return nil, biz.ErrFeeCorrectionInvalid
	}
	quantity, err := parsePlainDecimal(t.Quantity)
	if err != nil {
		return nil, biz.ErrFeeCorrectionInvalid
	}
	price, err := parsePlainDecimal(t.UnitPrice)
	if err != nil {
		return nil, biz.ErrFeeCorrectionInvalid
	}
	fee := &biz.OrderFee{Direction: direction, FeeSettingID: &setting, SettlementPartyID: party, BillingUnitID: &unit, Quantity: quantity, UnitPrice: price, Currency: t.Currency, Note: t.Note, TaxInclusive: true, FeeNameOverride: t.FeeNameOverride}
	if t.TaxInclusive != nil {
		fee.TaxInclusive = *t.TaxInclusive
	}
	if t.ExchangeRateOverride != nil {
		value, err := parsePlainDecimal(*t.ExchangeRateOverride)
		if err != nil {
			return nil, biz.ErrFeeCorrectionInvalid
		}
		fee.ExchangeRateOverride = &value
	}
	if t.TaxRateOverride != nil {
		value, err := parsePlainDecimal(*t.TaxRateOverride)
		if err != nil {
			return nil, biz.ErrFeeCorrectionInvalid
		}
		fee.TaxRateOverride = &value
	}
	return fee, nil
}
func (s *OrderFeeService) CreateOrderFeeCorrection(ctx context.Context, r *v1.CreateOrderFeeCorrectionRequest) (*v1.CreateOrderFeeCorrectionResponse, error) {
	p, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	order, err := uuid.Parse(r.OrderId)
	if err != nil {
		return nil, biz.ErrFeeCorrectionInvalid
	}
	operation := ""
	switch r.Operation {
	case v1.OrderFeeCorrectionOperation_ORDER_FEE_CORRECTION_OPERATION_ADD:
		operation = "ADD"
	case v1.OrderFeeCorrectionOperation_ORDER_FEE_CORRECTION_OPERATION_REPLACE:
		operation = "REPLACE"
	case v1.OrderFeeCorrectionOperation_ORDER_FEE_CORRECTION_OPERATION_VOID:
		operation = "VOID"
	}
	input := &biz.OrderFeeCorrectionInput{Operation: operation, Reason: r.Reason, IdempotencyKey: r.IdempotencyKey}
	if r.SourceFeeId != nil {
		value, err := uuid.Parse(*r.SourceFeeId)
		if err != nil {
			return nil, biz.ErrFeeCorrectionInvalid
		}
		input.SourceFeeID = &value
	}
	if r.SourceFeeExpectedVersion != nil {
		input.ExpectedFeeVersion = *r.SourceFeeExpectedVersion
	}
	input.After, err = parseCorrectionTarget(r.TargetFee)
	if err != nil {
		return nil, err
	}
	data, err := s.correction.Submit(ctx, p, p.Organization.ID, order, input, p.HasPermissionInScope(access.FinanceOrderFeeExchangeRateOverride, biz.DataScopeOrganization))
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateOrderFeeCorrectionResponse{Success: true, Data: correctionRequestToAPI(data)}), nil
}
func (s *OrderFeeService) ListOrderFeeCorrectionRequests(ctx context.Context, r *v1.ListOrderFeeCorrectionRequestsRequest) (*v1.ListOrderFeeCorrectionRequestsResponse, error) {
	p, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	order, err := uuid.Parse(r.OrderId)
	if err != nil {
		return nil, biz.ErrFeeCorrectionInvalid
	}
	page, size := int(r.Page), int(r.PageSize)
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 20
	}
	rows, total, err := s.correction.List(ctx, p, p.Organization.ID, order, page, size)
	if err != nil {
		return nil, err
	}
	out := &v1.ListOrderFeeCorrectionRequestsResponse{Success: true, Total: int32(total), Page: int32(page), PageSize: int32(size), Data: make([]*v1.OrderFeeCorrectionRequestData, 0, len(rows))}
	for _, row := range rows {
		out.Data = append(out.Data, correctionRequestToAPI(row))
	}
	return ok(ctx, out), nil
}
func correctionIDs(orderID, id string) (uuid.UUID, uuid.UUID, error) {
	order, err := uuid.Parse(orderID)
	if err != nil {
		return uuid.Nil, uuid.Nil, biz.ErrFeeCorrectionInvalid
	}
	request, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, uuid.Nil, biz.ErrFeeCorrectionInvalid
	}
	return order, request, nil
}
func (s *OrderFeeService) GetOrderFeeCorrectionRequest(ctx context.Context, r *v1.GetOrderFeeCorrectionRequestRequest) (*v1.GetOrderFeeCorrectionRequestResponse, error) {
	p, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	order, id, err := correctionIDs(r.OrderId, r.Id)
	if err != nil {
		return nil, err
	}
	data, err := s.correction.Get(ctx, p, p.Organization.ID, order, id)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.GetOrderFeeCorrectionRequestResponse{Success: true, Data: correctionRequestToAPI(data)}), nil
}
func (s *OrderFeeService) PreviewOrderFeeCorrectionApproval(ctx context.Context, r *v1.PreviewOrderFeeCorrectionApprovalRequest) (*v1.PreviewOrderFeeCorrectionApprovalResponse, error) {
	p, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	order, id, err := correctionIDs(r.OrderId, r.Id)
	if err != nil {
		return nil, err
	}
	data, err := s.correction.Preview(ctx, p, p.Organization.ID, order, id, r.ExpectedVersion)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.PreviewOrderFeeCorrectionApprovalResponse{Success: true, Data: correctionPreviewToAPI(data)}), nil
}
func (s *OrderFeeService) ApproveOrderFeeCorrection(ctx context.Context, r *v1.ApproveOrderFeeCorrectionRequest) (*v1.ApproveOrderFeeCorrectionResponse, error) {
	p, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	order, id, err := correctionIDs(r.OrderId, r.Id)
	if err != nil {
		return nil, err
	}
	data, err := s.correction.Approve(ctx, p, p.Organization.ID, order, id, r.ExpectedVersion, r.PreviewToken, r.Comment)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ApproveOrderFeeCorrectionResponse{Success: true, Data: correctionRequestToAPI(data)}), nil
}
func (s *OrderFeeService) RejectOrderFeeCorrection(ctx context.Context, r *v1.RejectOrderFeeCorrectionRequest) (*v1.RejectOrderFeeCorrectionResponse, error) {
	p, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	order, id, err := correctionIDs(r.OrderId, r.Id)
	if err != nil {
		return nil, err
	}
	data, err := s.correction.Decide(ctx, p, p.Organization.ID, order, id, r.ExpectedVersion, "REJECTED", r.Reason)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.RejectOrderFeeCorrectionResponse{Success: true, Data: correctionRequestToAPI(data)}), nil
}
func (s *OrderFeeService) WithdrawOrderFeeCorrection(ctx context.Context, r *v1.WithdrawOrderFeeCorrectionRequest) (*v1.WithdrawOrderFeeCorrectionResponse, error) {
	p, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	order, id, err := correctionIDs(r.OrderId, r.Id)
	if err != nil {
		return nil, err
	}
	data, err := s.correction.Decide(ctx, p, p.Organization.ID, order, id, r.ExpectedVersion, "WITHDRAWN", "")
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.WithdrawOrderFeeCorrectionResponse{Success: true, Data: correctionRequestToAPI(data)}), nil
}

func correctionCommissionImpactToAPI(line *biz.FeeCorrectionCommissionLineImpact) *v1.FeeCorrectionCommissionImpactData {
	return &v1.FeeCorrectionCommissionImpactData{CommissionId: line.CommissionID.String(), CommissionNo: line.CommissionNo, OrderId: line.OrderID.String(), EmployeeId: line.EmployeeID.String(), EmployeeName: line.EmployeeName, BaseCurrency: line.BaseCurrency, OriginalAmount: line.OriginalAmount.StringFixed(8), CurrentAmount: line.CurrentAmount.StringFixed(8), TargetAmount: line.TargetAmount.StringFixed(8), Difference: line.Difference.StringFixed(8)}
}
