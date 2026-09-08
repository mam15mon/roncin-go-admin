package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type financeBillTargetOrganizationRepoStub struct {
	FinanceBillRepo
	bill         *FinanceBill
	updateScopes []uuid.UUID
	updateInput  UpdateFinanceBillInput
	updateAudit  *AuditEvent
}

func (s *financeBillTargetOrganizationRepoStub) Get(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (*FinanceBill, error) {
	return s.bill, nil
}

func (s *financeBillTargetOrganizationRepoStub) Update(_ context.Context, organizationIDs []uuid.UUID, input UpdateFinanceBillInput, audit *AuditEvent) (*FinanceBill, error) {
	s.updateScopes = organizationIDs
	s.updateInput = input
	s.updateAudit = audit
	return s.bill, nil
}

type financeBillTargetExchangeRateRepoStub struct {
	ExchangeRateRepo
	resolvedContextOrganizationID uuid.UUID
	resolvedRateOrganizationID    uuid.UUID
	resolved                      *ResolvedExchangeRate
}

func (s *financeBillTargetExchangeRateRepoStub) ResolveContext(_ context.Context, organizationID uuid.UUID) (*ExchangeRateContext, error) {
	s.resolvedContextOrganizationID = organizationID
	return &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"}, nil
}

func (*financeBillTargetExchangeRateRepoStub) ListTimeStandards(context.Context, uuid.UUID) ([]*ExchangeRateTimeStandardSetting, error) {
	return []*ExchangeRateTimeStandardSetting{{RateType: BillRateType, TimeStandards: []string{BillDateStandard}}}, nil
}

func (s *financeBillTargetExchangeRateRepoStub) Resolve(_ context.Context, organizationID uuid.UUID, _ string, _ OrderFeeDirection, _, _, _ string) (*ResolvedExchangeRate, error) {
	s.resolvedRateOrganizationID = organizationID
	return s.resolved, nil
}

func TestApplyBillExchangeRateUsesBillDateSnapshot(t *testing.T) {
	organizationID := uuid.New()
	settingID := uuid.New()
	exchangeRepo := &exchangeRateRepoStub{
		rateContext:   &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"},
		timeStandards: []*ExchangeRateTimeStandardSetting{{RateType: BillRateType, TimeStandards: []string{BillDateStandard}}},
		resolved:      &ResolvedExchangeRate{Rate: decimal.RequireFromString("7.20"), Source: "SYSTEM", RateDate: "2026-08-26", SettingID: &settingID},
	}
	usecase := NewFinanceBillUsecase(nil, NewExchangeRateUsecase(exchangeRepo), &financeBillTransactorStub{})
	bill := &FinanceBill{
		Direction:    OrderFeeReceivable,
		Currency:     "USD",
		BaseCurrency: "CNY",
		TotalAmount:  decimal.RequireFromString("100"),
		BillDate:     "2026-08-26",
	}

	if err := usecase.applyBillExchangeRate(context.Background(), organizationID, bill); err != nil {
		t.Fatalf("应用账单日汇率失败: %v", err)
	}
	if bill.ExchangeRate.StringFixed(8) != "7.20000000" || bill.BaseCurrencyAmount.StringFixed(8) != "720.00000000" || bill.ExchangeRateSource != "SYSTEM" || bill.ExchangeRateDate != "2026-08-26" || bill.ExchangeRateSettingID == nil || *bill.ExchangeRateSettingID != settingID {
		t.Fatalf("账单汇率快照不完整: %#v", bill)
	}
}

func TestFinanceBillUpdateUsesTargetOrganizationForExchangeRateAndAudit(t *testing.T) {
	currentOrganizationID := uuid.New()
	targetOrganizationID := uuid.New()
	actorID := uuid.New()
	billID := uuid.New()
	settingID := uuid.New()
	billRepo := &financeBillTargetOrganizationRepoStub{bill: &FinanceBill{
		ID: billID, OrganizationID: targetOrganizationID, Direction: OrderFeeReceivable,
		Currency: "USD", TotalAmount: decimal.NewFromInt(100), Version: 2,
	}}
	exchangeRepo := &financeBillTargetExchangeRateRepoStub{resolved: &ResolvedExchangeRate{
		Rate: decimal.RequireFromString("7.20"), Source: "SYSTEM", RateDate: "2026-09-09", SettingID: &settingID,
	}}
	usecase := NewFinanceBillUsecase(billRepo, NewExchangeRateUsecase(exchangeRepo), nil)

	_, err := usecase.Update(t.Context(), []uuid.UUID{currentOrganizationID, targetOrganizationID}, actorID, UpdateFinanceBillInput{
		ID: billID, BillDate: "2026-09-09", ExpectedVersion: 2,
	})
	if err != nil {
		t.Fatalf("跨组织账单更新失败: %v", err)
	}
	if exchangeRepo.resolvedContextOrganizationID != targetOrganizationID || exchangeRepo.resolvedRateOrganizationID != targetOrganizationID {
		t.Fatalf("汇率必须按目标账单组织解析，context=%s rate=%s target=%s", exchangeRepo.resolvedContextOrganizationID, exchangeRepo.resolvedRateOrganizationID, targetOrganizationID)
	}
	if billRepo.updateAudit == nil || billRepo.updateAudit.OrganizationID == nil || *billRepo.updateAudit.OrganizationID != targetOrganizationID {
		t.Fatalf("更新审计必须记录目标账单组织，audit=%#v target=%s", billRepo.updateAudit, targetOrganizationID)
	}
	if billRepo.updateInput.ExchangeRate.StringFixed(8) != "7.20000000" || billRepo.updateInput.BaseCurrencyAmount.StringFixed(8) != "720.00000000" {
		t.Fatalf("目标组织汇率快照未传入仓储: %#v", billRepo.updateInput)
	}
}
