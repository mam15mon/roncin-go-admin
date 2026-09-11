package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func (s *SettlementService) ListVerificationCreationCandidates(ctx context.Context, request *v1.ListVerificationCreationCandidatesRequest) (*v1.ListVerificationCreationCandidatesResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	rawOrganizationID := strings.TrimSpace(request.GetOrganizationId())
	if rawOrganizationID == "" {
		return nil, biz.ErrVerificationInvalid
	}
	organizationIDs, err := organizationIDsForRequestedOrganization(principal, access.FinanceVerificationCreate, true, &rawOrganizationID)
	if err != nil {
		return nil, err
	}
	if len(organizationIDs) != 1 {
		return nil, biz.ErrVerificationInvalid
	}
	settlementPartyID, err := uuid.Parse(strings.TrimSpace(request.GetSettlementPartyId()))
	if err != nil {
		return nil, biz.ErrVerificationInvalid
	}
	candidates, err := s.verificationUsecase.ListCreationCandidates(ctx, organizationIDs[0], biz.VerificationCreationCandidateFilter{
		Direction:         biz.OrderFeeDirection(strings.ToUpper(strings.TrimSpace(request.GetDirection()))),
		SettlementPartyID: settlementPartyID,
		Currency:          request.GetCurrency(),
	})
	if err != nil {
		return nil, err
	}
	cashflows := make([]*v1.FinanceCashflow, 0, len(candidates.Cashflows))
	for _, cashflow := range candidates.Cashflows {
		cashflows = append(cashflows, cashflowToAPI(cashflow))
	}
	bills := make([]*v1.FinanceBill, 0, len(candidates.Bills))
	for _, bill := range candidates.Bills {
		bills = append(bills, financeBillToAPI(bill))
	}
	return okList(ctx, &v1.ListVerificationCreationCandidatesResponse{
		Data: &v1.VerificationCreationCandidates{Cashflows: cashflows, Bills: bills},
	}), nil
}

