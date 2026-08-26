package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceInvoiceLine 保存开票申请创建时按费用项目和税率归集的不可变税务明细快照。
type FinanceInvoiceLine struct{ ent.Schema }

func (FinanceInvoiceLine) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceInvoiceLine) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("invoice_id", uuid.Nil).Immutable(),
		field.Int("line_no").Positive().Immutable(),
		field.String("item_code").NotEmpty().MaxLen(30).Immutable(),
		field.String("item_name").NotEmpty().MaxLen(80).Immutable(),
		field.String("tax_rate").SchemaType(map[string]string{dialect.Postgres: "numeric(7,4)"}).Immutable(),
		field.String("net_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("tax_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("total_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.Int("source_line_count").Positive().Immutable(),
	}
}

func (FinanceInvoiceLine) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("invoice", FinanceInvoice.Type).Ref("lines").Field("invoice_id").Unique().Required().Immutable(),
	}
}

func (FinanceInvoiceLine) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("invoice_id", "line_no").Unique(),
	}
}
