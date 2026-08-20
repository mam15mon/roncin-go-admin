package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// NumberRule 定义组织内某类单据的编号格式。
type NumberRule struct{ ent.Schema }

func (NumberRule) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (NumberRule) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.Enum("document_type").Values("order", "booking", "hbl", "mbl", "bill", "statement", "payment", "invoice"),
		field.String("prefix").MaxLen(32),
		field.Enum("date_format").Values("yyyyMMdd", "yyyyMM", "yyyy", "none").Default("yyyyMMdd"),
		field.Int("sequence_length").Default(4).Min(1).Max(12),
		field.Enum("reset_policy").Values("daily", "monthly", "yearly", "never").Default("daily"),
		field.Bool("enabled").Default(true),
	}
}

func (NumberRule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("number_rules").Field("organization_id").Unique().Required(),
		edge.To("sequences", NumberSequence.Type),
	}
}

func (NumberRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "document_type").Unique(),
		index.Fields("organization_id", "enabled"),
	}
}
