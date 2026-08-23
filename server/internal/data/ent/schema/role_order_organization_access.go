package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// RoleOrderOrganizationAccess 定义角色可跨组织访问订单的范围。
type RoleOrderOrganizationAccess struct{ ent.Schema }

func (RoleOrderOrganizationAccess) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (RoleOrderOrganizationAccess) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("role_id", uuid.Nil),
		field.UUID("organization_id", uuid.Nil),
		field.Bool("writable").Default(false),
	}
}

func (RoleOrderOrganizationAccess) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("role", Role.Type).Ref("order_organization_accesses").Field("role_id").Unique().Required(),
		edge.From("organization", Organization.Type).Ref("role_order_organization_accesses").Field("organization_id").Unique().Required(),
	}
}

func (RoleOrderOrganizationAccess) Indexes() []ent.Index {
	return []ent.Index{index.Fields("role_id", "organization_id").Unique(), index.Fields("organization_id")}
}
