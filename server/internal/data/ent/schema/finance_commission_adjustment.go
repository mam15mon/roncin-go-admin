package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceCommissionAdjustment 保存已确认提成之后形成的独立增减调整，不改写原始提成快照。
type FinanceCommissionAdjustment struct{ ent.Schema }

func (FinanceCommissionAdjustment) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceCommissionAdjustment) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Checks(map[string]string{
			"commission_adjustment_direction_check":   "direction IN ('INCREASE', 'DECREASE')",
			"commission_adjustment_status_check":      "status IN ('DRAFT', 'CONFIRMED', 'PAID', 'CANCELLED')",
			"commission_adjustment_amount_positive":   "amount > 0",
			"commission_adjustment_source_type_check": "source_type IN ('MANUAL', 'VERIFICATION_REVERSAL', 'NETTING_REVERSAL', 'LOCKED_FEE_SUPPLEMENT')",
			// 来源关联互斥：仅锁后费用补录来源允许且必须携带补录申请关联；
			// 人工与核销/对冲冲减来源不得携带该关联，防止来源错配绕过唯一键。
			"commission_adjustment_source_supplement_check": "(source_type = 'LOCKED_FEE_SUPPLEMENT' AND source_fee_supplement_request_id IS NOT NULL) OR (source_type <> 'LOCKED_FEE_SUPPLEMENT' AND source_fee_supplement_request_id IS NULL)",
		}),
	}
}

func (FinanceCommissionAdjustment) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("commission_id", uuid.Nil).Immutable(),
		field.UUID("order_id", uuid.Nil).Immutable(),
		field.String("adjustment_no").NotEmpty().MaxLen(80).Immutable(),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.String("commission_no").NotEmpty().MaxLen(64).Immutable(),
		field.String("order_no").NotEmpty().MaxLen(64).Immutable(),
		field.UUID("employee_id", uuid.Nil).Immutable(),
		field.String("employee_name").NotEmpty().MaxLen(100).Immutable(),
		field.Enum("source_type").Values("MANUAL", "VERIFICATION_REVERSAL", "NETTING_REVERSAL", "LOCKED_FEE_SUPPLEMENT").Default("MANUAL").Immutable(),
		field.UUID("source_verification_id", uuid.Nil).Optional().Nillable().Immutable(),
		// 锁后费用补录来源的补录申请关联：数据库 CHECK 强制与 LOCKED_FEE_SUPPLEMENT
		// 双向对应，外键 NO ACTION 禁止删除申请造成孤儿调整。
		field.UUID("source_fee_supplement_request_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.Enum("direction").Values("INCREASE", "DECREASE").Immutable(),
		field.Enum("status").Values("DRAFT", "CONFIRMED", "PAID", "CANCELLED").Default("DRAFT"),
		field.String("base_currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("reason").NotEmpty().MaxLen(500).Immutable(),
		field.String("note").Optional().Nillable().MaxLen(500),
		field.Uint64("version").Default(1),
		field.Time("confirmed_at").Optional().Nillable(),
		field.UUID("confirmed_by", uuid.Nil).Optional().Nillable(),
		field.Time("paid_at").Optional().Nillable(),
		field.UUID("paid_by", uuid.Nil).Optional().Nillable(),
		field.Time("cancelled_at").Optional().Nillable(),
		field.UUID("cancelled_by", uuid.Nil).Optional().Nillable(),
		field.String("cancellation_reason").Optional().Nillable().MaxLen(500),
	}
}

func (FinanceCommissionAdjustment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("finance_commission_adjustments").Field("organization_id").Unique().Required().Immutable(),
		edge.From("commission", FinanceCommission.Type).Ref("adjustments").Field("commission_id").Unique().Required().Immutable(),
		edge.From("order", Order.Type).Ref("finance_commission_adjustments").Field("order_id").Unique().Required().Immutable(),
		edge.From("employee", User.Type).Ref("finance_commission_adjustments").Field("employee_id").Unique().Required().Immutable(),
		edge.From("source_verification", FinanceVerification.Type).Ref("commission_reversal_adjustments").Field("source_verification_id").Unique().Immutable(),
		edge.From("source_fee_supplement_request", OrderFeeSupplementRequest.Type).Ref("commission_adjustments").Field("source_fee_supplement_request_id").Unique().Immutable(),
		edge.From("confirmed_by_user", User.Type).Ref("confirmed_finance_commission_adjustments").Field("confirmed_by").Unique(),
		edge.From("paid_by_user", User.Type).Ref("paid_finance_commission_adjustments").Field("paid_by").Unique(),
		edge.From("cancelled_by_user", User.Type).Ref("cancelled_finance_commission_adjustments").Field("cancelled_by").Unique(),
	}
}

func (FinanceCommissionAdjustment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "adjustment_no").Unique(),
		index.Fields("organization_id", "idempotency_key").Unique(),
		index.Fields("commission_id", "status", "created_at"),
		index.Fields("order_id", "status"),
		index.Fields("commission_id", "order_id", "source_type", "source_verification_id").Unique(),
		// 锁后补录建议去重：同一原提成订单行对同一补录申请只生成一条调整。
		index.Fields("commission_id", "order_id", "source_type", "source_fee_supplement_request_id").Unique(),
	}
}
