package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// SeaTransportExecution 定义海运出口独立运输执行（实际航程事实）。
type SeaTransportExecution struct{ ent.Schema }

func (SeaTransportExecution) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (SeaTransportExecution) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.UUID("carrier_id", uuid.Nil).Optional().Nillable(),
		field.UUID("origin_location_id", uuid.Nil).Optional().Nillable(),
		field.UUID("discharge_location_id", uuid.Nil).Optional().Nillable(),
		field.UUID("transit_location_id", uuid.Nil).Optional().Nillable(),
		field.String("vessel_name").Default("").MaxLen(128),
		field.String("voyage_no").Default("").MaxLen(64),
		field.Time("etd").Optional().Nillable(),
		field.Time("eta").Optional().Nillable(),
		field.Uint64("version").Default(1),
	}
}

func (SeaTransportExecution) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("sea_transport_executions").Field("organization_id").Unique().Required(),
		edge.To("master_bills", SeaMasterBill.Type),
	}
}

func (SeaTransportExecution) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "carrier_id"),
		index.Fields("organization_id", "origin_location_id", "discharge_location_id"),
	}
}
