package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceBill 将同一结算单位、收付方向和币种的已确认费用固化为结算账单。
type FinanceBill struct{ ent.Schema }

func (FinanceBill) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceBill) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.String("bill_no").NotEmpty().MaxLen(64).Immutable(),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.UUID("batch_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.Enum("direction").Values("RECEIVABLE", "PAYABLE").Immutable(),
		field.Enum("status").Values("DRAFT", "CONFIRMED", "CANCELLED").Default("DRAFT"),
		field.UUID("settlement_party_id", uuid.Nil).Immutable(),
		field.String("settlement_party_name").NotEmpty().MaxLen(200).Immutable(),
		field.String("currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("base_currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("exchange_rate").SchemaType(map[string]string{dialect.Postgres: "numeric(18,8)"}),
		field.Enum("exchange_rate_source").Values("SYSTEM", "BASE_CURRENCY", "MANUAL", "DERIVED"),
		field.String("exchange_rate_date").NotEmpty().MinLen(10).MaxLen(10),
		field.UUID("exchange_rate_setting_id", uuid.Nil).Optional().Nillable(),
		field.String("total_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("net_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("tax_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("base_currency_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}),
		field.Int("fee_count").Positive().Immutable(),
		field.String("bill_date").NotEmpty().MinLen(10).MaxLen(10),
		field.String("statement_title").Optional().Nillable().MaxLen(200),
		field.Int("payment_terms_days").Optional().Nillable().Min(0).Max(3650),
		field.String("due_date").Optional().Nillable().MinLen(10).MaxLen(10),
		field.String("note").Optional().Nillable().MaxLen(500),
		field.Uint64("version").Default(1),
		field.Time("confirmed_at").Optional().Nillable(),
		field.UUID("confirmed_by", uuid.Nil).Optional().Nillable(),
		field.Time("cancelled_at").Optional().Nillable(),
		field.UUID("cancelled_by", uuid.Nil).Optional().Nillable(),
		field.String("cancellation_reason").Optional().Nillable().MaxLen(500),
	}
}

func (FinanceBill) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("finance_bills").Field("organization_id").Unique().Required().Immutable(),
		edge.From("batch", FinanceBillBatch.Type).Ref("bills").Field("batch_id").Unique().Immutable(),
		edge.From("settlement_party", Partner.Type).Ref("finance_bills").Field("settlement_party_id").Unique().Required().Immutable(),
		edge.From("confirmed_by_user", User.Type).Ref("confirmed_finance_bills").Field("confirmed_by").Unique(),
		edge.From("cancelled_by_user", User.Type).Ref("cancelled_finance_bills").Field("cancelled_by").Unique(),
		edge.To("lines", FinanceBillLine.Type),
		edge.To("invoice_links", FinanceInvoiceBill.Type),
		edge.To("verification_allocations", FinanceVerificationAllocation.Type),
	}
}

func (FinanceBill) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "bill_no").Unique(),
		index.Fields("organization_id", "idempotency_key").Unique(),
		index.Fields("organization_id", "status", "bill_date"),
		index.Fields("settlement_party_id", "direction", "currency"),
		index.Fields("batch_id"),
	}
}
