package biz

import (
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
)

func TestFinanceControlFlowErrorsUseProtoReasons(t *testing.T) {
	tests := []struct {
		name string
		err  *errors.Error
		want string
	}{
		{name: "费用汇率缺失", err: ErrExchangeRateMissing, want: "FEE_EXCHANGE_RATE_MISSING"},
		{name: "账单费用无效", err: ErrFinanceBillFeeInvalid, want: "FINANCE_BILL_FEE_INVALID"},
		{name: "账单预览过期", err: ErrFinanceBillPreviewStale, want: "FINANCE_BILL_PREVIEW_STALE"},
		{name: "开票抬头缺失", err: ErrFinanceInvoiceProfileRequired, want: "FINANCE_INVOICE_PROFILE_REQUIRED"},
		{name: "提成调整超限", err: ErrCommissionAdjustmentExceeds, want: "FINANCE_COMMISSION_ADJUSTMENT_EXCEEDS"},
		{name: "提成调整状态冲突", err: ErrCommissionAdjustmentTransition, want: "FINANCE_COMMISSION_ADJUSTMENT_TRANSITION"},
		{name: "提成来源变化", err: ErrCommissionSourceChanged, want: "FINANCE_COMMISSION_SOURCE_CHANGED"},
		{name: "提成费用未确认", err: ErrCommissionUnconfirmedFees, want: "FINANCE_COMMISSION_UNCONFIRMED_FEES"},
		{name: "提成状态冲突", err: ErrCommissionTransition, want: "FINANCE_COMMISSION_TRANSITION"},
		{name: "提成规则冲突", err: ErrCommissionRuleConflict, want: "FINANCE_COMMISSION_RULE_CONFLICT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Reason != tt.want {
				t.Fatalf("错误原因 = %q，期望 %q", tt.err.Reason, tt.want)
			}
		})
	}
}
