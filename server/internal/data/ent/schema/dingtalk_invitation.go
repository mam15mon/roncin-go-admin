package schema

import (
	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// DingTalkInvitation 是分公司管理员预建的扫码邀请：
// - kind = TARGETED：定向手机号邀请（单人一次性，手机号必填，消费后变 CONSUMED，参与自动激活）；
// - kind = GENERIC：分公司专属通用扩招码（多人多次扫码，无手机号，永不变为 CONSUMED，新员工扫码为 PENDING 待审批）。
type DingTalkInvitation struct{ ent.Schema }

func (DingTalkInvitation) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (DingTalkInvitation) Fields() []ent.Field {
	return []ent.Field{
		field.String("token").NotEmpty().MaxLen(64).Unique(),
		field.Enum("kind").Values("TARGETED", "GENERIC").Default("TARGETED"),
		field.UUID("organization_id", uuid.Nil),
		field.UUID("role_id", uuid.Nil).Optional().Nillable(),
		// 手机号存规范化明文（等值匹配需要），仅 TARGETED 必填，展示层统一脱敏。
		field.String("mobile").Optional().Nillable().MaxLen(32),
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
		edge.From("role", Role.Type).Ref("dingtalk_invitations").Field("role_id").Unique(),
		edge.From("inviter", User.Type).Ref("created_dingtalk_invitations").Field("invited_by").Unique().Required(),
		edge.From("consumer", User.Type).Ref("consumed_dingtalk_invitations").Field("consumed_by").Unique(),
	}
}

func (DingTalkInvitation) Indexes() []ent.Index {
	return []ent.Index{
		// 活跃定向邀请按组织 + 手机号唯一，过期/已消费/撤销后可重建；通用码 (GENERIC) 不受此唯一性限制。
		index.Fields("organization_id", "mobile").Unique().Annotations(entsql.IndexWhere("status = 'PENDING' AND kind = 'TARGETED' AND mobile IS NOT NULL AND mobile != ''")),
		index.Fields("expires_at"),
		index.Fields("invited_by"),
	}
}
