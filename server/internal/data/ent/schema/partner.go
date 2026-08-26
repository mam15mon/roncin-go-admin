package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Partner 是组织内往来单位法人档案。
type Partner struct{ ent.Schema }

func (Partner) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (Partner) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.String("code").NotEmpty().MaxLen(64).Immutable(),
		field.String("legal_name").NotEmpty().MaxLen(200),
		field.String("normalized_name").NotEmpty().MaxLen(200),
		field.String("unified_social_credit_code").Optional().Nillable().MaxLen(64),
		field.String("registered_address").Optional().MaxLen(500),
		field.Bool("enabled").Default(true),
	}
}

func (Partner) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("partners").Field("organization_id").Unique().Required(),
		edge.To("roles", PartnerRole.Type),
		edge.To("contacts", PartnerContact.Type),
		edge.To("aliases", PartnerAlias.Type),
		edge.To("profile", PartnerProfile.Type).Unique(),
		edge.To("assignments", PartnerAssignment.Type),
		edge.To("shipping_presets", PartnerShippingPreset.Type),
		edge.To("contracts", PartnerContract.Type),
		edge.To("attachments", PartnerAttachment.Type),
		edge.To("orders", Order.Type),
		edge.To("order_fees", OrderFee.Type),
		edge.To("finance_bills", FinanceBill.Type),
		edge.To("finance_invoices", FinanceInvoice.Type),
		edge.To("finance_cashflows", FinanceCashflow.Type),
		edge.To("finance_verifications", FinanceVerification.Type),
	}
}

func (Partner) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "code").Unique().StorageKey("partner_org_code_key"),
		index.Fields("organization_id", "normalized_name").Unique().StorageKey("partner_org_name_key"),
		index.Fields("organization_id", "unified_social_credit_code").Unique().StorageKey("partner_org_uscc_key"),
		index.Fields("organization_id", "enabled"),
	}
}
