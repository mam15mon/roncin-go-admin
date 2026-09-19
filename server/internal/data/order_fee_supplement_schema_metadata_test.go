package data

import (
	"testing"

	entsqlschema "entgo.io/ent/dialect/sql/schema"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent/migrate"
)

// Ent 生成的包级表对象（migrate.XxxTable）在进程内会被 Schema.Create 类
// 测试就地改写（AGENTS.md 警示的全局表指针污染）：外键与索引条目在首次
// 真实建库后被消费，后续测试读到的不再是生成元数据的原始状态。
// 因此在包初始化（任何测试运行前）快照本域三张表的 pristine 元数据，
// 断言全部基于快照进行，保证与测试执行顺序无关。
type feeSupplementFKSnapshot struct {
	symbol   string
	onDelete entsqlschema.ReferenceOption
}

type feeSupplementIndexSnapshot struct {
	unique  bool
	columns []string
}

var feeSupplementFKSnapshots = map[string][]feeSupplementFKSnapshot{
	"order_fees":                     snapshotFKs(migrate.OrderFeesTable),
	"finance_commission_adjustments": snapshotFKs(migrate.FinanceCommissionAdjustmentsTable),
	"order_fee_supplement_requests":  snapshotFKs(migrate.OrderFeeSupplementRequestsTable),
}

var feeSupplementIndexSnapshots = map[string]feeSupplementIndexSnapshot{
	"orderfee_supplement_request_id": snapshotIndex(migrate.OrderFeesTable, "orderfee_supplement_request_id"),
	"financecommissionadjustment_commission_id_order_id_source_type_source_fee_supplement_request_id": snapshotIndex(migrate.FinanceCommissionAdjustmentsTable, "financecommissionadjustment_commission_id_order_id_source_type_source_fee_supplement_request_id"),
	"orderfeesupplementrequest_organization_id_idempotency_key":                                       snapshotIndex(migrate.OrderFeeSupplementRequestsTable, "orderfeesupplementrequest_organization_id_idempotency_key"),
}

func snapshotFKs(table *entsqlschema.Table) []feeSupplementFKSnapshot {
	if table == nil {
		return nil
	}
	out := make([]feeSupplementFKSnapshot, 0, len(table.ForeignKeys))
	for _, fk := range table.ForeignKeys {
		out = append(out, feeSupplementFKSnapshot{symbol: fk.Symbol, onDelete: fk.OnDelete})
	}
	return out
}

func snapshotIndex(table *entsqlschema.Table, name string) feeSupplementIndexSnapshot {
	if table == nil {
		return feeSupplementIndexSnapshot{}
	}
	for _, idx := range table.Indexes {
		if idx.Name != name {
			continue
		}
		columns := make([]string, 0, len(idx.Columns))
		for _, col := range idx.Columns {
			columns = append(columns, col.Name)
		}
		return feeSupplementIndexSnapshot{unique: idx.Unique, columns: columns}
	}
	return feeSupplementIndexSnapshot{}
}

// TestFeeSupplementSchemaFKMetadata 断言锁后费用补录相关外键的删除策略与生成
// 元数据同源：审批生成的费用与关联冲减调整必须通过 NO ACTION 保留历史引用，
// 删除补录申请不得造成孤儿费用或孤儿调整；申请自身的发起人是永久审计事实，
// 决策人属于可逆审计引用保持 SET NULL。
func TestFeeSupplementSchemaFKMetadata(t *testing.T) {
	cases := []struct {
		name     string
		table    string
		symbol   string
		onDelete entsqlschema.ReferenceOption
	}{
		{
			name:     "order_fees 补录申请外键必须 NO ACTION",
			table:    "order_fees",
			symbol:   "order_fees_order_fee_supplement_requests_fees",
			onDelete: entsqlschema.NoAction,
		},
		{
			name:     "finance_commission_adjustments 补录申请外键必须 NO ACTION",
			table:    "finance_commission_adjustments",
			symbol:   "finance_commission_adjustments_order_fee_supplement_requests_commission_adjustments",
			onDelete: entsqlschema.NoAction,
		},
		{
			name:     "补录申请订单外键必须 NO ACTION",
			table:    "order_fee_supplement_requests",
			symbol:   "order_fee_supplement_requests_orders_fee_supplement_requests",
			onDelete: entsqlschema.NoAction,
		},
		{
			name:     "补录申请组织外键必须 NO ACTION",
			table:    "order_fee_supplement_requests",
			symbol:   "order_fee_supplement_requests_organizations_order_fee_supplement_requests",
			onDelete: entsqlschema.NoAction,
		},
		{
			name:     "补录申请发起人外键必须 NO ACTION",
			table:    "order_fee_supplement_requests",
			symbol:   "order_fee_supplement_requests_users_requested_order_fee_supplement_requests",
			onDelete: entsqlschema.NoAction,
		},
		{
			name:     "补录申请决策人外键保持 SET NULL",
			table:    "order_fee_supplement_requests",
			symbol:   "order_fee_supplement_requests_users_decided_order_fee_supplement_requests",
			onDelete: entsqlschema.SetNull,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, fk := range feeSupplementFKSnapshots[tc.table] {
				if fk.symbol != tc.symbol {
					continue
				}
				if fk.onDelete != tc.onDelete {
					t.Fatalf("外键 %s 删除策略 = %v，期望 %v", tc.symbol, fk.onDelete, tc.onDelete)
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
	expected := []struct {
		name    string
		columns []string
	}{
		{
			name:    "orderfee_supplement_request_id",
			columns: []string{"supplement_request_id"},
		},
		{
			name:    "financecommissionadjustment_commission_id_order_id_source_type_source_fee_supplement_request_id",
			columns: []string{"commission_id", "order_id", "source_type", "source_fee_supplement_request_id"},
		},
		{
			name:    "orderfeesupplementrequest_organization_id_idempotency_key",
			columns: []string{"organization_id", "idempotency_key"},
		},
	}
	for _, want := range expected {
		t.Run(want.name, func(t *testing.T) {
			snap, ok := feeSupplementIndexSnapshots[want.name]
			if !ok {
				t.Fatalf("生成元数据缺少唯一索引 %s", want.name)
			}
			if !snap.unique {
				t.Fatalf("索引 %s 必须唯一", want.name)
			}
			if len(snap.columns) != len(want.columns) {
				t.Fatalf("索引 %s 列数 = %d，期望 %d", want.name, len(snap.columns), len(want.columns))
			}
			for i, col := range want.columns {
				if snap.columns[i] != col {
					t.Fatalf("索引 %s 第 %d 列 = %s，期望 %s", want.name, i, snap.columns[i], col)
				}
			}
		})
	}
}
