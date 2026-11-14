package data

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	cash "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	netting "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	allocation "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	verification "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
)

func (f *reportFixture) verification(bill *ent.FinanceBill, amount string) (*ent.FinanceVerification, *ent.FinanceVerificationAllocation) {
	ctx := context.Background()
	flow := f.cashflow(cash.DirectionRECEIVABLE, cash.StatusCONFIRMED, "2026-02-01", amount)
	v := f.d.db.FinanceVerification.Create().SetOrganizationID(f.org).SetVerificationNo("V-" + uuid.NewString()[:12]).SetIdempotencyKey(uuid.NewString()).SetDirection(verification.DirectionRECEIVABLE).SetSettlementPartyID(f.party).SetSettlementPartyName("报表结算单位").SetCurrency("CNY").SetAmount(amount).SetBaseCurrency("CNY").SetBaseAmount(amount).SetBillBaseAmount(amount).SetCashflowBaseAmount(amount).SetExchangeGainLoss("0").SetVerificationDate("2026-02-01").SetVersion(1).SaveX(ctx)
	a := f.d.db.FinanceVerificationAllocation.Create().SetVerificationID(v.ID).SetCashflowID(flow.ID).SetBillID(bill.ID).SetCashflowNo(flow.FlowNo).SetBillNo(bill.BillNo).SetAmount(amount).SetBillBaseAmount(amount).SetCashflowBaseAmount(amount).SetExchangeGainLoss("0").SetActive(true).SaveX(ctx)
	return v, a
}
func (f *reportFixture) netting(bill *ent.FinanceBill, amount string, confirmed bool) (*ent.FinanceNetting, *ent.FinanceNettingAllocation) {
	ctx := context.Background()
	builder := f.d.db.FinanceNetting.Create().SetOrganizationID(f.org).SetNettingNo("N-" + uuid.NewString()[:12]).SetIdempotencyKey(uuid.NewString()).SetRequestHash(strings.Repeat("a", 64)).SetSettlementPartyID(f.party).SetSettlementPartyName("报表结算单位").SetCurrency("CNY").SetAmount(amount).SetBaseCurrency("CNY").SetBaseCurrencyAmount(amount).SetPayableBaseAmount(amount).SetExchangeGainLoss("0").SetVersion(1)
	if confirmed {
		builder.SetStatus(netting.StatusCONFIRMED).SetConfirmedAt(time.Now()).SetConfirmedBy(f.actor)
	}
	n := builder.SaveX(ctx)
	a := f.d.db.FinanceNettingAllocation.Create().SetNettingID(n.ID).SetBillID(bill.ID).SetBillNo(bill.BillNo).SetDirection(allocation.DirectionRECEIVABLE).SetAmount(amount).SetBaseCurrencyAmount(amount).SetActive(confirmed).SaveX(ctx)
	return n, a
}

func TestFinanceReportCurrentSettlementPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	f := newReportFixture(t, d)
	due := "2026-10-01"
	bill := f.bill(&due, "1000")
	v1, a1 := f.verification(bill, "100")
	f.verification(bill, "200")
	n1, na1 := f.netting(bill, "50", true)
	f.netting(bill, "100", true)
	f.netting(bill, "20", false)
	repo := NewFinanceReportRepo(d)
	now := time.Date(2026, 11, 14, 0, 0, 0, 0, biz.ExchangeRateBusinessLocation())
	filter := biz.FinanceReportFilter{Page: 1, PageSize: 20}
	aging, err := repo.Aging(ctx, f.org, filter, false, now)
	if err != nil {
		t.Fatal(err)
	}
	if aging.Items[0].Balance.String() != "550" || aging.Items[0].SettledAmount.String() != "450" || aging.NativeTotals[0].Days3160.String() != "550" {
		t.Fatalf("多核销×多对冲不得乘放大金额，草稿对冲不生效: %#v", aging)
	}
	cashReport, err := repo.Cashflow(ctx, f.org, filter, false)
	if err != nil {
		t.Fatal(err)
	}
	if cashReport.NativeTotals[0].ReceivableAmount.String() != "300" {
		t.Fatal("实际收款只计确认流水")
	}
	d.db.FinanceVerification.UpdateOneID(v1.ID).SetStatus(verification.StatusREVERSED).SetReversedAt(time.Now()).SetReversedBy(f.actor).SetReversalReason("测试反核销").SaveX(ctx)
	d.db.FinanceVerificationAllocation.UpdateOneID(a1.ID).SetActive(false).SaveX(ctx)
	d.db.FinanceNetting.UpdateOneID(n1.ID).SetStatus(netting.StatusREVERSED).SetReversedAt(time.Now()).SetReversedBy(f.actor).SetReversalReason("测试反对冲").SaveX(ctx)
	d.db.FinanceNettingAllocation.UpdateOneID(na1.ID).SetActive(false).SaveX(ctx)
	aging, err = repo.Aging(ctx, f.org, filter, false, now)
	if err != nil || aging.Items[0].Balance.String() != "700" {
		t.Fatalf("撤销后恢复真实账单余额: %#v %v", aging, err)
	}
	cashReport, err = repo.Cashflow(ctx, f.org, filter, false)
	if err != nil || cashReport.NativeTotals[0].ReceivableAmount.String() != "300" {
		t.Fatalf("反核销不减少实际收款: %#v %v", cashReport, err)
	}
}

func TestFinanceReportReadSnapshotPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	f := newReportFixture(t, d)
	flow := f.cashflow(cash.DirectionRECEIVABLE, cash.StatusCONFIRMED, "2026-02-01", "1000")
	repo := NewFinanceReportRepo(d)
	filter := biz.FinanceReportFilter{Page: 1, PageSize: 20}
	// 报表读取可合并外层一致快照。首次真实SELECT建立快照后，另一连接提交取消；
	// 后续行、完整合计仍须沿相同快照，不能经池直连到刚提交的新事实。
	err := d.withReadSnapshot(ctx, func(snapshotCtx context.Context) error {
		client, err := d.client(snapshotCtx)
		if err != nil {
			return err
		}
		if _, err := financeReportScan[reportCountRow](snapshotCtx, client, "SELECT COUNT(*) AS total FROM finance_cashflows WHERE organization_id=$1", []any{f.org}); err != nil {
			return err
		}
		if _, err := d.db.FinanceCashflow.UpdateOneID(flow.ID).SetStatus(cash.StatusCANCELLED).SetCancelledAt(time.Now()).SetCancelledBy(f.actor).SetCancellationReason("快照并发取消").Save(ctx); err != nil {
			return err
		}
		report, err := repo.Cashflow(snapshotCtx, f.org, filter, false)
		if err != nil {
			return err
		}
		if report.Total != 1 || len(report.Items) != 1 || report.BaseTotals[0].ReceivableAmount.String() != "1000" {
			t.Fatalf("同一快照不得混入取消后的新事实: %#v", report)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	latest, err := repo.Cashflow(ctx, f.org, filter, false)
	if err != nil || latest.Total != 0 || len(latest.Items) != 0 || len(latest.BaseTotals) != 0 {
		t.Fatalf("下一次查询须读取新事实: %#v %v", latest, err)
	}
}
