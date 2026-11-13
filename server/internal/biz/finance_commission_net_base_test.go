package biz

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestCommissionNetBase(t *testing.T) {
	for _, c := range []struct{ name, base, net, total, want string }{
		{"含税106未税100", "106", "100", "106", "100"},
		{"零总额", "100", "0", "0", "0"},
		{"尾差八位银行家舍入", "10", "1", "3", "3.33333333"},
		{"外币汇率同比例", "742.1", "100", "106", "700.09433962"},
		{"无税", "50", "50", "50", "50"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := CommissionNetBase(economicTestAmount(c.base), economicTestAmount(c.net), economicTestAmount(c.total))
			if !got.Equal(economicTestAmount(c.want)) {
				t.Fatalf("得到%s,期望%s", got, c.want)
			}
		})
	}
}

func netTestFee(order, party uuid.UUID, dir OrderFeeDirection, total, net string) *OrderFee {
	return &OrderFee{ID: uuid.New(), OrderID: order, Direction: dir, EconomicEffect: OrderFeeEconomicIncrease, SettlementPartyID: party, Currency: "CNY", BaseCurrency: "CNY",
		TotalAmount: economicTestAmount(total), NetAmount: economicTestAmount(net), BaseCurrencyAmount: economicTestAmount(total)}
}

// PRD 样例：应收含税10600/未税10000，应付含税6360/未税6000，核销占40%，比例10%。
func TestCommissionEconomicNetProfitSample(t *testing.T) {
	order, party := uuid.New(), uuid.New()
	rec := netTestFee(order, party, OrderFeeReceivable, "10600", "10000")
	pay := netTestFee(order, party, OrderFeePayable, "6360", "6000")
	src := uuid.New()
	line := &FinanceBillLineSettlement{FeeID: rec.ID, Direction: OrderFeeReceivable, Contributions: []FinanceSettlementContribution{{SourceType: "VERIFICATION", SourceID: src, AllocationID: uuid.New(), Amount: economicTestAmount("4240"), EffectiveAt: time.Now()}}}
	p, err := ProjectCommissionEconomics(order, 1, []*OrderFee{rec, pay}, []*OrderFee{rec, pay}, nil, []*FinanceBillLineSettlement{line})
	if err != nil {
		t.Fatal(err)
	}
	if !p.Receivable.Equal(economicTestAmount("10000")) || !p.Payable.Equal(economicTestAmount("6000")) {
		t.Fatalf("应收应付应为未税: %s/%s", p.Receivable, p.Payable)
	}
	realized := p.SourceRevenue("VERIFICATION", src)
	if !realized.Equal(economicTestAmount("4000")) {
		t.Fatalf("已实现未税收入应4000,得到%s", realized)
	}
	cost, profit, _, amount, err := CalculateCommissionLine(realized, p.Receivable, p.Payable, economicTestAmount("10"), CommissionBasisRealizedProfit)
	if err != nil || !cost.Equal(economicTestAmount("2400")) || !profit.Equal(economicTestAmount("1600")) || !amount.Equal(economicTestAmount("160")) {
		t.Fatalf("成本%s 利润%s 提成%s: %v", cost, profit, amount, err)
	}
	// 负利润取0。
	pay2 := netTestFee(order, party, OrderFeePayable, "12720", "12000")
	p2, err := ProjectCommissionEconomics(order, 1, []*OrderFee{rec, pay2}, []*OrderFee{rec, pay2}, nil, []*FinanceBillLineSettlement{line})
	if err != nil {
		t.Fatal(err)
	}
	amount, err = CommissionEconomicTarget(CommissionCalculationVersion, CommissionBasisRealizedProfit, p2.SourceRevenue("VERIFICATION", src), p2.Receivable, p2.Payable, economicTestAmount("10"))
	if err != nil || !amount.IsZero() {
		t.Fatalf("负利润应0,得到%s: %v", amount, err)
	}
	// 混合税率：6%与0%应收。
	r6 := netTestFee(order, party, OrderFeeReceivable, "106", "100")
	r0 := netTestFee(order, party, OrderFeeReceivable, "50", "50")
	p3, err := ProjectCommissionEconomics(order, 1, []*OrderFee{r6, r0}, []*OrderFee{r6, r0}, nil, nil)
	if err != nil || !p3.Receivable.Equal(decimal.NewFromInt(150)) {
		t.Fatalf("混合税率应收应150,得到%s: %v", p3.Receivable, err)
	}
	// 旧版本失败关闭。
	if _, err := CommissionEconomicTarget("CUSTOMER_REALIZED_PROFIT_V3", CommissionBasisRealizedProfit, realized, p.Receivable, p.Payable, economicTestAmount("10")); err != ErrCommissionSnapshotUnavailable {
		t.Fatalf("V3应失败关闭: %v", err)
	}
}
