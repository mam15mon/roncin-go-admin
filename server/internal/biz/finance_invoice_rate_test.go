package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type invoiceIssueRateRepoStub struct {
	FinanceInvoiceRepo
	item  *FinanceInvoice
	saved *FinanceInvoiceIssueInput
}

func (s *invoiceIssueRateRepoStub) Get(context.Context, uuid.UUID, uuid.UUID) (*FinanceInvoice, error) {
	return s.item, nil
}
func (s *invoiceIssueRateRepoStub) Issue(_ context.Context, _, _, _ uuid.UUID, _ uint64, in FinanceInvoiceIssueInput, _ *AuditEvent) (*FinanceInvoice, error) {
	s.saved = &in
	return s.item, nil
}

type invoiceDirectionRateRepoStub struct {
	exchangeRateRepoStub
	direction OrderFeeDirection
}

func (s *invoiceDirectionRateRepoStub) ResolveRate(_ context.Context, _ uuid.UUID, direction OrderFeeDirection, _, _, _ string) (ResolvedRate, error) {
	s.direction = direction
	if s.resolveErr != nil {
		return ResolvedRate{}, s.resolveErr
	}
	rate := decimal.RequireFromString("7.2")
	if direction == OrderFeePayable {
		rate = decimal.NewFromInt(7)
	}
	return ResolvedRate{Rate: rate, Source: ExchangeRateSourceWeekly}, nil
}

func TestInvoiceIssueResolvesRateByBillDirection(t *testing.T) {
	for _, direction := range []OrderFeeDirection{OrderFeeReceivable, OrderFeePayable} {
		for _, missing := range []bool{false, true} {
			name := string(direction)
			if missing {
				name += "/缺少汇率"
			}
			t.Run(name, func(t *testing.T) {
				org := uuid.New()
				repo := &invoiceIssueRateRepoStub{item: &FinanceInvoice{Direction: direction, Currency: "USD", BaseCurrency: "CNY", TotalAmount: decimal.NewFromInt(100)}}
				rates := &invoiceDirectionRateRepoStub{exchangeRateRepoStub: exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: org, BaseCurrency: "CNY"}}}
				if missing {
					rates.resolveErr = ErrExchangeRateMissing
				}
				uc := NewFinanceInvoiceUsecase(repo, nil, NewExchangeRateUsecase(rates, nil))
				_, err := uc.Issue(context.Background(), org, uuid.New(), uuid.New(), 1, "INV-001", "2026-11-04")
				if missing {
					if err != ErrExchangeRateMissing || repo.saved != nil {
						t.Fatalf("缺失方向汇率必须停止确认：err=%v saved=%+v", err, repo.saved)
					}
					return
				}
				amount := decimal.NewFromInt(720)
				if direction == OrderFeePayable {
					amount = decimal.NewFromInt(700)
				}
				if err != nil || rates.direction != direction || repo.saved == nil || !repo.saved.BaseCurrencyAmount.Equal(amount) || !repo.saved.ExchangeRate.Mul(decimal.NewFromInt(100)).Equal(amount) {
					t.Fatalf("确认发票应按方向固化正确本币：direction=%s err=%v saved=%+v", rates.direction, err, repo.saved)
				}
			})
		}
	}
}
