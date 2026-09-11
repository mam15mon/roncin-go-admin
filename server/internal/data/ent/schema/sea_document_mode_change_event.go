package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// SeaDocumentModeChangeEvent 记录海运单证模式切换（HOUSE <-> DIRECT）的不可变事件。
type SeaDocumentModeChangeEvent struct{ ent.Schema }

func (SeaDocumentModeChangeEvent) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }

func (SeaDocumentModeChangeEvent) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Checks(map[string]string{
			"sea_document_mode_change_events_mode_check": "previous_mode <> target_mode AND ((previous_mode = 'HOUSE' AND target_mode = 'DIRECT' AND previous_house_bill_id IS NOT NULL AND previous_house_bill_version_id IS NOT NULL AND target_house_bill_id IS NULL AND target_house_bill_version_id IS NULL) OR (previous_mode = 'DIRECT' AND target_mode = 'HOUSE' AND target_house_bill_id IS NOT NULL AND target_house_bill_version_id IS NOT NULL AND previous_house_bill_id IS NULL AND previous_house_bill_version_id IS NULL))",
		}),
	}
}

func (SeaDocumentModeChangeEvent) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Immutable().Default(time.Now),
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("order_id", uuid.Nil).Immutable(),
		field.Enum("previous_mode").Values("HOUSE", "DIRECT").Immutable(),
		field.Enum("target_mode").Values("HOUSE", "DIRECT").Immutable(),
		field.UUID("previous_house_bill_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("previous_house_bill_version_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("target_house_bill_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("target_house_bill_version_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.String("reason").NotEmpty().MaxLen(500).Immutable(),
		field.String("impact_summary").Optional().Nillable().MaxLen(1000).Immutable(),
		field.String("confirmed_by_party").NotEmpty().MaxLen(128).Immutable(),
		field.Time("confirmed_at").Immutable(),
		field.String("confirmation_note").NotEmpty().MaxLen(500).Immutable(),
		field.UUID("confirmation_attachment_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("created_by", uuid.Nil).Immutable(),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.String("request_fingerprint").NotEmpty().MaxLen(128).Immutable(),
	}
}

func (SeaDocumentModeChangeEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("sea_document_mode_change_events").Field("organization_id").Unique().Required().Immutable(),
		edge.From("order", Order.Type).Ref("sea_document_mode_change_events").Field("order_id").Unique().Required().Immutable(),
		edge.From("creator", User.Type).Ref("created_sea_document_mode_change_events").Field("created_by").Unique().Required().Immutable(),
		edge.From("confirmation_attachment", OrderAttachment.Type).Ref("sea_document_mode_change_events").Field("confirmation_attachment_id").Unique().Immutable(),
		edge.From("previous_house_bill", SeaHouseBill.Type).Ref("previous_mode_change_events").Field("previous_house_bill_id").Unique().Immutable(),
		edge.From("previous_house_bill_version", SeaHouseBillVersion.Type).Ref("previous_mode_change_events").Field("previous_house_bill_version_id").Unique().Immutable(),
		edge.From("target_house_bill", SeaHouseBill.Type).Ref("target_mode_change_events").Field("target_house_bill_id").Unique().Immutable(),
		edge.From("target_house_bill_version", SeaHouseBillVersion.Type).Ref("target_mode_change_events").Field("target_house_bill_version_id").Unique().Immutable(),
	}
}

func (SeaDocumentModeChangeEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "order_id"),
		index.Fields("organization_id", "idempotency_key").Unique().StorageKey("sea_doc_mode_change_idempotency_key"),
		index.Fields("organization_id", "request_fingerprint"),
	}
}
