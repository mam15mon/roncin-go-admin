package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceCustomSetting 保存总部组织的通用财务自定义策略。
type FinanceCustomSetting struct{ ent.Schema }

func (FinanceCustomSetting) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceCustomSetting) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.Bool("billed_fee_edit_enabled").Default(false),
		field.Bool("billed_fee_name_editable").Default(false),
		field.Bool("billed_fee_currency_editable").Default(false),
		field.Bool("billed_fee_exchange_rate_editable").Default(false),
		field.Bool("billed_fee_quantity_editable").Default(false),
		field.Bool("billed_fee_unit_price_editable").Default(false),
		field.Bool("billed_fee_tax_rate_editable").Default(false),
		field.Uint64("version").Default(1),
		field.UUID("updated_by", uuid.Nil),
	}
}

func (FinanceCustomSetting) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("finance_custom_setting").Field("organization_id").Unique().Required().Immutable(),
		edge.From("updated_by_user", User.Type).Ref("updated_finance_custom_settings").Field("updated_by").Unique().Required(),
	}
}

func (FinanceCustomSetting) Indexes() []ent.Index {
	return []ent.Index{index.Fields("organization_id").Unique().StorageKey("finance_custom_setting_organization_unique")}
}
