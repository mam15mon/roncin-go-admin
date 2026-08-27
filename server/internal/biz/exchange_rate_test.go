package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type exchangeRateRepoStub struct {
	ExchangeRateRepo
	rateContext        *ExchangeRateContext
	timeStandards      []*ExchangeRateTimeStandardSetting
	resolved           *ResolvedExchangeRate
	resolveErr         error
	resolvedByType     map[string]*ResolvedExchangeRate
	resolveErrByType   map[string]error
	resolveTypes       []string
	customSetting      *ExchangeRateCustomSetting
	savedCustomSetting *ExchangeRateCustomSetting
	savedExpected      uint64
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

func (s *exchangeRateRepoStub) GetCustomSetting(context.Context, uuid.UUID) (*ExchangeRateCustomSetting, error) {
	return s.customSetting, nil
}

func (s *exchangeRateRepoStub) SaveCustomSetting(_ context.Context, setting *ExchangeRateCustomSetting, expectedVersion uint64, _ *AuditEvent) (*ExchangeRateCustomSetting, error) {
	s.savedCustomSetting = setting
	s.savedExpected = expectedVersion
	return setting, nil
}

func (s *exchangeRateRepoStub) Resolve(_ context.Context, _ uuid.UUID, rateType string, _ OrderFeeDirection, _, _, _ string) (*ResolvedExchangeRate, error) {
	s.resolveTypes = append(s.resolveTypes, rateType)
	if s.resolvedByType != nil || s.resolveErrByType != nil {
		return s.resolvedByType[rateType], s.resolveErrByType[rateType]
	}
	return s.resolved, s.resolveErr
}

func TestNormalizeExchangeRateSettingPreservesEightDecimals(t *testing.T) {
	effectiveTo := "2026-09-01T23:59:59+08:00"
	input := &ExchangeRateSetting{
		RateType: SettlementRateType, FromCurrency: "usd", ToCurrency: "cny",
		EffectiveFrom: "2026-08-01T09:30:00+08:00", EffectiveTo: &effectiveTo,
		ReceivableRate: decimal.RequireFromString("7.12345678"), PayableRate: decimal.RequireFromString("7.02345678"),
	}
	value, err := normalizeExchangeRateSetting(input)
	if err != nil {
		t.Fatalf("规范化汇率失败: %v", err)
	}
	if value.FromCurrency != "USD" || value.ToCurrency != "CNY" || value.EffectiveFrom != "2026-08-01T09:30:00+08:00" || value.ReceivableRate.StringFixed(8) != "7.12345678" {
		t.Fatalf("汇率规范化结果不正确: %#v", value)
	}
}

func TestNormalizeExchangeRateSettingAcceptsSupportedRateTypes(t *testing.T) {
	types := []string{BaseCurrencyRateType, InvoiceRateType, SettlementRateType, WriteOffRateType, BillRateType}
	for _, rateType := range types {
		t.Run(rateType, func(t *testing.T) {
			input := &ExchangeRateSetting{
				RateType: rateType, FromCurrency: "USD", ToCurrency: "CNY",
				EffectiveFrom: "2026-08-01T00:00:00+08:00", ReceivableRate: decimal.NewFromInt(7), PayableRate: decimal.NewFromInt(7),
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
		EffectiveFrom: "2026-08-01T00:00:00+08:00", ReceivableRate: decimal.NewFromInt(7), PayableRate: decimal.NewFromInt(7),
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

func TestGetExchangeRateCustomSettingDefaultsToDisabled(t *testing.T) {
	ownerID := uuid.Must(uuid.NewV7())
	usecase := NewExchangeRateUsecase(&exchangeRateRepoStub{
		rateContext: &ExchangeRateContext{OwnerOrganizationID: ownerID, BaseCurrency: "CNY"},
	})
	setting, err := usecase.GetCustomSetting(context.Background(), uuid.Must(uuid.NewV7()))
	if err != nil {
		t.Fatalf("读取默认汇率自定义设置失败: %v", err)
	}
	if setting.OrganizationID != ownerID || setting.InheritBaseCurrencyRate || setting.Version != 0 || setting.UpdatedAt != nil || setting.UpdatedBy != nil {
		t.Fatalf("未保存的自定义设置应默认关闭且版本为 0: %#v", setting)
	}
}

func TestUpdateExchangeRateCustomSettingUsesOwnerOrganizationAndVersion(t *testing.T) {
	ownerID := uuid.Must(uuid.NewV7())
	actorID := uuid.Must(uuid.NewV7())
	repo := &exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: ownerID, BaseCurrency: "CNY"}}
	usecase := NewExchangeRateUsecase(repo)
	if _, err := usecase.UpdateCustomSetting(context.Background(), uuid.Must(uuid.NewV7()), actorID, true, 3); err != nil {
		t.Fatalf("更新汇率自定义设置失败: %v", err)
	}
	if repo.savedCustomSetting == nil || repo.savedCustomSetting.OrganizationID != ownerID || !repo.savedCustomSetting.InheritBaseCurrencyRate || repo.savedCustomSetting.UpdatedBy == nil || *repo.savedCustomSetting.UpdatedBy != actorID || repo.savedExpected != 3 {
		t.Fatalf("汇率自定义设置保存参数不正确: setting=%#v expected=%d", repo.savedCustomSetting, repo.savedExpected)
	}
}

func TestResolveUsesDedicatedRateBeforeInheritance(t *testing.T) {
	dedicatedID := uuid.Must(uuid.NewV7())
	repo := &exchangeRateRepoStub{
		rateContext:   &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "CNY"},
		timeStandards: []*ExchangeRateTimeStandardSetting{{RateType: BillRateType, TimeStandards: []string{BillDateStandard}}},
		resolvedByType: map[string]*ResolvedExchangeRate{
			BillRateType: {Rate: decimal.RequireFromString("7.20"), Source: "SYSTEM", RateDate: "2026-08-27", SettingID: &dedicatedID},
		},
		customSetting: &ExchangeRateCustomSetting{InheritBaseCurrencyRate: true},
	}
	resolved, err := NewExchangeRateUsecase(repo).Resolve(context.Background(), uuid.Must(uuid.NewV7()), BillRateType, OrderFeeReceivable, "USD", map[string]string{BillDateStandard: "2026-08-27"})
	if err != nil {
		t.Fatalf("解析专用账单汇率失败: %v", err)
	}
	if resolved.Source != "SYSTEM" || resolved.SettingID == nil || *resolved.SettingID != dedicatedID || len(repo.resolveTypes) != 1 || repo.resolveTypes[0] != BillRateType {
		t.Fatalf("专用汇率应始终优先，实际 result=%#v calls=%v", resolved, repo.resolveTypes)
	}
}

func TestResolveInheritsBaseCurrencyRateWhenEnabledAndDedicatedMissing(t *testing.T) {
	baseSettingID := uuid.Must(uuid.NewV7())
	repo := &exchangeRateRepoStub{
		rateContext:   &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "CNY"},
		timeStandards: []*ExchangeRateTimeStandardSetting{{RateType: InvoiceRateType, TimeStandards: []string{InvoiceDateStandard}}},
		resolvedByType: map[string]*ResolvedExchangeRate{
			BaseCurrencyRateType: {Rate: decimal.RequireFromString("7.18"), Source: "SYSTEM", RateDate: "2026-08-27", SettingID: &baseSettingID},
		},
		resolveErrByType: map[string]error{InvoiceRateType: ErrExchangeRateMissing},
		customSetting:    &ExchangeRateCustomSetting{InheritBaseCurrencyRate: true},
	}
	resolved, err := NewExchangeRateUsecase(repo).Resolve(context.Background(), uuid.Must(uuid.NewV7()), InvoiceRateType, OrderFeeReceivable, "USD", map[string]string{InvoiceDateStandard: "2026-08-27"})
	if err != nil {
		t.Fatalf("继承折本币汇率失败: %v", err)
	}
	if resolved.Rate.StringFixed(8) != "7.18000000" || resolved.Source != InheritedBaseCurrencySource || resolved.RateDate != "2026-08-27" || resolved.SettingID == nil || *resolved.SettingID != baseSettingID {
		t.Fatalf("继承结果未保留折本币汇率快照: %#v", resolved)
	}
	if len(repo.resolveTypes) != 2 || repo.resolveTypes[0] != InvoiceRateType || repo.resolveTypes[1] != BaseCurrencyRateType {
		t.Fatalf("汇率解析顺序不正确: %v", repo.resolveTypes)
	}
}

func TestResolveDoesNotInheritWhenCustomSettingDisabled(t *testing.T) {
	repo := &exchangeRateRepoStub{
		rateContext:      &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "CNY"},
		timeStandards:    []*ExchangeRateTimeStandardSetting{{RateType: SettlementRateType, TimeStandards: []string{TransactionDateStandard}}},
		resolveErrByType: map[string]error{SettlementRateType: ErrExchangeRateMissing},
		customSetting:    &ExchangeRateCustomSetting{InheritBaseCurrencyRate: false},
	}
	_, err := NewExchangeRateUsecase(repo).Resolve(context.Background(), uuid.Must(uuid.NewV7()), SettlementRateType, OrderFeePayable, "USD", map[string]string{TransactionDateStandard: "2026-08-27"})
	if err != ErrExchangeRateMissing {
		t.Fatalf("关闭继承时应保留专用汇率缺失错误，实际 %v", err)
	}
	if len(repo.resolveTypes) != 1 || repo.resolveTypes[0] != SettlementRateType {
		t.Fatalf("关闭继承时不应查询折本币汇率: %v", repo.resolveTypes)
	}
}

func TestResolveDoesNotMaskDedicatedRateConflict(t *testing.T) {
	repo := &exchangeRateRepoStub{
		rateContext:      &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "CNY"},
		timeStandards:    []*ExchangeRateTimeStandardSetting{{RateType: WriteOffRateType, TimeStandards: []string{WriteOffTimeStandard}}},
		resolveErrByType: map[string]error{WriteOffRateType: ErrExchangeRateConflict},
		customSetting:    &ExchangeRateCustomSetting{InheritBaseCurrencyRate: true},
	}
	_, err := NewExchangeRateUsecase(repo).Resolve(context.Background(), uuid.Must(uuid.NewV7()), WriteOffRateType, OrderFeeReceivable, "USD", map[string]string{WriteOffTimeStandard: "2026-08-27"})
	if err != ErrExchangeRateConflict {
		t.Fatalf("专用汇率冲突不应被继承逻辑掩盖，实际 %v", err)
	}
	if len(repo.resolveTypes) != 1 || repo.resolveTypes[0] != WriteOffRateType {
		t.Fatalf("专用汇率冲突时不应查询折本币汇率: %v", repo.resolveTypes)
	}
}

