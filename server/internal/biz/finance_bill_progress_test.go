package biz

import (
	"testing"

	"github.com/shopspring/decimal"
)

// 零额与负数账单本期只验证判定逻辑（结论 1），不放开创建入口与数据库约束。
func TestResolveFinanceBillProgress(t *testing.T) {
	d := decimal.RequireFromString
	cases := []struct {
		name       string
		status     FinanceBillStatus
		total      string
		settled    string
		invoiced   string
		exempt     bool
		settlement FinanceBillSettlementProgress
		invoice    FinanceBillInvoiceProgress
		completed  bool
	}{
		{"正数未结未开票", FinanceBillConfirmed, "100", "0", "0", false, FinanceBillSettlementUnsettled, FinanceBillInvoiceNone, false},
		{"正数仅结清未完成", FinanceBillConfirmed, "100", "100", "0", false, FinanceBillSettlementSettled, FinanceBillInvoiceNone, false},
		{"正数仅票齐未完成", FinanceBillConfirmed, "100", "0", "100", false, FinanceBillSettlementUnsettled, FinanceBillInvoiceFull, false},
		{"正数部分结清部分开票", FinanceBillConfirmed, "100", "40", "30", false, FinanceBillSettlementPartial, FinanceBillInvoicePartial, false},
		{"正数结清且票齐完成", FinanceBillConfirmed, "100", "100", "100", false, FinanceBillSettlementSettled, FinanceBillInvoiceFull, true},
		{"核销与对冲合计结清", FinanceBillConfirmed, "300.5", "300.50000000", "300.5", false, FinanceBillSettlementSettled, FinanceBillInvoiceFull, true},
		{"无需发票标记后结清即完成", FinanceBillConfirmed, "100", "100", "0", true, FinanceBillSettlementSettled, FinanceBillInvoiceNotRequired, true},
		{"无需发票但未结清不完成", FinanceBillConfirmed, "100", "60", "0", true, FinanceBillSettlementPartial, FinanceBillInvoiceNotRequired, false},
		{"零额创建即完成", FinanceBillConfirmed, "0", "0", "0", false, FinanceBillSettlementSettled, FinanceBillInvoiceNotRequired, true},
		{"负数未结清", FinanceBillConfirmed, "-100", "0", "0", false, FinanceBillSettlementUnsettled, FinanceBillInvoiceNone, false},
		{"负数部分结清", FinanceBillConfirmed, "-100", "-40", "0", false, FinanceBillSettlementPartial, FinanceBillInvoiceNone, false},
		{"负数带符号结清但未开票不完成", FinanceBillConfirmed, "-100", "-100", "0", false, FinanceBillSettlementSettled, FinanceBillInvoiceNone, false},
		{"负数正向冲抵不算结清", FinanceBillConfirmed, "-100", "100", "0", false, FinanceBillSettlementPartial, FinanceBillInvoiceNone, false},
		{"负数结清且按规则票齐完成", FinanceBillConfirmed, "-100", "-100", "-100", false, FinanceBillSettlementSettled, FinanceBillInvoiceFull, true},
		{"负数结清且标记无需发票完成", FinanceBillConfirmed, "-100", "-100", "0", true, FinanceBillSettlementSettled, FinanceBillInvoiceNotRequired, true},
		{"取消账单不算完成", FinanceBillCancelled, "0", "0", "0", false, FinanceBillSettlementSettled, FinanceBillInvoiceNotRequired, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			settlement, invoice, completed := ResolveFinanceBillProgress(tc.status, d(tc.total), d(tc.settled), d(tc.invoiced), tc.exempt)
			if settlement != tc.settlement || invoice != tc.invoice || completed != tc.completed {
				t.Fatalf("进度=%s/%s/%v，预期=%s/%s/%v", settlement, invoice, completed, tc.settlement, tc.invoice, tc.completed)
			}
		})
	}
}

func TestFinanceBillListProgressFilterValidation(t *testing.T) {
	if !ValidFinanceBillSettlementProgress(FinanceBillSettlementPartial) || ValidFinanceBillSettlementProgress("UNSPECIFIED") {
		t.Fatal("结算进度筛选值校验错误")
	}
	if !ValidFinanceBillInvoiceProgress(FinanceBillInvoiceNotRequired) || ValidFinanceBillInvoiceProgress("") {
		t.Fatal("开票进度筛选值校验错误")
	}
}
