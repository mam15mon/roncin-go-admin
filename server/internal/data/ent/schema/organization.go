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
		field.Enum("kind").Values("headquarters", "company", "department", "team").Immutable(),
		field.UUID("parent_id", uuid.Nil).Optional().Nillable(),
		field.Bool("enabled").Default(true),
		field.String("base_currency").Optional().Nillable().MinLen(3).MaxLen(3),
	}
}

func (Organization) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("children", Organization.Type).From("parent").Field("parent_id").Unique(),
		edge.To("memberships", Membership.Type),
		edge.To("roles", Role.Type),
		edge.To("role_order_organization_accesses", RoleOrderOrganizationAccess.Type),
		edge.To("sessions", Session.Type),
		edge.To("partners", Partner.Type),
		edge.To("partner_assignments", PartnerAssignment.Type),
		edge.To("master_data_items", MasterDataItem.Type),
		edge.To("ports", Port.Type),
		edge.To("airports", Airport.Type),
		edge.To("airlines", Airline.Type),
		edge.To("shipping_lines", ShippingLine.Type),
		edge.To("number_rules", NumberRule.Type),
		edge.To("status_templates", StatusTemplate.Type),
		edge.To("milestone_templates", MilestoneTemplate.Type),
		edge.To("orders", Order.Type),
		edge.To("background_tasks", BackgroundTask.Type),
	}
}

func (Organization) Indexes() []ent.Index {
	return []ent.Index{index.Fields("code").Unique(), index.Fields("parent_id")}
}
