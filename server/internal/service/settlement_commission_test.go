package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// commissionRepoStub 捕获列表筛选并返回固定的提成与预览结果，不触发数据库。
type commissionRepoStub struct {
	biz.CommissionRepo
	filter         biz.CommissionFilter
	listResult     *biz.CommissionListResult
	preview        *biz.CommissionCalculation
	generation     *biz.CommissionGenerationContext
	ruleFilter     biz.CommissionRuleFilter
	ruleScope      []uuid.UUID
	ruleResult     *biz.CommissionRuleListResult
	ruleErr        error
	rateContext    *biz.ExchangeRateContext
	rateStub       *exchangeRateStub
	exportTotal    int64
	exportBatch    []*biz.FinanceCommission
	exportFilter   biz.CommissionFilter
	exportAuditLog *biz.AuditEvent
}

func (s *commissionRepoStub) List(_ context.Context, _ uuid.UUID, f biz.CommissionFilter) (*biz.CommissionListResult, error) {
	s.filter = f
	return s.listResult, nil
}

func (s *commissionRepoStub) ListScoped(_ context.Context, _ []uuid.UUID, f biz.CommissionFilter) (*biz.CommissionListResult, error) {
	s.filter = f
	return s.listResult, nil
}

func (s *commissionRepoStub) Count(_ context.Context, _ uuid.UUID, f biz.CommissionFilter) (int64, error) {
	s.exportFilter = f
	return s.exportTotal, nil
}

func (s *commissionRepoStub) CountScoped(_ context.Context, _ []uuid.UUID, f biz.CommissionFilter) (int64, error) {
	s.exportFilter = f
	return s.exportTotal, nil
}

func (s *commissionRepoStub) ExportBatch(_ context.Context, _ uuid.UUID, f biz.CommissionFilter) ([]*biz.FinanceCommission, error) {
	s.exportFilter = f
	return s.exportBatch, nil
}

func (s *commissionRepoStub) ExportBatchScoped(_ context.Context, _ []uuid.UUID, f biz.CommissionFilter) ([]*biz.FinanceCommission, error) {
	s.exportFilter = f
	return s.exportBatch, nil
}

func (s *commissionRepoStub) SaveExportAudit(_ context.Context, event *biz.AuditEvent) error {
	s.exportAuditLog = event
	return nil
}

func (s *commissionRepoStub) Preview(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) (*biz.CommissionCalculation, error) {
	return s.preview, nil
}

func (s *commissionRepoStub) GetGenerationContext(context.Context, uuid.UUID, uuid.UUID) (*biz.CommissionGenerationContext, error) {
	return s.generation, nil
}

func (s *commissionRepoStub) GetRuleScoped(_ context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*biz.FinanceCommissionRule, error) {
	return &biz.FinanceCommissionRule{ID: id, OrganizationID: organizationIDs[0]}, nil
}

func (s *commissionRepoStub) ListRulesScoped(_ context.Context, organizationIDs []uuid.UUID, filter biz.CommissionRuleFilter) (*biz.CommissionRuleListResult, error) {
	s.ruleScope = organizationIDs
	s.ruleFilter = filter
	if s.ruleErr != nil {
		return nil, s.ruleErr
	}
	if s.ruleResult == nil {
		return &biz.CommissionRuleListResult{}, nil
	}
	return s.ruleResult, nil
}

// exchangeRateStub 模拟 CNY→本位币的按生成日汇率解析。
type exchangeRateStub struct {
	biz.ExchangeRateRepo
	context *biz.ExchangeRateContext
	rate    decimal.Decimal
}

func (s *exchangeRateStub) ResolveContext(context.Context, uuid.UUID) (*biz.ExchangeRateContext, error) {
	return s.context, nil
}

func (s *exchangeRateStub) ResolveRate(context.Context, uuid.UUID, string, string, string) (decimal.Decimal, error) {
	return s.rate, nil
}

