package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// ObjectStorageDeletion 保存对象存储删除任务的待删对象键，不含访问凭据。
type ObjectStorageDeletion struct{ ent.Schema }

func (ObjectStorageDeletion) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (ObjectStorageDeletion) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("background_task_id", uuid.Nil).Immutable(),
		field.String("object_key").NotEmpty().MaxLen(1024).Immutable(),
	}
}

func (ObjectStorageDeletion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("background_task", BackgroundTask.Type).Ref("object_storage_deletion").Field("background_task_id").Unique().Required().Immutable(),
	}
}

func (ObjectStorageDeletion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("background_task_id").Unique(),
	}
}
