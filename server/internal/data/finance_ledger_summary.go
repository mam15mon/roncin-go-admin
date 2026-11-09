package data

import (
	"context"
	"sort"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

// 财务实体共享金额聚合列；查询范围和实体字段由各仓储的列表查询决定。
func financeLedgerMatchedAggregates(amountField, baseAmountField string) []ent.AggregateFunc {
	return []ent.AggregateFunc{
		ent.As(ent.Count(), "record_count"),
		ent.As(ent.Sum(amountField), "amount"),
		ent.As(ent.Sum(baseAmountField), "base_amount"),
		func(s *entsql.Selector) string {
			return entsql.As("COUNT(*) - COUNT("+s.C(baseAmountField)+")", "unconverted_count")
		},
	}
}

type financeLedgerMatchedRow struct {
	Direction        string  `json:"direction"`
	Currency         string  `json:"currency"`
	BaseCurrency     *string `json:"base_currency"`
	Amount           string  `json:"amount"`
	BaseAmount       *string `json:"base_amount"`
	RecordCount      int64   `json:"record_count"`
	UnconvertedCount int64   `json:"unconverted_count"`
}

// scan 接收分页前的克隆聚合查询，不加载全量实体、不引入另一套筛选条件。
func loadFinanceLedgerMatchedSummary(ctx context.Context, scan func(context.Context, any) error) (biz.FinanceLedgerMatchedSummary, error) {
	rows := make([]financeLedgerMatchedRow, 0)
	if err := scan(ctx, &rows); err != nil {
		return biz.FinanceLedgerMatchedSummary{}, err
	}
	return financeLedgerMatchedSummaryFromRows(rows)
}

func financeLedgerMatchedSummaryFromRows(rows []financeLedgerMatchedRow) (biz.FinanceLedgerMatchedSummary, error) {
	result := biz.FinanceLedgerMatchedSummary{Amounts: make([]biz.FinanceLedgerMatchedAmount, 0, len(rows))}
	for _, row := range rows {
		amount, err := decimalOf(row.Amount)
		if err != nil {
			return biz.FinanceLedgerMatchedSummary{}, err
		}
		item := biz.FinanceLedgerMatchedAmount{Direction: row.Direction, Currency: row.Currency, Amount: amount, BaseCurrency: row.BaseCurrency, UnconvertedCount: row.UnconvertedCount}
		if row.BaseAmount != nil {
			base, err := decimalOf(*row.BaseAmount)
			if err != nil {
				return biz.FinanceLedgerMatchedSummary{}, err
			}
			item.BaseAmount = &base
		}
		result.Count += row.RecordCount
		result.Amounts = append(result.Amounts, item)
	}
	sort.Slice(result.Amounts, func(i, j int) bool {
		a, b := result.Amounts[i], result.Amounts[j]
		if a.Direction != b.Direction {
			return a.Direction < b.Direction
		}
		if a.Currency != b.Currency {
			return a.Currency < b.Currency
		}
		if a.BaseCurrency == nil {
			return b.BaseCurrency != nil
		}
		return b.BaseCurrency != nil && *a.BaseCurrency < *b.BaseCurrency
	})
	return result, nil
}