func newCommissionService(org uuid.UUID) (*SettlementService, *commissionRepoStub) {
	repo := &commissionRepoStub{
		listResult: &biz.CommissionListResult{Items: []*biz.FinanceCommission{{
			ID:                           uuid.New(),
			CommissionNo:                 "TC20260815000001",
			Status:                       biz.CommissionDraft,
			BaseCurrency:                 "USD",
			CommissionAmount:             decimal.RequireFromString("100"),
			CommissionDate:               "2026-08-15",
			CNYExchangeRate:              decimal.RequireFromString("2"),
			CNYExchangeRateSource:        biz.CommissionCNYRateSourceDerived,
			CNYExchangeRateDate:          "2026-08-15",
			CNYCommissionAmount:          decimal.RequireFromString("200"),
			CNYAdjustmentAmount:          decimal.RequireFromString("20"),
			CNYEffectiveCommissionAmount: decimal.RequireFromString("220"),
			AdjustmentAmount:             decimal.RequireFromString("10"),
			EffectiveCommissionAmount:    decimal.RequireFromString("110"),
		}}},
		preview:    &biz.CommissionCalculation{BaseCurrency: "USD", CommissionAmount: decimal.RequireFromString("100")},
		generation: &biz.CommissionGenerationContext{CommissionDate: "2026-08-15", BaseCurrency: "USD"},
		rateStub: &exchangeRateStub{
			context: &biz.ExchangeRateContext{OwnerOrganizationID: org, BaseCurrency: "USD"},
			rate:    decimal.RequireFromString("0.5"),
		},
	}
	usecase := biz.NewCommissionUsecase(repo, nil, biz.NewExchangeRateUsecase(repo.rateStub), &commissionTransactorStub{})
	verificationUsecase := biz.NewVerificationUsecase(&verificationRepoStub{}, nil, nil)
	service := NewSettlementService(nil, nil, nil, nil, verificationUsecase, nil, usecase, nil, nil, nil, nil)
	return service, repo
}

type commissionTransactorStub struct{}

type verificationRepoStub struct {
	biz.VerificationRepo
	organizationIDs []uuid.UUID
	filter          biz.VerificationFilter
	listResult      *biz.VerificationListResult
	listErr         error
}

func (*verificationRepoStub) GetScoped(_ context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*biz.FinanceVerification, error) {
	return &biz.FinanceVerification{ID: id, OrganizationID: organizationIDs[0]}, nil
}

func (s *verificationRepoStub) ListScoped(_ context.Context, organizationIDs []uuid.UUID, filter biz.VerificationFilter) (*biz.VerificationListResult, error) {
	s.organizationIDs = organizationIDs
	s.filter = filter
	if s.listErr != nil {
		return nil, s.listErr
	}
	if s.listResult == nil {
		return &biz.VerificationListResult{}, nil
	}
	return s.listResult, nil
}

func (commissionTransactorStub) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func commissionPrincipalContext(org uuid.UUID) context.Context {
	permissions := []string{access.FinanceCommissionRead, access.FinanceCommissionManage, access.FinanceCommissionExport}
	return biz.WithPrincipal(context.Background(), &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: org}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: org}}, RoleGrants: []biz.RoleGrant{{RoleCode: "finance", DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{permissions[0]: {}, permissions[1]: {}, permissions[2]: {}}}}})
}

func TestListCommissionsMapsCommissionDateFilterAndCNYFields(t *testing.T) {
	org := uuid.New()
	service, repo := newCommissionService(org)

	response, err := service.ListCommissions(commissionPrincipalContext(org), &v1.ListCommissionsRequest{
		Page:               1,
		PageSize:           20,
		CommissionDateFrom: ptrString("2026-07-01"),
		CommissionDateTo:   ptrString("2026-08-31"),
	})
	if err != nil {
		t.Fatalf("ListCommissions() error = %v", err)
	}
	if repo.filter.CommissionDateFrom != "2026-07-01" || repo.filter.CommissionDateTo != "2026-08-31" {
		t.Fatalf("归属日期筛选未透传: %#v", repo.filter)
	}
	item := response.Data[0]
	if item.CommissionDate != "2026-08-15" || item.CnyExchangeRate != "2.00000000" ||
		item.CnyExchangeRateSource != biz.CommissionCNYRateSourceDerived || item.CnyExchangeRateDate != "2026-08-15" ||
		item.CnyCommissionAmount != "200.00000000" ||
		item.CnyAdjustmentAmount != "20.00000000" || item.CnyEffectiveCommissionAmount != "220.00000000" {
		t.Fatalf("提成 CNY 字段未返回: %#v", item)
	}
}

