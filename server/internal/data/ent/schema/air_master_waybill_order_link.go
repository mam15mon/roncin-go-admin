package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"time"
)

// AirMasterWaybillOrderLink 保留当前和历史关系；活动订单只能有一个主单。
type AirMasterWaybillOrderLink struct{ ent.Schema }

func (AirMasterWaybillOrderLink) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }
func (AirMasterWaybillOrderLink) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil), field.UUID("master_waybill_id", uuid.Nil), field.UUID("order_id", uuid.Nil),
		field.String("status").Default("ACTIVE").MaxLen(10), field.Uint64("version").Default(1), field.Time("started_at").Default(time.Now),
		field.Time("ended_at").Optional().Nillable(), field.String("ended_reason").Default("").MaxLen(500),
	}
}
func (AirMasterWaybillOrderLink) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("organization", Organization.Type).Field("organization_id").Unique().Required(),
		edge.To("master_waybill", AirMasterWaybill.Type).Field("master_waybill_id").Unique().Required(),
		edge.To("order", Order.Type).Field("order_id").Unique().Required(),
	}
}
func (AirMasterWaybillOrderLink) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "master_waybill_id"), index.Fields("order_id").Unique().StorageKey("idx_air_links_active_order").Annotations(entsql.IndexWhere("status = 'ACTIVE'")),
	}
}
