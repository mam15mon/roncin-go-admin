package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LoginRateLimitBucket 保存密码登录失败计数，不存储明文账号或 IP。
type LoginRateLimitBucket struct{ ent.Schema }

func (LoginRateLimitBucket) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (LoginRateLimitBucket) Fields() []ent.Field {
	return []ent.Field{
		field.String("key_hash").NotEmpty().MaxLen(64).Sensitive(),
		field.Time("window_started_at"),
		field.Int("attempts").Positive(),
	}
}

func (LoginRateLimitBucket) Indexes() []ent.Index {
	return []ent.Index{index.Fields("key_hash").Unique()}
}