func TestExportCommissionsMapsFilterAndDualCurrencyExportFields(t *testing.T) {
	org := uuid.New()
	service, repo := newCommissionService(org)
	actor := uuid.New()
	repo.exportTotal = 1
	repo.exportBatch = []*biz.FinanceCommission{{
		ID:                           uuid.New(),
		OrganizationID:               org,
		OrganizationName:             "测试导出组织",
		CommissionNo:                 "TC20260815000001",
		VerificationNo:               "VR20260815000001",
		Status:                       biz.CommissionPaid,
		BaseCurrency:                 "USD",
		RatePercent:                  decimal.RequireFromString("2.5"),
		CommissionAmount:             decimal.RequireFromString("100"),
		AdjustmentAmount:             decimal.RequireFromString("10"),
		EffectiveCommissionAmount:    decimal.RequireFromString("110"),
		CommissionDate:               "2026-08-15",
		EmployeeName:                 "张三",
		RuleName:                     "销售提成",
		PersonnelRole:                biz.CommissionRoleSales,
		CalculationBasis:             biz.CommissionBasisRealizedProfit,
		CreatedAt:                    time.Date(2026, 8, 15, 8, 0, 0, 0, time.UTC),
		CNYCommissionAmount:          decimal.RequireFromString("200"),
		CNYAdjustmentAmount:          decimal.RequireFromString("20"),
		CNYEffectiveCommissionAmount: decimal.RequireFromString("220"),
	}}

	response, err := service.ExportCommissions(biz.WithPrincipal(context.Background(), &biz.Principal{UserID: actor, Organization: biz.Organization{ID: org}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: org}}, RoleGrants: []biz.RoleGrant{{RoleCode: "finance", DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{access.FinanceCommissionExport: {}}}}}), &v1.ExportCommissionsRequest{
		Keyword:            ptrString("TC2026"),
		Status:             v1.FinanceCommissionStatus(v1.FinanceCommissionStatus_FINANCE_COMMISSION_STATUS_PAID).Enum(),
		CommissionDateFrom: ptrString("2026-07-01"),
		CommissionDateTo:   ptrString("2026-08-31"),
	})
	if err != nil {
		t.Fatalf("ExportCommissions() error = %v", err)
	}
	if repo.exportFilter.Keyword != "TC2026" || repo.exportFilter.Status != biz.CommissionPaid ||
		repo.exportFilter.CommissionDateFrom != "2026-07-01" || repo.exportFilter.CommissionDateTo != "2026-08-31" {
		t.Fatalf("导出筛选未透传: %#v", repo.exportFilter)
	}
	if len(response.Data) != 1 {
		t.Fatalf("导出行数不符: %d", len(response.Data))
	}
	item := response.Data[0]
	if item.CommissionNo != "TC20260815000001" || item.VerificationNo != "VR20260815000001" ||
		item.Status != v1.FinanceCommissionStatus_FINANCE_COMMISSION_STATUS_PAID || item.CommissionDate != "2026-08-15" ||
		item.EmployeeName != "张三" || item.PersonnelRole != string(biz.CommissionRoleSales) ||
		item.RuleName != "销售提成" || item.CalculationBasis != string(biz.CommissionBasisRealizedProfit) ||
		item.RatePercent != "2.5000" || item.BaseCurrency != "USD" || item.OrganizationId != org.String() ||
		item.OrganizationName != "测试导出组织" || item.CreatedAt != "2026-08-15T08:00:00Z" {
		t.Fatalf("导出基础字段不符: %#v", item)
	}
	if item.CommissionAmount != "100.00000000" || item.AdjustmentAmount != "10.00000000" || item.EffectiveCommissionAmount != "110.00000000" ||
		item.CnyCommissionAmount != "200.00000000" || item.CnyAdjustmentAmount != "20.00000000" || item.CnyEffectiveCommissionAmount != "220.00000000" {
		t.Fatalf("导出双口径金额不符: %#v", item)
	}
	if repo.exportAuditLog == nil || repo.exportAuditLog.Action != "finance.commission.export" ||
		repo.exportAuditLog.UserID == nil || *repo.exportAuditLog.UserID != actor ||
		repo.exportAuditLog.Details["row_count"] != "1" {
		t.Fatalf("成功导出缺少审计: %#v", repo.exportAuditLog)
	}
}

