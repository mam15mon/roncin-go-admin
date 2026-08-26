package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceInvoice 记录对已确认账单发起并完成的销项或进项开票。
type FinanceInvoice struct{ ent.Schema }

func (FinanceInvoice) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceInvoice) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.String("record_no").NotEmpty().MaxLen(64).Immutable(),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.Enum("direction").Values("RECEIVABLE", "PAYABLE").Immutable(),
		field.Enum("status").Values("DRAFT", "ISSUED", "CANCELLED", "RED_FLUSHED").Default("DRAFT"),
		field.Enum("invoice_type").Values("NORMAL", "SPECIAL").Immutable(),
		field.UUID("invoice_profile_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("settlement_party_id", uuid.Nil).Immutable(),
		field.String("settlement_party_name").NotEmpty().MaxLen(200).Immutable(),
		field.String("invoice_title").Optional().Nillable().MaxLen(200).Immutable(),
		field.String("taxpayer_identification_no").Optional().Nillable().MaxLen(64).Immutable(),
		field.String("registered_address").Optional().Nillable().MaxLen(500).Immutable(),
		field.String("registered_phone").Optional().Nillable().MaxLen(50).Immutable(),
		field.String("bank_name").Optional().Nillable().MaxLen(200).Immutable(),
		field.String("bank_account").Optional().Nillable().MaxLen(100).Immutable(),
		field.String("currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("total_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("net_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("tax_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.Int("bill_count").Positive().Immutable(),
		field.String("tax_invoice_no").Optional().Nillable().MaxLen(100),
		field.String("invoice_date").Optional().Nillable().MinLen(10).MaxLen(10),
		field.String("note").Optional().Nillable().MaxLen(500),
		field.Uint64("version").Default(1),
		field.Time("issued_at").Optional().Nillable(),
		field.UUID("issued_by", uuid.Nil).Optional().Nillable(),
		field.Time("cancelled_at").Optional().Nillable(),
		field.UUID("cancelled_by", uuid.Nil).Optional().Nillable(),
		field.String("cancellation_reason").Optional().Nillable().MaxLen(500),
		field.String("red_invoice_no").Optional().Nillable().MaxLen(100),
		field.String("red_invoice_date").Optional().Nillable().MinLen(10).MaxLen(10),
		field.Time("red_flushed_at").Optional().Nillable(),
		field.UUID("red_flushed_by", uuid.Nil).Optional().Nillable(),
		field.String("red_flush_reason").Optional().Nillable().MaxLen(500),
	}
}

func (FinanceInvoice) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("finance_invoices").Field("organization_id").Unique().Required().Immutable(),
		edge.From("settlement_party", Partner.Type).Ref("finance_invoices").Field("settlement_party_id").Unique().Required().Immutable(),
		edge.From("invoice_profile", PartnerInvoiceProfile.Type).Ref("finance_invoices").Field("invoice_profile_id").Unique().Immutable(),
		edge.From("issued_by_user", User.Type).Ref("issued_finance_invoices").Field("issued_by").Unique(),
		edge.From("cancelled_by_user", User.Type).Ref("cancelled_finance_invoices").Field("cancelled_by").Unique(),
		edge.From("red_flushed_by_user", User.Type).Ref("red_flushed_finance_invoices").Field("red_flushed_by").Unique(),
		edge.To("bill_links", FinanceInvoiceBill.Type),
		edge.To("lines", FinanceInvoiceLine.Type),
	}
}

func (FinanceInvoice) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "record_no").Unique(),
		index.Fields("organization_id", "idempotency_key").Unique(),
		index.Fields("organization_id", "status", "created_at"),
		index.Fields("settlement_party_id", "direction", "currency"),
		index.Fields("invoice_profile_id"),
		index.Fields("organization_id", "tax_invoice_no").Unique().StorageKey("financeinvoice_org_tax_invoice_no").Annotations(entsql.IndexWhere("tax_invoice_no IS NOT NULL")),
		index.Fields("organization_id", "red_invoice_no").Unique().StorageKey("financeinvoice_org_red_invoice_no").Annotations(entsql.IndexWhere("red_invoice_no IS NOT NULL")),
	}
}
