package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
)

// 费用详情与台账两处投影都输出同一个本币未税计算值（含税 106 / 未税 100）。
func TestBaseCurrencyNetAmountProjection(t *testing.T) {
	fee := &biz.OrderFee{
		ID: uuid.New(), OrderID: uuid.New(), SettlementPartyID: uuid.New(),
		Direction:          biz.OrderFeeReceivable,
		TotalAmount:        decimal.NewFromInt(106),
		NetAmount:          decimal.NewFromInt(100),
		BaseCurrencyAmount: decimal.NewFromInt(742),
	}
	if got := orderFeeToAPI(fee).GetBaseCurrencyNetAmount(); got != "700.00000000" {
		t.Fatalf("费用详情本币未税 = %q", got)
	}
	ledger := feeLedgerItemToAPI(&biz.FeeLedgerItem{Fee: fee})
	if got := ledger.GetBaseCurrencyNetAmount(); got != "700.00000000" {
		t.Fatalf("台账本币未税 = %q", got)
	}
}
