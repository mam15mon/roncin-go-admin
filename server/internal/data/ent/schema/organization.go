package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Organization represents a company, branch, or department in the tenant tree.
type Organization struct{ ent.Schema }

func (Organization) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (Organization) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").NotEmpty().MaxLen(64).Immutable(),
		field.String("name").NotEmpty().MaxLen(200),
		field.Enum("kind").Values("headquarters", "company", "department", "team").Immutable(),
		field.UUID("parent_id", uuid.Nil).Optional().Nillable(),
		field.Bool("enabled").Default(true),
		field.String("base_currency").Optional().Nillable().MinLen(3).MaxLen(3),
		searchKeywordsField(),
	}
}

func (Organization) Hooks() []ent.Hook { return []ent.Hook{searchKeywordsHook("name")} }

func (Organization) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("children", Organization.Type).From("parent").Field("parent_id").Unique(),
		edge.To("memberships", Membership.Type),
		edge.To("roles", Role.Type),
		edge.To("role_order_organization_accesses", RoleOrderOrganizationAccess.Type),
		edge.To("sessions", Session.Type),
		edge.To("partners", Partner.Type),
		edge.To("partner_assignments", PartnerAssignment.Type),
		edge.To("master_data_items", MasterDataItem.Type),
		edge.To("billing_units", BillingUnit.Type),
		edge.To("taxable_services", TaxableService.Type),
		edge.To("fee_settings", FeeSetting.Type),
		edge.To("ports", Port.Type),
		edge.To("airports", Airport.Type),
		edge.To("airlines", Airline.Type),
		edge.To("shipping_lines", ShippingLine.Type),
		edge.To("number_rules", NumberRule.Type),
		edge.To("orders", Order.Type),
		edge.To("order_consolidations", OrderConsolidation.Type),
		edge.To("order_personnel", OrderPersonnel.Type),
		edge.To("background_tasks", BackgroundTask.Type),
		edge.To("finance_bills", FinanceBill.Type),
		edge.To("finance_bill_batches", FinanceBillBatch.Type),
		edge.To("partner_invoice_profiles", PartnerInvoiceProfile.Type),
		edge.To("finance_invoices", FinanceInvoice.Type),
		edge.To("finance_cashflows", FinanceCashflow.Type),
		edge.To("finance_verifications", FinanceVerification.Type),
		edge.To("finance_commissions", FinanceCommission.Type),
		edge.To("finance_commission_lines", FinanceCommissionLine.Type),
		edge.To("finance_commission_adjustments", FinanceCommissionAdjustment.Type),
		edge.To("finance_commission_rules", FinanceCommissionRule.Type),
		edge.To("order_commission_attributions", OrderCommissionAttribution.Type),
		edge.To("finance_fee_ledger_preferences", FinanceFeeLedgerPreference.Type),
		edge.To("exchange_rate_custom_setting", ExchangeRateCustomSetting.Type),
		edge.To("finance_custom_setting", FinanceCustomSetting.Type),
		edge.To("enterprise_resources", EnterpriseResource.Type),
		edge.To("enterprise_tag_groups", EnterpriseTagGroup.Type),
	}
}

func (Organization) Indexes() []ent.Index {
	return []ent.Index{index.Fields("code").Unique(), index.Fields("parent_id")}
}
