package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderServiceType 定义订单与服务类型的关联关系。
type OrderServiceType struct{ ent.Schema }

func (OrderServiceType) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderServiceType) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("order_id", uuid.Nil),
		field.UUID("master_data_item_id", uuid.Nil),
	}
}

func (OrderServiceType) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).Ref("service_types").Field("order_id").Unique().Required(),
	}
}

func (OrderServiceType) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "master_data_item_id").Unique(),
	}
}