func (s *SettlementService) ListVerifications(ctx context.Context, r *v1.ListVerificationsRequest) (*v1.ListVerificationsResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(r.GetPage(), r.GetPageSize(), biz.ErrVerificationInvalid)
	if err != nil {
		return nil, err
	}
	f := biz.VerificationFilter{Page: page, PageSize: pageSize, Keyword: financeOptionalString(r.Keyword), Status: financeVerificationStatusFromAPI(r.Status)}
	organizationIDs, scopeErr := organizationIDsForRequestedOrganization(p, access.FinanceVerificationRead, false, r.OrganizationId)
	if scopeErr != nil {
		return nil, scopeErr
	}
	result, e := s.verificationUsecase.ListScoped(ctx, organizationIDs, f)
	if e != nil {
		return nil, e
	}
	data := make([]*v1.FinanceVerification, 0, len(result.Items))
	for _, x := range result.Items {
		data = append(data, verificationToAPI(x))
	}
	return okList(ctx, &v1.ListVerificationsResponse{
		Data: data, Total: result.Total,
		Summary: &v1.FinanceVerificationSummary{AmountsByBaseCurrency: financeBaseCurrencyAmountsToAPI(result.Summary.AmountsByBaseCurrency)},
	}), nil
}
func (s *SettlementService) ListCommissionVerificationCandidates(ctx context.Context, r *v1.ListCommissionVerificationCandidatesRequest) (*v1.ListCommissionVerificationCandidatesResponse, error) {
	p, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(r.GetOrganizationId())
	if raw == "" {
		return nil, biz.ErrVerificationInvalid
	}
	ids, err := organizationIDsForRequestedOrganization(p, access.FinanceCommissionManage, true, &raw)
	if err != nil {
		return nil, err
	}
	if len(ids) != 1 {
		return nil, biz.ErrVerificationInvalid
	}
	page, size, err := listPageValues(r.GetPage(), r.GetPageSize(), biz.ErrVerificationInvalid)
	if err != nil {
		return nil, err
	}
	result, err := s.verificationUsecase.ListScoped(ctx, ids, biz.VerificationFilter{Page: page, PageSize: size, Keyword: financeOptionalString(r.Keyword), Status: biz.VerificationActive, Direction: biz.OrderFeeReceivable})
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceVerification, 0, len(result.Items))
	for _, x := range result.Items {
		data = append(data, verificationToAPI(x))
	}
	return okList(ctx, &v1.ListCommissionVerificationCandidatesResponse{Data: data, Total: result.Total}), nil
}
func (s *SettlementService) CreateVerification(ctx context.Context, r *v1.CreateVerificationRequest) (*v1.CreateVerificationResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	as := make([]*biz.VerificationAllocation, 0, len(r.GetAllocations()))
	for _, x := range r.GetAllocations() {
		c, e := uuid.Parse(x.GetCashflowId())
		if e != nil {
			return nil, biz.ErrVerificationInvalid
		}
		b, e := uuid.Parse(x.GetBillId())
		if e != nil {
			return nil, biz.ErrVerificationInvalid
		}
		z, e := decimal.NewFromString(x.GetAmount())
		if e != nil {
			return nil, biz.ErrVerificationInvalid
		}
		as = append(as, &biz.VerificationAllocation{CashflowID: c, BillID: b, Amount: z})
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceVerificationCreate, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	if len(as) == 0 {
		return nil, biz.ErrVerificationInvalid
	}
	firstCashflow, e := s.verificationUsecase.LoadCashflowContextScoped(ctx, organizationIDs, as[0].CashflowID)
	if e != nil {
		return nil, e
	}
	x, e := s.verificationUsecase.Create(ctx, firstCashflow.OrganizationID, p.UserID, biz.CreateVerificationInput{Allocations: as, VerificationDate: r.GetVerificationDate(), Note: r.Note, IdempotencyKey: r.GetIdempotencyKey()})
	if e != nil {
		return nil, e
	}
	return ok(ctx, &v1.CreateVerificationResponse{Data: verificationToAPI(x)}), nil
}
func (s *SettlementService) ReverseVerification(ctx context.Context, r *v1.ReverseVerificationRequest) (*v1.ReverseVerificationResponse, error) {
	p, id, e := financePrincipalAndID(ctx, r.GetId())
	if e != nil {
		return nil, e
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceVerificationReverse, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	existing, e := s.verificationUsecase.GetScoped(ctx, organizationIDs, id)
	if e != nil {
		return nil, e
	}
	x, e := s.verificationUsecase.Reverse(ctx, existing.OrganizationID, p.UserID, id, r.GetExpectedVersion(), r.GetReason())
	if e != nil {
		return nil, e
	}
	return ok(ctx, &v1.ReverseVerificationResponse{Data: verificationToAPI(x)}), nil
}
func verificationToAPI(x *biz.FinanceVerification) *v1.FinanceVerification {
	if x == nil {
		return nil
	}
	as := make([]*v1.FinanceVerificationAllocation, 0, len(x.Allocations))
	for _, a := range x.Allocations {
		as = append(as, &v1.FinanceVerificationAllocation{Id: a.ID.String(), CashflowId: a.CashflowID.String(), BillId: a.BillID.String(), CashflowNo: a.CashflowNo, BillNo: a.BillNo, Amount: a.Amount.StringFixed(8), BillBaseAmount: a.BillBaseAmount.StringFixed(8), CashflowBaseAmount: a.CashflowBaseAmount.StringFixed(8), WriteOffBaseAmount: a.WriteOffBaseAmount.StringFixed(8), ExchangeGainLoss: a.ExchangeGainLoss.StringFixed(8), Active: a.Active})
	}
	return &v1.FinanceVerification{Id: x.ID.String(), VerificationNo: x.VerificationNo, Status: financeVerificationStatusToAPI(x.Status), Direction: string(x.Direction), OrganizationId: x.OrganizationID.String(), OrganizationName: x.OrganizationName, SettlementPartyId: x.SettlementPartyID.String(), SettlementPartyName: x.SettlementPartyName, Currency: x.Currency, Amount: x.Amount.StringFixed(8), BaseCurrency: x.BaseCurrency, ExchangeRate: x.ExchangeRate.StringFixed(8), ExchangeRateSource: x.ExchangeRateSource, ExchangeRateDate: x.ExchangeRateDate, ExchangeRateSettingId: uuidStringPtr(x.ExchangeRateSettingID), BaseAmount: x.BaseAmount.StringFixed(8), BillBaseAmount: x.BillBaseAmount.StringFixed(8), CashflowBaseAmount: x.CashflowBaseAmount.StringFixed(8), ExchangeGainLoss: x.ExchangeGainLoss.StringFixed(8), VerificationDate: x.VerificationDate, Note: x.Note, Version: x.Version, ReversedAt: financeTime(x.ReversedAt), ReversalReason: x.ReversalReason, Allocations: as, CreatedAt: x.CreatedAt.UTC().Format(time.RFC3339)}
}
