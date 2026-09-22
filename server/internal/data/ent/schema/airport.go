package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Airport 定义 IATA 与 ICAO 标准标识的机场。
// 系统公共目录，所有公司使用相同的标准地点。
type Airport struct{ ent.Schema }

func (Airport) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (Airport) Fields() []ent.Field {
	return []ent.Field{
		field.String("iata_code").NotEmpty().MaxLen(3).Immutable(),
		field.String("icao_code").MaxLen(4).Optional().Nillable(),
		field.String("name_zh").MaxLen(200).Optional().Nillable(),
		field.String("name_en").NotEmpty().MaxLen(200),
		field.String("city_name_zh").MaxLen(100).Optional().Nillable(),
		field.String("city_name_en").MaxLen(100).Optional().Nillable(),
		field.String("country_code").NotEmpty().MaxLen(2),
		field.String("source").MaxLen(100).Default("manual"),
		field.String("source_version").MaxLen(100).Optional().Nillable(),
		field.String("source_hash").MaxLen(64).Optional().Nillable(),
		field.Int("sort_order").Default(100),
		field.Bool("enabled").Default(true),
		searchKeywordsField(),
	}
}

func (Airport) Hooks() []ent.Hook {
	return []ent.Hook{searchKeywordsHook("name_zh", "name_en", "city_name_zh", "city_name_en")}
}

func (Airport) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("iata_code").Unique().StorageKey("airports_iata_unique"),
		index.Fields("icao_code").Unique().StorageKey("airports_icao_unique").Annotations(entsql.IndexWhere("icao_code IS NOT NULL")),
		index.Fields("enabled", "sort_order"),
	}
}
