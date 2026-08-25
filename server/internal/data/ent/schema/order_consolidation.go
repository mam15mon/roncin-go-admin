package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderConsolidation 定义可被多个订单共享的主单拼载批次。
type OrderConsolidation struct{ ent.Schema }

func (OrderConsolidation) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderConsolidation) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.Enum("business_type").Values("SE", "SI", "AE", "AI", "LAND", "RAIL"),
		field.String("master_no").NotEmpty().MaxLen(64),
		field.String("normalized_master_no").NotEmpty().MaxLen(64),
	}
}

func (OrderConsolidation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("order_consolidations").Field("organization_id").Unique().Required(),
		edge.To("shipping_documents", OrderShippingDocument.Type),
	}
}

func (OrderConsolidation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "business_type", "normalized_master_no").Unique(),
		index.Fields("organization_id", "business_type", "master_no"),
	}
}
