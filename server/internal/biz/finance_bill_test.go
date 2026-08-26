package biz

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestBuildFinanceBillAggregatesExactSnapshots(t *testing.T) {
	organizationID := uuid.Must(uuid.NewV7())
	partyID := uuid.Must(uuid.NewV7())
	fees := []*FinanceBillableFee{
		financeBillableFeeForTest(partyID, "100.00000000", "94.33962264", "5.66037736", "100.00000000"),
		financeBillableFeeForTest(partyID, "0.02000000", "0.01886792", "0.00113208", "0.02000000"),
	}
	input := CreateFinanceBillInput{FeeIDs: []uuid.UUID{fees[0].Fee.ID, fees[1].Fee.ID}, BillDate: "2026-08-26", IdempotencyKey: "bill-test"}

	bill, err := buildFinanceBill(organizationID, fees, input)
	if err != nil {
		t.Fatalf("构建账单失败: %v", err)
	}
	if bill.TotalAmount.StringFixed(8) != "100.02000000" || bill.NetAmount.StringFixed(8) != "94.35849056" || bill.TaxAmount.StringFixed(8) != "5.66150944" {
		t.Fatalf("账单精确汇总不正确: total=%s net=%s tax=%s", bill.TotalAmount, bill.NetAmount, bill.TaxAmount)
	}
	if bill.FeeCount != 2 || len(bill.Lines) != 2 {
		t.Fatalf("账单费用快照数量不正确: count=%d lines=%d", bill.FeeCount, len(bill.Lines))
	}
}

func TestBuildFinanceBillRejectsMixedSettlementScope(t *testing.T) {
	firstPartyID := uuid.Must(uuid.NewV7())
	fees := []*FinanceBillableFee{
		financeBillableFeeForTest(firstPartyID, "100", "100", "0", "100"),
		financeBillableFeeForTest(uuid.Must(uuid.NewV7()), "100", "100", "0", "100"),
	}
	input := CreateFinanceBillInput{FeeIDs: []uuid.UUID{fees[0].Fee.ID, fees[1].Fee.ID}, BillDate: "2026-08-26", IdempotencyKey: "bill-test"}

	if _, err := buildFinanceBill(uuid.Must(uuid.NewV7()), fees, input); err != ErrFinanceBillFeeMismatch {
		t.Fatalf("混合结算单位应被拒绝，实际错误为 %v", err)
	}
}

func TestNormalizeCreateFinanceBillRejectsDuplicateFeesAndInvalidDueDate(t *testing.T) {
	feeID := uuid.Must(uuid.NewV7())
	if _, err := normalizeCreateFinanceBill(CreateFinanceBillInput{FeeIDs: []uuid.UUID{feeID, feeID}, BillDate: "2026-08-26", IdempotencyKey: "duplicate"}); err != ErrFinanceBillInvalidArgument {
		t.Fatalf("重复费用应被拒绝，实际错误为 %v", err)
	}
	dueDate := "2026-08-25"
	if _, err := normalizeCreateFinanceBill(CreateFinanceBillInput{FeeIDs: []uuid.UUID{feeID}, BillDate: "2026-08-26", DueDate: &dueDate, IdempotencyKey: "due-date"}); err != ErrFinanceBillInvalidArgument {
		t.Fatalf("早于账单日期的到期日应被拒绝，实际错误为 %v", err)
	}
}

func financeBillableFeeForTest(partyID uuid.UUID, total, net, tax, base string) *FinanceBillableFee {
	feeID := uuid.Must(uuid.NewV7())
	return &FinanceBillableFee{
		OrderNo: "SE2026082600001", BusinessType: "SE",
		Fee: &OrderFee{
			ID: feeID, OrderID: uuid.Must(uuid.NewV7()), Direction: OrderFeeReceivable, Status: OrderFeeConfirmed,
			SettlementPartyID: partyID, SettlementPartyName: "验收客户", Currency: "CNY", BaseCurrency: "CNY",
			FeeCode: "OCEAN", FeeName: "海运费", TotalAmount: decimal.RequireFromString(total),
			NetAmount: decimal.RequireFromString(net), TaxAmount: decimal.RequireFromString(tax),
			ExchangeRate: decimal.NewFromInt(1), BaseCurrencyAmount: decimal.RequireFromString(base),
		},
	}
}
