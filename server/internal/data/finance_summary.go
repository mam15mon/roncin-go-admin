package data

import (
	"sort"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
)

func financeBaseCurrencyAmountItems(values map[string]*biz.FinanceBaseCurrencyAmount) []biz.FinanceBaseCurrencyAmount {
	currencies := make([]string, 0, len(values))
	for currency := range values {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	items := make([]biz.FinanceBaseCurrencyAmount, 0, len(currencies))
	for _, currency := range currencies {
		items = append(items, *values[currency])
	}
	return items
}

func financeBaseCurrencyAmountFor(values map[string]*biz.FinanceBaseCurrencyAmount, currency string) *biz.FinanceBaseCurrencyAmount {
	if item := values[currency]; item != nil {
		return item
	}
	item := &biz.FinanceBaseCurrencyAmount{BaseCurrency: currency, ReceivableBaseAmount: decimal.Zero, PayableBaseAmount: decimal.Zero, UnverifiedBaseAmount: decimal.Zero}
	values[currency] = item
	return item
}
