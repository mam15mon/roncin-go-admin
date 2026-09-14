package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/shopspring/decimal"
)

const crossCalcIntegrationDate = "2026-09-12"

// newCrossCalcOrgTree 先补齐币种（组织本位币外键依赖），再创建总部（本位币 CNY）
// + 加拿大分公司（本位币 CAD）组织树。
func newCrossCalcOrgTree(t *testing.T, data *Data) (hqID, branchID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	for _, code := range []string{"USD", "CAD", "CNY"} {
		exists, existErr := data.db.Currency.Query().Where(currencyent.CodeEQ(code)).Exist(ctx)
		if existErr != nil {
			t.Fatalf("查询币种 %s: %v", code, existErr)
		}
		if exists {
			continue
		}
		if _, err := data.db.Currency.Create().SetCode(code).SetName(code).SetEnabled(true).Save(ctx); err != nil {
			t.Fatalf("创建币种 %s: %v", code, err)
		}
	}
	suffix := uuid.NewString()[:8]
	hq, err := data.db.Organization.Create().
		SetCode("XR-HQ-" + suffix).
		SetName("汇率套算测试总部-" + suffix).
		SetKind(organizationent.KindHeadquarters).
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建总部组织: %v", err)
	}
	branch, err := data.db.Organization.Create().
		SetCode("XR-CA-" + suffix).
		SetName("汇率套算测试加拿大分公司-" + suffix).
		SetKind(organizationent.KindCompany).
		SetParentID(hq.ID).
		SetBaseCurrency("CAD").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建分公司组织: %v", err)
	}
	return hq.ID, branch.ID
}

// createCrossCalcRate 插入一条启用汇率基线行（organization_id IS NULL，总部维护、
// 全网可见；from → CNY，上海时区生效起点）。
func createCrossCalcRate(t *testing.T, data *Data, fromCurrency, rate string, effectiveFrom time.Time) {
	t.Helper()
	if _, err := data.db.ExchangeRateSetting.Create().
		SetFromCurrency(fromCurrency).
		SetToCurrency("CNY").
		SetEffectiveFrom(effectiveFrom).
		SetRate(rate).
		SetIsActive(true).
		Save(context.Background()); err != nil {
		t.Fatalf("创建汇率基线行 %s→CNY %s: %v", fromCurrency, rate, err)
	}
}

// createCrossCalcOrgRate 插入一条本组织汇率行（organization_id 非空，本组织覆盖基线）。
func createCrossCalcOrgRate(t *testing.T, data *Data, organizationID uuid.UUID, fromCurrency, rate string, effectiveFrom time.Time) {
	t.Helper()
	if _, err := data.db.ExchangeRateSetting.Create().
		SetOrganizationID(organizationID).
		SetFromCurrency(fromCurrency).
		SetToCurrency("CNY").
		SetEffectiveFrom(effectiveFrom).
		SetRate(rate).
		SetIsActive(true).
		Save(context.Background()); err != nil {
		t.Fatalf("创建组织汇率行 %s→CNY %s: %v", fromCurrency, rate, err)
	}
}

func crossCalcEffectiveDate(month time.Month) time.Time {
	return time.Date(2026, month, 1, 0, 0, 0, 0, biz.ExchangeRateBusinessLocation())
}

