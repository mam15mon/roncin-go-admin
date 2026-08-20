package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// RoleAssignment assigns an organization role to a membership.
type RoleAssignment struct{ ent.Schema }

func (RoleAssignment) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (RoleAssignment) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("membership_id", uuid.Nil),
		field.UUID("role_id", uuid.Nil),
	}
}

func (RoleAssignment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("membership", Membership.Type).Ref("role_assignments").Field("membership_id").Unique().Required(),
		edge.From("role", Role.Type).Ref("assignments").Field("role_id").Unique().Required(),
	}
}

func (RoleAssignment) Indexes() []ent.Index {
	return []ent.Index{index.Fields("membership_id", "role_id").Unique(), index.Fields("role_id")}
}
