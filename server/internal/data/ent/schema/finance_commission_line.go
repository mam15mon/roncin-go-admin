package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceCommissionLine 保存提成计算时逐订单形成的不可变客户归属、费用和金额快照。
type FinanceCommissionLine struct{ ent.Schema }

func (FinanceCommissionLine) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceCommissionLine) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Checks(map[string]string{
			// 复算快照一致性：存量行在回填迁移执行前保持全空；READY 必须有完整
			// 历史总应收/总应付分母且不得携带不可用原因码，READY+MIGRATED 必须有
			// 回填算法版本与证据哈希（READY+NATIVE 不得伪填）；UNAVAILABLE 不得
			// 伪填估算分母且必须有稳定原因码。
			"finance_commission_lines_snapshot_consistency_check": "(snapshot_status IS NULL AND total_receivable_snapshot IS NULL AND total_payable_snapshot IS NULL AND snapshot_source IS NULL AND snapshot_backfill_version IS NULL AND snapshot_evidence_hash IS NULL AND snapshot_unavailable_reason_code IS NULL) OR (snapshot_status = 'READY' AND total_receivable_snapshot IS NOT NULL AND total_payable_snapshot IS NOT NULL AND snapshot_unavailable_reason_code IS NULL AND snapshot_source IS NOT NULL AND ((snapshot_source = 'NATIVE' AND snapshot_backfill_version IS NULL AND snapshot_evidence_hash IS NULL) OR (snapshot_source = 'MIGRATED' AND snapshot_backfill_version IS NOT NULL AND snapshot_evidence_hash IS NOT NULL))) OR (snapshot_status = 'UNAVAILABLE' AND total_receivable_snapshot IS NULL AND total_payable_snapshot IS NULL AND snapshot_source IS NULL AND snapshot_backfill_version IS NULL AND snapshot_evidence_hash IS NULL AND snapshot_unavailable_reason_code IS NOT NULL)",
		}),
	}
}

func (FinanceCommissionLine) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("commission_id", uuid.Nil).Immutable(),
		field.UUID("order_id", uuid.Nil).Immutable(),
		field.String("order_no").NotEmpty().MaxLen(64).Immutable(),
		field.String("order_date").NotEmpty().MaxLen(32).Immutable(),
		field.UUID("customer_id", uuid.Nil).Immutable(),
		field.String("customer_code").NotEmpty().MaxLen(64).Immutable(),
		field.String("customer_name").NotEmpty().MaxLen(200).Immutable(),
		field.UUID("personnel_assignment_id", uuid.Nil).Immutable(),
		field.UUID("personnel_organization_id", uuid.Nil).Immutable(),
		field.Time("personnel_assigned_at").Immutable(),
		field.Int("fee_count").NonNegative().Immutable(),
		field.String("fee_snapshot").Default("[]").SchemaType(map[string]string{dialect.Postgres: "jsonb"}).Immutable(),
		field.UUID("employee_id", uuid.Nil).Immutable(),
		field.String("employee_name").NotEmpty().MaxLen(100).Immutable(),
		field.String("personnel_role").NotEmpty().MaxLen(20).Immutable(),
		field.String("calculation_basis").NotEmpty().MaxLen(30).Immutable(),
		field.String("base_currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("realized_revenue").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("allocated_cost").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("realized_profit").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("commission_base_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("rate_percent").SchemaType(map[string]string{dialect.Postgres: "numeric(7,4)"}).Immutable(),
		field.String("commission_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		// 历史复算快照：原提成行固化时的历史总应收/总应付分母。新生成提成行固定
		// 写 READY + NATIVE；存量行由阶段回填迁移从原计算快照形成时点的不可变
		// 事实确定性还原，无法可信还原的成本敏感行写 UNAVAILABLE 及稳定原因码。
		field.String("total_receivable_snapshot").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("total_payable_snapshot").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.Enum("snapshot_status").Values("READY", "UNAVAILABLE").Optional().Nillable().Immutable(),
		field.Enum("snapshot_source").Values("NATIVE", "MIGRATED").Optional().Nillable().Immutable(),
		field.String("snapshot_backfill_version").Optional().Nillable().MaxLen(64).Immutable(),
		field.String("snapshot_evidence_hash").Optional().Nillable().MaxLen(64).Immutable(),
		field.String("snapshot_unavailable_reason_code").Optional().Nillable().MaxLen(64).Immutable(),
	}
}

func (FinanceCommissionLine) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("finance_commission_lines").Field("organization_id").Unique().Required().Immutable(),
		edge.From("commission", FinanceCommission.Type).Ref("lines").Field("commission_id").Unique().Required().Immutable(),
		edge.From("order", Order.Type).Ref("finance_commission_lines").Field("order_id").Unique().Required().Immutable(),
	}
}

func (FinanceCommissionLine) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("commission_id", "order_id").Unique(),
		index.Fields("organization_id", "employee_id"),
		index.Fields("order_id"),
	}
}
