package biz

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func economicTestAmount(s string) decimal.Decimal { return decimal.RequireFromString(s) }
func TestCommissionEconomicProjection(t *testing.T) {
	for _, scenario := range []struct{ name, revenue, cost, received, target string }{{"部分回款优惠", "800", "400", "500", "25"}, {"全收优惠封顶", "900", "100", "1000", "80"}, {"亏损", "800", "900", "500", "0"}} {
		t.Run(scenario.name, func(t *testing.T) {
			order, party, root, source := uuid.New(), uuid.New(), uuid.New(), uuid.New()
			old := &OrderFee{ID: root, OrderID: order, Direction: OrderFeeReceivable, EconomicEffect: OrderFeeEconomicIncrease, SettlementPartyID: party, Currency: "CNY", BaseCurrency: "CNY", TotalAmount: economicTestAmount("1000"), BaseCurrencyAmount: economicTestAmount("1000")}
			replacement := *old
			replacement.ID = uuid.New()
			replacement.SourceFeeID = &root
			replacement.CorrectionRequestID = &source
			replacement.TotalAmount = economicTestAmount(scenario.revenue)
			replacement.BaseCurrencyAmount = replacement.TotalAmount
			cost := &OrderFee{ID: uuid.New(), OrderID: order, Direction: OrderFeePayable, BaseCurrency: "CNY", BaseCurrencyAmount: economicTestAmount(scenario.cost)}
			line := &FinanceBillLineSettlement{FeeID: root, Direction: OrderFeeReceivable, Contributions: []FinanceSettlementContribution{{SourceType: "VERIFICATION", SourceID: source, AllocationID: uuid.New(), Amount: economicTestAmount(scenario.received), EffectiveAt: time.Now()}}}
			p, err := ProjectCommissionEconomics(order, 2, []*OrderFee{old, cost}, []*OrderFee{&replacement, cost}, nil, []*FinanceBillLineSettlement{line})
			if err != nil {
				t.Fatal(err)
			}
			amount, err := CommissionEconomicTarget(CommissionCalculationVersion, CommissionBasisRealizedProfit, p.SourceRevenue("VERIFICATION", source), p.Receivable, p.Payable, economicTestAmount("10"))
			if err != nil || !amount.Equal(economicTestAmount(scenario.target)) {
				t.Fatalf("目标%s,得到%s: %v", scenario.target, amount, err)
			}
		})
	}
}
func TestCommissionEconomicFIFOAndRefundReplay(t *testing.T) {
	order, party, root := uuid.New(), uuid.New(), uuid.New()
	at := time.Date(2026, 10, 13, 1, 0, 0, 0, time.UTC)
	old := &OrderFee{ID: root, OrderID: order, Direction: OrderFeeReceivable, SettlementPartyID: party, Currency: "CNY", BaseCurrency: "CNY", TotalAmount: economicTestAmount("1000"), BaseCurrencyAmount: economicTestAmount("1000")}
	correction := uuid.New()
	replacement := *old
	replacement.ID = uuid.New()
	replacement.SourceFeeID = &root
	replacement.CorrectionRequestID = &correction
	replacement.TotalAmount = economicTestAmount("900")
	replacement.BaseCurrencyAmount = replacement.TotalAmount
	reversal := *old
	reversal.ID = uuid.New()
	reversal.SourceFeeID = &root
	reversal.EconomicEffect = OrderFeeEconomicDecrease
	first, second, refund := uuid.New(), uuid.New(), uuid.New()
	lines := []*FinanceBillLineSettlement{{BillLineID: uuid.New(), FeeID: root, Direction: OrderFeeReceivable, Contributions: []FinanceSettlementContribution{{SourceType: "VERIFICATION", SourceID: first, AllocationID: uuid.New(), Amount: economicTestAmount("500"), EffectiveAt: at}, {SourceType: "VERIFICATION", SourceID: second, AllocationID: uuid.New(), Amount: economicTestAmount("500"), EffectiveAt: at.Add(time.Hour)}, {SourceType: "NETTING", SourceID: uuid.New(), AllocationID: uuid.New(), Amount: economicTestAmount("500"), EffectiveAt: at.Add(2 * time.Hour), IsCorrectionSettlement: true}}}, {BillLineID: uuid.New(), FeeID: reversal.ID, Direction: OrderFeePayable}}
	project := func() (*CommissionEconomicOrderProjection, error) {
		return ProjectCommissionEconomics(order, 2, []*OrderFee{old, &reversal}, []*OrderFee{&replacement}, nil, lines)
	}
	p, err := project()
	if err != nil {
		t.Fatal(err)
	}
	if !p.SourceRevenue("VERIFICATION", first).Equal(economicTestAmount("500")) || !p.SourceRevenue("VERIFICATION", second).Equal(economicTestAmount("400")) {
		t.Fatalf("FIFO不守恒: %+v", p.Quotas)
	}
	lines[1].Contributions = []FinanceSettlementContribution{{SourceType: "VERIFICATION", SourceID: refund, AllocationID: uuid.New(), Amount: economicTestAmount("100"), EffectiveAt: at.Add(3 * time.Hour)}}
	p, err = project()
	if err != nil {
		t.Fatal(err)
	}
	if !p.SourceRevenue("VERIFICATION", first).Equal(economicTestAmount("450")) || !p.SourceRevenue("VERIFICATION", second).Equal(economicTestAmount("450")) {
		t.Fatalf("退款应按原余额比例且总900: %+v", p.Quotas)
	}
	reversed := at.Add(4 * time.Hour)
	lines[1].Contributions[0].ReversedAt = &reversed
	p, err = project()
	if err != nil {
		t.Fatal(err)
	}
	if !p.SourceRevenue("VERIFICATION", first).Equal(economicTestAmount("500")) || !p.SourceRevenue("VERIFICATION", second).Equal(economicTestAmount("400")) {
		t.Fatal("反退款必须恢复原份额")
	}
	lines[1].Contributions[0].ReversedAt = nil
	lines[0].Contributions[0].ReversedAt = &reversed
	if _, err = project(); !errors.Is(err, ErrFinanceSettlementProjection) {
		t.Fatalf("有退款依赖的原来源反转必须拒绝: %v", err)
	}
	lines[0].Contributions[0].ReversedAt = nil
	replacement.SettlementPartyID = uuid.New()
	p, err = project()
	if err != nil || p.SourceRevenue("VERIFICATION", first).IsPositive() {
		t.Fatalf("换客户不继承实收: %v", err)
	}
	replacement.SettlementPartyID = party
	replacement.Currency = "USD"
	p, err = project()
	if err != nil || p.SourceRevenue("VERIFICATION", first).IsPositive() {
		t.Fatalf("换币不继承实收: %v", err)
	}
}
func TestCommissionSystemAdjustmentNetRebuild(t *testing.T) {
	for _, s := range []struct {
		name                                           string
		paid, pending, other, target, desired, rebuild string
	}{{"未执行全扣改小", "0", "-100", "0", "90", "-10", "-10"}, {"已追回后补发", "-100", "0", "0", "90", "-10", "90"}, {"人工不被抹掉", "-20", "0", "-80", "90", "-10", "10"}, {"连续成本封顶", "-100", "0", "0", "0", "-100", "0"}, {"不需要虚假往返", "0", "-10", "0", "90", "-10", "0"}} {
		t.Run(s.name, func(t *testing.T) {
			desired, rebuild, _ := PlanCommissionSystemAdjustment(economicTestAmount("100"), economicTestAmount(s.target), economicTestAmount(s.paid), economicTestAmount(s.pending), economicTestAmount(s.other), economicTestAmount("100").Add(economicTestAmount(s.other)), decimal.Zero, decimal.Zero)
			if !desired.Equal(economicTestAmount(s.desired)) || !rebuild.Equal(economicTestAmount(s.rebuild)) {
				t.Fatalf("desired=%s,rebuild=%s", desired, rebuild)
			}
		})
	}
	if _, err := CommissionEconomicTarget("UNKNOWN", CommissionBasisRealizedProfit, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero); !errors.Is(err, ErrCommissionSnapshotUnavailable) {
		t.Fatal("未知版本不能静默放行")
	}
}

