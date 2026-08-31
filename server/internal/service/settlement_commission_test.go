package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// commissionRepoStub 捕获列表筛选并返回固定的提成与预览结果，不触发数据库。
type commissionRepoStub struct {
	biz.CommissionRepo
	filter      biz.CommissionFilter
	listResult  *biz.CommissionListResult
	preview     *biz.CommissionCalculation
	generation  *biz.CommissionGenerationContext
	rateContext *biz.ExchangeRateContext
	rateStub    *exchangeRateStub
}

func (s *commissionRepoStub) List(_ context.Context, _ uuid.UUID, f biz.CommissionFilter) (*biz.CommissionListResult, error) {
	s.filter = f
	return s.listResult, nil
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
