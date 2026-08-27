package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderLifecycleEvent 记录订单流程、终止和结案维度的不可变变更历史。
type OrderLifecycleEvent struct{ ent.Schema }

func (OrderLifecycleEvent) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }

func (OrderLifecycleEvent) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("order_id", uuid.Nil).Immutable(),
		field.Enum("dimension").Values("FLOW", "TERMINATION", "CLOSURE").Immutable(),
		field.String("from_status").Optional().Nillable().MaxLen(64).Immutable(),
		field.String("to_status").NotEmpty().MaxLen(64).Immutable(),
		field.String("action").NotEmpty().MaxLen(64).Immutable(),
		field.String("reason").Optional().Nillable().MaxLen(500).Immutable(),
		field.UUID("operator_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.Time("changed_at").Default(time.Now).Immutable(),
	}
}

func (OrderLifecycleEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).Ref("lifecycle_events").Field("order_id").Unique().Required().Immutable(),
	}
}

func (OrderLifecycleEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "dimension", "changed_at"),
	}
}
