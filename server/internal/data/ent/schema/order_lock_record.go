package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderLockRecord 记录每次成功锁定的周期事实。
type OrderLockRecord struct{ ent.Schema }

func (OrderLockRecord) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }

func (OrderLockRecord) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Checks(map[string]string{
			"order_lock_records_business_type_document_refs_check": "(business_type = 'SE' AND master_bill_id IS NOT NULL AND master_bill_version_id IS NOT NULL AND transport_execution_id IS NOT NULL AND transport_execution_version_id IS NOT NULL) OR (business_type IN ('SI', 'AE', 'AI', 'LAND', 'RAIL') AND master_bill_id IS NULL AND master_bill_version_id IS NULL AND transport_execution_id IS NULL AND transport_execution_version_id IS NULL)",
			// 锁定来源一致性：人工锁定必须有 locked_by 且触发字段为空；
			// 系统自动锁定必须无 locked_by 且触发类型、触发单据与触发操作人完整。
			"order_lock_records_lock_source_check": "(lock_source = 'MANUAL' AND locked_by IS NOT NULL AND trigger_type IS NULL AND trigger_resource_id IS NULL AND triggered_by IS NULL) OR (lock_source = 'AUTO_SETTLEMENT' AND locked_by IS NULL AND trigger_type IS NOT NULL AND trigger_resource_id IS NOT NULL AND triggered_by IS NOT NULL)",
		}),
	}
}

func (OrderLockRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Immutable().Default(time.Now),
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("order_id", uuid.Nil).Immutable(),
		field.String("order_no").NotEmpty().MaxLen(64).Immutable(),
		field.Enum("business_type").Values("SE", "SI", "AE", "AI", "LAND", "RAIL").Immutable(),
		field.Uint64("generation").Immutable(),
		field.Enum("lock_source").Values("MANUAL", "AUTO_SETTLEMENT").Immutable().Default("MANUAL"),
		// 人工锁定归属实际锁定人；系统自动锁定 locked_by 为空，页面固定展示
		// 【系统自动锁定】，触发审计记录在 trigger_* 字段。
		field.UUID("locked_by", uuid.Nil).Optional().Nillable().Immutable(),
		field.Time("locked_at").Immutable(),
		field.Uint64("order_version_at_lock").Immutable(),
		field.Enum("trigger_type").Values("VERIFICATION", "NETTING", "FEE_CONFIRM", "FEE_CANCEL").Optional().Nillable().Immutable(),
		field.UUID("trigger_resource_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("triggered_by", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("master_bill_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("master_bill_version_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("transport_execution_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("transport_execution_version_id", uuid.Nil).Optional().Nillable().Immutable(),
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
		edge.From("locked_by_user", User.Type).Ref("order_lock_records").Field("locked_by").Unique().Immutable(),
		edge.From("triggered_by_user", User.Type).Ref("auto_triggered_order_lock_records").Field("triggered_by").Unique().Immutable(),
		edge.From("unlocked_by_user", User.Type).Ref("unlocked_order_lock_records").Field("unlocked_by").Unique(),
		edge.From("master_bill", SeaMasterBill.Type).Ref("lock_records").Field("master_bill_id").Unique().Immutable(),
		edge.From("master_bill_version", SeaMasterBillVersion.Type).Ref("lock_records").Field("master_bill_version_id").Unique().Immutable(),
		edge.From("transport_execution", SeaTransportExecution.Type).Ref("lock_records").Field("transport_execution_id").Unique().Immutable(),
		edge.From("transport_execution_version", SeaTransportExecutionVersion.Type).Ref("lock_records").Field("transport_execution_version_id").Unique().Immutable(),
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
