package biz

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestNormalizeOrderFeeCalculatesExactTotal(t *testing.T) {
	fee := validOrderFeeForTest()
	fee.Quantity = decimal.RequireFromString("0.1")
	fee.UnitPrice = decimal.RequireFromString("0.2")

	normalized, err := normalizeOrderFee(fee)
	if err != nil {
		t.Fatalf("规范化费用失败: %v", err)
	}
	if got := normalized.TotalAmount.StringFixed(8); got != "0.02000000" {
		t.Fatalf("总金额应精确为 0.02000000，实际为 %s", got)
	}
}

func TestNormalizeOrderFeePreservesEightDecimalProduct(t *testing.T) {
	fee := validOrderFeeForTest()
	fee.Quantity = decimal.RequireFromString("1.2345")
	fee.UnitPrice = decimal.RequireFromString("6.7891")

	normalized, err := normalizeOrderFee(fee)
	if err != nil {
		t.Fatalf("规范化费用失败: %v", err)
	}
	if got := normalized.TotalAmount.StringFixed(8); got != "8.38114395" {
		t.Fatalf("总金额应保留完整乘积 8.38114395，实际为 %s", got)
	}
}

func TestNormalizeOrderFeeRejectsExcessPrecision(t *testing.T) {
	fee := validOrderFeeForTest()
	fee.UnitPrice = decimal.RequireFromString("1.00001")

	if _, err := normalizeOrderFee(fee); err != ErrOrderFeeInvalidArgument {
		t.Fatalf("五位小数单价应被拒绝，实际错误为 %v", err)
	}
}

func validOrderFeeForTest() *OrderFee {
	return &OrderFee{
		Direction:         OrderFeeReceivable,
		FeeCode:           "OCEAN_FREIGHT",
		FeeName:           "海运费",
		SettlementPartyID: uuid.Must(uuid.NewV7()),
		BillingUnit:       "票",
		Quantity:          decimal.RequireFromString("1"),
		UnitPrice:         decimal.RequireFromString("100"),
		Currency:          "CNY",
		ExchangeRate:      decimal.RequireFromString("1"),
		ExpenseDate:       "2026-08-24",
	}
}
