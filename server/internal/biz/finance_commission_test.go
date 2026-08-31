package biz

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// commissionRepoStub 用于验证生成提成的用例编排，不触发数据库。
type commissionRepoStub struct {
	CommissionRepo
	generation      *CommissionGenerationContext
	preview         *CommissionCalculation
	created         *FinanceCommission
	createdSnapshot *CommissionCNYSnapshot
	createAudit     *AuditEvent
	createCalled    bool
	createErr       error
	listed          *CommissionFilter
	getResult       *FinanceCommission
}

func (s *commissionRepoStub) List(_ context.Context, _ uuid.UUID, f CommissionFilter) (*CommissionListResult, error) {
	s.listed = &f
	return &CommissionListResult{}, nil
}

func (s *commissionRepoStub) Get(context.Context, uuid.UUID, uuid.UUID) (*FinanceCommission, error) {
	return s.getResult, nil
}

func (s *commissionRepoStub) GetByKey(context.Context, uuid.UUID, string) (*FinanceCommission, error) {
	return nil, nil
}

func (s *commissionRepoStub) Preview(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) (*CommissionCalculation, error) {
	return s.preview, nil
}

func (s *commissionRepoStub) GetGenerationContext(context.Context, uuid.UUID, uuid.UUID) (*CommissionGenerationContext, error) {
	return s.generation, nil
}

func (s *commissionRepoStub) Create(_ context.Context, _ uuid.UUID, c *FinanceCommission, snapshot *CommissionCNYSnapshot, audit *AuditEvent) error {
	s.createCalled = true
	s.created = c
	s.createdSnapshot = snapshot
	s.createAudit = audit
	return s.createErr
}

// transactorStub 直接执行回调，模拟共享事务成功提交。
type transactorStub struct{}

func (transactorStub) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// orderConfigNumberStub 提供固定的单号分配结果。
type orderConfigNumberStub struct{ OrderConfigRepo }

func (orderConfigNumberStub) AllocateNumber(context.Context, uuid.UUID, DocumentType, time.Time) (*NumberRule, int64, error) {
	return &NumberRule{Prefix: "TC", DateFormat: DateFormatYYYYMMDD, SequenceLength: 6}, 1, nil
}

func newWriteOffRateStub(org uuid.UUID, baseCurrency string, resolved *ResolvedExchangeRate) *exchangeRateRepoStub {
	return &exchangeRateRepoStub{
		rateContext:   &ExchangeRateContext{OwnerOrganizationID: org, BaseCurrency: baseCurrency},
		timeStandards: []*ExchangeRateTimeStandardSetting{{RateType: WriteOffRateType, TimeStandards: []string{WriteOffTimeStandard}}},
		resolved:      resolved,
	}
}

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
	usecase := NewCommissionUsecase(repo, nil, nil, nil)
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
		usecase := NewCommissionUsecase(repo, nil, nil, nil)
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
			usecase := NewCommissionUsecase(&commissionAdjustmentRepoStub{}, nil, nil, nil)
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

