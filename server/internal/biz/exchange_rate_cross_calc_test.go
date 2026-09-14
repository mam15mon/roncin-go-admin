package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// newCrossCalcRepoStub 构造 CAD 本位币组织（总部基准币 CNY）的汇率仓储桩；
// rateByCurrency 表示总部维护的「X → CNY」单边行，rateByPair 表示显式直连行。
func newCrossCalcRepoStub(rateByCurrency map[string]decimal.Decimal, rateByPair map[string]decimal.Decimal) *exchangeRateRepoStub {
	return &exchangeRateRepoStub{
		rateContext:    &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "CAD", PivotCurrency: "CNY"},
		rateByCurrency: rateByCurrency,
		rateByPair:     rateByPair,
	}
}

// TestResolveRateCrossCalculatesUSDForCADBaseOrganization 验证 CAD 本位币组织的
// USD 费用按 (USD→CNY) ÷ (CAD→CNY) 折算，来源 DERIVED，且 pivot 透传到仓储。
func TestResolveRateCrossCalculatesUSDForCADBaseOrganization(t *testing.T) {
	repo := newCrossCalcRepoStub(map[string]decimal.Decimal{
		"USD": decimal.RequireFromString("7.20"),
		"CAD": decimal.RequireFromString("5.20"),
	}, nil)
	resolved, err := NewExchangeRateUsecase(repo, nil).ResolveRate(context.Background(), uuid.Must(uuid.NewV7()), OrderFeeReceivable, "USD", "2026-09-12")
	if err != nil {
		t.Fatalf("交叉套算 USD→CAD 失败: %v", err)
	}
	expected := decimal.RequireFromString("7.20").Div(decimal.RequireFromString("5.20")).RoundBank(8)
	if !resolved.Rate.Equal(expected) || resolved.Rate.StringFixed(8) != "1.38461538" {
		t.Fatalf("交叉汇率应为 (7.20÷5.20)=%s，实际 %s", expected, resolved.Rate)
	}
	if resolved.Source != ExchangeRateSourceDerived {
		t.Fatalf("套算来源应为 DERIVED，实际 %s", resolved.Source)
	}
	if len(repo.resolvePivots) != 1 || repo.resolvePivots[0] != "CNY" {
		t.Fatalf("应经总部基准币 CNY 套算，实际 %v", repo.resolvePivots)
	}
}

// TestResolveRateCrossTreatsPivotCurrencyLegAsIdentity 验证 CNY 费用折 CAD
// 本位币按 1 ÷ (CAD→CNY) 折算（from 即 pivot 的腿恒为 1）。
func TestResolveRateCrossTreatsPivotCurrencyLegAsIdentity(t *testing.T) {
	repo := newCrossCalcRepoStub(map[string]decimal.Decimal{
		"CAD": decimal.RequireFromString("5.20"),
	}, nil)
	resolved, err := NewExchangeRateUsecase(repo, nil).ResolveRate(context.Background(), uuid.Must(uuid.NewV7()), OrderFeeReceivable, "CNY", "2026-09-12")
	if err != nil {
		t.Fatalf("交叉套算 CNY→CAD 失败: %v", err)
	}
	expected := decimal.NewFromInt(1).Div(decimal.RequireFromString("5.20")).RoundBank(8)
	if !resolved.Rate.Equal(expected) || resolved.Rate.StringFixed(8) != "0.19230769" {
		t.Fatalf("交叉汇率应为 (1÷5.20)=%s，实际 %s", expected, resolved.Rate)
	}
	if resolved.Source != ExchangeRateSourceDerived {
		t.Fatalf("套算来源应为 DERIVED，实际 %s", resolved.Source)
	}
}

// TestResolveRatePrefersDirectRowOverCrossCalculation 验证财务显式维护的直连行
// 永远优先于推导值：两腿齐备时仍使用直连 USD→CAD 行且来源 SYSTEM。
func TestResolveRatePrefersDirectRowOverCrossCalculation(t *testing.T) {
	repo := newCrossCalcRepoStub(map[string]decimal.Decimal{
		"USD": decimal.RequireFromString("7.20"),
		"CAD": decimal.RequireFromString("5.20"),
	}, map[string]decimal.Decimal{
		"USD→CAD": decimal.RequireFromString("1.50"),
	})
	resolved, err := NewExchangeRateUsecase(repo, nil).ResolveRate(context.Background(), uuid.Must(uuid.NewV7()), OrderFeeReceivable, "USD", "2026-09-12")
	if err != nil {
		t.Fatalf("解析直连 USD→CAD 失败: %v", err)
	}
	if !resolved.Rate.Equal(decimal.RequireFromString("1.50")) || resolved.Source != ExchangeRateSourceSystem {
		t.Fatalf("直连行应优先且来源 SYSTEM，实际 rate=%s source=%s", resolved.Rate, resolved.Source)
	}
}

