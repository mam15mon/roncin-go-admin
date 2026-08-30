package biz

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type financeBillTransactionContextKey struct{}

type financeBillTransactorStub struct {
	calls int
}

func (s *financeBillTransactorStub) WithinTransaction(ctx context.Context, operation func(context.Context) error) error {
	s.calls++
	return operation(context.WithValue(ctx, financeBillTransactionContextKey{}, true))
}

func requireFinanceBillTransaction(ctx context.Context) error {
	if active, _ := ctx.Value(financeBillTransactionContextKey{}).(bool); !active {
		return errors.New("操作未使用共享事务上下文")
	}
	return nil
}

type financeBillTransactionRepoStub struct {
	FinanceBillRepo
	fee              *FinanceBillableFee
	created          *FinanceBill
	transactionCalls int
	responseReads    int
}

func (s *financeBillTransactionRepoStub) GetByIdempotencyKey(ctx context.Context, _ uuid.UUID, _ string) (*FinanceBill, error) {
	if err := requireFinanceBillTransaction(ctx); err != nil {
		return nil, err
	}
	s.transactionCalls++
	return nil, nil
}

func (s *financeBillTransactionRepoStub) LoadBillableFees(ctx context.Context, _ uuid.UUID, _ []uuid.UUID) ([]*FinanceBillableFee, error) {
	if err := requireFinanceBillTransaction(ctx); err != nil {
		return nil, err
	}
	s.transactionCalls++
	return []*FinanceBillableFee{s.fee}, nil
}

func (s *financeBillTransactionRepoStub) Create(ctx context.Context, bill *FinanceBill, _ *AuditEvent) (*FinanceBill, error) {
	if err := requireFinanceBillTransaction(ctx); err != nil {
		return nil, err
	}
	s.transactionCalls++
	bill.BillNo = "BILL-00001"
	s.created = bill
	return bill, nil
}

func (s *financeBillTransactionRepoStub) Get(ctx context.Context, _, _ uuid.UUID) (*FinanceBill, error) {
	if active, _ := ctx.Value(financeBillTransactionContextKey{}).(bool); active {
		return nil, errors.New("完整账单响应不能在写事务内读取")
	}
	s.responseReads++
	return s.created, nil
}

type financeBillExchangeRateTransactionStub struct {
	ExchangeRateRepo
	rateContext      *ExchangeRateContext
	resolved         *ResolvedExchangeRate
	transactionCalls int
}

func (s *financeBillExchangeRateTransactionStub) ResolveContext(ctx context.Context, _ uuid.UUID) (*ExchangeRateContext, error) {
	if err := requireFinanceBillTransaction(ctx); err != nil {
		return nil, err
	}
	s.transactionCalls++
	return s.rateContext, nil
}

func (s *financeBillExchangeRateTransactionStub) ListTimeStandards(ctx context.Context, _ uuid.UUID) ([]*ExchangeRateTimeStandardSetting, error) {
	if err := requireFinanceBillTransaction(ctx); err != nil {
		return nil, err
	}
	s.transactionCalls++
	return []*ExchangeRateTimeStandardSetting{{RateType: BillRateType, TimeStandards: []string{BillDateStandard}}}, nil
}

func (s *financeBillExchangeRateTransactionStub) Resolve(ctx context.Context, _ uuid.UUID, _ string, _ OrderFeeDirection, _, _, _ string) (*ResolvedExchangeRate, error) {
	if err := requireFinanceBillTransaction(ctx); err != nil {
		return nil, err
	}
	s.transactionCalls++
	return s.resolved, nil
}

func TestFinanceBillCreateUsesOneSharedTransaction(t *testing.T) {
	organizationID := uuid.New()
	actorID := uuid.New()
	feeID := uuid.New()
	settingID := uuid.New()
	repo := &financeBillTransactionRepoStub{fee: &FinanceBillableFee{
		Fee: &OrderFee{
			ID: feeID, OrderID: uuid.New(), Direction: OrderFeeReceivable, Status: OrderFeeConfirmed,
			FeeCode: "OCEAN_FREIGHT", FeeName: "海运费", SettlementPartyID: uuid.New(), SettlementPartyName: "测试客户",
			Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), TotalAmount: decimal.NewFromInt(100),
			NetAmount: decimal.NewFromInt(100), TaxAmount: decimal.Zero, Currency: "USD", BaseCurrency: "CNY",
			ExchangeRate: decimal.NewFromInt(7), BaseCurrencyAmount: decimal.NewFromInt(700),
		},
		OrderNo: "SE2026083000001", BusinessType: string(OrderBusinessSE),
	}}
	exchangeRepo := &financeBillExchangeRateTransactionStub{
		rateContext: &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"},
		resolved:    &ResolvedExchangeRate{Rate: decimal.RequireFromString("7.20"), Source: "SYSTEM", RateDate: "2026-08-30", SettingID: &settingID},
	}
	transactor := &financeBillTransactorStub{}
	usecase := NewFinanceBillUsecase(repo, NewExchangeRateUsecase(exchangeRepo), transactor)

	created, err := usecase.Create(context.Background(), organizationID, actorID, CreateFinanceBillInput{
		FeeIDs: []uuid.UUID{feeID}, BillDate: "2026-08-30", IdempotencyKey: "bill-transaction-test",
	})
	if err != nil {
		t.Fatalf("创建账单失败: %v", err)
	}
	if created.BillNo != "BILL-00001" || created.ExchangeRate.StringFixed(8) != "7.20000000" {
		t.Fatalf("账单创建结果不正确: %#v", created)
	}
	if transactor.calls != 1 {
		t.Fatalf("共享事务调用次数 = %d，期望 1", transactor.calls)
	}
	if repo.transactionCalls != 3 {
		t.Fatalf("账单仓储事务内调用次数 = %d，期望 3", repo.transactionCalls)
	}
	if repo.responseReads != 1 {
		t.Fatalf("提交后账单读取次数 = %d，期望 1", repo.responseReads)
	}
	if exchangeRepo.transactionCalls != 4 {
		t.Fatalf("汇率仓储事务内调用次数 = %d，期望 4", exchangeRepo.transactionCalls)
	}
}

var _ Transactor = (*financeBillTransactorStub)(nil)
