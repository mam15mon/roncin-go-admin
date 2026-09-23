package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// BillingUnit 定义费用计费单位。
// A 型全局主数据：客观通用单位全球唯一，无组织归属。
type BillingUnit struct{ ent.Schema }

func (BillingUnit) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (BillingUnit) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").NotEmpty().MaxLen(32),
		field.String("name").NotEmpty().MaxLen(64),
		field.Bool("is_container_unit").Default(false),
		field.Bool("quantity_must_be_integer").Default(false),
		field.Int("sort_order").Default(100),
		field.Bool("enabled").Default(true),
		searchKeywordsField(),
	}
}

func (BillingUnit) Hooks() []ent.Hook { return []ent.Hook{searchKeywordsHook("name")} }

func (BillingUnit) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("fee_settings", FeeSetting.Type),
		edge.To("order_fees", OrderFee.Type),
	}
}

func (BillingUnit) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code").Unique(),
		index.Fields("enabled", "sort_order"),
	}
}