func TestCommissionEconomicTinyRefundConservation(t *testing.T) {
	order, party, root := uuid.New(), uuid.New(), uuid.New()
	at := time.Now()
	tick := decimal.RequireFromString("0.00000001")
	normal := &OrderFee{ID: root, OrderID: order, Direction: OrderFeeReceivable, SettlementPartyID: party, Currency: "CNY", BaseCurrency: "CNY", TotalAmount: tick.Mul(decimal.NewFromInt(10)), BaseCurrencyAmount: tick.Mul(decimal.NewFromInt(10))}
	reversal := *normal
	reversal.ID = uuid.New()
	reversal.SourceFeeID = &root
	reversal.EconomicEffect = OrderFeeEconomicDecrease
	line := &FinanceBillLineSettlement{FeeID: root, Direction: OrderFeeReceivable}
	for i := 0; i < 10; i++ {
		line.Contributions = append(line.Contributions, FinanceSettlementContribution{SourceType: "VERIFICATION", SourceID: uuid.New(), AllocationID: uuid.New(), Amount: tick, EffectiveAt: at.Add(time.Duration(i) * time.Second)})
	}
	refund := &FinanceBillLineSettlement{FeeID: reversal.ID, Direction: OrderFeePayable, Contributions: []FinanceSettlementContribution{{SourceType: "VERIFICATION", SourceID: uuid.New(), AllocationID: uuid.New(), Amount: tick.Mul(decimal.NewFromInt(6)), EffectiveAt: at.Add(time.Hour)}}}
	p, err := ProjectCommissionEconomics(order, 1, []*OrderFee{normal, &reversal}, []*OrderFee{normal}, nil, []*FinanceBillLineSettlement{line, refund})
	if err != nil {
		t.Fatal(err)
	}
	total := decimal.Zero
	for _, q := range p.Quotas {
		if q.NativeNetAmount.IsNegative() {
			t.Fatal("退款不能负余额")
		}
		total = total.Add(q.BaseQuota)
	}
	if !total.Equal(tick.Mul(decimal.NewFromInt(4))) {
		t.Fatalf("微额退款后总额应4tick,得到%s", total)
	}
}

