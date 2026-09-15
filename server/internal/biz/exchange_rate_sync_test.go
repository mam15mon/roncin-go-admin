package biz

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// quoteProviderStub 按 CNY/直盘两个入口返回预置结果，可注入失败模拟主源故障。
type quoteProviderStub struct {
	cnySet    *ExchangeRateQuoteSet
	cnyErr    error
	directSet *ExchangeRateQuoteSet
	directErr error
}

func (s *quoteProviderStub) FetchCNYBankQuotes(context.Context, []string, ...string) (*ExchangeRateQuoteSet, error) {
	return s.cnySet, s.cnyErr
}

func (s *quoteProviderStub) FetchDirectQuotes(context.Context, string, []string) (*ExchangeRateQuoteSet, error) {
	return s.directSet, s.directErr
}

func newSyncUsecaseForTest(provider ExchangeRateQuoteProvider, now time.Time) (*ExchangeRateUsecase, *exchangeRateRepoStub) {
	repo := &exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "CNY", PivotCurrency: "CNY"}}
	uc := NewExchangeRateUsecase(repo, provider)
	uc.now = func() time.Time { return now }
	return uc, repo
}

func TestFetchExchangeRatesRoutesCNYBaseToBankQuotes(t *testing.T) {
	now := time.Date(2026, 9, 16, 9, 0, 0, 0, ExchangeRateBusinessLocation())
	provider := &quoteProviderStub{cnySet: &ExchangeRateQuoteSet{Source: "新浪财经中行专线", Quotes: map[string]ExchangeRateQuote{
		"USD": {ARRate: decimal.RequireFromString("6.75"), APRate: decimal.RequireFromString("6.70"), Rate: decimal.RequireFromString("6.725")},
	}}}
	uc, repo := newSyncUsecaseForTest(provider, now)
	repo.rateContext.BaseCurrency = "CNY"
	preview, err := uc.FetchExchangeRates(context.Background(), uuid.Must(uuid.NewV7()), ExchangeRateSyncTargetCurrentWeek)
	if err != nil {
		t.Fatalf("CNY 本币抓取失败: %v", err)
	}
	if preview.Source != "新浪财经中行专线" || preview.FallbackUsed || len(preview.Rows) != 1 {
		t.Fatalf("CNY 本币应走中行专线主源: %#v", preview)
	}
	if preview.EffectiveFrom != "2026-09-14T00:00:00+08:00" || preview.EffectiveTo != "2026-09-20T23:59:59+08:00" {
		t.Fatalf("预览周窗口不正确: %s ~ %s", preview.EffectiveFrom, preview.EffectiveTo)
	}
}

func TestFetchExchangeRatesPrefersDirectQuotesForNonCNYBase(t *testing.T) {
	now := time.Date(2026, 9, 16, 9, 0, 0, 0, ExchangeRateBusinessLocation())
	provider := &quoteProviderStub{directSet: &ExchangeRateQuoteSet{Source: "国际直盘行情", Quotes: map[string]ExchangeRateQuote{
		"USD": {ARRate: decimal.RequireFromString("7.8226"), APRate: decimal.RequireFromString("7.8222"), Rate: decimal.RequireFromString("7.8224")},
	}}}
	uc, repo := newSyncUsecaseForTest(provider, now)
	repo.rateContext.BaseCurrency = "HKD"
	preview, err := uc.FetchExchangeRates(context.Background(), uuid.Must(uuid.NewV7()), ExchangeRateSyncTargetCurrentWeek)
	if err != nil {
		t.Fatalf("非 CNY 本币抓取失败: %v", err)
	}
	if preview.Source != "国际直盘行情" || preview.FallbackUsed {
		t.Fatalf("非 CNY 本币应首选国际直盘: %#v", preview)
	}
}