// TestExchangeRateCrossCalculationPostgres 在真实汇率行上验证交叉套算矩阵：
// 直连优先、两腿套算、恒等腿、fail-closed、除零守卫、冲突沿用、精度与 CNY 组织回归。
func TestExchangeRateCrossCalculationPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	hqID, branchID := newCrossCalcOrgTree(t, data)
	// USD 腿自 1 月生效；CAD 腿自 6 月生效；GBP/AUD 用于精度断言。
	createCrossCalcRate(t, data, "USD", "7.20000000", crossCalcEffectiveDate(time.January))
	createCrossCalcRate(t, data, "CAD", "5.20000000", crossCalcEffectiveDate(time.June))
	createCrossCalcRate(t, data, "GBP", "7.12345678", crossCalcEffectiveDate(time.January))
	createCrossCalcRate(t, data, "AUD", "5.12345678", crossCalcEffectiveDate(time.January))
	repo := NewExchangeRateRepo(data)

	t.Run("解析上下文携带总部基准币", func(t *testing.T) {
		rateContext, err := repo.ResolveContext(ctx, branchID)
		if err != nil {
			t.Fatalf("解析汇率上下文失败: %v", err)
		}
		if rateContext.OwnerOrganizationID != hqID || rateContext.BaseCurrency != "CAD" || rateContext.PivotCurrency != "CNY" {
			t.Fatalf("上下文应为总部/CAD/CNY，实际 %#v", rateContext)
		}
	})

	t.Run("CAD组织USD费用按两腿套算", func(t *testing.T) {
		resolved, err := repo.ResolveRate(ctx, hqID, biz.OrderFeeReceivable, "USD", "CAD", "CNY", crossCalcIntegrationDate)
		if err != nil {
			t.Fatalf("套算 USD→CAD 失败: %v", err)
		}
		if resolved.Rate.StringFixed(8) != "1.38461538" || resolved.Source != biz.ExchangeRateSourceDerived {
			t.Fatalf("应得 (7.2÷5.2)=1.38461538/DERIVED，实际 %s/%s", resolved.Rate, resolved.Source)
		}
	})

	t.Run("CAD组织CNY费用按恒等腿套算", func(t *testing.T) {
		resolved, err := repo.ResolveRate(ctx, hqID, biz.OrderFeeReceivable, "CNY", "CAD", "CNY", crossCalcIntegrationDate)
		if err != nil {
			t.Fatalf("套算 CNY→CAD 失败: %v", err)
		}
		if resolved.Rate.StringFixed(8) != "0.19230769" || resolved.Source != biz.ExchangeRateSourceDerived {
			t.Fatalf("应得 (1÷5.2)=0.19230769/DERIVED，实际 %s/%s", resolved.Rate, resolved.Source)
		}
	})

	t.Run("to腿未生效时fail-closed", func(t *testing.T) {
		if _, err := repo.ResolveRate(ctx, hqID, biz.OrderFeeReceivable, "USD", "CAD", "CNY", "2026-03-01"); !errors.Is(err, biz.ErrExchangeRateMissing) {
			t.Fatalf("CAD 腿未生效应报缺失，实际 %v", err)
		}
	})

	t.Run("from腿缺失时fail-closed", func(t *testing.T) {
		if _, err := repo.ResolveRate(ctx, hqID, biz.OrderFeeReceivable, "EUR", "CAD", "CNY", crossCalcIntegrationDate); !errors.Is(err, biz.ErrExchangeRateMissing) {
			t.Fatalf("EUR 腿缺失应报缺失，实际 %v", err)
		}
	})

	t.Run("to腿非正数视为缺失", func(t *testing.T) {
		// rate 列无数据库正数约束，直接落零行模拟脏数据，验证除零守卫。
		createCrossCalcRate(t, data, "KRW", "0.00000000", crossCalcEffectiveDate(time.January))
		if _, err := repo.ResolveRate(ctx, hqID, biz.OrderFeeReceivable, "USD", "KRW", "CNY", crossCalcIntegrationDate); !errors.Is(err, biz.ErrExchangeRateMissing) {
			t.Fatalf("to 腿为零应视为缺失，实际 %v", err)
		}
	})

	t.Run("任一腿命中多行沿用冲突错误", func(t *testing.T) {
		createCrossCalcRate(t, data, "USD", "7.30000000", crossCalcEffectiveDate(time.August))
		if _, err := repo.ResolveRate(ctx, hqID, biz.OrderFeeReceivable, "USD", "CAD", "CNY", crossCalcIntegrationDate); !errors.Is(err, biz.ErrExchangeRateConflict) {
			t.Fatalf("USD 腿命中多行应报冲突，实际 %v", err)
		}
	})

	t.Run("直连行优先于套算", func(t *testing.T) {
		if _, err := data.db.ExchangeRateSetting.Create().
			SetOrganizationID(hqID).
			SetFromCurrency("USD").SetToCurrency("CAD").
			SetEffectiveFrom(crossCalcEffectiveDate(time.January)).
			SetRate("1.50000000").SetIsActive(true).
			Save(ctx); err != nil {
			t.Fatalf("创建直连 USD→CAD 行: %v", err)
		}
		resolved, err := repo.ResolveRate(ctx, hqID, biz.OrderFeeReceivable, "USD", "CAD", "CNY", crossCalcIntegrationDate)
		if err != nil {
			t.Fatalf("直连解析失败: %v", err)
		}
		// 本组织行（手工维护）命中：快照来源为 WEEKLY（周汇率口径）。
		if resolved.Rate.StringFixed(8) != "1.50000000" || resolved.Source != biz.ExchangeRateSourceWeekly {
			t.Fatalf("直连行应优先且来源 WEEKLY，实际 %s/%s", resolved.Rate, resolved.Source)
		}
	})

	t.Run("套算精度RoundBank八位", func(t *testing.T) {
		resolved, err := repo.ResolveRate(ctx, hqID, biz.OrderFeeReceivable, "GBP", "AUD", "CNY", crossCalcIntegrationDate)
		if err != nil {
			t.Fatalf("套算 GBP→AUD 失败: %v", err)
		}
		expected := decimal.RequireFromString("7.12345678").Div(decimal.RequireFromString("5.12345678")).RoundBank(8)
		if !resolved.Rate.Equal(expected) || resolved.Rate.StringFixed(8) != "1.39036145" || resolved.Source != biz.ExchangeRateSourceDerived {
			t.Fatalf("应得 1.39036145/DERIVED，实际 %s/%s", resolved.Rate, resolved.Source)
		}
	})

	t.Run("CNY总部组织直连行为不变", func(t *testing.T) {
		// 2026-07-15 仅一条 USD 行生效（第二条自 8 月起），直连命中且无套算参与。
		resolved, err := repo.ResolveRate(ctx, hqID, biz.OrderFeeReceivable, "USD", "CNY", "CNY", "2026-07-15")
		if err != nil {
			t.Fatalf("总部直连解析失败: %v", err)
		}
		if resolved.Rate.StringFixed(8) != "7.20000000" || resolved.Source != biz.ExchangeRateSourceSystem {
			t.Fatalf("总部直连应得 7.20000000/SYSTEM，实际 %s/%s", resolved.Rate, resolved.Source)
		}
		identity, err := repo.ResolveRate(ctx, hqID, biz.OrderFeeReceivable, "CNY", "CNY", "CNY", "2026-07-15")
		if err != nil || !identity.Rate.Equal(decimal.NewFromInt(1)) || identity.Source != biz.ExchangeRateSourceSystem {
			t.Fatalf("同币应恒为 1/SYSTEM，实际 %s/%s error=%v", identity.Rate, identity.Source, err)
		}
	})
}

