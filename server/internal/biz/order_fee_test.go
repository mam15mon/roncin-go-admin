package biz

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type orderFeeExchangeRateRepoStub struct {
	ExchangeRateRepo
	resolveCalls int
	resolved     *ResolvedExchangeRate
	rateType     string
}

func (r *orderFeeExchangeRateRepoStub) ResolveContext(context.Context, uuid.UUID) (*ExchangeRateContext, error) {
	return &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "CNY"}, nil
}

func (r *orderFeeExchangeRateRepoStub) ListTimeStandards(context.Context, uuid.UUID) ([]*ExchangeRateTimeStandardSetting, error) {
	return []*ExchangeRateTimeStandardSetting{{RateType: BaseCurrencyRateType, TimeStandards: []string{OrderCreatedAtStandard}}}, nil
}

func (r *orderFeeExchangeRateRepoStub) Resolve(_ context.Context, _ uuid.UUID, rateType string, _ OrderFeeDirection, _, _, _ string) (*ResolvedExchangeRate, error) {
	r.resolveCalls++
	r.rateType = rateType
	return r.resolved, nil
}

type orderFeeRepoStub struct {
	OrderFeeRepo
	rateContext *OrderFeeExchangeRateContext
}

func (r *orderFeeRepoStub) ExchangeRateContext(context.Context, uuid.UUID, uuid.UUID) (*OrderFeeExchangeRateContext, error) {
	return r.rateContext, nil
}

func TestNormalizeOrderFeeCalculatesExactTotal(t *testing.T) {
	fee := validOrderFeeForTest()
	fee.Quantity = decimal.RequireFromString("0.1")
	fee.UnitPrice = decimal.RequireFromString("0.2")

	normalized, err := normalizeOrderFee(fee)
	if err != nil {
		t.Fatalf("规范化费用失败: %v", err)
	}
	if got := normalized.TotalAmount.StringFixed(8); got != "0.02000000" {
		t.Fatalf("总金额应精确为 0.02000000，实际为 %s", got)
	}
}

func TestNormalizeOrderFeePreservesEightDecimalProduct(t *testing.T) {
	fee := validOrderFeeForTest()
	fee.Quantity = decimal.RequireFromString("1.2345")
	fee.UnitPrice = decimal.RequireFromString("6.7891")

	normalized, err := normalizeOrderFee(fee)
	if err != nil {
		t.Fatalf("规范化费用失败: %v", err)
	}
	if got := normalized.TotalAmount.StringFixed(8); got != "8.38114395" {
		t.Fatalf("总金额应保留完整乘积 8.38114395，实际为 %s", got)
	}
}

func TestNormalizeOrderFeeRejectsExcessPrecision(t *testing.T) {
	fee := validOrderFeeForTest()
	fee.UnitPrice = decimal.RequireFromString("1.00001")

	if _, err := normalizeOrderFee(fee); err != ErrOrderFeeInvalidArgument {
		t.Fatalf("五位小数单价应被拒绝，实际错误为 %v", err)
	}
}

func TestNormalizeOrderFeeRequiresCatalogReferences(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*OrderFee)
	}{
		{name: "缺少费用设置", modify: func(value *OrderFee) { value.FeeSettingID = nil }},
		{name: "缺少计费单位", modify: func(value *OrderFee) { value.BillingUnitID = nil }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fee := validOrderFeeForTest()
			test.modify(fee)
			if _, err := normalizeOrderFee(fee); err != ErrOrderFeeInvalidArgument {
				t.Fatalf("%s应被拒绝，实际错误为 %v", test.name, err)
			}
		})
	}
}

func TestResolveOrderFeeExchangeRateRejectsUnauthorizedOverride(t *testing.T) {
	rateRepo := &orderFeeExchangeRateRepoStub{}
	usecase := NewOrderFeeUsecase(nil, NewExchangeRateUsecase(rateRepo))
	override := decimal.RequireFromString("7.12345678")
	fee := validOrderFeeForTest()
	fee.ExchangeRateOverride = &override

	err := usecase.resolveExchangeRate(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), fee, false)
	if err != ErrOrderFeeExchangeRateOverrideForbidden {
		t.Fatalf("无权用户手工覆盖汇率应被拒绝，实际错误为 %v", err)
	}
	if rateRepo.resolveCalls != 0 {
		t.Fatalf("无权覆盖不应继续解析系统汇率，实际调用 %d 次", rateRepo.resolveCalls)
	}
}

