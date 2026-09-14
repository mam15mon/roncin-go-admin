package biz

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type exchangeRateRepoStub struct {
	ExchangeRateRepo
	rateContext    *ExchangeRateContext
	rateByCurrency map[string]decimal.Decimal
	rateByPair     map[string]decimal.Decimal
	resolveErr     error
	resolveDates   []string
	resolvePairs   []string
	resolvePivots  []string
	savedSettings  []*ExchangeRateSetting
	savedSource    string
	updatedSetting *ExchangeRateSetting
	disabledID     uuid.UUID
}

func (s *exchangeRateRepoStub) ResolveContext(context.Context, uuid.UUID) (*ExchangeRateContext, error) {
	return s.rateContext, nil
}

func (*exchangeRateRepoStub) List(context.Context, uuid.UUID) ([]*ExchangeRateSetting, error) {
	return nil, nil
}

func (s *exchangeRateRepoStub) UpsertWeeklyBatch(_ context.Context, source string, inputs []*ExchangeRateSetting, _ *AuditEvent) ([]*ExchangeRateSetting, error) {
	s.savedSettings = append(s.savedSettings, inputs...)
	s.savedSource = source
	return inputs, nil
}

func (s *exchangeRateRepoStub) UpdateScoped(_ context.Context, _ uuid.UUID, input *ExchangeRateSetting, _ bool, _ *AuditEvent) (*ExchangeRateSetting, error) {
	s.updatedSetting = input
	saved := *input
	return &saved, nil
}

func (s *exchangeRateRepoStub) DisableScoped(_ context.Context, _ uuid.UUID, id uuid.UUID, _ bool, _ *AuditEvent) error {
	s.disabledID = id
	return nil
}

func (*exchangeRateRepoStub) EnabledCurrencyCodes(context.Context) ([]string, error) {
	return []string{"USD", "EUR", "HKD", "CNY", "CAD"}, nil
}

// ResolveRate 模拟仓储解析：rateByCurrency 表示维护的「X → pivot」单边行；
// 目标即 pivot 时单边行就是直连行（SYSTEM）；目标非 pivot 时先查 rateByPair
// 直连行，缺失再按两腿交叉套算（DERIVED），任一腿缺失或 to 腿非正即报缺失。
func (s *exchangeRateRepoStub) ResolveRate(_ context.Context, _ uuid.UUID, _ OrderFeeDirection, fromCurrency, toCurrency, pivotCurrency, rateDate string) (ResolvedRate, error) {
	s.resolveDates = append(s.resolveDates, rateDate)
	s.resolvePairs = append(s.resolvePairs, fromCurrency+"→"+toCurrency)
	s.resolvePivots = append(s.resolvePivots, pivotCurrency)
	if s.resolveErr != nil {
		return ResolvedRate{}, s.resolveErr
	}
	if toCurrency == pivotCurrency {
		rate, ok := s.rateByCurrency[fromCurrency]
		if !ok {
			return ResolvedRate{}, ErrExchangeRateMissing
		}
		return ResolvedRate{Rate: rate, Source: ExchangeRateSourceSystem}, nil
	}
	if rate, ok := s.rateByPair[fromCurrency+"→"+toCurrency]; ok {
		return ResolvedRate{Rate: rate, Source: ExchangeRateSourceSystem}, nil
	}
	leg := func(currency string) (decimal.Decimal, bool) {
		if currency == pivotCurrency {
			return decimal.NewFromInt(1), true
		}
		rate, ok := s.rateByCurrency[currency]
		return rate, ok
	}
	legFrom, fromOK := leg(fromCurrency)
	legTo, toOK := leg(toCurrency)
	if !fromOK || !toOK || !legTo.IsPositive() {
		return ResolvedRate{}, ErrExchangeRateMissing
	}
	return ResolvedRate{Rate: legFrom.Div(legTo).RoundBank(8), Source: ExchangeRateSourceDerived}, nil
}

func validDualRateForTest(ar, ap string) (*decimal.Decimal, *decimal.Decimal) {
	arRate := decimal.RequireFromString(ar)
	apRate := decimal.RequireFromString(ap)
	return &arRate, &apRate
}

