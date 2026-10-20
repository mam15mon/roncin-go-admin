package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"time"
)

// AirHouseWaybillVersion 保存不可变本票单证版本，签发后修改必须留原因。
type AirHouseWaybillVersion struct{ ent.Schema }

func (AirHouseWaybillVersion) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }
func (AirHouseWaybillVersion) Fields() []ent.Field {
	return []ent.Field{field.Time("created_at").Default(time.Now).Immutable(), field.UUID("organization_id", uuid.Nil).Immutable(), field.UUID("order_id", uuid.Nil).Immutable(), field.UUID("house_waybill_id", uuid.Nil).Immutable(), field.Uint64("version").Immutable(), field.String("house_no").MaxLen(64).Immutable(), field.String("status").MaxLen(20).Immutable(), field.String("content").MaxLen(4000).Immutable(), field.UUID("actor_id", uuid.Nil).Immutable(), field.String("reason").MaxLen(500).Immutable()}
}
func (AirHouseWaybillVersion) Edges() []ent.Edge {
	return []ent.Edge{edge.To("house_waybill", AirHouseWaybill.Type).Field("house_waybill_id").Required().Unique().Immutable()}
}
func (AirHouseWaybillVersion) Indexes() []ent.Index {
	return []ent.Index{index.Fields("house_waybill_id", "version").Unique(), index.Fields("organization_id", "order_id")}
}
