package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Airport 定义 IATA 与 ICAO 标准标识的机场。
type Airport struct{ ent.Schema }

func (Airport) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (Airport) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.String("iata_code").NotEmpty().MaxLen(3).Immutable(),
		field.String("icao_code").MaxLen(4).Optional().Nillable(),
		field.String("name_zh").NotEmpty().MaxLen(200),
		field.String("name_en").NotEmpty().MaxLen(200),
		field.String("city_name_zh").NotEmpty().MaxLen(100),
		field.String("city_name_en").MaxLen(100).Optional().Nillable(),
		field.String("country_code").NotEmpty().MaxLen(2),
		field.String("source").MaxLen(100).Default("manual"),
		field.Int("sort_order").Default(100),
		field.Bool("enabled").Default(true),
	}
}

func (Airport) Edges() []ent.Edge {
	return []ent.Edge{edge.From("organization", Organization.Type).Ref("airports").Field("organization_id").Unique().Required()}
}

func (Airport) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "iata_code").Unique(),
		index.Fields("organization_id", "icao_code").Unique(),
		index.Fields("organization_id", "enabled", "sort_order"),
	}
}
