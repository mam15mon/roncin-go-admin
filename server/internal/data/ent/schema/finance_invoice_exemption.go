package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceInvoiceExemption 保留每次账单免票决定及撤销事实，不删除历史决定。
type FinanceInvoiceExemption struct{ ent.Schema }

func (FinanceInvoiceExemption) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceInvoiceExemption) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("bill_id", uuid.Nil).Immutable(),
		field.UUID("marked_by", uuid.Nil).Immutable(),
		field.String("reason").NotEmpty().MaxRuneLen(500).Immutable(),
		field.Uint64("version").Default(1),
		field.Time("revoked_at").Optional().Nillable(),
		field.UUID("revoked_by", uuid.Nil).Optional().Nillable(),
		field.String("revocation_reason").Optional().Nillable().MaxRuneLen(500),
	}
}

func (FinanceInvoiceExemption) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("bill", FinanceBill.Type).Ref("invoice_exemptions").Field("bill_id").Unique().Required().Immutable(),
		edge.From("organization", Organization.Type).Ref("finance_invoice_exemptions").Field("organization_id").Unique().Required().Immutable(),
	}
}

func (FinanceInvoiceExemption) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("bill_id").Unique().StorageKey("finance_invoice_exemption_active_bill").Annotations(entsql.IndexWhere("revoked_at IS NULL")),
		index.Fields("organization_id", "bill_id", "created_at"),
	}
}

func (FinanceInvoiceExemption) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Checks(map[string]string{
		"finance_invoice_exemption_revoke_check": "(revoked_at IS NULL AND revoked_by IS NULL AND revocation_reason IS NULL) OR (revoked_at IS NOT NULL AND revoked_by IS NOT NULL AND revocation_reason IS NOT NULL AND length(trim(revocation_reason)) > 0)",
		"finance_invoice_exemption_reason_check": "length(trim(reason)) > 0",
	})}
}
