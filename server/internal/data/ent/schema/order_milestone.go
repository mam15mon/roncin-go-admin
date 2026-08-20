package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderMilestone 定义订单里程碑记录。
type OrderMilestone struct{ ent.Schema }

func (OrderMilestone) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderMilestone) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("order_id", uuid.Nil),
		field.String("type").NotEmpty().MaxLen(64),
		field.String("template_node_code").Optional().Nillable().MaxLen(64),
		field.String("template_node_label").Optional().Nillable().MaxLen(100),
		field.Time("occurred_at").Optional().Nillable(),
		field.String("note").Optional().Nillable().MaxLen(500),
		field.UUID("updated_by", uuid.Nil).Optional().Nillable(),
	}
}

func (OrderMilestone) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).Ref("milestones").Field("order_id").Unique().Required(),
	}
}

func (OrderMilestone) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "type").Unique(),
		index.Fields("order_id", "occurred_at"),
	}
}
