package data

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financeinvoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	financeinvoicebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	financenettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	financeverificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
)

// TestFeeHasProtectedFinanceFactPostgres 验证账单级受保护财务事实判定：
// 发票 DRAFT/ISSUED、核销 ACTIVE、对冲 CONFIRMED 才保护；取消/红冲发票、反转核销与对冲、
// 草稿对冲、已取消账单均不保护；事实落在账单上，同账单其他费用同样受保护。
func TestFeeHasProtectedFinanceFactPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	data, cleanupData := getIntegrationData(t)
	t.Cleanup(cleanupData)
	f := newFeeLedgerPostgresFixture(t, data)
	ctx := context.Background()

	var invoiceIDs []uuid.UUID
	// 注册在夹具清理之后，LIFO 先执行：发票及其账单关联引用账单，需先于账单删除。
	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := data.db.FinanceInvoiceBill.Delete().Where(financeinvoicebillent.InvoiceIDIn(invoiceIDs...)).Exec(cctx); err != nil {
			t.Errorf("清理发票账单关联: %v", err)
		}
		if _, err := data.db.FinanceInvoice.Delete().Where(financeinvoiceent.IDIn(invoiceIDs...)).Exec(cctx); err != nil {
			t.Errorf("清理发票: %v", err)
		}
	})

	newBilledFee := func(key string) (uuid.UUID, *ent.FinanceBill) {
		feeID := f.createLedgerFee(key, "100")
		bill := f.createConfirmedBill(key, "100", financeBillIntegrationDate)
		f.linkBillFee(key, bill, feeID, "100")
		return feeID, bill
	}
	addInvoice := func(key string, bill *ent.FinanceBill, status financeinvoiceent.Status, linkActive bool) {
		t.Helper()
		create := data.db.FinanceInvoice.Create().
			SetOrganizationID(f.organizationID).SetRecordNo("INV-" + key + "-" + f.suffix).SetIdempotencyKey("inv-" + key + "-" + f.suffix).
			SetDirection(financeinvoiceent.DirectionRECEIVABLE).SetInvoiceType(financeinvoiceent.InvoiceTypeNORMAL).
			SetSettlementPartyID(f.partnerID).SetSettlementPartyName("保护事实").SetCurrency("CNY").SetBaseCurrency("CNY").
			SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetBillCount(1)
		switch status {
		case financeinvoiceent.StatusISSUED:
			create.SetBaseCurrencyAmount("100").SetExchangeRate("1").SetExchangeRateSource(financeinvoiceent.ExchangeRateSourceSYSTEM).SetExchangeRateDate(financeBillIntegrationDate)
			create.SetStatus(status)
		case financeinvoiceent.StatusCANCELLED:
			create.SetStatus(status).SetCancelledAt(time.Now()).SetCancelledBy(f.actorID).SetCancellationReason("测试")
		case financeinvoiceent.StatusRED_FLUSHED:
			create.SetBaseCurrencyAmount("100").SetExchangeRate("1").SetExchangeRateSource(financeinvoiceent.ExchangeRateSourceSYSTEM).SetExchangeRateDate(financeBillIntegrationDate)
			create.SetStatus(status).SetRedInvoiceNo("RED-" + key).SetRedInvoiceDate(financeBillIntegrationDate).
				SetRedFlushedAt(time.Now()).SetRedFlushedBy(f.actorID).SetRedFlushReason("测试")
		}
		invoice, err := create.Save(ctx)
		if err != nil {
			t.Fatalf("创建发票 %s: %v", key, err)
		}
		invoiceIDs = append(invoiceIDs, invoice.ID)
		if _, err := data.db.FinanceInvoiceBill.Create().SetInvoiceID(invoice.ID).SetBillID(bill.ID).SetBillNo(bill.BillNo).
			SetAmount("100").SetTaxAmount("0").SetActive(linkActive).Save(ctx); err != nil {
			t.Fatalf("创建发票账单关联 %s: %v", key, err)
		}
	}
	protected := func(feeID uuid.UUID) bool {
		t.Helper()
		got, err := feeHasProtectedFinanceFact(ctx, data.db, feeID)
		if err != nil {
			t.Fatalf("判定失败: %v", err)
		}
		return got
	}
	expect := func(name string, feeID uuid.UUID, want bool) {
		t.Helper()
		if got := protected(feeID); got != want {
			t.Fatalf("%s: 受保护=%v，期望 %v", name, got, want)
		}
	}

	t.Run("已建账但无任何事实不受保护", func(t *testing.T) {
		feeID, _ := newBilledFee("none")
		expect("无事实", feeID, false)
	})
	t.Run("未建账费用不受保护", func(t *testing.T) {
		expect("未建账", f.createLedgerFee("unbilled", "100"), false)
	})
	t.Run("发票状态与关联", func(t *testing.T) {
		for _, tc := range []struct {
			key        string
			status     financeinvoiceent.Status
			linkActive bool
			want       bool
		}{
			{"inv-draft", financeinvoiceent.StatusDRAFT, true, true},
			{"inv-issued", financeinvoiceent.StatusISSUED, true, true},
			{"inv-cancelled", financeinvoiceent.StatusCANCELLED, true, false},
			{"inv-red", financeinvoiceent.StatusRED_FLUSHED, true, false},
			{"inv-released", financeinvoiceent.StatusDRAFT, false, false},
		} {
			feeID, bill := newBilledFee(tc.key)
			addInvoice(tc.key, bill, tc.status, tc.linkActive)
			expect(tc.key, feeID, tc.want)
		}
	})
	t.Run("核销状态", func(t *testing.T) {
		feeID, bill := newBilledFee("ver-active")
		cash := f.createConfirmedCashflow("ver-active", "100", financeBillIntegrationDate)
		f.createActiveVerification("ver-active", "100", func(id uuid.UUID) []*ent.FinanceVerificationAllocationCreate {
			return []*ent.FinanceVerificationAllocationCreate{f.verificationAllocationCreate(id, cash, bill, "100")}
		})
		expect("有效核销", feeID, true)

		feeID2, bill2 := newBilledFee("ver-reversed")
		cash2 := f.createConfirmedCashflow("ver-reversed", "100", financeBillIntegrationDate)
		v := f.createActiveVerification("ver-reversed", "100", func(id uuid.UUID) []*ent.FinanceVerificationAllocationCreate {
			return []*ent.FinanceVerificationAllocationCreate{f.verificationAllocationCreate(id, cash2, bill2, "100")}
		})
		// 反转：核销单置 REVERSED，分摊失效。
		v.Update().SetStatus(financeverificationent.StatusREVERSED).SetReversedAt(time.Now()).SetReversedBy(f.actorID).SetReversalReason("测试").ExecX(ctx)
		data.db.FinanceVerificationAllocation.Update().SetActive(false).ExecX(ctx)
		expect("已反核销", feeID2, false)
		// 单独校验父单已反转但分摊仍活动也不保护（父单状态是判定条件）。
		data.db.FinanceVerificationAllocation.Update().SetActive(true).ExecX(ctx)
		expect("核销单已反转", feeID2, false)
		expect("有效核销仍受保护", feeID, true)
	})
	t.Run("对冲状态", func(t *testing.T) {
		feeID, bill := newBilledFee("net-confirmed")
		f.createNetting("net-confirmed", financenettingent.StatusCONFIRMED, bill, "100", true)
		expect("已确认对冲", feeID, true)

		feeID2, bill2 := newBilledFee("net-draft")
		f.createNetting("net-draft", financenettingent.StatusDRAFT, bill2, "100", false)
		expect("草稿对冲", feeID2, false)

		feeID3, bill3 := newBilledFee("net-reversed")
		f.createNetting("net-reversed", financenettingent.StatusREVERSED, bill3, "100", false)
		expect("已反转对冲", feeID3, false)
	})
	t.Run("账单级判定与已取消账单", func(t *testing.T) {
		feeA, bill := newBilledFee("bill-level")
		feeB := f.createLedgerFee("bill-level-b", "50")
		f.linkBillFee("bill-level-b", bill, feeB, "50")
		addInvoice("bill-level", bill, financeinvoiceent.StatusISSUED, true)
		expect("同账单费用 A", feeA, true)
		expect("同账单费用 B", feeB, true)

		feeC, billC := newBilledFee("cancelled-bill")
		addInvoice("cancelled-bill", billC, financeinvoiceent.StatusDRAFT, true)
		billC.Update().SetStatus(financebillent.StatusCANCELLED).SetCancelledAt(time.Now()).SetCancelledBy(f.actorID).SetCancellationReason("测试").ExecX(ctx)
		expect("已取消账单", feeC, false)
	})
}
