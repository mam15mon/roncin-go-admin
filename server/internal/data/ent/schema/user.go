package schema

import (
	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// User 存储系统用户及其可选登录身份，密码哈希不会离开数据层。
type User struct{ ent.Schema }

func (User) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("username").MaxLen(100).Optional().Immutable(),
		field.String("display_name").NotEmpty().MaxLen(100),
		field.String("email").MaxLen(254).Optional().Nillable(),
		field.String("avatar_url").MaxLen(2048).Optional().Nillable(),
		field.String("password_hash").Optional().Nillable().Sensitive(),
		field.String("wecom_userid").MaxLen(64).Optional().Nillable(),
		field.String("wecom_name").MaxLen(100).Optional().Nillable(),
		field.String("dingtalk_unionid").MaxLen(128).Optional().Nillable(),
		field.String("dingtalk_name").MaxLen(100).Optional().Nillable(),
		field.Bool("enabled").Default(true),
		searchKeywordsField(),
	}
}

func (User) Hooks() []ent.Hook {
	return []ent.Hook{searchKeywordsHook("display_name", "wecom_name", "dingtalk_name")}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("memberships", Membership.Type),
		edge.To("sessions", Session.Type),
		edge.To("order_personnel", OrderPersonnel.Type),
		edge.To("partner_assignments", PartnerAssignment.Type),
		edge.To("cancelled_order_fees", OrderFee.Type),
		edge.To("confirmed_finance_bills", FinanceBill.Type),
		edge.To("cancelled_finance_bills", FinanceBill.Type),
		edge.To("created_finance_bill_batches", FinanceBillBatch.Type),
		edge.To("issued_finance_invoices", FinanceInvoice.Type),
		edge.To("cancelled_finance_invoices", FinanceInvoice.Type),
		edge.To("red_flushed_finance_invoices", FinanceInvoice.Type),
		edge.To("confirmed_finance_cashflows", FinanceCashflow.Type),
		edge.To("cancelled_finance_cashflows", FinanceCashflow.Type),
		edge.To("reversed_finance_verifications", FinanceVerification.Type),
		edge.To("finance_commissions", FinanceCommission.Type),
		edge.To("confirmed_finance_commissions", FinanceCommission.Type),
		edge.To("paid_finance_commissions", FinanceCommission.Type),
		edge.To("cancelled_finance_commissions", FinanceCommission.Type),
		edge.To("finance_commission_adjustments", FinanceCommissionAdjustment.Type),
		edge.To("order_commission_attributions", OrderCommissionAttribution.Type),
		edge.To("confirmed_finance_commission_adjustments", FinanceCommissionAdjustment.Type),
		edge.To("paid_finance_commission_adjustments", FinanceCommissionAdjustment.Type),
		edge.To("cancelled_finance_commission_adjustments", FinanceCommissionAdjustment.Type),
		edge.To("finance_fee_ledger_preferences", FinanceFeeLedgerPreference.Type),
		edge.To("updated_exchange_rate_custom_settings", ExchangeRateCustomSetting.Type),
		edge.To("updated_finance_custom_settings", FinanceCustomSetting.Type),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("username").Unique().Annotations(entsql.IndexWhere("username IS NOT NULL AND username <> ''")),
		index.Fields("email"),
		index.Fields("wecom_userid").Unique(),
		index.Fields("dingtalk_unionid").Unique(),
	}
}
