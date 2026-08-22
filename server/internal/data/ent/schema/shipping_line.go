package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// ShippingLine 定义班轮公司的 SCAC 标准标识。
type ShippingLine struct{ ent.Schema }

func (ShippingLine) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (ShippingLine) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.String("scac_code").NotEmpty().MaxLen(4).Immutable(),
		field.String("name_zh").NotEmpty().MaxLen(200),
		field.String("name_en").NotEmpty().MaxLen(200),
		field.String("country_code").NotEmpty().MaxLen(2),
		field.String("tracking_url").MaxLen(500).Optional().Nillable(),
		field.String("alliance").MaxLen(100).Optional().Nillable(),
		field.String("source").MaxLen(100).Default("manual"),
		field.Int("sort_order").Default(100),
		field.Bool("enabled").Default(true),
	}
}

func (ShippingLine) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("shipping_lines").Field("organization_id").Unique().Required(),
		edge.To("container_prefixes", ShippingLineContainerPrefix.Type),
	}
}

func (ShippingLine) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "scac_code").Unique(),
		index.Fields("organization_id", "enabled", "sort_order"),
	}
}
