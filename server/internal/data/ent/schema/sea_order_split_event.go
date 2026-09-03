package schema

import (
	"encoding/json"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// SeaOrderSplitEvent 记录海运出口操作票部分拆票的不可变事件。
type SeaOrderSplitEvent struct{ ent.Schema }

func (SeaOrderSplitEvent) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }

func (SeaOrderSplitEvent) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Immutable().Default(time.Now),
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("source_order_id", uuid.Nil).Immutable(),
		field.String("source_order_no").NotEmpty().MaxLen(64).Immutable(),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.String("request_fingerprint").NotEmpty().MaxLen(128).Immutable(),
		field.String("note").Optional().Nillable().MaxLen(500).Immutable(),
		field.Uint64("source_order_version").Immutable(),
		field.UUID("source_link_id", uuid.Nil).Immutable(),
		field.Uint64("source_link_version").Immutable(),
		field.Uint64("source_allocation_version").Immutable(),
		field.JSON("before_snapshot", json.RawMessage{}).Immutable(),
		field.JSON("conservation_snapshot", json.RawMessage{}).Immutable(),
		field.UUID("created_by", uuid.Nil).Optional().Nillable().Immutable(),
	}
}

func (SeaOrderSplitEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("sea_order_split_events").Field("organization_id").Unique().Required().Immutable(),
		edge.From("source_order", Order.Type).Ref("sea_order_split_events").Field("source_order_id").Unique().Required().Immutable(),
		edge.From("creator", User.Type).Ref("created_sea_order_split_events").Field("created_by").Unique().Immutable(),
		edge.To("results", SeaOrderSplitResult.Type),
		edge.To("reassignments", SeaOrderReassignmentEvent.Type),
	}
}

func (SeaOrderSplitEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "idempotency_key").Unique().StorageKey("sea_order_split_event_idempotency_key"),
		index.Fields("organization_id", "source_order_id", "created_at"),
		index.Fields("organization_id", "request_fingerprint"),
	}
}
