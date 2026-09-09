package biz

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type verificationTransactionContextKey struct{}

type verificationTransactorStub struct {
	calls int
}

func (s *verificationTransactorStub) WithinTransaction(ctx context.Context, operation func(context.Context) error) error {
	s.calls++
	return operation(context.WithValue(ctx, verificationTransactionContextKey{}, true))
}

func requireVerificationTransaction(ctx context.Context) error {
	if active, _ := ctx.Value(verificationTransactionContextKey{}).(bool); !active {
		return errors.New("操作未使用共享事务上下文")
	}
	return nil
}

type verificationTransactionRepoStub struct {
	VerificationRepo
	cashflow         *FinanceCashflow
	created          *FinanceVerification
	transactionCalls int
	responseReads    int
}

type verificationCandidateRepoStub struct {
	VerificationRepo
	organizationID uuid.UUID
	filter         VerificationCreationCandidateFilter
	result         *VerificationCreationCandidates
	err            error
}

type verificationListRepoStub struct {
	VerificationRepo
	organizationIDs []uuid.UUID
	filter          VerificationFilter
	err             error
}

func (s *verificationListRepoStub) ListScoped(_ context.Context, organizationIDs []uuid.UUID, filter VerificationFilter) (*VerificationListResult, error) {
	s.organizationIDs = organizationIDs
	s.filter = filter
	return &VerificationListResult{}, s.err
}

func (s *verificationCandidateRepoStub) ListCreationCandidates(_ context.Context, organizationID uuid.UUID, filter VerificationCreationCandidateFilter) (*VerificationCreationCandidates, error) {
	s.organizationID = organizationID
	s.filter = filter
	return s.result, s.err
}

func (s *verificationTransactionRepoStub) GetByKey(ctx context.Context, _ uuid.UUID, _ string) (*FinanceVerification, error) {
	if err := requireVerificationTransaction(ctx); err != nil {
		return nil, err
	}
	s.transactionCalls++
	return nil, nil
}

func (s *verificationTransactionRepoStub) LoadCashflowContext(ctx context.Context, _, _ uuid.UUID) (*FinanceCashflow, error) {
	if err := requireVerificationTransaction(ctx); err != nil {
		return nil, err
	}
	s.transactionCalls++
	return s.cashflow, nil
}

func (s *verificationTransactionRepoStub) Create(ctx context.Context, _, _ uuid.UUID, verification *FinanceVerification, _ *AuditEvent) (*FinanceVerification, error) {
	if err := requireVerificationTransaction(ctx); err != nil {
		return nil, err
	}
	s.transactionCalls++
	verification.VerificationNo = "WO-0001"
	s.created = verification
	return verification, nil
}

func (s *verificationTransactionRepoStub) Get(ctx context.Context, _, _ uuid.UUID) (*FinanceVerification, error) {
	if active, _ := ctx.Value(verificationTransactionContextKey{}).(bool); active {
		return nil, errors.New("完整核销响应不能在写事务内读取")
	}
	s.responseReads++
	return s.created, nil
}

type verificationExchangeRateTransactionStub struct {
	ExchangeRateRepo
	rateContext      *ExchangeRateContext
	resolved         *ResolvedExchangeRate
	transactionCalls int
}

func (s *verificationExchangeRateTransactionStub) ResolveContext(ctx context.Context, _ uuid.UUID) (*ExchangeRateContext, error) {
	if err := requireVerificationTransaction(ctx); err != nil {
		return nil, err
	}
	s.transactionCalls++
	return s.rateContext, nil
}

func (s *verificationExchangeRateTransactionStub) ListTimeStandards(ctx context.Context, _ uuid.UUID) ([]*ExchangeRateTimeStandardSetting, error) {
	if err := requireVerificationTransaction(ctx); err != nil {
		return nil, err
	}
	s.transactionCalls++
	return []*ExchangeRateTimeStandardSetting{{RateType: WriteOffRateType, TimeStandards: []string{WriteOffTimeStandard}}}, nil
}

func (s *verificationExchangeRateTransactionStub) Resolve(ctx context.Context, _ uuid.UUID, _ string, _ OrderFeeDirection, _, _, _ string) (*ResolvedExchangeRate, error) {
	if err := requireVerificationTransaction(ctx); err != nil {
		return nil, err
	}
	s.transactionCalls++
	return s.resolved, nil
}

