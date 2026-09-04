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

// SeaDocumentVoidEvent 记录海运单证作废的不可变事件。
type SeaDocumentVoidEvent struct{ ent.Schema }

func (SeaDocumentVoidEvent) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }

func (SeaDocumentVoidEvent) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Checks(map[string]string{
			"sea_document_void_events_document_type_check": "((document_type = 'MASTER' AND master_bill_id IS NOT NULL AND house_bill_id IS NULL) OR (document_type = 'HOUSE' AND house_bill_id IS NOT NULL AND master_bill_id IS NULL))",
		}),
	}
}

func (SeaDocumentVoidEvent) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Immutable().Default(time.Now),
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("order_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.Enum("document_type").Values("MASTER", "HOUSE").Immutable(),
		field.UUID("master_bill_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("master_bill_version_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("house_bill_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("house_bill_version_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.String("previous_status").NotEmpty().MaxLen(32).Immutable(),
		field.String("voided_status").NotEmpty().MaxLen(32).Immutable(),
		field.String("reason").NotEmpty().MaxLen(500).Immutable(),
		field.String("impact_summary").Optional().Nillable().MaxLen(1000).Immutable(),
		field.UUID("created_by", uuid.Nil).Immutable(),
	}
}

func (SeaDocumentVoidEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("sea_document_void_events").Field("organization_id").Unique().Required().Immutable(),
		edge.From("order", Order.Type).Ref("sea_document_void_events").Field("order_id").Unique().Immutable(),
		edge.From("master_bill", SeaMasterBill.Type).Ref("void_events").Field("master_bill_id").Unique().Immutable(),
		edge.From("master_bill_version", SeaMasterBillVersion.Type).Ref("void_events").Field("master_bill_version_id").Unique().Immutable(),
		edge.From("house_bill", SeaHouseBill.Type).Ref("void_events").Field("house_bill_id").Unique().Immutable(),
		edge.From("house_bill_version", SeaHouseBillVersion.Type).Ref("void_events").Field("house_bill_version_id").Unique().Immutable(),
		edge.From("creator", User.Type).Ref("created_sea_document_void_events").Field("created_by").Unique().Required().Immutable(),
	}
}

func (SeaDocumentVoidEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "document_type"),
		index.Fields("organization_id", "master_bill_id"),
		index.Fields("organization_id", "house_bill_id"),
	}
}
