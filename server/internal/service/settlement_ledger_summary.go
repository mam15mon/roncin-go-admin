package service

import (
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func financeLedgerMatchedSummaryToAPI(summary biz.FinanceLedgerMatchedSummary) *v1.FinanceLedgerMatchedSummary {
	result := &v1.FinanceLedgerMatchedSummary{Count: summary.Count, Amounts: make([]*v1.FinanceLedgerMatchedAmount, 0, len(summary.Amounts))}
	for _, item := range summary.Amounts {
		value := &v1.FinanceLedgerMatchedAmount{Direction: item.Direction, Currency: item.Currency, Amount: item.Amount.StringFixed(8), BaseCurrency: item.BaseCurrency, UnconvertedCount: item.UnconvertedCount}
		if item.BaseAmount != nil {
			amount := item.BaseAmount.StringFixed(8)
			value.BaseAmount = &amount
		}
		result.Amounts = append(result.Amounts, value)
	}
	return result
}
