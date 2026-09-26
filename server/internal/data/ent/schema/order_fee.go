package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderFee 定义订单费用录入明细。
// 十进制值在 Go 侧按字符串持久化，并在 PostgreSQL 中使用 numeric，避免浮点转换。
// 费用不再持有独立状态：是否已建账由有效账单关联（活动账单行且所属账单未取消）
// 这一关联事实决定，取消账单后历史行来源引用置空、费用本体物理删除。
type OrderFee struct{ ent.Schema }

func (OrderFee) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderFee) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("order_id", uuid.Nil),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.Enum("direction").Values("RECEIVABLE", "PAYABLE"),
		field.UUID("fee_setting_id", uuid.Nil).Optional().Nillable(),
		field.String("fee_code").NotEmpty().MaxLen(30),
		field.String("fee_name").NotEmpty().MaxLen(80),
		field.String("fee_name_en").Optional().Nillable().MaxLen(128),
		field.UUID("settlement_party_id", uuid.Nil),
		field.UUID("billing_unit_id", uuid.Nil).Optional().Nillable(),
		field.String("billing_unit").NotEmpty().MaxLen(32),
		field.String("tax_rate").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "numeric(5,2)"}),
		field.String("taxable_service_name").Optional().Nillable().MaxLen(128),
		field.String("quantity").SchemaType(map[string]string{dialect.Postgres: "numeric(18,4)"}),
		field.String("unit_price").SchemaType(map[string]string{dialect.Postgres: "numeric(18,4)"}),
		field.String("total_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}),
		field.Bool("tax_inclusive").Default(true),
		field.String("net_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}),
		field.String("tax_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}),
		field.String("currency").NotEmpty().MinLen(3).MaxLen(3),
		field.String("exchange_rate").SchemaType(map[string]string{dialect.Postgres: "numeric(18,8)"}),
		// 汇率快照来源：WEEKLY=本组织当周行（手工维护）；BOC_SYNC=本组织当周行（牌价同步）；
		// INHERITED_LAST_WEEK、DERIVED 仅供历史快照展示；SYSTEM 包含同币种恒等
		// 折算及历史公共基线快照；MANUAL=费用行现场手工覆盖。
		// BASE_CURRENCY 已随 ResolveBaseRate 退役清理。
		field.Enum("exchange_rate_source").Values("SYSTEM", "MANUAL", "DERIVED", "WEEKLY", "INHERITED_LAST_WEEK", "BOC_SYNC"),
		field.String("exchange_rate_date").NotEmpty().MinLen(10).MaxLen(10),
		field.UUID("exchange_rate_setting_id", uuid.Nil).Optional().Nillable(),
		field.String("base_currency").NotEmpty().MinLen(3).MaxLen(3),
		field.String("base_currency_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}),
		// 发生日期支持分钟精度（YYYY-MM-DD HH:mm），纯日期仍为 10 字符。
		field.String("expense_date").NotEmpty().MinLen(10).MaxLen(16),
		field.String("note").Optional().MaxLen(500),
		// 补录申请来源：审批通过时由专用事务反向关联生成的费用；一对一且创建后
		// 不可变，普通费用入口不得携带该来源。补录费用的删除仅经由补录申请专用
		// 撤销（物理删除），普通删除入口一律拒绝。
		field.UUID("supplement_request_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.Uint64("version").Default(1),
	}
}

func (OrderFee) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).Ref("fees").Field("order_id").Unique().Required(),
		edge.From("fee_setting", FeeSetting.Type).Ref("order_fees").Field("fee_setting_id").Unique(),
		edge.From("settlement_party", Partner.Type).Ref("order_fees").Field("settlement_party_id").Unique().Required(),
		edge.From("billing_unit_ref", BillingUnit.Type).Ref("order_fees").Field("billing_unit_id").Unique(),
		edge.From("supplement_request", OrderFeeSupplementRequest.Type).Ref("fees").Field("supplement_request_id").Unique().Immutable(),
		edge.To("finance_bill_lines", FinanceBillLine.Type),
		edge.To("enterprise_tag_links", OrderFeeEnterpriseTag.Type),
	}
}

func (OrderFee) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "idempotency_key").Unique(),
		// 一条补录申请只能生成一条费用；PostgreSQL 唯一索引允许多行 NULL。
		index.Fields("supplement_request_id").Unique(),
		index.Fields("order_id", "direction", "created_at"),
		index.Fields("fee_setting_id"),
		index.Fields("billing_unit_id"),
		index.Fields("settlement_party_id", "direction", "currency"),
	}
}
