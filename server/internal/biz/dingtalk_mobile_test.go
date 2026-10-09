package biz

import "testing"

func TestNormalizeDingTalkMobileStripsPrefixesAndSeparators(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{raw: "13800138000", want: "13800138000"},
		{raw: " 138 0013-8000 ", want: "13800138000"},
		{raw: "+8613800138000", want: "13800138000"},
		{raw: "008613800138000", want: "13800138000"},
		{raw: "+86 138 0013 8000", want: "13800138000"},
		// 裸 86 开头的 11 位号码是合法手机号，不能被当国际前缀剥离。
		{raw: "8613813800", want: "8613813800"},
	}
	for _, test := range tests {
		if got := NormalizeDingTalkMobile(test.raw); got != test.want {
			t.Fatalf("NormalizeDingTalkMobile(%q) = %q, want %q", test.raw, got, test.want)
		}
	}
}

func TestMaskDingTalkMobileNeverLeaksFullNumber(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{raw: "13800138000", want: "138****8000"},
		{raw: "+8613800138000", want: "138****8000"},
		{raw: "12345", want: "12****"},
		{raw: "ab", want: "****"},
	}
	for _, test := range tests {
		if got := MaskDingTalkMobile(test.raw); got != test.want {
			t.Fatalf("MaskDingTalkMobile(%q) = %q, want %q", test.raw, got, test.want)
		}
		if got := MaskDingTalkMobile(test.raw); got == NormalizeDingTalkMobile(test.raw) && len(got) > 4 {
			t.Fatalf("MaskDingTalkMobile(%q) 泄漏完整号码: %q", test.raw, got)
		}
	}
}
