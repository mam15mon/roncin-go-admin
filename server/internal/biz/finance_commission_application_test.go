package biz

import (
	"context"
	"errors"
	"strings"
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
	lastScope          WorkbenchScope
	summary            *WorkbenchApplicationSummary
	candidates         *PagedList[*WorkbenchApplicationCandidate]
	submitted          *FinanceCommissionApplication
	applications       *PagedList[*FinanceCommissionApplication]
	detail             *FinanceCommissionApplicationDetail
	financeFilter      CommissionApplicationFinanceFilter
	financeScope       []uuid.UUID
	financeDetail      *FinanceCommissionApplicationDetail
	financeApplication *FinanceCommissionApplication
	approveArgs        struct {
		organizationIDs []uuid.UUID
		decisionMaker   uuid.UUID
		id              uuid.UUID
		expectedVersion uint64
	}
	rejectArgs struct {
		organizationIDs []uuid.UUID
		decisionMaker   uuid.UUID
		id              uuid.UUID
		expectedVersion uint64
		reason          string
	}
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

func (r *financeCommissionApplicationFakeRepo) ListForOrganization(_ context.Context, organizationIDs []uuid.UUID, filter CommissionApplicationFinanceFilter) (*PagedList[*FinanceCommissionApplication], error) {
	r.financeScope = organizationIDs
	r.financeFilter = filter
	return r.applications, nil
}

func (r *financeCommissionApplicationFakeRepo) GetForOrganization(_ context.Context, organizationIDs []uuid.UUID, _ uuid.UUID) (*FinanceCommissionApplicationDetail, error) {
	r.financeScope = organizationIDs
	return r.financeDetail, nil
}

func (r *financeCommissionApplicationFakeRepo) Approve(_ context.Context, organizationIDs []uuid.UUID, decisionMaker, id uuid.UUID, expectedVersion uint64) (*FinanceCommissionApplication, error) {
	r.approveArgs.organizationIDs = organizationIDs
	r.approveArgs.decisionMaker = decisionMaker
	r.approveArgs.id = id
	r.approveArgs.expectedVersion = expectedVersion
	return r.financeApplication, nil
}

func (r *financeCommissionApplicationFakeRepo) Reject(_ context.Context, organizationIDs []uuid.UUID, decisionMaker, id uuid.UUID, expectedVersion uint64, reason string) (*FinanceCommissionApplication, error) {
	r.rejectArgs.organizationIDs = organizationIDs
	r.rejectArgs.decisionMaker = decisionMaker
	r.rejectArgs.id = id
	r.rejectArgs.expectedVersion = expectedVersion
	r.rejectArgs.reason = reason
	return r.financeApplication, nil
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

// TestCommissionApplicationFinanceUsecaseValidation 验证财务侧用例入口的校验：
// 组织范围必须显式提供、驳回原因必填、审批版本必须非零；合法请求把解析好的
// 组织范围与会话决策人原样委托仓储。
func TestCommissionApplicationFinanceUsecaseValidation(t *testing.T) {
	fake := &financeCommissionApplicationFakeRepo{}
	usecase := NewFinanceCommissionApplicationUsecase(fake)
	org, otherOrg := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	actor, applicationID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	orgs := []uuid.UUID{org}
	longReason := strings.Repeat("驳", 501)

	if _, err := usecase.ListForOrganization(context.Background(), nil, CommissionApplicationFinanceFilter{Page: 1, PageSize: 20}); !errors.Is(err, ErrCommissionApplicationInvalid) {
		t.Fatalf("空组织范围的列表应拒绝: %v", err)
	}
	if _, err := usecase.ListForOrganization(context.Background(), orgs, CommissionApplicationFinanceFilter{Page: 1, PageSize: 20, ApplicationMonth: "2026-13"}); !errors.Is(err, ErrCommissionApplicationInvalid) {
		t.Fatalf("非法提交月过滤应拒绝: %v", err)
	}
	if _, err := usecase.ListForOrganization(context.Background(), orgs, CommissionApplicationFinanceFilter{Page: 1, PageSize: 20, Status: "WORMHOLE"}); !errors.Is(err, ErrCommissionApplicationInvalid) {
		t.Fatalf("非法状态过滤应拒绝: %v", err)
	}
	fake.applications = &PagedList[*FinanceCommissionApplication]{Items: []*FinanceCommissionApplication{{ID: applicationID}}, Total: 1, Page: 1, PageSize: 20}
	if _, err := usecase.ListForOrganization(context.Background(), orgs, CommissionApplicationFinanceFilter{Page: 2, PageSize: 20, EmployeeID: actor, Status: CommissionApplicationPendingReview, ApplicationMonth: "2026-04"}); err != nil {
		t.Fatalf("合法财务列表应委托仓储: %v", err)
	}
	if len(fake.financeScope) != 1 || fake.financeScope[0] != org || fake.financeFilter.EmployeeID != actor {
		t.Fatalf("财务列表应透传组织范围与过滤: scope=%+v filter=%+v", fake.financeScope, fake.financeFilter)
	}

	if _, err := usecase.GetForOrganization(context.Background(), orgs, uuid.Nil); !errors.Is(err, ErrCommissionApplicationNotFound) {
		t.Fatalf("空申请 ID 的财务详情应按不存在处理: %v", err)
	}
	fake.financeDetail = &FinanceCommissionApplicationDetail{}
	if _, err := usecase.GetForOrganization(context.Background(), []uuid.UUID{}, applicationID); !errors.Is(err, ErrCommissionApplicationNotFound) {
		t.Fatalf("空组织范围的财务详情应按不存在处理: %v", err)
	}
	if _, err := usecase.GetForOrganization(context.Background(), orgs, applicationID); err != nil {
		t.Fatalf("合法财务详情应委托仓储: %v", err)
	}

	// 审批入参：组织范围、决策人、申请 ID 与 expected_version 均不可缺省。
	if _, err := usecase.Approve(context.Background(), nil, actor, applicationID, 1); !errors.Is(err, ErrCommissionApplicationInvalid) {
		t.Fatalf("空组织范围的批准应拒绝: %v", err)
	}
	if _, err := usecase.Approve(context.Background(), orgs, uuid.Nil, applicationID, 1); !errors.Is(err, ErrCommissionApplicationInvalid) {
		t.Fatalf("缺决策人的批准应拒绝: %v", err)
	}
	if _, err := usecase.Approve(context.Background(), orgs, actor, applicationID, 0); !errors.Is(err, ErrCommissionApplicationInvalid) {
		t.Fatalf("零版本的批准应拒绝: %v", err)
	}
	fake.financeApplication = &FinanceCommissionApplication{ID: applicationID, Status: CommissionApplicationApproved, Version: 2}
	if _, err := usecase.Approve(context.Background(), orgs, actor, applicationID, 1); err != nil {
		t.Fatalf("合法批准应委托仓储: %v", err)
	}
	if fake.approveArgs.decisionMaker != actor || fake.approveArgs.expectedVersion != 1 || len(fake.approveArgs.organizationIDs) != 1 {
		t.Fatalf("批准应透传决策人与组织范围: %+v", fake.approveArgs)
	}

	// 驳回：原因必填且不超过 500 字。
	if _, err := usecase.Reject(context.Background(), orgs, actor, applicationID, 1, "  "); !errors.Is(err, ErrCommissionApplicationInvalid) {
		t.Fatalf("空原因的驳回应拒绝: %v", err)
	}
	if _, err := usecase.Reject(context.Background(), orgs, actor, applicationID, 1, longReason); !errors.Is(err, ErrCommissionApplicationInvalid) {
		t.Fatalf("超长原因的驳回应拒绝: %v", err)
	}
	if _, err := usecase.Reject(context.Background(), []uuid.UUID{org, otherOrg, {}}, actor, applicationID, 1, "原因"); !errors.Is(err, ErrCommissionApplicationInvalid) {
		t.Fatalf("含零值组织的驳回应拒绝: %v", err)
	}
	if _, err := usecase.Reject(context.Background(), orgs, actor, applicationID, 2, " 明细存疑 "); err != nil {
		t.Fatalf("合法驳回应委托仓储: %v", err)
	}
	if fake.rejectArgs.reason != "明细存疑" || fake.rejectArgs.expectedVersion != 2 {
		t.Fatalf("驳回原因应归一化后透传: %+v", fake.rejectArgs)
	}
}

// TestCommissionApplicationDecisionAuditShape 验证批准/驳回审计的动作名与字段：
// 驳回携带原因，批准不携带；金额与版本以固定精度字符串记录。
func TestCommissionApplicationDecisionAuditShape(t *testing.T) {
	org, actor, applicationID, employeeID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

	approve := CommissionApplicationDecisionAudit(org, actor, applicationID, employeeID,
		CommissionApplicationDecisionAction(true), "2026-04", "2026-03-31", 3, decimal.NewFromInt(180), 2, "")
	if approve.Action != "finance.commission_application.approve" || approve.ResourceID != applicationID.String() {
		t.Fatalf("批准审计动作/资源不符: %+v", approve)
	}
	if approve.Details["employee_id"] != employeeID.String() || approve.Details["version"] != "2" ||
		approve.Details["commission_count"] != "3" || approve.Details["total_amount"] != "180.00000000" {
		t.Fatalf("批准审计字段不符: %+v", approve.Details)
	}
	if _, hasReason := approve.Details["reason"]; hasReason {
		t.Fatalf("批准审计不应携带原因: %+v", approve.Details)
	}

	reject := CommissionApplicationDecisionAudit(org, actor, applicationID, employeeID,
		CommissionApplicationDecisionAction(false), "2026-04", "2026-03-31", 3, decimal.NewFromInt(180), 2, "明细存疑")
	if reject.Action != "finance.commission_application.reject" || reject.Details["reason"] != "明细存疑" {
		t.Fatalf("驳回审计应携带原因: %+v", reject)
	}
}
