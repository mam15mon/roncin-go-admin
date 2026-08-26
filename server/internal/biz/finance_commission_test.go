package biz

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestCalculateCommissionAmount(t *testing.T) {
	tests := []struct {
		name    string
		revenue string
		profit  string
		rate    string
		basis   CommissionCalculationBasis
		want    string
		wantErr bool
	}{
		{name: "按已实现毛利计提", revenue: "1000", profit: "250", rate: "10", basis: CommissionBasisRealizedProfit, want: "25.00000000"},
		{name: "按已实现收入计提", revenue: "1000", profit: "250", rate: "1.5", basis: CommissionBasisRealizedRevenue, want: "15.00000000"},
		{name: "亏损毛利按零计提", revenue: "1000", profit: "-50", rate: "10", basis: CommissionBasisRealizedProfit, want: "0.00000000"},
		{name: "拒绝未知口径", revenue: "1000", profit: "250", rate: "10", basis: "UNKNOWN", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateCommissionAmount(decimal.RequireFromString(tt.revenue), decimal.RequireFromString(tt.profit), decimal.RequireFromString(tt.rate), tt.basis)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CalculateCommissionAmount() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got.StringFixed(8) != tt.want {
				t.Fatalf("CalculateCommissionAmount() = %s, want %s", got.StringFixed(8), tt.want)
			}
		})
	}
}
