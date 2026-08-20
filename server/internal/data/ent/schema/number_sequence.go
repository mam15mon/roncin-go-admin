package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// NumberSequence 保存编号规则在一个重置周期内的当前序列值。
type NumberSequence struct{ ent.Schema }

func (NumberSequence) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (NumberSequence) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("rule_id", uuid.Nil),
		field.String("period_key").NotEmpty().MaxLen(16).Immutable(),
		field.Int64("current_value").Default(0).Min(0),
	}
}

func (NumberSequence) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("rule", NumberRule.Type).Ref("sequences").Field("rule_id").Unique().Required(),
	}
}

func (NumberSequence) Indexes() []ent.Index {
	return []ent.Index{index.Fields("rule_id", "period_key").Unique()}
}
