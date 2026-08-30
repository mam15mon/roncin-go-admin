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
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
)

func (s *SettlementService) ListCashflows(ctx context.Context, r *v1.ListCashflowsRequest) (*v1.ListCashflowsResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(r.GetPage(), r.GetPageSize(), biz.ErrFinanceCashflowInvalidArgument)
	if err != nil {
		return nil, err
	}
	f := biz.FinanceCashflowFilter{Page: page, PageSize: pageSize, Keyword: financeOptionalString(r.Keyword), Direction: biz.OrderFeeDirection(strings.ToUpper(financeOptionalString(r.Direction))), Status: financeCashflowStatusFromAPI(r.Status), Currency: strings.ToUpper(financeOptionalString(r.Currency))}
	if r.SettlementPartyId != nil && strings.TrimSpace(*r.SettlementPartyId) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*r.SettlementPartyId))
		if err != nil {
			return nil, biz.ErrFinanceCashflowInvalidArgument
		}
		f.SettlementPartyID = &id
	}
	result, e := s.cashflowUsecase.List(ctx, p.Organization.ID, f)
	if e != nil {
		return nil, e
	}
	data := make([]*v1.FinanceCashflow, 0, len(result.Items))
	for _, x := range result.Items {
		data = append(data, cashflowToAPI(x))
	}
	return &v1.ListCashflowsResponse{
		Success: true,
		Message: "OK",
		Data:    data,
		Total:   result.Total,
		TraceId: requestmeta.TraceID(ctx),
		Summary: &v1.FinanceCashflowSummary{
			ReceivableBaseAmount: result.Summary.ReceivableBaseAmount.StringFixed(8),
			PayableBaseAmount:    result.Summary.PayableBaseAmount.StringFixed(8),
			UnverifiedBaseAmount: result.Summary.UnverifiedBaseAmount.StringFixed(8),
			BaseCurrency:         result.Summary.BaseCurrency,
		},
	}, nil
}
func (s *SettlementService) CreateCashflow(ctx context.Context, r *v1.CreateCashflowRequest) (*v1.CreateCashflowResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	party, e := uuid.Parse(strings.TrimSpace(r.GetSettlementPartyId()))
	if e != nil {
		return nil, biz.ErrFinanceCashflowInvalidArgument
	}
	amount, e := decimal.NewFromString(r.GetAmount())
	if e != nil {
		return nil, biz.ErrFinanceCashflowInvalidArgument
	}
	var rateOverride *decimal.Decimal
	if r.ExchangeRate != nil {
		rate, parseErr := decimal.NewFromString(strings.TrimSpace(*r.ExchangeRate))
		if parseErr != nil {
			return nil, biz.ErrFinanceCashflowInvalidArgument
		}
		rateOverride = &rate
	}
	x, e := s.cashflowUsecase.Create(ctx, p.Organization.ID, p.UserID, biz.CreateFinanceCashflowInput{Direction: biz.OrderFeeDirection(strings.ToUpper(r.GetDirection())), SettlementPartyID: party, Currency: r.GetCurrency(), Amount: amount, ExchangeRateOverride: rateOverride, BaseCurrency: r.GetBaseCurrency(), TransactionDate: r.GetTransactionDate(), OurAccount: r.GetOurAccount(), CounterpartyAccount: r.CounterpartyAccount, PaymentMethod: r.GetPaymentMethod(), BankReferenceNo: r.BankReferenceNo, Note: r.Note, IdempotencyKey: r.GetIdempotencyKey()}, p.HasPermission(access.FinanceExchangeRateOverride))
	if e != nil {
		return nil, e
	}
	return &v1.CreateCashflowResponse{Success: true, Message: "OK", Data: cashflowToAPI(x), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) ConfirmCashflow(ctx context.Context, r *v1.ConfirmCashflowRequest) (*v1.ConfirmCashflowResponse, error) {
	p, id, e := financePrincipalAndID(ctx, r.GetId())
	if e != nil {
		return nil, e
	}
	x, e := s.cashflowUsecase.Confirm(ctx, p.Organization.ID, p.UserID, id, r.GetExpectedVersion())
	if e != nil {
		return nil, e
	}
	return &v1.ConfirmCashflowResponse{Success: true, Message: "OK", Data: cashflowToAPI(x), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) CancelCashflow(ctx context.Context, r *v1.CancelCashflowRequest) (*v1.CancelCashflowResponse, error) {
	p, id, e := financePrincipalAndID(ctx, r.GetId())
	if e != nil {
		return nil, e
	}
	x, e := s.cashflowUsecase.Cancel(ctx, p.Organization.ID, p.UserID, id, r.GetExpectedVersion(), r.GetReason())
	if e != nil {
		return nil, e
	}
	return &v1.CancelCashflowResponse{Success: true, Message: "OK", Data: cashflowToAPI(x), TraceId: requestmeta.TraceID(ctx)}, nil
}
func cashflowToAPI(x *biz.FinanceCashflow) *v1.FinanceCashflow {
	if x == nil {
		return nil
	}
	return &v1.FinanceCashflow{Id: x.ID.String(), FlowNo: x.FlowNo, Direction: string(x.Direction), Status: financeCashflowStatusToAPI(x.Status), SettlementPartyId: x.SettlementPartyID.String(), SettlementPartyName: x.SettlementPartyName, Currency: x.Currency, Amount: x.Amount.StringFixed(8), ExchangeRate: x.ExchangeRate.StringFixed(8), ExchangeRateSource: x.ExchangeRateSource, ExchangeRateDate: x.ExchangeRateDate, ExchangeRateSettingId: uuidStringPtr(x.ExchangeRateSettingID), BaseCurrency: x.BaseCurrency, BaseAmount: x.BaseAmount.StringFixed(8), TransactionDate: x.TransactionDate, OurAccount: x.OurAccount, CounterpartyAccount: x.CounterpartyAccount, PaymentMethod: x.PaymentMethod, BankReferenceNo: x.BankReferenceNo, Note: x.Note, Version: x.Version, ConfirmedAt: financeTime(x.ConfirmedAt), CancelledAt: financeTime(x.CancelledAt), CancellationReason: x.CancellationReason, CreatedAt: x.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: x.UpdatedAt.UTC().Format(time.RFC3339), VerifiedAmount: x.VerifiedAmount.StringFixed(8), UnverifiedAmount: x.UnverifiedAmount.StringFixed(8)}
}
