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
	savedSetting   *ExchangeRateSetting
	disabledID     uuid.UUID
}

func (s *exchangeRateRepoStub) ResolveContext(context.Context, uuid.UUID) (*ExchangeRateContext, error) {
	return s.rateContext, nil
}

func (*exchangeRateRepoStub) List(context.Context, uuid.UUID) ([]*ExchangeRateSetting, error) {
	return nil, nil
}

func (s *exchangeRateRepoStub) Create(_ context.Context, organizationID uuid.UUID, input *ExchangeRateSetting, _ *AuditEvent) (*ExchangeRateSetting, error) {
	s.savedSetting = input
	saved := *input
	saved.OrganizationID = organizationID
	return &saved, nil
}

func (s *exchangeRateRepoStub) Update(_ context.Context, organizationID uuid.UUID, input *ExchangeRateSetting, _ *AuditEvent) (*ExchangeRateSetting, error) {
	s.savedSetting = input
	saved := *input
	saved.OrganizationID = organizationID
	return &saved, nil
}

func (s *exchangeRateRepoStub) Disable(_ context.Context, _ uuid.UUID, id uuid.UUID, _ *AuditEvent) error {
	s.disabledID = id
	return nil
}

// ResolveRate 模拟仓储解析：rateByCurrency 表示总部维护的「X → pivot」单边行；
// 目标即 pivot 时单边行就是直连行（SYSTEM）；目标非 pivot 时先查 rateByPair
// 直连行，缺失再按两腿交叉套算（DERIVED），任一腿缺失或 to 腿非正即报缺失。
func (s *exchangeRateRepoStub) ResolveRate(_ context.Context, _ uuid.UUID, fromCurrency, toCurrency, pivotCurrency, rateDate string) (ResolvedRate, error) {
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

func TestNormalizeExchangeRateSettingPreservesEightDecimals(t *testing.T) {
	effectiveTo := "2026-09-01T23:59:59+08:00"
	input := &ExchangeRateSetting{
		FromCurrency: "usd", ToCurrency: "cny",
		EffectiveFrom: "2026-08-01T09:30:00+08:00", EffectiveTo: &effectiveTo,
		Rate: decimal.RequireFromString("7.12345678"),
	}
	value, err := normalizeExchangeRateSetting(input)
	if err != nil {
		t.Fatalf("规范化汇率失败: %v", err)
	}
	if value.FromCurrency != "USD" || value.ToCurrency != "CNY" || value.EffectiveFrom != "2026-08-01T09:30:00+08:00" || value.Rate.StringFixed(8) != "7.12345678" {
		t.Fatalf("汇率规范化结果不正确: %#v", value)
	}
}

func TestNormalizeExchangeRateSettingRejectsNonPositiveRate(t *testing.T) {
	for _, rate := range []decimal.Decimal{decimal.Zero, decimal.NewFromInt(-7), decimal.RequireFromString("0.00000000")} {
		input := &ExchangeRateSetting{
			FromCurrency: "USD", ToCurrency: "CNY",
			EffectiveFrom: "2026-08-01T00:00:00+08:00", Rate: rate,
		}
		if _, err := normalizeExchangeRateSetting(input); err != ErrExchangeRateInvalidArgument {
			t.Fatalf("非正汇率 %s 应被拒绝，实际错误为 %v", rate, err)
		}
	}
}

func TestNormalizeExchangeRateSettingRejectsInvalidInterval(t *testing.T) {
	effectiveTo := "2026-08-01T09:29:59+08:00"
	input := &ExchangeRateSetting{
		FromCurrency: "USD", ToCurrency: "CNY",
		EffectiveFrom: "2026-08-01T09:30:00+08:00", EffectiveTo: &effectiveTo,
		Rate: decimal.NewFromInt(7),
	}
	if _, err := normalizeExchangeRateSetting(input); err != ErrExchangeRateInvalidArgument {
		t.Fatalf("无效生效区间应被拒绝，实际错误为 %v", err)
	}
}

func TestNormalizeExchangeRateSettingRejectsExcessPrecision(t *testing.T) {
	input := &ExchangeRateSetting{
		FromCurrency: "USD", ToCurrency: "CNY",
		EffectiveFrom: "2026-08-01T00:00:00+08:00", Rate: decimal.RequireFromString("7.123456789"),
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
			FromCurrency: "USD", ToCurrency: "CNY",
			EffectiveFrom: effectiveFrom, Rate: decimal.NewFromInt(7),
		}
		if _, err := normalizeExchangeRateSetting(input); err != ErrExchangeRateInvalidArgument {
			t.Fatalf("非带时区秒级时间 %q 应被拒绝，实际错误为 %v", effectiveFrom, err)
		}
	}
}

