package data

import (
	"fmt"

	"github.com/shopspring/decimal"
)

func decimalOf(value string) (decimal.Decimal, error) {
	parsed, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, fmt.Errorf("parse stored decimal: %w", err)
	}
	return parsed, nil
}
