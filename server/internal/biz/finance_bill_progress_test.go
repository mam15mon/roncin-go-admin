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

func TestResolveFinanceBillBusinessStatus(t *testing.T) {
	cases := []struct {
		name       string
		status     FinanceBillStatus
		invoice    FinanceBillInvoiceProgress
		settlement FinanceBillSettlementProgress
		done       bool
		verified   string
		want       FinanceBillBusinessStatus
	}{
		{"无票且未结", FinanceBillConfirmed, FinanceBillInvoiceNone, FinanceBillSettlementUnsettled, false, "0", FinanceBillBusinessUninvoiced},
		{"历史先核销后开票", FinanceBillConfirmed, FinanceBillInvoiceNone, FinanceBillSettlementSettled, false, "100", FinanceBillBusinessUninvoiced},
		{"部分票及部分核销优先补票", FinanceBillConfirmed, FinanceBillInvoicePartial, FinanceBillSettlementPartial, false, "40", FinanceBillBusinessPartiallyInvoiced},
		{"部分票已结清仍需补票", FinanceBillConfirmed, FinanceBillInvoicePartial, FinanceBillSettlementSettled, false, "100", FinanceBillBusinessPartiallyInvoiced},
		{"票齐尚未核销", FinanceBillConfirmed, FinanceBillInvoiceFull, FinanceBillSettlementUnsettled, false, "0", FinanceBillBusinessUnverified},
		{"纯对冲不冒充核销", FinanceBillConfirmed, FinanceBillInvoiceFull, FinanceBillSettlementPartial, false, "0", FinanceBillBusinessUnverified},
		{"票齐部分核销", FinanceBillConfirmed, FinanceBillInvoiceFull, FinanceBillSettlementPartial, false, "40", FinanceBillBusinessPartiallyVerified},
		{"带符号核销", FinanceBillConfirmed, FinanceBillInvoiceFull, FinanceBillSettlementPartial, false, "-40", FinanceBillBusinessPartiallyVerified},
		{"免票部分核销", FinanceBillConfirmed, FinanceBillInvoiceNotRequired, FinanceBillSettlementPartial, false, "40", FinanceBillBusinessPartiallyVerified},
		{"票齐且结清", FinanceBillConfirmed, FinanceBillInvoiceFull, FinanceBillSettlementSettled, true, "100", FinanceBillBusinessCompleted},
		{"零额或免票已结清", FinanceBillConfirmed, FinanceBillInvoiceNotRequired, FinanceBillSettlementSettled, true, "0", FinanceBillBusinessCompleted},
		{"取消不提供业务待办", FinanceBillCancelled, FinanceBillInvoiceNone, FinanceBillSettlementUnsettled, false, "0", ""},
		{"未经投影不猜测", FinanceBillConfirmed, "", "", false, "0", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveFinanceBillBusinessStatus(tc.status, tc.settlement, tc.invoice, tc.done, decimal.RequireFromString(tc.verified))
			if got != tc.want {
				t.Fatalf("业务状态=%s，期望%s", got, tc.want)
			}
		})
	}
}

func TestVerificationInvoicePrerequisite(t *testing.T) {
	for _, invoice := range []FinanceBillInvoiceProgress{FinanceBillInvoiceNotRequired, FinanceBillInvoicePartial, FinanceBillInvoiceFull} {
		if !CanVerifyFinanceBillInvoiceProgress(invoice) {
			t.Fatalf("%s应允许核销", invoice)
		}
	}
	for _, invoice := range []FinanceBillInvoiceProgress{FinanceBillInvoiceNone, "", "UNKNOWN"} {
		if CanVerifyFinanceBillInvoiceProgress(invoice) {
			t.Fatalf("%s不能核销", invoice)
		}
	}
	for _, status := range []FinanceBillBusinessStatus{FinanceBillBusinessUninvoiced, FinanceBillBusinessPartiallyInvoiced, FinanceBillBusinessUnverified, FinanceBillBusinessPartiallyVerified, FinanceBillBusinessCompleted} {
		if !ValidFinanceBillBusinessStatus(status) {
			t.Fatalf("合法状态%s被拒绝", status)
		}
	}
	if ValidFinanceBillBusinessStatus("") || ValidFinanceBillBusinessStatus("UNSPECIFIED") || ValidFinanceBillBusinessStatus("UNKNOWN") {
		t.Fatal("空或未知状态不得接受")
	}
}
