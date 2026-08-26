package biz

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestBuildFinanceInvoiceAggregatesConfirmedBills(t *testing.T) {
	organizationID := uuid.Must(uuid.NewV7())
	partyID := uuid.Must(uuid.NewV7())
	taxRate := decimal.RequireFromString("6")
	bills := []*FinanceBill{
		{ID: uuid.Must(uuid.NewV7()), BillNo: "BI001", Status: FinanceBillConfirmed, Direction: OrderFeeReceivable, SettlementPartyID: partyID, SettlementPartyName: "客户", Currency: "CNY", TotalAmount: decimal.RequireFromString("100"), NetAmount: decimal.RequireFromString("94"), TaxAmount: decimal.RequireFromString("6"), Lines: []*FinanceBillLine{{FeeCode: "OCEAN", FeeName: "海运费", Currency: "CNY", TotalAmount: decimal.RequireFromString("100"), NetAmount: decimal.RequireFromString("94"), TaxAmount: decimal.RequireFromString("6"), TaxRate: &taxRate, Active: true}}},
		{ID: uuid.Must(uuid.NewV7()), BillNo: "BI002", Status: FinanceBillConfirmed, Direction: OrderFeeReceivable, SettlementPartyID: partyID, SettlementPartyName: "客户", Currency: "CNY", TotalAmount: decimal.RequireFromString("0.02"), NetAmount: decimal.RequireFromString("0.0188"), TaxAmount: decimal.RequireFromString("0.0012"), Lines: []*FinanceBillLine{{FeeCode: "OCEAN", FeeName: "海运费", Currency: "CNY", TotalAmount: decimal.RequireFromString("0.02"), NetAmount: decimal.RequireFromString("0.0188"), TaxAmount: decimal.RequireFromString("0.0012"), TaxRate: &taxRate, Active: true}}},
	}
	input := CreateFinanceInvoiceInput{BillIDs: []uuid.UUID{bills[0].ID, bills[1].ID}, InvoiceType: FinanceInvoiceSpecial, IdempotencyKey: "invoice-test"}
	profile := &PartnerInvoiceProfile{ID: uuid.Must(uuid.NewV7()), OrganizationID: organizationID, PartnerID: partyID, InvoiceTitle: "客户", TaxpayerIdentificationNo: "91310000TEST", RegisteredAddress: "上海市", RegisteredPhone: "021-12345678", BankName: "测试银行", BankAccount: "62220000"}
	invoice, err := buildFinanceInvoice(organizationID, bills, profile, input)
	if err != nil {
		t.Fatalf("构建开票记录失败: %v", err)
	}
	if invoice.TotalAmount.StringFixed(8) != "100.02000000" || invoice.NetAmount.StringFixed(8) != "94.01880000" || invoice.TaxAmount.StringFixed(8) != "6.00120000" || invoice.BillCount != 2 || len(invoice.Lines) != 1 || invoice.Lines[0].SourceLineCount != 2 {
		t.Fatalf("开票汇总不正确: total=%s tax=%s count=%d", invoice.TotalAmount, invoice.TaxAmount, invoice.BillCount)
	}
}

func TestBuildFinanceInvoiceRejectsMixedParties(t *testing.T) {
	bills := []*FinanceBill{
		{ID: uuid.Must(uuid.NewV7()), Status: FinanceBillConfirmed, Direction: OrderFeeReceivable, SettlementPartyID: uuid.Must(uuid.NewV7()), Currency: "CNY"},
		{ID: uuid.Must(uuid.NewV7()), Status: FinanceBillConfirmed, Direction: OrderFeeReceivable, SettlementPartyID: uuid.Must(uuid.NewV7()), Currency: "CNY"},
	}
	organizationID := uuid.Must(uuid.NewV7())
	profile := &PartnerInvoiceProfile{ID: uuid.Must(uuid.NewV7()), OrganizationID: organizationID, PartnerID: bills[0].SettlementPartyID, InvoiceTitle: "客户", TaxpayerIdentificationNo: "91310000TEST"}
	_, err := buildFinanceInvoice(organizationID, bills, profile, CreateFinanceInvoiceInput{BillIDs: []uuid.UUID{bills[0].ID, bills[1].ID}, InvoiceType: FinanceInvoiceNormal})
	if err != ErrFinanceInvoiceBillMismatch {
		t.Fatalf("混合结算单位应被拒绝，实际 %v", err)
	}
}
