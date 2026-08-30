package service

import (
	"testing"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestParsePlainDecimalRejectsScientificNotation(t *testing.T) {
	if _, err := parsePlainDecimal("1e2"); err == nil {
		t.Fatal("科学计数法不应进入费用金额链路")
	}
}

func TestParsePlainDecimalRejectsSignedValue(t *testing.T) {
	if _, err := parsePlainDecimal("+1.25"); err == nil {
		t.Fatal("带符号十进制文本不应进入费用金额链路")
	}
}

func TestOrderFeeStatusAPIConversions(t *testing.T) {
	status := v1.OrderFeeStatus_ORDER_FEE_STATUS_CONFIRMED
	if got := orderFeeStatusFromAPI(&status); got != biz.OrderFeeConfirmed {
		t.Fatalf("费用领域状态 = %q", got)
	}
	if got := orderFeeStatusToAPI(biz.OrderFeeBilled); got != v1.OrderFeeStatus_ORDER_FEE_STATUS_BILLED {
		t.Fatalf("费用 API 状态 = %v", got)
	}
	if got := orderFeeStatusFromAPI(nil); got != "" {
		t.Fatalf("空费用状态 = %q", got)
	}
}

func TestOrderFeeDirectionFromAPISupportsReceivableAndPayable(t *testing.T) {
	tests := []struct {
		name       string
		input      v1.OrderFeeDirection
		expected   biz.OrderFeeDirection
		expectedOK bool
	}{
		{name: "应收", input: v1.OrderFeeDirection_ORDER_FEE_DIRECTION_RECEIVABLE, expected: biz.OrderFeeReceivable, expectedOK: true},
		{name: "应付", input: v1.OrderFeeDirection_ORDER_FEE_DIRECTION_PAYABLE, expected: biz.OrderFeePayable, expectedOK: true},
		{name: "未指定", input: v1.OrderFeeDirection_ORDER_FEE_DIRECTION_UNSPECIFIED, expectedOK: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, ok := orderFeeDirectionFromAPI(test.input)
			if ok != test.expectedOK || actual != test.expected {
				t.Fatalf("orderFeeDirectionFromAPI(%v) = (%q, %t), want (%q, %t)", test.input, actual, ok, test.expected, test.expectedOK)
			}
		})
	}
}
