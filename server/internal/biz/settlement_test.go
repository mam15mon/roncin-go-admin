package biz

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestResolveFeeLedgerFinancialProgressCoversSevenStates(t *testing.T) {
	tests := []struct {
		name     string
		hasBill  bool
		invoiced bool
		bill     string
		verified string
		expected FeeLedgerFinancialProgress
	}{
		{name: "账单未建立", hasBill: false, bill: "100", verified: "0", expected: FeeLedgerUnbilled},
		{name: "未核销未开票", hasBill: true, bill: "100", verified: "0", expected: FeeLedgerUnverifiedUninvoiced},
		{name: "已开票未核销", hasBill: true, invoiced: true, bill: "100", verified: "0", expected: FeeLedgerInvoicedUnverified},
		{name: "已核销未开票", hasBill: true, bill: "100", verified: "100", expected: FeeLedgerVerifiedUninvoiced},
		{name: "已开票部分核销", hasBill: true, invoiced: true, bill: "100", verified: "40", expected: FeeLedgerInvoicedPartiallyVerified},
		{name: "部分核销未开票", hasBill: true, bill: "100", verified: "40", expected: FeeLedgerPartiallyVerifiedUninvoiced},
		{name: "已完成", hasBill: true, invoiced: true, bill: "100", verified: "100", expected: FeeLedgerCompleted},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := ResolveFeeLedgerFinancialProgress(test.hasBill, test.invoiced, decimal.RequireFromString(test.bill), decimal.RequireFromString(test.verified))
			if actual != test.expected {
				t.Fatalf("财务进度 = %s，期望 %s", actual, test.expected)
			}
		})
	}
}
