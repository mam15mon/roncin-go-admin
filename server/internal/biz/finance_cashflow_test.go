package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type financeCashflowRepoStub struct {
	created  *FinanceCashflow
	existing *FinanceCashflow
}

func (*financeCashflowRepoStub) List(context.Context, uuid.UUID, FinanceCashflowFilter) (*FinanceCashflowListResult, error) {
	return nil, nil
}
func (*financeCashflowRepoStub) Get(context.Context, uuid.UUID, uuid.UUID) (*FinanceCashflow, error) {
	return nil, nil
}
func (s *financeCashflowRepoStub) GetByIdempotencyKey(context.Context, uuid.UUID, string) (*FinanceCashflow, error) {
	return s.existing, nil
}
func (*financeCashflowRepoStub) ResolveParty(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "测试结算单位", nil
}
func (s *financeCashflowRepoStub) Create(_ context.Context, item *FinanceCashflow, _ *AuditEvent) (*FinanceCashflow, error) {
	s.created = item
	return item, nil
}
func (*financeCashflowRepoStub) Confirm(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, *AuditEvent) (*FinanceCashflow, error) {
	return nil, nil
}
func (*financeCashflowRepoStub) Cancel(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, string, *AuditEvent) (*FinanceCashflow, error) {
	return nil, nil
}

func TestSameFinanceCashflowIntent(t *testing.T) {
	partnerID := uuid.New()
	counterpartyAccount := "客户账户"
	bankReferenceNo := "BANK-20260827-001"
	note := "同一笔收款"
	old := &FinanceCashflow{
		Direction:           OrderFeeReceivable,
		SettlementPartyID:   partnerID,
		Currency:            "CNY",
		Amount:              decimal.RequireFromString("100.00000000"),
		ExchangeRate:        decimal.RequireFromString("1.00000000"),
		BaseCurrency:        "CNY",
		TransactionDate:     "2026-08-27",
		OurAccount:          "基本户",
		PaymentMethod:       "银行转账",
		CounterpartyAccount: &counterpartyAccount,
		BankReferenceNo:     &bankReferenceNo,
		Note:                &note,
	}
	rateOverride := decimal.RequireFromString("1")
	requested := CreateFinanceCashflowInput{
		Direction:            OrderFeeReceivable,
		SettlementPartyID:    partnerID,
		Currency:             "CNY",
		Amount:               decimal.RequireFromString("100"),
		ExchangeRateOverride: &rateOverride,
		BaseCurrency:         "CNY",
		TransactionDate:      "2026-08-27",
		OurAccount:           "基本户",
		PaymentMethod:        "银行转账",
		CounterpartyAccount:  &counterpartyAccount,
		BankReferenceNo:      &bankReferenceNo,
		Note:                 &note,
	}
	if !sameFinanceCashflowIntent(old, requested) {
		t.Fatal("等值金额和相同字段应识别为同一幂等请求")
	}
	requested.Amount = decimal.RequireFromString("100.01")
	if sameFinanceCashflowIntent(old, requested) {
		t.Fatal("金额不同的请求不应复用已有资金流水")
	}
}

func TestCreateFinanceCashflowUsesSettlementRateSnapshot(t *testing.T) {
	settingID := uuid.New()
	exchangeRepo := &exchangeRateRepoStub{
		rateContext:   &ExchangeRateContext{OwnerOrganizationID: uuid.New(), BaseCurrency: "CNY"},
		timeStandards: []*ExchangeRateTimeStandardSetting{{RateType: SettlementRateType, TimeStandards: []string{TransactionDateStandard}}},
		resolved:      &ResolvedExchangeRate{Rate: decimal.RequireFromString("7.25"), Source: "SYSTEM", RateDate: "2026-08-27", SettingID: &settingID},
	}
	repo := &financeCashflowRepoStub{}
	usecase := NewFinanceCashflowUsecase(repo, NewExchangeRateUsecase(exchangeRepo))
	item, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), CreateFinanceCashflowInput{
		Direction:         OrderFeeReceivable,
		SettlementPartyID: uuid.New(),
		Currency:          "USD",
		Amount:            decimal.RequireFromString("100"),
		TransactionDate:   "2026-08-27",
		OurAccount:        "基本户",
		PaymentMethod:     "银行转账",
		IdempotencyKey:    "cashflow-rate-snapshot",
	}, false)
	if err != nil {
		t.Fatalf("按结算日汇率创建资金流水失败: %v", err)
	}
	if item.ExchangeRate.StringFixed(8) != "7.25000000" || item.BaseCurrency != "CNY" || item.BaseAmount.StringFixed(8) != "725.00000000" || item.ExchangeRateSource != "SYSTEM" || item.ExchangeRateSettingID == nil || *item.ExchangeRateSettingID != settingID {
		t.Fatalf("资金流水汇率快照不完整: %#v", item)
	}
}

