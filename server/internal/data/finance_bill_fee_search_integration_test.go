package data

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/migrate"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
	"github.com/roncin/roncin-go-admin/server/internal/platform/searchtext"
)

type billFeeSearchFixture struct {
	t              *testing.T
	data           *Data
	ctx            context.Context
	organizationID uuid.UUID
	partyID        uuid.UUID
	order          *ent.Order
	billID         uuid.UUID
	usecase        *biz.FinanceBillUsecase
}

// 检索用例仅验证正式迁移后的查询与名称维护，数据由隔离 Schema 整体清理。
func newBillFeeSearchFixture(t *testing.T, data *Data) *billFeeSearchFixture {
	t.Helper()
	ctx := context.Background()
	suffix := uuid.NewString()[:8]
	organizationID, partyID, _, _, _ := seedAutoConfirmMigrationBase(t, data, "SEARCH-"+suffix)
	order, err := data.db.Order.Create().SetOrganizationID(organizationID).SetCustomerID(partyID).
		SetOrderNo("SE-SEARCH-" + suffix).SetIdempotencyKey(uuid.NewString()).SetBusinessType(orderent.BusinessTypeSE).
		SetTradeDirection(orderent.TradeDirectionExport).SetPaymentTerm(orderent.PaymentTermPREPAID).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	bill := createAutoConfirmMigrationBill(t, data, organizationID, partyID, "SEARCH-"+suffix, financebillent.StatusCONFIRMED, nil)
	repo := NewFinanceBillRepo(data)
	f := &billFeeSearchFixture{t: t, data: data, ctx: ctx, organizationID: organizationID, partyID: partyID, order: order, billID: bill.ID, usecase: biz.NewFinanceBillUsecase(repo, nil, nil, nil)}
	attached := f.fee("BASE", "基础账单费用", "", nil)
	_, err = data.db.FinanceBillLine.Create().SetBillID(bill.ID).SetOrderFeeID(attached.ID).SetOrderID(order.ID).
		SetOrderNo(order.OrderNo).SetFeeCode(attached.FeeCode).SetFeeName(attached.FeeName).SetQuantity("1").SetUnitPrice("100").
		SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetCurrency("CNY").SetExchangeRate("1").
		SetBaseCurrency("CNY").SetBaseCurrencyAmount("100").SetActive(true).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *billFeeSearchFixture) fee(code, name, nameEN string, settingID *uuid.UUID) *ent.OrderFee {
	f.t.Helper()
	create := f.data.db.OrderFee.Create().SetOrderID(f.order.ID).SetSettlementPartyID(f.partyID).
		SetIdempotencyKey(uuid.NewString()).SetDirection(orderfeeent.DirectionRECEIVABLE).SetFeeCode(code).SetFeeName(name).
		SetBillingUnit("票").SetQuantity("1").SetUnitPrice("100").SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").
		SetCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(orderfeeent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeBillIntegrationDate).SetBaseCurrency("CNY").SetBaseCurrencyAmount("100").SetVersion(3).
		SetNillableFeeSettingID(settingID)
	if nameEN != "" {
		create.SetFeeNameEn(nameEN)
	}
	fee, err := create.Save(f.ctx)
	if err != nil {
		f.t.Fatal(err)
	}
	return fee
}

func (f *billFeeSearchFixture) feeSetting(alias string) *ent.FeeSetting {
	f.t.Helper()
	suffix := uuid.NewString()[:8]
	unit, err := f.data.db.BillingUnit.Create().SetCode("SEARCH-UNIT-" + suffix).SetName("检索测试计费单位").Save(f.ctx)
	if err != nil {
		f.t.Fatal(err)
	}
	taxable, err := f.data.db.TaxableService.Create().SetOrganizationID(f.organizationID).SetName("检索测试税务名称-" + suffix).SetDefaultTaxRate("0").Save(f.ctx)
	if err != nil {
		f.t.Fatal(err)
	}
	setting, err := f.data.db.FeeSetting.Create().SetOrganizationID(f.organizationID).SetFeeCode("SEARCH-CATALOG-" + suffix).
		SetNameZh("字典费用名称").SetAliasName(alias).SetDefaultCurrency("CNY").SetBillingUnitID(unit.ID).
		SetTaxRate("0").SetTaxableServiceID(taxable.ID).Save(f.ctx)
	if err != nil {
		f.t.Fatal(err)
	}
	return setting
}

