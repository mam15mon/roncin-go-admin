package data

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	cash "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	financeinvoice "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
	"github.com/roncin/roncin-go-admin/server/internal/platform/searchtext"
)

func TestSystemPinyinFeeLedgerPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	f := newBillFeeSearchFixture(t, d)
	setting := f.feeSetting("运费别名")
	fee := f.fee("OCEAN", "海运费", "Ocean Freight", &setting.ID)
	repo := NewSettlementRepo(d)
	for _, keyword := range []string{"haiyunfei", "HYF", "yunfeibieming", "yfbm", "Freight", "OCEAN"} {
		result, err := repo.ListFeeLedger(f.ctx, []uuid.UUID{f.organizationID}, biz.FeeLedgerFilter{Page: 1, PageSize: 1, Keyword: keyword})
		if err != nil {
			t.Fatal(err)
		}
		if result.Total != 1 || len(result.Items) != 1 || result.Items[0].Fee.ID != fee.ID || result.MatchedSummary.Count != 1 {
			t.Fatalf("费用台账 %q 结果与完整筛选汇总不一致: %+v", keyword, result)
		}
		other, err := repo.ListFeeLedger(f.ctx, []uuid.UUID{uuid.New()}, biz.FeeLedgerFilter{Page: 1, PageSize: 1, Keyword: keyword})
		if err != nil || other.Total != 0 {
			t.Fatalf("拼音查询越过公司范围: %+v %v", other, err)
		}
	}
}

func TestSystemPinyinOrderFiltersPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	f := newBillFeeSearchFixture(t, d)
	o := d.db.Order.UpdateOneID(f.order.ID).SetShipperShortName("上海发货人").SetConsigneeShortName("北京收货人").SetVesselVoyage("远洋航轮").SetGoodsDescription("冷冻货物").SaveX(f.ctx)
	scopes := []biz.OrderOrganizationScope{{BusinessType: biz.OrderBusinessSE, OrganizationIDs: []uuid.UUID{f.organizationID}}}
	for _, test := range []struct {
		filter biz.OrderListOptions
		want   int
	}{
		{biz.OrderListOptions{ShipperShortName: "shanghaifahuoren"}, 1},
		{biz.OrderListOptions{ShipperShortName: "shfhr"}, 1},
		{biz.OrderListOptions{ConsigneeShortName: "bjshr"}, 1},
		{biz.OrderListOptions{ShipperShortName: "bjshr"}, 0},
		{biz.OrderListOptions{ConsigneeShortName: "shfhr"}, 0},
		{biz.OrderListOptions{Keyword: "ldhw"}, 1},
		{biz.OrderListOptions{Keyword: "yuanyanghanglun"}, 1},
	} {
		test.filter.BusinessType = biz.OrderBusinessSE
		count, err := orderListQuery(d.db, scopes, test.filter).Count(f.ctx)
		if err != nil || count != test.want {
			t.Fatalf("订单拼音条件 %+v: count=%d want=%d err=%v", test.filter, count, test.want, err)
		}
	}
	updated := o.Update().SetShipperShortName("广州发货人").SaveX(f.ctx)
	if strings.Contains(updated.ShipperSearchKeywords, "SHANGHAI") || !strings.Contains(updated.ShipperSearchKeywords, "GUANGZHOU") || updated.ConsigneeSearchKeywords != o.ConsigneeSearchKeywords || updated.SearchKeywords != o.SearchKeywords {
		t.Fatal("修改发货人必须更新自身拼音并保留其他条件检索键")
	}
}

func TestSystemPinyinFinanceSnapshotsPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	t.Run("账单发票按名称快照和当前别名搜索", func(t *testing.T) {
		f := newFinanceInvoicePostgresFixture(t, d)
		bill := f.createConfirmedBill("pinyin")
		invoice := f.createInvoiceWithStatus("pinyin", financeinvoice.StatusISSUED, bill)
		// 名称快照与当前主档不同，仍须能按界面上的快照名称搜索。
		if _, err := d.sqlDB.ExecContext(ctx, `UPDATE partners SET legal_name='新的主档名称',search_keywords=$1 WHERE id=$2`, searchtext.Build("新的主档名称"), f.partnerID); err != nil {
			t.Fatal(err)
		}
		for _, keyword := range []string{"fapiaoshiwuceshikehu", "fpswcskh"} {
			invoices, err := NewFinanceInvoiceRepo(d).List(ctx, f.organizationID, biz.FinanceInvoiceFilter{Page: 1, PageSize: 20, Keyword: keyword})
			if err != nil || invoices.Total != 1 || invoices.Items[0].ID != invoice.ID {
				t.Fatalf("发票快照拼音 %q: %+v %v", keyword, invoices, err)
			}
		}
		bills, err := NewFinanceBillRepo(d).List(ctx, []uuid.UUID{f.organizationID}, biz.FinanceBillFilter{Page: 1, PageSize: 20, Keyword: "fpswcskh"})
		if err != nil || bills.Total != 1 {
			t.Fatalf("账单快照拼音: %+v %v", bills, err)
		}
	})
	t.Run("对冲列表及计提候选", func(t *testing.T) {
		f := newNettingPostgresFixture(t, d)
		created, err := f.newUsecase().Create(ctx, f.organizationID, f.actorID, f.input("pinyin"))
		if err != nil {
			t.Fatal(err)
		}
		result, err := NewFinanceNettingRepo(d).List(ctx, []uuid.UUID{f.organizationID}, biz.FinanceNettingFilter{Page: 1, PageSize: 20, Keyword: "zdswcskh"})
		if err != nil || result.Total != 1 || result.Items[0].ID != created.ID {
			t.Fatalf("对冲拼音: %+v %v", result, err)
		}
	})
	t.Run("收付和核销按原始单位名称检索", func(t *testing.T) {
		f := newVerificationPostgresFixture(t, d)
		created, err := f.newUsecase(NewVerificationRepo(d), NewExchangeRateRepo(d)).Create(ctx, f.organizationID, f.actorID, f.input("pinyin"))
		if err != nil {
			t.Fatal(err)
		}
		cashflows, err := NewFinanceCashflowRepo(d).List(ctx, f.organizationID, biz.FinanceCashflowFilter{Page: 1, PageSize: 20, Keyword: "zdswcskh"})
		if err != nil || cashflows.Total != 1 {
			t.Fatalf("收付拼音: %+v %v", cashflows, err)
		}
		verifications, err := NewVerificationRepo(d).List(ctx, f.organizationID, biz.VerificationFilter{Page: 1, PageSize: 20, Keyword: "zhangdanshiwuceshikehu"})
		if err != nil || verifications.Total != 1 || verifications.Items[0].ID != created.ID {
			t.Fatalf("核销拼音: %+v %v", verifications, err)
		}
	})

}

