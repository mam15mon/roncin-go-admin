package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// MilestoneTemplateItem 定义里程碑模板版本中的具体节点配置。
type MilestoneTemplateItem struct{ ent.Schema }

func (MilestoneTemplateItem) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (MilestoneTemplateItem) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("template_id", uuid.Nil),
		field.String("code").NotEmpty().MaxLen(64).Immutable(),
		field.String("label").NotEmpty().MaxLen(100).Immutable(),
		field.String("description").MaxLen(500).Optional().Nillable().Immutable(),
		field.String("category").MaxLen(64).Optional().Nillable().Immutable(),
		field.Int("sort_order").Default(100).Immutable(),
		field.Bool("enabled").Default(true).Immutable(),
		field.JSON("depends_on", []string{}).Optional().Immutable(),
	}
}

func (MilestoneTemplateItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("template", MilestoneTemplate.Type).Ref("items").Field("template_id").Unique().Required(),
	}
}

func (MilestoneTemplateItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("template_id", "code").Unique(),
		index.Fields("template_id", "sort_order"),
	}
}
