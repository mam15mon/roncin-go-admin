package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// User stores a local account. Password hashes never leave the data layer.
type User struct{ ent.Schema }

func (User) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("username").NotEmpty().MaxLen(100).Immutable(),
		field.String("display_name").NotEmpty().MaxLen(100),
		field.String("email").MaxLen(254).Optional().Nillable(),
		field.String("password_hash").NotEmpty().Sensitive(),
		field.Bool("enabled").Default(true),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("memberships", Membership.Type),
		edge.To("sessions", Session.Type),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{index.Fields("username").Unique(), index.Fields("email")}
}
