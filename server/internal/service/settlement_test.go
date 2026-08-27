package service

import (
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestFeeLedgerStatusFiltersSupportsLegacyFinancialProgress(t *testing.T) {
	status, progress, err := feeLedgerStatusFilters("completed", "")
	if err != nil {
		t.Fatalf("解析兼容财务进度失败: %v", err)
	}
	if status != "" || progress != biz.FeeLedgerCompleted {
		t.Fatalf("费用状态 = %q，财务进度 = %q", status, progress)
	}
}

func TestFeeLedgerStatusFiltersKeepsOrderFeeStatus(t *testing.T) {
	status, progress, err := feeLedgerStatusFilters("confirmed", "")
	if err != nil {
		t.Fatalf("解析费用状态失败: %v", err)
	}
	if status != biz.OrderFeeConfirmed || progress != "" {
		t.Fatalf("费用状态 = %q，财务进度 = %q", status, progress)
	}
}

func TestFeeLedgerStatusFiltersRejectsConflictingProgress(t *testing.T) {
	_, _, err := feeLedgerStatusFilters("COMPLETED", "UNBILLED")
	if err != biz.ErrFinanceLedgerInvalidArgument {
		t.Fatalf("错误 = %v，期望 ErrFinanceLedgerInvalidArgument", err)
	}
}
