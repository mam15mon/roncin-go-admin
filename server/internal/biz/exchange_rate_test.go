package biz

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestNormalizeExchangeRateSettingPreservesEightDecimals(t *testing.T) {
	effectiveTo := "2026-09-01"
	input := &ExchangeRateSetting{
		RateType: SettlementRateType, FromCurrency: "usd", ToCurrency: "cny", TimeStandard: ExpenseDateStandard,
		EffectiveFrom: "2026-08-01", EffectiveTo: &effectiveTo,
		ReceivableRate: decimal.RequireFromString("7.12345678"), PayableRate: decimal.RequireFromString("7.02345678"),
	}
	value, err := normalizeExchangeRateSetting(input)
	if err != nil {
		t.Fatalf("规范化汇率失败: %v", err)
	}
	if value.FromCurrency != "USD" || value.ToCurrency != "CNY" || value.ReceivableRate.StringFixed(8) != "7.12345678" {
		t.Fatalf("汇率规范化结果不正确: %#v", value)
	}
}

func TestNormalizeExchangeRateSettingRejectsInvalidInterval(t *testing.T) {
	effectiveTo := "2026-08-01"
	input := &ExchangeRateSetting{
		RateType: SettlementRateType, FromCurrency: "USD", ToCurrency: "CNY", TimeStandard: ExpenseDateStandard,
		EffectiveFrom: "2026-08-01", EffectiveTo: &effectiveTo,
		ReceivableRate: decimal.NewFromInt(7), PayableRate: decimal.NewFromInt(7),
	}
	if _, err := normalizeExchangeRateSetting(input); err != ErrExchangeRateInvalidArgument {
		t.Fatalf("无效生效区间应被拒绝，实际错误为 %v", err)
	}
}

func TestNormalizeExchangeRateSettingRejectsExcessPrecision(t *testing.T) {
	input := &ExchangeRateSetting{
		RateType: SettlementRateType, FromCurrency: "USD", ToCurrency: "CNY", TimeStandard: ExpenseDateStandard,
		EffectiveFrom: "2026-08-01", ReceivableRate: decimal.RequireFromString("7.123456789"), PayableRate: decimal.NewFromInt(7),
	}
	if _, err := normalizeExchangeRateSetting(input); err != ErrExchangeRateInvalidArgument {
		t.Fatalf("九位小数汇率应被拒绝，实际错误为 %v", err)
	}
}
