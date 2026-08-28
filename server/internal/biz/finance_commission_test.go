package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type commissionAdjustmentRepoStub struct {
	CommissionRepo
	created         *FinanceCommissionAdjustment
	employeeOptions SelectorListOptions
}

func (r *commissionAdjustmentRepoStub) ListEmployees(_ context.Context, _ uuid.UUID, options SelectorListOptions) (*PagedList[*CommissionEmployeeOption], error) {
	r.employeeOptions = options
	return &PagedList[*CommissionEmployeeOption]{Page: options.Page, PageSize: options.PageSize}, nil
}

func (r *commissionAdjustmentRepoStub) GetAdjustmentByKey(context.Context, uuid.UUID, string) (*FinanceCommissionAdjustment, error) {
	return nil, nil
}

func (r *commissionAdjustmentRepoStub) CreateAdjustment(_ context.Context, _ uuid.UUID, _ uuid.UUID, item *FinanceCommissionAdjustment, _ *AuditEvent) (*FinanceCommissionAdjustment, error) {
	r.created = item
	return item, nil
}

func TestCalculateCommissionAmount(t *testing.T) {
	tests := []struct {
		name       string
		revenue    string
		profit     string
		rate       string
		basis      CommissionCalculationBasis
		wantBase   string
		wantAmount string
		wantErr    bool
	}{
		{name: "按已实现毛利计提", revenue: "1000", profit: "250", rate: "10", basis: CommissionBasisRealizedProfit, wantBase: "250.00000000", wantAmount: "25.00000000"},
		{name: "按已实现收入计提", revenue: "1000", profit: "250", rate: "1.5", basis: CommissionBasisRealizedRevenue, wantBase: "1000.00000000", wantAmount: "15.00000000"},
		{name: "亏损毛利按零计提", revenue: "1000", profit: "-50", rate: "10", basis: CommissionBasisRealizedProfit, wantBase: "0.00000000", wantAmount: "0.00000000"},
		{name: "拒绝未知口径", revenue: "1000", profit: "250", rate: "10", basis: "UNKNOWN", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, amount, err := CalculateCommissionAmount(decimal.RequireFromString(tt.revenue), decimal.RequireFromString(tt.profit), decimal.RequireFromString(tt.rate), tt.basis)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CalculateCommissionAmount() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && (base.StringFixed(8) != tt.wantBase || amount.StringFixed(8) != tt.wantAmount) {
				t.Fatalf("CalculateCommissionAmount() base=%s amount=%s", base.StringFixed(8), amount.StringFixed(8))
			}
		})
	}
}

func TestListCommissionEmployeesNormalizesAndValidatesPagination(t *testing.T) {
	repo := &commissionAdjustmentRepoStub{}
	usecase := NewCommissionUsecase(repo, nil)
	_, err := usecase.ListEmployees(context.Background(), uuid.New(), SelectorListOptions{
		Page: 1, PageSize: MaxListPageSize, Keyword: "  张三  ",
	})
	if err != nil {
		t.Fatalf("查询提成员工失败: %v", err)
	}
	if repo.employeeOptions.Keyword != "张三" || repo.employeeOptions.PageSize != MaxListPageSize {
		t.Fatalf("提成员工查询参数未正确归一化: %#v", repo.employeeOptions)
	}
	if _, err := usecase.ListEmployees(context.Background(), uuid.New(), SelectorListOptions{Page: 1, PageSize: MaxListPageSize + 1}); err != ErrCommissionInvalid {
		t.Fatalf("超出统一分页上限的错误 = %v，期望 ErrCommissionInvalid", err)
	}
}

