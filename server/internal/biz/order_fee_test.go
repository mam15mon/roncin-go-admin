package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type orderFeeExchangeRateRepoStub struct {
	ExchangeRateRepo
	resolveCalls int
	rate         decimal.Decimal
	resolveDates []string
}

func (r *orderFeeExchangeRateRepoStub) ResolveContext(context.Context, uuid.UUID) (*ExchangeRateContext, error) {
	return &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "CNY"}, nil
}

func (r *orderFeeExchangeRateRepoStub) ResolveRate(_ context.Context, _ uuid.UUID, _ OrderFeeDirection, _, _, _, rateDate string) (ResolvedRate, error) {
	r.resolveCalls++
	r.resolveDates = append(r.resolveDates, rateDate)
	return ResolvedRate{Rate: r.rate, Source: ExchangeRateSourceSystem}, nil
}

type orderFeeRepoStub struct {
	OrderFeeRepo
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

func TestNormalizeOrderFeeSupportsReceivableAndPayable(t *testing.T) {
	for _, direction := range []OrderFeeDirection{OrderFeeReceivable, OrderFeePayable} {
		t.Run(string(direction), func(t *testing.T) {
			fee := validOrderFeeForTest()
			fee.Direction = direction
			normalized, err := normalizeOrderFee(fee)
			if err != nil {
				t.Fatalf("规范化%s费用失败: %v", direction, err)
			}
			if normalized.Direction != direction {
				t.Fatalf("费用方向 = %s，期望 %s", normalized.Direction, direction)
			}
		})
	}
	fee := validOrderFeeForTest()
	fee.Direction = ""
	if _, err := normalizeOrderFee(fee); err != ErrOrderFeeInvalidArgument {
		t.Fatalf("未指定收付方向应被拒绝，实际错误为 %v", err)
	}
}

func TestResolveOrderFeeExchangeRateRejectsUnauthorizedOverride(t *testing.T) {
	rateRepo := &orderFeeExchangeRateRepoStub{}
	usecase := NewOrderFeeUsecase(nil, NewExchangeRateUsecase(rateRepo, nil), nil, newReminderModeCreditControl(), nil, nil)
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
	usecase := NewOrderFeeUsecase(nil, NewExchangeRateUsecase(rateRepo, nil), nil, newReminderModeCreditControl(), nil, nil)
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
	rateRepo := &orderFeeExchangeRateRepoStub{rate: decimal.RequireFromString("7.12345678")}
	usecase := NewOrderFeeUsecase(&orderFeeRepoStub{}, NewExchangeRateUsecase(rateRepo, nil), nil, newReminderModeCreditControl(), nil, nil)
	fee := validOrderFeeForTest()
	fee.Currency = "USD"

	if err := usecase.resolveExchangeRate(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), fee, false); err != nil {
		t.Fatalf("系统汇率解析失败: %v", err)
	}
	if got := fee.ExchangeRate.StringFixed(8); got != "7.12345678" || fee.ExchangeRateSource != "SYSTEM" || fee.ExchangeRateSettingID != nil {
		t.Fatalf("未覆盖时应使用系统汇率快照，实际 rate=%s source=%s setting_id=%v", got, fee.ExchangeRateSource, fee.ExchangeRateSettingID)
	}
	if rateRepo.resolveCalls != 1 {
		t.Fatalf("未覆盖时应解析一次系统汇率，实际调用 %d 次", rateRepo.resolveCalls)
	}
	if len(rateRepo.resolveDates) != 1 || rateRepo.resolveDates[0] != fee.ExpenseDate {
		t.Fatalf("订单费用汇率必须按费用发生日解析，实际 %v 期望 %s", rateRepo.resolveDates, fee.ExpenseDate)
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

func TestSameOrderFeeCreateIntentAcceptsRetryAndRejectsKeyReuse(t *testing.T) {
	original := validOrderFeeForTest()
	original.TotalAmount = original.Quantity.Mul(original.UnitPrice)
	original.ExchangeRate = decimal.NewFromInt(1)
	retry := *original

	if !sameOrderFeeCreateIntent(original, &retry) {
		t.Fatal("相同费用创建请求应被识别为幂等重试")
	}
	retry.UnitPrice = decimal.RequireFromString("101")
	retry.TotalAmount = retry.Quantity.Mul(retry.UnitPrice)
	if sameOrderFeeCreateIntent(original, &retry) {
		t.Fatal("复用幂等键但金额不同的请求不应返回原费用")
	}
}

func TestCalculateOrderFeeAmountsUsesTaxInclusiveAndBaseCurrencySnapshots(t *testing.T) {
	rateRepo := &orderFeeExchangeRateRepoStub{}
	usecase := NewOrderFeeUsecase(nil, NewExchangeRateUsecase(rateRepo, nil), nil, newReminderModeCreditControl(), nil, nil)
	taxRate := decimal.RequireFromString("6")
	fee := validOrderFeeForTest()
	fee.TaxRate = &taxRate
	fee.TotalAmount = decimal.RequireFromString("106")
	fee.ExchangeRate = decimal.RequireFromString("7.1")
	fee.TaxInclusive = true

	if err := usecase.calculateAmounts(context.Background(), uuid.Must(uuid.NewV7()), fee); err != nil {
		t.Fatalf("计算含税费用金额失败: %v", err)
	}
	if fee.NetAmount.StringFixed(8) != "100.00000000" || fee.TaxAmount.StringFixed(8) != "6.00000000" {
		t.Fatalf("含税拆分结果不正确: net=%s tax=%s", fee.NetAmount, fee.TaxAmount)
	}
	if fee.BaseCurrency != "CNY" || fee.BaseCurrencyAmount.StringFixed(8) != "752.60000000" {
		t.Fatalf("折本币快照不正确: currency=%s amount=%s", fee.BaseCurrency, fee.BaseCurrencyAmount)
	}
}

func TestCalculateOrderFeeAmountsAddsTaxForExclusivePrice(t *testing.T) {
	rateRepo := &orderFeeExchangeRateRepoStub{}
	usecase := NewOrderFeeUsecase(nil, NewExchangeRateUsecase(rateRepo, nil), nil, newReminderModeCreditControl(), nil, nil)
	taxRate := decimal.RequireFromString("6")
	fee := validOrderFeeForTest()
	fee.TaxRate = &taxRate
	fee.TotalAmount = decimal.RequireFromString("100")
	fee.ExchangeRate = decimal.NewFromInt(1)
	fee.TaxInclusive = false

	if err := usecase.calculateAmounts(context.Background(), uuid.Must(uuid.NewV7()), fee); err != nil {
		t.Fatalf("计算不含税费用金额失败: %v", err)
	}
	if fee.TotalAmount.StringFixed(8) != "106.00000000" || fee.NetAmount.StringFixed(8) != "100.00000000" || fee.TaxAmount.StringFixed(8) != "6.00000000" {
		t.Fatalf("不含税加税结果不正确: total=%s net=%s tax=%s", fee.TotalAmount, fee.NetAmount, fee.TaxAmount)
	}
}

func TestNormalizeOrderFeeRequiresIdempotencyKeyForCreation(t *testing.T) {
	fee := validOrderFeeForTest()
	fee.IdempotencyKey = ""
	if _, err := normalizeOrderFee(fee); err != ErrOrderFeeInvalidArgument {
		t.Fatalf("新建费用缺少幂等键应被拒绝，实际错误为 %v", err)
	}
}

func TestOrderFeeCreditGateBlocksExceededReceivablePartyOnly(t *testing.T) {
	ctx := context.Background()
	organizationID := uuid.Must(uuid.NewV7())
	exceededPartyID := uuid.Must(uuid.NewV7())
	creditLimit := decimal.NewFromInt(100)
	creditControl := NewPartnerCreditUsecase(&partnerCreditRepoStub{summaries: map[uuid.UUID]*PartnerCreditSummary{
		exceededPartyID: {PartnerID: exceededPartyID, CreditLimitBase: &creditLimit, UnsettledReceivableBase: decimal.NewFromInt(200)},
	}}, NewFinanceCustomSettingUsecase(&interventionModeSettingRepo{allowSelection: false}))
	usecase := NewOrderFeeUsecase(nil, NewExchangeRateUsecase(&orderFeeExchangeRateRepoStub{}, nil), nil, creditControl, nil, nil)

	receivable := validOrderFeeForTest()
	receivable.SettlementPartyID = exceededPartyID
	if err := usecase.ensureReceivablePartySelectionAllowed(ctx, organizationID, receivable); err != ErrPartnerCreditLimitExceeded {
		t.Fatalf("直接干预模式下应收费用超额结算单位应被拦截, got %v", err)
	}

	payable := validOrderFeeForTest()
	payable.Direction = OrderFeePayable
	payable.SettlementPartyID = exceededPartyID
	if err := usecase.ensureReceivablePartySelectionAllowed(ctx, organizationID, payable); err != nil {
		t.Fatalf("应付方向不占用客户信用额度，不应被拦截: %v", err)
	}

	reminderUsecase := NewOrderFeeUsecase(nil, NewExchangeRateUsecase(&orderFeeExchangeRateRepoStub{}, nil), nil, newReminderModeCreditControl(), nil, nil)
	if err := reminderUsecase.ensureReceivablePartySelectionAllowed(ctx, organizationID, receivable); err != nil {
		t.Fatalf("仅提醒模式下超额客户应放行: %v", err)
	}
}

func TestValidateBilledFeeUpdateEnforcesSwitchAndFieldAllowlist(t *testing.T) {
	current := billedOrderFeeForTest()
	requested := *current
	requested.Quantity = decimal.RequireFromString("2")

	if err := ValidateBilledFeeUpdate(current, &requested, &BilledFeeEditPolicy{}); err != ErrBilledFeeEditDisabled {
		t.Fatalf("总开关关闭时应拒绝修改，实际错误为 %v", err)
	}
	if err := ValidateBilledFeeUpdate(current, &requested, &BilledFeeEditPolicy{Enabled: true}); err != ErrBilledFeeFieldForbidden {
		t.Fatalf("数量未授权时应拒绝修改，实际错误为 %v", err)
	}
	if err := ValidateBilledFeeUpdate(current, &requested, &BilledFeeEditPolicy{Enabled: true, EditableFields: []BilledFeeEditableField{BilledFeeFieldQuantity}}); err != nil {
		t.Fatalf("数量已授权时应允许修改，实际错误为 %v", err)
	}
}

func TestValidateBilledFeeUpdateAllowsSystemRateChangeWithCurrencyPermission(t *testing.T) {
	current := billedOrderFeeForTest()
	requested := *current
	requested.Currency = "USD"
	requested.ExchangeRate = decimal.RequireFromString("7.2")

	policy := &BilledFeeEditPolicy{Enabled: true, EditableFields: []BilledFeeEditableField{BilledFeeFieldCurrency}}
	if err := ValidateBilledFeeUpdate(current, &requested, policy); err != nil {
		t.Fatalf("币种授权应包含系统重新解析出的汇率，实际错误为 %v", err)
	}

	manualRate := decimal.RequireFromString("7.3")
	requested.ExchangeRate = manualRate
	requested.ExchangeRateOverride = &manualRate
	if err := ValidateBilledFeeUpdate(current, &requested, policy); err != ErrBilledFeeFieldForbidden {
		t.Fatalf("币种授权不应隐含手工汇率权限，实际错误为 %v", err)
	}
	policy.EditableFields = append(policy.EditableFields, BilledFeeFieldExchangeRate)
	if err := ValidateBilledFeeUpdate(current, &requested, policy); err != nil {
		t.Fatalf("币种与汇率均授权时应允许手工汇率，实际错误为 %v", err)
	}
}

func TestValidateBilledFeeUpdateRejectsUnlistedBusinessFields(t *testing.T) {
	current := billedOrderFeeForTest()
	requested := *current
	requested.SettlementPartyID = uuid.Must(uuid.NewV7())
	policy := &BilledFeeEditPolicy{Enabled: true, EditableFields: []BilledFeeEditableField{
		BilledFeeFieldFeeName, BilledFeeFieldCurrency, BilledFeeFieldExchangeRate,
		BilledFeeFieldQuantity, BilledFeeFieldUnitPrice, BilledFeeFieldTaxRate,
	}}
	if err := ValidateBilledFeeUpdate(current, &requested, policy); err != ErrBilledFeeFieldForbidden {
		t.Fatalf("结算单位不在允许字段中，必须拒绝修改，实际错误为 %v", err)
	}
}

func billedOrderFeeForTest() *OrderFee {
	fee := validOrderFeeForTest()
	taxRate := decimal.RequireFromString("6")
	fee.Status = OrderFeeBilled
	fee.FeeCode = "OCEAN_FREIGHT"
	fee.FeeName = "海运费"
	fee.BillingUnit = "票"
	fee.TaxRate = &taxRate
	fee.TaxableServiceName = stringPointer("国际货物运输代理服务")
	fee.ExchangeRate = decimal.NewFromInt(1)
	fee.ExchangeRateSource = "BASE_CURRENCY"
	fee.ExchangeRateDate = fee.ExpenseDate
	fee.BaseCurrency = "CNY"
	fee.TotalAmount = decimal.RequireFromString("106")
	fee.NetAmount = decimal.RequireFromString("100")
	fee.TaxAmount = decimal.RequireFromString("6")
	fee.BaseCurrencyAmount = decimal.RequireFromString("106")
	fee.Version = 3
	return fee
}

func validOrderFeeForTest() *OrderFee {
	feeSettingID := uuid.Must(uuid.NewV7())
	billingUnitID := uuid.Must(uuid.NewV7())
	return &OrderFee{
		IdempotencyKey:    uuid.Must(uuid.NewV7()).String(),
		Direction:         OrderFeeReceivable,
		FeeSettingID:      &feeSettingID,
		SettlementPartyID: uuid.Must(uuid.NewV7()),
		BillingUnitID:     &billingUnitID,
		Quantity:          decimal.RequireFromString("1"),
		UnitPrice:         decimal.RequireFromString("100"),
		Currency:          "CNY",
		ExchangeRate:      decimal.RequireFromString("1"),
		ExpenseDate:       "2026-08-24",
		TaxInclusive:      true,
	}
}
