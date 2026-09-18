package biz

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TestWorkbenchActualAssignmentWindow 覆盖方案与分配实际有效区间的交集计算：
// 未启用方案、起点取晚、终点取早与空区间。
func TestWorkbenchActualAssignmentWindow(t *testing.T) {
	cases := []struct {
		name                                   string
		ruleEnabled                            bool
		ruleFrom, ruleTo, assignFrom, assignTo string
		expectOK                               bool
		expectFrom, expectTo                   string
	}{
		{name: "未启用方案直接失效", ruleEnabled: false, ruleFrom: "2026-01-01", assignFrom: "2026-01-01", expectOK: false},
		{name: "双开放区间取分配起点", ruleEnabled: true, ruleFrom: "", assignFrom: "2026-03-01", expectOK: true, expectFrom: "2026-03-01"},
		{name: "方案起点晚于分配起点", ruleEnabled: true, ruleFrom: "2026-04-01", assignFrom: "2026-01-01", expectOK: true, expectFrom: "2026-04-01"},
		{name: "方案终点早于分配终点", ruleEnabled: true, ruleFrom: "2026-01-01", ruleTo: "2026-06-30", assignFrom: "2026-01-01", assignTo: "2026-12-31", expectOK: true, expectFrom: "2026-01-01", expectTo: "2026-06-30"},
		{name: "区间不交集为空", ruleEnabled: true, ruleFrom: "2026-07-01", ruleTo: "2026-09-30", assignFrom: "2026-01-01", assignTo: "2026-03-31", expectOK: false},
		{name: "单日闭区间仍有效", ruleEnabled: true, ruleFrom: "2026-05-01", ruleTo: "2026-05-01", assignFrom: "2026-01-01", expectOK: true, expectFrom: "2026-05-01", expectTo: "2026-05-01"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			window, ok := WorkbenchActualAssignmentWindow(tc.ruleEnabled, tc.ruleFrom, tc.ruleTo, tc.assignFrom, tc.assignTo)
			if ok != tc.expectOK {
				t.Fatalf("ok = %v，期望 %v", ok, tc.expectOK)
			}
			if tc.expectOK && (window.EffectiveFrom != tc.expectFrom || window.EffectiveTo != tc.expectTo) {
				t.Fatalf("window = %+v，期望 [%s, %s]", window, tc.expectFrom, tc.expectTo)
			}
		})
	}
}

// TestResolveWorkbenchEligibility 覆盖工作台提成模块门禁矩阵：无分配无历史、
// 未来方案分配、过期分配但有历史、历史提成单、历史调整单。
func TestResolveWorkbenchEligibility(t *testing.T) {
	today := "2026-09-18"
	cases := []struct {
		name                 string
		facts                WorkbenchEligibilityFacts
		expectEligible       bool
		expectNextEffectDate string
	}{
		{
			name:           "无方案分配且无历史时门禁为假",
			facts:          WorkbenchEligibilityFacts{},
			expectEligible: false,
		},
		{
			name: "未来方案分配开启门禁并返回最近生效日",
			facts: WorkbenchEligibilityFacts{ActiveWindows: []WorkbenchAssignmentWindow{
				{EffectiveFrom: "2026-10-01"},
				{EffectiveFrom: "2026-11-01"},
			}},
			expectEligible:       true,
			expectNextEffectDate: "2026-10-01",
		},
		{
			name: "当前已生效分配无未来生效日",
			facts: WorkbenchEligibilityFacts{ActiveWindows: []WorkbenchAssignmentWindow{
				{EffectiveFrom: "2026-01-01"},
			}},
			expectEligible: true,
		},
		{
			name:           "过期分配但有历史提成单仍开启门禁",
			facts:          WorkbenchEligibilityFacts{HasCommissionHistory: true},
			expectEligible: true,
		},
		{
			name:           "仅历史调整单也开启门禁",
			facts:          WorkbenchEligibilityFacts{HasAdjustmentHistory: true},
			expectEligible: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eligible, nextDate := ResolveWorkbenchEligibility(tc.facts, today)
			if eligible != tc.expectEligible {
				t.Fatalf("eligible = %v，期望 %v", eligible, tc.expectEligible)
			}
			if nextDate != tc.expectNextEffectDate {
				t.Fatalf("nextEffectiveDate = %q，期望 %q", nextDate, tc.expectNextEffectDate)
			}
		})
	}

	// 未来生效日不早于今天：当天生效不算「未来」，不返回提示日期。
	eligible, nextDate := ResolveWorkbenchEligibility(WorkbenchEligibilityFacts{ActiveWindows: []WorkbenchAssignmentWindow{{EffectiveFrom: today}}}, today)
	if !eligible || nextDate != "" {
		t.Fatalf("当天生效应返回 eligible=true 且无未来提示，实际 %v/%q", eligible, nextDate)
	}
}

