package service

import "testing"

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
