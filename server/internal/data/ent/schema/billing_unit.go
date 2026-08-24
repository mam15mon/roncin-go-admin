package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// BillingUnit 定义组织级费用计费单位。
type BillingUnit struct{ ent.Schema }

func (BillingUnit) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (BillingUnit) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.String("code").NotEmpty().MaxLen(32),
		field.String("name").NotEmpty().MaxLen(64),
		field.Int("sort_order").Default(100),
		field.Bool("enabled").Default(true),
	}
}

func (BillingUnit) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("billing_units").Field("organization_id").Unique().Required(),
		edge.To("fee_settings", FeeSetting.Type),
	}
}

func (BillingUnit) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "code").Unique(),
		index.Fields("organization_id", "enabled", "sort_order"),
	}
}
