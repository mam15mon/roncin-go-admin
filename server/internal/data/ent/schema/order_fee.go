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
		field.Enum("direction").Values("RECEIVABLE", "PAYABLE"),
		field.String("fee_code").NotEmpty().MaxLen(30),
		field.String("fee_name").NotEmpty().MaxLen(80),
		field.UUID("settlement_party_id", uuid.Nil),
		field.String("billing_unit").NotEmpty().MaxLen(32),
		field.String("quantity").SchemaType(map[string]string{dialect.Postgres: "numeric(18,4)"}),
		field.String("unit_price").SchemaType(map[string]string{dialect.Postgres: "numeric(18,4)"}),
		field.String("total_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}),
		field.String("currency").NotEmpty().MinLen(3).MaxLen(3),
		field.String("exchange_rate").SchemaType(map[string]string{dialect.Postgres: "numeric(18,6)"}),
		field.String("expense_date").NotEmpty().MinLen(10).MaxLen(10),
		field.String("note").Optional().MaxLen(500),
	}
}

func (OrderFee) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).Ref("fees").Field("order_id").Unique().Required(),
		edge.From("settlement_party", Partner.Type).Ref("order_fees").Field("settlement_party_id").Unique().Required(),
	}
}

func (OrderFee) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "direction", "created_at"),
		index.Fields("settlement_party_id", "direction", "currency"),
	}
}
