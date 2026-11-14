package biz

import (
	"testing"

	"github.com/shopspring/decimal"
)

// 本币未税由各费用自身汇率快照折算，与提成口径同源；多币种各算各的，不跨币种相加。
func TestOrderFeeBaseCurrencyNetAmount(t *testing.T) {
	cases := []struct {
		name             string
		total, net, base string
		wantBaseNet      string
	}{
		{"CNY 含税106未税100", "106", "100", "106", "100"},
		{"CNY 含税53未税50", "53", "50", "53", "50"},
		{"USD 汇率7 含税106未税100", "106", "100", "742", "700"},
		{"EUR 汇率7.8 含税53未税50", "53", "50", "413.4", "390"},
		{"零税率", "100", "100", "700", "700"},
		{"含税为零", "0", "0", "0", "0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fee := &OrderFee{
				TotalAmount:        decimal.RequireFromString(c.total),
				NetAmount:          decimal.RequireFromString(c.net),
				BaseCurrencyAmount: decimal.RequireFromString(c.base),
			}
			if got := fee.BaseCurrencyNetAmount(); !got.Equal(decimal.RequireFromString(c.wantBaseNet)) {
				t.Fatalf("本币未税 = %s，期望 %s", got, c.wantBaseNet)
			}
			if !fee.BaseCurrencyNetAmount().Equal(CommissionNetBase(fee.BaseCurrencyAmount, fee.NetAmount, fee.TotalAmount)) {
				t.Fatal("必须与提成不含税口径一致")
			}
		})
	}
}
