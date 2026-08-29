package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// EnterpriseResourceShippingText 保存英文品名、HS 编码和唛头详情。
type EnterpriseResourceShippingText struct{ ent.Schema }

func (EnterpriseResourceShippingText) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (EnterpriseResourceShippingText) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("resource_id", uuid.Nil).Immutable(),
		field.String("content").Optional().Nillable().MaxLen(4000),
		field.String("code").Optional().Nillable().MaxLen(64),
		field.String("remark").Optional().Nillable().MaxLen(500),
	}
}

func (EnterpriseResourceShippingText) Edges() []ent.Edge {
	return []ent.Edge{edge.From("resource", EnterpriseResource.Type).Ref("shipping_text").Field("resource_id").Unique().Required().Immutable()}
}

func (EnterpriseResourceShippingText) Indexes() []ent.Index {
	return []ent.Index{index.Fields("resource_id").Unique()}
}