func TestSameVerificationIntentIgnoresAllocationOrder(t *testing.T) {
	cashflowA, cashflowB := uuid.New(), uuid.New()
	billA, billB := uuid.New(), uuid.New()
	note := "同一请求"
	existing := &FinanceVerification{
		VerificationDate: "2026-08-26",
		Note:             &note,
		Allocations: []*VerificationAllocation{
			{CashflowID: cashflowA, BillID: billA, Amount: decimal.RequireFromString("10")},
			{CashflowID: cashflowB, BillID: billB, Amount: decimal.RequireFromString("20")},
		},
	}
	requested := CreateVerificationInput{
		VerificationDate: "2026-08-26",
		Note:             &note,
		Allocations: []*VerificationAllocation{
			{CashflowID: cashflowB, BillID: billB, Amount: decimal.RequireFromString("20.00000000")},
			{CashflowID: cashflowA, BillID: billA, Amount: decimal.RequireFromString("10.0")},
		},
	}
	if !sameVerificationIntent(existing, requested) {
		t.Fatal("相同分配集合应被识别为同一幂等请求")
	}
	requested.Allocations[0].Amount = decimal.RequireFromString("20.01")
	if sameVerificationIntent(existing, requested) {
		t.Fatal("金额不同的请求不应复用已有核销")
	}
}

func TestCalculateVerificationAllocationAmountsRecognizesExchangeGainLoss(t *testing.T) {
	amount := decimal.RequireFromString("40")
	billTotal := decimal.RequireFromString("100")
	billBaseTotal := decimal.RequireFromString("720")
	cashflowTotal := decimal.RequireFromString("100")
	cashflowBaseTotal := decimal.RequireFromString("725")
	writeOffRate := decimal.RequireFromString("7.30")

	billBase, cashBase, writeOffBase, receivableGainLoss, err := CalculateVerificationAllocationAmounts(OrderFeeReceivable, amount, billTotal, billBaseTotal, cashflowTotal, cashflowBaseTotal, writeOffRate)
	if err != nil {
		t.Fatalf("计算应收核销汇兑损益失败: %v", err)
	}
	if billBase.StringFixed(8) != "288.00000000" || cashBase.StringFixed(8) != "290.00000000" || writeOffBase.StringFixed(8) != "292.00000000" || receivableGainLoss.StringFixed(8) != "2.00000000" {
		t.Fatalf("应收核销汇率快照不正确: bill=%s cash=%s writeOff=%s gainLoss=%s", billBase, cashBase, writeOffBase, receivableGainLoss)
	}

	_, _, _, payableGainLoss, err := CalculateVerificationAllocationAmounts(OrderFeePayable, amount, billTotal, billBaseTotal, cashflowTotal, cashflowBaseTotal, writeOffRate)
	if err != nil {
		t.Fatalf("计算应付核销汇兑损益失败: %v", err)
	}
	if payableGainLoss.StringFixed(8) != "-2.00000000" {
		t.Fatalf("应付核销时资金本币金额高于账面金额应形成汇兑损失，实际 %s", payableGainLoss)
	}
}

