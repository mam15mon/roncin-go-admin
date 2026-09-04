package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// DingTalkApprovalInboxEvent 存储验签解密后的钉钉审批入站回调事件。
type DingTalkApprovalInboxEvent struct{ ent.Schema }

func (DingTalkApprovalInboxEvent) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }

func (DingTalkApprovalInboxEvent) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Checks(map[string]string{
		"ding_talk_approval_inbox_events_status_check":   "status IN ('RECEIVED', 'PROCESSING', 'PROCESSED', 'IGNORED', 'FAILED')",
		"ding_talk_approval_inbox_events_attempts_check": "attempts >= 0",
	})}
}

func (DingTalkApprovalInboxEvent) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Immutable().Default(time.Now),
		field.UUID("organization_id", uuid.Nil).Optional().Nillable(),
		field.String("event_id").NotEmpty().MaxLen(128).Immutable(),
		field.String("corp_id").NotEmpty().MaxLen(64).Immutable(),
		field.String("event_type").NotEmpty().MaxLen(64).Immutable(),
		field.String("process_instance_id").NotEmpty().MaxLen(128).Immutable(),
		field.Time("received_at").Immutable().Default(time.Now),
		field.String("encrypted_payload_hash").NotEmpty().MaxLen(64).Immutable(),
		field.String("parsed_summary").Optional().Nillable().MaxLen(1000),
		field.Enum("status").Values("RECEIVED", "PROCESSING", "PROCESSED", "IGNORED", "FAILED").Default("RECEIVED"),
		field.Int("attempts").Default(0).NonNegative(),
		field.Time("next_run_at").Default(time.Now),
		field.String("processing_token").Optional().Nillable().MaxLen(64),
		field.Time("processing_expires_at").Optional().Nillable(),
		field.String("result_code").Optional().Nillable().MaxLen(64),
		field.String("error_message").Optional().Nillable().MaxLen(500),
	}
}

func (DingTalkApprovalInboxEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("event_id").Unique().StorageKey("dingtalk_approval_inbox_event_id_unique"),
		index.Fields("process_instance_id"),
		index.Fields("status", "next_run_at", "received_at"),
	}
}
