package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AdministrativeRegion 保存民政部省、市、区县三级行政区划。
type AdministrativeRegion struct{ ent.Schema }

func (AdministrativeRegion) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (AdministrativeRegion) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").NotEmpty().MinLen(12).MaxLen(12).Immutable(),
		field.String("name").NotEmpty().MaxLen(100),
		field.Int("level").Min(1).Max(3),
		field.String("parent_code").Optional().Nillable().MinLen(12).MaxLen(12),
		field.String("region_type").Optional().Nillable().MaxLen(50),
		field.String("source").NotEmpty().MaxLen(50).Default("MCA_DMFW"),
		field.String("source_version").Optional().Nillable().MaxLen(100),
		field.Bool("enabled").Default(true),
		searchKeywordsField(),
	}
}

func (AdministrativeRegion) Hooks() []ent.Hook {
	return []ent.Hook{searchKeywordsHook("name")}
}

func (AdministrativeRegion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code").Unique(),
		index.Fields("level", "code"),
		index.Fields("parent_code", "level", "code"),
		index.Fields("name"),
	}
}
