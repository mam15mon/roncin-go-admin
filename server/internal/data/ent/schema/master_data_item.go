package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// MasterDataItem is a fixed-kind option catalog used by order forms.
// Business rules and dynamic page definitions do not belong in this table.
// A 型全局主数据：客观事实只有一份真相，无组织归属，业务码全局唯一。
type MasterDataItem struct{ ent.Schema }

// MasterDataAttributes 是主数据专属属性的 JSONB 持久化结构。
type MasterDataAttributes struct {
	Continent    *string `json:"continent,omitempty"`
	CurrencyCode *string `json:"currency_code,omitempty"`
	RegionLevel  *int    `json:"region_level,omitempty"`
}

func (MasterDataItem) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (MasterDataItem) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("kind").Values("currency", "country", "region", "container_spec", "charge_category", "cargo_category", "abnormal_case"),
		field.String("code").NotEmpty().MaxLen(64).Immutable(),
		field.String("name").NotEmpty().MaxLen(200),
		field.String("name_en").MaxLen(200).Optional().Nillable(),
		field.String("parent_code").MaxLen(64).Optional().Nillable(),
		field.String("teu_factor").MaxLen(32).Optional().Nillable(),
		field.String("source").MaxLen(100).Default("manual"),
		field.Int("sort_order").Default(100),
		field.Bool("enabled").Default(true),
		field.JSON("attributes", &MasterDataAttributes{}).Default(&MasterDataAttributes{}),
		searchKeywordsField(),
	}
}

func (MasterDataItem) Hooks() []ent.Hook {
	return []ent.Hook{searchKeywordsHook("name", "name_en")}
}

func (MasterDataItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("charge_category_fee_settings", FeeSetting.Type),
		edge.To("abnormal_case_fee_settings", FeeSetting.Type),
	}
}

func (MasterDataItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("kind", "code").Unique(),
		index.Fields("kind", "enabled", "sort_order"),
		index.Fields("kind", "name"),
	}
}
