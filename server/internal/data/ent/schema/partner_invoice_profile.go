package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// PartnerInvoiceProfile 保存往来单位当前有效的税务开票资料；历史发票仍使用自身快照。
type PartnerInvoiceProfile struct{ ent.Schema }

func (PartnerInvoiceProfile) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (PartnerInvoiceProfile) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("partner_id", uuid.Nil).Immutable(),
		field.String("invoice_title").NotEmpty().MaxLen(200),
		field.String("taxpayer_identification_no").NotEmpty().MaxLen(64),
		field.String("registered_address").Optional().MaxLen(500),
		field.String("registered_phone").Optional().MaxLen(50),
		field.String("bank_name").Optional().MaxLen(200),
		field.String("bank_account").Optional().MaxLen(100),
		field.Enum("default_invoice_type").Values("NORMAL", "SPECIAL").Default("NORMAL"),
		field.Uint64("version").Default(1),
	}
}

func (PartnerInvoiceProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("partner_invoice_profiles").Field("organization_id").Unique().Required().Immutable(),
		edge.From("partner", Partner.Type).Ref("invoice_profile").Field("partner_id").Unique().Required().Immutable(),
		edge.To("finance_invoices", FinanceInvoice.Type),
	}
}

func (PartnerInvoiceProfile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "partner_id").Unique(),
		index.Fields("organization_id", "taxpayer_identification_no"),
	}
}
