package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderAttachmentAsset 保存唯一物理文件资产与元数据，一个物理文件只有一条记录。
type OrderAttachmentAsset struct{ ent.Schema }

func (OrderAttachmentAsset) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderAttachmentAsset) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Checks(map[string]string{
			"order_attachment_assets_file_size_check": "file_size > 0",
		}),
	}
}

func (OrderAttachmentAsset) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.String("object_key").NotEmpty().MaxLen(1024).Immutable(),
		field.String("file_name").NotEmpty().MaxLen(255),
		field.String("mime_type").NotEmpty().MaxLen(127),
		field.Int64("file_size").Positive(),
		field.String("checksum").Optional().Nillable().MaxLen(128),
		field.UUID("uploaded_by", uuid.Nil).Optional().Nillable(),
	}
}

func (OrderAttachmentAsset) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("attachment_assets").Field("organization_id").Unique().Required().Immutable(),
		edge.From("uploader", User.Type).Ref("uploaded_attachment_assets").Field("uploaded_by").Unique(),
		edge.To("attachments", OrderAttachment.Type),
	}
}

func (OrderAttachmentAsset) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("object_key").Unique().StorageKey("order_attachment_asset_object_key"),
		index.Fields("organization_id", "created_at"),
	}
}