func (f *billFeeSearchFixture) requireSearch(keyword string, want ...uuid.UUID) {
	f.t.Helper()
	for _, entry := range []string{"账单追加", "初始建账"} {
		var result *biz.FinanceBillCreationCandidateResult
		var err error
		if entry == "账单追加" {
			result, err = f.usecase.ListFeeCandidates(f.ctx, []uuid.UUID{f.organizationID}, f.billID, biz.FinanceBillFeeCandidateFilter{Page: 1, PageSize: 200, Keyword: keyword})
		} else {
			result, err = f.usecase.ListCreationCandidates(f.ctx, f.organizationID, biz.FinanceBillCreationCandidateFilter{Page: 1, PageSize: 200, Keyword: keyword})
		}
		if err != nil {
			f.t.Fatalf("%s 查询 %q: %v", entry, keyword, err)
		}
		got := make([]uuid.UUID, 0, len(result.Items))
		for _, item := range result.Items {
			if !item.CanBill {
				f.t.Fatalf("%s 查询 %q 返回不可建账费用: %v", entry, keyword, item.BlockedReasons)
			}
			got = append(got, item.Fee.ID)
		}
		if result.Total != int64(len(want)) || !equalUUIDs(sortedUUIDs(got), sortedUUIDs(want)) {
			f.t.Fatalf("%s 查询 %q: ids=%v total=%d want=%v", entry, keyword, got, result.Total, want)
		}
	}
}

func TestFinanceBillFeeCandidateSearchPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	t.Run("快照代码中英文拼音及同公司费目别名均可搜索", func(t *testing.T) {
		f := newBillFeeSearchFixture(t, data)
		setting := f.feeSetting("船运别称")
		target := f.fee("SEARCH_LOOKUP", "海运费", "Ocean Freight", &setting.ID)
		other := newBillFeeSearchFixture(t, data)
		other.fee("SEARCH_LOOKUP", "海运费", "Ocean Freight", nil)
		// 费目已改名时，费用快照名称及其拼音仍然可检索。
		if err := data.db.FeeSetting.UpdateOne(setting).SetNameZh("变更后的字典名称").Exec(f.ctx); err != nil {
			t.Fatal(err)
		}
		for _, keyword := range []string{"search_lookup", "海运费", "ocean freight", "haiyunfei", "hyf", "船运别称", "chuanyunbiecheng", "cybc", f.order.OrderNo} {
			f.requireSearch(keyword, target.ID)
		}
		stored := data.db.OrderFee.GetX(f.ctx, target.ID)
		if stored.SearchKeywords != searchtext.Build("海运费", "Ocean Freight") {
			t.Fatalf("费用快照检索键不应依赖当前费目名称: %q", stored.SearchKeywords)
		}
	})

	t.Run("别名关联受公司限制且不扩大费用公司边界", func(t *testing.T) {
		f := newBillFeeSearchFixture(t, data)
		foreign := newBillFeeSearchFixture(t, data)
		foreignSetting := foreign.feeSetting("仅外公司可见别名")
		foreign.fee("FOREIGN", "其他费用", "", &foreignSetting.ID)
		// 数据库只有 ID 引用约束；即使遗留错挂，查询也不能借其他公司的字典别名命中。
		f.fee("MISMATCH", "当前费用快照", "", &foreignSetting.ID)
		f.requireSearch("仅外公司可见别名")
		f.requireSearch("jwgskjbm")
	})

	t.Run("单行改名清除英文重建键而非名称更新保留键", func(t *testing.T) {
		f := newBillFeeSearchFixture(t, data)
		target := f.fee("NAME_UPDATE", "海运费", "Ocean Freight", nil)
		renamed := data.db.OrderFee.UpdateOne(target).SetFeeName("报关服务").SaveX(f.ctx)
		if renamed.SearchKeywords != searchtext.Build("报关服务", "Ocean Freight") {
			t.Fatalf("仅改中文名必须保留英文来源: %q", renamed.SearchKeywords)
		}
		f.requireSearch("baoguanfuwu", target.ID)
		f.requireSearch("bgfw", target.ID)
		f.requireSearch("haiyunfei")
		cleared := data.db.OrderFee.UpdateOneID(target.ID).ClearFeeNameEn().SaveX(f.ctx)
		if cleared.SearchKeywords != searchtext.Build("报关服务") {
			t.Fatalf("清除英文名应同步去除旧检索词: %q", cleared.SearchKeywords)
		}
		f.requireSearch("ocean freight")
		if err := data.db.OrderFee.UpdateOneID(target.ID).SetNote("仅改备注").AddVersion(1).Exec(f.ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := data.db.OrderFee.Update().Where(orderfeeent.IDEQ(target.ID)).AddVersion(1).Save(f.ctx); err != nil {
			t.Fatal(err)
		}
		stored := data.db.OrderFee.GetX(f.ctx, target.ID)
		if stored.SearchKeywords != cleared.SearchKeywords || stored.Version != target.Version+2 {
			t.Fatalf("非名称单改或批量版本推进不应改写键: %#v", stored)
		}
	})

	t.Run("候选分页边界及关键字过滤仍在数据库分页之前", func(t *testing.T) {
		f := newBillFeeSearchFixture(t, data)
		f.fee("PAGE_ONE", "测试港杂费", "", nil)
		f.fee("PAGE_TWO", "测试港杂费", "", nil)
		page, err := f.usecase.ListFeeCandidates(f.ctx, []uuid.UUID{f.organizationID}, f.billID, biz.FinanceBillFeeCandidateFilter{Page: 2, PageSize: 1, Keyword: "g z f"})
		if err != nil || page.Total != 0 {
			t.Fatalf("不应静默重排用户关键字: result=%#v err=%v", page, err)
		}
		page, err = f.usecase.ListFeeCandidates(f.ctx, []uuid.UUID{f.organizationID}, f.billID, biz.FinanceBillFeeCandidateFilter{Page: 2, PageSize: 1, Keyword: "gangzafei"})
		if err != nil || page.Total != 2 || len(page.Items) != 1 {
			t.Fatalf("服务端过滤与分页不一致: result=%#v err=%v", page, err)
		}
		for _, pagination := range [][2]int{{0, 20}, {-1, 20}, {1, 0}, {1, -1}, {1, 201}} {
			_, err = f.usecase.ListFeeCandidates(f.ctx, []uuid.UUID{f.organizationID}, f.billID, biz.FinanceBillFeeCandidateFilter{Page: pagination[0], PageSize: pagination[1]})
			if !errors.Is(err, biz.ErrFinanceBillInvalidArgument) {
				t.Fatalf("追加候选非法分页 %v 应拒绝: %v", pagination, err)
			}
			_, err = f.usecase.ListCreationCandidates(f.ctx, f.organizationID, biz.FinanceBillCreationCandidateFilter{Page: pagination[0], PageSize: pagination[1]})
			if !errors.Is(err, biz.ErrFinanceBillInvalidArgument) {
				t.Fatalf("建账候选非法分页 %v 应拒绝: %v", pagination, err)
			}
		}
	})
}

func TestFinanceBillFeeOverrideSearchPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	uc := newCorrectionTestUsecase(t, f)
	order := f.createOrder("SEARCH-OVERRIDE-" + f.suffix)
	f.approveInitialFees(order)
	input := correctionTestInput(f, "ADD", "override-search-"+f.suffix, nil, 100)
	name := "港口应急操作"
	input.After.FeeNameOverride = &name
	request, err := uc.Submit(f.ctx, f.approverPrincipal(), f.organizationID, order.ID, input, false)
	if err != nil {
		t.Fatal(err)
	}
	approved := approveCorrectionForTest(t, f, uc, order, request)
	created := f.data.db.OrderFee.Query().Where(orderfeeent.CorrectionRequestIDEQ(approved.ID), orderfeeent.EconomicEffectEQ(orderfeeent.EconomicEffectINCREASE)).OnlyX(f.ctx)
	if created.FeeName != name || created.SearchKeywords != searchtext.Build(name) {
		t.Fatalf("自定义费用名称须经真实更正写入维护检索键: name=%q keywords=%q", created.FeeName, created.SearchKeywords)
	}
	repo := NewFinanceBillRepo(f.data)
	for _, keyword := range []string{"gangkouyingjicaozuo", "gkyjcz"} {
		result, err := repo.ListCreationCandidates(f.ctx, f.organizationID, biz.FinanceBillCreationCandidateFilter{Page: 1, PageSize: 200, Keyword: keyword})
		if err != nil || result.Total != 1 || result.Items[0].Fee.ID != created.ID {
			t.Fatalf("更正名称建账候选未命中 %q: %#v err=%v", keyword, result, err)
		}
		bill := &biz.FinanceBill{OrganizationID: f.organizationID, Direction: biz.OrderFeePayable, SettlementPartyID: f.customerID, Currency: "CNY", BaseCurrency: "CNY"}
		result, err = repo.ListBillFeeCandidates(f.ctx, bill, biz.FinanceBillFeeCandidateFilter{Page: 1, PageSize: 200, Keyword: keyword})
		if err != nil || result.Total != 1 || result.Items[0].Fee.ID != created.ID {
			t.Fatalf("更正名称追加候选未命中 %q: %#v err=%v", keyword, result, err)
		}
	}
}

