package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Port 定义使用 UN/LOCODE 标识的海运港口。
type Port struct{ ent.Schema }

func (Port) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (Port) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.String("un_locode").NotEmpty().MaxLen(5).Immutable(),
		field.String("name_zh").MaxLen(200).Optional().Nillable(),
		field.String("name_en").NotEmpty().MaxLen(200),
		field.String("country_code").NotEmpty().MaxLen(2),
		field.JSON("transport_modes", []string{}),
		field.String("source").MaxLen(100).Default("manual"),
		field.String("source_version").MaxLen(100).Optional().Nillable(),
		field.String("source_hash").MaxLen(64).Optional().Nillable(),
		field.Int("sort_order").Default(100),
		field.Bool("enabled").Default(true),
		searchKeywordsField(),
	}
}

func (Port) Hooks() []ent.Hook { return []ent.Hook{searchKeywordsHook("name_zh", "name_en")} }

func (Port) Edges() []ent.Edge {
	return []ent.Edge{edge.From("organization", Organization.Type).Ref("ports").Field("organization_id").Unique().Required()}
}

func (Port) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "un_locode").Unique(),
		index.Fields("organization_id", "enabled", "sort_order"),
	}
}
