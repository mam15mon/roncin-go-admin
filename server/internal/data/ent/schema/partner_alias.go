package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// PartnerAlias 定义往来单位别名。
type PartnerAlias struct{ ent.Schema }

func (PartnerAlias) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (PartnerAlias) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("partner_id", uuid.Nil),
		field.String("alias_name").NotEmpty().MaxLen(200),
		field.String("normalized_alias_name").NotEmpty().MaxLen(200),
		field.Int("sort_order").Default(0).NonNegative(),
	}
}

func (PartnerAlias) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("partner", Partner.Type).Ref("aliases").Field("partner_id").Unique().Required(),
	}
}

func (PartnerAlias) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("partner_id", "normalized_alias_name").Unique().StorageKey("partner_alias_name_key"),
		index.Fields("partner_id", "sort_order"),
	}
}
