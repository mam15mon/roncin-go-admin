package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// RoleOrganizationAccess 定义角色可访问组织的范围。
type RoleOrganizationAccess struct{ ent.Schema }

func (RoleOrganizationAccess) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (RoleOrganizationAccess) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("role_id", uuid.Nil),
		field.UUID("organization_id", uuid.Nil),
		field.Bool("writable").Default(false),
	}
}

func (RoleOrganizationAccess) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("role", Role.Type).Ref("organization_accesses").Field("role_id").Unique().Required(),
		edge.From("organization", Organization.Type).Ref("role_organization_accesses").Field("organization_id").Unique().Required(),
	}
}

func (RoleOrganizationAccess) Indexes() []ent.Index {
	return []ent.Index{index.Fields("role_id", "organization_id").Unique(), index.Fields("organization_id")}
}
