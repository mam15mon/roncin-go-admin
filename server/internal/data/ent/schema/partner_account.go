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

// PartnerAccount 定义往来单位级结算账户。账户用途独立于往来单位的业务角色。
type PartnerAccount struct{ ent.Schema }

func (PartnerAccount) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (PartnerAccount) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("partner_id", uuid.Nil),
		field.String("name").NotEmpty().MaxLen(200),
		field.String("account_holder").NotEmpty().MaxLen(200),
		field.String("currency").NotEmpty().MaxLen(3),
		field.String("bank_name").NotEmpty().MaxLen(200),
		field.String("account_no").NotEmpty().MaxLen(100),
		field.String("swift_code").Optional().MaxLen(32),
		field.Enum("usage").Values("RECEIVABLE", "PAYABLE", "BOTH"),
		field.Bool("is_default_receivable").Default(false),
		field.Bool("is_default_payable").Default(false),
		field.Bool("enabled").Default(true),
		field.String("remark").Optional().MaxLen(500),
	}
}

func (PartnerAccount) Edges() []ent.Edge {
	return []ent.Edge{edge.From("partner", Partner.Type).Ref("accounts").Field("partner_id").Unique().Required()}
}

func (PartnerAccount) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Checks(map[string]string{
		"partner_accounts_default_usage_check": "((NOT is_default_receivable OR (enabled AND usage IN ('RECEIVABLE', 'BOTH'))) AND (NOT is_default_payable OR (enabled AND usage IN ('PAYABLE', 'BOTH'))))",
	})}
}

func (PartnerAccount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("partner_id", "currency").
			Unique().
			StorageKey("partner_account_default_receivable_key").
			Annotations(entsql.IndexWhere("is_default_receivable")),
		index.Fields("partner_id", "currency").
			Unique().
			StorageKey("partner_account_default_payable_key").
			Annotations(entsql.IndexWhere("is_default_payable")),
		index.Fields("partner_id", "enabled", "currency"),
		index.Fields("partner_id", "created_at"),
	}
}
