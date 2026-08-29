package schema

import (
	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// EnterpriseResourceParty 保存发货人、收货人和通知人详情。
type EnterpriseResourceParty struct{ ent.Schema }

func (EnterpriseResourceParty) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (EnterpriseResourceParty) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("resource_id", uuid.Nil).Immutable(),
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.Enum("resource_type").Values("SHIPPER", "CONSIGNEE", "NOTIFY_PARTY").Immutable(),
		field.String("company_name").NotEmpty().MaxLen(200),
		field.String("business_code").Optional().Nillable().MaxLen(64),
		field.String("normalized_business_code").Optional().Nillable().MaxLen(64),
		field.String("address").Optional().Nillable().MaxLen(1000),
		field.String("country_code").NotEmpty().MinLen(2).MaxLen(2).Default("CN"),
		field.String("contact_name").Optional().Nillable().MaxLen(100),
		field.String("contact_phone").Optional().Nillable().MaxLen(64),
		field.String("email").Optional().Nillable().MaxLen(254),
		field.String("tax_identifier").Optional().Nillable().MaxLen(64),
		field.String("aeo_code").Optional().Nillable().MaxLen(64),
		field.Bool("custom_display").Default(false),
		field.String("display_content").Optional().Nillable().MaxLen(4000),
		field.String("remark").Optional().Nillable().MaxLen(500),
	}
}

func (EnterpriseResourceParty) Edges() []ent.Edge {
	return []ent.Edge{edge.From("resource", EnterpriseResource.Type).Ref("party").Field("resource_id").Unique().Required().Immutable()}
}

func (EnterpriseResourceParty) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("resource_id").Unique(),
		index.Fields("organization_id", "resource_type", "normalized_business_code").Unique().Annotations(entsql.IndexWhere("normalized_business_code IS NOT NULL AND normalized_business_code <> ''")),
	}
}
