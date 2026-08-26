package biz

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestSameFinanceCashflowIntent(t *testing.T) {
	partnerID := uuid.New()
	counterpartyAccount := "客户账户"
	bankReferenceNo := "BANK-20260827-001"
	note := "同一笔收款"
	old := &FinanceCashflow{
		Direction:           OrderFeeReceivable,
		SettlementPartyID:   partnerID,
		Currency:            "CNY",
		Amount:              decimal.RequireFromString("100.00000000"),
		ExchangeRate:        decimal.RequireFromString("1.00000000"),
		BaseCurrency:        "CNY",
		TransactionDate:     "2026-08-27",
		OurAccount:          "基本户",
		PaymentMethod:       "银行转账",
		CounterpartyAccount: &counterpartyAccount,
		BankReferenceNo:     &bankReferenceNo,
		Note:                &note,
	}
	requested := CreateFinanceCashflowInput{
		Direction:           OrderFeeReceivable,
		SettlementPartyID:   partnerID,
		Currency:            "CNY",
		Amount:              decimal.RequireFromString("100"),
		ExchangeRate:        decimal.RequireFromString("1"),
		BaseCurrency:        "CNY",
		TransactionDate:     "2026-08-27",
		OurAccount:          "基本户",
		PaymentMethod:       "银行转账",
		CounterpartyAccount: &counterpartyAccount,
		BankReferenceNo:     &bankReferenceNo,
		Note:                &note,
	}
	if !sameFinanceCashflowIntent(old, requested) {
		t.Fatal("等值金额和相同字段应识别为同一幂等请求")
	}
	requested.Amount = decimal.RequireFromString("100.01")
	if sameFinanceCashflowIntent(old, requested) {
		t.Fatal("金额不同的请求不应复用已有资金流水")
	}
}