func TestPreviewCommissionReturnsCNYRateBasis(t *testing.T) {
	org := uuid.New()
	service, _ := newCommissionService(org)

	response, err := service.PreviewCommission(commissionPrincipalContext(org), &v1.PreviewCommissionRequest{
		VerificationId: uuid.New().String(), EmployeeId: uuid.New().String(), RuleId: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PreviewCommission() error = %v", err)
	}
	if response.Data.CnyExchangeRate != "2.00000000" || response.Data.CnyExchangeRateSource != biz.CommissionCNYRateSourceDerived ||
		response.Data.CnyExchangeRateDate != "2026-08-15" || response.Data.CnyCommissionAmount != "200.00000000" {
		t.Fatalf("预览折算依据未返回: %#v", response.Data)
	}
}

func TestListCommissionVerificationCandidatesUsesManageWritableOrganization(t *testing.T) {
	allowed, denied := uuid.New(), uuid.New()
	repo := &verificationRepoStub{listResult: &biz.VerificationListResult{Items: []*biz.FinanceVerification{{
		ID: uuid.New(), OrganizationID: allowed, VerificationNo: "VR202609100001", Status: biz.VerificationActive,
		Direction: biz.OrderFeeReceivable, SettlementPartyName: "测试结算单位", Currency: "CNY",
	}}}}
	service := NewSettlementService(nil, nil, nil, nil, biz.NewVerificationUsecase(repo, nil, nil), nil, nil, nil, nil, nil, nil)
	principal := &biz.Principal{
		UserID: uuid.New(), Organization: biz.Organization{ID: uuid.New()}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: allowed}},
		RoleGrants: []biz.RoleGrant{{RoleCode: "commission-manager", DataScope: biz.DataScopeOrganization,
			Permissions:          map[string]struct{}{access.FinanceCommissionManage: {}},
			OrganizationAccesses: []biz.OrganizationAccess{{OrganizationID: allowed, Writable: true}}}},
	}
	ctx := biz.WithPrincipal(context.Background(), principal)

	response, err := service.ListCommissionVerificationCandidates(ctx, &v1.ListCommissionVerificationCandidatesRequest{OrganizationId: allowed.String(), Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("Manage-only 候选查询失败: %v", err)
	}
	if len(response.Data) != 1 || len(repo.organizationIDs) != 1 || repo.organizationIDs[0] != allowed ||
		repo.filter.Status != biz.VerificationActive || repo.filter.Direction != biz.OrderFeeReceivable {
		t.Fatalf("候选未按提成管理权限和固定核销条件查询: result=%#v ids=%v filter=%+v", response.Data, repo.organizationIDs, repo.filter)
	}

	if _, err := service.ListCommissionVerificationCandidates(ctx, &v1.ListCommissionVerificationCandidatesRequest{OrganizationId: denied.String(), Page: 1, PageSize: 20}); !errors.Is(err, biz.ErrPermissionDenied) {
		t.Fatalf("越权组织错误 = %v，期望 %v", err, biz.ErrPermissionDenied)
	}
	if _, err := service.ListCommissionVerificationCandidates(ctx, &v1.ListCommissionVerificationCandidatesRequest{Page: 1, PageSize: 20}); !errors.Is(err, biz.ErrVerificationInvalid) {
		t.Fatalf("空组织错误 = %v，期望 %v", err, biz.ErrVerificationInvalid)
	}
	if _, err := service.ListCommissionVerificationCandidates(ctx, &v1.ListCommissionVerificationCandidatesRequest{OrganizationId: "not-a-uuid", Page: 1, PageSize: 20}); !errors.Is(err, biz.ErrFinanceLedgerInvalidArgument) {
		t.Fatalf("非法组织错误 = %v，期望 %v", err, biz.ErrFinanceLedgerInvalidArgument)
	}

	expected := errors.New("核销候选仓储错误")
	repo.listErr = expected
	if _, err := service.ListCommissionVerificationCandidates(ctx, &v1.ListCommissionVerificationCandidatesRequest{OrganizationId: allowed.String(), Page: 1, PageSize: 20}); !errors.Is(err, expected) {
		t.Fatalf("仓储错误未透传: %v", err)
	}
}

