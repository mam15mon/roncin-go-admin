package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// StatusTemplate 定义带版本与发布状态的订单状态模板。
type StatusTemplate struct{ ent.Schema }

func (StatusTemplate) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (StatusTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.String("code").NotEmpty().MaxLen(64).Immutable(),
		field.String("name").NotEmpty().MaxLen(100),
		field.Enum("business_type").Values("SE", "SI", "AE", "AI", "LAND", "RAIL"),
		field.Int("version").Min(1).Immutable(),
		field.Bool("is_default").Default(false),
		field.Time("published_at").Optional().Nillable(),
		field.Bool("enabled").Default(true),
	}
}

func (StatusTemplate) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("status_templates").Field("organization_id").Unique().Required(),
		edge.To("items", StatusTemplateItem.Type),
		edge.To("orders", Order.Type),
	}
}

func (StatusTemplate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "business_type", "code", "version").Unique(),
		index.Fields("organization_id", "business_type").Unique().Annotations(entsql.IndexWhere("is_default")),
		index.Fields("organization_id", "business_type", "published_at", "enabled"),
	}
}
