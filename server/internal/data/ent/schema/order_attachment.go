package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderAttachment 只保存外部对象存储引用及文件元数据，不在数据库保存文件内容。
type OrderAttachment struct{ ent.Schema }

func (OrderAttachment) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderAttachment) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("order_id", uuid.Nil),
		field.String("doc_type").NotEmpty().MaxLen(64),
		field.String("idempotency_key").NotEmpty().MaxLen(128),
		field.String("file_name").NotEmpty().MaxLen(255),
		field.String("mime_type").NotEmpty().MaxLen(127),
		field.Int64("file_size").Positive(),
		field.String("object_key").NotEmpty().MaxLen(1024),
		field.String("checksum").Optional().Nillable().MaxLen(128),
		field.UUID("uploaded_by", uuid.Nil).Optional().Nillable(),
	}
}

func (OrderAttachment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).Ref("attachments").Field("order_id").Unique().Required(),
	}
}

func (OrderAttachment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "idempotency_key").Unique().StorageKey("order_attachment_idempotency_key"),
		index.Fields("object_key").Unique().StorageKey("order_attachment_object_key"),
		index.Fields("order_id", "doc_type", "created_at"),
	}
}
