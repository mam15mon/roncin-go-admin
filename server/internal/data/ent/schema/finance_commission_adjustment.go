package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceCommissionAdjustment 保存已确认提成之后形成的独立增减调整，不改写原始提成快照。
type FinanceCommissionAdjustment struct{ ent.Schema }

func (FinanceCommissionAdjustment) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceCommissionAdjustment) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("commission_id", uuid.Nil).Immutable(),
		field.UUID("order_id", uuid.Nil).Immutable(),
		field.String("adjustment_no").NotEmpty().MaxLen(80).Immutable(),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.String("commission_no").NotEmpty().MaxLen(64).Immutable(),
		field.String("order_no").NotEmpty().MaxLen(64).Immutable(),
		field.UUID("employee_id", uuid.Nil).Immutable(),
		field.String("employee_name").NotEmpty().MaxLen(100).Immutable(),
		field.Enum("direction").Values("INCREASE", "DECREASE").Immutable(),
		field.Enum("status").Values("DRAFT", "CONFIRMED", "PAID", "CANCELLED").Default("DRAFT"),
		field.String("base_currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("reason").NotEmpty().MaxLen(500).Immutable(),
		field.String("note").Optional().Nillable().MaxLen(500),
		field.Uint64("version").Default(1),
		field.Time("confirmed_at").Optional().Nillable(),
		field.UUID("confirmed_by", uuid.Nil).Optional().Nillable(),
		field.Time("paid_at").Optional().Nillable(),
		field.UUID("paid_by", uuid.Nil).Optional().Nillable(),
		field.Time("cancelled_at").Optional().Nillable(),
		field.UUID("cancelled_by", uuid.Nil).Optional().Nillable(),
		field.String("cancellation_reason").Optional().Nillable().MaxLen(500),
	}
}

func (FinanceCommissionAdjustment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("finance_commission_adjustments").Field("organization_id").Unique().Required().Immutable(),
		edge.From("commission", FinanceCommission.Type).Ref("adjustments").Field("commission_id").Unique().Required().Immutable(),
		edge.From("order", Order.Type).Ref("finance_commission_adjustments").Field("order_id").Unique().Required().Immutable(),
		edge.From("employee", User.Type).Ref("finance_commission_adjustments").Field("employee_id").Unique().Required().Immutable(),
		edge.From("confirmed_by_user", User.Type).Ref("confirmed_finance_commission_adjustments").Field("confirmed_by").Unique(),
		edge.From("paid_by_user", User.Type).Ref("paid_finance_commission_adjustments").Field("paid_by").Unique(),
		edge.From("cancelled_by_user", User.Type).Ref("cancelled_finance_commission_adjustments").Field("cancelled_by").Unique(),
	}
}

func (FinanceCommissionAdjustment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "adjustment_no").Unique(),
		index.Fields("organization_id", "idempotency_key").Unique(),
		index.Fields("commission_id", "status", "created_at"),
		index.Fields("order_id", "status"),
	}
}
