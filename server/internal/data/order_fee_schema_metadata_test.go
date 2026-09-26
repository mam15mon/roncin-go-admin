package data

import (
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent/migrate"
)

// TestOrderFeeStatusColumnsRemoved 断言费用状态已彻底退役：生成元数据中不再存在
// status/cancelled_* 列、order_fees_status_check 约束与 (order_id,status,created_at)
// 索引；是否已建账只由有效账单关联（活动账单行且所属账单未取消）这一关联事实表达，
// 防止状态列或状态 CHECK 以漂移形式回流（数据库规范「正式 CHECK 与 Ent 同源」的
// 删除侧守卫）。
func TestOrderFeeStatusColumnsRemoved(t *testing.T) {
	columns := map[string]struct{}{}
	for _, column := range migrate.OrderFeesColumns {
		columns[column.Name] = struct{}{}
	}
	for _, name := range []string{"status", "cancelled_at", "cancelled_by", "cancellation_reason"} {
		if _, exists := columns[name]; exists {
			t.Fatalf("order_fees.%s 应已随费用状态退役删除，仍存在于生成元数据", name)
		}
	}
	if migrate.OrderFeesTable.Annotation != nil && migrate.OrderFeesTable.Annotation.Checks != nil {
		if _, exists := migrate.OrderFeesTable.Annotation.Checks["order_fees_status_check"]; exists {
			t.Fatalf("order_fees_status_check 应已随费用状态退役删除，仍存在于生成元数据")
		}
	}
	for _, index := range migrate.OrderFeesTable.Indexes {
		if index.Name == "orderfee_order_id_status_created_at" {
			t.Fatalf("orderfee_order_id_status_created_at 索引应已随费用状态退役删除")
		}
	}
}
