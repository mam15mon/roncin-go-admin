package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// PartnerRole 定义往来单位业务角色。
type PartnerRole struct{ ent.Schema }

func (PartnerRole) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (PartnerRole) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("partner_id", uuid.Nil),
		field.Enum("role_type").Values("customer", "supplier", "agent", "carrier"),
		field.Bool("enabled").Default(true),
		field.Bool("blacklisted").Default(false),
		field.String("blacklist_reason").Optional().MaxLen(500),
		field.Time("blacklisted_at").Optional().Nillable(),
		field.UUID("blacklisted_by", uuid.Nil).Optional().Nillable(),
	}
}

func (PartnerRole) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("partner", Partner.Type).Ref("roles").Field("partner_id").Unique().Required(),
		edge.To("accounts", PartnerAccount.Type),
		edge.To("settlement_rules", PartnerSettlementRule.Type),
	}
}

func (PartnerRole) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("partner_id", "role_type").Unique().StorageKey("partner_role_type_key"),
		index.Fields("role_type", "enabled"),
		index.Fields("partner_id", "blacklisted"),
	}
}
