package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Session stores only a hash of the browser session token.
type Session struct{ ent.Schema }

func (Session) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (Session) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("user_id", uuid.Nil).Immutable(),
		field.UUID("organization_id", uuid.Nil),
		field.String("token_hash").NotEmpty().MaxLen(64).Unique().Sensitive(),
		field.Time("expires_at"),
		field.Time("last_seen_at").Default(time.Now),
		field.Time("revoked_at").Optional().Nillable(),
		field.String("ip_address").MaxLen(64).Optional(),
		field.String("user_agent").MaxLen(512).Optional(),
	}
}

func (Session) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("sessions").Field("user_id").Unique().Required(),
		edge.From("organization", Organization.Type).Ref("sessions").Field("organization_id").Unique().Required(),
	}
}

func (Session) Indexes() []ent.Index {
	return []ent.Index{index.Fields("user_id", "expires_at"), index.Fields("expires_at", "revoked_at")}
}