func TestNormalizeExchangeRateSettingRejectsInvalidInterval(t *testing.T) {
	effectiveTo := "2026-08-01T09:29:59+08:00"
	input := &ExchangeRateSetting{
		RateType: SettlementRateType, FromCurrency: "USD", ToCurrency: "CNY",
		EffectiveFrom: "2026-08-01T09:30:00+08:00", EffectiveTo: &effectiveTo,
		ReceivableRate: decimal.NewFromInt(7), PayableRate: decimal.NewFromInt(7),
	}
	if _, err := normalizeExchangeRateSetting(input); err != ErrExchangeRateInvalidArgument {
		t.Fatalf("无效生效区间应被拒绝，实际错误为 %v", err)
	}
}

func TestNormalizeExchangeRateSettingRejectsExcessPrecision(t *testing.T) {
	input := &ExchangeRateSetting{
		RateType: SettlementRateType, FromCurrency: "USD", ToCurrency: "CNY",
		EffectiveFrom: "2026-08-01T00:00:00+08:00", ReceivableRate: decimal.RequireFromString("7.123456789"), PayableRate: decimal.NewFromInt(7),
	}
	if _, err := normalizeExchangeRateSetting(input); err != ErrExchangeRateInvalidArgument {
		t.Fatalf("九位小数汇率应被拒绝，实际错误为 %v", err)
	}
}