func TestNormalizeExchangeRateSettingSnapsToNaturalWeek(t *testing.T) {
	arRate, apRate := validDualRateForTest("6.7500", "6.7000")
	input := &ExchangeRateSetting{
		FromCurrency: "usd", ToCurrency: "cny",
		EffectiveFrom: "2026-08-05T09:30:00+08:00",
		ARRate:        arRate, APRate: apRate,
	}
	value, err := normalizeExchangeRateSetting(input)
	if err != nil {
		t.Fatalf("规范化汇率失败: %v", err)
	}
	// 周内时刻归一化为所在自然周：周一 00:00:00 至周日 23:59:59（Asia/Shanghai）。
	if value.EffectiveFrom != "2026-08-03T00:00:00+08:00" {
		t.Fatalf("生效起点应归一化为周一 00:00:00，实际 %s", value.EffectiveFrom)
	}
	if value.EffectiveTo == nil || *value.EffectiveTo != "2026-08-09T23:59:59+08:00" {
		t.Fatalf("生效终点应归一化为周日 23:59:59，实际 %v", value.EffectiveTo)
	}
	// 基准价缺省按应收/应付中间价记录。
	if value.Rate.StringFixed(8) != "6.72500000" {
		t.Fatalf("基准价缺省应为中间价 6.72500000，实际 %s", value.Rate)
	}
	if value.ARRate.StringFixed(4) != "6.7500" || value.APRate.StringFixed(4) != "6.7000" {
		t.Fatalf("双轨点差应保留，实际 ar=%s ap=%s", value.ARRate, value.APRate)
	}
}

func TestNormalizeExchangeRateSettingRequiresDualRates(t *testing.T) {
	arRate, apRate := validDualRateForTest("6.7500", "6.7000")
	cases := map[string]*ExchangeRateSetting{
		"缺少应收汇率": {FromCurrency: "USD", ToCurrency: "CNY", EffectiveFrom: "2026-08-05T00:00:00+08:00", APRate: apRate},
		"缺少应付汇率": {FromCurrency: "USD", ToCurrency: "CNY", EffectiveFrom: "2026-08-05T00:00:00+08:00", ARRate: arRate},
		"应收非正数":  {FromCurrency: "USD", ToCurrency: "CNY", EffectiveFrom: "2026-08-05T00:00:00+08:00", ARRate: decimalPtr(decimal.Zero), APRate: apRate},
	}
	for name, input := range cases {
		if _, err := normalizeExchangeRateSetting(input); err != ErrExchangeRateInvalidArgument {
			t.Fatalf("%s 应被拒绝，实际错误为 %v", name, err)
		}
	}
}

func TestNormalizeExchangeRateSettingAcceptsExplicitBaselineRate(t *testing.T) {
	arRate, apRate := validDualRateForTest("6.7500", "6.7000")
	input := &ExchangeRateSetting{
		FromCurrency: "USD", ToCurrency: "CNY",
		EffectiveFrom: "2026-08-05", // 纯日期同样按业务时区归周。
		ARRate:        arRate, APRate: apRate,
		Rate: decimal.RequireFromString("6.72"),
	}
	value, err := normalizeExchangeRateSetting(input)
	if err != nil {
		t.Fatalf("规范化汇率失败: %v", err)
	}
	if value.Rate.StringFixed(8) != "6.72000000" {
		t.Fatalf("显式基准价应保留，实际 %s", value.Rate)
	}
}

func TestNormalizeExchangeRateSettingRejectsInvalidInterval(t *testing.T) {
	arRate, apRate := validDualRateForTest("6.7500", "6.7000")
	input := &ExchangeRateSetting{
		FromCurrency: "USD", ToCurrency: "CNY",
		EffectiveFrom: "2026-08-27T09:30:00", // 无时区时间拒绝。
		ARRate:        arRate, APRate: apRate,
	}
	if _, err := normalizeExchangeRateSetting(input); err != ErrExchangeRateInvalidArgument {
		t.Fatalf("无时区生效时刻应被拒绝，实际错误为 %v", err)
	}
}

func TestResolveRateReturnsExactOneForBaseCurrency(t *testing.T) {
	repo := &exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "USD"}}
	rate, err := NewExchangeRateUsecase(repo, nil).ResolveRate(context.Background(), uuid.Must(uuid.NewV7()), OrderFeeReceivable, "USD", "2026-08-27")
	if err != nil {
		t.Fatalf("解析本币汇率失败: %v", err)
	}
	if !rate.Rate.Equal(decimal.NewFromInt(1)) {
		t.Fatalf("本币汇率应为 1，实际 %s", rate.Rate)
	}
	if len(repo.resolveDates) != 0 {
		t.Fatalf("本币汇率不应查询仓储: %v", repo.resolveDates)
	}
}

