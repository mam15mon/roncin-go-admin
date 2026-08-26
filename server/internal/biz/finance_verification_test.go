package biz

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestSameVerificationIntentIgnoresAllocationOrder(t *testing.T) {
	cashflowA, cashflowB := uuid.New(), uuid.New()
	billA, billB := uuid.New(), uuid.New()
	note := "同一请求"
	existing := &FinanceVerification{
		VerificationDate: "2026-08-26",
		Note:             &note,
		Allocations: []*VerificationAllocation{
			{CashflowID: cashflowA, BillID: billA, Amount: decimal.RequireFromString("10")},
			{CashflowID: cashflowB, BillID: billB, Amount: decimal.RequireFromString("20")},
		},
	}
	requested := CreateVerificationInput{
		VerificationDate: "2026-08-26",
		Note:             &note,
		Allocations: []*VerificationAllocation{
			{CashflowID: cashflowB, BillID: billB, Amount: decimal.RequireFromString("20.00000000")},
			{CashflowID: cashflowA, BillID: billA, Amount: decimal.RequireFromString("10.0")},
		},
	}
	if !sameVerificationIntent(existing, requested) {
		t.Fatal("相同分配集合应被识别为同一幂等请求")
	}
	requested.Allocations[0].Amount = decimal.RequireFromString("20.01")
	if sameVerificationIntent(existing, requested) {
		t.Fatal("金额不同的请求不应复用已有核销")
	}
}
