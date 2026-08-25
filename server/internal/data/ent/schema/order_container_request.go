package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderContainerRequest 定义订单订舱阶段的箱型箱量计划。
type OrderContainerRequest struct{ ent.Schema }

func (OrderContainerRequest) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderContainerRequest) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("order_id", uuid.Nil),
		field.UUID("container_spec_id", uuid.Nil),
		field.Int("quantity").Positive(),
	}
}

func (OrderContainerRequest) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).Ref("container_requests").Field("order_id").Unique().Required(),
	}
}

func (OrderContainerRequest) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "container_spec_id").Unique(),
		index.Fields("container_spec_id"),
	}
}
