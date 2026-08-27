package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Airline 定义航空承运人的行业标准编码。
type Airline struct{ ent.Schema }

func (Airline) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (Airline) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.String("iata_code").NotEmpty().MaxLen(2).Immutable(),
		field.String("icao_code").MaxLen(3).Optional().Nillable(),
		field.String("awb_prefix").NotEmpty().MaxLen(3),
		field.String("name_zh").NotEmpty().MaxLen(200),
		field.String("name_en").NotEmpty().MaxLen(200),
		field.String("country_code").NotEmpty().MaxLen(2),
		field.Bool("cargo_only").Default(false),
		field.String("source").MaxLen(100).Default("manual"),
		field.Int("sort_order").Default(100),
		field.Bool("enabled").Default(true),
		searchKeywordsField(),
	}
}

func (Airline) Hooks() []ent.Hook { return []ent.Hook{searchKeywordsHook("name_zh", "name_en")} }

func (Airline) Edges() []ent.Edge {
	return []ent.Edge{edge.From("organization", Organization.Type).Ref("airlines").Field("organization_id").Unique().Required()}
}

func (Airline) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "iata_code").Unique(),
		index.Fields("organization_id", "icao_code").Unique(),
		index.Fields("organization_id", "awb_prefix").Unique(),
		index.Fields("organization_id", "enabled", "sort_order"),
	}
}
