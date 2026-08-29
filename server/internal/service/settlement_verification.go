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

func (s *SettlementService) ListVerifications(ctx context.Context, r *v1.ListVerificationsRequest) (*v1.ListVerificationsResponse, error) {
	p, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	f := biz.VerificationFilter{Page: int(r.GetPage()), PageSize: int(r.GetPageSize()), Keyword: financeOptionalString(r.Keyword), Status: biz.VerificationStatus(strings.ToUpper(financeOptionalString(r.Status)))}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	result, e := s.verificationUsecase.List(ctx, p.Organization.ID, f)
	if e != nil {
		return nil, e
	}
	data := make([]*v1.FinanceVerification, 0, len(result.Items))
	for _, x := range result.Items {
		data = append(data, verificationToAPI(x))
	}
	return &v1.ListVerificationsResponse{Success: true, Message: "OK", Data: data, Total: result.Total, TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) CreateVerification(ctx context.Context, r *v1.CreateVerificationRequest) (*v1.CreateVerificationResponse, error) {
	p, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
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
	x, e := s.verificationUsecase.Create(ctx, p.Organization.ID, p.UserID, biz.CreateVerificationInput{Allocations: as, VerificationDate: r.GetVerificationDate(), Note: r.Note, IdempotencyKey: r.GetIdempotencyKey()})
	if e != nil {
		return nil, e
	}
	return &v1.CreateVerificationResponse{Success: true, Message: "OK", Data: verificationToAPI(x), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) ReverseVerification(ctx context.Context, r *v1.ReverseVerificationRequest) (*v1.ReverseVerificationResponse, error) {
	p, id, e := financePrincipalAndID(ctx, r.GetId())
	if e != nil {
		return nil, e
	}
	x, e := s.verificationUsecase.Reverse(ctx, p.Organization.ID, p.UserID, id, r.GetExpectedVersion(), r.GetReason())
	if e != nil {
		return nil, e
	}
	return &v1.ReverseVerificationResponse{Success: true, Message: "OK", Data: verificationToAPI(x), TraceId: requestmeta.TraceID(ctx)}, nil
}
func verificationToAPI(x *biz.FinanceVerification) *v1.FinanceVerification {
	if x == nil {
		return nil
	}
	as := make([]*v1.FinanceVerificationAllocation, 0, len(x.Allocations))
	for _, a := range x.Allocations {
		as = append(as, &v1.FinanceVerificationAllocation{Id: a.ID.String(), CashflowId: a.CashflowID.String(), BillId: a.BillID.String(), CashflowNo: a.CashflowNo, BillNo: a.BillNo, Amount: a.Amount.StringFixed(8), BillBaseAmount: a.BillBaseAmount.StringFixed(8), CashflowBaseAmount: a.CashflowBaseAmount.StringFixed(8), WriteOffBaseAmount: a.WriteOffBaseAmount.StringFixed(8), ExchangeGainLoss: a.ExchangeGainLoss.StringFixed(8), Active: a.Active})
	}
	return &v1.FinanceVerification{Id: x.ID.String(), VerificationNo: x.VerificationNo, Status: string(x.Status), Direction: string(x.Direction), SettlementPartyId: x.SettlementPartyID.String(), SettlementPartyName: x.SettlementPartyName, Currency: x.Currency, Amount: x.Amount.StringFixed(8), BaseCurrency: x.BaseCurrency, ExchangeRate: x.ExchangeRate.StringFixed(8), ExchangeRateSource: x.ExchangeRateSource, ExchangeRateDate: x.ExchangeRateDate, ExchangeRateSettingId: financeUUID(x.ExchangeRateSettingID), BaseAmount: x.BaseAmount.StringFixed(8), BillBaseAmount: x.BillBaseAmount.StringFixed(8), CashflowBaseAmount: x.CashflowBaseAmount.StringFixed(8), ExchangeGainLoss: x.ExchangeGainLoss.StringFixed(8), VerificationDate: x.VerificationDate, Note: x.Note, Version: x.Version, ReversedAt: financeTime(x.ReversedAt), ReversalReason: x.ReversalReason, Allocations: as, CreatedAt: x.CreatedAt.UTC().Format(time.RFC3339)}
}
