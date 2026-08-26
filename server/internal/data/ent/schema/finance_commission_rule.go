package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceCommissionRule 定义按订单人员角色和计提口径生效的提成考核规则。
type FinanceCommissionRule struct{ ent.Schema }

func (FinanceCommissionRule) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceCommissionRule) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.String("name").NotEmpty().MaxLen(100),
		field.Enum("personnel_role").Values("SALES", "OPERATOR"),
		field.Enum("calculation_basis").Values("REALIZED_PROFIT", "REALIZED_REVENUE"),
		field.String("rate_percent").SchemaType(map[string]string{dialect.Postgres: "numeric(7,4)"}),
		field.String("effective_from").Optional().Nillable().MinLen(10).MaxLen(10),
		field.String("effective_to").Optional().Nillable().MinLen(10).MaxLen(10),
		field.Bool("enabled").Default(true),
		field.String("note").Optional().Nillable().MaxLen(500),
		field.Uint64("version").Default(1),
	}
}

func (FinanceCommissionRule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("finance_commission_rules").Field("organization_id").Unique().Required().Immutable(),
		edge.To("commissions", FinanceCommission.Type),
	}
}

func (FinanceCommissionRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "name").Unique(),
		index.Fields("organization_id", "enabled", "personnel_role"),
	}
}
