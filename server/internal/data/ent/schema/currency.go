package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Currency 是全局 ISO 4217 币种字典，不随组织重复维护。
type Currency struct{ ent.Schema }

func (Currency) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (Currency) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("name").NotEmpty().MaxLen(100),
		field.String("symbol").Optional().MaxLen(16),
		field.Int("minor_unit").Default(2).Min(0).Max(4),
		field.Bool("enabled").Default(true),
		searchKeywordsField(),
	}
}

func (Currency) Hooks() []ent.Hook { return []ent.Hook{searchKeywordsHook("name")} }

func (Currency) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code").Unique(),
		index.Fields("enabled", "code"),
	}
}
