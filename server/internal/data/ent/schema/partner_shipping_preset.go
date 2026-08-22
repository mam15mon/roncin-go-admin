package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// PartnerShippingPreset 保存企业常用货运单证资料。
type PartnerShippingPreset struct{ ent.Schema }

func (PartnerShippingPreset) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (PartnerShippingPreset) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("partner_id", uuid.Nil),
		field.Enum("preset_type").Values("SHIPPER", "CONSIGNEE", "NOTIFY_PARTY", "ENGLISH_CARGO_NAME", "HS_CODE", "MARKS"),
		field.String("title").NotEmpty().MaxLen(100),
		field.String("company_name").Optional().Nillable().MaxLen(200),
		field.String("address").Optional().Nillable().MaxLen(500),
		field.String("contact_name").Optional().Nillable().MaxLen(100),
		field.String("phone").Optional().Nillable().MaxLen(64),
		field.String("email").Optional().Nillable().MaxLen(254),
		field.String("country_code").Optional().Nillable().MinLen(2).MaxLen(2),
		field.String("tax_identifier").Optional().Nillable().MaxLen(64),
		field.String("content").Optional().Nillable().MaxLen(4000),
		field.String("code").Optional().Nillable().MaxLen(64),
		field.Bool("is_default").Default(false),
		field.Int("sort_order").Default(0).NonNegative(),
		field.String("remark").Optional().MaxLen(500),
		field.Bool("enabled").Default(true),
	}
}

func (PartnerShippingPreset) Edges() []ent.Edge {
	return []ent.Edge{edge.From("partner", Partner.Type).Ref("shipping_presets").Field("partner_id").Unique().Required()}
}

func (PartnerShippingPreset) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("partner_id", "preset_type", "enabled", "sort_order"),
		index.Fields("partner_id", "preset_type", "is_default"),
	}
}
