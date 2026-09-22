package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Port 定义使用 UN/LOCODE 标识的海运港口。
// 系统公共目录，所有公司使用相同的标准地点。
type Port struct{ ent.Schema }

func (Port) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (Port) Fields() []ent.Field {
	return []ent.Field{
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

func (Port) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("un_locode").Unique().StorageKey("ports_locode_unique"),
		index.Fields("enabled", "sort_order"),
	}
}
