package biz

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestVerificationCreateRejectsUnrepresentableAmounts(t *testing.T) {
	for _, entry := range []struct {
		name    string
		amounts []string
		valid   bool
	}{
		{"八位小数保留", []string{"0.12345678"}, true},
		{"零额", []string{"0"}, false},
		{"负数", []string{"-1"}, false},
		{"九位小数", []string{"0.123456789"}, false},
		{"巨大指数", []string{"1e1000000000"}, false},
		{"超出数据库范围", []string{"100000000000000000000"}, false},
		{"逐项合法但合计越界", []string{"60000000000000000000", "60000000000000000000"}, false},
	} {
		for _, within := range []bool{false, true} {
			t.Run(entry.name+map[bool]string{false: "独立入口", true: "组合入口"}[within], func(t *testing.T) {
				org, actor, cashID := uuid.New(), uuid.New(), uuid.New()
				repo := &verificationTransactionRepoStub{cashflow: &FinanceCashflow{ID: cashID, Direction: OrderFeeReceivable, Currency: "CNY", BaseCurrency: "CNY"}}
				rate := &verificationExchangeRateTransactionStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: org, BaseCurrency: "CNY"}}
				tx := &verificationTransactorStub{}
				uc := NewVerificationUsecase(repo, NewExchangeRateUsecase(rate, nil), tx, nil)
				input := CreateVerificationInput{VerificationDate: "2026-08-30", IdempotencyKey: "precision"}
				for _, amount := range entry.amounts {
					input.Allocations = append(input.Allocations, &VerificationAllocation{CashflowID: cashID, BillID: uuid.New(), Amount: decimal.RequireFromString(amount)})
				}
				var err error
				if within {
					_, err = uc.CreateWithinTransaction(context.Background(), org, actor, input)
				} else {
					_, err = uc.Create(context.Background(), org, actor, input)
				}
				if entry.valid {
					if err != nil || repo.created == nil || !repo.created.Amount.Equal(decimal.RequireFromString(entry.amounts[0])) {
						t.Fatalf("合法8位金额没有保留: %v", err)
					}
					return
				}
				if !errors.Is(err, ErrVerificationInvalid) || tx.calls != 0 || repo.created != nil {
					t.Fatalf("非法金额应在写入前拒绝: err=%v tx=%d", err, tx.calls)
				}
			})
		}
	}
}
