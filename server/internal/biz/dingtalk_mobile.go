package biz

import (
	"regexp"
	"strings"
	"unicode"
)

var dingTalkMobilePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)

// NormalizeDingTalkMobile 归一化国内手机号：去除空白与连字符，剥离 +86/0086
// 国际前缀后返回 11 位本体；不剥离裸 86 前缀（11 位号码可能合法地以 86 开头）。
func NormalizeDingTalkMobile(raw string) string {
	var builder strings.Builder
	for _, symbol := range strings.TrimSpace(raw) {
		if unicode.IsSpace(symbol) || symbol == '-' {
			continue
		}
		builder.WriteRune(symbol)
	}
	value := builder.String()
	for _, prefix := range []string{"+86", "0086"} {
		if strings.HasPrefix(value, prefix) {
			return strings.TrimPrefix(value, prefix)
		}
	}
	return value
}

// ValidDingTalkMobile 校验是否为合法的 11 位国内手机号。
func ValidDingTalkMobile(mobile string) bool {
	return dingTalkMobilePattern.MatchString(mobile)
}

// MaskDingTalkMobile 返回 138****1234 形式的脱敏手机号；位数不足时降低保留位数，
// 任何情况下都不输出完整号码。日志与审计 Details 只允许记录该形式。
func MaskDingTalkMobile(mobile string) string {
	mobile = NormalizeDingTalkMobile(mobile)
	digits := []rune(mobile)
	switch {
	case len(digits) >= 11:
		return string(digits[:3]) + "****" + string(digits[len(digits)-4:])
	case len(digits) >= 6:
		return string(digits[:3]) + "****" + string(digits[len(digits)-2:])
	case len(digits) >= 3:
		return string(digits[:2]) + "****"
	default:
		return "****"
	}
}