func TestSystemPinyinMigrationBackfillPostgres(t *testing.T) {
	d, cleanup := getIntegrationDataBeforeMigration(t, "20261116180000")
	defer cleanup()
	ctx := context.Background()
	org, party, _, _, _ := seedAutoConfirmMigrationBase(t, d, "PINYIN-UPGRADE")
	ruleID, contactID, orderID := uuid.New(), uuid.New(), uuid.New()
	for _, q := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO finance_commission_rules(id,created_at,updated_at,organization_id,name,personnel_role,calculation_basis,rate_percent) VALUES($1,now(),now(),$2,'海运提成方案','SALES','REALIZED_PROFIT',10)`, []any{ruleID, org}},
		{`INSERT INTO partner_contacts(id,created_at,updated_at,partner_id,name) VALUES($1,now(),now(),$2,'张三')`, []any{contactID, party}},
		{`INSERT INTO orders(id,created_at,updated_at,organization_id,customer_id,order_no,idempotency_key,business_type,trade_direction,payment_term,shipper_short_name,consignee_short_name,vessel_voyage,goods_description) VALUES($1,now(),now(),$2,$3,'PINYIN-ORDER','PINYIN-ORDER','SE','export','PREPAID','上海发货人','北京收货人','远洋航轮','冷冻货物')`, []any{orderID, org, party}},
	} {
		if _, err := d.sqlDB.ExecContext(ctx, q.query, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if err := migration.Apply(ctx, d.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
			t.Fatal(err)
		}
		if err := BackfillSelectorSearchKeywords(ctx, d.sqlDB); err != nil {
			t.Fatal(err)
		}
	}
	rule := d.db.FinanceCommissionRule.GetX(ctx, ruleID)
	if rule.SearchKeywords != searchtext.Build(rule.Name) || rule.Version != 1 {
		t.Fatal("方案已有数据未正确回填或版本被改变")
	}
	rules, listErr := NewCommissionRepo(d).ListRulesScoped(ctx, []uuid.UUID{org}, biz.CommissionRuleFilter{Page: 1, PageSize: 20, Keyword: "hytcfa"})
	if listErr != nil || rules.Total != 1 || rules.Items[0].ID != ruleID {
		t.Fatalf("提成方案拼音: %+v %v", rules, listErr)
	}

	contact := d.db.PartnerContact.GetX(ctx, contactID)
	if contact.SearchKeywords != searchtext.Build(contact.Name) {
		t.Fatal("联系人未正确回填")
	}
	order := d.db.Order.GetX(ctx, orderID)
	if order.ShipperSearchKeywords != searchtext.Build(order.ShipperShortName) || order.ConsigneeSearchKeywords != searchtext.Build(order.ConsigneeShortName) || order.SearchKeywords != searchtext.Build(order.VesselVoyage, order.GoodsDescription) {
		t.Fatal("订单的字段检索键未分别回填")
	}
	for _, spec := range selectorSearchKeywordBackfills {
		target := spec.targetColumn
		if target == "" {
			target = "search_keywords"
		}
		rows, err := d.sqlDB.QueryContext(ctx, `SELECT `+`"`+target+`" FROM "`+spec.table+`" LIMIT 0`)
		if err != nil {
			t.Fatal(err)
		}
		rows.Close()
	}
	updated := rule.Update().SetName("空运提成方案").SaveX(ctx)
	if updated.SearchKeywords != searchtext.Build("空运提成方案") {
		t.Fatal("方案更名后拼音键未更新")
	}
	// 多字段 hook 的局部修改和清空必须保留另一字段的检索键。
	resource := d.db.EnterpriseResource.Create().SetOrganizationID(org).SetResourceType("ADDRESS").SetShortName("收货地址").SaveX(ctx)
	address := d.db.EnterpriseResourceAddress.Create().SetResourceID(resource.ID).SetCountryCode("CN").SetAddressDetail("浦东新区").SetContactName("李四").SaveX(ctx)
	address = address.Update().ClearContactName().SaveX(ctx)
	if address.SearchKeywords != searchtext.Build(address.AddressDetail) {
		t.Fatal("清空联系人后地址拼音丢失或残留旧名称")
	}
}

func TestSystemPinyinResourceUpdatesPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	org, party, actor, _, _ := seedAutoConfirmMigrationBase(t, d, "PINYIN-RESOURCE")
	contact := d.db.PartnerContact.Create().SetPartnerID(party).SetName("张三").SaveX(ctx)
	repo := NewEnterpriseResourceRepo(d)
	for _, keyword := range []string{"zhangsan", "zs"} {
		items, total, err := repo.SearchPartnerOptions(ctx, org, keyword, 1, 20)
		if err != nil || total != 1 || items[0].ID != party {
			t.Fatalf("客商联系人拼音 %q: %+v %v", keyword, items, err)
		}
	}
	contact.Update().SetName("李四").SaveX(ctx)
	_, total, err := repo.SearchPartnerOptions(ctx, org, "zs", 1, 20)
	if err != nil || total != 0 {
		t.Fatalf("联系人更名残留旧拼音: %d %v", total, err)
	}
	resource := d.db.EnterpriseResource.Create().SetOrganizationID(org).SetResourceType("ADDRESS").SetShortName("客户地址").SaveX(ctx)
	d.db.EnterpriseResourceAddress.Create().SetResourceID(resource.ID).SetCountryCode("CN").SetAddressDetail("浦东新区").SetContactName("李四").SaveX(ctx)
	requireResource := func(keyword string, want int) {
		t.Helper()
		items, total, err := repo.List(ctx, org, biz.EnterpriseResourceListOptions{ResourceType: biz.EnterpriseResourceAddressType, Page: 1, PageSize: 20, Keyword: keyword})
		if err != nil || total != int64(want) || len(items) != want {
			t.Fatalf("资源拼音 %q: total=%d err=%v", keyword, total, err)
		}
	}
	requireResource("pdxq", 1)
	input := &biz.EnterpriseResource{ResourceType: biz.EnterpriseResourceAddressType, Address: &biz.EnterpriseResourceAddress{CountryCode: "CN", AddressDetail: "闵行新区", ContactName: "王五"}}
	if err := d.WithTx(ctx, func(tx *ent.Tx) error { return updateEnterpriseResourceDetail(ctx, tx, resource.ID, org, input) }); err != nil {
		t.Fatal(err)
	}
	requireResource("pdxq", 0)
	requireResource("minhangxinqu", 1)
	requireResource("ww", 1)
	remarkResource := d.db.EnterpriseResource.Create().SetOrganizationID(org).SetResourceType("REMARK").SetShortName("备注模板").SaveX(ctx)
	d.db.EnterpriseResourceRemark.Create().SetResourceID(remarkResource.ID).SetRemarkType("ORDER").SetContent("货物需要冷藏").SaveX(ctx)
	for _, keyword := range []string{"huowuxuyaolengcang", "hwxylc"} {
		items, total, err := repo.List(ctx, org, biz.EnterpriseResourceListOptions{ResourceType: biz.EnterpriseResourceRemarkType, Page: 1, PageSize: 20, Keyword: keyword})
		if err != nil || total != 1 || items[0].ID != remarkResource.ID {
			t.Fatalf("备注内容拼音 %q: %+v %v", keyword, items, err)
		}
	}

	partyResource := d.db.EnterpriseResource.Create().SetOrganizationID(org).SetResourceType("SHIPPER").SetShortName("原名称").SetCreatedBy(actor).SaveX(ctx)
	d.db.EnterpriseResourceParty.Create().SetResourceID(partyResource.ID).SetOrganizationID(org).SetResourceType("SHIPPER").SetCompanyName("旧公司").SetCountryCode("CN").SaveX(ctx)
	if err := d.WithTx(ctx, func(tx *ent.Tx) error {
		return updateImportedEnterpriseResourceParty(ctx, tx, partyResource.ID, &biz.EnterpriseResourceParty{CompanyName: "新公司", CountryCode: "CN", Address: "广州", ContactName: "赵六"})
	}); err != nil {
		t.Fatal(err)
	}
	rows, count, err := repo.List(ctx, org, biz.EnterpriseResourceListOptions{ResourceType: biz.EnterpriseResourceShipperType, Page: 1, PageSize: 20, Keyword: "xgs"})
	if err != nil || count != 1 || rows[0].ID != partyResource.ID {
		t.Fatalf("导入更新未维护拼音: %+v %v", rows, err)
	}
}

func TestSystemPinyinFinanceReportsPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	f := newReportFixture(t, d)
	f.fee(f.order.ID, feeent.DirectionRECEIVABLE, "100", "100")
	f.cashflow(cash.DirectionRECEIVABLE, cash.StatusCONFIRMED, "2026-02-01", "100")
	f.bill(nil, "100")
	repo := NewFinanceReportRepo(d)
	for _, keyword := range []string{"baobiaokehu", "bbkh"} {
		result, err := repo.Profit(ctx, f.org, biz.FinanceReportFilter{Page: 1, PageSize: 1, Keyword: keyword}, false)
		if err != nil || result.Total != 1 {
			t.Fatalf("利润报表拼音 %q: %+v %v", keyword, result, err)
		}
	}
	for _, keyword := range []string{"baobiaojiesuandanwei", "bbjsdw"} {
		for _, export := range []bool{false, true} {
			filter := biz.FinanceReportFilter{Page: 1, PageSize: 1, Keyword: keyword}
			aging, err := repo.Aging(ctx, f.org, filter, export, time.Now())
			if err != nil || aging.Total != 1 {
				t.Fatalf("账龄报表拼音 %q: %+v %v", keyword, aging, err)
			}
			flows, err := repo.Cashflow(ctx, f.org, filter, export)
			if err != nil || flows.Total != 1 {
				t.Fatalf("收付报表拼音 %q: %+v %v", keyword, flows, err)
			}
		}
	}
}