func TestFetchExchangeRatesFallsBackToBOCCrossQuotes(t *testing.T) {
	now := time.Date(2026, 9, 16, 9, 0, 0, 0, ExchangeRateBusinessLocation())
	provider := &quoteProviderStub{
		directErr: errors.New("network unreachable"),
		cnySet: &ExchangeRateQuoteSet{Source: "中国银行官方牌价", FallbackUsed: true, Quotes: map[string]ExchangeRateQuote{
			"USD": {ARRate: decimal.RequireFromString("6.75"), APRate: decimal.RequireFromString("6.70"), Rate: decimal.RequireFromString("6.725")},
			"HKD": {ARRate: decimal.RequireFromString("0.87"), APRate: decimal.RequireFromString("0.85"), Rate: decimal.RequireFromString("0.86")},
		}},
	}
	uc, repo := newSyncUsecaseForTest(provider, now)
	repo.rateContext.BaseCurrency = "HKD"
	preview, err := uc.FetchExchangeRates(context.Background(), uuid.Must(uuid.NewV7()), ExchangeRateSyncTargetNextWeek)
	if err != nil {
		t.Fatalf("交叉盘兜底失败: %v", err)
	}
	// 备选源必须在预览中明示（来源 + FallbackUsed），换算路径写明交叉算法。
	if !preview.FallbackUsed || preview.Source != "中国银行交叉盘换算" {
		t.Fatalf("备选来源应明示: %#v", preview)
	}
	// 预设下周窗口应为下周一至下周日。
	if preview.EffectiveFrom != "2026-09-21T00:00:00+08:00" || preview.EffectiveTo != "2026-09-27T23:59:59+08:00" {
		t.Fatalf("预设下周窗口不正确: %s ~ %s", preview.EffectiveFrom, preview.EffectiveTo)
	}
	row := preview.Rows[0]
	if row.ConversionPath != "中行交叉盘 USD→CNY ÷ HKD→CNY" {
		t.Fatalf("换算路径应明示交叉算法: %s", row.ConversionPath)
	}
	// 银行交叉商法：ar = 卖出价腿 ÷ 买入价腿 = 6.75 ÷ 0.85。
	expectedAR := decimal.RequireFromString("6.75").Div(decimal.RequireFromString("0.85")).RoundBank(8)
	expectedAP := decimal.RequireFromString("6.70").Div(decimal.RequireFromString("0.87")).RoundBank(8)
	if !row.ARRate.Equal(expectedAR) || !row.APRate.Equal(expectedAP) {
		t.Fatalf("交叉报价应为 ar=(6.75÷0.85)=%s ap=(6.70÷0.87)=%s，实际 %s/%s", expectedAR, expectedAP, row.ARRate, row.APRate)
	}
}

func TestFetchExchangeRatesFailsClosedWhenAllSourcesUnavailable(t *testing.T) {
	now := time.Date(2026, 9, 16, 9, 0, 0, 0, ExchangeRateBusinessLocation())
	provider := &quoteProviderStub{directErr: errors.New("down"), cnyErr: errors.New("down")}
	uc, repo := newSyncUsecaseForTest(provider, now)
	repo.rateContext.BaseCurrency = "HKD"
	if _, err := uc.FetchExchangeRates(context.Background(), uuid.Must(uuid.NewV7()), ExchangeRateSyncTargetCurrentWeek); !errors.Is(err, ErrExchangeRateQuoteUnavailable) {
		t.Fatalf("全部数据源失败应返回业务错误，实际 %v", err)
	}
}

