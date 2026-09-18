package data

import (
	"testing"

	entsqlschema "entgo.io/ent/dialect/sql/schema"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/migrate"
)

// TestFeeSupplementSchemaFKMetadata 断言锁后费用补录相关外键的删除策略与生成
// 元数据同源：审批生成的费用与关联冲减调整必须通过 NO ACTION 保留历史引用，
// 删除补录申请不得造成孤儿费用或孤儿调整；申请自身的发起人是永久审计事实，
// 决策人属于可逆审计引用保持 SET NULL。
func TestFeeSupplementSchemaFKMetadata(t *testing.T) {
	cases := []struct {
		name     string
		fks      []*entsqlschema.ForeignKey
		symbol   string
		onDelete entsqlschema.ReferenceOption
	}{
		{
			name:     "order_fees 补录申请外键必须 NO ACTION",
			fks:      migrate.OrderFeesTable.ForeignKeys,
			symbol:   "order_fees_order_fee_supplement_requests_fees",
			onDelete: entsqlschema.NoAction,
		},
		{
			name:     "finance_commission_adjustments 补录申请外键必须 NO ACTION",
			fks:      migrate.FinanceCommissionAdjustmentsTable.ForeignKeys,
			symbol:   "finance_commission_adjustments_order_fee_supplement_requests_commission_adjustments",
			onDelete: entsqlschema.NoAction,
		},
		{
			name:     "补录申请订单外键必须 NO ACTION",
			fks:      migrate.OrderFeeSupplementRequestsTable.ForeignKeys,
			symbol:   "order_fee_supplement_requests_orders_fee_supplement_requests",
			onDelete: entsqlschema.NoAction,
		},
		{
			name:     "补录申请组织外键必须 NO ACTION",
			fks:      migrate.OrderFeeSupplementRequestsTable.ForeignKeys,
			symbol:   "order_fee_supplement_requests_organizations_order_fee_supplement_requests",
			onDelete: entsqlschema.NoAction,
		},
		{
			name:     "补录申请发起人外键必须 NO ACTION",
			fks:      migrate.OrderFeeSupplementRequestsTable.ForeignKeys,
			symbol:   "order_fee_supplement_requests_users_requested_order_fee_supplement_requests",
			onDelete: entsqlschema.NoAction,
		},
		{
			name:     "补录申请决策人外键保持 SET NULL",
			fks:      migrate.OrderFeeSupplementRequestsTable.ForeignKeys,
			symbol:   "order_fee_supplement_requests_users_decided_order_fee_supplement_requests",
			onDelete: entsqlschema.SetNull,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, fk := range tc.fks {
				if fk.Symbol != tc.symbol {
					continue
				}
				if fk.OnDelete != tc.onDelete {
					t.Fatalf("外键 %s 删除策略 = %v，期望 %v", tc.symbol, fk.OnDelete, tc.onDelete)
				}
				return
			}
			t.Fatalf("生成元数据缺少外键 %s", tc.symbol)
		})
	}
}

// TestFeeSupplementSourceUniqueIndexMetadata 断言来源唯一键与一对一费用关联
// 的唯一索引存在于生成元数据，防止审批重试或并发审批重复入账、重复冲减。
func TestFeeSupplementSourceUniqueIndexMetadata(t *testing.T) {
	expected := map[string]struct {
		table  *entsqlschema.Table
		column []string
	}{
		"orderfee_supplement_request_id": {
			table:  migrate.OrderFeesTable,
			column: []string{"supplement_request_id"},
		},
		"financecommissionadjustment_commission_id_order_id_source_type_source_fee_supplement_request_id": {
			table:  migrate.FinanceCommissionAdjustmentsTable,
			column: []string{"commission_id", "order_id", "source_type", "source_fee_supplement_request_id"},
		},
		"orderfeesupplementrequest_organization_id_idempotency_key": {
			table:  migrate.OrderFeeSupplementRequestsTable,
			column: []string{"organization_id", "idempotency_key"},
		},
	}
	for name, want := range expected {
		t.Run(name, func(t *testing.T) {
			for _, idx := range want.table.Indexes {
				if idx.Name != name {
					continue
				}
				if !idx.Unique {
					t.Fatalf("索引 %s 必须唯一", name)
				}
				if len(idx.Columns) != len(want.column) {
					t.Fatalf("索引 %s 列数 = %d，期望 %d", name, len(idx.Columns), len(want.column))
				}
				for i, col := range want.column {
					if idx.Columns[i].Name != col {
						t.Fatalf("索引 %s 第 %d 列 = %s，期望 %s", name, i, idx.Columns[i].Name, col)
					}
				}
				return
			}
			t.Fatalf("生成元数据缺少唯一索引 %s", name)
		})
	}
}