func TestCalculateCommissionLine(t *testing.T) {
	t.Run("按核销比例分摊订单成本", func(t *testing.T) {
		cost, profit, base, amount, err := CalculateCommissionLine(
			decimal.RequireFromString("500"),
			decimal.RequireFromString("1000"),
			decimal.RequireFromString("600"),
			decimal.RequireFromString("10"),
			CommissionBasisRealizedProfit,
		)
		if err != nil {
			t.Fatalf("CalculateCommissionLine() error = %v", err)
		}
		if cost.StringFixed(8) != "300.00000000" || profit.StringFixed(8) != "200.00000000" || base.StringFixed(8) != "200.00000000" || amount.StringFixed(8) != "20.00000000" {
			t.Fatalf("计算结果不符: cost=%s profit=%s base=%s amount=%s", cost.StringFixed(8), profit.StringFixed(8), base.StringFixed(8), amount.StringFixed(8))
		}
	})

	t.Run("亏损订单独立按零计提", func(t *testing.T) {
		_, lossProfit, lossBase, lossAmount, err := CalculateCommissionLine(
			decimal.RequireFromString("500"),
			decimal.RequireFromString("1000"),
			decimal.RequireFromString("1200"),
			decimal.RequireFromString("10"),
			CommissionBasisRealizedProfit,
		)
		if err != nil {
			t.Fatalf("CalculateCommissionLine() error = %v", err)
		}
		_, gainProfit, gainBase, gainAmount, err := CalculateCommissionLine(
			decimal.RequireFromString("500"),
			decimal.RequireFromString("1000"),
			decimal.RequireFromString("200"),
			decimal.RequireFromString("10"),
			CommissionBasisRealizedProfit,
		)
		if err != nil {
			t.Fatalf("CalculateCommissionLine() error = %v", err)
		}
		if lossProfit.StringFixed(8) != "-100.00000000" || lossBase.StringFixed(8) != "0.00000000" || lossAmount.StringFixed(8) != "0.00000000" {
			t.Fatalf("亏损订单应保留负毛利且按零计提: profit=%s base=%s amount=%s", lossProfit.StringFixed(8), lossBase.StringFixed(8), lossAmount.StringFixed(8))
		}
		if gainProfit.StringFixed(8) != "400.00000000" || gainBase.StringFixed(8) != "400.00000000" || gainAmount.StringFixed(8) != "40.00000000" {
			t.Fatalf("盈利订单计算不符: profit=%s base=%s amount=%s", gainProfit.StringFixed(8), gainBase.StringFixed(8), gainAmount.StringFixed(8))
		}
		if lossAmount.Add(gainAmount).StringFixed(8) != "40.00000000" {
			t.Fatalf("逐订单计提汇总金额不符: %s", lossAmount.Add(gainAmount).StringFixed(8))
		}
	})

	t.Run("拒绝无应收基数", func(t *testing.T) {
		_, _, _, _, err := CalculateCommissionLine(decimal.NewFromInt(1), decimal.Zero, decimal.Zero, decimal.NewFromInt(10), CommissionBasisRealizedProfit)
		if err != ErrCommissionSource {
			t.Fatalf("error = %v, want %v", err, ErrCommissionSource)
		}
	})
}

func TestNormalizeCommissionRuleInputSupportsCustomerRoles(t *testing.T) {
	for _, role := range []CommissionPersonnelRole{CommissionRoleSales, CommissionRoleOperator, CommissionRoleCustomerService} {
		t.Run(string(role), func(t *testing.T) {
			input, err := normalizeCommissionRuleInput(CreateCommissionRuleInput{
				Name: "客户角色提成", PersonnelRole: role, CalculationBasis: CommissionBasisRealizedProfit,
				RatePercent: decimal.RequireFromString("12.5"), Enabled: true,
			})
			if err != nil {
				t.Fatalf("normalizeCommissionRuleInput() error = %v", err)
			}
			if input.PersonnelRole != role || input.CalculationBasis != CommissionBasisRealizedProfit {
				t.Fatalf("规则归一化结果不符: %#v", input)
			}
		})
	}

	if _, err := normalizeCommissionRuleInput(CreateCommissionRuleInput{
		Name: "不支持角色", PersonnelRole: "FINANCE", CalculationBasis: CommissionBasisRealizedProfit,
		RatePercent: decimal.NewFromInt(10), Enabled: true,
	}); err != ErrCommissionRuleInvalid {
		t.Fatalf("财务角色当前不应参与提成，error = %v", err)
	}
}

