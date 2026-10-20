package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// AirMasterWaybillHistory 是不可变的主单内容及成员关系事件。
type AirMasterWaybillHistory struct{ ent.Schema }

func (AirMasterWaybillHistory) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }
func (AirMasterWaybillHistory) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil), field.UUID("master_waybill_id", uuid.Nil), field.Uint64("version"),
		field.String("action").MaxLen(40), field.String("reason").MaxLen(500), field.UUID("actor_id", uuid.Nil), field.Bytes("content"),
	}
}
func (AirMasterWaybillHistory) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("master_waybill", AirMasterWaybill.Type).Field("master_waybill_id").Unique().Required(),
	}
}
func (AirMasterWaybillHistory) Indexes() []ent.Index {
	return []ent.Index{index.Fields("master_waybill_id", "version").Unique()}
}
