package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceCommissionLine 保存提成计算时逐订单形成的不可变金额与人员归属快照。
type FinanceCommissionLine struct{ ent.Schema }

func (FinanceCommissionLine) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceCommissionLine) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("commission_id", uuid.Nil).Immutable(),
		field.UUID("order_id", uuid.Nil).Immutable(),
		field.String("order_no").NotEmpty().MaxLen(64).Immutable(),
		field.UUID("personnel_assignment_id", uuid.Nil).Immutable(),
		field.UUID("personnel_organization_id", uuid.Nil).Immutable(),
		field.Time("personnel_assigned_at").Immutable(),
		field.UUID("employee_id", uuid.Nil).Immutable(),
		field.String("employee_name").NotEmpty().MaxLen(100).Immutable(),
		field.String("personnel_role").NotEmpty().MaxLen(20).Immutable(),
		field.String("calculation_basis").NotEmpty().MaxLen(30).Immutable(),
		field.String("base_currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("realized_revenue").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("allocated_cost").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("realized_profit").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("rate_percent").SchemaType(map[string]string{dialect.Postgres: "numeric(7,4)"}).Immutable(),
		field.String("commission_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
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
