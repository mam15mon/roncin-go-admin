package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// StatusTemplateItem 定义模板版本中的一个订单状态。
type StatusTemplateItem struct{ ent.Schema }

func (StatusTemplateItem) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (StatusTemplateItem) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("template_id", uuid.Nil),
		field.String("code").NotEmpty().MaxLen(64).Immutable(),
		field.String("label").NotEmpty().MaxLen(100),
		field.Int("sort_order").Default(100),
		field.Bool("enabled").Default(true),
		field.String("color_token").MaxLen(64).Optional().Nillable(),
		field.Bool("system").Default(false),
	}
}

func (StatusTemplateItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("template", StatusTemplate.Type).Ref("items").Field("template_id").Unique().Required(),
	}
}

func (StatusTemplateItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("template_id", "code").Unique(),
		index.Fields("template_id", "sort_order"),
	}
}
