package biz

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TestCommissionApplicationPeriod 验证提交月与覆盖截止日的自然月推导：
// coverage_to 固定为提交月份前一自然月最后一天。
func TestCommissionApplicationPeriod(t *testing.T) {
	cases := []struct {
		today          string
		wantMonth      string
		wantCoverageTo string
		wantValid      bool
	}{
		{today: "2026-04-18", wantMonth: "2026-04", wantCoverageTo: "2026-03-31", wantValid: true},
		{today: "2026-01-01", wantMonth: "2026-01", wantCoverageTo: "2025-12-31", wantValid: true},
		{today: "2026-12-31", wantMonth: "2026-12", wantCoverageTo: "2026-11-30", wantValid: true},
		{today: "2026-13-01", wantValid: false},
		{today: "", wantValid: false},
		{today: "20260401", wantValid: false},
	}
	for _, tc := range cases {
		month, coverageTo, valid := CommissionApplicationPeriod(tc.today)
		if valid != tc.wantValid {
			t.Fatalf("CommissionApplicationPeriod(%q) valid = %t，期望 %t", tc.today, valid, tc.wantValid)
		}
		if !tc.wantValid {
			continue
		}
		if month != tc.wantMonth || coverageTo != tc.wantCoverageTo {
			t.Fatalf("CommissionApplicationPeriod(%q) = (%s, %s)，期望 (%s, %s)", tc.today, month, coverageTo, tc.wantMonth, tc.wantCoverageTo)
		}
	}
}

// TestValidCommissionApplicationMonth 验证 YYYY-MM 归属月过滤参数格式。
func TestValidCommissionApplicationMonth(t *testing.T) {
	for value, want := range map[string]bool{
		"2026-04": true,
		"2026-12": true,
		"2026-13": false,
		"2026-4":  false,
		"26-04":   false,
		"abcd-04": false,
		"":        false,
	} {
		if got := ValidCommissionApplicationMonth(value); got != want {
			t.Fatalf("ValidCommissionApplicationMonth(%q) = %t，期望 %t", value, got, want)
		}
	}
}

// financeCommissionApplicationFakeRepo 捕获用例委托的仓储入参。
type financeCommissionApplicationFakeRepo struct {
	lastScope    WorkbenchScope
	summary      *WorkbenchApplicationSummary
	candidates   *PagedList[*WorkbenchApplicationCandidate]
	submitted    *FinanceCommissionApplication
	applications *PagedList[*FinanceCommissionApplication]
	detail       *FinanceCommissionApplicationDetail
}

func (r *financeCommissionApplicationFakeRepo) ApplicationSummary(_ context.Context, scope WorkbenchScope, _ string) (*WorkbenchApplicationSummary, error) {
	r.lastScope = scope
	return r.summary, nil
}

func (r *financeCommissionApplicationFakeRepo) ListMyCandidates(_ context.Context, scope WorkbenchScope, _ WorkbenchApplicationCandidateFilter) (*PagedList[*WorkbenchApplicationCandidate], error) {
	r.lastScope = scope
	return r.candidates, nil
}

func (r *financeCommissionApplicationFakeRepo) Submit(_ context.Context, scope WorkbenchScope) (*FinanceCommissionApplication, error) {
	r.lastScope = scope
	return r.submitted, nil
}

func (r *financeCommissionApplicationFakeRepo) ListMyApplications(_ context.Context, scope WorkbenchScope, _ WorkbenchApplicationFilter) (*PagedList[*FinanceCommissionApplication], error) {
	r.lastScope = scope
	return r.applications, nil
}

func (r *financeCommissionApplicationFakeRepo) GetMyApplication(_ context.Context, scope WorkbenchScope, _ uuid.UUID) (*FinanceCommissionApplicationDetail, error) {
	r.lastScope = scope
	return r.detail, nil
}

func newApplicationTestScope() WorkbenchScope {
	now := time.Now()
	return WorkbenchScope{
		OrganizationID: uuid.Must(uuid.NewV7()),
		UserID:         uuid.Must(uuid.NewV7()),
		Today:          FinanceBusinessDate(now),
		Now:            now,
	}
}

// TestFinanceCommissionApplicationUsecaseValidation 验证用例入口的主体与参数
// 校验：跨组织/跨员工改写参数与非法过滤统一拒绝，合法请求原样委托仓储。
func TestFinanceCommissionApplicationUsecaseValidation(t *testing.T) {
	fake := &financeCommissionApplicationFakeRepo{}
	usecase := NewFinanceCommissionApplicationUsecase(fake)
	scope := newApplicationTestScope()
	invalidScope := WorkbenchScope{UserID: scope.UserID, Today: scope.Today, Now: scope.Now}

	if _, err := usecase.Submit(context.Background(), invalidScope); !errors.Is(err, ErrWorkbenchInvalid) {
		t.Fatalf("缺组织主体的 Submit 应拒绝: %v", err)
	}
	if _, err := usecase.ListMyCandidates(context.Background(), invalidScope, WorkbenchApplicationCandidateFilter{Page: 1, PageSize: 20}); !errors.Is(err, ErrWorkbenchInvalid) {
		t.Fatalf("缺组织主体的候选查询应拒绝: %v", err)
	}
	if _, err := usecase.ListMyCandidates(context.Background(), scope, WorkbenchApplicationCandidateFilter{Page: 0, PageSize: 20}); !errors.Is(err, ErrWorkbenchInvalid) {
		t.Fatalf("非法分页应拒绝: %v", err)
	}
	if _, err := usecase.ListMyCandidates(context.Background(), scope, WorkbenchApplicationCandidateFilter{Page: 1, PageSize: 20, CommissionMonth: "2026-13"}); !errors.Is(err, ErrWorkbenchInvalid) {
		t.Fatalf("非法归属月过滤应拒绝: %v", err)
	}
	if _, err := usecase.ListMyApplications(context.Background(), scope, WorkbenchApplicationFilter{Page: 1, PageSize: 20, Status: "WORMHOLE"}); !errors.Is(err, ErrWorkbenchInvalid) {
		t.Fatalf("非法申请状态过滤应拒绝: %v", err)
	}
	if _, err := usecase.GetMyApplication(context.Background(), scope, uuid.Nil); !errors.Is(err, ErrCommissionApplicationNotFound) {
		t.Fatalf("空申请 ID 应按不存在处理: %v", err)
	}

	fake.submitted = &FinanceCommissionApplication{ID: uuid.Must(uuid.NewV7()), Status: CommissionApplicationPendingReview}
	if _, err := usecase.Submit(context.Background(), scope); err != nil {
		t.Fatalf("合法 Submit 应委托仓储: %v", err)
	}
	if fake.lastScope.UserID != scope.UserID || fake.lastScope.OrganizationID != scope.OrganizationID {
		t.Fatalf("Submit 主体应固定为会话组织与本人: %+v", fake.lastScope)
	}
	_ = decimal.Zero
}
