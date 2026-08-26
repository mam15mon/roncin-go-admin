package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceBillLine 保存费用进入账单时的不可变快照；取消账单只停用关联，不删除历史。
type FinanceBillLine struct{ ent.Schema }

func (FinanceBillLine) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceBillLine) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("bill_id", uuid.Nil).Immutable(),
		field.UUID("order_fee_id", uuid.Nil).Immutable(),
		field.UUID("order_id", uuid.Nil).Immutable(),
		field.String("order_no").NotEmpty().MaxLen(64).Immutable(),
		field.String("fee_code").NotEmpty().MaxLen(30).Immutable(),
		field.String("fee_name").NotEmpty().MaxLen(80).Immutable(),
		field.String("total_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("net_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("tax_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("tax_rate").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "numeric(7,4)"}).Immutable(),
		field.String("currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("exchange_rate").SchemaType(map[string]string{dialect.Postgres: "numeric(18,8)"}).Immutable(),
		field.String("base_currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("base_currency_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.Bool("active").Default(true),
	}
}

func (FinanceBillLine) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("bill", FinanceBill.Type).Ref("lines").Field("bill_id").Unique().Required().Immutable(),
		edge.From("order_fee", OrderFee.Type).Ref("finance_bill_lines").Field("order_fee_id").Unique().Required().Immutable(),
		edge.From("order", Order.Type).Ref("finance_bill_lines").Field("order_id").Unique().Required().Immutable(),
	}
}

func (FinanceBillLine) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("bill_id", "active"),
		index.Fields("order_fee_id", "active"),
		index.Fields("order_id"),
	}
}