// TestOrderFeeCrossRateSnapshotPostgres 验证费用创建往返：加拿大分公司订单上的
// USD 费用经真实解析链路套算，快照 source=DERIVED 且金额与手工推导一致；
// 财务补直连行后新费用改用直连值且来源 SYSTEM。
func TestOrderFeeCrossRateSnapshotPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	hqID, branchID := newCrossCalcOrgTree(t, data)
	createCrossCalcRate(t, data, "USD", "7.20000000", crossCalcEffectiveDate(time.January))
	createCrossCalcRate(t, data, "CAD", "5.20000000", crossCalcEffectiveDate(time.January))
	suffix := uuid.NewString()[:8]
	partner, err := data.db.Partner.Create().
		SetOrganizationID(branchID).
		SetCode("XR-PARTY-" + suffix).
		SetLegalName("汇率套算测试客户-" + suffix).
		SetNormalizedName("汇率套算测试客户-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建往来单位: %v", err)
	}
	order, err := data.db.Order.Create().
		SetIdempotencyKey(uuid.NewString()).
		SetOrganizationID(branchID).
		SetOrderNo("SE" + suffix).
		SetCustomerID(partner.ID).
		SetBusinessType(orderent.BusinessTypeSE).
		SetTradeDirection(orderent.TradeDirectionExport).
		SetTradeTerm(orderent.TradeTermFOB).
		SetPaymentTerm(orderent.PaymentTermPREPAID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建订单: %v", err)
	}
	billingUnit, err := data.db.BillingUnit.Create().
		SetCode("XR-UNIT-" + suffix).
		SetName("票").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建计费单位: %v", err)
	}
	taxableService, err := data.db.TaxableService.Create().
		SetOrganizationID(hqID).
		SetName("汇率套算测试应税服务-" + suffix).
		SetDefaultTaxRate("0").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建应税服务: %v", err)
	}
	// A 型费用大类（charge_category）为费用科目必挂外键。
	chargeCategory, err := data.db.MasterDataItem.Create().
		SetKind("charge_category").
		SetCode("XR-CATEGORY-" + suffix).
		SetName("汇率套算测试费用大类").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建费用大类: %v", err)
	}
	// 订单挂载该费用大类为服务类型，费用科目适用性校验（feeSettingApplies）才能命中。
	if _, err := data.db.OrderServiceType.Create().
		SetOrderID(order.ID).
		SetMasterDataItemID(chargeCategory.ID).
		Save(ctx); err != nil {
		t.Fatalf("创建订单服务类型: %v", err)
	}
	// 总部维护的费用科目为 NULL 基线行：订单组织（分公司）经基线共享可见。
	feeSetting, err := data.db.FeeSetting.Create().
		SetFeeCode("XR-FREIGHT-" + suffix).
		SetNameZh("汇率套算测试海运费").
		SetChargeCategoryID(chargeCategory.ID).
		SetDefaultCurrency("USD").
		SetBillingUnitID(billingUnit.ID).
		SetTaxableServiceID(taxableService.ID).
		SetTaxRate("0").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建费用科目: %v", err)
	}
	usecase := biz.NewOrderFeeUsecase(NewOrderFeeRepo(data), biz.NewExchangeRateUsecase(NewExchangeRateRepo(data), nil), nil, biz.NewPartnerCreditUsecase(NewFinanceBillRepo(data), biz.NewFinanceCustomSettingUsecase(NewFinanceCustomSettingRepo(data))))
	newFeeInput := func(key string) *biz.OrderFee {
		return &biz.OrderFee{
			IdempotencyKey:    "fee-" + key + "-" + suffix,
			Direction:         biz.OrderFeeReceivable,
			FeeSettingID:      &feeSetting.ID,
			SettlementPartyID: partner.ID,
			BillingUnitID:     &billingUnit.ID,
			Quantity:          decimal.RequireFromString("2"),
			UnitPrice:         decimal.RequireFromString("100"),
			Currency:          "USD",
			ExpenseDate:       crossCalcIntegrationDate,
			TaxInclusive:      true,
		}
	}

	created, err := usecase.Add(ctx, branchID, uuid.New(), order.ID, newFeeInput("derived"), false)
	if err != nil {
		t.Fatalf("创建套算费用失败: %v", err)
	}
	crossRate := decimal.RequireFromString("7.20000000").Div(decimal.RequireFromString("5.20000000")).RoundBank(8)
	if created.ExchangeRate.StringFixed(8) != crossRate.StringFixed(8) || created.ExchangeRateSource != "DERIVED" {
		t.Fatalf("费用快照应为套算汇率 %s/DERIVED，实际 %s/%s", crossRate, created.ExchangeRate, created.ExchangeRateSource)
	}
	if created.BaseCurrency != "CAD" || !created.BaseCurrencyAmount.Equal(decimal.RequireFromString("200").Mul(crossRate).RoundBank(8)) {
		t.Fatalf("费用本位币金额应按套算汇率折 CAD: %s %s", created.BaseCurrency, created.BaseCurrencyAmount)
	}
	persisted, err := data.db.OrderFee.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("回读费用快照失败: %v", err)
	}
	if persisted.ExchangeRate != crossRate.StringFixed(8) || persisted.ExchangeRateSource != orderfeeent.ExchangeRateSourceDERIVED {
		t.Fatalf("落库快照应为 %s/DERIVED，实际 %s/%s", crossRate, persisted.ExchangeRate, persisted.ExchangeRateSource)
	}

	// 财务补维护直连 USD→CAD 基线行后，新费用改用直连值且来源 SYSTEM
	//（NULL 基线行对分公司费用解析链路可见）。
	if _, err = data.db.ExchangeRateSetting.Create().
		SetFromCurrency("USD").SetToCurrency("CAD").
		SetEffectiveFrom(crossCalcEffectiveDate(time.January)).
		SetRate("1.50000000").SetIsActive(true).
		Save(ctx); err != nil {
		t.Fatalf("创建直连 USD→CAD 行: %v", err)
	}
	direct, err := usecase.Add(ctx, branchID, uuid.New(), order.ID, newFeeInput("direct"), false)
	if err != nil {
		t.Fatalf("创建直连费用失败: %v", err)
	}
	if direct.ExchangeRate.StringFixed(8) != "1.50000000" || direct.ExchangeRateSource != "SYSTEM" || !direct.BaseCurrencyAmount.Equal(decimal.RequireFromString("300.00000000")) {
		t.Fatalf("直连费用应为 1.50000000/SYSTEM/300，实际 %s/%s/%s", direct.ExchangeRate, direct.ExchangeRateSource, direct.BaseCurrencyAmount)
	}
}

