package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Partner is a customer, supplier, or a business party serving both roles.
type Partner struct{ ent.Schema }

func (Partner) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (Partner) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.String("code").NotEmpty().MaxLen(64).Immutable(),
		field.String("name").NotEmpty().MaxLen(200),
		field.Enum("type").Values("customer", "supplier", "both"),
		field.String("contact_name").MaxLen(100).Optional(),
		field.String("phone").MaxLen(64).Optional(),
		field.String("email").MaxLen(254).Optional(),
		field.String("address").MaxLen(500).Optional(),
		field.Bool("enabled").Default(true),
	}
}

func (Partner) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("partners").Field("organization_id").Unique().Required(),
	}
}

func (Partner) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "code").Unique(),
		index.Fields("organization_id", "enabled"),
		index.Fields("organization_id", "type"),
	}
}