func TestCreateCommissionAdjustment(t *testing.T) {
	org, actor, commissionID, orderID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	t.Run("创建逐订单增提草稿", func(t *testing.T) {
		repo := &commissionAdjustmentRepoStub{}
		usecase := NewCommissionUsecase(repo, nil)
		item, err := usecase.CreateAdjustment(context.Background(), org, actor, CreateCommissionAdjustmentInput{
			CommissionID: commissionID, OrderID: orderID, Direction: CommissionAdjustmentIncrease,
			Amount: decimal.RequireFromString("1.25000000"), Reason: "补提", IdempotencyKey: "adjustment-test",
		})
		if err != nil {
			t.Fatalf("CreateAdjustment() error = %v", err)
		}
		if item.Status != CommissionDraft || item.SourceType != CommissionAdjustmentSourceManual || item.Direction != CommissionAdjustmentIncrease || item.Amount.StringFixed(8) != "1.25000000" {
			t.Fatalf("调整草稿不符: %#v", item)
		}
	})

	for _, tt := range []struct {
		name  string
		input CreateCommissionAdjustmentInput
	}{
		{name: "拒绝零金额", input: CreateCommissionAdjustmentInput{CommissionID: commissionID, OrderID: orderID, Direction: CommissionAdjustmentIncrease, Amount: decimal.Zero, Reason: "补提", IdempotencyKey: "zero"}},
		{name: "拒绝未知方向", input: CreateCommissionAdjustmentInput{CommissionID: commissionID, OrderID: orderID, Direction: "UNKNOWN", Amount: decimal.NewFromInt(1), Reason: "补提", IdempotencyKey: "direction"}},
		{name: "拒绝空原因", input: CreateCommissionAdjustmentInput{CommissionID: commissionID, OrderID: orderID, Direction: CommissionAdjustmentDecrease, Amount: decimal.NewFromInt(1), IdempotencyKey: "reason"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			usecase := NewCommissionUsecase(&commissionAdjustmentRepoStub{}, nil)
			if _, err := usecase.CreateAdjustment(context.Background(), org, actor, tt.input); err != ErrCommissionAdjustmentInvalid {
				t.Fatalf("error = %v, want %v", err, ErrCommissionAdjustmentInvalid)
			}
		})
	}
}

func TestPlanCommissionReversal(t *testing.T) {
	firstID, secondID := uuid.New(), uuid.New()
	lines := []CommissionReversalLine{
		{OrderID: firstID, Amount: decimal.RequireFromString("30")},
		{OrderID: secondID, Amount: decimal.RequireFromString("0")},
	}

	t.Run("未支付提成和调整直接取消", func(t *testing.T) {
		draftID, confirmedID := uuid.New(), uuid.New()
		plan, err := PlanCommissionReversal(CommissionConfirmed, decimal.NewFromInt(30), lines, []*FinanceCommissionAdjustment{
			{ID: draftID, Status: CommissionDraft},
			{ID: confirmedID, Status: CommissionConfirmed},
		})
		if err != nil {
			t.Fatalf("PlanCommissionReversal() error = %v", err)
		}
		if !plan.CancelCommission || len(plan.CancelAdjustmentIDs) != 2 || len(plan.Recoveries) != 0 {
			t.Fatalf("取消计划不符: %#v", plan)
		}
	})

	t.Run("已支付提成形成等额待追回冲减", func(t *testing.T) {
		plan, err := PlanCommissionReversal(CommissionPaid, decimal.NewFromInt(30), lines, []*FinanceCommissionAdjustment{
			{OrderID: firstID, Status: CommissionConfirmed, Direction: CommissionAdjustmentIncrease, Amount: decimal.NewFromInt(10)},
		})
		if err != nil {
			t.Fatalf("PlanCommissionReversal() error = %v", err)
		}
		if plan.CancelCommission || len(plan.Recoveries) != 1 || plan.Recoveries[0].Amount.StringFixed(8) != "40.00000000" {
			t.Fatalf("冲减计划不符: %#v", plan)
		}
	})

	t.Run("负向订单不导致超额冲减", func(t *testing.T) {
		plan, err := PlanCommissionReversal(CommissionPaid, decimal.NewFromInt(20), []CommissionReversalLine{
			{OrderID: firstID, Amount: decimal.NewFromInt(30)},
			{OrderID: secondID, Amount: decimal.NewFromInt(-10)},
		}, nil)
		if err != nil {
			t.Fatalf("PlanCommissionReversal() error = %v", err)
		}
		if len(plan.Recoveries) != 1 || plan.Recoveries[0].OrderID != firstID || plan.Recoveries[0].Amount.StringFixed(8) != "20.00000000" {
			t.Fatalf("冲减分配不符: %#v", plan.Recoveries)
		}
	})
}
