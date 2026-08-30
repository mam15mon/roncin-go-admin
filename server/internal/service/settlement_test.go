package service

import (
	"testing"

	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestFinanceStatusEnumConversions(t *testing.T) {
	progress := v1.FeeLedgerFinancialProgress_FEE_LEDGER_FINANCIAL_PROGRESS_INVOICED_PARTIALLY_VERIFIED
	if got := feeLedgerFinancialProgressFromAPI(&progress); got != biz.FeeLedgerInvoicedPartiallyVerified {
		t.Fatalf("财务进度 = %q", got)
	}
	if got := feeLedgerFinancialProgressToAPI(biz.FeeLedgerCompleted); got != v1.FeeLedgerFinancialProgress_FEE_LEDGER_FINANCIAL_PROGRESS_COMPLETED {
		t.Fatalf("财务进度 API 值 = %v", got)
	}
	if got := feeLedgerFinancialProgressFromAPI(nil); got != "" {
		t.Fatalf("空财务进度 = %q", got)
	}

	bill := v1.FinanceBillStatus_FINANCE_BILL_STATUS_CONFIRMED
	if got := financeBillStatusFromAPI(&bill); got != biz.FinanceBillConfirmed {
		t.Fatalf("账单状态 = %q", got)
	}
	if got := financeBillStatusToAPI(biz.FinanceBillCancelled); got != v1.FinanceBillStatus_FINANCE_BILL_STATUS_CANCELLED {
		t.Fatalf("账单 API 状态 = %v", got)
	}

	invoice := v1.FinanceInvoiceStatus_FINANCE_INVOICE_STATUS_RED_FLUSHED
	if got := financeInvoiceStatusFromAPI(&invoice); got != biz.FinanceInvoiceRedFlushed {
		t.Fatalf("发票状态 = %q", got)
	}
	if got := financeInvoiceStatusToAPI(biz.FinanceInvoiceIssued); got != v1.FinanceInvoiceStatus_FINANCE_INVOICE_STATUS_ISSUED {
		t.Fatalf("发票 API 状态 = %v", got)
	}

	cashflow := v1.FinanceCashflowStatus_FINANCE_CASHFLOW_STATUS_CONFIRMED
	if got := financeCashflowStatusFromAPI(&cashflow); got != biz.FinanceCashflowConfirmed {
		t.Fatalf("资金流水状态 = %q", got)
	}
	if got := financeCashflowStatusToAPI(biz.FinanceCashflowDraft); got != v1.FinanceCashflowStatus_FINANCE_CASHFLOW_STATUS_DRAFT {
		t.Fatalf("资金流水 API 状态 = %v", got)
	}

	verification := v1.FinanceVerificationStatus_FINANCE_VERIFICATION_STATUS_REVERSED
	if got := financeVerificationStatusFromAPI(&verification); got != biz.VerificationReversed {
		t.Fatalf("核销状态 = %q", got)
	}
	if got := financeVerificationStatusToAPI(biz.VerificationActive); got != v1.FinanceVerificationStatus_FINANCE_VERIFICATION_STATUS_ACTIVE {
		t.Fatalf("核销 API 状态 = %v", got)
	}

	commission := v1.FinanceCommissionStatus_FINANCE_COMMISSION_STATUS_PAID
	if got := financeCommissionStatusFromAPI(&commission); got != biz.CommissionPaid {
		t.Fatalf("提成状态 = %q", got)
	}
	if got := financeCommissionStatusToAPI(biz.CommissionConfirmed); got != v1.FinanceCommissionStatus_FINANCE_COMMISSION_STATUS_CONFIRMED {
		t.Fatalf("提成 API 状态 = %v", got)
	}
}

func TestFinanceStatusEnumConversionsKeepEmptyFilter(t *testing.T) {
	if got := financeBillStatusFromAPI(nil); got != "" {
		t.Fatalf("空账单筛选 = %q", got)
	}
	if got := financeInvoiceStatusFromAPI(nil); got != "" {
		t.Fatalf("空发票筛选 = %q", got)
	}
	if got := financeCashflowStatusFromAPI(nil); got != "" {
		t.Fatalf("空流水筛选 = %q", got)
	}
	if got := financeVerificationStatusFromAPI(nil); got != "" {
		t.Fatalf("空核销筛选 = %q", got)
	}
	if got := financeCommissionStatusFromAPI(nil); got != "" {
		t.Fatalf("空提成筛选 = %q", got)
	}
}
