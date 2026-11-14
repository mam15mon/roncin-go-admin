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

// FinanceInvoiceFeeAllocation 保存本次发票到具体账单费用的不可变金额与来源税率。
// 活动性只沿父账单关联及发票状态推导，修订停用旧关联并追加新事实。
type FinanceInvoiceFeeAllocation struct{ ent.Schema }

func (FinanceInvoiceFeeAllocation) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }
func (FinanceInvoiceFeeAllocation) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("invoice_id", uuid.Nil).Immutable(),
		field.UUID("invoice_bill_id", uuid.Nil).Immutable(),
		field.UUID("bill_line_id", uuid.Nil).Immutable(),
		// 来源ID保留为不可变历史身份；费用删除不改写已登记凭证的来源及金额快照。
		field.UUID("order_fee_id", uuid.Nil).Immutable(),
		field.String("fee_code").NotEmpty().MaxLen(30).Immutable(),
		field.String("fee_name").NotEmpty().MaxRuneLen(80).Immutable(),
		field.String("currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("source_tax_rate").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "numeric(7,4)"}).Immutable(),
		field.String("amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("net_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("tax_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
	}
}
func (FinanceInvoiceFeeAllocation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("invoice", FinanceInvoice.Type).Ref("fee_allocations").Field("invoice_id").Unique().Required().Immutable(),
		edge.From("invoice_bill", FinanceInvoiceBill.Type).Ref("fee_allocations").Field("invoice_bill_id").Unique().Required().Immutable(),
		edge.From("bill_line", FinanceBillLine.Type).Ref("invoice_allocations").Field("bill_line_id").Unique().Required().Immutable(),
	}
}
func (FinanceInvoiceFeeAllocation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("invoice_bill_id", "bill_line_id").Unique().StorageKey("finance_invoice_fee_allocation_link_line"),
		index.Fields("bill_line_id"), index.Fields("order_fee_id"),
	}
}
func (FinanceInvoiceFeeAllocation) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Checks(map[string]string{
		"finance_invoice_fee_allocation_amount_check": "amount > 0 AND tax_amount >= 0 AND tax_amount <= amount AND net_amount = amount - tax_amount",
	})}
}
