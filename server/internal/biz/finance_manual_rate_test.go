package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestCashflowManualRatePreservesCurrencyIdentity(t *testing.T) {
	for _, base := range []string{"CNY", "CAD"} {
		for _, rate := range []string{"7", "1"} {
			t.Run(base+"/"+rate, func(t *testing.T) {
				org := uuid.New()
				repo := &financeCashflowRepoStub{}
				rates := NewExchangeRateUsecase(&exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: org, BaseCurrency: base}}, nil)
				manual := decimal.RequireFromString(rate)
				item, err := NewFinanceCashflowUsecase(repo, rates).Create(context.Background(), org, uuid.New(), CreateFinanceCashflowInput{
					Direction: OrderFeeReceivable, SettlementPartyID: uuid.New(), Currency: base,
					Amount: decimal.NewFromInt(100), ExchangeRateOverride: &manual,
					// 客户端本币不是同币种判定依据。
					BaseCurrency: "USD", TransactionDate: "2026-11-04", OurAccount: "基本户", PaymentMethod: "银行转账", IdempotencyKey: "identity",
				}, true)
				if rate != "1" {
					if err != ErrExchangeRateSameCurrencyInvalid || item != nil || repo.created != nil {
						t.Fatalf("非 1 同币种汇率应拒绝且零写入：item=%+v err=%v", item, err)
					}
					return
				}
				if err != nil || item == nil || !item.BaseAmount.Equal(decimal.NewFromInt(100)) || item.BaseCurrency != base || item.ExchangeRateSource != ExchangeRateSourceManual {
					t.Fatalf("同币种手工 1 应保留 100 本币：item=%+v err=%v", item, err)
				}
			})
		}
	}
}

type identityFeeRepoStub struct {
	orderFeeUpdateCaptureRepoStub
	fees []*OrderFee
}

func (s *identityFeeRepoStub) List(context.Context, uuid.UUID, uuid.UUID) ([]*OrderFee, error) {
	return s.fees, nil
}

func TestOrderFeeManualRatePreservesCurrencyIdentity(t *testing.T) {
	for _, operation := range []string{"新增", "修改"} {
		for _, base := range []string{"CNY", "CAD"} {
			for _, tc := range []struct{ currency, rate string }{{base, "7"}, {base, "1"}, {"USD", "1.5"}} {
				t.Run(operation+"/"+base+"/"+tc.currency+"/"+tc.rate, func(t *testing.T) {
					org := uuid.New()
					fee := validOrderFeeForTest()
					fee.ID, fee.Version, fee.Currency = uuid.New(), 1, tc.currency
					counterpart := &OrderFee{ID: uuid.New(), Direction: OrderFeePayable, Currency: tc.currency}
					repo := &identityFeeRepoStub{fees: []*OrderFee{counterpart}}
					repo.current = fee
					manual := decimal.RequireFromString(tc.rate)
					input := *fee
					input.ExchangeRateOverride = &manual
					rates := NewExchangeRateUsecase(&exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: org, BaseCurrency: base}}, nil)
					uc := NewOrderFeeUsecase(repo, rates, nil, newReminderModeCreditControl(), nil)
					var item *OrderFee
					var err error
					if operation == "新增" {
						item, err = uc.Add(context.Background(), org, uuid.New(), uuid.New(), &input, true)
					} else {
						item, err = uc.Update(context.Background(), org, uuid.New(), uuid.New(), fee.ID, &input, true)
					}
					if tc.currency == base && tc.rate != "1" {
						if err != ErrExchangeRateSameCurrencyInvalid || item != nil || repo.added != nil || repo.updated != nil {
							t.Fatalf("同币种错误汇率必须零写入：item=%+v err=%v", item, err)
						}
						return
					}
					if err != nil || item == nil || !item.BaseCurrencyAmount.Equal(decimal.NewFromInt(100).Mul(manual)) || item.BaseCurrency != base || item.ExchangeRateSource != ExchangeRateSourceManual {
						t.Fatalf("合法手工汇率金额不正确：item=%+v err=%v", item, err)
					}
				})
			}
		}
	}
}

type identityCorrectionRepoStub struct {
	OrderFeeCorrectionRepo
	fees    []*OrderFee
	created *OrderFeeCorrectionRequest
}

func (*identityCorrectionRepoStub) Authorize(context.Context, *Principal, uuid.UUID, uuid.UUID, string) error {
	return nil
}
func (*identityCorrectionRepoStub) LockOrder(context.Context, uuid.UUID, uuid.UUID) (*OrderFeeSupplementLockEvidence, error) {
	lockID := uuid.New()
	return &OrderFeeSupplementLockEvidence{LockRecordID: &lockID}, nil
}
func (*identityCorrectionRepoStub) GetByKey(context.Context, uuid.UUID, string) (*OrderFeeCorrectionRequest, error) {
	return nil, nil
}
func (s *identityCorrectionRepoStub) CurrentFees(context.Context, uuid.UUID) ([]*OrderFee, error) {
	return s.fees, nil
}
func (s *identityCorrectionRepoStub) Create(_ context.Context, r *OrderFeeCorrectionRequest, _ *AuditEvent) (*OrderFeeCorrectionRequest, error) {
	s.created = r
	return r, nil
}
func (s *identityCorrectionRepoStub) Get(context.Context, uuid.UUID, uuid.UUID) (*OrderFeeCorrectionRequest, error) {
	return s.created, nil
}

func TestFeeCorrectionRejectsInvalidIdentityRateBeforeCreatingRequest(t *testing.T) {
	for _, operation := range []string{"ADD", "REPLACE"} {
		t.Run(operation, func(t *testing.T) {
			org, order := uuid.New(), uuid.New()
			source := validOrderFeeForTest()
			source.ID, source.Version = uuid.New(), 1
			repo := &identityCorrectionRepoStub{fees: []*OrderFee{source, {ID: uuid.New(), Direction: OrderFeePayable, Currency: "CNY"}}}
			feeUC := NewOrderFeeUsecase(&orderFeeAddCaptureRepoStub{}, NewExchangeRateUsecase(&exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: org, BaseCurrency: "CNY"}}, nil), nil, newReminderModeCreditControl(), nil)
			uc := NewOrderFeeCorrectionUsecase(repo, feeUC, nil, inlineTransactor{})
			manual := decimal.NewFromInt(7)
			after := validOrderFeeForTest()
			after.ExchangeRateOverride = &manual
			in := &OrderFeeCorrectionInput{Operation: operation, After: after, Reason: "费用更正", IdempotencyKey: "invalid-identity"}
			if operation == "REPLACE" {
				in.SourceFeeID, in.ExpectedFeeVersion = &source.ID, source.Version
			}
			item, err := uc.Submit(context.Background(), &Principal{UserID: uuid.New()}, org, order, in, true)
			if err != ErrExchangeRateSameCurrencyInvalid || item != nil || repo.created != nil {
				t.Fatalf("补审提交必须拒绝同币种非 1 且不创建申请：item=%+v err=%v", item, err)
			}
		})
	}
}
