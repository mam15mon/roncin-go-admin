package biz

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestValidateFeeQuantityForUnit(t *testing.T) {
	cases := []struct {
		name           string
		quantity       string
		mustBeInteger  bool
		wantIntegerErr bool
	}{
		{name: "整数", quantity: "1", mustBeInteger: true},
		{name: "尾零整数", quantity: "2.0000", mustBeInteger: true},
		{name: "小数单位", quantity: "4.4", mustBeInteger: false},
		{name: "整数单位小数", quantity: "4.4", mustBeInteger: true, wantIntegerErr: true},
		{name: "整数单位半票", quantity: "0.5", mustBeInteger: true, wantIntegerErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateFeeQuantityForUnit(decimal.RequireFromString(tc.quantity), tc.mustBeInteger)
			if (err == ErrOrderFeeQuantityMustBeInteger) != tc.wantIntegerErr {
				t.Fatalf("数量 %s 校验结果: %v", tc.quantity, err)
			}
		})
	}
}
