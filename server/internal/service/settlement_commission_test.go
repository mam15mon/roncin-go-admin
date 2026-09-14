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
	nettingFilter  biz.CommissionNettingCandidateFilter
	nettingResult  *biz.CommissionNettingCandidateListResult
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

func (s *commissionRepoStub) Preview(_ context.Context, _ uuid.UUID, verificationID, nettingID, employeeID, ruleID uuid.UUID) (*biz.CommissionCalculation, error) {
	if s.preview != nil {
		s.preview.VerificationID = verificationID
		s.preview.NettingID = nettingID
		s.preview.EmployeeID = employeeID
		s.preview.RuleID = ruleID
	}
	return s.preview, nil
}

func (s *commissionRepoStub) GetGenerationContext(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*biz.CommissionGenerationContext, error) {
	return s.generation, nil
}

func (s *commissionRepoStub) ListNettingCandidates(_ context.Context, _ uuid.UUID, f biz.CommissionNettingCandidateFilter) (*biz.CommissionNettingCandidateListResult, error) {
	s.nettingFilter = f
	if s.nettingResult == nil {
		return &biz.CommissionNettingCandidateListResult{}, nil
	}
	return s.nettingResult, nil
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

func (s *exchangeRateStub) ResolveRate(context.Context, uuid.UUID, biz.OrderFeeDirection, string, string, string, string) (biz.ResolvedRate, error) {
	return biz.ResolvedRate{Rate: s.rate, Source: biz.ExchangeRateSourceSystem}, nil
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
	usecase := biz.NewCommissionUsecase(repo, nil, &commissionTransactorStub{})
	verificationUsecase := biz.NewVerificationUsecase(&verificationRepoStub{}, nil, nil)
	nettingUsecase := biz.NewFinanceNettingUsecase(&nettingRepoStub{}, nil)
	service := NewSettlementService(nil, nil, nil, nil, verificationUsecase, nettingUsecase, usecase, nil, nil, nil, nil)
	return service, repo
}

// nettingRepoStub 提供提成来源校验所需的最小对冲仓储行为。
type nettingRepoStub struct {
	biz.FinanceNettingRepo
	netting *biz.FinanceNetting
	getErr  error
}

func (s *nettingRepoStub) Get(_ context.Context, _ []uuid.UUID, id uuid.UUID) (*biz.FinanceNetting, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.netting != nil {
		return s.netting, nil
	}
	return &biz.FinanceNetting{ID: id}, nil
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
	if item.CommissionNo != "TC20260815000001" || item.VerificationNo == nil || *item.VerificationNo != "VR20260815000001" || item.NettingNo != nil ||
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
	verificationID := uuid.New().String()

	response, err := service.PreviewCommission(commissionPrincipalContext(org), &v1.PreviewCommissionRequest{
		VerificationId: &verificationID, EmployeeId: uuid.New().String(), RuleId: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PreviewCommission() error = %v", err)
	}
	// 原币记账口径：CNY 快照恒等固化（汇率 1、来源 BASE_CURRENCY、金额恒等）。
	if response.Data.CnyExchangeRate != "1.00000000" || response.Data.CnyExchangeRateSource != biz.CommissionCNYRateSourceBaseCurrency ||
		response.Data.CnyExchangeRateDate != "2026-08-15" || response.Data.CnyCommissionAmount != "100.00000000" {
		t.Fatalf("预览折算依据未返回: %#v", response.Data)
	}
	if response.Data.VerificationId == nil || *response.Data.VerificationId != verificationID || response.Data.NettingId != nil {
		t.Fatalf("预览来源字段不符: %#v", response.Data)
	}
}

func TestCommissionSourceExclusiveChoice(t *testing.T) {
	org := uuid.New()
	service, _ := newCommissionService(org)
	ctx := commissionPrincipalContext(org)
	verificationID, nettingID := uuid.New().String(), uuid.New().String()

	for name, request := range map[string]*v1.PreviewCommissionRequest{
		"预览来源双空": {EmployeeId: uuid.New().String(), RuleId: uuid.New().String()},
		"预览来源双填": {VerificationId: &verificationID, NettingId: &nettingID, EmployeeId: uuid.New().String(), RuleId: uuid.New().String()},
	} {
		if _, err := service.PreviewCommission(ctx, request); !errors.Is(err, biz.ErrCommissionInvalid) {
			t.Fatalf("%s 错误 = %v，期望 %v", name, err, biz.ErrCommissionInvalid)
		}
	}
	for name, request := range map[string]*v1.CreateCommissionRequest{
		"创建来源双空": {EmployeeId: uuid.New().String(), RuleId: uuid.New().String(), IdempotencyKey: "dup-src"},
		"创建来源双填": {VerificationId: &verificationID, NettingId: &nettingID, EmployeeId: uuid.New().String(), RuleId: uuid.New().String(), IdempotencyKey: "dup-src"},
	} {
		if _, err := service.CreateCommission(ctx, request); !errors.Is(err, biz.ErrCommissionInvalid) {
			t.Fatalf("%s 错误 = %v，期望 %v", name, err, biz.ErrCommissionInvalid)
		}
	}
}

func TestListCommissionNettingCandidatesUsesManageWritableOrganization(t *testing.T) {
	allowed, denied := uuid.New(), uuid.New()
	service, repo := newCommissionService(allowed)
	repo.nettingResult = &biz.CommissionNettingCandidateListResult{
		Items: []*biz.FinanceNetting{{
			ID: uuid.New(), OrganizationID: allowed, NettingNo: "NT202609100001", Status: biz.FinanceNettingConfirmed,
			SettlementPartyName: "测试对冲单位", Currency: "CNY", Amount: decimal.RequireFromString("60"),
			BaseCurrency: "CNY", BaseCurrencyAmount: decimal.RequireFromString("60"),
		}},
		Total: 1, Page: 1, PageSize: 20,
	}
	principal := &biz.Principal{
		UserID: uuid.New(), Organization: biz.Organization{ID: allowed}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: allowed}},
		RoleGrants: []biz.RoleGrant{{RoleCode: "commission-manager", DataScope: biz.DataScopeOrganization,
			Permissions: map[string]struct{}{access.FinanceCommissionManage: {}}}},
	}
	ctx := biz.WithPrincipal(context.Background(), principal)
	keyword := "NT2026"

	response, err := service.ListCommissionNettingCandidates(ctx, &v1.ListCommissionNettingCandidatesRequest{OrganizationId: allowed.String(), Page: 1, PageSize: 20, Keyword: &keyword})
	if err != nil {
		t.Fatalf("Manage-only 对冲候选查询失败: %v", err)
	}
	if len(response.Data) != 1 || response.Total != 1 || response.Data[0].NettingNo != "NT202609100001" ||
		response.Data[0].Status != v1.FinanceNettingStatus_FINANCE_NETTING_STATUS_CONFIRMED {
		t.Fatalf("对冲候选结果不符: %#v", response.Data)
	}
	if repo.nettingFilter.Page != 1 || repo.nettingFilter.PageSize != 20 || repo.nettingFilter.Keyword != "NT2026" {
		t.Fatalf("对冲候选筛选未透传: %+v", repo.nettingFilter)
	}

	if _, err := service.ListCommissionNettingCandidates(ctx, &v1.ListCommissionNettingCandidatesRequest{OrganizationId: denied.String(), Page: 1, PageSize: 20}); !errors.Is(err, biz.ErrPermissionDenied) {
		t.Fatalf("越权组织错误 = %v，期望 %v", err, biz.ErrPermissionDenied)
	}
	if _, err := service.ListCommissionNettingCandidates(ctx, &v1.ListCommissionNettingCandidatesRequest{Page: 1, PageSize: 20}); !errors.Is(err, biz.ErrCommissionInvalid) {
		t.Fatalf("空组织错误 = %v，期望 %v", err, biz.ErrCommissionInvalid)
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
		UserID: uuid.New(), Organization: biz.Organization{ID: allowed}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: allowed}},
		RoleGrants: []biz.RoleGrant{{RoleCode: "commission-manager", DataScope: biz.DataScopeOrganization,
			Permissions: map[string]struct{}{access.FinanceCommissionManage: {}}}},
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
		UserID: uuid.New(), Organization: biz.Organization{ID: allowed}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: allowed}},
		RoleGrants: []biz.RoleGrant{{RoleCode: "commission-manager", DataScope: biz.DataScopeOrganization,
			Permissions: map[string]struct{}{access.FinanceCommissionManage: {}}}},
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
