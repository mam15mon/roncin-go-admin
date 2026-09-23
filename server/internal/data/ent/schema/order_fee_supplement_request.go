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

// OrderFeeSupplementRequest 记录业务锁或提成净额财务锁成立期间发起的锁后
// 应付费用补录申请。申请保存审批所需的不可变费用快照与提交当时成立的锁依据；
// 审批通过后按快照原样创建一条 CONFIRMED 订单费用并通过 supplement_request_id
// 反向关联本申请。
// 不可变边界：除 status、version 与 decided_* 外全部字段创建后不可变，费用
// 快照字段由仓储在申请行锁内复核（禁止静默改金额或关键字段）；审批、驳回与
// 撤回统一锁定申请行并比较乐观锁版本，PENDING 只能进入一个终态。
type OrderFeeSupplementRequest struct{ ent.Schema }

func (OrderFeeSupplementRequest) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderFeeSupplementRequest) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Checks(map[string]string{
			// 锁依据完整性：BUSINESS 仅保存大于零的业务锁代次；FINANCIAL 仅保存
			// 非空的财务证据版本、证据哈希与大于零的净额快照；BOTH 同时保存两组
			// 依据。锁类型、锁代次与财务证据均由服务端在 Order 行锁内判定固化，
			// 不接受客户端覆盖。
			"order_fee_supplement_requests_lock_basis_check": "(lock_basis = 'BUSINESS' AND business_lock_generation IS NOT NULL AND business_lock_generation > 0 AND financial_lock_evidence_version IS NULL AND financial_lock_evidence_hash IS NULL AND financial_lock_net_amount_snapshot IS NULL) OR (lock_basis = 'FINANCIAL' AND business_lock_generation IS NULL AND financial_lock_evidence_version IS NOT NULL AND financial_lock_evidence_hash IS NOT NULL AND financial_lock_net_amount_snapshot IS NOT NULL AND financial_lock_net_amount_snapshot > 0) OR (lock_basis = 'BOTH' AND business_lock_generation IS NOT NULL AND business_lock_generation > 0 AND financial_lock_evidence_version IS NOT NULL AND financial_lock_evidence_hash IS NOT NULL AND financial_lock_net_amount_snapshot IS NOT NULL AND financial_lock_net_amount_snapshot > 0)",
			// 申请状态取值；PENDING 只能进入一个终态由状态机与仓储校验保证。
			"order_fee_supplement_requests_status_check": "status IN ('PENDING', 'APPROVED', 'REJECTED', 'WITHDRAWN')",
		}),
	}
}

func (OrderFeeSupplementRequest) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("order_id", uuid.Nil).Immutable(),
		// 提交当时成立的锁依据：仅业务锁、仅财务锁或双锁。
		field.Enum("lock_basis").Values("BUSINESS", "FINANCIAL", "BOTH").Immutable(),
		field.Uint64("business_lock_generation").Optional().Nillable().Immutable(),
		// 财务锁证据版本标识证据规范编码算法；哈希对当时参与净额的
		// CONFIRMED/PAID 提成行与调整行按「组件类型 + 主键 + 方向 + 符号化 8 位
		// 金额」排序后的规范编码计算，CONFIRMED 与 PAID 统一编码为 ACTIVE。
		field.String("financial_lock_evidence_version").Optional().Nillable().MaxLen(64).Immutable(),
		field.String("financial_lock_evidence_hash").Optional().Nillable().MaxLen(64).Immutable(),
		field.String("financial_lock_net_amount_snapshot").Optional().Nillable().
			SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		// 组织级幂等键；同一幂等键同一 request_fingerprint 的语义重放返回原申请，
		// 同键不同指纹返回稳定幂等冲突。
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.String("request_fingerprint").NotEmpty().MaxLen(128).Immutable(),
		// 以下为不可变应付费用快照，字段语义与 order_fees 对齐；仅允许正数应付
		// 成本，应收方向在领域与传输边界拒绝。
		field.Enum("direction").Values("RECEIVABLE", "PAYABLE").Immutable(),
		field.UUID("fee_setting_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.String("fee_code").NotEmpty().MaxLen(30).Immutable(),
		field.String("fee_name").NotEmpty().MaxLen(80).Immutable(),
		field.String("fee_name_en").Optional().Nillable().MaxLen(128).Immutable(),
		field.UUID("settlement_party_id", uuid.Nil).Immutable(),
		field.UUID("billing_unit_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.String("billing_unit").NotEmpty().MaxLen(32).Immutable(),
		field.String("tax_rate").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "numeric(5,2)"}).Immutable(),
		field.String("taxable_service_name").Optional().Nillable().MaxLen(128).Immutable(),
		field.String("quantity").SchemaType(map[string]string{dialect.Postgres: "numeric(18,4)"}).Immutable(),
		field.String("unit_price").SchemaType(map[string]string{dialect.Postgres: "numeric(18,4)"}).Immutable(),
		field.String("total_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.Bool("tax_inclusive").Default(true).Immutable(),
		field.String("net_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("tax_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("exchange_rate").SchemaType(map[string]string{dialect.Postgres: "numeric(18,8)"}).Immutable(),
		field.Enum("exchange_rate_source").Values("SYSTEM", "MANUAL", "DERIVED", "WEEKLY", "INHERITED_LAST_WEEK", "BOC_SYNC").Immutable(),
		field.String("exchange_rate_date").NotEmpty().MinLen(10).MaxLen(10).Immutable(),
		field.UUID("exchange_rate_setting_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.String("base_currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("base_currency_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("expense_date").NotEmpty().MinLen(10).MaxLen(16).Immutable(),
		field.String("note").Optional().MaxLen(500).Immutable(),
		// 补录原因必填，随快照一起参与 request_fingerprint。
		field.String("reason").NotEmpty().MaxLen(500).Immutable(),
		field.UUID("requested_by", uuid.Nil).Immutable(),
		field.Time("requested_at").Immutable(),
		field.Enum("status").Values("PENDING", "APPROVED", "REJECTED", "WITHDRAWN").Default("PENDING"),
		field.Uint64("version").Default(1),
		// 审批或驳回的决策人与时间；WITHDRAWN 由发起人触发，决策字段保持为空，
		// 撤回事实记录在业务审计中。
		field.UUID("decided_by", uuid.Nil).Optional().Nillable(),
		field.Time("decided_at").Optional().Nillable(),
		field.String("decision_reason").Optional().Nillable().MaxLen(500),
	}
}

func (OrderFeeSupplementRequest) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("order_fee_supplement_requests").Field("organization_id").Unique().Required().Immutable(),
		edge.From("order", Order.Type).Ref("fee_supplement_requests").Field("order_id").Unique().Required().Immutable(),
		edge.From("requested_by_user", User.Type).Ref("requested_order_fee_supplement_requests").Field("requested_by").Unique().Required().Immutable(),
		edge.From("decided_by_user", User.Type).Ref("decided_order_fee_supplement_requests").Field("decided_by").Unique(),
		// 审批生成的一对一费用与关联冲减调整必须保留历史引用：申请不可删除，
		// 外键显式 NO ACTION，禁止删除申请造成孤儿费用或孤儿调整。
		edge.To("fees", OrderFee.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
		edge.To("commission_adjustments", FinanceCommissionAdjustment.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
	}
}

func (OrderFeeSupplementRequest) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "idempotency_key").Unique(),
		index.Fields("organization_id", "order_id"),
		index.Fields("order_id", "status"),
	}
}
