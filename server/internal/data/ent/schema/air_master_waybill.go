package schema

import (
	"encoding/json"
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// AirMasterWaybill 是同公司内显式共享的航空主运单。
type AirMasterWaybill struct{ ent.Schema }

func (AirMasterWaybill) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }
func (AirMasterWaybill) Fields() []ent.Field {
	return []ent.Field{
		field.JSON("waybill_content", json.RawMessage{}).Default(json.RawMessage(`{}`)), field.String("content_hash").MaxLen(64).Default("44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"),
		field.String("business_type").NotEmpty().MaxLen(2).Immutable(),
		field.UUID("organization_id", uuid.Nil), field.String("master_no").MaxLen(11).NotEmpty(),
		field.UUID("airline_id", uuid.Nil), field.UUID("origin_airport_id", uuid.Nil), field.UUID("destination_airport_id", uuid.Nil),
		field.String("flight_no").Default("").MaxLen(40), field.String("etd").Default("").MaxLen(64), field.String("eta").Default("").MaxLen(64),
		field.String("actual_departure_at").Default("").MaxLen(64), field.String("actual_arrival_at").Default("").MaxLen(64),
		field.String("document_cutoff_at").Default("").MaxLen(64), field.String("customs_cutoff_at").Default("").MaxLen(64),
		field.String("status").Default("DRAFT").MaxLen(20), field.Uint64("version").Default(1),
		field.Int("declared_packages").Optional().Nillable(), field.Float("declared_gross_weight_kg").Optional().Nillable(),
		field.Float("declared_volume_cbm").Optional().Nillable(), field.String("notes").Default("").MaxRuneLen(1000),
	}
}
func (AirMasterWaybill) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("organization", Organization.Type).Field("organization_id").Unique().Required(),
		edge.To("airline", Airline.Type).Field("airline_id").Unique().Required(),
		edge.To("origin_airport", Airport.Type).Field("origin_airport_id").Unique().Required(),
		edge.To("destination_airport", Airport.Type).Field("destination_airport_id").Unique().Required(),
	}
}
func (AirMasterWaybill) Indexes() []ent.Index {
	return []ent.Index{index.Fields("organization_id", "business_type", "master_no").Unique().StorageKey("air_master_waybill_business_no"), index.Fields("organization_id", "business_type")}
}

func (AirMasterWaybill) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Checks(map[string]string{"air_master_waybill_business_type_check": "business_type IN ('AE','AI')"})}
}
