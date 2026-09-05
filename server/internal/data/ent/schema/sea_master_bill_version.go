package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// SeaMasterBillVersion 记录海运出口共享 MBL 的不可变快照版本。
type SeaMasterBillVersion struct{ ent.Schema }

func (SeaMasterBillVersion) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }

func (SeaMasterBillVersion) Fields() []ent.Field {
	fields := []ent.Field{
		field.Time("created_at").Immutable().Default(time.Now),
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("master_bill_id", uuid.Nil).Immutable(),
		field.Uint64("version_no").Immutable(),
		field.Uint64("source_entity_version").Immutable(),
		field.UUID("issuer_partner_id", uuid.Nil).Immutable(),
		field.UUID("transport_execution_id", uuid.Nil).Immutable(),
		field.String("master_no").NotEmpty().MaxLen(64).Immutable(),
		field.String("normalized_master_no").NotEmpty().MaxLen(64).Immutable(),
		field.Enum("status").Values("DRAFT", "CONFIRMED", "RELEASED", "VOIDED").Immutable(),
		field.String("vessel_voyage_snapshot").Optional().Nillable().MaxLen(100).Immutable(),
		field.String("etd_snapshot").Optional().Nillable().MaxLen(64).Immutable(),
		field.String("eta_snapshot").Optional().Nillable().MaxLen(64).Immutable(),
		field.UUID("carrier_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("origin_location_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("discharge_location_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("transit_location_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.String("vessel_name").Default("").MaxLen(128).Immutable(),
		field.String("voyage_no").Default("").MaxLen(64).Immutable(),
		field.Time("etd").Optional().Nillable().Immutable(),
		field.Time("eta").Optional().Nillable().Immutable(),
		field.String("content_hash").NotEmpty().MaxLen(64).Immutable(),
		field.Enum("source").Values("ORDER_LOCK", "AMENDMENT", "SWITCH", "VOID").Immutable(),
		field.String("reason").Optional().Nillable().MaxLen(500).Immutable(),
		field.UUID("created_by", uuid.Nil).Optional().Nillable().Immutable(),
		field.String("idempotency_key").Optional().Nillable().MaxLen(128).Immutable(),
		field.String("request_fingerprint").Optional().Nillable().MaxLen(128).Immutable(),
	}
	return append(fields, immutableSeaBillContentFields()...)
}

func (SeaMasterBillVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("sea_master_bill_versions").Field("organization_id").Unique().Required().Immutable(),
		edge.From("master_bill", SeaMasterBill.Type).Ref("versions").Field("master_bill_id").Unique().Required().Immutable(),
		edge.From("issuer_partner", Partner.Type).Ref("sea_master_bill_versions").Field("issuer_partner_id").Unique().Required().Immutable(),
		edge.From("transport_execution", SeaTransportExecution.Type).Ref("master_bill_versions").Field("transport_execution_id").Unique().Required().Immutable(),
		edge.From("creator", User.Type).Ref("created_sea_master_bill_versions").Field("created_by").Unique().Immutable(),
		edge.To("lock_records", OrderLockRecord.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
		edge.To("void_events", SeaDocumentVoidEvent.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
		edge.To("previous_void_events", SeaDocumentVoidEvent.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
	}
}

func (SeaMasterBillVersion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("master_bill_id", "version_no").Unique().StorageKey("sea_mbl_version_master_version_no"),
		index.Fields("master_bill_id", "source_entity_version", "content_hash").Unique().StorageKey("sea_mbl_version_source_hash"),
		index.Fields("organization_id", "master_bill_id"),
		index.Fields("organization_id", "idempotency_key").Unique().StorageKey("sea_mbl_version_idempotency_key"),
	}
}
