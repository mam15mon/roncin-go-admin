package schema

import (
	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// DingTalkInvitation 是分公司管理员预建的扫码邀请：只承载管理员决策字段
// （手机号/目标组织/初始角色），账号身份字段一律以钉钉扫码返回为准。
type DingTalkInvitation struct{ ent.Schema }

func (DingTalkInvitation) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (DingTalkInvitation) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.UUID("role_id", uuid.Nil),
		// 手机号存规范化明文（等值匹配需要），展示层统一脱敏。
		field.String("mobile").NotEmpty().MaxLen(32),
		field.String("display_name").MaxLen(100).Optional().Default(""),
		field.UUID("invited_by", uuid.Nil),
		field.Enum("status").Values("PENDING", "CONSUMED", "EXPIRED", "REVOKED").Default("PENDING"),
		field.UUID("consumed_by", uuid.Nil).Optional().Nillable(),
		field.Time("consumed_at").Optional().Nillable(),
		field.Time("expires_at"),
	}
}

func (DingTalkInvitation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("dingtalk_invitations").Field("organization_id").Unique().Required(),
		edge.From("role", Role.Type).Ref("dingtalk_invitations").Field("role_id").Unique().Required(),
		edge.From("inviter", User.Type).Ref("created_dingtalk_invitations").Field("invited_by").Unique().Required(),
		edge.From("consumer", User.Type).Ref("consumed_dingtalk_invitations").Field("consumed_by").Unique(),
	}
}

func (DingTalkInvitation) Indexes() []ent.Index {
	return []ent.Index{
		// 活跃邀请按组织 + 手机号唯一，过期/已消费/撤销后可重建。
		index.Fields("organization_id", "mobile").Unique().Annotations(entsql.IndexWhere("status = 'PENDING'")),
		index.Fields("expires_at"),
		index.Fields("invited_by"),
	}
}
