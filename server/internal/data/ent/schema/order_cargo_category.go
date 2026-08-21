package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderCargoCategory 定义订单与货物类别的关联关系。
type OrderCargoCategory struct{ ent.Schema }

func (OrderCargoCategory) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderCargoCategory) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("order_id", uuid.Nil),
		field.UUID("master_data_item_id", uuid.Nil),
	}
}

func (OrderCargoCategory) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).Ref("cargo_categories").Field("order_id").Unique().Required(),
	}
}

func (OrderCargoCategory) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "master_data_item_id").Unique(),
	}
}
