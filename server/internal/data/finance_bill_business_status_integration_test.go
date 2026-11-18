package data

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financeinvoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	financeinvoicebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	financenettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
)

// addTestBillInvoice 构造已有合法登记事实，供非发票领域的核销测试使用，不跳过开票准入。
func addTestBillInvoice(t *testing.T, data *Data, bill *ent.FinanceBill, amount string, actorID uuid.UUID) *ent.FinanceInvoice {
	t.Helper()
	ctx := context.Background()
	invoice, err := data.db.FinanceInvoice.Create().SetOrganizationID(bill.OrganizationID).
		SetRecordNo("INV-READY-" + uuid.NewString()).SetIdempotencyKey(uuid.NewString()).
		SetDirection(financeinvoiceent.Direction(bill.Direction)).SetInvoiceType(financeinvoiceent.InvoiceTypeDIGITAL_NORMAL).
		SetSettlementPartyID(bill.SettlementPartyID).SetSettlementPartyName(bill.SettlementPartyName).
		SetCurrency(bill.Currency).SetBaseCurrency(bill.BaseCurrency).SetTotalAmount(amount).SetNetAmount(amount).SetTaxAmount("0").
		SetBillCount(1).SetStatus(financeinvoiceent.StatusISSUED).
		SetTaxInvoiceNo("READY-" + uuid.NewString()).SetInvoiceDate(bill.BillDate).SetIssuedAt(time.Now()).SetIssuedBy(actorID).
		SetInvoiceTitle(bill.SettlementPartyName).SetExchangeRate(bill.ExchangeRate).SetExchangeRateSource(financeinvoiceent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(bill.BillDate).SetBaseCurrencyAmount(decimal.RequireFromString(amount).Mul(decimal.RequireFromString(bill.ExchangeRate)).Round(8).StringFixed(8)).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := data.db.FinanceInvoiceBill.Create().SetInvoiceID(invoice.ID).SetBillID(bill.ID).SetBillNo(bill.BillNo).
		SetAmount(amount).SetTaxAmount("0").SetActive(true).Save(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := data.db.FinanceInvoiceBill.Delete().Where(financeinvoicebillent.InvoiceIDEQ(invoice.ID)).Exec(ctx); err != nil {
			t.Error(err)
		}
		if err := data.db.FinanceInvoice.DeleteOneID(invoice.ID).Exec(ctx); err != nil {
			t.Error(err)
		}
	})
	return invoice
}

// addTestBillInvoiceExemption 为专注结算与更正的夹具补齐真实免票决定，不改变被测账单版本。
func addTestBillInvoiceExemption(t *testing.T, data *Data, bill *ent.FinanceBill, actorID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	decision, err := data.db.FinanceInvoiceExemption.Create().SetOrganizationID(bill.OrganizationID).
		SetBillID(bill.ID).SetMarkedBy(actorID).SetReason("测试结算前置：本账单明确无需发票").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := data.db.FinanceInvoiceExemption.DeleteOneID(decision.ID).Exec(ctx); err != nil {
			t.Error(err)
		}
	})
}

func TestFinanceBillBusinessStatusFilterPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	t.Run("五态投影与多选分页汇总同源", func(t *testing.T) {
		f := newFeeLedgerPostgresFixture(t, data)
		other := newFeeLedgerPostgresFixture(t, data)
		ctx := context.Background()
		repo := NewFinanceBillRepo(data)
		bills := map[string]*ent.FinanceBill{}
		newBill := func(key string) *ent.FinanceBill {
			b := f.createConfirmedBill(key, "100", financeBillIntegrationDate)
			f.linkBillFee(key, b, f.createLedgerFee(key, "100"), "100")
			bills[key] = b
			return b
		}
		verify := func(key string, bill *ent.FinanceBill, amount string) {
			cash := f.createConfirmedCashflow(key, amount, financeBillIntegrationDate)
			f.createActiveVerification(key, amount, func(id uuid.UUID) []*ent.FinanceVerificationAllocationCreate {
				return []*ent.FinanceVerificationAllocationCreate{f.verificationAllocationCreate(id, cash, bill, amount)}
			})
		}
		newBill("none")
		paidWithoutInvoice := newBill("historical-paid-uninvoiced")
		verify("historical-paid-uninvoiced", paidWithoutInvoice, "100")
		partial := newBill("partial-invoice")
		addTestBillInvoice(t, data, partial, "30", f.actorID)
		verify("partial-invoice", partial, "20")
		invoiceFull := newBill("invoice-full")
		addTestBillInvoice(t, data, invoiceFull, "100", f.actorID)
		pureNetting := newBill("pure-netting")
		addTestBillInvoice(t, data, pureNetting, "100", f.actorID)
		f.createNetting("pure-netting", financenettingent.StatusCONFIRMED, pureNetting, "20", true)
		partlyVerified := newBill("partly-verified")
		addTestBillInvoice(t, data, partlyVerified, "100", f.actorID)
		verify("partly-verified", partlyVerified, "20")
		done := newBill("done")
		addTestBillInvoice(t, data, done, "100", f.actorID)
		verify("done", done, "40")
		f.createNetting("done", financenettingent.StatusCONFIRMED, done, "60", true)
		exempt := newBill("exempt")
		exemption, err := data.db.FinanceInvoiceExemption.Create().SetOrganizationID(f.organizationID).
			SetBillID(exempt.ID).SetMarkedBy(f.actorID).SetReason("已批准免票").Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := data.db.FinanceInvoiceExemption.DeleteOneID(exemption.ID).Exec(ctx); err != nil {
				t.Error(err)
			}
		})
		exemptDone := newBill("exempt-done")
		addTestBillInvoiceExemption(t, data, exemptDone, f.actorID)
		verify("exempt-done", exemptDone, "100")
		cancelled := newBill("cancelled")
		if err := cancelled.Update().SetStatus(financebillent.StatusCANCELLED).SetCancelledAt(time.Now()).SetCancelledBy(f.actorID).SetCancellationReason("测试取消").Exec(ctx); err != nil {
			t.Fatal(err)
		}
		other.createConfirmedBill("outside", "999", financeBillIntegrationDate)

		wantByKey := map[string]biz.FinanceBillBusinessStatus{
			"none": biz.FinanceBillBusinessUninvoiced, "historical-paid-uninvoiced": biz.FinanceBillBusinessUninvoiced,
			"partial-invoice": biz.FinanceBillBusinessPartiallyInvoiced, "invoice-full": biz.FinanceBillBusinessUnverified,
			"pure-netting": biz.FinanceBillBusinessUnverified, "partly-verified": biz.FinanceBillBusinessPartiallyVerified,
			"done": biz.FinanceBillBusinessCompleted, "exempt": biz.FinanceBillBusinessUnverified, "exempt-done": biz.FinanceBillBusinessCompleted, "cancelled": "",
		}
		for key, bill := range bills {
			got, err := repo.Get(ctx, []uuid.UUID{f.organizationID}, bill.ID)
			if err != nil || got.BusinessStatus != wantByKey[key] {
				t.Fatalf("%s投影=%+v err=%v", key, got, err)
			}
		}
		states := []biz.FinanceBillBusinessStatus{biz.FinanceBillBusinessUninvoiced, biz.FinanceBillBusinessPartiallyInvoiced, biz.FinanceBillBusinessUnverified, biz.FinanceBillBusinessPartiallyVerified, biz.FinanceBillBusinessCompleted}
		filters := [][]biz.FinanceBillBusinessStatus{{}, states, {states[0], states[1]}, {states[2], states[3]}, {states[0], states[0]}}
		for _, state := range states {
			filters = append(filters, []biz.FinanceBillBusinessStatus{state})
		}
		for _, selected := range filters {
			wanted := map[uuid.UUID]bool{}
			for key, status := range wantByKey {
				if status == "" {
					continue
				}
				keep := len(selected) == 0
				for _, state := range selected {
					keep = keep || state == status
				}
				if keep {
					wanted[bills[key].ID] = true
				}
			}
			seen := map[uuid.UUID]bool{}
			for page := 1; page <= len(wanted)+1; page++ {
				result, err := repo.List(ctx, []uuid.UUID{f.organizationID}, biz.FinanceBillFilter{Page: page, PageSize: 1, Status: biz.FinanceBillConfirmed, BusinessStatuses: selected})
				if err != nil {
					t.Fatal(err)
				}
				if result.Total != int64(len(wanted)) || result.MatchedSummary.Count != result.Total {
					t.Fatalf("筛选%v total=%d matched=%d want=%d", selected, result.Total, result.MatchedSummary.Count, len(wanted))
				}
				if len(result.MatchedSummary.Amounts) != 1 || !result.MatchedSummary.Amounts[0].Amount.Equal(decimal.NewFromInt(int64(len(wanted)*100))) {
					t.Fatalf("完整筛选金额与页码不一致: %+v", result.MatchedSummary)
				}
				for _, row := range result.Items {
					if !wanted[row.ID] || seen[row.ID] {
						t.Fatalf("跨页重复或越界:%s", row.ID)
					}
					seen[row.ID] = true
				}
			}
			if len(seen) != len(wanted) {
				t.Fatalf("筛选%v缺行", selected)
			}
		}
		result, err := repo.List(ctx, []uuid.UUID{f.organizationID}, biz.FinanceBillFilter{Page: 1, PageSize: 200, BusinessStatuses: []biz.FinanceBillBusinessStatus{biz.FinanceBillBusinessUninvoiced, biz.FinanceBillBusinessPartiallyInvoiced}, InvoiceProgress: biz.FinanceBillInvoicePartial})
		if err != nil || result.Total != 1 || result.Items[0].ID != partial.ID {
			t.Fatalf("多选与独立条件未取交集: %+v %v", result, err)
		}
		empty, err := repo.List(ctx, []uuid.UUID{f.organizationID}, biz.FinanceBillFilter{Page: 1, PageSize: 1,
			BusinessStatuses: []biz.FinanceBillBusinessStatus{biz.FinanceBillBusinessCompleted}, InvoiceProgress: biz.FinanceBillInvoiceNone})
		if err != nil || empty.Total != 0 || empty.MatchedSummary.Count != 0 || len(empty.Items) != 0 || len(empty.MatchedSummary.Amounts) != 0 {
			t.Fatalf("冲突筛选应返回真实空汇总: %+v %v", empty, err)
		}
	})
}
