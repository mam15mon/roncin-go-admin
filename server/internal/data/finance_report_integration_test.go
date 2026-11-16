package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	cash "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	personnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
	"github.com/shopspring/decimal"
)

type reportFixture struct {
	t                 *testing.T
	d                 *Data
	org, party, actor uuid.UUID
	order             *ent.Order
}

func newReportFixture(t *testing.T, d *Data) *reportFixture {
	t.Helper()
	ctx := context.Background()
	suffix := uuid.NewString()[:8]
	org := d.db.Organization.Create().SetCode("REP-" + suffix).SetName("报表测试公司" + suffix).SetKind("company").SetBaseCurrency("CNY").SaveX(ctx)
	party := d.db.Partner.Create().SetOrganizationID(org.ID).SetCode("P-" + suffix).SetLegalName("报表客户" + suffix).SetNormalizedName("报表客户" + suffix).SaveX(ctx)
	actor := d.db.User.Create().SetDisplayName("测试销售" + suffix).SaveX(ctx)
	order := d.db.Order.Create().SetIdempotencyKey(uuid.NewString()).SetOrganizationID(org.ID).SetOrderNo("SE-" + suffix).SetCustomerID(party.ID).SetBusinessType(orderent.BusinessTypeSE).SetTradeDirection(orderent.TradeDirectionExport).SetPaymentTerm(orderent.PaymentTermPREPAID).SetEtd("2026-01-31T17:00:00Z").SaveX(ctx)
	return &reportFixture{t, d, org.ID, party.ID, actor.ID, order}
}
func (f *reportFixture) fee(orderID uuid.UUID, direction feeent.Direction, total, net string, effects ...feeent.EconomicEffect) *ent.OrderFee {
	f.t.Helper()
	effect := feeent.EconomicEffectINCREASE
	if len(effects) > 0 {
		effect = effects[0]
	}
	gross := decimal.RequireFromString(total)
	tax := gross.Sub(decimal.RequireFromString(net))
	return f.d.db.OrderFee.Create().SetOrderID(orderID).SetIdempotencyKey(uuid.NewString()).SetEconomicEffect(effect).SetDirection(direction).SetFeeCode("OCEAN").SetFeeName("海运费").SetSettlementPartyID(f.party).SetBillingUnit("票").SetQuantity("1").SetUnitPrice(total).SetTotalAmount(total).SetNetAmount(net).SetTaxAmount(tax.String()).SetTaxRate("6").SetCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(feeent.ExchangeRateSourceSYSTEM).SetExchangeRateDate("2026-02-01").SetBaseCurrency("CNY").SetBaseCurrencyAmount(total).SetAccountingRequired(true).SetVersion(1).SaveX(context.Background())
}
func (f *reportFixture) cashflow(direction cash.Direction, status cash.Status, date, amount string) *ent.FinanceCashflow {
	builder := f.d.db.FinanceCashflow.Create().SetOrganizationID(f.org).SetFlowNo("FLOW-" + uuid.NewString()[:12]).SetIdempotencyKey(uuid.NewString()).SetDirection(direction).SetStatus(status).SetSettlementPartyID(f.party).SetSettlementPartyName("报表结算单位").SetCurrency("CNY").SetAmount(amount).SetExchangeRate("1").SetExchangeRateSource(cash.ExchangeRateSourceSYSTEM).SetExchangeRateDate(date).SetBaseCurrency("CNY").SetBaseAmount(amount).SetTransactionDate(date).SetOurAccount("测试账户").SetPaymentMethod("银行转账").SetVersion(1)
	if status == cash.StatusCANCELLED {
		builder.SetCancelledAt(time.Now()).SetCancelledBy(f.actor).SetCancellationReason("测试取消")
	}
	return builder.SaveX(context.Background())
}
func (f *reportFixture) bill(due *string, total string) *ent.FinanceBill {
	return withTestFinanceBillSettlementAccountSnapshot(f.d.db.FinanceBill.Create(), uuid.New(), "CNY").SetOrganizationID(f.org).SetBillNo("BILL-" + uuid.NewString()[:12]).SetIdempotencyKey(uuid.NewString()).SetDirection(billent.DirectionRECEIVABLE).SetStatus(billent.StatusCONFIRMED).SetConfirmedAt(time.Now()).SetConfirmedBy(f.actor).SetSettlementPartyID(f.party).SetSettlementPartyName("报表结算单位").SetCurrency("CNY").SetBaseCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(billent.ExchangeRateSourceSYSTEM).SetExchangeRateDate("2026-02-01").SetTotalAmount(total).SetNetAmount(total).SetTaxAmount("0").SetBaseCurrencyAmount(total).SetFeeCount(1).SetBillDate("2026-02-01").SetNillableDueDate(due).SetVersion(1).SaveX(context.Background())
}

func TestFinanceReportProfitPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	f := newReportFixture(t, d)
	other := newReportFixture(t, d)
	ar := f.fee(f.order.ID, feeent.DirectionRECEIVABLE, "10600", "10000")
	f.fee(f.order.ID, feeent.DirectionPAYABLE, "6360", "6000")
	other.fee(other.order.ID, feeent.DirectionRECEIVABLE, "999999", "999999")
	// 同一人兼任两个角色，EXISTS筛选不能令费用金额翻倍。
	for _, role := range []string{"SALES", "OPERATOR"} {
		d.db.OrderPersonnel.Create().SetOrderID(f.order.ID).SetOrganizationID(f.org).SetUserID(f.actor).SetRole(personnelent.Role(role)).SaveX(ctx)
	}
	repo := NewFinanceReportRepo(d)
	filters := biz.FinanceReportFilter{Page: 1, PageSize: 1, DateFrom: "2026-02-01", DateTo: "2026-04-30", SalesID: &f.actor, OperatorID: &f.actor}
	report, err := repo.Profit(ctx, f.org, filters, false)
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 1 || report.OrderCount != 1 || report.FeeCount != 2 || len(report.Items) != 1 || len(report.Totals) != 1 {
		t.Fatalf("范围/计数不正确: %#v", report)
	}
	total := report.Totals[0]
	if total.ProfitAmount.String() != "4240" || total.NetProfitAmount.String() != "4000" {
		t.Fatalf("含税/未税利润错误: %#v", total)
	}
	if report.Items[0].BusinessDate == nil || *report.Items[0].BusinessDate != "2026-02-01" || len(report.Months) != 3 || report.Months[1].Month != "2026-03" || !report.Months[1].Amounts[0].ProfitAmount.IsZero() {
		t.Fatalf("业务时区/月序列缺零月: %#v", report.Months)
	}
	exported, err := repo.Profit(ctx, f.org, filters, true)
	if err != nil || exported.Totals[0].ProfitAmount.String() != "4240" {
		t.Fatalf("同源导出失败: %#v %v", exported, err)
	}
	// 缺日期不回退到创建时间，且不应因完整CTE中日期参数与缺日期统计并用而发生绑定错误。
	undated := d.db.Order.Create().SetOrganizationID(f.org).SetIdempotencyKey(uuid.NewString()).SetOrderNo("UNDATED").SetCustomerID(f.party).SetBusinessType(orderent.BusinessTypeSI).SetFlowStatus(orderent.FlowStatusIMPORT_NOT_APPLICABLE).SetTradeDirection(orderent.TradeDirectionImport).SetPaymentTerm(orderent.PaymentTermPREPAID).SaveX(ctx)
	f.fee(undated.ID, feeent.DirectionRECEIVABLE, "100", "100")
	filters.SalesID, filters.OperatorID = nil, nil
	report, err = repo.Profit(ctx, f.org, filters, false)
	if err != nil || report.UndatedOrderCount != 1 || report.Total != 1 {
		t.Fatalf("缺日期统计与日期筛选错误: %#v %v", report, err)
	}
	filters.DateFrom, filters.DateTo = "", ""
	report, err = repo.Profit(ctx, f.org, filters, false)
	if err != nil || report.Total != 2 || len(report.Items) != 1 || report.Totals[0].ReceivableAmount.String() != "10700" {
		t.Fatalf("分页不得代替完整合计: %#v %v", report, err)
	}
	// 已替代的旧经营叶子与反向清偿不再次进入利润。
	request := d.db.OrderFeeCorrectionRequest.Create().SetOrganizationID(f.org).SetOrderID(f.order.ID).SetOperation("REPLACE").SetSourceFeeID(ar.ID).SetSourceFeeVersion(1).SetBeforeSnapshot(json.RawMessage(`{}`)).SetAfterSnapshot(json.RawMessage(`{}`)).SetApprovedFeeContentVersion(1).SetApprovedFeeReviewVersion(1).SetReason("测试当前叶子").SetIdempotencyKey(uuid.NewString()).SetRequestFingerprint(strings.Repeat("a", 64)).SetRequestedBy(f.actor).SetRequestedAt(time.Now()).SaveX(ctx)
	d.db.OrderFee.UpdateOneID(ar.ID).SetSupersededByRequestID(request.ID).SaveX(ctx)
	f.fee(f.order.ID, feeent.DirectionRECEIVABLE, "5300", "5000")
	d.db.OrderFee.Create().SetOrderID(f.order.ID).SetIdempotencyKey(uuid.NewString()).SetEconomicEffect(feeent.EconomicEffectDECREASE).SetSourceFeeID(ar.ID).SetCorrectionRequestID(request.ID).SetDirection(feeent.DirectionRECEIVABLE).SetFeeCode("OCEAN").SetFeeName("海运费冲回").SetSettlementPartyID(f.party).SetBillingUnit("票").SetQuantity("1").SetUnitPrice("10600").SetTotalAmount("10600").SetNetAmount("10000").SetTaxAmount("600").SetTaxRate("6").SetCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(feeent.ExchangeRateSourceSYSTEM).SetExchangeRateDate("2026-02-01").SetBaseCurrency("CNY").SetBaseCurrencyAmount("10600").SetAccountingRequired(true).SetVersion(1).SaveX(ctx)
	report, err = repo.Profit(ctx, f.org, filters, false)
	if err != nil || report.Totals[0].ReceivableAmount.String() != "5400" {
		t.Fatalf("当前叶子应排除旧行及反向清偿: %#v %v", report, err)
	}
}

func TestFinanceReportCashflowPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	f := newReportFixture(t, d)
	other := newReportFixture(t, d)
	active := f.cashflow(cash.DirectionRECEIVABLE, cash.StatusCONFIRMED, "2026-02-01", "1000")
	f.cashflow(cash.DirectionPAYABLE, cash.StatusCONFIRMED, "2026-02-28", "300")
	f.cashflow(cash.DirectionRECEIVABLE, cash.StatusDRAFT, "2026-02-01", "900")
	f.cashflow(cash.DirectionRECEIVABLE, cash.StatusCANCELLED, "2026-02-01", "800")
	f.cashflow(cash.DirectionRECEIVABLE, cash.StatusCONFIRMED, "2026-03-01", "700")
	other.cashflow(cash.DirectionRECEIVABLE, cash.StatusCONFIRMED, "2026-02-01", "99999")
	repo := NewFinanceReportRepo(d)
	filter := biz.FinanceReportFilter{Page: 1, PageSize: 1, DateFrom: "2026-02-01", DateTo: "2026-02-28"}
	report, err := repo.Cashflow(ctx, f.org, filter, false)
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 2 || len(report.Items) != 1 || len(report.BaseTotals) != 1 || report.BaseTotals[0].NetAmount.String() != "700" {
		t.Fatalf("只计确认流水且完整筛选合计: %#v", report)
	}
	exported, err := repo.Cashflow(ctx, f.org, filter, true)
	if err != nil || len(exported.Items) != 2 {
		t.Fatalf("独立导出应返回全部筛选: %#v %v", exported, err)
	}
	d.db.FinanceCashflow.UpdateOneID(active.ID).SetStatus(cash.StatusCANCELLED).SetCancelledAt(time.Now()).SetCancelledBy(f.actor).SetCancellationReason("测试取消").SaveX(ctx)
	report, err = repo.Cashflow(ctx, f.org, filter, false)
	if err != nil || report.BaseTotals[0].NetAmount.String() != "-300" {
		t.Fatalf("取消流水后应剔除真实金额: %#v %v", report, err)
	}
}

func TestFinanceReportAgingPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	f := newReportFixture(t, d)
	other := newReportFixture(t, d)
	today := time.Date(2026, 11, 14, 0, 0, 0, 0, biz.ExchangeRateBusinessLocation())
	for _, days := range []int{-1, 0, 1, 30, 31, 60, 61, 90, 91} {
		due := today.AddDate(0, 0, -days).Format("2006-01-02")
		f.bill(&due, "100")
	}
	f.bill(nil, "100")
	other.bill(nil, "99999")
	repo := NewFinanceReportRepo(d)
	filter := biz.FinanceReportFilter{Page: 1, PageSize: 1}
	report, err := repo.Aging(ctx, f.org, filter, false, today)
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 10 || len(report.Items) != 1 || report.AsOfDate != "2026-11-14" {
		t.Fatalf("当前业务日/分页/范围错误: %#v", report)
	}
	b := report.NativeTotals[0]
	if b.NotDue.String() != "200" || b.Days130.String() != "200" || b.Days3160.String() != "200" || b.Days6190.String() != "200" || b.Over90.String() != "100" || b.Undated.String() != "100" || b.Total.String() != "1000" {
		t.Fatalf("账龄边界/缺日期桶不守恒: %#v", b)
	}
	filter.Bucket = "UNDATED"
	report, err = repo.Aging(ctx, f.org, filter, true, today)
	if err != nil || report.Total != 1 || report.Items[0].OverdueDays != nil {
		t.Fatalf("缺日期不伪装逾期0天: %#v %v", report, err)
	}
}

func TestFinanceReportBankRoundPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	for _, raw := range []string{"0.000000005", "0.000000015", "-0.000000005", "-0.000000015", "7.005000005", "123.456789015"} {
		rows, err := financeReportScan[struct{ Value decimal.Decimal }](ctx, d.db, "SELECT "+reportBankRound("$1::numeric")+" AS value", []any{raw})
		if err != nil {
			t.Fatal(err)
		}
		want := decimal.RequireFromString(raw).RoundBank(8)
		if !rows[0].Value.Equal(want) {
			t.Fatalf("%s数据库舍入=%s，领域=%s", raw, rows[0].Value, want)
		}
	}
}

func TestFinanceReportExportLimitPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	f := newReportFixture(t, d)
	// 使用正式表约束批量构造真实流水，验证独立导出契约，不能改分页上限或截断后冒充成功。
	for start := 0; start < 10001; start += 500 {
		size := 500
		if start+size > 10001 {
			size = 10001 - start
		}
		builders := make([]*ent.FinanceCashflowCreate, 0, size)
		for i := 0; i < size; i++ {
			builders = append(builders, d.db.FinanceCashflow.Create().SetOrganizationID(f.org).SetFlowNo(fmt.Sprintf("LIMIT-%d", start+i)).SetIdempotencyKey(uuid.NewString()).SetDirection(cash.DirectionRECEIVABLE).SetStatus(cash.StatusCONFIRMED).SetSettlementPartyID(f.party).SetSettlementPartyName("限额测试").SetCurrency("CNY").SetAmount("1").SetExchangeRate("1").SetExchangeRateSource(cash.ExchangeRateSourceSYSTEM).SetExchangeRateDate("2026-02-01").SetBaseCurrency("CNY").SetBaseAmount("1").SetTransactionDate("2026-02-01").SetOurAccount("账号").SetPaymentMethod("银行转账").SetVersion(1))
		}
		if _, err := d.db.FinanceCashflow.CreateBulk(builders...).Save(ctx); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewFinanceReportRepo(d)
	_, err := repo.Cashflow(ctx, f.org, biz.FinanceReportFilter{}, true)
	if !errors.Is(err, biz.ErrFinanceReportExportLimit) {
		t.Fatalf("超限应明确失败，实际: %v", err)
	}
	last, err := d.db.FinanceCashflow.Query().Where(cash.FlowNoEQ("LIMIT-10000")).Only(ctx)
	if err != nil {
		t.Fatal(err)
	}
	d.db.FinanceCashflow.UpdateOneID(last.ID).SetStatus(cash.StatusCANCELLED).SetCancelledAt(time.Now()).SetCancelledBy(f.actor).SetCancellationReason("测试取消").SaveX(ctx)
	report, err := repo.Cashflow(ctx, f.org, biz.FinanceReportFilter{}, true)
	if err != nil || len(report.Items) != 10000 {
		t.Fatalf("恰好限额应全量导出: count=%d err=%v", len(report.Items), err)
	}
}
