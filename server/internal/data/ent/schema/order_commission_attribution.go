package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderCommissionAttribution 固化订单创建时客户档案中的提成人员归属。
type OrderCommissionAttribution struct{ ent.Schema }

func (OrderCommissionAttribution) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderCommissionAttribution) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("order_id", uuid.Nil).Immutable(),
		field.UUID("customer_id", uuid.Nil).Immutable(),
		field.UUID("source_assignment_id", uuid.Nil).Immutable(),
		field.UUID("employee_id", uuid.Nil).Immutable(),
		field.String("employee_name").NotEmpty().MaxLen(100).Immutable(),
		field.Enum("personnel_role").Values("SALES", "OPERATOR", "CUSTOMER_SERVICE").Immutable(),
		field.Time("attributed_at").Immutable(),
	}
}

func (OrderCommissionAttribution) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("order_commission_attributions").Field("organization_id").Unique().Required().Immutable(),
		edge.From("order", Order.Type).Ref("commission_attributions").Field("order_id").Unique().Required().Immutable(),
		edge.From("customer", Partner.Type).Ref("order_commission_attributions").Field("customer_id").Unique().Required().Immutable(),
		edge.From("employee", User.Type).Ref("order_commission_attributions").Field("employee_id").Unique().Required().Immutable(),
	}
}

func (OrderCommissionAttribution) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "employee_id", "personnel_role").Unique(),
		index.Fields("organization_id", "employee_id", "personnel_role"),
		index.Fields("customer_id", "personnel_role"),
	}
}
