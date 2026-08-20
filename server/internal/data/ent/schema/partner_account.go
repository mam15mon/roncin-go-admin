package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// PartnerAccount 定义角色级结算账户，不把账户信息继续嵌入往来单位主表。
type PartnerAccount struct{ ent.Schema }

func (PartnerAccount) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (PartnerAccount) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("partner_role_id", uuid.Nil),
		field.Enum("account_type").Values("customer_settlement"),
		field.String("currency").NotEmpty().MaxLen(3),
		field.String("invoice_title").NotEmpty().MaxLen(200),
		field.String("unified_social_credit_code").Optional().Nillable().MaxLen(64),
		field.String("billing_address").Optional().MaxLen(500),
		field.String("billing_phone").Optional().MaxLen(64),
		field.String("bank_name").Optional().MaxLen(200),
		field.String("bank_account").Optional().MaxLen(100),
		field.String("swift_code").Optional().MaxLen(32),
		field.Bool("is_default").Default(false),
		field.Enum("status").Values("active", "inactive").Default("active"),
		field.String("remark").Optional().MaxLen(500),
	}
}

func (PartnerAccount) Edges() []ent.Edge {
	return []ent.Edge{edge.From("partner_role", PartnerRole.Type).Ref("accounts").Field("partner_role_id").Unique().Required()}
}

func (PartnerAccount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("partner_role_id", "account_type").
			Unique().
			StorageKey("partner_account_default_key").
			Annotations(entsql.IndexWhere("is_default")),
		index.Fields("partner_role_id", "status"),
		index.Fields("partner_role_id", "created_at"),
	}
}
