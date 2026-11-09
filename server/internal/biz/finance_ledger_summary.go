package biz

import "github.com/shopspring/decimal"

// FinanceLedgerMatchedSummary 与列表完整筛选范围一致，不隐含有效业务状态过滤。
type FinanceLedgerMatchedSummary struct {
	Count   int64
	Amounts []FinanceLedgerMatchedAmount
}

type FinanceLedgerMatchedAmount struct {
	Direction        string
	Currency         string
	Amount           decimal.Decimal
	BaseCurrency     *string
	BaseAmount       *decimal.Decimal
	UnconvertedCount int64
}
