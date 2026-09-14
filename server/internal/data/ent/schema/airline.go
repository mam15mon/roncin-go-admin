package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Airline 定义航空承运人的行业标准编码。
// A 型全局主数据：IATA/ICAO 等客观编码全球唯一，无组织归属。
type Airline struct{ ent.Schema }

func (Airline) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (Airline) Fields() []ent.Field {
	return []ent.Field{
		field.String("iata_code").NotEmpty().MaxLen(2).Immutable(),
		field.String("icao_code").MaxLen(3).Optional().Nillable(),
		field.String("awb_prefix").MaxLen(3).Optional().Nillable(),
		field.String("name_zh").MaxLen(200).Optional().Nillable(),
		field.String("name_en").NotEmpty().MaxLen(200),
		field.String("country_code").NotEmpty().MaxLen(2),
		field.Bool("cargo_only").Default(false),
		field.String("source").MaxLen(100).Default("manual"),
		field.String("source_version").MaxLen(100).Optional().Nillable(),
		field.String("source_hash").MaxLen(64).Optional().Nillable(),
		field.Int("sort_order").Default(100),
		field.Bool("enabled").Default(true),
		searchKeywordsField(),
	}
}

func (Airline) Hooks() []ent.Hook { return []ent.Hook{searchKeywordsHook("name_zh", "name_en")} }

func (Airline) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("iata_code").Unique(),
		index.Fields("icao_code").Unique(),
		index.Fields("awb_prefix").Unique(),
		index.Fields("enabled", "sort_order"),
	}
}
