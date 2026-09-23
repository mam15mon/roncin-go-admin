package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestExchangeRateNonCNYCompanyRequiresDirectWeeklyRate(t *testing.T) {
	companyID := uuid.New()
	repo := &exchangeRateRepoStub{
		rateContext: &ExchangeRateContext{OwnerOrganizationID: companyID, BaseCurrency: "CAD"},
		rateByPair: map[string]decimal.Decimal{
			"USD→CNY": decimal.RequireFromString("7.20"),
			"CAD→CNY": decimal.RequireFromString("5.20"),
		},
	}
	usecase := NewExchangeRateUsecase(repo, nil)
	_, err := usecase.ResolveRate(context.Background(), companyID, OrderFeeReceivable, "USD", "2026-09-12")
	if err != ErrExchangeRateMissing {
		t.Fatalf("仅有公共交叉腿时必须报缺失，实际 %v", err)
	}
	if len(repo.resolvePairs) != 1 || repo.resolvePairs[0] != "USD→CAD" {
		t.Fatalf("只应查询公司本币直连行，实际 %v", repo.resolvePairs)
	}

	repo.rateByPair["USD→CAD"] = decimal.RequireFromString("1.50")
	resolved, err := usecase.ResolveRate(context.Background(), companyID, OrderFeeReceivable, "USD", "2026-09-12")
	if err != nil || !resolved.Rate.Equal(decimal.RequireFromString("1.50")) || resolved.Source != ExchangeRateSourceWeekly {
		t.Fatalf("公司直连汇率应生效，实际 %+v err=%v", resolved, err)
	}
}

func TestExchangeRateNonCNYCompanySnapshotUsesDirectRate(t *testing.T) {
	companyID := uuid.New()
	repo := &exchangeRateRepoStub{
		rateContext: &ExchangeRateContext{OwnerOrganizationID: companyID, BaseCurrency: "CAD"},
		rateByPair:  map[string]decimal.Decimal{"USD→CAD": decimal.RequireFromString("1.50")},
	}
	rates := NewExchangeRateUsecase(repo, nil)
	feeUsecase := NewOrderFeeUsecase(nil, rates, nil, newReminderModeCreditControl(), nil, nil)
	fee := validOrderFeeForTest()
	fee.Currency = "USD"
	if err := feeUsecase.resolveExchangeRate(context.Background(), companyID, uuid.New(), fee, false); err != nil {
		t.Fatalf("费用汇率解析失败: %v", err)
	}
	if !fee.ExchangeRate.Equal(decimal.RequireFromString("1.50")) || fee.ExchangeRateSource != ExchangeRateSourceWeekly {
		t.Fatalf("费用快照应记录公司直连汇率，实际 %s/%s", fee.ExchangeRate, fee.ExchangeRateSource)
	}

	billUsecase := NewFinanceBillUsecase(nil, rates, nil, nil, nil)
	bill := &FinanceBill{Direction: OrderFeeReceivable, Currency: "USD", BaseCurrency: "CAD", TotalAmount: decimal.RequireFromString("100"), BillDate: "2026-09-12"}
	if err := billUsecase.applyBillExchangeRate(context.Background(), companyID, bill); err != nil {
		t.Fatalf("账单汇率解析失败: %v", err)
	}
	if !bill.ExchangeRate.Equal(decimal.RequireFromString("1.50")) || bill.ExchangeRateSource != ExchangeRateSourceWeekly || !bill.BaseCurrencyAmount.Equal(decimal.RequireFromString("150")) {
		t.Fatalf("账单快照应使用公司直连汇率，实际 %+v", bill)
	}
}
