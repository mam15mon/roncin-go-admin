package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// ShippingLineContainerPrefix 定义船司持有的 BIC 箱主前缀。
type ShippingLineContainerPrefix struct{ ent.Schema }

func (ShippingLineContainerPrefix) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (ShippingLineContainerPrefix) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.UUID("shipping_line_id", uuid.Nil),
		field.String("prefix").NotEmpty().MaxLen(4).Immutable(),
	}
}

func (ShippingLineContainerPrefix) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("shipping_line", ShippingLine.Type).Ref("container_prefixes").Field("shipping_line_id").Unique().Required(),
	}
}

func (ShippingLineContainerPrefix) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "prefix").Unique(),
		index.Fields("shipping_line_id", "prefix").Unique(),
	}
}
