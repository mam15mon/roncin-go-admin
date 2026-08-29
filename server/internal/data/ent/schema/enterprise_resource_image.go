package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// EnterpriseResourceImage 保存私有对象存储中的图片元数据。
type EnterpriseResourceImage struct{ ent.Schema }

func (EnterpriseResourceImage) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (EnterpriseResourceImage) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("resource_id", uuid.Nil).Immutable(),
		field.String("file_name").NotEmpty().MaxLen(255),
		field.String("mime_type").NotEmpty().MaxLen(127),
		field.Int64("file_size").Positive(),
		field.String("object_key").NotEmpty().MaxLen(1024).Immutable(),
		field.String("checksum").NotEmpty().MaxLen(128),
		field.Int("width").Optional().Nillable().Positive(),
		field.Int("height").Optional().Nillable().Positive(),
		field.UUID("uploaded_by", uuid.Nil).Immutable(),
	}
}

func (EnterpriseResourceImage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("resource", EnterpriseResource.Type).Ref("image").Field("resource_id").Unique().Required().Immutable(),
		edge.From("uploader", User.Type).Ref("uploaded_enterprise_resource_images").Field("uploaded_by").Unique().Required().Immutable(),
	}
}

func (EnterpriseResourceImage) Indexes() []ent.Index {
	return []ent.Index{index.Fields("resource_id").Unique(), index.Fields("object_key").Unique()}
}
