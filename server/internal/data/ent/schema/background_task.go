package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// BackgroundTask 只保存后台任务的调度元数据与执行状态，不保存具体的业务 JSON payload。
type BackgroundTask struct{ ent.Schema }

func (BackgroundTask) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (BackgroundTask) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.Enum("kind").Values("MASTER_DATA_IMPORT", "UNLOCODE_IMPORT", "ORDER_REMINDER", "INTEGRATION", "DINGTALK_NOTIFICATION"),
		field.String("idempotency_key").NotEmpty().MaxLen(128),
		field.Enum("status").Values("PENDING", "RUNNING", "SUCCEEDED", "FAILED", "DEAD_LETTER").Default("PENDING"),
		field.Int("attempts").Default(0).NonNegative(),
		field.Int("max_attempts").Default(3).Min(1).Max(10),
		field.Time("next_run_at").Default(time.Now),
		field.String("lease_token").Optional().Nillable().MaxLen(128),
		field.Time("lease_expires_at").Optional().Nillable(),
		field.String("last_error").Optional().Nillable().MaxLen(2000),
	}
}

func (BackgroundTask) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("background_tasks").Field("organization_id").Unique().Required(),
		edge.To("notification_delivery", NotificationDelivery.Type).Unique(),
	}
}

func (BackgroundTask) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "kind", "idempotency_key").Unique(),
		index.Fields("status", "next_run_at"),
		index.Fields("status", "lease_expires_at"),
		index.Fields("organization_id", "created_at"),
	}
}