func TestResolveRateUsesOrganizationOwnerAndTargetDate(t *testing.T) {
	repo := &exchangeRateRepoStub{
		rateContext:    &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "CNY", PivotCurrency: "CNY"},
		rateByCurrency: map[string]decimal.Decimal{"USD": decimal.RequireFromString("7.20")},
	}
	rate, err := NewExchangeRateUsecase(repo, nil).ResolveRate(context.Background(), uuid.Must(uuid.NewV7()), OrderFeeReceivable, "usd", "2026-08-27")
	if err != nil {
		t.Fatalf("解析折本币汇率失败: %v", err)
	}
	if rate.Rate.StringFixed(8) != "7.20000000" || rate.Source != ExchangeRateSourceSystem || len(repo.resolveDates) != 1 || repo.resolveDates[0] != "2026-08-27" {
		t.Fatalf("汇率解析结果不正确: rate=%s dates=%v", rate.Rate, repo.resolveDates)
	}
}

func TestResolveRateRejectsInvalidDirectionDateOrCurrency(t *testing.T) {
	usecase := NewExchangeRateUsecase(&exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "CNY"}}, nil)
	org := uuid.Must(uuid.NewV7())
	if _, err := usecase.ResolveRate(context.Background(), org, OrderFeeDirection("INVALID"), "USD", "2026-08-27"); err != ErrExchangeRateInvalidArgument {
		t.Fatalf("非法收支方向应被拒绝，实际 %v", err)
	}
	if _, err := usecase.ResolveRate(context.Background(), org, OrderFeeReceivable, "USD", "2026-8-27"); err != ErrExchangeRateInvalidArgument {
		t.Fatalf("非法日期应被拒绝，实际 %v", err)
	}
	if _, err := usecase.ResolveRate(context.Background(), org, OrderFeeReceivable, "US", "2026-08-27"); err != ErrExchangeRateInvalidArgument {
		t.Fatalf("非法币种应被拒绝，实际 %v", err)
	}
	if _, err := usecase.ResolveRate(context.Background(), uuid.Nil, OrderFeeReceivable, "USD", "2026-08-27"); err != ErrExchangeRateInvalidArgument {
		t.Fatalf("空组织应被拒绝，实际 %v", err)
	}
}

func TestCreateWritesBaselineRowForHeadquarters(t *testing.T) {
	caller := uuid.Must(uuid.NewV7())
	owner := uuid.Must(uuid.NewV7())
	arRate, apRate := validDualRateForTest("6.75", "6.70")
	repo := &exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: owner, BaseCurrency: "CNY"}}
	usecase := NewExchangeRateUsecase(repo, nil)
	ctx := principalContext(headquartersPrincipal("system.finance.exchange_rate.create"))
	created, err := usecase.Create(ctx, caller, uuid.Must(uuid.NewV7()), &ExchangeRateSetting{
		FromCurrency: "USD", ToCurrency: "CNY", EffectiveFrom: "2026-08-05T00:00:00+08:00", ARRate: arRate, APRate: apRate,
	})
	if err != nil {
		t.Fatalf("创建汇率失败: %v", err)
	}
	// 总部写 NULL 基线行（公共兜底），与存储 B 型结构一致。
	if created.OrganizationID != nil || len(repo.savedSettings) != 1 || repo.savedSettings[0].OrganizationID != nil {
		t.Fatalf("总部写入应落基线行（organization_id 为空），实际 %v", repo.savedSettings)
	}
	if repo.savedSource != ExchangeRateSettingSourceManual {
		t.Fatalf("页面写入来源应为 MANUAL，实际 %s", repo.savedSource)
	}
}

func TestCreateWritesOrganizationRowForBranch(t *testing.T) {
	branchOrg := uuid.Must(uuid.NewV7())
	actor := uuid.Must(uuid.NewV7())
	arRate, apRate := validDualRateForTest("1.10", "1.08")
	repo := &exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: branchOrg, BaseCurrency: "HKD"}}
	usecase := NewExchangeRateUsecase(repo, nil)
	// 分公司上下文只持有权限码即可维护本组织周汇率行。
	ctx := principalContext(companyPrincipal("system.finance.exchange_rate.create"))
	created, err := usecase.Create(ctx, branchOrg, actor, &ExchangeRateSetting{
		FromCurrency: "USD", ToCurrency: "HKD", EffectiveFrom: "2026-08-05T00:00:00+08:00", ARRate: arRate, APRate: apRate,
	})
	if err != nil {
		t.Fatalf("分公司创建汇率失败: %v", err)
	}
	if created.OrganizationID == nil || *created.OrganizationID != branchOrg {
		t.Fatalf("分公司写入应落本组织行，实际 %v", created.OrganizationID)
	}
}