func TestCommissionEconomicBaseQuotaRoundingConservation(t *testing.T) {
	for _, base := range []string{"0.00000006", "60.00000006"} {
		t.Run(base, func(t *testing.T) {
			order, party, root := uuid.New(), uuid.New(), uuid.New()
			at := time.Now()
			normal := &OrderFee{ID: root, OrderID: order, Direction: OrderFeeReceivable, SettlementPartyID: party, Currency: "USD", BaseCurrency: "CNY", TotalAmount: decimal.NewFromInt(10), BaseCurrencyAmount: economicTestAmount(base)}
			line := &FinanceBillLineSettlement{FeeID: root, Direction: OrderFeeReceivable}
			for i := 0; i < 10; i++ {
				line.Contributions = append(line.Contributions, FinanceSettlementContribution{SourceType: "VERIFICATION", SourceID: uuid.New(), AllocationID: uuid.New(), Amount: decimal.NewFromInt(1), EffectiveAt: at.Add(time.Duration(i) * time.Second)})
			}
			p, err := ProjectCommissionEconomics(order, 1, []*OrderFee{normal}, []*OrderFee{normal}, nil, []*FinanceBillLineSettlement{line})
			if err != nil {
				t.Fatal(err)
			}
			total := decimal.Zero
			for i, q := range p.Quotas {
				if q.BaseQuota.IsNegative() {
					t.Fatalf("来源%d出现负本位币配额%s", i, q.BaseQuota)
				}
				expected := economicTestAmount(base).Div(decimal.NewFromInt(10)).Truncate(8)
				if i < 6 {
					expected = expected.Add(economicTestAmount("0.00000001"))
				}
				if !q.BaseQuota.Equal(expected) {
					t.Fatalf("来源%d稳定余数应%s，得到%s", i, expected, q.BaseQuota)
				}
				total = total.Add(q.BaseQuota)
			}
			if !total.Equal(economicTestAmount(base)) {
				t.Fatalf("来源本位币合计%s，经济基数%s", total, base)
			}
		})
	}
}
