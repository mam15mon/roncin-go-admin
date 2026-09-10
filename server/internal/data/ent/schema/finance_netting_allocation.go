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

// FinanceNettingAllocation 是对冲确认后在原始账单上形成的抵销分摊。
// 只有 active 分摊参与账单可用余额计算；反转后置为失效并保留审计。
type FinanceNettingAllocation struct{ ent.Schema }

func (FinanceNettingAllocation) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceNettingAllocation) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("netting_id", uuid.Nil).Immutable(),
		field.UUID("bill_id", uuid.Nil).Immutable(),
		field.String("bill_no").NotEmpty().MaxLen(64).Immutable(),
		field.Enum("direction").Values("RECEIVABLE", "PAYABLE").Immutable(),
		field.String("amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("base_currency_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.Bool("active").Default(false),
	}
}

func (FinanceNettingAllocation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("netting", FinanceNetting.Type).Ref("allocations").Field("netting_id").Unique().Required().Immutable(),
		edge.From("bill", FinanceBill.Type).Ref("netting_allocations").Field("bill_id").Unique().Required().Immutable(),
	}
}

func (FinanceNettingAllocation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("netting_id", "active"),
		index.Fields("bill_id", "active"),
		index.Fields("netting_id", "bill_id").Unique().StorageKey("netting_allocation_pair_unique"),
	}
}

func (FinanceNettingAllocation) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Checks(map[string]string{
		"financenettingallocation_direction_check": "direction IN ('RECEIVABLE', 'PAYABLE')",
		"financenettingallocation_amount_positive": "amount > 0",
	})}
}
