package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Permission is a globally registered capability from the backend manifest.
type Permission struct{ ent.Schema }

func (Permission) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (Permission) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").NotEmpty().MaxLen(160).Immutable(),
		field.String("name").NotEmpty().MaxLen(100),
		field.String("group").NotEmpty().MaxLen(100),
		field.String("description").MaxLen(500).Optional(),
	}
}

func (Permission) Edges() []ent.Edge {
	return []ent.Edge{edge.From("roles", Role.Type).Ref("permissions")}
}

func (Permission) Indexes() []ent.Index { return []ent.Index{index.Fields("key").Unique()} }
