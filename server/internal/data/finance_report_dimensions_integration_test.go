package data

import (
	"context"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/shopspring/decimal"
)

func TestFinanceReportCurrencyDimensionsPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	f := newReportFixture(t, d)
	f.fee(f.order.ID, feeent.DirectionRECEIVABLE, "100", "100")
	usd := f.fee(f.order.ID, feeent.DirectionRECEIVABLE, "20", "16")
	d.db.OrderFee.UpdateOneID(usd.ID).SetCurrency("USD").SetExchangeRate("7").SetBaseCurrencyAmount("140").SaveX(ctx)
	eurBase := f.fee(f.order.ID, feeent.DirectionPAYABLE, "10", "8")
	d.db.OrderFee.UpdateOneID(eurBase.ID).SetCurrency("EUR").SetBaseCurrency("EUR").SaveX(ctx)
	result, err := NewFinanceReportRepo(d).Profit(ctx, f.org, biz.FinanceReportFilter{Page: 1, PageSize: 20}, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.OrderCount != 1 || result.Total != 2 || len(result.Items) != 2 || len(result.Totals) != 2 || len(result.NativeTotals) != 3 {
		t.Fatalf("同一订单按本币分行，原币分组不能跨币种相加: %#v", result)
	}
	if result.Totals[0].BaseCurrency != "CNY" || result.Totals[0].ProfitAmount.String() != "240" || result.Totals[0].NetProfitAmount.String() != "212" || result.Totals[1].BaseCurrency != "EUR" || result.Totals[1].ProfitAmount.String() != "-10" {
		t.Fatalf("应沿每笔费用的本币快照及未税折算汇总: %#v", result.Totals)
	}
	if len(result.Months) != 1 || len(result.Months[0].Amounts) != 2 {
		t.Fatalf("同源月序列保留两个独立本币桶: %#v", result.Months)
	}
}

func TestFinanceReportOptionsSearchAndCompanyPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	f, other := newReportFixture(t, d), newReportFixture(t, d)
	d.db.Partner.UpdateOneID(f.party).SetCode("REPORT-QDYH").SetLegalName("青岛远航").SetNormalizedName("青岛远航").SaveX(ctx)
	d.db.PartnerProfile.Create().SetPartnerID(f.party).SetNameEn("OceanReport").SaveX(ctx)
	d.db.PartnerAlias.Create().SetPartnerID(f.party).SetAliasName("报表分支").SetNormalizedAliasName("报表分支").SaveX(ctx)
	d.db.Partner.UpdateOneID(other.party).SetLegalName("青岛远航外公司").SetNormalizedName("青岛远航外公司").SaveX(ctx)
	d.db.User.UpdateOneID(f.actor).SetDisplayName("报表人员").SaveX(ctx)
	d.db.User.UpdateOneID(other.actor).SetDisplayName("报表人员外公司").SaveX(ctx)
	d.db.Membership.Create().SetUserID(f.actor).SetOrganizationID(f.org).SaveX(ctx)
	d.db.Membership.Create().SetUserID(other.actor).SetOrganizationID(other.org).SaveX(ctx)
	repo := NewFinanceReportRepo(d)
	for _, keyword := range []string{"REPORT-QDYH", "青岛远航", "qingdaoyuanhang", "qdyh", "OceanReport", "报表分支", "baobiaofenzhi", "bbfz"} {
		result, err := repo.Options(ctx, f.org, biz.SelectorListOptions{Page: 1, PageSize: 20, Keyword: keyword}, false)
		if err != nil || result.Total != 1 || len(result.Items) != 1 || result.Items[0].ID != f.party {
			t.Fatalf("报表客商候选%s须覆盖中英代码别名拼音并限当前公司: %#v %v", keyword, result, err)
		}
	}
	for _, keyword := range []string{"报表人员", "baobiaorenyuan", "bbry"} {
		result, err := repo.Options(ctx, f.org, biz.SelectorListOptions{Page: 1, PageSize: 20, Keyword: keyword}, true)
		if err != nil || result.Total != 1 || len(result.Items) != 1 || result.Items[0].ID != f.actor {
			t.Fatalf("报表人员候选%s须保留拼音及公司成员范围: %#v %v", keyword, result, err)
		}
	}
}

func TestFinanceReportNetAmountMatchesDomainDivisionPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	for _, direction := range []feeent.Direction{feeent.DirectionRECEIVABLE, feeent.DirectionPAYABLE} {
		f := newReportFixture(t, d)
		// 商在八位半偶中点上方、与中点相差不足16位除法的一个舍入单位。
		// 域先Div(16)再RoundBank(8)，SQL若直接舍入精确商会得到6.00000001。
		fee := f.fee(f.order.ID, direction, "1000000000.00000001", "600000000.50000001")
		d.db.OrderFee.UpdateOneID(fee.ID).SetExchangeRate("0.00000001").SetBaseCurrencyAmount("10.00000000").SaveX(ctx)
		report, err := NewFinanceReportRepo(d).Profit(ctx, f.org, biz.FinanceReportFilter{Page: 1, PageSize: 20}, false)
		if err != nil {
			t.Fatal(err)
		}
		snapshot := &biz.OrderFee{TotalAmount: decimal.RequireFromString(fee.TotalAmount), NetAmount: decimal.RequireFromString(fee.NetAmount), BaseCurrencyAmount: decimal.NewFromInt(10)}
		want := snapshot.BaseCurrencyNetAmount()
		if !want.Equal(decimal.NewFromInt(6)) {
			t.Fatalf("领域夹具应命中八位半偶中点：%s", want)
		}
		if direction == feeent.DirectionPAYABLE {
			want = want.Neg()
		}
		if len(report.Items) != 1 || !report.Items[0].NetProfitAmount.Equal(want) || !report.Totals[0].NetProfitAmount.Equal(want) || !report.Months[0].Amounts[0].NetProfitAmount.Equal(want) {
			t.Fatalf("%s利润行、合计和月图必须与费用域的中间除法精度一致：域=%s，报表=%#v", direction, want, report)
		}
	}
}
