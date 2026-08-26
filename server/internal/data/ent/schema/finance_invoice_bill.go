package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceInvoiceBill 保存发票与账单的关联及金额快照。
type FinanceInvoiceBill struct{ ent.Schema }

func (FinanceInvoiceBill) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceInvoiceBill) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("invoice_id", uuid.Nil).Immutable(),
		field.UUID("bill_id", uuid.Nil).Immutable(),
		field.String("bill_no").NotEmpty().MaxLen(64).Immutable(),
		field.String("amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("tax_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.Bool("active").Default(true),
	}
}

func (FinanceInvoiceBill) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("invoice", FinanceInvoice.Type).Ref("bill_links").Field("invoice_id").Unique().Required().Immutable(),
		edge.From("bill", FinanceBill.Type).Ref("invoice_links").Field("bill_id").Unique().Required().Immutable(),
	}
}

func (FinanceInvoiceBill) Indexes() []ent.Index {
	return []ent.Index{index.Fields("invoice_id", "active"), index.Fields("bill_id", "active")}
}
