package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
)

func (s *SettlementService) ListCommissions(ctx context.Context, r *v1.ListCommissionsRequest) (*v1.ListCommissionsResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	f := biz.CommissionFilter{Page: int(r.GetPage()), PageSize: int(r.GetPageSize()), Keyword: financeOptionalString(r.Keyword), Status: biz.CommissionStatus(strings.ToUpper(financeOptionalString(r.Status)))}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	result, err := s.commissionUsecase.List(ctx, p.Organization.ID, f)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceCommission, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, commissionToAPI(item))
	}
	return &v1.ListCommissionsResponse{Success: true, Message: "OK", Data: data, Total: result.Total, TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) GetCommission(ctx context.Context, r *v1.GetCommissionRequest) (*v1.GetCommissionResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.Get(ctx, p.Organization.ID, id)
	if err != nil {
		return nil, err
	}
	return &v1.GetCommissionResponse{Success: true, Message: "OK", Data: commissionToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) ListCommissionEmployees(ctx context.Context, request *v1.ListCommissionEmployeesRequest) (*v1.ListCommissionEmployeesResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize := biz.ListPagination(int(request.GetPage()), int(request.GetPageSize()), 20)
	result, err := s.commissionUsecase.ListEmployees(ctx, p.Organization.ID, biz.SelectorListOptions{
		Page: page, PageSize: pageSize, Keyword: financeOptionalString(request.Keyword),
	})
	if err != nil {
		return nil, err
	}
	data := make([]*v1.CommissionEmployeeOption, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, &v1.CommissionEmployeeOption{Id: item.ID.String(), DisplayName: item.DisplayName})
	}
	return &v1.ListCommissionEmployeesResponse{
		Success: true, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx),
		Total: int64(result.Total), Page: int32(result.Page), PageSize: int32(result.PageSize),
	}, nil
}
func (s *SettlementService) ListCommissionCandidates(ctx context.Context, r *v1.ListCommissionCandidatesRequest) (*v1.ListCommissionCandidatesResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	ruleID, err := uuid.Parse(strings.TrimSpace(r.GetRuleId()))
	if err != nil {
		return nil, biz.ErrCommissionInvalid
	}
	verificationID, err := uuid.Parse(strings.TrimSpace(r.GetVerificationId()))
	if err != nil {
		return nil, biz.ErrCommissionInvalid
	}
	f := biz.CommissionCandidateFilter{Page: int(r.GetPage()), PageSize: int(r.GetPageSize()), Keyword: financeOptionalString(r.Keyword), VerificationID: verificationID, RuleID: ruleID}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	result, err := s.commissionUsecase.ListCandidates(ctx, p.Organization.ID, f)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.CommissionCandidateSummary, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, commissionCandidateSummaryToAPI(item))
	}
	return &v1.ListCommissionCandidatesResponse{
		Success: true, Message: "OK", Data: data, Total: result.Total, TraceId: requestmeta.TraceID(ctx),
		Page: int32(result.Page), PageSize: int32(result.PageSize),
	}, nil
}
func (s *SettlementService) ListCommissionRules(ctx context.Context, r *v1.ListCommissionRulesRequest) (*v1.ListCommissionRulesResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	f := biz.CommissionRuleFilter{Page: int(r.GetPage()), PageSize: int(r.GetPageSize()), Keyword: financeOptionalString(r.Keyword), PersonnelRole: biz.CommissionPersonnelRole(strings.ToUpper(financeOptionalString(r.PersonnelRole))), Enabled: r.Enabled}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	result, err := s.commissionUsecase.ListRules(ctx, p.Organization.ID, f)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceCommissionRule, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, commissionRuleToAPI(item))
	}
	return &v1.ListCommissionRulesResponse{Success: true, Message: "OK", Data: data, Total: result.Total, TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) CreateCommissionRule(ctx context.Context, r *v1.CreateCommissionRuleRequest) (*v1.CreateCommissionRuleResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	in, err := commissionRuleInputFromAPI(r.GetRule())
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.CreateRule(ctx, p.Organization.ID, p.UserID, in)
	if err != nil {
		return nil, err
	}
	return &v1.CreateCommissionRuleResponse{Success: true, Message: "OK", Data: commissionRuleToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) UpdateCommissionRule(ctx context.Context, r *v1.UpdateCommissionRuleRequest) (*v1.UpdateCommissionRuleResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	in, err := commissionRuleInputFromAPI(r.GetRule())
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.UpdateRule(ctx, p.Organization.ID, p.UserID, biz.UpdateCommissionRuleInput{ID: id, CreateCommissionRuleInput: in, ExpectedVersion: r.GetExpectedVersion()})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateCommissionRuleResponse{Success: true, Message: "OK", Data: commissionRuleToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func commissionRuleInputFromAPI(r *v1.CommissionRuleInput) (biz.CreateCommissionRuleInput, error) {
	if r == nil {
		return biz.CreateCommissionRuleInput{}, biz.ErrCommissionRuleInvalid
	}
	rate, err := decimal.NewFromString(r.GetRatePercent())
	if err != nil {
		return biz.CreateCommissionRuleInput{}, biz.ErrCommissionRuleInvalid
	}
	return biz.CreateCommissionRuleInput{Name: r.GetName(), PersonnelRole: biz.CommissionPersonnelRole(strings.ToUpper(r.GetPersonnelRole())), CalculationBasis: biz.CommissionCalculationBasis(strings.ToUpper(r.GetCalculationBasis())), RatePercent: rate, EffectiveFrom: r.EffectiveFrom, EffectiveTo: r.EffectiveTo, Enabled: r.GetEnabled(), Note: r.Note}, nil
}
func commissionRuleToAPI(x *biz.FinanceCommissionRule) *v1.FinanceCommissionRule {
	if x == nil {
		return nil
	}
	return &v1.FinanceCommissionRule{Id: x.ID.String(), Name: x.Name, PersonnelRole: string(x.PersonnelRole), CalculationBasis: string(x.CalculationBasis), RatePercent: x.RatePercent.StringFixed(4), EffectiveFrom: x.EffectiveFrom, EffectiveTo: x.EffectiveTo, Enabled: x.Enabled, Note: x.Note, Version: x.Version, CreatedAt: x.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: x.UpdatedAt.UTC().Format(time.RFC3339)}
}
func (s *SettlementService) PreviewCommission(ctx context.Context, r *v1.PreviewCommissionRequest) (*v1.PreviewCommissionResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	verificationID, err := uuid.Parse(strings.TrimSpace(r.GetVerificationId()))
	if err != nil {
		return nil, biz.ErrCommissionInvalid
	}
	employeeID, err := uuid.Parse(strings.TrimSpace(r.GetEmployeeId()))
	if err != nil {
		return nil, biz.ErrCommissionInvalid
	}
	ruleID, err := uuid.Parse(strings.TrimSpace(r.GetRuleId()))
	if err != nil {
		return nil, biz.ErrCommissionInvalid
	}
	item, err := s.commissionUsecase.Preview(ctx, p.Organization.ID, verificationID, employeeID, ruleID)
	if err != nil {
		return nil, err
	}
	return &v1.PreviewCommissionResponse{Success: true, Message: "OK", Data: commissionCalculationToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) CreateCommission(ctx context.Context, r *v1.CreateCommissionRequest) (*v1.CreateCommissionResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	verificationID, err := uuid.Parse(strings.TrimSpace(r.GetVerificationId()))
	if err != nil {
		return nil, biz.ErrCommissionInvalid
	}
	employeeID, err := uuid.Parse(strings.TrimSpace(r.GetEmployeeId()))
	if err != nil {
		return nil, biz.ErrCommissionInvalid
	}
	ruleID, err := uuid.Parse(strings.TrimSpace(r.GetRuleId()))
	if err != nil {
		return nil, biz.ErrCommissionInvalid
	}
	item, err := s.commissionUsecase.Create(ctx, p.Organization.ID, p.UserID, biz.CreateCommissionInput{VerificationID: verificationID, EmployeeID: employeeID, RuleID: ruleID, Note: r.Note, IdempotencyKey: r.GetIdempotencyKey()})
	if err != nil {
		return nil, err
	}
	return &v1.CreateCommissionResponse{Success: true, Message: "OK", Data: commissionToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) ConfirmCommission(ctx context.Context, r *v1.ConfirmCommissionRequest) (*v1.ConfirmCommissionResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.Confirm(ctx, p.Organization.ID, p.UserID, id, r.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return &v1.ConfirmCommissionResponse{Success: true, Message: "OK", Data: commissionToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) MarkCommissionPaid(ctx context.Context, r *v1.MarkCommissionPaidRequest) (*v1.MarkCommissionPaidResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.MarkPaid(ctx, p.Organization.ID, p.UserID, id, r.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return &v1.MarkCommissionPaidResponse{Success: true, Message: "OK", Data: commissionToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) CancelCommission(ctx context.Context, r *v1.CancelCommissionRequest) (*v1.CancelCommissionResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.Cancel(ctx, p.Organization.ID, p.UserID, id, r.GetExpectedVersion(), r.GetReason())
	if err != nil {
		return nil, err
	}
	return &v1.CancelCommissionResponse{Success: true, Message: "OK", Data: commissionToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) CreateCommissionAdjustment(ctx context.Context, r *v1.CreateCommissionAdjustmentRequest) (*v1.CreateCommissionAdjustmentResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	commissionID, err := uuid.Parse(strings.TrimSpace(r.GetCommissionId()))
	if err != nil {
		return nil, biz.ErrCommissionAdjustmentInvalid
	}
	orderID, err := uuid.Parse(strings.TrimSpace(r.GetOrderId()))
	if err != nil {
		return nil, biz.ErrCommissionAdjustmentInvalid
	}
	amount, err := decimal.NewFromString(strings.TrimSpace(r.GetAmount()))
	if err != nil {
		return nil, biz.ErrCommissionAdjustmentInvalid
	}
	item, err := s.commissionUsecase.CreateAdjustment(ctx, p.Organization.ID, p.UserID, biz.CreateCommissionAdjustmentInput{
		CommissionID: commissionID, OrderID: orderID, Direction: biz.CommissionAdjustmentDirection(strings.ToUpper(r.GetDirection())),
		Amount: amount, Reason: r.GetReason(), Note: r.Note, IdempotencyKey: r.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateCommissionAdjustmentResponse{Success: true, Message: "OK", Data: commissionAdjustmentToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) ConfirmCommissionAdjustment(ctx context.Context, r *v1.ConfirmCommissionAdjustmentRequest) (*v1.ConfirmCommissionAdjustmentResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.ConfirmAdjustment(ctx, p.Organization.ID, p.UserID, id, r.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return &v1.ConfirmCommissionAdjustmentResponse{Success: true, Message: "OK", Data: commissionAdjustmentToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) MarkCommissionAdjustmentPaid(ctx context.Context, r *v1.MarkCommissionAdjustmentPaidRequest) (*v1.MarkCommissionAdjustmentPaidResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.MarkAdjustmentPaid(ctx, p.Organization.ID, p.UserID, id, r.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return &v1.MarkCommissionAdjustmentPaidResponse{Success: true, Message: "OK", Data: commissionAdjustmentToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) CancelCommissionAdjustment(ctx context.Context, r *v1.CancelCommissionAdjustmentRequest) (*v1.CancelCommissionAdjustmentResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.CancelAdjustment(ctx, p.Organization.ID, p.UserID, id, r.GetExpectedVersion(), r.GetReason())
	if err != nil {
		return nil, err
	}
	return &v1.CancelCommissionAdjustmentResponse{Success: true, Message: "OK", Data: commissionAdjustmentToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}

func commissionToAPI(x *biz.FinanceCommission) *v1.FinanceCommission {
	if x == nil {
		return nil
	}
	var ruleID, ruleName, personnelRole, calculationBasis *string
	if x.RuleID != uuid.Nil {
		value := x.RuleID.String()
		ruleID = &value
	}
	if x.RuleName != "" {
		value := x.RuleName
		ruleName = &value
	}
	if x.PersonnelRole != "" {
		value := string(x.PersonnelRole)
		personnelRole = &value
	}
	if x.CalculationBasis != "" {
		value := string(x.CalculationBasis)
		calculationBasis = &value
	}
	lines := make([]*v1.FinanceCommissionLine, 0, len(x.Lines))
	for _, line := range x.Lines {
		lines = append(lines, commissionLineToAPI(line))
	}
	adjustments := make([]*v1.FinanceCommissionAdjustment, 0, len(x.Adjustments))
	for _, item := range x.Adjustments {
		adjustments = append(adjustments, commissionAdjustmentToAPI(item))
	}
	return &v1.FinanceCommission{Id: x.ID.String(), CommissionNo: x.CommissionNo, VerificationId: x.VerificationID.String(), VerificationNo: x.VerificationNo, EmployeeId: x.EmployeeID.String(), EmployeeName: x.EmployeeName, Status: string(x.Status), BaseCurrency: x.BaseCurrency, CustomerCount: int32(x.CustomerCount), OrderCount: int32(x.OrderCount), FeeCount: int32(x.FeeCount), RealizedRevenue: x.RealizedRevenue.StringFixed(8), AllocatedCost: x.AllocatedCost.StringFixed(8), RealizedProfit: x.RealizedProfit.StringFixed(8), CommissionBaseAmount: x.CommissionBaseAmount.StringFixed(8), RatePercent: x.RatePercent.StringFixed(4), CommissionAmount: x.CommissionAmount.StringFixed(8), Note: x.Note, Version: x.Version, ConfirmedAt: financeTime(x.ConfirmedAt), PaidAt: financeTime(x.PaidAt), CancelledAt: financeTime(x.CancelledAt), CancellationReason: x.CancellationReason, CreatedAt: x.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: x.UpdatedAt.UTC().Format(time.RFC3339), RuleId: ruleID, RuleName: ruleName, PersonnelRole: personnelRole, CalculationBasis: calculationBasis, RuleVersion: x.RuleVersion, CalculationVersion: x.CalculationVersion, Lines: lines, Adjustments: adjustments, AdjustmentAmount: x.AdjustmentAmount.StringFixed(8), EffectiveCommissionAmount: x.EffectiveCommissionAmount.StringFixed(8)}
}

func commissionAdjustmentToAPI(x *biz.FinanceCommissionAdjustment) *v1.FinanceCommissionAdjustment {
	if x == nil {
		return nil
	}
	return &v1.FinanceCommissionAdjustment{
		Id: x.ID.String(), AdjustmentNo: x.AdjustmentNo, CommissionId: x.CommissionID.String(), CommissionNo: x.CommissionNo,
		OrderId: x.OrderID.String(), OrderNo: x.OrderNo, EmployeeId: x.EmployeeID.String(), EmployeeName: x.EmployeeName,
		Direction: string(x.Direction), Status: string(x.Status), BaseCurrency: x.BaseCurrency, Amount: x.Amount.StringFixed(8),
		Reason: x.Reason, Note: x.Note, Version: x.Version, ConfirmedAt: financeTime(x.ConfirmedAt), PaidAt: financeTime(x.PaidAt),
		CancelledAt: financeTime(x.CancelledAt), CancellationReason: x.CancellationReason,
		SourceType: string(x.SourceType), SourceVerificationId: financeUUID(x.SourceVerificationID),
		CreatedAt: x.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: x.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func commissionCalculationToAPI(x *biz.CommissionCalculation) *v1.CommissionCalculation {
	if x == nil {
		return nil
	}
	lines := make([]*v1.FinanceCommissionLine, 0, len(x.Lines))
	for _, line := range x.Lines {
		lines = append(lines, commissionLineToAPI(line))
	}
	return &v1.CommissionCalculation{VerificationId: x.VerificationID.String(), VerificationNo: x.VerificationNo, EmployeeId: x.EmployeeID.String(), EmployeeName: x.EmployeeName, RuleId: x.RuleID.String(), RuleName: x.RuleName, PersonnelRole: string(x.PersonnelRole), CalculationBasis: string(x.CalculationBasis), RuleVersion: x.RuleVersion, CalculationVersion: x.CalculationVersion, BaseCurrency: x.BaseCurrency, CustomerCount: int32(x.CustomerCount), OrderCount: int32(x.OrderCount), FeeCount: int32(x.FeeCount), RealizedRevenue: x.RealizedRevenue.StringFixed(8), AllocatedCost: x.AllocatedCost.StringFixed(8), RealizedProfit: x.RealizedProfit.StringFixed(8), CommissionBaseAmount: x.CommissionBaseAmount.StringFixed(8), RatePercent: x.RatePercent.StringFixed(4), CommissionAmount: x.CommissionAmount.StringFixed(8), Lines: lines}
}

func commissionLineToAPI(x *biz.FinanceCommissionLine) *v1.FinanceCommissionLine {
	if x == nil {
		return nil
	}
	fees := make([]*v1.CommissionFeeDetail, 0, len(x.Fees))
	for _, item := range x.Fees {
		fees = append(fees, &v1.CommissionFeeDetail{FeeId: item.FeeID.String(), Direction: item.Direction, FeeCode: item.FeeCode, FeeName: item.FeeName, SettlementPartyId: item.SettlementPartyID.String(), SettlementPartyName: item.SettlementPartyName, Currency: item.Currency, TotalAmount: item.TotalAmount.StringFixed(8), ExchangeRate: item.ExchangeRate.StringFixed(8), BaseCurrency: item.BaseCurrency, BaseCurrencyAmount: item.BaseCurrencyAmount.StringFixed(8), ExpenseDate: item.ExpenseDate, Status: item.Status})
	}
	return &v1.FinanceCommissionLine{Id: x.ID.String(), OrderId: x.OrderID.String(), OrderNo: x.OrderNo, OrderDate: x.OrderDate, CustomerId: x.CustomerID.String(), CustomerCode: x.CustomerCode, CustomerName: x.CustomerName, EmployeeId: x.EmployeeID.String(), EmployeeName: x.EmployeeName, PersonnelRole: string(x.PersonnelRole), CalculationBasis: string(x.CalculationBasis), BaseCurrency: x.BaseCurrency, RealizedRevenue: x.RealizedRevenue.StringFixed(8), AllocatedCost: x.AllocatedCost.StringFixed(8), RealizedProfit: x.RealizedProfit.StringFixed(8), CommissionBaseAmount: x.CommissionBaseAmount.StringFixed(8), RatePercent: x.RatePercent.StringFixed(4), CommissionAmount: x.CommissionAmount.StringFixed(8), CustomerAssignmentId: x.CustomerAssignmentID.String(), CustomerAssignmentOrganizationId: x.CustomerAssignmentOrganizationID.String(), CustomerAssignedAt: x.CustomerAssignedAt.UTC().Format(time.RFC3339), FeeCount: int32(x.FeeCount), Fees: fees, PersonnelOrganizationId: x.CustomerAssignmentOrganizationID.String(), PersonnelAssignedAt: x.CustomerAssignedAt.UTC().Format(time.RFC3339)}
}

func commissionCandidateSummaryToAPI(x *biz.CommissionCalculation) *v1.CommissionCandidateSummary {
	return &v1.CommissionCandidateSummary{EmployeeId: x.EmployeeID.String(), EmployeeName: x.EmployeeName, PersonnelRole: string(x.PersonnelRole), CustomerCount: int32(x.CustomerCount), OrderCount: int32(x.OrderCount), FeeCount: int32(x.FeeCount), BaseCurrency: x.BaseCurrency, RealizedRevenue: x.RealizedRevenue.StringFixed(8), AllocatedCost: x.AllocatedCost.StringFixed(8), RealizedProfit: x.RealizedProfit.StringFixed(8), CommissionBaseAmount: x.CommissionBaseAmount.StringFixed(8), RatePercent: x.RatePercent.StringFixed(4), CommissionAmount: x.CommissionAmount.StringFixed(8), Id: x.EmployeeID.String(), DisplayName: x.EmployeeName}
}
