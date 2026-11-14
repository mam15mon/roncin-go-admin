package biz

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestInvoiceRegistrationAmountExponentBounds(t *testing.T) {
	for _, tc := range []struct {
		name     string
		amount   decimal.Decimal
		positive bool
		valid    bool
	}{
		{"最大合法整数与小数", decimal.RequireFromString("99999999999999999999.99999999"), true, true},
		{"合法正指数", decimal.New(1, 19), true, true},
		{"超出整数位", decimal.New(1, 20), true, false},
		{"极大正指数", decimal.New(1, 2147483647), true, false},
		{"极大小指数", decimal.New(1, -2147483647), true, false},
		{"超出八位小数", decimal.New(1, -9), true, false},
		{"允许明确零税额", decimal.Zero, false, true},
		{"含税金额拒绝零", decimal.Zero, true, false},
		{"拒绝负税额", decimal.NewFromInt(-1), false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if invoiceAmountValid(tc.amount, tc.positive) != tc.valid {
				t.Fatal("金额必须先限指数，再校验numeric(28,8)值域；不得展开巨大指数")
			}
		})
	}
}