func TestVerificationCreateUsesOneSharedTransaction(t *testing.T) {
	organizationID := uuid.New()
	actorID := uuid.New()
	cashflowID := uuid.New()
	billID := uuid.New()
	settingID := uuid.New()
	repo := &verificationTransactionRepoStub{cashflow: &FinanceCashflow{
		ID: cashflowID, Direction: OrderFeeReceivable, Currency: "USD", BaseCurrency: "CNY",
	}}
	exchangeRepo := &verificationExchangeRateTransactionStub{
		rateContext: &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"},
		resolved:    &ResolvedExchangeRate{Rate: decimal.RequireFromString("7.30"), Source: "SYSTEM", RateDate: "2026-08-30", SettingID: &settingID},
	}
	transactor := &verificationTransactorStub{}
	usecase := NewVerificationUsecase(repo, NewExchangeRateUsecase(exchangeRepo), transactor)

	created, err := usecase.Create(context.Background(), organizationID, actorID, CreateVerificationInput{
		Allocations:      []*VerificationAllocation{{CashflowID: cashflowID, BillID: billID, Amount: decimal.RequireFromString("40")}},
		VerificationDate: "2026-08-30",
		IdempotencyKey:   "verification-transaction-test",
	})
	if err != nil {
		t.Fatalf("创建核销失败: %v", err)
	}
	if created.VerificationNo != "WO-0001" || created.ExchangeRate.StringFixed(8) != "7.30000000" || created.BaseAmount.StringFixed(8) != "292.00000000" {
		t.Fatalf("核销创建结果不正确: %#v", created)
	}
	if transactor.calls != 1 {
		t.Fatalf("共享事务调用次数 = %d，期望 1", transactor.calls)
	}
	if repo.transactionCalls != 3 {
		t.Fatalf("核销仓储事务内调用次数 = %d，期望 3", repo.transactionCalls)
	}
	if repo.responseReads != 1 {
		t.Fatalf("提交后核销读取次数 = %d，期望 1", repo.responseReads)
	}
	if exchangeRepo.transactionCalls != 4 {
		t.Fatalf("汇率仓储事务内调用次数 = %d，期望 4", exchangeRepo.transactionCalls)
	}
}

func TestVerificationCreationCandidatesUseOneExplicitOrganizationAndPreserveRepositoryError(t *testing.T) {
	organizationID := uuid.New()
	partyID := uuid.New()
	expected := errors.New("候选查询失败")
	repo := &verificationCandidateRepoStub{err: expected}
	usecase := NewVerificationUsecase(repo, nil, nil)

	_, err := usecase.ListCreationCandidates(context.Background(), organizationID, VerificationCreationCandidateFilter{
		Direction:         OrderFeeReceivable,
		SettlementPartyID: partyID,
		Currency:          " usd ",
	})
	if !errors.Is(err, expected) {
		t.Fatalf("候选查询错误 = %v，期望原样返回 %v", err, expected)
	}
	if repo.organizationID != organizationID || repo.filter.Direction != OrderFeeReceivable || repo.filter.SettlementPartyID != partyID || repo.filter.Currency != "USD" {
		t.Fatalf("候选组织或筛选条件未完整传递: org=%s filter=%+v", repo.organizationID, repo.filter)
	}
	if _, err := usecase.ListCreationCandidates(context.Background(), uuid.Nil, VerificationCreationCandidateFilter{Direction: OrderFeeReceivable, SettlementPartyID: partyID, Currency: "USD"}); err != ErrVerificationInvalid {
		t.Fatalf("空组织错误 = %v，期望 %v", err, ErrVerificationInvalid)
	}
	if _, err := usecase.ListCreationCandidates(context.Background(), organizationID, VerificationCreationCandidateFilter{Direction: OrderFeeReceivable, Currency: "USD"}); err != ErrVerificationInvalid {
		t.Fatalf("空结算单位错误 = %v，期望 %v", err, ErrVerificationInvalid)
	}
}

func TestVerificationListScopedValidatesDirectionAndPreservesRepositoryError(t *testing.T) {
	organizationID := uuid.New()
	expected := errors.New("核销列表查询失败")
	repo := &verificationListRepoStub{err: expected}
	usecase := NewVerificationUsecase(repo, nil, nil)
	filter := VerificationFilter{Page: 1, PageSize: 20, Direction: OrderFeeReceivable}
	if _, err := usecase.ListScoped(context.Background(), []uuid.UUID{organizationID}, filter); !errors.Is(err, expected) {
		t.Fatalf("列表错误 = %v，期望原样返回 %v", err, expected)
	}
	if len(repo.organizationIDs) != 1 || repo.organizationIDs[0] != organizationID || repo.filter.Direction != OrderFeeReceivable {
		t.Fatalf("组织或方向未完整传给仓储: ids=%v filter=%+v", repo.organizationIDs, repo.filter)
	}
	if _, err := usecase.ListScoped(context.Background(), []uuid.UUID{organizationID}, VerificationFilter{Page: 1, PageSize: 20, Direction: OrderFeeDirection("OTHER")}); !errors.Is(err, ErrVerificationInvalid) {
		t.Fatalf("非法方向错误 = %v，期望 %v", err, ErrVerificationInvalid)
	}
}

var _ Transactor = (*verificationTransactorStub)(nil)
