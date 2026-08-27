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

func TestCalculateVerificationAllocationAmountsRecognizesExchangeGainLoss(t *testing.T) {
	amount := decimal.RequireFromString("40")
	billTotal := decimal.RequireFromString("100")
	billBaseTotal := decimal.RequireFromString("720")
	cashflowTotal := decimal.RequireFromString("100")
	cashflowBaseTotal := decimal.RequireFromString("725")
	writeOffRate := decimal.RequireFromString("7.30")

	billBase, cashBase, writeOffBase, receivableGainLoss, err := CalculateVerificationAllocationAmounts(OrderFeeReceivable, amount, billTotal, billBaseTotal, cashflowTotal, cashflowBaseTotal, writeOffRate)
	if err != nil {
		t.Fatalf("计算应收核销汇兑损益失败: %v", err)
	}
	if billBase.StringFixed(8) != "288.00000000" || cashBase.StringFixed(8) != "290.00000000" || writeOffBase.StringFixed(8) != "292.00000000" || receivableGainLoss.StringFixed(8) != "2.00000000" {
		t.Fatalf("应收核销汇率快照不正确: bill=%s cash=%s writeOff=%s gainLoss=%s", billBase, cashBase, writeOffBase, receivableGainLoss)
	}

	_, _, _, payableGainLoss, err := CalculateVerificationAllocationAmounts(OrderFeePayable, amount, billTotal, billBaseTotal, cashflowTotal, cashflowBaseTotal, writeOffRate)
	if err != nil {
		t.Fatalf("计算应付核销汇兑损益失败: %v", err)
	}
	if payableGainLoss.StringFixed(8) != "-2.00000000" {
		t.Fatalf("应付核销时资金本币金额高于账面金额应形成汇兑损失，实际 %s", payableGainLoss)
	}
}
