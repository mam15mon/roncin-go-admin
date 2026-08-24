package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderPersonnel 定义订单协作人员。
type OrderPersonnel struct{ ent.Schema }

func (OrderPersonnel) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderPersonnel) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("order_id", uuid.Nil),
		field.UUID("user_id", uuid.Nil),
		field.UUID("organization_id", uuid.Nil),
		field.Enum("role").Values("CREATOR", "OPERATOR", "SALES", "CUSTOMER_SERVICE", "DOCUMENT", "COMMERCIAL", "ASSOCIATE", "ASSOCIATE2"),
		field.Time("assigned_at").Default(time.Now),
	}
}

func (OrderPersonnel) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).Ref("personnel").Field("order_id").Unique().Required(),
		edge.From("user", User.Type).Ref("order_personnel").Field("user_id").Unique().Required(),
		edge.From("organization", Organization.Type).Ref("order_personnel").Field("organization_id").Unique().Required(),
	}
}

func (OrderPersonnel) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "role").Unique(),
		index.Fields("order_id"),
		index.Fields("user_id", "role"),
		index.Fields("organization_id", "role"),
	}
}