func TestCreateRejectsBranchContextWithoutPermission(t *testing.T) {
	branchOrg := uuid.Must(uuid.NewV7())
	actor := uuid.Must(uuid.NewV7())
	arRate, apRate := validDualRateForTest("1.10", "1.08")
	repo := &exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: branchOrg, BaseCurrency: "HKD"}}
	usecase := NewExchangeRateUsecase(repo, nil)
	ctx := principalContext(companyPrincipal())
	if _, err := usecase.Create(ctx, branchOrg, actor, &ExchangeRateSetting{
		FromCurrency: "USD", ToCurrency: "HKD", EffectiveFrom: "2026-08-05T00:00:00+08:00", ARRate: arRate, APRate: apRate,
	}); err != ErrExchangeRatePermissionDenied {
		t.Fatalf("无权限分公司写汇率应返回 403 业务错误，实际 %v", err)
	}
}

func TestCreateRejectsNonBaseTargetCurrency(t *testing.T) {
	arRate, apRate := validDualRateForTest("6.75", "6.70")
	repo := &exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "CNY"}}
	_, err := NewExchangeRateUsecase(repo, nil).Create(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), &ExchangeRateSetting{
		FromCurrency: "USD", ToCurrency: "EUR", EffectiveFrom: "2026-08-05T00:00:00+08:00", ARRate: arRate, APRate: apRate,
	})
	if err != ErrExchangeRateCurrencyInvalid {
		t.Fatalf("目标币种不是组织本币应被拒绝，实际 %v", err)
	}
}

func TestExchangeRateWeekWindowBounds(t *testing.T) {
	// 2026-09-16 是周三：窗口应锚定本周一 00:00:00 至周日 23:59:59。
	monday, sunday := ExchangeRateWeekWindow(time.Date(2026, 9, 16, 15, 30, 0, 0, ExchangeRateBusinessLocation()))
	if monday.Format("2006-01-02 15:04:05") != "2026-09-14 00:00:00" {
		t.Fatalf("周窗口起点应为周一 00:00:00，实际 %s", monday)
	}
	if sunday.Format("2006-01-02 15:04:05") != "2026-09-20 23:59:59" {
		t.Fatalf("周窗口终点应为周日 23:59:59，实际 %s", sunday)
	}
	// 周日时刻仍属于同一周。
	weekEndFrom, weekEndTo := ExchangeRateWeekWindow(sunday)
	if !weekEndFrom.Equal(monday) || !weekEndTo.Equal(sunday) {
		t.Fatalf("周日时刻应命中同一周窗口: %s ~ %s", weekEndFrom, weekEndTo)
	}
}

func TestExchangeRateSyncWeekWindowSupportsNextWeekPreset(t *testing.T) {
	now := time.Date(2026, 9, 16, 8, 0, 0, 0, ExchangeRateBusinessLocation())
	from, to, ok := ExchangeRateSyncWeekWindow(ExchangeRateSyncTargetCurrentWeek, now)
	if !ok || from.Format("2006-01-02") != "2026-09-14" || to.Format("2006-01-02") != "2026-09-20" {
		t.Fatalf("本周窗口不正确: %s ~ %s ok=%t", from, to, ok)
	}
	nextFrom, nextTo, ok := ExchangeRateSyncWeekWindow(ExchangeRateSyncTargetNextWeek, now)
	if !ok || nextFrom.Format("2006-01-02") != "2026-09-21" || nextTo.Format("2006-01-02") != "2026-09-27" {
		t.Fatalf("预设下周窗口不正确: %s ~ %s ok=%t", nextFrom, nextTo, ok)
	}
	if _, _, ok := ExchangeRateSyncWeekWindow("BOGUS", now); ok {
		t.Fatal("非法目标周应返回 false")
	}
}

func TestParseExchangeRateLookupTimeAcceptsDateAndRFC3339(t *testing.T) {
	dateValue, valid := parseExchangeRateLookupTime("2026-08-27")
	if !valid || dateValue.Format("2006-01-02") != "2026-08-27" {
		t.Fatalf("ISO 日期解析失败: %v %t", dateValue, valid)
	}
	zoneValue, valid := parseExchangeRateLookupTime("2026-08-27T09:30:00+08:00")
	if !valid || zoneValue.In(ExchangeRateBusinessLocation()).Format(time.RFC3339) != "2026-08-27T09:30:00+08:00" {
		t.Fatalf("带时区时间解析失败: %v %t", zoneValue, valid)
	}
	if _, valid := parseExchangeRateLookupTime("2026-08-27 09:30:00"); valid {
		t.Fatal("无时区空格分隔时间不应被接受")
	}
}