func TestOrderFeeSearchKeywordsMigrationPostgres(t *testing.T) {
	data, cleanup := getIntegrationDataBeforeMigration(t, "20261114130000")
	t.Cleanup(cleanup)
	ctx := context.Background()
	organizationID, partyID, _, _, _ := seedAutoConfirmMigrationBase(t, data, "SEARCH-UPGRADE")
	order := data.db.Order.Create().SetOrganizationID(organizationID).SetCustomerID(partyID).
		SetOrderNo("SE-SEARCH-UPGRADE").SetIdempotencyKey(uuid.NewString()).SetBusinessType(orderent.BusinessTypeSE).
		SetTradeDirection(orderent.TradeDirectionExport).SetPaymentTerm(orderent.PaymentTermPREPAID).SaveX(ctx)
	feeID := uuid.New()
	if _, err := data.sqlDB.ExecContext(ctx, `INSERT INTO order_fees
		(id,created_at,updated_at,order_id,settlement_party_id,idempotency_key,direction,fee_code,fee_name,fee_name_en,billing_unit,
		 quantity,unit_price,total_amount,net_amount,tax_amount,currency,exchange_rate,exchange_rate_source,
		 exchange_rate_date,base_currency,base_currency_amount,version)
		VALUES($1,now(),now(),$2,$3,$4,'RECEIVABLE','LEGACY_SEARCH','仓储操作费','Storage Handling','票',1,123.45,123.45,123.45,0,'CNY',1,'SYSTEM','2026-11-09','CNY',123.45,3)`, feeID, order.ID, partyID, uuid.NewString()); err != nil {
		t.Fatal(err)
	}
	readBusinessSnapshot := func() string {
		var snapshot string
		if err := data.sqlDB.QueryRowContext(ctx, `SELECT (to_jsonb(f) - 'search_keywords')::text FROM order_fees f WHERE id=$1`, feeID).Scan(&snapshot); err != nil {
			t.Fatal(err)
		}
		return snapshot
	}
	before := readBusinessSnapshot()
	for attempt := 0; attempt < 2; attempt++ {
		if err := migration.Apply(ctx, data.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
			t.Fatal(err)
		}
		if err := BackfillSelectorSearchKeywords(ctx, data.sqlDB); err != nil {
			t.Fatal(err)
		}
		stored := data.db.OrderFee.GetX(ctx, feeID)
		if stored.SearchKeywords != searchtext.Build("仓储操作费", "Storage Handling") || readBusinessSnapshot() != before {
			t.Fatalf("第%d次升级/回填须补键且保留金额、名称、版本和时间: %#v", attempt+1, stored)
		}
	}
	var indexDefinition string
	if err := data.sqlDB.QueryRowContext(ctx, `SELECT indexdef FROM pg_indexes WHERE schemaname=current_schema() AND indexname='order_fees_search_keywords_trgm'`).Scan(&indexDefinition); err != nil || !strings.Contains(indexDefinition, "USING gin (search_keywords gin_trgm_ops)") {
		t.Fatalf("正式 GIN 索引缺失: definition=%q err=%v", indexDefinition, err)
	}
	found := false
	for _, index := range migrate.OrderFeesTable.Indexes {
		if index.Name == "order_fees_search_keywords_trgm" {
			found = index.Annotation != nil && index.Annotation.Type == "GIN" && index.Annotation.OpClass == "gin_trgm_ops"
		}
	}
	if !found {
		t.Fatal("Ent Schema 生成元数据必须保留同名 GIN 与 trigram 操作类")
	}
}
