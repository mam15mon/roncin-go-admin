package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// PartnerSettlementRule 定义角色级结算与信用规则。
type PartnerSettlementRule struct{ ent.Schema }

func (PartnerSettlementRule) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (PartnerSettlementRule) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("partner_role_id", uuid.Nil),
		field.Enum("statement_mode").Values("single", "multi"),
		field.Enum("settlement_method").Values("by_ticket", "monthly", "weekly", "semi_monthly", "bi_monthly", "quarterly", "days_45", "prepaid"),
		field.Int("settlement_day").Optional().Nillable(),
		field.Int("settlement_cycle_days").Optional().Nillable(),
		field.Enum("settlement_base").Values("bill_date", "sailing_date", "arrival_date").Optional().Nillable(),
		field.String("settlement_currency").NotEmpty().MinLen(3).MaxLen(3),
		field.Int64("credit_limit_minor").Optional().Nillable().Min(0),
		field.String("credit_currency").Optional().Nillable().MinLen(3).MaxLen(3),
		field.Bool("is_active").Default(true),
	}
}

func (PartnerSettlementRule) Edges() []ent.Edge {
	return []ent.Edge{edge.From("partner_role", PartnerRole.Type).Ref("settlement_rules").Field("partner_role_id").Unique().Required()}
}

func (PartnerSettlementRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("partner_role_id", "statement_mode", "settlement_method").Unique().StorageKey("partner_settlement_rule_key"),
		index.Fields("partner_role_id", "is_active"),
	}
}