func TestListCommissionRuleCandidatesUsesManageWritableOrganization(t *testing.T) {
	allowed, denied := uuid.New(), uuid.New()
	service, repo := newCommissionService(allowed)
	repo.ruleResult = &biz.CommissionRuleListResult{Items: []*biz.FinanceCommissionRule{{
		ID: uuid.New(), OrganizationID: allowed, Name: "销售基础提成", PersonnelRole: biz.CommissionRoleSales,
		CalculationBasis: biz.CommissionBasisRealizedProfit, RatePercent: decimal.RequireFromString("2.5"), Enabled: true,
	}}}
	principal := &biz.Principal{
		UserID: uuid.New(), Organization: biz.Organization{ID: uuid.New()}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: allowed}},
		RoleGrants: []biz.RoleGrant{{RoleCode: "commission-manager", DataScope: biz.DataScopeOrganization,
			Permissions:          map[string]struct{}{access.FinanceCommissionManage: {}},
			OrganizationAccesses: []biz.OrganizationAccess{{OrganizationID: allowed, Writable: true}}}},
	}
	ctx := biz.WithPrincipal(context.Background(), principal)

	response, err := service.ListCommissionRuleCandidates(ctx, &v1.ListCommissionRuleCandidatesRequest{OrganizationId: allowed.String(), Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("Manage-only 规则候选查询失败: %v", err)
	}
	if len(response.Data) != 1 || len(repo.ruleScope) != 1 || repo.ruleScope[0] != allowed ||
		repo.ruleFilter.Enabled == nil || !*repo.ruleFilter.Enabled {
		t.Fatalf("候选未按提成管理可写组织和已启用规则查询: result=%#v scope=%v filter=%+v", response.Data, repo.ruleScope, repo.ruleFilter)
	}

	if _, err := service.ListCommissionRuleCandidates(ctx, &v1.ListCommissionRuleCandidatesRequest{OrganizationId: denied.String(), Page: 1, PageSize: 20}); !errors.Is(err, biz.ErrPermissionDenied) {
		t.Fatalf("越权组织错误 = %v，期望 %v", err, biz.ErrPermissionDenied)
	}
	if _, err := service.ListCommissionRuleCandidates(ctx, &v1.ListCommissionRuleCandidatesRequest{Page: 1, PageSize: 20}); !errors.Is(err, biz.ErrCommissionRuleInvalid) {
		t.Fatalf("空组织错误 = %v，期望 %v", err, biz.ErrCommissionRuleInvalid)
	}
	readOnly := &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: allowed}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: allowed}}, RoleGrants: []biz.RoleGrant{{RoleCode: "commission-reader", DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{access.FinanceCommissionRead: {}}}}}
	if _, err := service.ListCommissionRuleCandidates(biz.WithPrincipal(context.Background(), readOnly), &v1.ListCommissionRuleCandidatesRequest{OrganizationId: allowed.String(), Page: 1, PageSize: 20}); !errors.Is(err, biz.ErrPermissionDenied) {
		t.Fatalf("只读权限错误 = %v，期望 %v", err, biz.ErrPermissionDenied)
	}
}

func ptrString(value string) *string { return &value }
