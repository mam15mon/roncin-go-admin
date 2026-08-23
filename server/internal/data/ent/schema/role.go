package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Role groups permissions for one organization and defines its data scope.
type Role struct{ ent.Schema }

func (Role) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (Role) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.String("code").NotEmpty().MaxLen(64).Immutable(),
		field.String("name").NotEmpty().MaxLen(100),
		field.Enum("data_scope").Values("all", "organization", "organization_tree", "self").Default("organization"),
		field.Bool("enabled").Default(true),
	}
}

func (Role) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("roles").Field("organization_id").Unique().Required(),
		edge.To("permissions", Permission.Type),
		edge.To("assignments", RoleAssignment.Type),
		edge.To("order_organization_accesses", RoleOrderOrganizationAccess.Type),
	}
}

func (Role) Indexes() []ent.Index {
	return []ent.Index{index.Fields("organization_id", "code").Unique(), index.Fields("organization_id", "enabled")}
}