// TestResolveRateFailsClosedWhenCrossLegMissing 验证任一套算腿缺失时
// fail-closed 报 ErrExchangeRateMissing，不做回退或静默 1。
func TestResolveRateFailsClosedWhenCrossLegMissing(t *testing.T) {
	cases := map[string]map[string]decimal.Decimal{
		"缺少to腿(CAD→CNY)":   {"USD": decimal.RequireFromString("7.20")},
		"缺少from腿(USD→CNY)": {"CAD": decimal.RequireFromString("5.20")},
	}
	for name, rates := range cases {
		repo := newCrossCalcRepoStub(rates, nil)
		if _, err := NewExchangeRateUsecase(repo, nil).ResolveRate(context.Background(), uuid.Must(uuid.NewV7()), OrderFeeReceivable, "USD", "2026-09-12"); err != ErrExchangeRateMissing {
			t.Fatalf("%s 应报 ErrExchangeRateMissing，实际 %v", name, err)
		}
	}
}

// TestResolveRateTreatsNonPositiveCrossLegAsMissing 验证 to 腿非正数视为缺失
// （除零守卫），与缺腿同语义。
func TestResolveRateTreatsNonPositiveCrossLegAsMissing(t *testing.T) {
	repo := newCrossCalcRepoStub(map[string]decimal.Decimal{
		"USD": decimal.RequireFromString("7.20"),
		"CAD": decimal.Zero,
	}, nil)
	if _, err := NewExchangeRateUsecase(repo, nil).ResolveRate(context.Background(), uuid.Must(uuid.NewV7()), OrderFeeReceivable, "USD", "2026-09-12"); err != ErrExchangeRateMissing {
		t.Fatalf("to 腿为零应视为缺失，实际 %v", err)
	}
}

// 退役回归：ResolveBaseRate 已随阶段二退役，基线交叉套算（DERIVED）仅作为
// 组织解析链的兜底来源存在；跨组织提成与往来按单据原币记账（见
// ResolveCommissionCNYRate 的原币记账口径与其测试）。

// TestResolveOrderFeeExchangeRatePropagatesDerivedSource 验证费用快照来源透传：
// CAD 组织 USD 费用经套算后快照记录 DERIVED（SYSTEM/MANUAL 分别由既有测试锚定）。
func TestResolveOrderFeeExchangeRatePropagatesDerivedSource(t *testing.T) {
	repo := newCrossCalcRepoStub(map[string]decimal.Decimal{
		"USD": decimal.RequireFromString("7.20"),
		"CAD": decimal.RequireFromString("5.20"),
	}, nil)
	usecase := NewOrderFeeUsecase(nil, NewExchangeRateUsecase(repo, nil), nil, newReminderModeCreditControl())
	fee := validOrderFeeForTest()
	fee.Currency = "USD"

	if err := usecase.resolveExchangeRate(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), fee, false); err != nil {
		t.Fatalf("费用汇率解析失败: %v", err)
	}
	expected := decimal.RequireFromString("7.20").Div(decimal.RequireFromString("5.20")).RoundBank(8)
	if !fee.ExchangeRate.Equal(expected) || fee.ExchangeRateSource != "DERIVED" || fee.ExchangeRateDate != fee.ExpenseDate || fee.ExchangeRateSettingID != nil {
		t.Fatalf("费用快照应记录套算汇率与 DERIVED 来源: rate=%s source=%s date=%s setting=%v", fee.ExchangeRate, fee.ExchangeRateSource, fee.ExchangeRateDate, fee.ExchangeRateSettingID)
	}
}

// TestApplyBillExchangeRatePropagatesDerivedSource 验证账单头汇率快照透传导出来源：
// CAD 组织 USD 账单经套算后记录 DERIVED，头本位币金额按套算汇率计算。
func TestApplyBillExchangeRatePropagatesDerivedSource(t *testing.T) {
	repo := newCrossCalcRepoStub(map[string]decimal.Decimal{
		"USD": decimal.RequireFromString("7.20"),
		"CAD": decimal.RequireFromString("5.20"),
	}, nil)
	usecase := NewFinanceBillUsecase(nil, NewExchangeRateUsecase(repo, nil), nil)
	organizationID := uuid.New()
	bill := &FinanceBill{
		Direction:    OrderFeeReceivable,
		Currency:     "USD",
		BaseCurrency: "CAD",
		TotalAmount:  decimal.RequireFromString("100"),
		BillDate:     "2026-09-12",
	}

	if err := usecase.applyBillExchangeRate(context.Background(), organizationID, bill); err != nil {
		t.Fatalf("应用账单日汇率失败: %v", err)
	}
	expected := decimal.RequireFromString("7.20").Div(decimal.RequireFromString("5.20")).RoundBank(8)
	if !bill.ExchangeRate.Equal(expected) || bill.ExchangeRateSource != "DERIVED" {
		t.Fatalf("账单快照应记录套算汇率与 DERIVED 来源: rate=%s source=%s", bill.ExchangeRate, bill.ExchangeRateSource)
	}
	if !bill.BaseCurrencyAmount.Equal(decimal.RequireFromString("100").Mul(expected).RoundBank(8)) {
		t.Fatalf("头本位币金额应按套算汇率计算，实际 %s", bill.BaseCurrencyAmount)
	}
}
