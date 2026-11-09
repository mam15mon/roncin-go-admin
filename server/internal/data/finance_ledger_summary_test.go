package data

import (
	"github.com/shopspring/decimal"
	"testing"
)

func TestFinanceLedgerMatchedSummaryRowsPreservePrecisionAndPresence(t *testing.T) {
	cny, usd := "CNY", "USD"
	zero, base := "0.00000000", "12345678901234567890.12345678"
	got, err := financeLedgerMatchedSummaryFromRows([]financeLedgerMatchedRow{
		{Direction: "PAYABLE", Currency: "USD", BaseCurrency: &cny, Amount: "-0.00000001", BaseAmount: &zero, RecordCount: 1},
		{Direction: "RECEIVABLE", Currency: "EUR", BaseCurrency: &usd, Amount: "1.10000001", RecordCount: 2, UnconvertedCount: 2},
		{Direction: "RECEIVABLE", Currency: "USD", BaseCurrency: &cny, Amount: "12345678901234567890.12345678", BaseAmount: &base, RecordCount: 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != 6 || len(got.Amounts) != 3 {
		t.Fatalf("计数或币种被合并: %+v", got)
	}
	if got.Amounts[0].BaseAmount == nil || !got.Amounts[0].BaseAmount.IsZero() {
		t.Fatal("显式本币零值丢失")
	}
	if got.Amounts[1].BaseAmount != nil || got.Amounts[1].UnconvertedCount != 2 {
		t.Fatal("未折算误作零")
	}
	if !got.Amounts[2].Amount.Equal(decimal.RequireFromString(base)) {
		t.Fatal("数值精度丢失")
	}
}

func TestFinanceLedgerMatchedSummaryRowsRejectInvalidAmount(t *testing.T) {
	bad := "invalid"
	for _, rows := range [][]financeLedgerMatchedRow{{{Amount: bad}}, {{Amount: "0", BaseAmount: &bad}}} {
		if _, err := financeLedgerMatchedSummaryFromRows(rows); err == nil {
			t.Fatal("损坏金额必须显式失败")
		}
	}
	empty, err := financeLedgerMatchedSummaryFromRows(nil)
	if err != nil || empty.Count != 0 || len(empty.Amounts) != 0 {
		t.Fatal("空筛选结果应为真实零集合")
	}
}