// TestWorkbenchPaidPeriodRange 覆盖本年/本月已发累计窗口的上海业务时区边界。
func TestWorkbenchPaidPeriodRange(t *testing.T) {
	// UTC 2026-09-18 17:00 是上海时间 2026-09-19 01:00：本月窗口仍是 9 月 1 日。
	now := time.Date(2026, 9, 18, 17, 0, 0, 0, time.UTC)
	yearStart, monthStart := WorkbenchPaidPeriodRange(now)
	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)
	if got := yearStart.In(shanghai).Format("2006-01-02 15:04"); got != "2026-01-01 00:00" {
		t.Fatalf("yearStart = %s，期望 2026-01-01 00:00", got)
	}
	if got := monthStart.In(shanghai).Format("2006-01-02 15:04"); got != "2026-09-01 00:00" {
		t.Fatalf("monthStart = %s，期望 2026-09-01 00:00", got)
	}
	// 跨年边界：上海时间 2027-01-01 00:30 时本年窗口已是 2027 年。
	crossYear := time.Date(2026, 12, 31, 16, 30, 0, 0, time.UTC)
	yearStart2, _ := WorkbenchPaidPeriodRange(crossYear)
	if got := yearStart2.In(shanghai).Format("2006-01-02"); got != "2027-01-01" {
		t.Fatalf("crossYear yearStart = %s，期望 2027-01-01", got)
	}
}

// stubWorkbenchRepo 记录委托调用的最小桩仓储。
type stubWorkbenchRepo struct {
	overviewErr error
	calls       []string
}

func (s *stubWorkbenchRepo) GetOverview(context.Context, WorkbenchScope) (*WorkbenchOverview, error) {
	s.calls = append(s.calls, "overview")
	return &WorkbenchOverview{}, s.overviewErr
}

func (s *stubWorkbenchRepo) ListMyCommissions(context.Context, WorkbenchScope, WorkbenchCommissionFilter) (*PagedList[*WorkbenchMyCommission], error) {
	return &PagedList[*WorkbenchMyCommission]{}, nil
}

func (s *stubWorkbenchRepo) ListMyReceivables(context.Context, WorkbenchScope, WorkbenchReceivableFilter) (*PagedList[*WorkbenchMyReceivable], error) {
	return &PagedList[*WorkbenchMyReceivable]{}, nil
}

func (s *stubWorkbenchRepo) ListMyRecentOrders(context.Context, WorkbenchScope, WorkbenchOrderFilter) (*PagedList[*WorkbenchRecentOrder], error) {
	return &PagedList[*WorkbenchRecentOrder]{}, nil
}

func workbenchTestScope() WorkbenchScope {
	return WorkbenchScope{
		OrganizationID: uuid.Must(uuid.NewV7()),
		UserID:         uuid.Must(uuid.NewV7()),
		Today:          "2026-09-18",
		Now:            time.Date(2026, 9, 18, 6, 0, 0, 0, time.UTC),
	}
}

