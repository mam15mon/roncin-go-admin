package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceBillBatch 保存一次原子批量建单的拆单策略、请求摘要和汇总快照。
type FinanceBillBatch struct{ ent.Schema }

func (FinanceBillBatch) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceBillBatch) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.String("batch_no").NotEmpty().MaxLen(64).Immutable(),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.String("request_hash").NotEmpty().MinLen(64).MaxLen(64).Immutable(),
		field.Bool("split_by_order").Default(false).Immutable(),
		field.Bool("split_by_tax_rate").Default(false).Immutable(),
		field.Int("fee_count").Positive().Immutable(),
		field.Int("bill_count").Positive().Immutable(),
		field.String("total_base_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("base_currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.UUID("created_by", uuid.Nil).Immutable(),
	}
}

func (FinanceBillBatch) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("finance_bill_batches").Field("organization_id").Unique().Required().Immutable(),
		edge.From("creator", User.Type).Ref("created_finance_bill_batches").Field("created_by").Unique().Required().Immutable(),
		edge.To("bills", FinanceBill.Type),
	}
}

func (FinanceBillBatch) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "batch_no").Unique(),
		index.Fields("organization_id", "idempotency_key").Unique(),
		index.Fields("organization_id", "created_at"),
	}
}
