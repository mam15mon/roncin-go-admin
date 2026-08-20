package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Organization represents a company, branch, or department in the tenant tree.
type Organization struct{ ent.Schema }

func (Organization) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (Organization) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").NotEmpty().MaxLen(64).Immutable(),
		field.String("name").NotEmpty().MaxLen(200),
		field.UUID("parent_id", uuid.Nil).Optional().Nillable(),
		field.Bool("enabled").Default(true),
	}
}

func (Organization) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("children", Organization.Type).From("parent").Field("parent_id").Unique(),
		edge.To("memberships", Membership.Type),
		edge.To("roles", Role.Type),
		edge.To("sessions", Session.Type),
		edge.To("partners", Partner.Type),
		edge.To("master_data_items", MasterDataItem.Type),
	}
}

func (Organization) Indexes() []ent.Index {
	return []ent.Index{index.Fields("code").Unique(), index.Fields("parent_id")}
}
