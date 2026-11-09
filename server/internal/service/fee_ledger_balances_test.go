package service

import (
	"encoding/json"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestFeeLedgerBalanceResponse(t *testing.T) {
	item := &biz.FeeLedgerItem{Fee: &biz.OrderFee{}, VerifiedAmount: decimal.RequireFromString("1234.5"), UnverifiedAmount: decimal.Zero, NettedAmount: decimal.RequireFromString("25")}
	got := feeLedgerItemToAPI(item)
	if got.VerifiedAmount != "1234.50000000" || got.UnverifiedAmount != "0.00000000" || got.NettedAmount != "25.00000000" {
		t.Fatalf("核销余额转换错误: %+v", got)
	}
	wire, err := protojson.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(wire, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["unverifiedAmount"] != "0.00000000" {
		t.Fatalf("显式零余额不应丢失: %s", wire)
	}
}