func TestNormalizeExchangeRateSettingConvertsToBusinessTimezone(t *testing.T) {
	input := &ExchangeRateSetting{
		FromCurrency: "USD", ToCurrency: "CNY",
		EffectiveFrom: "2026-08-27T01:30:00Z", Rate: decimal.NewFromInt(7),
	}
	normalized, err := normalizeExchangeRateSetting(input)
	if err != nil {
		t.Fatalf("规范化带时区时间失败: %v", err)
	}
	if normalized.EffectiveFrom != "2026-08-27T09:30:00+08:00" {
		t.Fatalf("汇率时间未统一到业务时区: %s", normalized.EffectiveFrom)
	}
}

func TestResolveRateReturnsExactOneForBaseCurrency(t *testing.T) {
	repo := &exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "USD"}}
	rate, err := NewExchangeRateUsecase(repo).ResolveRate(context.Background(), uuid.Must(uuid.NewV7()), "USD", "2026-08-27")
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

func TestResolveRateUsesHeadquartersOwnerAndTargetDate(t *testing.T) {
	repo := &exchangeRateRepoStub{
		rateContext:    &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "CNY", PivotCurrency: "CNY"},
		rateByCurrency: map[string]decimal.Decimal{"USD": decimal.RequireFromString("7.20")},
	}
	rate, err := NewExchangeRateUsecase(repo).ResolveRate(context.Background(), uuid.Must(uuid.NewV7()), "usd", "2026-08-27")
	if err != nil {
		t.Fatalf("解析折本币汇率失败: %v", err)
	}
	if rate.Rate.StringFixed(8) != "7.20000000" || rate.Source != ExchangeRateSourceSystem || len(repo.resolveDates) != 1 || repo.resolveDates[0] != "2026-08-27" {
		t.Fatalf("汇率解析结果不正确: rate=%s dates=%v", rate.Rate, repo.resolveDates)
	}
}

func TestResolveRateKeepsMissingError(t *testing.T) {
	repo := &exchangeRateRepoStub{
		rateContext: &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "CNY"},
		resolveErr:  ErrExchangeRateMissing,
	}
	_, err := NewExchangeRateUsecase(repo).ResolveRate(context.Background(), uuid.Must(uuid.NewV7()), "USD", "2026-08-27")
	if err != ErrExchangeRateMissing {
		t.Fatalf("未命中生效汇率应保留缺失错误，实际 %v", err)
	}
}

func TestResolveRateRejectsInvalidDateOrCurrency(t *testing.T) {
	usecase := NewExchangeRateUsecase(&exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "CNY"}})
	org := uuid.Must(uuid.NewV7())
	if _, err := usecase.ResolveRate(context.Background(), org, "USD", "2026-8-27"); err != ErrExchangeRateInvalidArgument {
		t.Fatalf("非法日期应被拒绝，实际 %v", err)
	}
	if _, err := usecase.ResolveRate(context.Background(), org, "US", "2026-08-27"); err != ErrExchangeRateInvalidArgument {
		t.Fatalf("非法币种应被拒绝，实际 %v", err)
	}
	if _, err := usecase.ResolveRate(context.Background(), uuid.Nil, "USD", "2026-08-27"); err != ErrExchangeRateInvalidArgument {
		t.Fatalf("空组织应被拒绝，实际 %v", err)
	}
}

func TestCreateKeepsCallerOrganizationWithoutRedirect(t *testing.T) {
	caller := uuid.Must(uuid.NewV7())
	owner := uuid.Must(uuid.NewV7())
	repo := &exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: owner, BaseCurrency: "CNY"}}
	usecase := NewExchangeRateUsecase(repo)
	created, err := usecase.Create(context.Background(), caller, uuid.Must(uuid.NewV7()), &ExchangeRateSetting{
		FromCurrency: "USD", ToCurrency: "CNY", EffectiveFrom: "2026-08-01T00:00:00+08:00", Rate: decimal.NewFromInt(7),
	})
	if err != nil {
		t.Fatalf("创建汇率失败: %v", err)
	}
	// 写入不再重定向到总部行：调用组织即目标组织，越权由数据层总部校验拒绝。
	if created.OrganizationID != caller || repo.savedSetting.OrganizationID != caller {
		t.Fatalf("汇率写入应保留调用组织 %s，实际 %s / %s", caller, created.OrganizationID, repo.savedSetting.OrganizationID)
	}
}

func TestCreateRejectsNonBaseTargetCurrency(t *testing.T) {
	repo := &exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "CNY"}}
	_, err := NewExchangeRateUsecase(repo).Create(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), &ExchangeRateSetting{
		FromCurrency: "USD", ToCurrency: "EUR", EffectiveFrom: "2026-08-01T00:00:00+08:00", Rate: decimal.NewFromInt(7),
	})
	if err != ErrExchangeRateCurrencyInvalid {
		t.Fatalf("目标币种不是组织本币应被拒绝，实际 %v", err)
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
