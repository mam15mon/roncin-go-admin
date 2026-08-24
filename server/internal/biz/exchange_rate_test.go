package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type exchangeRateRepoStub struct {
	rateContext *ExchangeRateContext
}

func (s *exchangeRateRepoStub) ResolveContext(context.Context, uuid.UUID) (*ExchangeRateContext, error) {
	return s.rateContext, nil
}

func (*exchangeRateRepoStub) List(context.Context, uuid.UUID) ([]*ExchangeRateSetting, error) {
	return nil, nil
}

func (*exchangeRateRepoStub) Create(context.Context, *ExchangeRateSetting, *AuditEvent) (*ExchangeRateSetting, error) {
	return nil, nil
}

func (*exchangeRateRepoStub) Update(context.Context, *ExchangeRateSetting, *AuditEvent) (*ExchangeRateSetting, error) {
	return nil, nil
}

func (*exchangeRateRepoStub) Disable(context.Context, uuid.UUID, uuid.UUID, *AuditEvent) error {
	return nil
}

func (*exchangeRateRepoStub) Resolve(context.Context, uuid.UUID, OrderFeeDirection, string, string, string) (*ResolvedExchangeRate, error) {
	return nil, nil
}

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

func TestResolveBaseCurrencyUsesExactOne(t *testing.T) {
	usecase := NewExchangeRateUsecase(&exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "USD"}})
	resolved, err := usecase.Resolve(context.Background(), uuid.Must(uuid.NewV7()), OrderFeeReceivable, "USD", "2026-08-24")
	if err != nil {
		t.Fatalf("解析本币汇率失败: %v", err)
	}
	if resolved.Rate.StringFixed(8) != "1.00000000" || resolved.Source != "BASE_CURRENCY" || resolved.SettingID != nil {
		t.Fatalf("本币汇率解析结果不正确: %#v", resolved)
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
