package data

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func TestDecimalOf(t *testing.T) {
	value, err := decimalOf("123.4500")
	if err != nil {
		t.Fatalf("decimalOf() error = %v", err)
	}
	if !value.Equal(decimal.RequireFromString("123.45")) {
		t.Fatalf("decimalOf() = %s, want 123.45", value)
	}

	_, err = decimalOf("not-a-decimal")
	if err == nil {
		t.Fatal("decimalOf() error = nil, want parse error")
	}
	if !strings.Contains(err.Error(), "parse stored decimal") {
		t.Fatalf("decimalOf() error = %q, want storage context", err)
	}
}
