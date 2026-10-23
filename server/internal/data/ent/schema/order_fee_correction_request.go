package schema

import (
	"encoding/json"
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderFeeCorrectionRequest 保存受控费用更正的不可变申请，批准保留原费用及账务事实。
type OrderFeeCorrectionRequest struct{ ent.Schema }

func (OrderFeeCorrectionRequest) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }
func (OrderFeeCorrectionRequest) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("order_id", uuid.Nil).Immutable(),
		field.Enum("operation").Values("ADD", "REPLACE", "VOID").Immutable(),
		field.UUID("source_fee_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.Uint64("source_fee_version").Optional().Nillable().Immutable(),
		field.JSON("before_snapshot", json.RawMessage{}).Optional().Immutable(),
		field.JSON("after_snapshot", json.RawMessage{}).Optional().Immutable(),
		field.UUID("approved_fee_review_event_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("lock_record_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.Uint64("lock_generation").Optional().Nillable().Immutable(),
		field.Uint64("approved_fee_content_version").Immutable(),
		field.Uint64("approved_fee_review_version").Immutable(),
		field.JSON("approval_result_snapshot", json.RawMessage{}).Optional(),
		field.String("reason").NotEmpty().MaxRuneLen(1000).Immutable(),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.String("request_fingerprint").NotEmpty().MaxLen(64).Immutable(),
		field.UUID("requested_by", uuid.Nil).Immutable(),
		field.Time("requested_at").Immutable(),
		field.Enum("status").Values("PENDING", "APPROVED", "REJECTED", "WITHDRAWN").Default("PENDING"),
		field.Uint64("version").Default(1),
		field.UUID("decided_by", uuid.Nil).Optional().Nillable(),
		field.Time("decided_at").Optional().Nillable(),
		field.String("decision_reason").Optional().Nillable().MaxRuneLen(1000),
		field.String("approval_token").Optional().Nillable().MaxLen(64),
	}
}
func (OrderFeeCorrectionRequest) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("lock_record", OrderLockRecord.Type).Ref("fee_correction_requests").Field("lock_record_id").Unique().Immutable().Annotations(entsql.OnDelete(entsql.NoAction)),
		edge.From("organization", Organization.Type).Ref("fee_correction_requests").Field("organization_id").Unique().Required().Immutable(),
		edge.From("order", Order.Type).Ref("fee_correction_requests").Field("order_id").Unique().Required().Immutable(),
		edge.From("requested_by_user", User.Type).Ref("requested_fee_correction_requests").Field("requested_by").Unique().Required().Immutable(),
		edge.From("decided_by_user", User.Type).Ref("decided_fee_correction_requests").Field("decided_by").Unique(),
		edge.From("source_fee", OrderFee.Type).Ref("targeted_correction_requests").Field("source_fee_id").Unique().Immutable(),
		edge.To("nettings", FinanceNetting.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
		edge.To("commission_adjustments", FinanceCommissionAdjustment.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
		edge.To("superseded_commission_adjustments", FinanceCommissionAdjustment.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
		edge.To("generated_fees", OrderFee.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
		edge.To("superseded_fees", OrderFee.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
	}
}
func (OrderFeeCorrectionRequest) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "idempotency_key").Unique(),
		index.Fields("organization_id", "order_id", "requested_at"),
		index.Fields("order_id").Unique().StorageKey("order_fee_correction_one_pending").Annotations(entsql.IndexWhere("status = 'PENDING'")),
	}
}
func (OrderFeeCorrectionRequest) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Checks(map[string]string{
		"order_fee_correction_lock_basis_check":     "(lock_record_id IS NULL AND lock_generation IS NULL) OR (lock_record_id IS NOT NULL AND lock_generation > 0)",
		"order_fee_correction_status_check":         "status IN ('PENDING','APPROVED','REJECTED','WITHDRAWN')",
		"order_fee_correction_operation_check":      "(operation = 'ADD' AND source_fee_id IS NULL AND before_snapshot IS NULL AND after_snapshot IS NOT NULL) OR (operation = 'REPLACE' AND source_fee_id IS NOT NULL AND before_snapshot IS NOT NULL AND after_snapshot IS NOT NULL) OR (operation = 'VOID' AND source_fee_id IS NOT NULL AND before_snapshot IS NOT NULL AND after_snapshot IS NULL)",
		"order_fee_correction_approval_basis_check": "approved_fee_content_version > 0 AND version > 0",
	})}
}
