package schema

import (
	"encoding/json"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// PasskeyCredential 仅保存本人通行密钥的公共凭证数据，不保存私钥或仪式挑战。
type PasskeyCredential struct{ ent.Schema }

func (PasskeyCredential) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }
func (PasskeyCredential) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("user_id", uuid.Nil),
		field.String("rp_id").NotEmpty().MaxLen(253),
		field.String("credential_id").NotEmpty().MaxLen(2048).Sensitive(),
		field.JSON("credential_json", json.RawMessage{}).Sensitive(),
		field.String("name").NotEmpty().MaxLen(100),
		field.Time("last_used_at").Optional().Nillable(),
	}
}
func (PasskeyCredential) Edges() []ent.Edge {
	return []ent.Edge{edge.From("user", User.Type).Ref("passkey_credentials").Field("user_id").Unique().Required().Annotations(entsql.OnDelete(entsql.NoAction))}
}
func (PasskeyCredential) Indexes() []ent.Index {
	return []ent.Index{index.Fields("rp_id", "credential_id").Unique().StorageKey("passkey_credentials_rp_credential_key"), index.Fields("user_id", "rp_id").StorageKey("passkey_credentials_user_rp")}
}
