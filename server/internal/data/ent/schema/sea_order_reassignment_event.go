package schema

import (
	"encoding/json"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// SeaOrderReassignmentEvent 记录海运出口操作票整体改配的不可变事件。
type SeaOrderReassignmentEvent struct{ ent.Schema }

func (SeaOrderReassignmentEvent) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }

func (SeaOrderReassignmentEvent) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Checks(map[string]string{
			"sea_order_reassignment_events_responsibility_type_check": "responsibility_type IN ('CARRIER', 'CUSTOMER', 'CUSTOMS', 'OWN_COMPANY', 'FORCE_MAJEURE', 'OTHER')",
		}),
	}
}

func (SeaOrderReassignmentEvent) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Immutable().Default(time.Now),
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("order_id", uuid.Nil).Immutable(),
		field.String("order_no").NotEmpty().MaxLen(64).Immutable(),
		field.UUID("split_event_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("split_result_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.String("request_fingerprint").NotEmpty().MaxLen(128).Immutable(),
		field.UUID("previous_master_bill_id", uuid.Nil).Immutable(),
		field.UUID("target_master_bill_id", uuid.Nil).Immutable(),
		field.UUID("previous_transport_execution_id", uuid.Nil).Immutable(),
		field.UUID("target_transport_execution_id", uuid.Nil).Immutable(),
		field.UUID("previous_link_id", uuid.Nil).Immutable(),
		field.UUID("target_link_id", uuid.Nil).Immutable(),
		field.Uint64("previous_link_version").Immutable(),
		field.Uint64("target_link_version").Immutable(),
		field.String("reason").NotEmpty().MaxLen(500).Immutable(),
		field.Enum("responsibility_type").Values("CARRIER", "CUSTOMER", "CUSTOMS", "OWN_COMPANY", "FORCE_MAJEURE", "OTHER").Immutable(),
		field.UUID("responsible_partner_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.String("responsible_partner_name").Optional().Nillable().MaxLen(255).Immutable(),
		field.JSON("before_snapshot", json.RawMessage{}).Immutable(),
		field.JSON("after_snapshot", json.RawMessage{}).Immutable(),
		field.UUID("created_by", uuid.Nil).Optional().Nillable().Immutable(),
	}
}

func (SeaOrderReassignmentEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("sea_order_reassignment_events").Field("organization_id").Unique().Required().Immutable(),
		edge.From("order", Order.Type).Ref("sea_order_reassignment_events").Field("order_id").Unique().Required().Immutable(),
		edge.From("split_event", SeaOrderSplitEvent.Type).Ref("reassignments").Field("split_event_id").Unique().Immutable(),
		edge.From("split_result", SeaOrderSplitResult.Type).Ref("reassignment_events").Field("split_result_id").Unique().Immutable(),
		edge.From("previous_master_bill", SeaMasterBill.Type).Ref("previous_sea_order_reassignments").Field("previous_master_bill_id").Unique().Required().Immutable(),
		edge.From("target_master_bill", SeaMasterBill.Type).Ref("target_sea_order_reassignments").Field("target_master_bill_id").Unique().Required().Immutable(),
		edge.From("responsible_partner", Partner.Type).Ref("sea_order_reassignments").Field("responsible_partner_id").Unique().Immutable(),
		edge.From("creator", User.Type).Ref("created_sea_order_reassignment_events").Field("created_by").Unique().Immutable(),
	}
}

func (SeaOrderReassignmentEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "idempotency_key").Unique().StorageKey("sea_order_reassignment_event_idempotency_key"),
		index.Fields("organization_id", "order_id", "created_at"),
		index.Fields("organization_id", "request_fingerprint"),
	}
}
