package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// ExchangeRateCustomSetting 保存总部组织的汇率自定义策略。
// 子组织沿用其汇率归属总部的策略，避免同一套汇率主数据出现不同继承口径。
type ExchangeRateCustomSetting struct{ ent.Schema }

func (ExchangeRateCustomSetting) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (ExchangeRateCustomSetting) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.Bool("inherit_base_currency_rate").Default(false),
		field.Uint64("version").Default(1),
		field.UUID("updated_by", uuid.Nil),
	}
}

func (ExchangeRateCustomSetting) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("exchange_rate_custom_setting").Field("organization_id").Unique().Required().Immutable(),
		edge.From("updated_by_user", User.Type).Ref("updated_exchange_rate_custom_settings").Field("updated_by").Unique().Required(),
	}
}

func (ExchangeRateCustomSetting) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id").Unique().StorageKey("exchange_rate_custom_setting_organization_unique"),
	}
}
