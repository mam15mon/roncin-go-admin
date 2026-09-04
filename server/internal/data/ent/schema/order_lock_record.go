package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderLockRecord 记录每次成功锁定的周期事实。
type OrderLockRecord struct{ ent.Schema }

func (OrderLockRecord) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }

func (OrderLockRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Immutable().Default(time.Now),
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("order_id", uuid.Nil).Immutable(),
		field.String("order_no").NotEmpty().MaxLen(64).Immutable(),
		field.Uint64("generation").Immutable(),
		field.UUID("locked_by", uuid.Nil).Immutable(),
		field.Time("locked_at").Immutable(),
		field.Uint64("order_version_at_lock").Immutable(),
		field.UUID("master_bill_id", uuid.Nil).Immutable(),
		field.UUID("master_bill_version_id", uuid.Nil).Immutable(),
		field.UUID("unlocked_by", uuid.Nil).Optional().Nillable(),
		field.Time("unlocked_at").Optional().Nillable(),
		field.Uint64("order_version_at_unlock").Optional().Nillable(),
		field.UUID("unlock_request_id", uuid.Nil).Optional().Nillable(),
		field.String("unlock_reason").Optional().Nillable().MaxLen(500),
		field.Enum("unlock_mode").Values("ROLE_DIRECT", "ADMIN_EMERGENCY", "DINGTALK_APPROVED").Optional().Nillable(),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.String("request_fingerprint").NotEmpty().MaxLen(128).Immutable(),
	}
}

func (OrderLockRecord) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("order_lock_records").Field("organization_id").Unique().Required().Immutable(),
		edge.From("order", Order.Type).Ref("lock_records").Field("order_id").Unique().Required().Immutable(),
		edge.From("locked_by_user", User.Type).Ref("order_lock_records").Field("locked_by").Unique().Required().Immutable(),
		edge.From("unlocked_by_user", User.Type).Ref("unlocked_order_lock_records").Field("unlocked_by").Unique(),
		edge.From("master_bill", SeaMasterBill.Type).Ref("lock_records").Field("master_bill_id").Unique().Required().Immutable(),
		edge.From("master_bill_version", SeaMasterBillVersion.Type).Ref("lock_records").Field("master_bill_version_id").Unique().Required().Immutable(),
		edge.To("unlock_requests", OrderUnlockRequest.Type),
		edge.To("applied_unlock_request", OrderUnlockRequest.Type).Field("unlock_request_id").Unique(),
		edge.To("house_bill_snapshots", OrderLockHouseBillSnapshot.Type),
	}
}

func (OrderLockRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "generation").Unique().StorageKey("order_lock_record_order_generation"),
		index.Fields("organization_id", "idempotency_key").Unique().StorageKey("order_lock_record_idempotency_key"),
		index.Fields("organization_id", "order_id"),
		index.Fields("organization_id", "locked_at"),
	}
}
