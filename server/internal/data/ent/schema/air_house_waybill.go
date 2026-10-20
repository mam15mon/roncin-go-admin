package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// AirHouseWaybill 的号码在主单内永久占用，跨主单允许重号。
type AirHouseWaybill struct{ ent.Schema }

func (AirHouseWaybill) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }
func (AirHouseWaybill) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil), field.UUID("master_waybill_id", uuid.Nil), field.UUID("order_id", uuid.Nil),
		field.String("house_no").NotEmpty().MaxLen(64), field.String("status").Default("DRAFT").MaxLen(20), field.String("content").Default("").MaxLen(4000), field.Uint64("version").Default(1),
	}
}
func (AirHouseWaybill) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("organization", Organization.Type).Field("organization_id").Unique().Required(),
		edge.To("master_waybill", AirMasterWaybill.Type).Field("master_waybill_id").Unique().Required(),
		edge.To("order", Order.Type).Field("order_id").Unique().Required(),
	}
}
func (AirHouseWaybill) Indexes() []ent.Index {
	return []ent.Index{index.Fields("master_waybill_id", "house_no").Unique(), index.Fields("order_id").Unique()}
}
