package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// EnterpriseTagGroup 保存企业资源标签组。
type EnterpriseTagGroup struct{ ent.Schema }

func (EnterpriseTagGroup) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (EnterpriseTagGroup) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.String("name").NotEmpty().MaxLen(100),
		field.String("normalized_name").NotEmpty().MaxLen(100),
		searchKeywordsField(),
		field.String("color").Optional().Nillable().MaxLen(7),
		field.Int("sort_order").Default(0).NonNegative(),
	}
}

func (EnterpriseTagGroup) Hooks() []ent.Hook {
	return []ent.Hook{searchKeywordsHook("name")}
}

func (EnterpriseTagGroup) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("enterprise_tag_groups").Field("organization_id").Unique().Required().Immutable(),
		edge.To("tags", EnterpriseTag.Type),
	}
}

func (EnterpriseTagGroup) Indexes() []ent.Index {
	return []ent.Index{index.Fields("organization_id", "normalized_name").Unique(), index.Fields("organization_id", "sort_order")}
}

// EnterpriseTag 保存资源基表与标签组之间的标签详情。
type EnterpriseTag struct{ ent.Schema }

func (EnterpriseTag) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (EnterpriseTag) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("resource_id", uuid.Nil).Immutable(),
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("group_id", uuid.Nil),
		field.String("normalized_name").NotEmpty().MaxLen(200),
		field.Int("sort_order").Default(0).NonNegative(),
	}
}

func (EnterpriseTag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("resource", EnterpriseResource.Type).Ref("tag").Field("resource_id").Unique().Required().Immutable(),
		edge.From("group", EnterpriseTagGroup.Type).Ref("tags").Field("group_id").Unique().Required(),
	}
}

func (EnterpriseTag) Indexes() []ent.Index {
	return []ent.Index{index.Fields("resource_id").Unique(), index.Fields("group_id", "normalized_name").Unique(), index.Fields("organization_id", "group_id", "sort_order")}
}
