package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// MasterDataItem is a fixed-kind option catalog used by order forms.
// Business rules and dynamic page definitions do not belong in this table.
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
		field.UUID("organization_id", uuid.Nil),
		field.Enum("kind").Values("currency", "country", "region", "container_spec", "service_type", "cargo_category", "abnormal_case"),
		field.String("code").NotEmpty().MaxLen(64).Immutable(),
		field.String("name").NotEmpty().MaxLen(200),
		field.String("name_en").MaxLen(200).Optional().Nillable(),
		field.String("parent_code").MaxLen(64).Optional().Nillable(),
		field.String("teu_factor").MaxLen(32).Optional().Nillable(),
		field.String("source").MaxLen(100).Default("manual"),
		field.Int("sort_order").Default(100),
		field.Bool("enabled").Default(true),
		field.JSON("attributes", &MasterDataAttributes{}).Default(&MasterDataAttributes{}),
	}
}

func (MasterDataItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("master_data_items").Field("organization_id").Unique().Required(),
	}
}

func (MasterDataItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "kind", "code").Unique(),
		index.Fields("organization_id", "kind", "enabled", "sort_order"),
		index.Fields("organization_id", "kind", "name"),
	}
}
