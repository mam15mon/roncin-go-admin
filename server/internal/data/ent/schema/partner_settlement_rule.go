package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// PartnerSettlementRule 定义客户角色唯一的收款期限与本币信用额度。
type PartnerSettlementRule struct{ ent.Schema }

func (PartnerSettlementRule) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (PartnerSettlementRule) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("partner_role_id", uuid.Nil),
		field.Int64("credit_limit_minor").Optional().Nillable().Min(0),
		field.String("credit_currency").Optional().Nillable().MinLen(3).MaxLen(3),
		// NULL 表示未约定，0 表示现结，正数表示建账后 N 天。
		field.Int("payment_terms_days").Optional().Nillable().Min(0).Max(3650).Comment("收款期限天数"),
	}
}

func (PartnerSettlementRule) Edges() []ent.Edge {
	return []ent.Edge{edge.From("partner_role", PartnerRole.Type).Ref("settlement_rules").Field("partner_role_id").Unique().Required()}
}

func (PartnerSettlementRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("partner_role_id").Unique().StorageKey("partner_settlement_rule_role_key"),
	}
}
