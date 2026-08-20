package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// MilestoneTemplate 定义带版本与发布状态的里程碑模板。
type MilestoneTemplate struct{ ent.Schema }

func (MilestoneTemplate) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (MilestoneTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.String("code").NotEmpty().MaxLen(64).Immutable(),
		field.String("name").NotEmpty().MaxLen(100).Immutable(),
		field.Enum("business_type").Values("SE", "SI", "AE", "AI", "LAND", "RAIL").Immutable(),
		field.String("trade_term").Default("").MaxLen(16).Immutable(),
		field.Int("version").Min(1).Immutable(),
		field.Bool("is_default").Default(false),
		field.Time("published_at").Optional().Nillable(),
		field.Bool("enabled").Default(true),
	}
}

func (MilestoneTemplate) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("milestone_templates").Field("organization_id").Unique().Required(),
		edge.To("items", MilestoneTemplateItem.Type),
	}
}

func (MilestoneTemplate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "business_type", "trade_term", "code", "version").Unique(),
		index.Fields("organization_id", "business_type", "trade_term").Unique().Annotations(entsql.IndexWhere("is_default")),
		index.Fields("organization_id", "business_type", "trade_term", "published_at", "enabled"),
	}
}
