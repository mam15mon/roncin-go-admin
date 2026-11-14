package data

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financeinvoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	financeinvoicebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	financenettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	verificationallocent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
)

// TestFinanceBillProgressProjectionPostgres 验证账单进度投影、费用锁定事实与列表筛选同源：
//   - 仅结清、仅票齐均不完成；核销与已确认对冲合计结清且票齐才完成；部分核销/部分开票为部分；
//   - 费用锁定事实：发票、核销、已确认对冲、草稿对冲分别产出，草稿对冲不改变受保护事实口径；
//     两张发票只作废一张仍锁定，全部作废后解除；
//   - 结算/开票进度与完成筛选返回的集合与逐行投影一致，未结清筛选保持原口径。
//
// 零额与负数账单受数据库正数约束限制本期无法落库（结论 1），由 biz 表格测试覆盖。
func TestFinanceBillProgressProjectionPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	data, cleanupData := getIntegrationData(t)
	t.Cleanup(cleanupData)
	f := newFeeLedgerPostgresFixture(t, data)
	ctx := context.Background()
	repo := NewFinanceBillRepo(data)
	scope := []uuid.UUID{f.organizationID}

	var invoiceIDs []uuid.UUID
	// 注册在夹具清理之后，LIFO 先执行：发票关联引用账单，需先于账单删除。
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
	addInvoice := func(key string, bill *ent.FinanceBill, amount string) *ent.FinanceInvoice {
		t.Helper()
		amount8 := decimal.RequireFromString(amount).StringFixed(8)
		invoice, err := data.db.FinanceInvoice.Create().
			SetOrganizationID(f.organizationID).SetRecordNo("INV-PG-" + key + "-" + f.suffix).SetIdempotencyKey("inv-pg-" + key + "-" + f.suffix).
			SetDirection(financeinvoiceent.DirectionRECEIVABLE).SetInvoiceType(financeinvoiceent.InvoiceTypeNORMAL).
			SetSettlementPartyID(f.partnerID).SetSettlementPartyName("进度投影").SetCurrency("CNY").SetBaseCurrency("CNY").
			SetTotalAmount(amount8).SetNetAmount(amount8).SetTaxAmount("0").SetBillCount(1).SetStatus(financeinvoiceent.StatusDRAFT).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建发票 %s: %v", key, err)
		}
		invoiceIDs = append(invoiceIDs, invoice.ID)
		if _, err := data.db.FinanceInvoiceBill.Create().SetInvoiceID(invoice.ID).SetBillID(bill.ID).SetBillNo(bill.BillNo).
			SetAmount(amount8).SetTaxAmount("0").SetActive(true).Save(ctx); err != nil {
			t.Fatalf("创建发票账单关联 %s: %v", key, err)
		}
		return invoice
	}
	cancelInvoice := func(invoice *ent.FinanceInvoice) {
		t.Helper()
		invoice.Update().SetStatus(financeinvoiceent.StatusCANCELLED).SetCancelledAt(time.Now()).SetCancelledBy(f.actorID).SetCancellationReason("测试").ExecX(ctx)
		data.db.FinanceInvoiceBill.Update().Where(financeinvoicebillent.InvoiceIDEQ(invoice.ID)).SetActive(false).ExecX(ctx)
	}
	verify := func(key string, bill *ent.FinanceBill, amount string) {
		t.Helper()
		cash := f.createConfirmedCashflow(key, amount, financeBillIntegrationDate)
		f.createActiveVerification(key, amount, func(id uuid.UUID) []*ent.FinanceVerificationAllocationCreate {
			return []*ent.FinanceVerificationAllocationCreate{f.verificationAllocationCreate(id, cash, bill, amount)}
		})
	}
	newBill := func(key string) *ent.FinanceBill {
		feeID := f.createLedgerFee(key, "100")
		bill := f.createConfirmedBill(key, "100", financeBillIntegrationDate)
		f.linkBillFee(key, bill, feeID, "100")
		return bill
	}
	get := func(bill *ent.FinanceBill) *biz.FinanceBill {
		t.Helper()
		got, err := repo.Get(ctx, scope, bill.ID)
		if err != nil {
			t.Fatalf("读取账单: %v", err)
		}
		return got
	}

	none := newBill("none")
	settledOnly := newBill("settled-only")
	verify("settled-only", settledOnly, "100")
	invoicedOnly := newBill("invoiced-only")
	addInvoice("invoiced-only", invoicedOnly, "100")
	completed := newBill("completed")
	verify("completed", completed, "40")
	f.createNetting("completed", financenettingent.StatusCONFIRMED, completed, "60", true)
	addInvoice("completed", completed, "100")
	partial := newBill("partial")
	verify("partial", partial, "40")
	addInvoice("partial", partial, "30")
	draftNetting := newBill("draft-netting")
	f.createNetting("draft-netting", financenettingent.StatusDRAFT, draftNetting, "100", false)

	expected := map[uuid.UUID]struct {
		settlement biz.FinanceBillSettlementProgress
		invoice    biz.FinanceBillInvoiceProgress
		completed  bool
		facts      []biz.FinanceBillFeeLockFact
		protected  bool
	}{
		none.ID:         {biz.FinanceBillSettlementUnsettled, biz.FinanceBillInvoiceNone, false, nil, false},
		settledOnly.ID:  {biz.FinanceBillSettlementSettled, biz.FinanceBillInvoiceNone, false, []biz.FinanceBillFeeLockFact{biz.FinanceBillFeeLockVerification}, true},
		invoicedOnly.ID: {biz.FinanceBillSettlementUnsettled, biz.FinanceBillInvoiceFull, false, []biz.FinanceBillFeeLockFact{biz.FinanceBillFeeLockInvoice}, true},
		completed.ID:    {biz.FinanceBillSettlementSettled, biz.FinanceBillInvoiceFull, true, []biz.FinanceBillFeeLockFact{biz.FinanceBillFeeLockInvoice, biz.FinanceBillFeeLockVerification, biz.FinanceBillFeeLockNetting}, true},
		partial.ID:      {biz.FinanceBillSettlementPartial, biz.FinanceBillInvoicePartial, false, []biz.FinanceBillFeeLockFact{biz.FinanceBillFeeLockInvoice, biz.FinanceBillFeeLockVerification}, true},
		// 草稿对冲只锁定费用增删，不进入受保护事实（账单字段与汇率联动口径不变），也不抵减余额。
		draftNetting.ID: {biz.FinanceBillSettlementUnsettled, biz.FinanceBillInvoiceNone, false, []biz.FinanceBillFeeLockFact{biz.FinanceBillFeeLockNettingDraft}, false},
	}

	t.Run("详情投影与费用锁定事实", func(t *testing.T) {
		for _, bill := range []*ent.FinanceBill{none, settledOnly, invoicedOnly, completed, partial, draftNetting} {
			want := expected[bill.ID]
			got := get(bill)
			if got.SettlementProgress != want.settlement || got.InvoiceProgress != want.invoice || got.Completed != want.completed {
				t.Fatalf("%s 进度=%s/%s/%v，期望 %s/%s/%v", bill.BillNo, got.SettlementProgress, got.InvoiceProgress, got.Completed, want.settlement, want.invoice, want.completed)
			}
			if !equalFeeLockFacts(got.FeeLockFacts, want.facts) || got.FeesLocked() != (len(want.facts) > 0) {
				t.Fatalf("%s 锁定事实=%v，期望 %v", bill.BillNo, got.FeeLockFacts, want.facts)
			}
			protected, err := billHasProtectedFinanceFact(ctx, data.db, bill.ID)
			if err != nil || protected != want.protected {
				t.Fatalf("%s 受保护=%v（%v），期望 %v", bill.BillNo, protected, err, want.protected)
			}
		}
	})

	t.Run("列表筛选与逐行投影同源", func(t *testing.T) {
		list := func(filter biz.FinanceBillFilter) []uuid.UUID {
			t.Helper()
			filter.Page, filter.PageSize = 1, 200
			result, err := repo.List(ctx, scope, filter)
			if err != nil {
				t.Fatalf("列表查询: %v", err)
			}
			if result.Total != int64(len(result.Items)) {
				t.Fatalf("总数 %d 与条目 %d 不一致", result.Total, len(result.Items))
			}
			ids := make([]uuid.UUID, 0, len(result.Items))
			for _, item := range result.Items {
				want := expected[item.ID]
				if item.SettlementProgress != want.settlement || item.InvoiceProgress != want.invoice || item.Completed != want.completed || !equalFeeLockFacts(item.FeeLockFacts, want.facts) {
					t.Fatalf("列表行 %s 投影与详情不一致: %#v", item.BillNo, item)
				}
				ids = append(ids, item.ID)
			}
			return sortedUUIDs(ids)
		}
		matching := func(keep func(id uuid.UUID) bool) []uuid.UUID {
			ids := make([]uuid.UUID, 0)
			for id := range expected {
				if keep(id) {
					ids = append(ids, id)
				}
			}
			return sortedUUIDs(ids)
		}
		for _, progress := range []biz.FinanceBillSettlementProgress{biz.FinanceBillSettlementUnsettled, biz.FinanceBillSettlementPartial, biz.FinanceBillSettlementSettled} {
			got := list(biz.FinanceBillFilter{SettlementProgress: progress})
			want := matching(func(id uuid.UUID) bool { return expected[id].settlement == progress })
			if !equalUUIDs(got, want) {
				t.Fatalf("结算进度 %s 筛选=%v，期望 %v", progress, got, want)
			}
		}
		for _, progress := range []biz.FinanceBillInvoiceProgress{biz.FinanceBillInvoiceNotRequired, biz.FinanceBillInvoiceNone, biz.FinanceBillInvoicePartial, biz.FinanceBillInvoiceFull} {
			got := list(biz.FinanceBillFilter{InvoiceProgress: progress})
			want := matching(func(id uuid.UUID) bool { return expected[id].invoice == progress })
			if !equalUUIDs(got, want) {
				t.Fatalf("开票进度 %s 筛选=%v，期望 %v", progress, got, want)
			}
		}
		for _, done := range []bool{true, false} {
			value := done
			got := list(biz.FinanceBillFilter{Completed: &value})
			want := matching(func(id uuid.UUID) bool { return expected[id].completed == done })
			if !equalUUIDs(got, want) {
				t.Fatalf("完成=%v 筛选=%v，期望 %v", done, got, want)
			}
		}
		got := list(biz.FinanceBillFilter{OnlyUnsettled: true})
		want := matching(func(id uuid.UUID) bool { return expected[id].settlement != biz.FinanceBillSettlementSettled })
		if !equalUUIDs(got, want) {
			t.Fatalf("未结清筛选=%v，期望 %v", got, want)
		}
	})

	// 当前同一账单同时只允许一条有效发票关联（finance_invoice_bills_active_bill_unique），
	// 「两张发票只作废一张」待发票任务放开跨票分配后补充；此处验证部分撤销不解锁、全部撤销后恢复。
	t.Run("部分撤销不解锁，全部撤销后恢复", func(t *testing.T) {
		bill := newBill("revoke")
		invoice := addInvoice("revoke", bill, "60")
		verify("revoke", bill, "30")
		if got := get(bill); got.InvoiceProgress != biz.FinanceBillInvoicePartial || !equalFeeLockFacts(got.FeeLockFacts, []biz.FinanceBillFeeLockFact{biz.FinanceBillFeeLockInvoice, biz.FinanceBillFeeLockVerification}) {
			t.Fatalf("部分开票与核销应锁定: %v %v", got.InvoiceProgress, got.FeeLockFacts)
		}
		cancelInvoice(invoice)
		if got := get(bill); got.InvoiceProgress != biz.FinanceBillInvoiceNone || !equalFeeLockFacts(got.FeeLockFacts, []biz.FinanceBillFeeLockFact{biz.FinanceBillFeeLockVerification}) {
			t.Fatalf("发票全部作废但仍有核销应保持锁定: %v %v", got.InvoiceProgress, got.FeeLockFacts)
		}
		data.db.FinanceVerificationAllocation.Update().Where(verificationallocent.BillIDEQ(bill.ID)).SetActive(false).ExecX(ctx)
		got := get(bill)
		if got.SettlementProgress != biz.FinanceBillSettlementUnsettled || got.FeesLocked() {
			t.Fatalf("最后一项撤销后应恢复: %v %v", got.SettlementProgress, got.FeeLockFacts)
		}
		facts, err := billFeeLockFacts(ctx, data.db, bill.ID)
		if err != nil || len(facts) != 0 {
			t.Fatalf("单账单锁定事实=%v（%v）", facts, err)
		}
	})
}

func equalFeeLockFacts(left, right []biz.FinanceBillFeeLockFact) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func equalUUIDs(left, right []uuid.UUID) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