func TestSyncExchangeRatesValidatesAndUpserts(t *testing.T) {
	now := time.Date(2026, 9, 18, 9, 0, 0, 0, ExchangeRateBusinessLocation())
	uc, repo := newSyncUsecaseForTest(&quoteProviderStub{}, now)
	orgID, actorID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	rows := []*ExchangeRateSyncRow{
		{FromCurrency: "USD", ARRate: decimal.RequireFromString("6.7500"), APRate: decimal.RequireFromString("6.7000")},
		{FromCurrency: "eur", ARRate: decimal.RequireFromString("7.9000"), APRate: decimal.RequireFromString("7.8000"), Rate: decimal.RequireFromString("7.85")},
	}
	ctx := principalContext(headquartersPrincipal("system.finance.exchange_rate.create"))
	count, from, to, err := uc.SyncExchangeRates(ctx, orgID, actorID, ExchangeRateSyncTargetCurrentWeek, rows)
	if err != nil {
		t.Fatalf("同步发布失败: %v", err)
	}
	if count != 2 || from != "2026-09-14T00:00:00+08:00" || to != "2026-09-20T23:59:59+08:00" {
		t.Fatalf("同步结果不符: count=%d from=%s to=%s", count, from, to)
	}
	if repo.savedSource != ExchangeRateSettingSourceBOCSync {
		t.Fatalf("同步行来源应标记 BOC_SYNC，实际 %s", repo.savedSource)
	}
	if len(repo.savedSettings) != 2 {
		t.Fatalf("应批量写入 2 行，实际 %d", len(repo.savedSettings))
	}
	for _, input := range repo.savedSettings {
		if input.ToCurrency != "CNY" || input.EffectiveFrom != "2026-09-14T00:00:00+08:00" {
			t.Fatalf("同步行 to_currency 必须是组织本币且锚定当周周一: %#v", input)
		}
	}
	// 总部同步落 NULL 基线行（公共兜底）。
	if repo.savedSettings[0].OrganizationID != nil {
		t.Fatalf("总部同步应写 NULL 基线行，实际 %v", repo.savedSettings[0].OrganizationID)
	}
}

// TestSyncExchangeRatesWritesOrganizationRowForBranch 防越权回归：分公司一键同步
// 必须落本组织 org 行，严禁覆盖全网 NULL 基线行；无权限主体直接 403。
func TestSyncExchangeRatesWritesOrganizationRowForBranch(t *testing.T) {
	now := time.Date(2026, 9, 18, 9, 0, 0, 0, ExchangeRateBusinessLocation())
	uc, repo := newSyncUsecaseForTest(&quoteProviderStub{}, now)
	branchOrg, actorID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	repo.rateContext = &ExchangeRateContext{OwnerOrganizationID: branchOrg, BaseCurrency: "HKD", PivotCurrency: "CNY"}
	rows := []*ExchangeRateSyncRow{
		{FromCurrency: "USD", ARRate: decimal.RequireFromString("7.8226"), APRate: decimal.RequireFromString("7.8222")},
	}
	ctx := principalContext(companyPrincipal("system.finance.exchange_rate.create"))
	count, _, _, err := uc.SyncExchangeRates(ctx, branchOrg, actorID, ExchangeRateSyncTargetCurrentWeek, rows)
	if err != nil || count != 1 {
		t.Fatalf("分公司同步发布失败: count=%d err=%v", count, err)
	}
	if repo.savedSettings[0].OrganizationID == nil || *repo.savedSettings[0].OrganizationID != branchOrg {
		t.Fatalf("分公司同步应写本组织 org 行，实际 %v", repo.savedSettings[0].OrganizationID)
	}
	// 无权限主体（未持汇率维护权限码）应被拒绝，不产生任何写入。
	repo.savedSettings = nil
	denied, _, _, err := uc.SyncExchangeRates(principalContext(companyPrincipal()), branchOrg, actorID, ExchangeRateSyncTargetCurrentWeek, rows)
	if denied != 0 || err != ErrExchangeRatePermissionDenied || len(repo.savedSettings) != 0 {
		t.Fatalf("无权限同步应 403 且不写入: count=%d err=%v saved=%d", denied, err, len(repo.savedSettings))
	}
}

