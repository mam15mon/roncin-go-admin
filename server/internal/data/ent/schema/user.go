package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// User stores a local account. Password hashes never leave the data layer.
type User struct{ ent.Schema }

func (User) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("username").NotEmpty().MaxLen(100).Immutable(),
		field.String("display_name").NotEmpty().MaxLen(100),
		field.String("email").MaxLen(254).Optional().Nillable(),
		field.String("avatar_url").MaxLen(2048).Optional().Nillable(),
		field.String("password_hash").Optional().Nillable().Sensitive(),
		field.String("wecom_userid").MaxLen(64).Optional().Nillable(),
		field.String("wecom_name").MaxLen(100).Optional().Nillable(),
		field.String("dingtalk_unionid").MaxLen(128).Optional().Nillable(),
		field.String("dingtalk_name").MaxLen(100).Optional().Nillable(),
		field.Bool("enabled").Default(true),
	}
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
		edge.To("issued_finance_invoices", FinanceInvoice.Type),
		edge.To("cancelled_finance_invoices", FinanceInvoice.Type),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{index.Fields("username").Unique(), index.Fields("email"), index.Fields("wecom_userid").Unique(), index.Fields("dingtalk_unionid").Unique()}
}
