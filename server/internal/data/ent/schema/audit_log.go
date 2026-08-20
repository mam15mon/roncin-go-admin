package schema

import (
	"encoding/json"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// AuditLog is an append-only business security event.
type AuditLog struct{ ent.Schema }

func (AuditLog) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (AuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("user_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.String("action").NotEmpty().MaxLen(160).Immutable(),
		field.String("resource_type").MaxLen(100).Optional().Immutable(),
		field.String("resource_id").MaxLen(160).Optional().Immutable(),
		field.Enum("result").Values("success", "failure").Immutable(),
		field.String("request_id").MaxLen(64).Optional().Immutable(),
		field.String("trace_id").MaxLen(64).Optional().Immutable(),
		field.String("ip_address").MaxLen(64).Optional().Immutable(),
		field.JSON("details", json.RawMessage{}).Optional().Immutable(),
	}
}

func (AuditLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "created_at"),
		index.Fields("user_id", "created_at"),
		index.Fields("action", "created_at"),
		index.Fields("request_id"),
	}
}
