package biz

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"testing"
)

func TestCommissionPaymentNetAmountAndEvidence(t *testing.T) {
	parent := &FinanceCommission{ID: uuid.New(), Version: 3, Status: CommissionConfirmed, BaseCurrency: "CNY", CommissionAmount: decimal.NewFromInt(200)}
	increase := &FinanceCommissionAdjustment{ID: uuid.New(), Version: 2, Status: CommissionConfirmed, Direction: CommissionAdjustmentIncrease, BaseCurrency: "CNY", Amount: decimal.NewFromInt(20)}
	decrease := &FinanceCommissionAdjustment{ID: uuid.New(), Version: 4, Status: CommissionConfirmed, Direction: CommissionAdjustmentDecrease, BaseCurrency: "CNY", Amount: decimal.NewFromInt(50)}
	preview, err := CalculateCommissionPayment(parent, []*FinanceCommissionAdjustment{increase, decrease})
	if err != nil {
		t.Fatal(err)
	}
	if !preview.ActualAmount.Equal(decimal.NewFromInt(170)) || len(preview.Adjustments) != 2 || preview.PreviewToken == "" {
		t.Fatalf("净额发放应170: %+v", preview)
	}
	old := preview.PreviewToken
	decrease.Version++
	changed, err := CalculateCommissionPayment(parent, []*FinanceCommissionAdjustment{increase, decrease})
	if err != nil {
		t.Fatal(err)
	}
	if changed.PreviewToken == old {
		t.Fatal("调整版本变化必须使预览失效")
	}
	decrease.Status = CommissionPaid
	if _, err := CalculateCommissionPayment(parent, []*FinanceCommissionAdjustment{decrease}); err != ErrCommissionPaymentEvidence {
		t.Fatalf("未发父单含已执行调整须阻断: %v", err)
	}
	parent.Status = CommissionPaid
	if _, err := CalculateCommissionPayment(parent, nil); err != ErrCommissionTransition {
		t.Fatal("已发父单不可再发")
	}
}

func TestCommissionPaymentZeroAndNegative(t *testing.T) {
	parent := &FinanceCommission{ID: uuid.New(), Version: 1, Status: CommissionConfirmed, BaseCurrency: "CNY", CommissionAmount: decimal.NewFromInt(10)}
	adjustment := &FinanceCommissionAdjustment{ID: uuid.New(), Version: 1, Status: CommissionConfirmed, Direction: CommissionAdjustmentDecrease, BaseCurrency: "CNY", Amount: decimal.NewFromInt(10)}
	result, err := CalculateCommissionPayment(parent, []*FinanceCommissionAdjustment{adjustment})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OffsetSettlement || !result.ActualAmount.IsZero() {
		t.Fatal("零额应抵扣结清")
	}
	adjustment.Amount = decimal.NewFromInt(11)
	if _, err := CalculateCommissionPayment(parent, []*FinanceCommissionAdjustment{adjustment}); err != ErrCommissionAdjustmentExceeds {
		t.Fatal("负额必须拒绝，不可静默归零")
	}
}

func TestCommissionReversalPendingIncrease(t *testing.T) {
	orderID := uuid.New()
	lines := []CommissionReversalLine{{OrderID: orderID, Amount: decimal.NewFromInt(200)}}
	a := &FinanceCommissionAdjustment{ID: uuid.New(), OrderID: orderID, Status: CommissionConfirmed, Direction: CommissionAdjustmentIncrease, Amount: decimal.NewFromInt(20)}
	if _, err := PlanCommissionReversal(CommissionPaid, decimal.NewFromInt(200), lines, []*FinanceCommissionAdjustment{a}); err != ErrCommissionPendingIncrease {
		t.Fatalf("未执行增提不能当现金追回: %v", err)
	}
	plan, err := PlanCommissionReversal(CommissionConfirmed, decimal.NewFromInt(200), lines, []*FinanceCommissionAdjustment{a})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.CancelCommission || len(plan.Recoveries) != 0 {
		t.Fatal("未付权益只能取消")
	}
}
