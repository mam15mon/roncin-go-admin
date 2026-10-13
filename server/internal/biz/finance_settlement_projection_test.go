package biz

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"testing"
	"time"
)

func TestFinanceSettlementProjectionMergedCorrection(t *testing.T) {
	old, reverse, newBill := uuid.New(), uuid.New(), uuid.New()
	a, b, rev, replacement := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	at := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	superseded := at.Add(time.Minute)
	request := uuid.New()
	lines := []FinanceSettlementLine{{BillID: old, ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), FeeID: a, TotalAmount: decimal.NewFromInt(1000), SupersededAt: &superseded}, {BillID: old, ID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), FeeID: b, TotalAmount: decimal.NewFromInt(200)}, {BillID: reverse, ID: uuid.New(), FeeID: rev, TotalAmount: decimal.NewFromInt(1000)}, {BillID: newBill, ID: uuid.New(), FeeID: replacement, TotalAmount: decimal.NewFromInt(100)}}
	event := func(kind string, billID uuid.UUID, feeID *uuid.UUID, amount string, minute int, correction bool) FinanceSettlementEvent {
		e := FinanceSettlementEvent{SourceType: kind, SourceID: uuid.New(), AllocationID: uuid.New(), BillID: billID, SourceFeeID: feeID, Amount: decimal.RequireFromString(amount), EffectiveAt: at.Add(time.Duration(minute) * time.Minute)}
		if correction {
			e.CorrectionRequestID = &request
		}
		return e
	}
	events := []FinanceSettlementEvent{event("VERIFICATION", old, nil, "400", 0, false), event("NETTING", old, &a, "666.66666667", 2, true), event("NETTING", reverse, &rev, "666.66666667", 2, true), event("NETTING", reverse, &rev, "100", 3, true), event("NETTING", newBill, &replacement, "100", 3, true), event("VERIFICATION", old, nil, "133.33333333", 4, false)}
	projection, err := ProjectFinanceBillLineSettlements(lines, events)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[uuid.UUID]string{a: "0", b: "0", rev: "233.33333333", replacement: "0"}
	for _, line := range projection {
		if !line.RemainingAmount.Equal(decimal.RequireFromString(expected[line.FeeID])) {
			t.Fatalf("fee=%s remaining=%s expected=%s", line.FeeID, line.RemainingAmount, expected[line.FeeID])
		}
	}
}

func TestFinanceSettlementProjectionReversalKeepsLaterAssignment(t *testing.T) {
	bill := uuid.New()
	a, b := uuid.New(), uuid.New()
	at := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	reversed := at.Add(3 * time.Minute)
	lines := []FinanceSettlementLine{{BillID: bill, ID: a, FeeID: a, TotalAmount: decimal.NewFromInt(100)}, {BillID: bill, ID: b, FeeID: b, TotalAmount: decimal.NewFromInt(200)}}
	first := FinanceSettlementEvent{SourceType: "VERIFICATION", SourceID: uuid.New(), AllocationID: uuid.New(), BillID: bill, Amount: decimal.NewFromInt(100), EffectiveAt: at, ReversedAt: &reversed}
	later := FinanceSettlementEvent{SourceType: "VERIFICATION", SourceID: uuid.New(), AllocationID: uuid.New(), BillID: bill, Amount: decimal.NewFromInt(100), EffectiveAt: at.Add(time.Minute)}
	prior, err := ProjectFinanceBillLineSettlements(lines, []FinanceSettlementEvent{first, later})
	if err != nil {
		t.Fatal(err)
	}
	first.ReversedAt = nil
	original, err := ProjectFinanceBillLineSettlements(lines, []FinanceSettlementEvent{later, first})
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range prior {
		if line.Contributions[0].ReversedAt == nil || !line.SettledAmount.Equal(original[i].Contributions[1].Amount) {
			t.Fatalf("后续归属改变: %+v", line)
		}
	}
}

func TestFinanceSettlementProjectionBlocksUnclearedCorrection(t *testing.T) {
	bill, fee := uuid.New(), uuid.New()
	at := time.Now()
	superseded := at.Add(-time.Minute)
	_, err := ProjectFinanceBillLineSettlements([]FinanceSettlementLine{{BillID: bill, ID: uuid.New(), FeeID: fee, TotalAmount: decimal.NewFromInt(100), SupersededAt: &superseded}}, []FinanceSettlementEvent{{SourceType: "VERIFICATION", SourceID: uuid.New(), AllocationID: uuid.New(), BillID: bill, Amount: decimal.NewFromInt(1), EffectiveAt: at}})
	if err != ErrFinanceCorrectionSettlementRequired {
		t.Fatalf("应先内部清偿: %v", err)
	}
}

func TestFinanceSettlementProjectionTinyRoundingConservesAmount(t *testing.T) {
	bill := uuid.New()
	lines := make([]FinanceSettlementLine, 10)
	for i := range lines {
		lines[i] = FinanceSettlementLine{BillID: bill, ID: uuid.New(), FeeID: uuid.New(), TotalAmount: decimal.RequireFromString("0.00000001")}
	}
	events := []FinanceSettlementEvent{{SourceType: "VERIFICATION", SourceID: uuid.New(), AllocationID: uuid.New(), BillID: bill, Amount: decimal.RequireFromString("0.00000006"), EffectiveAt: time.Now()}}
	results, err := ProjectFinanceBillLineSettlements(lines, events)
	if err != nil {
		t.Fatal(err)
	}
	settled := decimal.Zero
	for _, line := range results {
		if line.RemainingAmount.IsNegative() {
			t.Fatal("负余额")
		}
		settled = settled.Add(line.SettledAmount)
	}
	if !settled.Equal(events[0].Amount) {
		t.Fatalf("合法tiny结算不守恒: %s", settled)
	}
}