// TestExchangeRateBaselineShadowingPostgres 验证汇率点查形态谓词（B 型基线+本地）：
// 无本组织行的组织命中 NULL 基线行；同货币对本组织行覆盖基线行（本组织行优先）；
// uuid.Nil 仅解析基线行（跨组织资金流结算域）。
func TestExchangeRateBaselineShadowingPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	hqID, branchID := newCrossCalcOrgTree(t, data)
	// NULL 基线行（总部维护兜底基线，不挂组织）。
	if _, err := data.db.ExchangeRateSetting.Create().
		SetFromCurrency("USD").SetToCurrency("CNY").
		SetEffectiveFrom(crossCalcEffectiveDate(time.January)).
		SetRate("7.20000000").SetIsActive(true).
		Save(ctx); err != nil {
		t.Fatalf("创建 NULL 基线行: %v", err)
	}
	// 加拿大分公司本组织行：同货币对本地差异化汇率。
	createCrossCalcOrgRate(t, data, branchID, "USD", "7.50000000", crossCalcEffectiveDate(time.January))
	// 无任何本组织行的观察组织。
	observer, err := data.db.Organization.Create().
		SetCode("XR-SG-" + uuid.NewString()[:8]).
		SetName("汇率基线观察组织").
		SetKind(organizationent.KindCompany).
		SetParentID(hqID).
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建观察组织: %v", err)
	}
	repo := NewExchangeRateRepo(data)

	t.Run("无本组织行命中 NULL 基线行", func(t *testing.T) {
		resolved, err := repo.ResolveRate(ctx, observer.ID, biz.OrderFeeReceivable, "USD", "CNY", "CNY", crossCalcIntegrationDate)
		if err != nil {
			t.Fatalf("观察组织解析失败: %v", err)
		}
		if resolved.Rate.StringFixed(8) != "7.20000000" || resolved.Source != biz.ExchangeRateSourceSystem {
			t.Fatalf("无本组织行应命中基线 7.20000000/SYSTEM，实际 %s/%s", resolved.Rate, resolved.Source)
		}
	})

	t.Run("本组织行覆盖基线行", func(t *testing.T) {
		resolved, err := repo.ResolveRate(ctx, branchID, biz.OrderFeeReceivable, "USD", "CNY", "CNY", crossCalcIntegrationDate)
		if err != nil {
			t.Fatalf("分公司解析失败: %v", err)
		}
		// 本组织行覆盖基线行：命中组织行，快照来源为 WEEKLY。
		if resolved.Rate.StringFixed(8) != "7.50000000" || resolved.Source != biz.ExchangeRateSourceWeekly {
			t.Fatalf("本组织行应覆盖基线得 7.50000000/WEEKLY，实际 %s/%s", resolved.Rate, resolved.Source)
		}
	})

	// 退役回归：ResolveRate 的 uuid.Nil「仅基线域」入口已随 ResolveBaseRate 退役
	// 删除——跨组织协同结算按单据原币对账，不再锁定基线行二次折算。
}
