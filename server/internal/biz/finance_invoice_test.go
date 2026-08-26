package biz

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestBuildFinanceInvoiceAggregatesConfirmedBills(t *testing.T) {
	partyID := uuid.Must(uuid.NewV7())
	bills := []*FinanceBill{
		{ID: uuid.Must(uuid.NewV7()), BillNo: "BI001", Status: FinanceBillConfirmed, Direction: OrderFeeReceivable, SettlementPartyID: partyID, SettlementPartyName: "客户", Currency: "CNY", TotalAmount: decimal.RequireFromString("100"), TaxAmount: decimal.RequireFromString("6")},
		{ID: uuid.Must(uuid.NewV7()), BillNo: "BI002", Status: FinanceBillConfirmed, Direction: OrderFeeReceivable, SettlementPartyID: partyID, SettlementPartyName: "客户", Currency: "CNY", TotalAmount: decimal.RequireFromString("0.02"), TaxAmount: decimal.RequireFromString("0.0012")},
	}
	input := CreateFinanceInvoiceInput{BillIDs: []uuid.UUID{bills[0].ID, bills[1].ID}, InvoiceType: FinanceInvoiceSpecial, IdempotencyKey: "invoice-test"}
	invoice, err := buildFinanceInvoice(uuid.Must(uuid.NewV7()), bills, input)
	if err != nil {
		t.Fatalf("构建开票记录失败: %v", err)
	}
	if invoice.TotalAmount.StringFixed(8) != "100.02000000" || invoice.TaxAmount.StringFixed(8) != "6.00120000" || invoice.BillCount != 2 {
		t.Fatalf("开票汇总不正确: total=%s tax=%s count=%d", invoice.TotalAmount, invoice.TaxAmount, invoice.BillCount)
	}
}

func TestBuildFinanceInvoiceRejectsMixedParties(t *testing.T) {
	bills := []*FinanceBill{
		{ID: uuid.Must(uuid.NewV7()), Status: FinanceBillConfirmed, Direction: OrderFeeReceivable, SettlementPartyID: uuid.Must(uuid.NewV7()), Currency: "CNY"},
		{ID: uuid.Must(uuid.NewV7()), Status: FinanceBillConfirmed, Direction: OrderFeeReceivable, SettlementPartyID: uuid.Must(uuid.NewV7()), Currency: "CNY"},
	}
	_, err := buildFinanceInvoice(uuid.Must(uuid.NewV7()), bills, CreateFinanceInvoiceInput{BillIDs: []uuid.UUID{bills[0].ID, bills[1].ID}, InvoiceType: FinanceInvoiceNormal})
	if err != ErrFinanceInvoiceBillMismatch {
		t.Fatalf("混合结算单位应被拒绝，实际 %v", err)
	}
}