func TestResolveCommissionCNYRate(t *testing.T) {
	settingID := uuid.New()

	t.Run("本位币为CNY时汇率恒为一且来源为本位币", func(t *testing.T) {
		snapshot, err := ResolveCommissionCNYRate("CNY", "2026-08-15", "2026-08-14", nil)
		if err != nil {
			t.Fatalf("ResolveCommissionCNYRate() error = %v", err)
		}
		if snapshot.CommissionDate != "2026-08-15" || snapshot.ExchangeRate.StringFixed(8) != "1.00000000" ||
			snapshot.ExchangeRateSource != CommissionCNYRateSourceBaseCurrency || snapshot.ExchangeRateDate != "2026-08-14" ||
			snapshot.ExchangeRateSettingID != nil {
			t.Fatalf("CNY 快照汇率部分不符: %#v", snapshot)
		}
		snapshot.ApplyCommissionAmount(decimal.RequireFromString("123.45"))
		if snapshot.CommissionAmount.StringFixed(8) != "123.45000000" {
			t.Fatalf("CNY 原始提成金额应等于本位币金额: %s", snapshot.CommissionAmount.StringFixed(8))
		}
	})

	t.Run("非CNY本币按倒数派生并回填来源配置", func(t *testing.T) {
		resolved := &ResolvedExchangeRate{Rate: decimal.RequireFromString("3"), RateDate: "2026-08-14", SettingID: &settingID}
		snapshot, err := ResolveCommissionCNYRate("USD", "2026-08-15", "2026-08-14", resolved)
		if err != nil {
			t.Fatalf("ResolveCommissionCNYRate() error = %v", err)
		}
		if snapshot.ExchangeRate.StringFixed(8) != "0.33333333" || snapshot.ExchangeRateSource != CommissionCNYRateSourceDerived ||
			snapshot.ExchangeRateDate != "2026-08-14" || snapshot.ExchangeRateSettingID == nil || *snapshot.ExchangeRateSettingID != settingID {
			t.Fatalf("倒数派生快照不符: %#v", snapshot)
		}
		snapshot.ApplyCommissionAmount(decimal.RequireFromString("100"))
		if snapshot.CommissionAmount.StringFixed(8) != "33.33333300" {
			t.Fatalf("CNY 提成金额公式不符: %s", snapshot.CommissionAmount.StringFixed(8))
		}
	})

	t.Run("倒数精度按八位舍入且不补差", func(t *testing.T) {
		resolved := &ResolvedExchangeRate{Rate: decimal.RequireFromString("7"), RateDate: "2026-08-14", SettingID: &settingID}
		snapshot, err := ResolveCommissionCNYRate("USD", "2026-08-15", "2026-08-14", resolved)
		if err != nil {
			t.Fatalf("ResolveCommissionCNYRate() error = %v", err)
		}
		if snapshot.ExchangeRate.StringFixed(8) != "0.14285714" {
			t.Fatalf("倒数精度应为 0.14285714: %s", snapshot.ExchangeRate.StringFixed(8))
		}
		snapshot.ApplyCommissionAmount(decimal.RequireFromString("7"))
		if snapshot.CommissionAmount.StringFixed(8) != "0.99999998" {
			t.Fatalf("金额乘法不应补差: %s", snapshot.CommissionAmount.StringFixed(8))
		}
	})

	for _, tt := range []struct {
		name          string
		baseCurrency  string
		commissionDay string
		rateDay       string
		resolved      *ResolvedExchangeRate
	}{
		{name: "拒绝非法归属日期", baseCurrency: "USD", commissionDay: "2026/08/15", rateDay: "2026-08-14", resolved: &ResolvedExchangeRate{Rate: decimal.NewFromInt(1)}},
		{name: "拒绝无效汇率日期", baseCurrency: "USD", commissionDay: "2026-08-15", rateDay: "2026-02-30", resolved: &ResolvedExchangeRate{Rate: decimal.NewFromInt(1)}},
		{name: "非CNY缺少解析结果", baseCurrency: "USD", commissionDay: "2026-08-15", rateDay: "2026-08-14"},
		{name: "拒绝非正汇率", baseCurrency: "USD", commissionDay: "2026-08-15", rateDay: "2026-08-14", resolved: &ResolvedExchangeRate{Rate: decimal.Zero, RateDate: "2026-08-14", SettingID: &settingID}},
		{name: "非CNY缺少来源配置", baseCurrency: "USD", commissionDay: "2026-08-15", rateDay: "2026-08-14", resolved: &ResolvedExchangeRate{Rate: decimal.NewFromInt(1), RateDate: "2026-08-14"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ResolveCommissionCNYRate(tt.baseCurrency, tt.commissionDay, tt.rateDay, tt.resolved); err == nil {
				t.Fatalf("ResolveCommissionCNYRate() 应返回错误")
			}
		})
	}
}

