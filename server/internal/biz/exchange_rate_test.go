package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type exchangeRateRepoStub struct {
	rateContext   *ExchangeRateContext
	timeStandards []*ExchangeRateTimeStandardSetting
	resolved      *ResolvedExchangeRate
	resolveErr    error
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

func (s *exchangeRateRepoStub) ListTimeStandards(context.Context, uuid.UUID) ([]*ExchangeRateTimeStandardSetting, error) {
	return s.timeStandards, nil
}

func (*exchangeRateRepoStub) ReplaceTimeStandards(context.Context, uuid.UUID, []*ExchangeRateTimeStandardSetting, *AuditEvent) error {
	return nil
}

func (s *exchangeRateRepoStub) Resolve(context.Context, uuid.UUID, string, OrderFeeDirection, string, string, string) (*ResolvedExchangeRate, error) {
	return s.resolved, s.resolveErr
}

func TestNormalizeExchangeRateSettingPreservesEightDecimals(t *testing.T) {
	effectiveTo := "2026-09-01"
	input := &ExchangeRateSetting{
		RateType: SettlementRateType, FromCurrency: "usd", ToCurrency: "cny",
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

func TestNormalizeExchangeRateSettingAcceptsSupportedRateTypes(t *testing.T) {
	types := []string{BaseCurrencyRateType, InvoiceRateType, SettlementRateType, WriteOffRateType, BillRateType}
	for _, rateType := range types {
		t.Run(rateType, func(t *testing.T) {
			input := &ExchangeRateSetting{
				RateType: rateType, FromCurrency: "USD", ToCurrency: "CNY",
				EffectiveFrom: "2026-08-01", ReceivableRate: decimal.NewFromInt(7), PayableRate: decimal.NewFromInt(7),
			}
			if _, err := normalizeExchangeRateSetting(input); err != nil {
				t.Fatalf("合法汇率类型 %s 不应被拒绝: %v", rateType, err)
			}
		})
	}
}

func TestNormalizeExchangeRateSettingRejectsUnknownRateType(t *testing.T) {
	input := &ExchangeRateSetting{
		RateType: "UNKNOWN", FromCurrency: "USD", ToCurrency: "CNY",
		EffectiveFrom: "2026-08-01", ReceivableRate: decimal.NewFromInt(7), PayableRate: decimal.NewFromInt(7),
	}
	if _, err := normalizeExchangeRateSetting(input); err != ErrExchangeRateInvalidArgument {
		t.Fatalf("未知汇率类型应被拒绝，实际错误为 %v", err)
	}
}

func TestNormalizeExchangeRateTimeStandardsPreservesOrder(t *testing.T) {
	settings := []*ExchangeRateTimeStandardSetting{
		{RateType: BaseCurrencyRateType, TimeStandards: []string{BusinessTimeStandard, ETDETAOrTrainDateStandard}},
		{RateType: InvoiceRateType, TimeStandards: []string{InvoiceDateStandard}},
		{RateType: SettlementRateType, TimeStandards: []string{TransactionDateStandard}},
		{RateType: WriteOffRateType, TimeStandards: []string{WriteOffTimeStandard}},
		{RateType: BillRateType, TimeStandards: []string{BillDateStandard}},
	}
	value, err := normalizeExchangeRateTimeStandards(settings)
	if err != nil {
		t.Fatalf("规范化汇率时间标准失败: %v", err)
	}
	if value[0].TimeStandards[0] != BusinessTimeStandard || value[0].TimeStandards[1] != ETDETAOrTrainDateStandard {
		t.Fatalf("时间标准顺序未保留: %#v", value[0].TimeStandards)
	}
}

func TestNodeExchangeRateTypesUseIndependentBusinessDates(t *testing.T) {
	cases := []struct {
		rateType string
		standard string
	}{
		{InvoiceRateType, InvoiceDateStandard},
		{SettlementRateType, TransactionDateStandard},
		{WriteOffRateType, WriteOffTimeStandard},
		{BillRateType, BillDateStandard},
	}
	for _, test := range cases {
		settings := []*ExchangeRateTimeStandardSetting{{RateType: test.rateType, TimeStandards: []string{test.standard}}}
		if got := resolveExchangeRateDate(test.rateType, settings, map[string]string{test.standard: "2026-08-27"}); got != "2026-08-27" {
			t.Fatalf("汇率类型 %s 未使用独立业务日期 %s，实际 %s", test.rateType, test.standard, got)
		}
	}
}

func TestNormalizeExchangeRateTimeStandardsRejectsUnsupportedCombination(t *testing.T) {
	settings := []*ExchangeRateTimeStandardSetting{
		{RateType: BaseCurrencyRateType, TimeStandards: []string{WriteOffTimeStandard}},
		{RateType: InvoiceRateType, TimeStandards: []string{InvoiceDateStandard}},
		{RateType: SettlementRateType, TimeStandards: []string{TransactionDateStandard}},
		{RateType: WriteOffRateType, TimeStandards: []string{WriteOffTimeStandard}},
		{RateType: BillRateType, TimeStandards: []string{BillDateStandard}},
	}
	if _, err := normalizeExchangeRateTimeStandards(settings); err != ErrExchangeRateInvalidArgument {
		t.Fatalf("不适用于汇率类型的时间标准应被拒绝，实际错误为 %v", err)
	}
}

func TestResolveBaseCurrencyUsesExactOne(t *testing.T) {
	usecase := NewExchangeRateUsecase(&exchangeRateRepoStub{
		rateContext:   &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "USD"},
		timeStandards: []*ExchangeRateTimeStandardSetting{{RateType: BaseCurrencyRateType, TimeStandards: []string{BusinessTimeStandard, OrderCreatedAtStandard}}},
	})
	resolved, err := usecase.Resolve(context.Background(), uuid.Must(uuid.NewV7()), BaseCurrencyRateType, OrderFeeReceivable, "USD", map[string]string{OrderCreatedAtStandard: "2026-08-24"})
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
		RateType: SettlementRateType, FromCurrency: "USD", ToCurrency: "CNY",
		EffectiveFrom: "2026-08-01", EffectiveTo: &effectiveTo,
		ReceivableRate: decimal.NewFromInt(7), PayableRate: decimal.NewFromInt(7),
	}
	if _, err := normalizeExchangeRateSetting(input); err != ErrExchangeRateInvalidArgument {
		t.Fatalf("无效生效区间应被拒绝，实际错误为 %v", err)
	}
}

func TestNormalizeExchangeRateSettingRejectsExcessPrecision(t *testing.T) {
	input := &ExchangeRateSetting{
		RateType: SettlementRateType, FromCurrency: "USD", ToCurrency: "CNY",
		EffectiveFrom: "2026-08-01", ReceivableRate: decimal.RequireFromString("7.123456789"), PayableRate: decimal.NewFromInt(7),
	}
	if _, err := normalizeExchangeRateSetting(input); err != ErrExchangeRateInvalidArgument {
		t.Fatalf("九位小数汇率应被拒绝，实际错误为 %v", err)
	}
}
