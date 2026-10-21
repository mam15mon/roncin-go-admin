package service

import (
	"testing"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestOrderDangerousPackingGroupFromAPI(t *testing.T) {
	tests := []struct {
		name     string
		input    *v1.DangerousPackingGroup
		expected *biz.OrderDangerousPackingGroup
	}{
		{name: "nil 不携带保持未设置", input: nil, expected: nil},
		{
			name:     "UNSPECIFIED 视为清空（未填 / 不适用）",
			input:    apiEnumPtr(v1.DangerousPackingGroup_DANGEROUS_PACKING_GROUP_UNSPECIFIED),
			expected: nil,
		},
		{
			name:     "Ⅰ类包装",
			input:    apiEnumPtr(v1.DangerousPackingGroup_DANGEROUS_PACKING_GROUP_I),
			expected: bizEnumPtr(biz.OrderDangerousPackingI),
		},
		{
			name:     "Ⅱ类包装",
			input:    apiEnumPtr(v1.DangerousPackingGroup_DANGEROUS_PACKING_GROUP_II),
			expected: bizEnumPtr(biz.OrderDangerousPackingII),
		},
		{
			name:     "Ⅲ类包装",
			input:    apiEnumPtr(v1.DangerousPackingGroup_DANGEROUS_PACKING_GROUP_III),
			expected: bizEnumPtr(biz.OrderDangerousPackingIII),
		},
		{
			name:     "未知取值按未设置处理",
			input:    apiEnumPtr(v1.DangerousPackingGroup(99)),
			expected: nil,
		},
	}
	for _, tc := range tests {
		if got := orderDangerousPackingGroupFromAPI(tc.input); !sameDangerousPackingGroup(got, tc.expected) {
			t.Fatalf("%s: orderDangerousPackingGroupFromAPI(%v) = %v, want %v", tc.name, tc.input, got, tc.expected)
		}
	}
}

func TestOrderDangerousPackingGroupToAPI(t *testing.T) {
	tests := []struct {
		name     string
		input    *biz.OrderDangerousPackingGroup
		expected *v1.DangerousPackingGroup
	}{
		{name: "nil 未填不投影", input: nil, expected: nil},
		{
			name:     "Ⅰ类包装",
			input:    bizEnumPtr(biz.OrderDangerousPackingI),
			expected: apiEnumPtr(v1.DangerousPackingGroup_DANGEROUS_PACKING_GROUP_I),
		},
		{
			name:     "Ⅱ类包装",
			input:    bizEnumPtr(biz.OrderDangerousPackingII),
			expected: apiEnumPtr(v1.DangerousPackingGroup_DANGEROUS_PACKING_GROUP_II),
		},
		{
			name:     "Ⅲ类包装",
			input:    bizEnumPtr(biz.OrderDangerousPackingIII),
			expected: apiEnumPtr(v1.DangerousPackingGroup_DANGEROUS_PACKING_GROUP_III),
		},
		{
			name:     "未知取值按未设置处理",
			input:    bizEnumPtr(biz.OrderDangerousPackingGroup("PG_UNKNOWN")),
			expected: nil,
		},
	}
	for _, tc := range tests {
		if got := orderDangerousPackingGroupToAPI(tc.input); !sameAPIDangerousPackingGroup(got, tc.expected) {
			t.Fatalf("%s: orderDangerousPackingGroupToAPI(%v) = %v, want %v", tc.name, tc.input, got, tc.expected)
		}
	}
}

func apiEnumPtr(value v1.DangerousPackingGroup) *v1.DangerousPackingGroup { return &value }

func bizEnumPtr(value biz.OrderDangerousPackingGroup) *biz.OrderDangerousPackingGroup {
	return &value
}

func sameDangerousPackingGroup(got, want *biz.OrderDangerousPackingGroup) bool {
	if got == nil || want == nil {
		return got == want
	}
	return *got == *want
}

func sameAPIDangerousPackingGroup(got, want *v1.DangerousPackingGroup) bool {
	if got == nil || want == nil {
		return got == want
	}
	return *got == *want
}
