package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// PasswordResetChallenge 仅保存当前代找回资格及摘要，不持久化验证码或浏览器令牌。
type PasswordResetChallenge struct{ ent.Schema }

func (PasswordResetChallenge) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }
func (PasswordResetChallenge) Fields() []ent.Field {
	return []ent.Field{
		field.String("mobile_key_hash").MaxLen(64).NotEmpty().Sensitive(),
		field.String("token_hash").MaxLen(64).NotEmpty().Sensitive(),
		field.String("code_hash").MaxLen(64).Default("").Sensitive(),
		field.UUID("user_id", uuid.Nil).Optional().Nillable(),
		field.String("dingtalk_userid").MaxLen(64).Default("").Sensitive(),
		field.String("dingtalk_unionid").MaxLen(128).Default("").Sensitive(),
		field.Enum("status").Values("PENDING", "READY", "FAILED", "CONSUMED"),
		field.Time("expires_at"),
		field.Time("resend_available_at"),
		field.Int("failed_attempts").Default(0).Min(0).Max(5),
	}
}
func (PasswordResetChallenge) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("mobile_key_hash").Unique(), index.Fields("token_hash").Unique(), index.Fields("user_id"),
	}
}
func (PasswordResetChallenge) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Checks(map[string]string{
		"password_reset_challenges_status_check":          "status IN ('PENDING', 'READY', 'FAILED', 'CONSUMED')",
		"password_reset_challenges_failed_attempts_check": "failed_attempts BETWEEN 0 AND 5",
	})}
}
