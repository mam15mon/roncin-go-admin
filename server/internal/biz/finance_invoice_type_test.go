package biz

import "testing"

func TestFinanceInvoiceTypeValidAndSpecial(t *testing.T) {
	specials := map[FinanceInvoiceType]bool{
		FinanceInvoiceDigitalNormal: false, FinanceInvoiceDigitalSpecial: true,
		FinanceInvoicePaperNormal: false, FinanceInvoicePaperSpecial: true,
		FinanceInvoiceElectronicNormal: false, FinanceInvoiceElectronicSpecial: true,
	}
	for value, special := range specials {
		if !value.Valid() || value.Special() != special {
			t.Fatalf("票种判定错误: %s", value)
		}
	}
	for _, value := range []FinanceInvoiceType{"NORMAL", "SPECIAL", "PROFORMA", ""} {
		if value.Valid() || value.Special() {
			t.Fatalf("旧值与形式发票不得被接受: %q", value)
		}
	}
}