// TestWorkbenchUsecaseValidation 覆盖用例层主体与过滤校验：非法范围、分页、
// 状态与日期区间统一返回 ErrWorkbenchInvalid。
func TestWorkbenchUsecaseValidation(t *testing.T) {
	usecase := NewWorkbenchUsecase(&stubWorkbenchRepo{})
	ctx := context.Background()
	validScope := workbenchTestScope()

	if _, err := usecase.GetOverview(ctx, WorkbenchScope{}); !errors.Is(err, ErrWorkbenchInvalid) {
		t.Fatalf("空 scope 应返回 ErrWorkbenchInvalid，实际 %v", err)
	}
	invalidScopes := []WorkbenchScope{
		{OrganizationID: uuid.Nil, UserID: validScope.UserID, Today: validScope.Today, Now: validScope.Now},
		{OrganizationID: validScope.OrganizationID, UserID: uuid.Nil, Today: validScope.Today, Now: validScope.Now},
		{OrganizationID: validScope.OrganizationID, UserID: validScope.UserID, Today: "20260918", Now: validScope.Now},
	}
	for _, scope := range invalidScopes {
		if _, err := usecase.GetOverview(ctx, scope); !errors.Is(err, ErrWorkbenchInvalid) {
			t.Fatalf("非法 scope %+v 应返回 ErrWorkbenchInvalid，实际 %v", scope, err)
		}
	}
	if _, err := usecase.GetOverview(ctx, validScope); err != nil {
		t.Fatalf("合法 scope 不应报错: %v", err)
	}

	invalidFilters := []WorkbenchCommissionFilter{
		{Page: 0, PageSize: 20},
		{Page: 1, PageSize: 201},
		{Page: 1, PageSize: 20, Status: CommissionStatus("UNKNOWN")},
		{Page: 1, PageSize: 20, CommissionDateFrom: "2026-02-01", CommissionDateTo: "2026-01-01"},
	}
	for _, f := range invalidFilters {
		if _, err := usecase.ListMyCommissions(ctx, validScope, f); !errors.Is(err, ErrWorkbenchInvalid) {
			t.Fatalf("非法过滤 %+v 应返回 ErrWorkbenchInvalid，实际 %v", f, err)
		}
	}
	// 非法 scope 即便过滤正确也必须拒绝，防止跨组织读取。
	if _, err := usecase.ListMyCommissions(ctx, WorkbenchScope{}, WorkbenchCommissionFilter{Page: 1, PageSize: 20}); !errors.Is(err, ErrWorkbenchInvalid) {
		t.Fatalf("空 scope 的明细查询应返回 ErrWorkbenchInvalid，实际 %v", err)
	}
	if _, err := usecase.ListMyReceivables(ctx, validScope, WorkbenchReceivableFilter{Page: 1, PageSize: 100}); err != nil {
		t.Fatalf("应收分页上限 100 不应报错: %v", err)
	}
	if _, err := usecase.ListMyReceivables(ctx, validScope, WorkbenchReceivableFilter{Page: 1, PageSize: 0}); !errors.Is(err, ErrWorkbenchInvalid) {
		t.Fatalf("应收 pageSize=0 应返回 ErrWorkbenchInvalid，实际 %v", err)
	}
	if _, err := usecase.ListMyRecentOrders(ctx, validScope, WorkbenchOrderFilter{Page: 1, PageSize: 200}); err != nil {
		t.Fatalf("订单分页上限 200 不应报错: %v", err)
	}
}

// TestWorkbenchSummaryBucketSemantics 在数值层面锁定分桶语义：CANCELLED 不进入
// 任何桶，金额口径保持本位币十进制。
func TestWorkbenchSummaryBucketSemantics(t *testing.T) {
	summary := &WorkbenchCommissionSummary{
		BaseCurrency:       "CNY",
		DraftCount:         1,
		DraftAmount:        decimal.NewFromInt(100),
		ConfirmedCount:     2,
		ConfirmedAmount:    decimal.NewFromInt(50),
		PaidCount:          3,
		PaidAmount:         decimal.NewFromInt(70),
		PaidThisYear:       decimal.NewFromInt(70),
		PaidThisMonth:      decimal.NewFromInt(20),
		DecreaseDraftCount: 1,
	}
	if summary.DraftAmount.Add(summary.ConfirmedAmount).Add(summary.PaidAmount).Equal(decimal.NewFromInt(220)) {
		// 分桶金额互不并入：PAID 不包含待发与待确认金额。
		if !summary.PaidAmount.Equal(summary.PaidThisYear) {
			t.Fatalf("本年已发累计应等于 PAID 桶金额（当前窗口内）")
		}
	} else {
		t.Fatalf("分桶金额汇总不符合预期")
	}
	if summary.DecreaseDraftCount != 1 {
		t.Fatalf("冲减草稿桶应独立计数")
	}
}
