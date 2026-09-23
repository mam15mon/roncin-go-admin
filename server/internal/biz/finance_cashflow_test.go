package biz

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type financeCashflowRepoStub struct {
	existing *FinanceCashflow
	created  *FinanceCashflow
	isCasual bool
}

func (*financeCashflowRepoStub) List(context.Context, uuid.UUID, FinanceCashflowFilter) (*FinanceCashflowListResult, error) {
	return nil, nil
}
func (*financeCashflowRepoStub) ListScoped(context.Context, []uuid.UUID, FinanceCashflowFilter) (*FinanceCashflowListResult, error) {
	return nil, nil
}
func (*financeCashflowRepoStub) Get(context.Context, uuid.UUID, uuid.UUID) (*FinanceCashflow, error) {
	return nil, nil
}
func (*financeCashflowRepoStub) GetScoped(context.Context, []uuid.UUID, uuid.UUID) (*FinanceCashflow, error) {
	return nil, nil
}
func (s *financeCashflowRepoStub) GetByIdempotencyKey(context.Context, uuid.UUID, string) (*FinanceCashflow, error) {
	return s.existing, nil
}
func (s *financeCashflowRepoStub) ResolveParty(context.Context, uuid.UUID, uuid.UUID) (string, bool, error) {
	return "测试结算单位", s.isCasual, nil
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

func TestCreateFinanceCashflowUsesTransactionDateRateSnapshot(t *testing.T) {
	exchangeRepo := &exchangeRateRepoStub{
		rateContext:    &ExchangeRateContext{OwnerOrganizationID: uuid.New(), BaseCurrency: "CNY"},
		rateByCurrency: map[string]decimal.Decimal{"USD": decimal.RequireFromString("7.25")},
	}
	repo := &financeCashflowRepoStub{}
	usecase := NewFinanceCashflowUsecase(repo, NewExchangeRateUsecase(exchangeRepo, nil))
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
		t.Fatalf("按到账日汇率创建资金流水失败: %v", err)
	}
	if item.ExchangeRate.StringFixed(8) != "7.25000000" || item.BaseCurrency != "CNY" || item.BaseAmount.StringFixed(8) != "725.00000000" || item.ExchangeRateSource != ExchangeRateSourceWeekly || item.ExchangeRateDate != "2026-08-27" || item.ExchangeRateSettingID != nil {
		t.Fatalf("资金流水汇率快照不完整: %#v", item)
	}
	if len(exchangeRepo.resolveDates) != 1 || exchangeRepo.resolveDates[0] != "2026-08-27" {
		t.Fatalf("资金流水汇率必须按到账日解析: %v", exchangeRepo.resolveDates)
	}
}

func TestCreateFinanceCashflowRejectsUnauthorizedRateOverride(t *testing.T) {
	exchangeRepo := &exchangeRateRepoStub{
		rateContext:    &ExchangeRateContext{OwnerOrganizationID: uuid.New(), BaseCurrency: "CNY"},
		rateByCurrency: map[string]decimal.Decimal{"USD": decimal.RequireFromString("7.25")},
	}
	usecase := NewFinanceCashflowUsecase(&financeCashflowRepoStub{}, NewExchangeRateUsecase(exchangeRepo, nil))
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

func TestCreateFinanceCashflowAllowsAuthorizedManualRateWithoutWeeklyRow(t *testing.T) {
	companyID := uuid.New()
	repo := &financeCashflowRepoStub{}
	rates := NewExchangeRateUsecase(&exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: companyID, BaseCurrency: "CNY"}}, nil)
	usecase := NewFinanceCashflowUsecase(repo, rates)
	manualRate := decimal.RequireFromString("7.30")
	item, err := usecase.Create(context.Background(), companyID, uuid.New(), CreateFinanceCashflowInput{
		Direction: OrderFeeReceivable, SettlementPartyID: uuid.New(), Currency: "USD", Amount: decimal.RequireFromString("100"),
		ExchangeRateOverride: &manualRate, TransactionDate: "2026-08-27", OurAccount: "基本户", PaymentMethod: "银行转账", IdempotencyKey: "cashflow-manual-without-week",
	}, true)
	if err != nil || item.ExchangeRateSource != ExchangeRateSourceManual || !item.ExchangeRate.Equal(manualRate) || item.ExchangeRateSettingID != nil || repo.created != item {
		t.Fatalf("有权限手工汇率应无需周配置并保存 MANUAL 快照: item=%+v err=%v", item, err)
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
	usecase := NewFinanceCashflowUsecase(repo, NewExchangeRateUsecase(&exchangeRateRepoStub{}, nil))
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
	if err != ErrFinanceCashflowRateOverrideForbidden || replayed != nil {
		t.Fatalf("无权限手工汇率即使幂等重放也应拒绝: item=%v err=%v", replayed, err)
	}
	replayed, err = usecase.Create(context.Background(), organizationID, actorID, CreateFinanceCashflowInput{
		Direction: OrderFeeReceivable, SettlementPartyID: partyID, Currency: "USD",
		Amount: decimal.RequireFromString("100"), ExchangeRateOverride: &override,
		TransactionDate: "2026-08-27", OurAccount: "基本户", PaymentMethod: "银行转账", IdempotencyKey: "cashflow-rate-replay",
	}, true)
	if err != nil || replayed != existing {
		t.Fatalf("有权限的等值请求应直接幂等重放: item=%v err=%v", replayed, err)
	}
}

