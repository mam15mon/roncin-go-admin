package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderUnlockRequest 记录所有解锁路径的本地请求事实。
type OrderUnlockRequest struct{ ent.Schema }

func (OrderUnlockRequest) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }

func (OrderUnlockRequest) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Immutable().Default(time.Now),
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("order_id", uuid.Nil).Immutable(),
		field.String("order_no").NotEmpty().MaxLen(64).Immutable(),
		field.Enum("business_type").Values("SE", "SI", "AE", "AI", "LAND", "RAIL").Immutable(),
		field.UUID("lock_record_id", uuid.Nil).Immutable(),
		field.Uint64("lock_generation").Immutable(),
		field.UUID("requested_by", uuid.Nil).Immutable(),
		field.Time("requested_at").Immutable(),
		field.String("reason").Optional().Nillable().MaxLen(500),
		field.Uint64("expected_order_version").Immutable(),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.String("request_fingerprint").NotEmpty().MaxLen(128).Immutable(),
		field.Enum("route").Values("ROLE_DIRECT", "ADMIN_EMERGENCY", "DINGTALK_APPROVAL").Immutable(),
		field.Enum("status").Values(
			"PENDING_DISPATCH",
			"PENDING_APPROVAL",
			"APPROVED_PENDING_APPLY",
			"APPROVED",
			"REJECTED",
			"CONFIGURATION_FAILED",
			"DISPATCH_FAILED",
			"DISPATCH_UNKNOWN",
			"STALE",
		).Default("PENDING_DISPATCH"),
		field.String("dingtalk_process_instance_id").Optional().Nillable().MaxLen(128),
		field.String("dingtalk_process_code").Optional().Nillable().MaxLen(128),
		field.UUID("decided_by", uuid.Nil).Optional().Nillable(),
		field.Time("decided_at").Optional().Nillable(),
		field.String("decision_source").Optional().Nillable().MaxLen(64),
		field.String("failure_code").Optional().Nillable().MaxLen(64),
		field.String("failure_message").Optional().Nillable().MaxLen(500),
		field.UUID("superseded_by_request_id", uuid.Nil).Optional().Nillable(),
		field.Time("unlocked_at").Optional().Nillable(),
		field.Uint64("result_order_version").Optional().Nillable(),
	}
}

func (OrderUnlockRequest) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("order_unlock_requests").Field("organization_id").Unique().Required().Immutable(),
		edge.From("order", Order.Type).Ref("unlock_requests").Field("order_id").Unique().Required().Immutable(),
		edge.From("lock_record", OrderLockRecord.Type).Ref("unlock_requests").Field("lock_record_id").Unique().Required().Immutable(),
		edge.From("requested_by_user", User.Type).Ref("order_unlock_requests").Field("requested_by").Unique().Required().Immutable(),
		edge.From("decided_by_user", User.Type).Ref("decided_order_unlock_requests").Field("decided_by").Unique(),
		edge.To("superseded_by_request", OrderUnlockRequest.Type).Field("superseded_by_request_id").Unique(),
		edge.From("superseded_requests", OrderUnlockRequest.Type).Ref("superseded_by_request"),
		edge.To("approver_candidates", OrderUnlockApproverCandidate.Type),
		edge.To("dispatch", DingTalkApprovalDispatch.Type).Unique(),
	}
}

func (OrderUnlockRequest) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "idempotency_key").Unique().StorageKey("order_unlock_request_idempotency_key"),
		index.Fields("dingtalk_process_instance_id").Unique().StorageKey("order_unlock_request_process_instance_id").Annotations(entsql.IndexWhere("dingtalk_process_instance_id IS NOT NULL")),
		index.Fields("order_id", "lock_generation").StorageKey("order_unlock_request_active_unique").Unique().Annotations(entsql.IndexWhere("status IN ('PENDING_DISPATCH', 'PENDING_APPROVAL', 'APPROVED_PENDING_APPLY', 'DISPATCH_UNKNOWN')")),
		index.Fields("organization_id", "order_id", "created_at"),
	}
}
