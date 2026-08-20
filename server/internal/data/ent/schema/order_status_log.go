package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderStatusLog 记录订单状态流转历史。
type OrderStatusLog struct{ ent.Schema }

func (OrderStatusLog) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }

func (OrderStatusLog) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("order_id", uuid.Nil).Immutable(),
		field.String("from_status").Optional().Nillable().MaxLen(64).Immutable(),
		field.String("to_status").NotEmpty().MaxLen(64).Immutable(),
		field.String("action").Optional().MaxLen(64).Immutable(),
		field.String("reason").Optional().MaxLen(500).Immutable(),
		field.UUID("operator_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.Time("changed_at").Immutable().Default(time.Now),
	}
}

func (OrderStatusLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).Ref("status_logs").Field("order_id").Unique().Required().Immutable(),
	}
}

func (OrderStatusLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "changed_at"),
	}
}
