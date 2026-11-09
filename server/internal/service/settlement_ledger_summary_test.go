package service

import (
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
	"testing"
)

func TestFinanceLedgerMatchedSummaryToAPIPreservesMissingAndZeroBase(t *testing.T) {
	baseCurrency := "CNY"
	zero := decimal.Zero
	got := financeLedgerMatchedSummaryToAPI(biz.FinanceLedgerMatchedSummary{Count: 2, Amounts: []biz.FinanceLedgerMatchedAmount{
		{Direction: "RECEIVABLE", Currency: "USD", Amount: decimal.RequireFromString("1.23456789"), BaseCurrency: &baseCurrency, UnconvertedCount: 1},
		{Currency: "CNY", Amount: decimal.Zero, BaseCurrency: &baseCurrency, BaseAmount: &zero},
	}})
	if got.Count != 2 || len(got.Amounts) != 2 {
		t.Fatal("计数丢失")
	}
	if got.Amounts[0].Amount != "1.23456789" || got.Amounts[0].BaseAmount != nil || got.Amounts[0].UnconvertedCount != 1 {
		t.Fatalf("未折算或精度丢失: %+v", got.Amounts[0])
	}
	if got.Amounts[1].BaseAmount == nil || *got.Amounts[1].BaseAmount != "0.00000000" {
		t.Fatal("显式零本币值丢失")
	}
}
