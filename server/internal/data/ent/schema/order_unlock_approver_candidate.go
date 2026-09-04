package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderUnlockApproverCandidate 保存普通编辑人申请解锁时解析到的审批候选人快照。
type OrderUnlockApproverCandidate struct{ ent.Schema }

func (OrderUnlockApproverCandidate) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }

func (OrderUnlockApproverCandidate) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Immutable().Default(time.Now),
		field.UUID("request_id", uuid.Nil).Immutable(),
		field.UUID("user_id", uuid.Nil).Immutable(),
		field.UUID("membership_id", uuid.Nil).Immutable(),
		field.UUID("role_id", uuid.Nil).Immutable(),
		field.String("display_name_snapshot").NotEmpty().MaxLen(100).Immutable(),
		field.String("dingtalk_userid_snapshot").NotEmpty().MaxLen(64).Immutable(),
	}
}

func (OrderUnlockApproverCandidate) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("request", OrderUnlockRequest.Type).Ref("approver_candidates").Field("request_id").Unique().Required().Immutable(),
		edge.From("user", User.Type).Ref("order_unlock_approver_candidates").Field("user_id").Unique().Required().Immutable(),
		edge.From("membership", Membership.Type).Ref("order_unlock_approver_candidates").Field("membership_id").Unique().Required().Immutable(),
		edge.From("role", Role.Type).Ref("order_unlock_approver_candidates").Field("role_id").Unique().Required().Immutable(),
	}
}

func (OrderUnlockApproverCandidate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("request_id", "user_id").Unique().StorageKey("order_unlock_approver_candidate_unique"),
		index.Fields("request_id"),
	}
}
