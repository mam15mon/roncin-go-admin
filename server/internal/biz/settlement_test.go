package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type settlementRepoStub struct {
	filter FeeLedgerFilter
}

func (stub *settlementRepoStub) ListFeeLedger(_ context.Context, _ uuid.UUID, filter FeeLedgerFilter) (*FeeLedgerResult, error) {
	stub.filter = filter
	return &FeeLedgerResult{}, nil
}

func TestResolveFeeLedgerFinancialProgressCoversSevenStates(t *testing.T) {
	tests := []struct {
		name     string
		hasBill  bool
		invoiced bool
		bill     string
		verified string
		expected FeeLedgerFinancialProgress
	}{
		{name: "账单未建立", hasBill: false, bill: "100", verified: "0", expected: FeeLedgerUnbilled},
		{name: "未核销未开票", hasBill: true, bill: "100", verified: "0", expected: FeeLedgerUnverifiedUninvoiced},
		{name: "已开票未核销", hasBill: true, invoiced: true, bill: "100", verified: "0", expected: FeeLedgerInvoicedUnverified},
		{name: "已核销未开票", hasBill: true, bill: "100", verified: "100", expected: FeeLedgerVerifiedUninvoiced},
		{name: "已开票部分核销", hasBill: true, invoiced: true, bill: "100", verified: "40", expected: FeeLedgerInvoicedPartiallyVerified},
		{name: "部分核销未开票", hasBill: true, bill: "100", verified: "40", expected: FeeLedgerPartiallyVerifiedUninvoiced},
		{name: "已完成", hasBill: true, invoiced: true, bill: "100", verified: "100", expected: FeeLedgerCompleted},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := ResolveFeeLedgerFinancialProgress(test.hasBill, test.invoiced, decimal.RequireFromString(test.bill), decimal.RequireFromString(test.verified))
			if actual != test.expected {
				t.Fatalf("财务进度 = %s，期望 %s", actual, test.expected)
			}
		})
	}
}

func TestListFeeLedgerNormalizesFinancialProgressAndBillNumber(t *testing.T) {
	repo := &settlementRepoStub{}
	usecase := NewSettlementUsecase(repo)
	financeLocked := true
	_, err := usecase.ListFeeLedger(context.Background(), uuid.Must(uuid.NewV7()), FeeLedgerFilter{
		Page:              1,
		PageSize:          200,
		FinancialProgress: "completed",
		BillNo:            "  FB202608280001  ",
		FinanceLocked:     &financeLocked,
	})
	if err != nil {
		t.Fatalf("查询费用台账失败: %v", err)
	}
	if repo.filter.FinancialProgress != FeeLedgerCompleted {
		t.Fatalf("财务进度 = %q，期望 %q", repo.filter.FinancialProgress, FeeLedgerCompleted)
	}
	if repo.filter.BillNo != "FB202608280001" {
		t.Fatalf("账单编号 = %q，期望去除首尾空格", repo.filter.BillNo)
	}
	if repo.filter.FinanceLocked == nil || !*repo.filter.FinanceLocked {
		t.Fatal("费用锁定筛选未传递到数据层")
	}
}

func TestListFeeLedgerRejectsInvalidFinancialProgress(t *testing.T) {
	usecase := NewSettlementUsecase(&settlementRepoStub{})
	_, err := usecase.ListFeeLedger(context.Background(), uuid.Must(uuid.NewV7()), FeeLedgerFilter{
		Page:              1,
		PageSize:          20,
		FinancialProgress: "PAID",
	})
	if err != ErrFinanceLedgerInvalidArgument {
		t.Fatalf("错误 = %v，期望 ErrFinanceLedgerInvalidArgument", err)
	}
}
