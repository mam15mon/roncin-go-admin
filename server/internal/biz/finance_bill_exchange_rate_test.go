package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestApplyBillExchangeRateUsesBillDateSnapshot(t *testing.T) {
	organizationID := uuid.New()
	settingID := uuid.New()
	exchangeRepo := &exchangeRateRepoStub{
		rateContext:   &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"},
		timeStandards: []*ExchangeRateTimeStandardSetting{{RateType: BillRateType, TimeStandards: []string{BillDateStandard}}},
		resolved:      &ResolvedExchangeRate{Rate: decimal.RequireFromString("7.20"), Source: "SYSTEM", RateDate: "2026-08-26", SettingID: &settingID},
	}
	usecase := NewFinanceBillUsecase(nil, NewExchangeRateUsecase(exchangeRepo), &financeBillTransactorStub{})
	bill := &FinanceBill{
		Direction:    OrderFeeReceivable,
		Currency:     "USD",
		BaseCurrency: "CNY",
		TotalAmount:  decimal.RequireFromString("100"),
		BillDate:     "2026-08-26",
	}

	if err := usecase.applyBillExchangeRate(context.Background(), organizationID, bill); err != nil {
		t.Fatalf("应用账单日汇率失败: %v", err)
	}
	if bill.ExchangeRate.StringFixed(8) != "7.20000000" || bill.BaseCurrencyAmount.StringFixed(8) != "720.00000000" || bill.ExchangeRateSource != "SYSTEM" || bill.ExchangeRateDate != "2026-08-26" || bill.ExchangeRateSettingID == nil || *bill.ExchangeRateSettingID != settingID {
		t.Fatalf("账单汇率快照不完整: %#v", bill)
	}
}
