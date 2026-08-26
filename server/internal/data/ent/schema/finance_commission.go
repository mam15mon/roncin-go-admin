package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceCommission 保存基于有效应收核销计算出的已实现毛利及提成快照。
type FinanceCommission struct{ ent.Schema }

func (FinanceCommission) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceCommission) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.String("commission_no").NotEmpty().MaxLen(64).Immutable(),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.UUID("verification_id", uuid.Nil).Immutable(),
		field.String("verification_no").NotEmpty().MaxLen(64).Immutable(),
		field.UUID("employee_id", uuid.Nil).Immutable(),
		field.String("employee_name").NotEmpty().MaxLen(100).Immutable(),
		field.UUID("rule_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.String("rule_name").Optional().Nillable().MaxLen(100).Immutable(),
		field.String("personnel_role").Optional().Nillable().MaxLen(20).Immutable(),
		field.String("calculation_basis").Optional().Nillable().MaxLen(30).Immutable(),
		field.Enum("status").Values("DRAFT", "CONFIRMED", "PAID", "CANCELLED").Default("DRAFT"),
		field.String("base_currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("realized_revenue").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("allocated_cost").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("realized_profit").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("rate_percent").SchemaType(map[string]string{dialect.Postgres: "numeric(7,4)"}).Immutable(),
		field.String("commission_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
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

func (FinanceCommission) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("finance_commissions").Field("organization_id").Unique().Required().Immutable(),
		edge.From("verification", FinanceVerification.Type).Ref("commissions").Field("verification_id").Unique().Required().Immutable(),
		edge.From("employee", User.Type).Ref("finance_commissions").Field("employee_id").Unique().Required().Immutable(),
		edge.From("rule", FinanceCommissionRule.Type).Ref("commissions").Field("rule_id").Unique().Immutable(),
		edge.From("confirmed_by_user", User.Type).Ref("confirmed_finance_commissions").Field("confirmed_by").Unique(),
		edge.From("paid_by_user", User.Type).Ref("paid_finance_commissions").Field("paid_by").Unique(),
		edge.From("cancelled_by_user", User.Type).Ref("cancelled_finance_commissions").Field("cancelled_by").Unique(),
	}
}

func (FinanceCommission) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "commission_no").Unique(),
		index.Fields("organization_id", "idempotency_key").Unique(),
		index.Fields("verification_id", "employee_id", "status"),
		index.Fields("organization_id", "status", "created_at"),
	}
}
