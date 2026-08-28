package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// NotificationDelivery 保存后台通知任务的收件人和业务引用，不保存访问令牌。
type NotificationDelivery struct{ ent.Schema }

func (NotificationDelivery) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (NotificationDelivery) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("background_task_id", uuid.Nil).Immutable(),
		field.UUID("recipient_user_id", uuid.Nil).Immutable(),
		field.Enum("channel").Values("DINGTALK").Immutable(),
		field.Enum("template").Values("ORDER_PERSONNEL_ASSIGNED", "USER_AUTHORIZED").Immutable(),
		field.String("resource_type").NotEmpty().MaxLen(64).Immutable(),
		field.UUID("resource_id", uuid.Nil).Immutable(),
		field.String("reference_code").NotEmpty().MaxLen(64).Immutable(),
		field.String("parameter").Optional().MaxLen(64).Immutable(),
		field.String("external_message_id").Optional().Nillable().MaxLen(256),
	}
}

func (NotificationDelivery) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("background_task", BackgroundTask.Type).Ref("notification_delivery").Field("background_task_id").Unique().Required().Immutable(),
		edge.From("recipient_user", User.Type).Ref("notification_deliveries").Field("recipient_user_id").Unique().Required().Immutable(),
	}
}

func (NotificationDelivery) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("background_task_id").Unique(),
		index.Fields("recipient_user_id", "created_at"),
		index.Fields("resource_type", "resource_id"),
	}
}
