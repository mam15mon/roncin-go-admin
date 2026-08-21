package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// PartnerProfile 保存企业主体的通用业务资料，供客户、供应商和国外代理角色复用。
type PartnerProfile struct{ ent.Schema }

func (PartnerProfile) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (PartnerProfile) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("partner_id", uuid.Nil),
		field.String("name_en").Optional().MaxLen(200),
		field.String("address_en").Optional().MaxLen(500),
		field.String("country_code").NotEmpty().MinLen(2).MaxLen(2).Default("CN"),
		field.String("province_code").Optional().MaxLen(12),
		field.String("city_code").Optional().MaxLen(12),
		field.String("district_code").Optional().MaxLen(12),
		field.String("address_detail").Optional().MaxLen(500),
		field.String("nature").Optional().MaxLen(100),
		field.String("development_method").Optional().MaxLen(100),
		field.Strings("customer_types").Optional(),
		field.Strings("business_types").Optional(),
		field.String("remark").Optional().MaxLen(2000),
	}
}

func (PartnerProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("partner", Partner.Type).Ref("profile").Field("partner_id").Unique().Required(),
	}
}

func (PartnerProfile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("partner_id").Unique(),
		index.Fields("province_code", "city_code", "district_code"),
	}
}