func TestCommissionUsecasePreviewResolvesCNYSnapshot(t *testing.T) {
	org := uuid.New()
	settingID := uuid.New()
	calculation := &CommissionCalculation{BaseCurrency: "USD", CommissionAmount: decimal.RequireFromString("100")}
	repo := &commissionRepoStub{
		preview:    calculation,
		generation: &CommissionGenerationContext{CommissionDate: "2026-08-15", ExchangeRateDate: "2026-08-14", BaseCurrency: "USD"},
	}
	rateStub := newWriteOffRateStub(org, "USD", &ResolvedExchangeRate{Rate: decimal.RequireFromString("0.5"), RateDate: "2026-08-14", SettingID: &settingID})
	usecase := NewCommissionUsecase(repo, nil, NewExchangeRateUsecase(rateStub), &transactorStub{})

	result, err := usecase.Preview(context.Background(), org, uuid.New(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if result.CNY == nil || result.CNY.ExchangeRate.StringFixed(8) != "2.00000000" ||
		result.CNY.ExchangeRateSource != CommissionCNYRateSourceDerived || result.CNY.CommissionAmount.StringFixed(8) != "200.00000000" {
		t.Fatalf("预览 CNY 折算依据不符: %#v", result.CNY)
	}
}

func TestCommissionUsecaseCreateResolvesCNYRateInsideTransaction(t *testing.T) {
	org, actor := uuid.New(), uuid.New()
	settingID := uuid.New()
	verificationID, employeeID, ruleID := uuid.New(), uuid.New(), uuid.New()
	input := CreateCommissionInput{VerificationID: verificationID, EmployeeID: employeeID, RuleID: ruleID, IdempotencyKey: "commission-cny"}

	newUsecase := func(rateStub *exchangeRateRepoStub, repo *commissionRepoStub) *CommissionUsecase {
		return NewCommissionUsecase(repo, NewOrderConfigUsecase(orderConfigNumberStub{}), NewExchangeRateUsecase(rateStub), &transactorStub{})
	}
	newRepo := func() *commissionRepoStub {
		return &commissionRepoStub{
			generation: &CommissionGenerationContext{CommissionDate: "2026-08-15", ExchangeRateDate: "2026-08-14", BaseCurrency: "USD"},
			getResult:  &FinanceCommission{ID: uuid.New()},
		}
	}

	t.Run("非CNY本币按事务内解析结果写入快照", func(t *testing.T) {
		repo := newRepo()
		rateStub := newWriteOffRateStub(org, "USD", &ResolvedExchangeRate{Rate: decimal.RequireFromString("0.5"), RateDate: "2026-08-14", SettingID: &settingID})
		usecase := newUsecase(rateStub, repo)
		created, err := usecase.Create(context.Background(), org, actor, input)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if created != repo.getResult {
			t.Fatalf("创建响应应来自提交后的重读结果")
		}
		snapshot := repo.createdSnapshot
		if snapshot == nil || !repo.createCalled {
			t.Fatalf("事务内未写入提成快照")
		}
		if snapshot.CommissionDate != "2026-08-15" || snapshot.ExchangeRate.StringFixed(8) != "2.00000000" ||
			snapshot.ExchangeRateSource != CommissionCNYRateSourceDerived || snapshot.ExchangeRateDate != "2026-08-14" ||
			snapshot.ExchangeRateSettingID == nil || *snapshot.ExchangeRateSettingID != settingID {
			t.Fatalf("创建快照不符: %#v", snapshot)
		}
		if repo.createAudit == nil || repo.createAudit.Details["cny.exchange_rate"] != "2.00000000" ||
			repo.createAudit.Details["cny.rate_date"] != "2026-08-14" || repo.createAudit.Details["cny.source"] != CommissionCNYRateSourceDerived ||
			repo.createAudit.Details["commission_date"] != "2026-08-15" || repo.createAudit.Details["cny.setting_id"] != settingID.String() {
			t.Fatalf("创建审计详情不符: %#v", repo.createAudit)
		}
	})

	t.Run("本位币为CNY时无需外部汇率配置", func(t *testing.T) {
		repo := newRepo()
		repo.generation.BaseCurrency = "CNY"
		rateStub := newWriteOffRateStub(org, "CNY", nil)
		usecase := newUsecase(rateStub, repo)
		if _, err := usecase.Create(context.Background(), org, actor, input); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		snapshot := repo.createdSnapshot
		if snapshot.ExchangeRate.StringFixed(8) != "1.00000000" || snapshot.ExchangeRateSource != CommissionCNYRateSourceBaseCurrency || snapshot.ExchangeRateSettingID != nil {
			t.Fatalf("CNY 本币快照不符: %#v", snapshot)
		}
	})

	t.Run("预览后汇率变化时按创建事务内重新解析结果写入", func(t *testing.T) {
		repo := newRepo()
		repo.preview = &CommissionCalculation{BaseCurrency: "USD", CommissionAmount: decimal.RequireFromString("100")}
		rateStub := newWriteOffRateStub(org, "USD", &ResolvedExchangeRate{Rate: decimal.RequireFromString("0.5"), RateDate: "2026-08-14", SettingID: &settingID})
		usecase := newUsecase(rateStub, repo)
		preview, err := usecase.Preview(context.Background(), org, verificationID, employeeID, ruleID)
		if err != nil {
			t.Fatalf("Preview() error = %v", err)
		}
		if preview.CNY.ExchangeRate.StringFixed(8) != "2.00000000" {
			t.Fatalf("预览折算率不符: %s", preview.CNY.ExchangeRate.StringFixed(8))
		}
		rateStub.resolved = &ResolvedExchangeRate{Rate: decimal.RequireFromString("0.25"), RateDate: "2026-08-14", SettingID: &settingID}
		if _, err = usecase.Create(context.Background(), org, actor, input); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if repo.createdSnapshot.ExchangeRate.StringFixed(8) != "4.00000000" {
			t.Fatalf("创建应按事务内重新解析的汇率写入: %s", repo.createdSnapshot.ExchangeRate.StringFixed(8))
		}
	})

	t.Run("同核销单重生成按新快照不沿用旧值", func(t *testing.T) {
		repo := newRepo()
		rateStub := newWriteOffRateStub(org, "USD", &ResolvedExchangeRate{Rate: decimal.RequireFromString("0.5"), RateDate: "2026-08-14", SettingID: &settingID})
		usecase := newUsecase(rateStub, repo)
		if _, err := usecase.Create(context.Background(), org, actor, input); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		first := *repo.createdSnapshot
		repo.createdSnapshot = nil
		repo.generation = &CommissionGenerationContext{CommissionDate: "2026-09-01", ExchangeRateDate: "2026-09-01", BaseCurrency: "USD"}
		rateStub.resolved = &ResolvedExchangeRate{Rate: decimal.RequireFromString("0.4"), RateDate: "2026-09-01", SettingID: &settingID}
		if _, err := usecase.Create(context.Background(), org, actor, input); err != nil {
			t.Fatalf("重新生成 Create() error = %v", err)
		}
		second := repo.createdSnapshot
		if second.CommissionDate != "2026-09-01" || second.ExchangeRate.StringFixed(8) != "2.50000000" || second.ExchangeRateDate != "2026-09-01" {
			t.Fatalf("重新生成应按新核销单派生快照: %#v", second)
		}
		if first.CommissionDate == second.CommissionDate {
			t.Fatalf("重新生成不应沿用旧快照")
		}
	})

	t.Run("汇率缺失时报错且不写入", func(t *testing.T) {
		repo := newRepo()
		rateStub := newWriteOffRateStub(org, "USD", nil)
		rateStub.resolveErrByType = map[string]error{WriteOffRateType: ErrExchangeRateMissing}
		usecase := newUsecase(rateStub, repo)
		if _, err := usecase.Create(context.Background(), org, actor, input); err != ErrExchangeRateMissing {
			t.Fatalf("Create() error = %v, want %v", err, ErrExchangeRateMissing)
		}
		if repo.createCalled {
			t.Fatalf("汇率缺失时不应写入提成")
		}
	})

	t.Run("写入失败时错误原样外传", func(t *testing.T) {
		repo := newRepo()
		repo.createErr = ErrCommissionDuplicate
		rateStub := newWriteOffRateStub(org, "USD", &ResolvedExchangeRate{Rate: decimal.RequireFromString("0.5"), RateDate: "2026-08-14", SettingID: &settingID})
		usecase := newUsecase(rateStub, repo)
		if _, err := usecase.Create(context.Background(), org, actor, input); err != ErrCommissionDuplicate {
			t.Fatalf("Create() error = %v, want %v", err, ErrCommissionDuplicate)
		}
	})
}

func TestCommissionUsecaseListValidatesCommissionDateFilter(t *testing.T) {
	org := uuid.New()
	repo := &commissionRepoStub{}
	usecase := NewCommissionUsecase(repo, nil, nil, nil)

	for _, tt := range []struct {
		name     string
		from, to string
		wantErr  bool
		wantFrom string
		wantTo   string
	}{
		{name: "仅起始日期", from: "2026-07-01", wantFrom: "2026-07-01"},
		{name: "仅结束日期", to: "2026-08-31", wantTo: "2026-08-31"},
		{name: "双边区间", from: "2026-07-01", to: "2026-08-31", wantFrom: "2026-07-01", wantTo: "2026-08-31"},
		{name: "同日", from: "2026-07-01", to: "2026-07-01", wantFrom: "2026-07-01", wantTo: "2026-07-01"},
		{name: "非法格式", from: "2026/07/01", wantErr: true},
		{name: "起始晚于结束", from: "2026-08-01", to: "2026-07-01", wantErr: true},
		{name: "空白等价于未传", from: "  ", to: " ", wantFrom: "", wantTo: ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := usecase.List(context.Background(), org, CommissionFilter{Page: 1, PageSize: 20, CommissionDateFrom: tt.from, CommissionDateTo: tt.to})
			if (err != nil) != tt.wantErr {
				t.Fatalf("List() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if repo.listed != nil {
					t.Fatalf("校验失败时不应下发仓储查询")
				}
				return
			}
			if repo.listed == nil || repo.listed.CommissionDateFrom != tt.wantFrom || repo.listed.CommissionDateTo != tt.wantTo {
				t.Fatalf("日期筛选未正确透传: %#v", repo.listed)
			}
			repo.listed = nil
		})
	}
}
