package data

import (
	"strings"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent/migrate"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
)

// TestOrderFeeStatusCheckMetadata 断言费用状态 CHECK 与正式迁移同名同表达式：
// 历史 DRAFT/CONFIRMED 已合并为 UNBILLED，允许值收紧为未建账/已建账/已作废，
// 枚举默认值与迁移默认值同源（ADR 0003 CHECK 同源规范）。
func TestOrderFeeStatusCheckMetadata(t *testing.T) {
	const wantExpr = "status IN ('UNBILLED', 'BILLED', 'CANCELLED')"
	if migrate.OrderFeesTable.Annotation == nil || migrate.OrderFeesTable.Annotation.Checks == nil {
		t.Fatalf("order_fees 生成元数据缺少 CHECK 注解")
	}
	expr, ok := migrate.OrderFeesTable.Annotation.Checks["order_fees_status_check"]
	if !ok {
		t.Fatalf("生成元数据缺少 order_fees_status_check: %#v", migrate.OrderFeesTable.Annotation.Checks)
	}
	if normalized := strings.Join(strings.Fields(expr), " "); normalized != wantExpr {
		t.Fatalf("order_fees_status_check 表达式 = %q，期望 %q", normalized, wantExpr)
	}
	if orderfeeent.DefaultStatus != orderfeeent.StatusUNBILLED {
		t.Fatalf("order_fees.status 默认值 = %q，期望 UNBILLED", orderfeeent.DefaultStatus)
	}
}
