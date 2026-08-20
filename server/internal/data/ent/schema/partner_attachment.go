package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// PartnerAttachment 只保存外部对象存储引用及文件元数据，不在数据库保存文件内容。
type PartnerAttachment struct{ ent.Schema }

func (PartnerAttachment) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (PartnerAttachment) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("partner_id", uuid.Nil),
		field.String("idempotency_key").NotEmpty().MaxLen(128),
		field.String("file_name").NotEmpty().MaxLen(255),
		field.String("mime_type").NotEmpty().MaxLen(127),
		field.Int64("file_size").Positive(),
		field.String("object_key").NotEmpty().MaxLen(1024),
		field.String("checksum").Optional().Nillable().MaxLen(128),
		field.UUID("uploaded_by", uuid.Nil).Optional().Nillable(),
	}
}

func (PartnerAttachment) Edges() []ent.Edge {
	return []ent.Edge{edge.From("partner", Partner.Type).Ref("attachments").Field("partner_id").Unique().Required()}
}

func (PartnerAttachment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("partner_id", "idempotency_key").Unique().StorageKey("partner_attachment_idempotency_key"),
		index.Fields("partner_id", "created_at"),
	}
}
