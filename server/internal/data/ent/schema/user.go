package schema

import (
	"fmt"
	"regexp"
	"unicode/utf8"

	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

var usernamePattern = regexp.MustCompile(`^[a-z0-9_.-]+$`)

func validateUsername(value string) error {
	length := utf8.RuneCountInString(value)
	if length < 3 || length > 64 || !usernamePattern.MatchString(value) {
		return fmt.Errorf("用户名必须为 3 至 64 位小写字母、数字、点号、下划线或连字符")
	}
	return nil
}

// User 存储系统用户及其可选登录身份，密码哈希不会离开数据层。
type User struct{ ent.Schema }

func (User) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("username").MaxLen(100).Optional().Validate(validateUsername),
		field.String("display_name").NotEmpty().MaxLen(100),
		field.String("email").MaxLen(254).Optional().Nillable(),
		field.String("avatar_url").MaxLen(2048).Optional().Nillable(),
		field.String("password_hash").Optional().Nillable().Sensitive(),
		field.String("wecom_userid").MaxLen(64).Optional().Nillable(),
		field.String("wecom_name").MaxLen(100).Optional().Nillable(),
		field.String("dingtalk_unionid").MaxLen(128).Optional().Nillable(),
		field.String("dingtalk_userid").MaxLen(64).Optional().Nillable(),
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
		edge.To("notification_deliveries", NotificationDelivery.Type),
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
		edge.To("created_enterprise_resources", EnterpriseResource.Type),
		edge.To("updated_enterprise_resources", EnterpriseResource.Type),
		edge.To("uploaded_enterprise_resource_images", EnterpriseResourceImage.Type),
		edge.To("enterprise_resource_assignments", EnterpriseResourceAssignee.Type),
		edge.To("confirmed_sea_cargo_allocation_links", SeaMasterBillOrderLink.Type),
		edge.To("created_sea_order_split_events", SeaOrderSplitEvent.Type),
		edge.To("created_sea_order_reassignment_events", SeaOrderReassignmentEvent.Type),
		edge.To("uploaded_attachment_assets", OrderAttachmentAsset.Type),
		edge.To("created_order_attachments", OrderAttachment.Type),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("username").Unique().Annotations(entsql.IndexWhere("username IS NOT NULL AND username <> ''")),
		index.Fields("email"),
		index.Fields("wecom_userid").Unique(),
		index.Fields("dingtalk_unionid").Unique(),
		index.Fields("dingtalk_userid").Unique(),
	}
}
