package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// PartnerAssignment 保存企业与内部人员、所属组织的角色指派。
type PartnerAssignment struct{ ent.Schema }

func (PartnerAssignment) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (PartnerAssignment) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("partner_id", uuid.Nil),
		field.UUID("user_id", uuid.Nil),
		field.UUID("organization_id", uuid.Nil),
		field.Enum("role").Values("CREATOR", "OPERATOR", "SALES", "CUSTOMER_SERVICE", "FINANCE", "COMMERCIAL", "INTERNAL_CONTACT", "DOCUMENT"),
		field.Int("sort_order").Default(0).NonNegative(),
	}
}

func (PartnerAssignment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("partner", Partner.Type).Ref("assignments").Field("partner_id").Unique().Required(),
		edge.From("user", User.Type).Ref("partner_assignments").Field("user_id").Unique().Required(),
		edge.From("organization", Organization.Type).Ref("partner_assignments").Field("organization_id").Unique().Required(),
	}
}

func (PartnerAssignment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("partner_id", "role", "sort_order").Unique(),
		index.Fields("user_id", "role"),
		index.Fields("organization_id", "role"),
	}
}
