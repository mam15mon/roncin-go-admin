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
type OrderFee struct{ ent.Schema }

func (OrderFee) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderFee) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("order_id", uuid.Nil),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.Enum("direction").Values("RECEIVABLE", "PAYABLE"),
		field.Enum("status").Values("DRAFT", "CONFIRMED", "BILLED", "CANCELLED").Default("DRAFT"),
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
		field.Enum("exchange_rate_source").Values("SYSTEM", "BASE_CURRENCY", "MANUAL"),
		field.String("exchange_rate_date").NotEmpty().MinLen(10).MaxLen(10),
		field.UUID("exchange_rate_setting_id", uuid.Nil).Optional().Nillable(),
		field.String("base_currency").NotEmpty().MinLen(3).MaxLen(3),
		field.String("base_currency_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}),
		field.String("expense_date").NotEmpty().MinLen(10).MaxLen(10),
		field.String("note").Optional().MaxLen(500),
		field.Uint64("version").Default(1),
		field.Time("cancelled_at").Optional().Nillable(),
		field.UUID("cancelled_by", uuid.Nil).Optional().Nillable(),
		field.String("cancellation_reason").Optional().Nillable().MaxLen(500),
	}
}

func (OrderFee) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).Ref("fees").Field("order_id").Unique().Required(),
		edge.From("fee_setting", FeeSetting.Type).Ref("order_fees").Field("fee_setting_id").Unique(),
		edge.From("settlement_party", Partner.Type).Ref("order_fees").Field("settlement_party_id").Unique().Required(),
		edge.From("billing_unit_ref", BillingUnit.Type).Ref("order_fees").Field("billing_unit_id").Unique(),
		edge.From("cancelled_by_user", User.Type).Ref("cancelled_order_fees").Field("cancelled_by").Unique(),
	}
}

func (OrderFee) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "idempotency_key").Unique(),
		index.Fields("order_id", "direction", "created_at"),
		index.Fields("order_id", "status", "created_at"),
		index.Fields("fee_setting_id"),
		index.Fields("billing_unit_id"),
		index.Fields("settlement_party_id", "direction", "currency"),
	}
}