func TestNormalizeExchangeRateSettingRequiresZonedSecondTimestamp(t *testing.T) {
	cases := []string{
		"2026-08-27",
		"2026-08-27T09:30:00",
		"2026-08-27T09:30:00.123+08:00",
	}
	for _, effectiveFrom := range cases {
		input := &ExchangeRateSetting{
			RateType: BillRateType, FromCurrency: "USD", ToCurrency: "CNY",
			EffectiveFrom: effectiveFrom, ReceivableRate: decimal.NewFromInt(7), PayableRate: decimal.NewFromInt(7),
		}
		if _, err := normalizeExchangeRateSetting(input); err != ErrExchangeRateInvalidArgument {
			t.Fatalf("非带时区秒级时间 %q 应被拒绝，实际错误为 %v", effectiveFrom, err)
		}
	}
}

func TestNormalizeExchangeRateSettingConvertsToBusinessTimezone(t *testing.T) {
	input := &ExchangeRateSetting{
		RateType: BillRateType, FromCurrency: "USD", ToCurrency: "CNY",
		EffectiveFrom: "2026-08-27T01:30:00Z", ReceivableRate: decimal.NewFromInt(7), PayableRate: decimal.NewFromInt(7),
	}
	normalized, err := normalizeExchangeRateSetting(input)
	if err != nil {
		t.Fatalf("规范化带时区时间失败: %v", err)
	}
	if normalized.EffectiveFrom != "2026-08-27T09:30:00+08:00" {
		t.Fatalf("汇率时间未统一到业务时区: %s", normalized.EffectiveFrom)
	}
}
