package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Membership connects a user to an organization.
type Membership struct{ ent.Schema }

func (Membership) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (Membership) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("user_id", uuid.Nil),
		field.UUID("organization_id", uuid.Nil),
		field.Bool("primary").Default(false),
		field.Bool("enabled").Default(true),
	}
}

func (Membership) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("memberships").Field("user_id").Unique().Required(),
		edge.From("organization", Organization.Type).Ref("memberships").Field("organization_id").Unique().Required(),
		edge.To("role_assignments", RoleAssignment.Type),
	}
}

func (Membership) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "organization_id").Unique(),
		index.Fields("organization_id", "enabled"),
	}
}