func TestFinanceCashflow_CasualSupplierAccountConstraint(t *testing.T) {
	organizationID := uuid.New()
	actorID := uuid.New()
	partyID := uuid.New()
	exchangeRepo := &exchangeRateRepoStub{
		rateContext: &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"},
	}
	rateUsecase := NewExchangeRateUsecase(exchangeRepo, nil)

	t.Run("散客供应商付款无对方账户直接拒绝", func(t *testing.T) {
		repo := &financeCashflowRepoStub{isCasual: true}
		usecase := NewFinanceCashflowUsecase(repo, rateUsecase)
		_, err := usecase.Create(context.Background(), organizationID, actorID, CreateFinanceCashflowInput{
			Direction:         OrderFeePayable,
			SettlementPartyID: partyID,
			Currency:          "CNY",
			Amount:            decimal.RequireFromString("500"),
			TransactionDate:   "2026-08-27",
			OurAccount:        "基本户",
			PaymentMethod:     "银行转账",
			IdempotencyKey:    "casual-payable-no-account",
		}, false)
		if !errors.Is(err, ErrFinanceCashflowCasualSupplierAccountRequired) {
			t.Fatalf("散客供应商出款无账户应返回 ErrFinanceCashflowCasualSupplierAccountRequired, 得到: %v", err)
		}
	})

	t.Run("散客供应商付款对方账户为空白字符时直接拒绝", func(t *testing.T) {
		repo := &financeCashflowRepoStub{isCasual: true}
		usecase := NewFinanceCashflowUsecase(repo, rateUsecase)
		blank := "   "
		_, err := usecase.Create(context.Background(), organizationID, actorID, CreateFinanceCashflowInput{
			Direction:           OrderFeePayable,
			SettlementPartyID:   partyID,
			Currency:            "CNY",
			Amount:              decimal.RequireFromString("500"),
			TransactionDate:     "2026-08-27",
			OurAccount:          "基本户",
			PaymentMethod:       "银行转账",
			CounterpartyAccount: &blank,
			IdempotencyKey:      "casual-payable-blank-account",
		}, false)
		if !errors.Is(err, ErrFinanceCashflowCasualSupplierAccountRequired) {
			t.Fatalf("散客供应商出款账户为空白应返回 ErrFinanceCashflowCasualSupplierAccountRequired, 得到: %v", err)
		}
	})

	t.Run("散客供应商付款有对方账户允许成功", func(t *testing.T) {
		repo := &financeCashflowRepoStub{isCasual: true}
		usecase := NewFinanceCashflowUsecase(repo, rateUsecase)
		acc := "6222021234567890"
		created, err := usecase.Create(context.Background(), organizationID, actorID, CreateFinanceCashflowInput{
			Direction:           OrderFeePayable,
			SettlementPartyID:   partyID,
			Currency:            "CNY",
			Amount:              decimal.RequireFromString("500"),
			TransactionDate:     "2026-08-27",
			OurAccount:          "基本户",
			PaymentMethod:       "银行转账",
			CounterpartyAccount: &acc,
			IdempotencyKey:      "casual-payable-with-account",
		}, false)
		if err != nil {
			t.Fatalf("散客供应商出款有账户应创建成功: %v", err)
		}
		if created == nil || created.CounterpartyAccount == nil || *created.CounterpartyAccount != acc {
			t.Fatal("创建结果应正确保留对方账户")
		}
	})

	t.Run("正式供应商付款无对方账户不受影响", func(t *testing.T) {
		repo := &financeCashflowRepoStub{isCasual: false}
		usecase := NewFinanceCashflowUsecase(repo, rateUsecase)
		created, err := usecase.Create(context.Background(), organizationID, actorID, CreateFinanceCashflowInput{
			Direction:         OrderFeePayable,
			SettlementPartyID: partyID,
			Currency:          "CNY",
			Amount:            decimal.RequireFromString("500"),
			TransactionDate:   "2026-08-27",
			OurAccount:        "基本户",
			PaymentMethod:     "银行转账",
			IdempotencyKey:    "formal-payable-no-account",
		}, false)
		if err != nil {
			t.Fatalf("正式供应商出款无账户应正常创建: %v", err)
		}
		if created == nil {
			t.Fatal("创建结果不应为空")
		}
	})

	t.Run("散客客户收款无对方账户不受影响", func(t *testing.T) {
		repo := &financeCashflowRepoStub{isCasual: true}
		usecase := NewFinanceCashflowUsecase(repo, rateUsecase)
		created, err := usecase.Create(context.Background(), organizationID, actorID, CreateFinanceCashflowInput{
			Direction:         OrderFeeReceivable,
			SettlementPartyID: partyID,
			Currency:          "CNY",
			Amount:            decimal.RequireFromString("500"),
			TransactionDate:   "2026-08-27",
			OurAccount:        "基本户",
			PaymentMethod:     "银行转账",
			IdempotencyKey:    "casual-receivable-no-account",
		}, false)
		if err != nil {
			t.Fatalf("散客客户收款方向无账户应正常创建: %v", err)
		}
		if created == nil {
			t.Fatal("创建结果不应为空")
		}
	})
}
