package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderAttachment 是操作票对文件资产的引用关系。
type OrderAttachment struct{ ent.Schema }

func (OrderAttachment) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderAttachment) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("order_id", uuid.Nil),
		field.UUID("asset_id", uuid.Nil),
		field.String("doc_type").NotEmpty().MaxLen(64),
		field.String("idempotency_key").NotEmpty().MaxLen(128),
		field.UUID("created_by", uuid.Nil).Optional().Nillable(),
	}
}

func (OrderAttachment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).Ref("attachments").Field("order_id").Unique().Required(),
		edge.From("asset", OrderAttachmentAsset.Type).Ref("attachments").Field("asset_id").Unique().Required(),
		edge.From("creator", User.Type).Ref("created_order_attachments").Field("created_by").Unique(),
	}
}

func (OrderAttachment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "asset_id").Unique().StorageKey("order_attachment_order_asset"),
		index.Fields("order_id", "idempotency_key").Unique().StorageKey("order_attachment_idempotency_key"),
		index.Fields("order_id", "doc_type", "created_at"),
		index.Fields("asset_id"),
	}
}
