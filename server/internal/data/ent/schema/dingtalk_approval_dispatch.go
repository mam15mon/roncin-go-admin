package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// DingTalkApprovalDispatch 存储后台任务派发钉钉审批的载荷与状态。
type DingTalkApprovalDispatch struct{ ent.Schema }

func (DingTalkApprovalDispatch) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }

func (DingTalkApprovalDispatch) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("background_task_id", uuid.Nil).Immutable(),
		field.UUID("unlock_request_id", uuid.Nil).Immutable(),
		field.String("process_code_snapshot").NotEmpty().MaxLen(128).Immutable(),
		field.String("applicant_dingtalk_userid").NotEmpty().MaxLen(64).Immutable(),
		field.JSON("candidate_dingtalk_userids", []string{}).Immutable(),
		field.String("request_payload_hash").NotEmpty().MaxLen(64).Immutable(),
		field.Enum("dispatch_status").Values("PENDING", "DISPATCHED", "FAILED", "UNKNOWN").Default("PENDING"),
		field.String("process_instance_id").Optional().Nillable().MaxLen(128),
		field.String("response_digest").Optional().Nillable().MaxLen(500),
		field.String("error_category").Optional().Nillable().MaxLen(64),
	}
}

func (DingTalkApprovalDispatch) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("dingtalk_approval_dispatches").Field("organization_id").Unique().Required().Immutable(),
		edge.From("background_task", BackgroundTask.Type).Ref("dingtalk_approval_dispatch").Field("background_task_id").Unique().Required().Immutable(),
		edge.From("unlock_request", OrderUnlockRequest.Type).Ref("dispatch").Field("unlock_request_id").Unique().Required().Immutable(),
	}
}

func (DingTalkApprovalDispatch) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("background_task_id").Unique().StorageKey("dingtalk_approval_dispatch_bg_task_unique"),
		index.Fields("unlock_request_id").Unique().StorageKey("dingtalk_approval_dispatch_unlock_req_unique"),
		index.Fields("organization_id", "dispatch_status"),
	}
}
