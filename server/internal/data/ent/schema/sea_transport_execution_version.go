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

// SeaTransportExecutionVersion 记录海运出口运输执行（实际航程）的不可变快照版本。
type SeaTransportExecutionVersion struct{ ent.Schema }

func (SeaTransportExecutionVersion) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }

func (SeaTransportExecutionVersion) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Immutable().Default(time.Now),
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("transport_execution_id", uuid.Nil).Immutable(),
		field.Uint64("version_no").Immutable(),
		field.Uint64("source_entity_version").Immutable(),
		field.UUID("shipping_line_id", uuid.Nil).Immutable(),
		field.UUID("origin_location_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("discharge_location_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("transit_location_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.String("vessel_name").Default("").MaxLen(128).Immutable(),
		field.String("voyage_no").Default("").MaxLen(64).Immutable(),
		field.Time("etd").Optional().Nillable().Immutable(),
		field.Time("eta").Optional().Nillable().Immutable(),
		field.String("content_hash").NotEmpty().MaxLen(64).Immutable(),
		field.Enum("source").Values("ORDER_LOCK", "SHARED_UPDATE", "REASSIGNMENT").Immutable(),
		field.String("reason").Optional().Nillable().MaxLen(500).Immutable(),
		field.UUID("created_by", uuid.Nil).Optional().Nillable().Immutable(),
		field.String("idempotency_key").Optional().Nillable().MaxLen(128).Immutable(),
		field.String("request_fingerprint").Optional().Nillable().MaxLen(128).Immutable(),
		field.String("confirmed_by_party").Optional().Nillable().MaxLen(128).Immutable(),
		field.Time("confirmed_at").Optional().Nillable().Immutable(),
		field.String("confirmation_note").Optional().Nillable().MaxLen(500).Immutable(),
		field.UUID("confirmation_attachment_id", uuid.Nil).Optional().Nillable().Immutable(),
	}
}

func (SeaTransportExecutionVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("sea_transport_execution_versions").Field("organization_id").Unique().Required().Immutable(),
		edge.From("transport_execution", SeaTransportExecution.Type).Ref("versions").Field("transport_execution_id").Unique().Required().Immutable(),
		edge.From("shipping_line", ShippingLine.Type).Ref("sea_transport_execution_versions").Field("shipping_line_id").Unique().Required().Immutable(),
		edge.From("creator", User.Type).Ref("created_sea_transport_execution_versions").Field("created_by").Unique().Immutable(),
		edge.From("confirmation_attachment", OrderAttachment.Type).Ref("sea_transport_execution_versions").Field("confirmation_attachment_id").Unique().Immutable(),
		edge.To("lock_records", OrderLockRecord.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
	}
}

func (SeaTransportExecutionVersion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("transport_execution_id", "version_no").Unique().StorageKey("sea_te_version_transport_version_no"),
		index.Fields("transport_execution_id", "source_entity_version", "content_hash").Unique().StorageKey("sea_te_version_source_hash"),
		index.Fields("organization_id", "transport_execution_id"),
		index.Fields("organization_id", "idempotency_key").Unique().StorageKey("sea_te_version_idempotency_key"),
	}
}