func TestCreateFinanceCashflowRejectsUnauthorizedRateOverride(t *testing.T) {
	exchangeRepo := &exchangeRateRepoStub{
		rateContext:   &ExchangeRateContext{OwnerOrganizationID: uuid.New(), BaseCurrency: "CNY"},
		timeStandards: []*ExchangeRateTimeStandardSetting{{RateType: SettlementRateType, TimeStandards: []string{TransactionDateStandard}}},
		resolved:      &ResolvedExchangeRate{Rate: decimal.RequireFromString("7.25"), Source: "SYSTEM", RateDate: "2026-08-27"},
	}
	usecase := NewFinanceCashflowUsecase(&financeCashflowRepoStub{}, NewExchangeRateUsecase(exchangeRepo))
	override := decimal.RequireFromString("7.30")
	_, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), CreateFinanceCashflowInput{
		Direction:            OrderFeeReceivable,
		SettlementPartyID:    uuid.New(),
		Currency:             "USD",
		Amount:               decimal.RequireFromString("100"),
		ExchangeRateOverride: &override,
		TransactionDate:      "2026-08-27",
		OurAccount:           "基本户",
		PaymentMethod:        "银行转账",
		IdempotencyKey:       "cashflow-rate-override",
	}, false)
	if err != ErrFinanceCashflowRateOverrideForbidden {
		t.Fatalf("无权限覆盖资金汇率应被拒绝，实际错误为 %v", err)
	}
}

func TestCreateFinanceCashflowReplaysBeforeResolvingCurrentRate(t *testing.T) {
	organizationID := uuid.New()
	actorID := uuid.New()
	partyID := uuid.New()
	existing := &FinanceCashflow{
		OrganizationID:    organizationID,
		IdempotencyKey:    "cashflow-rate-replay",
		Direction:         OrderFeeReceivable,
		SettlementPartyID: partyID,
		Currency:          "USD",
		Amount:            decimal.RequireFromString("100"),
		ExchangeRate:      decimal.RequireFromString("7.20"),
		BaseCurrency:      "CNY",
		TransactionDate:   "2026-08-27",
		OurAccount:        "基本户",
		PaymentMethod:     "银行转账",
	}
	repo := &financeCashflowRepoStub{existing: existing}
	usecase := NewFinanceCashflowUsecase(repo, NewExchangeRateUsecase(&exchangeRateRepoStub{}))
	override := decimal.RequireFromString("7.20")

	replayed, err := usecase.Create(context.Background(), organizationID, actorID, CreateFinanceCashflowInput{
		Direction:            OrderFeeReceivable,
		SettlementPartyID:    partyID,
		Currency:             "USD",
		Amount:               decimal.RequireFromString("100"),
		ExchangeRateOverride: &override,
		TransactionDate:      "2026-08-27",
		OurAccount:           "基本户",
		PaymentMethod:        "银行转账",
		IdempotencyKey:       "cashflow-rate-replay",
	}, false)
	if err != nil {
		t.Fatalf("相同流水请求应直接幂等重放: %v", err)
	}
	if replayed != existing {
		t.Fatal("幂等重放应返回原资金流水")
	}
}
