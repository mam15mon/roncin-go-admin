package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// EnterpriseResourceAddress 保存地址资源详情。
type EnterpriseResourceAddress struct{ ent.Schema }

func (EnterpriseResourceAddress) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (EnterpriseResourceAddress) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("resource_id", uuid.Nil).Immutable(),
		field.String("contact_name").Optional().Nillable().MaxLen(100),
		field.String("contact_phone").Optional().Nillable().MaxLen(64),
		field.String("country_code").NotEmpty().MinLen(2).MaxLen(2),
		field.String("province_code").Optional().Nillable().MaxLen(32),
		field.String("city_code").Optional().Nillable().MaxLen(32),
		field.String("district_code").Optional().Nillable().MaxLen(32),
		field.String("address_detail").NotEmpty().MaxLen(1000),
		field.String("remark").Optional().Nillable().MaxLen(500),
	}
}

func (EnterpriseResourceAddress) Edges() []ent.Edge {
	return []ent.Edge{edge.From("resource", EnterpriseResource.Type).Ref("address").Field("resource_id").Unique().Required().Immutable()}
}

func (EnterpriseResourceAddress) Indexes() []ent.Index {
	return []ent.Index{index.Fields("resource_id").Unique()}
}
