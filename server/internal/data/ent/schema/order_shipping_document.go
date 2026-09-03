package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderShippingDocument 定义订单提单。
// 状态仅允许 DRAFT→CONFIRMED→RELEASED 向前流转；与集装箱的关联在提单建模后另行接入。
type OrderShippingDocument struct{ ent.Schema }

func (OrderShippingDocument) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderShippingDocument) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("order_id", uuid.Nil),
		field.String("house_no").NotEmpty().MaxLen(64),
		field.String("release_type").Optional().MaxLen(64),
		field.Enum("status").Values("DRAFT", "CONFIRMED", "RELEASED").Default("DRAFT"),
		field.String("note").Optional().MaxLen(500),
	}
}

func (OrderShippingDocument) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).Ref("shipping_documents").Field("order_id").Unique().Required(),
		edge.To("release_pods", OrderReleasePod.Type),
	}
}

func (OrderShippingDocument) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "house_no").Unique(),
		index.Fields("order_id", "status"),
	}
}
