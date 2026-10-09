package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// DingTalkDirectorySync 保存通讯录同步的业务结果，后台任务仅保存调度元数据。
type DingTalkDirectorySync struct{ ent.Schema }

func (DingTalkDirectorySync) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }
func (DingTalkDirectorySync) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("background_task_id", uuid.Nil).Immutable(),
		field.UUID("actor_id", uuid.Nil).Immutable(),
		field.String("corp_id").NotEmpty().MaxLen(128).Immutable(),
		field.Bool("active").Default(true),
		field.String("scope_summary").Default("").MaxLen(500),
		field.Bool("full_scope").Default(false),
		field.JSON("available_fields", []string{}).Optional(),
		field.Int("total_count").Default(0).NonNegative(), field.Int("created_count").Default(0).NonNegative(), field.Int("updated_count").Default(0).NonNegative(), field.Int("unchanged_count").Default(0).NonNegative(), field.Int("failed_count").Default(0).NonNegative(),
		field.JSON("failures", []DirectoryFailure{}).Optional(),
		field.Time("completed_at").Optional().Nillable(),
	}
}
func (DingTalkDirectorySync) Edges() []ent.Edge {
	return []ent.Edge{edge.From("background_task", BackgroundTask.Type).Ref("dingtalk_directory_sync").Field("background_task_id").Unique().Required().Immutable()}
}
func (DingTalkDirectorySync) Indexes() []ent.Index {
	return []ent.Index{index.Fields("background_task_id").Unique(), index.Fields("corp_id").Unique().StorageKey("dingtalk_directory_sync_active_corp_unique").Annotations(entsql.IndexWhere("active = true"))}
}

type DirectoryFailure struct {
	UserID       string `json:"userId"`
	DepartmentID int64  `json:"departmentId"`
	Reason       string `json:"reason"`
	Message      string `json:"message"`
}