func TestResolveOrderFeeExchangeRateUsesExactManualSnapshot(t *testing.T) {
	rateRepo := &orderFeeExchangeRateRepoStub{}
	usecase := NewOrderFeeUsecase(nil, NewExchangeRateUsecase(rateRepo))
	override := decimal.RequireFromString("0.1")
	settingID := uuid.Must(uuid.NewV7())
	fee := validOrderFeeForTest()
	fee.ExchangeRateOverride = &override
	fee.ExchangeRateSettingID = &settingID

	if err := usecase.resolveExchangeRate(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), fee, true); err != nil {
		t.Fatalf("有权用户手工覆盖汇率失败: %v", err)
	}
	if got := fee.ExchangeRate.StringFixed(8); got != "0.10000000" {
		t.Fatalf("手工汇率应精确保存为 0.10000000，实际为 %s", got)
	}
	if fee.ExchangeRateSource != "MANUAL" || fee.ExchangeRateDate != fee.ExpenseDate || fee.ExchangeRateSettingID != nil {
		t.Fatalf("手工汇率快照元数据不正确: source=%s date=%s setting_id=%v", fee.ExchangeRateSource, fee.ExchangeRateDate, fee.ExchangeRateSettingID)
	}
	if rateRepo.resolveCalls != 0 {
		t.Fatalf("手工覆盖时不应解析系统汇率，实际调用 %d 次", rateRepo.resolveCalls)
	}
}

func TestResolveOrderFeeExchangeRateUsesSystemRateWithoutOverride(t *testing.T) {
	settingID := uuid.Must(uuid.NewV7())
	rateRepo := &orderFeeExchangeRateRepoStub{resolved: &ResolvedExchangeRate{
		Rate:      decimal.RequireFromString("7.12345678"),
		Source:    "SYSTEM",
		RateDate:  "2026-08-24",
		SettingID: &settingID,
	}}
	usecase := NewOrderFeeUsecase(&orderFeeRepoStub{rateContext: &OrderFeeExchangeRateContext{OrderCreatedAt: time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)}}, NewExchangeRateUsecase(rateRepo))
	fee := validOrderFeeForTest()
	fee.Currency = "USD"

	if err := usecase.resolveExchangeRate(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), fee, false); err != nil {
		t.Fatalf("系统汇率解析失败: %v", err)
	}
	if got := fee.ExchangeRate.StringFixed(8); got != "7.12345678" || fee.ExchangeRateSource != "SYSTEM" || fee.ExchangeRateSettingID == nil || *fee.ExchangeRateSettingID != settingID {
		t.Fatalf("未覆盖时应使用系统汇率快照，实际 rate=%s source=%s setting_id=%v", got, fee.ExchangeRateSource, fee.ExchangeRateSettingID)
	}
	if rateRepo.resolveCalls != 1 {
		t.Fatalf("未覆盖时应解析一次系统汇率，实际调用 %d 次", rateRepo.resolveCalls)
	}
	if rateRepo.rateType != BaseCurrencyRateType {
		t.Fatalf("订单费用应使用折本币汇率，实际类型为 %s", rateRepo.rateType)
	}
}

func TestNormalizeOrderFeeRejectsExchangeRateOverrideExcessPrecision(t *testing.T) {
	fee := validOrderFeeForTest()
	override := decimal.RequireFromString("1.123456789")
	fee.ExchangeRateOverride = &override

	if _, err := normalizeOrderFee(fee); err != ErrOrderFeeInvalidArgument {
		t.Fatalf("九位小数汇率应被拒绝，实际错误为 %v", err)
	}
}

func validOrderFeeForTest() *OrderFee {
	feeSettingID := uuid.Must(uuid.NewV7())
	billingUnitID := uuid.Must(uuid.NewV7())
	return &OrderFee{
		Direction:         OrderFeeReceivable,
		FeeSettingID:      &feeSettingID,
		SettlementPartyID: uuid.Must(uuid.NewV7()),
		BillingUnitID:     &billingUnitID,
		Quantity:          decimal.RequireFromString("1"),
		UnitPrice:         decimal.RequireFromString("100"),
		Currency:          "CNY",
		ExchangeRate:      decimal.RequireFromString("1"),
		ExpenseDate:       "2026-08-24",
	}
}
