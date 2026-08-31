package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// commissionRepoStub 捕获列表筛选并返回固定的提成与预览结果，不触发数据库。
type commissionRepoStub struct {
	biz.CommissionRepo
	filter         biz.CommissionFilter
	listResult     *biz.CommissionListResult
	preview        *biz.CommissionCalculation
	generation     *biz.CommissionGenerationContext
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

func (s *commissionRepoStub) Count(_ context.Context, _ uuid.UUID, f biz.CommissionFilter) (int64, error) {
	s.exportFilter = f
	return s.exportTotal, nil
}

func (s *commissionRepoStub) ExportBatch(_ context.Context, _ uuid.UUID, f biz.CommissionFilter) ([]*biz.FinanceCommission, error) {
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

// exchangeRateStub 模拟 CNY→本位币的核销汇率解析。
type exchangeRateStub struct {
	biz.ExchangeRateRepo
	context  *biz.ExchangeRateContext
	resolved *biz.ResolvedExchangeRate
}

func (s *exchangeRateStub) ResolveContext(context.Context, uuid.UUID) (*biz.ExchangeRateContext, error) {
	return s.context, nil
}

func (s *exchangeRateStub) ListTimeStandards(context.Context, uuid.UUID) ([]*biz.ExchangeRateTimeStandardSetting, error) {
	return []*biz.ExchangeRateTimeStandardSetting{{RateType: biz.WriteOffRateType, TimeStandards: []string{biz.WriteOffTimeStandard}}}, nil
}

func (s *exchangeRateStub) Resolve(context.Context, uuid.UUID, string, biz.OrderFeeDirection, string, string, string) (*biz.ResolvedExchangeRate, error) {
	return s.resolved, nil
}

func newCommissionService(org uuid.UUID) (*SettlementService, *commissionRepoStub) {
	settingID := uuid.New()
	setting := &settingID
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
			CNYExchangeRateDate:          "2026-08-14",
			CNYExchangeRateSettingID:     setting,
			CNYCommissionAmount:          decimal.RequireFromString("200"),
			CNYAdjustmentAmount:          decimal.RequireFromString("20"),
			CNYEffectiveCommissionAmount: decimal.RequireFromString("220"),
			AdjustmentAmount:             decimal.RequireFromString("10"),
			EffectiveCommissionAmount:    decimal.RequireFromString("110"),
		}}},
		preview:    &biz.CommissionCalculation{BaseCurrency: "USD", CommissionAmount: decimal.RequireFromString("100")},
		generation: &biz.CommissionGenerationContext{CommissionDate: "2026-08-15", ExchangeRateDate: "2026-08-14", BaseCurrency: "USD"},
		rateStub: &exchangeRateStub{
			context:  &biz.ExchangeRateContext{OwnerOrganizationID: org, BaseCurrency: "USD"},
			resolved: &biz.ResolvedExchangeRate{Rate: decimal.RequireFromString("0.5"), RateDate: "2026-08-14", SettingID: setting},
		},
	}
	usecase := biz.NewCommissionUsecase(repo, nil, biz.NewExchangeRateUsecase(repo.rateStub), &commissionTransactorStub{})
	service := NewSettlementService(nil, nil, nil, nil, nil, usecase, nil, nil, nil)
	return service, repo
}

type commissionTransactorStub struct{}

func (commissionTransactorStub) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func commissionPrincipalContext(org uuid.UUID) context.Context {
	return biz.WithPrincipal(context.Background(), &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: org}})
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
		item.CnyExchangeRateSource != biz.CommissionCNYRateSourceDerived || item.CnyExchangeRateDate != "2026-08-14" ||
		item.CnyExchangeRateSettingId == nil || item.CnyCommissionAmount != "200.00000000" ||
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

	response, err := service.ExportCommissions(biz.WithPrincipal(context.Background(), &biz.Principal{UserID: actor, Organization: biz.Organization{ID: org}}), &v1.ExportCommissionsRequest{
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
		item.RatePercent != "2.5000" || item.BaseCurrency != "USD" || item.CreatedAt != "2026-08-15T08:00:00Z" {
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
		response.Data.CnyExchangeRateDate != "2026-08-14" || response.Data.CnyCommissionAmount != "200.00000000" {
		t.Fatalf("预览折算依据未返回: %#v", response.Data)
	}
}

func ptrString(value string) *string { return &value }
