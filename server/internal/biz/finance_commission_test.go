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

func TestCalculateCommissionLine(t *testing.T) {
	t.Run("按回款比例分摊订单成本", func(t *testing.T) {
		cost, profit, amount, err := CalculateCommissionLine(
			decimal.RequireFromString("500"),
			decimal.RequireFromString("1000"),
			decimal.RequireFromString("600"),
			decimal.RequireFromString("10"),
			CommissionBasisRealizedProfit,
		)
		if err != nil {
			t.Fatalf("CalculateCommissionLine() error = %v", err)
		}
		if cost.StringFixed(8) != "300.00000000" || profit.StringFixed(8) != "200.00000000" || amount.StringFixed(8) != "20.00000000" {
			t.Fatalf("计算结果不符: cost=%s profit=%s amount=%s", cost.StringFixed(8), profit.StringFixed(8), amount.StringFixed(8))
		}
	})

	t.Run("亏损订单独立按零计提", func(t *testing.T) {
		_, lossProfit, lossAmount, err := CalculateCommissionLine(
			decimal.RequireFromString("500"),
			decimal.RequireFromString("1000"),
			decimal.RequireFromString("1200"),
			decimal.RequireFromString("10"),
			CommissionBasisRealizedProfit,
		)
		if err != nil {
			t.Fatalf("CalculateCommissionLine() error = %v", err)
		}
		_, gainProfit, gainAmount, err := CalculateCommissionLine(
			decimal.RequireFromString("500"),
			decimal.RequireFromString("1000"),
			decimal.RequireFromString("200"),
			decimal.RequireFromString("10"),
			CommissionBasisRealizedProfit,
		)
		if err != nil {
			t.Fatalf("CalculateCommissionLine() error = %v", err)
		}
		if lossProfit.StringFixed(8) != "-100.00000000" || lossAmount.StringFixed(8) != "0.00000000" {
			t.Fatalf("亏损订单应保留负毛利且按零计提: profit=%s amount=%s", lossProfit.StringFixed(8), lossAmount.StringFixed(8))
		}
		if gainProfit.StringFixed(8) != "400.00000000" || gainAmount.StringFixed(8) != "40.00000000" {
			t.Fatalf("盈利订单计算不符: profit=%s amount=%s", gainProfit.StringFixed(8), gainAmount.StringFixed(8))
		}
		if lossAmount.Add(gainAmount).StringFixed(8) != "40.00000000" {
			t.Fatalf("逐订单计提汇总金额不符: %s", lossAmount.Add(gainAmount).StringFixed(8))
		}
	})

	t.Run("拒绝无应收基数", func(t *testing.T) {
		_, _, _, err := CalculateCommissionLine(decimal.NewFromInt(1), decimal.Zero, decimal.Zero, decimal.NewFromInt(10), CommissionBasisRealizedProfit)
		if err != ErrCommissionSource {
			t.Fatalf("error = %v, want %v", err, ErrCommissionSource)
		}
	})
}
