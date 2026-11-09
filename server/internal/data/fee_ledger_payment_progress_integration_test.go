package data

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	invoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	invoicelinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	nettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	verificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	verificationallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
)

// 付款进度要求发生真实核销；对冲只减少待付余额，开票状态单独取有效已开票事实。
// 每个阶段同时校验 SQL 筛选、行投影、订单详情及金额，防止三套口径再次偏离。
func TestFeeLedgerPaymentProgressPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	for _, tc := range []struct {
		name     string
		status   invoiceent.Status
		inactive bool
		invoiced bool
	}{
		{name: "未建发票"},
		{name: "草稿发票", status: invoiceent.StatusDRAFT},
		{name: "有效已开票", status: invoiceent.StatusISSUED, invoiced: true},
		{name: "已取消发票", status: invoiceent.StatusCANCELLED},
		{name: "已红冲发票", status: invoiceent.StatusRED_FLUSHED},
		{name: "已开票但关联停用", status: invoiceent.StatusISSUED, inactive: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFeeLedgerPostgresFixture(t, d)
			orgs := []uuid.UUID{f.organizationID}
			feeID := f.createLedgerFee("payment", "300")
			bill := f.createConfirmedBill("payment", "300", financeBillIntegrationDate)
			f.linkBillFee("payment", bill, feeID, "300")
			if tc.status != "" {
				invoiceFixture := &financeInvoicePostgresFixture{t: t, data: d, organizationID: f.organizationID, partnerID: f.partnerID, orderID: f.orderID, actorID: f.actorID, suffix: f.suffix}
				invoice := invoiceFixture.createInvoiceWithStatus("payment", tc.status, bill)
				if tc.inactive {
					d.db.FinanceInvoiceBill.Update().Where(invoicelinkent.InvoiceIDEQ(invoice.ID)).SetActive(false).SaveX(ctx)
				}
				t.Cleanup(func() {
					if _, err := d.db.FinanceInvoiceBill.Delete().Where(invoicelinkent.InvoiceIDEQ(invoice.ID)).Exec(ctx); err != nil {
						t.Error(err)
					}
					if err := d.db.FinanceInvoice.DeleteOneID(invoice.ID).Exec(ctx); err != nil {
						t.Error(err)
					}
				})
			}
			unpaid, partial, paid := biz.FeeLedgerUnverifiedUninvoiced, biz.FeeLedgerPartiallyVerifiedUninvoiced, biz.FeeLedgerVerifiedUninvoiced
			if tc.invoiced {
				unpaid, partial, paid = biz.FeeLedgerInvoicedUnverified, biz.FeeLedgerInvoicedPartiallyVerified, biz.FeeLedgerCompleted
			}
			repo := NewSettlementRepo(d)
			check := func(stage string, progress biz.FeeLedgerFinancialProgress, verified, netted, remaining string) {
				t.Helper()
				checkItem := func(scope string, item *biz.FeeLedgerItem) {
					if item == nil || item.BillNo != bill.BillNo || item.FinancialProgress != progress || item.VerifiedAmount.String() != verified || item.NettedAmount.String() != netted || item.UnverifiedAmount.String() != remaining {
						t.Fatalf("%s/%s 进度或金额错误: %+v，期望进度=%s 核销=%s 对冲=%s 余额=%s", stage, scope, item, progress, verified, netted, remaining)
					}
				}
				for _, filterProgress := range []biz.FeeLedgerFinancialProgress{biz.FeeLedgerUnbilled, biz.FeeLedgerUnverifiedUninvoiced, biz.FeeLedgerInvoicedUnverified, biz.FeeLedgerInvoicedPartiallyVerified, biz.FeeLedgerPartiallyVerifiedUninvoiced, biz.FeeLedgerVerifiedUninvoiced, biz.FeeLedgerCompleted} {
					result := listFeeLedgerByProgress(t, repo, orgs, filterProgress)
					var want int64
					if filterProgress == progress {
						want = 1
					}
					if result.Total != want || result.MatchedSummary.Count != result.Total || int64(len(result.Items)) != want {
						t.Fatalf("%s 筛选%s应有%d条: total=%d matched=%d items=%d", stage, filterProgress, want, result.Total, result.MatchedSummary.Count, len(result.Items))
					}
					if want == 1 {
						checkItem("筛选列表", findFeeLedgerItem(result.Items, feeID))
					}
				}
				result, err := repo.ListFeeLedger(ctx, orgs, biz.FeeLedgerFilter{Page: 1, PageSize: 1})
				if err != nil {
					t.Fatal(err)
				}
				if result.Total != 1 || result.MatchedSummary.Count != result.Total {
					t.Fatalf("%s 未筛选列表统计错误: %+v", stage, result)
				}
				checkItem("未筛选列表", findFeeLedgerItem(result.Items, feeID))
				detail, err := repo.GetFeeLedgerOrderDetail(ctx, orgs, f.orderID)
				if err != nil {
					t.Fatal(err)
				}
				checkItem("订单详情", findFeeLedgerItem(detail.Items, feeID))
			}
			createPayment := func(key string, amount string) *ent.FinanceVerification {
				cashflow := f.createConfirmedCashflow(key, amount, financeBillIntegrationDate)
				return f.createActiveVerification(key, amount, func(id uuid.UUID) []*ent.FinanceVerificationAllocationCreate {
					return []*ent.FinanceVerificationAllocationCreate{f.verificationAllocationCreate(id, cashflow, bill, amount)}
				})
			}
			reversePayment := func(payment *ent.FinanceVerification) {
				d.db.FinanceVerification.UpdateOneID(payment.ID).SetStatus(verificationent.StatusREVERSED).SetReversedAt(time.Now()).SetReversedBy(f.actorID).SetReversalReason("验证反核销后的付款进度").SaveX(ctx)
				d.db.FinanceVerificationAllocation.Update().Where(verificationallocationent.VerificationIDEQ(payment.ID)).SetActive(false).SaveX(ctx)
			}
			reverseNetting := func(netting *ent.FinanceNetting, allocation *ent.FinanceNettingAllocation) {
				d.db.FinanceNetting.UpdateOneID(netting.ID).SetStatus(nettingent.StatusREVERSED).SetReversedAt(time.Now()).SetReversedBy(f.actorID).SetReversalReason("验证反对冲后的付款进度").SaveX(ctx)
				d.db.FinanceNettingAllocation.UpdateOneID(allocation.ID).SetActive(false).SaveX(ctx)
			}

			check("尚未结算", unpaid, "0", "0", "300")
			netting, allocation := f.createNetting("payment-partial", nettingent.StatusCONFIRMED, bill, "200", true)
			check("纯部分对冲仍未付款", unpaid, "0", "200", "100")
			extraNetting, extraAllocation := f.createNetting("payment-full", nettingent.StatusCONFIRMED, bill, "100", true)
			check("纯全额对冲仍未付款", unpaid, "0", "300", "0")
			reverseNetting(extraNetting, extraAllocation)
			check("撤掉部分对冲仍未付款", unpaid, "0", "200", "100")
			first := createPayment("payment-first", "50")
			check("对冲后付一半为部分付款", partial, "50", "200", "50")
			second := createPayment("payment-second", "50")
			check("实际付完对冲后的余额为已付款", paid, "100", "200", "0")
			reversePayment(second)
			check("反核销部分回到部分付款", partial, "50", "200", "50")
			reversePayment(first)
			check("反掉全部实际核销但保留对冲仍未付款", unpaid, "0", "200", "100")
			remainingPayment := createPayment("payment-remaining", "100")
			check("再次付清对冲后余额", paid, "100", "200", "0")
			reverseNetting(netting, allocation)
			check("反对冲按恢复余额判为部分付款", partial, "100", "0", "200")
			reversePayment(remainingPayment)
			check("无实际核销回到未付款", unpaid, "0", "0", "300")
			createPayment("payment-cash-full", "300")
			check("纯现金全额付款", paid, "300", "0", "0")
		})
	}
}