func TestSyncExchangeRatesRejectsInvalidRows(t *testing.T) {
	now := time.Date(2026, 9, 18, 9, 0, 0, 0, ExchangeRateBusinessLocation())
	orgID, actorID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	cases := map[string][]*ExchangeRateSyncRow{
		"空数据":   {},
		"本币自身":  {{FromCurrency: "CNY", ARRate: decimal.NewFromInt(1), APRate: decimal.NewFromInt(1)}},
		"未启用币种": {{FromCurrency: "XYZ", ARRate: decimal.NewFromInt(7), APRate: decimal.NewFromInt(7)}},
		"应收非正数": {{FromCurrency: "USD", ARRate: decimal.Zero, APRate: decimal.NewFromInt(7)}},
		"非法目标周": nil,
	}
	uc, _ := newSyncUsecaseForTest(&quoteProviderStub{}, now)
	if _, _, _, err := uc.SyncExchangeRates(context.Background(), orgID, actorID, "BOGUS", nil); !errors.Is(err, ErrExchangeRateSyncTargetInvalid) {
		t.Fatalf("非法目标周应被拒绝，实际 %v", err)
	}
	if _, _, _, err := uc.SyncExchangeRates(context.Background(), orgID, actorID, ExchangeRateSyncTargetCurrentWeek, nil); !errors.Is(err, ErrExchangeRateSyncRowsEmpty) {
		t.Fatalf("空数据应被拒绝，实际 %v", err)
	}
	for name, rows := range cases {
		if name == "非法目标周" {
			continue
		}
		if _, _, _, err := uc.SyncExchangeRates(context.Background(), orgID, actorID, ExchangeRateSyncTargetCurrentWeek, rows); err == nil {
			t.Fatalf("%s 应被拒绝", name)
		}
	}
}

func TestDeriveCrossQuotesPreservesSpreadDirection(t *testing.T) {
	cnyQuotes := map[string]ExchangeRateQuote{
		"USD": {ARRate: decimal.RequireFromString("6.75"), APRate: decimal.RequireFromString("6.70"), Rate: decimal.RequireFromString("6.725")},
		"SGD": {ARRate: decimal.RequireFromString("5.10"), APRate: decimal.RequireFromString("5.00"), Rate: decimal.RequireFromString("5.05")},
	}
	quotes := DeriveCrossQuotes("SGD", []string{"USD"}, cnyQuotes)
	quote, ok := quotes["USD"]
	if !ok {
		t.Fatalf("USD 交叉报价缺失: %#v", quotes)
	}
	// 卖出价腿 ÷ 卖出价腿保留点差方向：ar > ap。
	if !quote.ARRate.GreaterThan(quote.APRate) {
		t.Fatalf("交叉报价应保留买卖点差方向: ar=%s ap=%s", quote.ARRate, quote.APRate)
	}
	expectedAR := decimal.RequireFromString("6.75").Div(decimal.RequireFromString("5.00")).RoundBank(8)
	expectedAP := decimal.RequireFromString("6.70").Div(decimal.RequireFromString("5.10")).RoundBank(8)
	if !quote.ARRate.Equal(expectedAR) || !quote.APRate.Equal(expectedAP) {
		t.Fatalf("交叉报价应为 ar=(6.75÷5.00)=%s ap=(6.70÷5.10)=%s，实际 %s/%s", expectedAR, expectedAP, quote.ARRate, quote.APRate)
	}
	// 缺少本币 CNY 腿时整体跳过。
	if DeriveCrossQuotes("JPY", []string{"USD"}, cnyQuotes) != nil {
		t.Fatal("本币缺 CNY 腿时应返回空")
	}
}

func TestFetchExchangeRatesRejectsInvalidTarget(t *testing.T) {
	uc, _ := newSyncUsecaseForTest(&quoteProviderStub{}, time.Now())
	if _, err := uc.FetchExchangeRates(context.Background(), uuid.Must(uuid.NewV7()), "BOGUS"); !errors.Is(err, ErrExchangeRateSyncTargetInvalid) {
		t.Fatalf("非法目标周应被拒绝，实际 %v", err)
	}
}
