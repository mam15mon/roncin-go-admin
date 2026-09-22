package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FeeSettingTemplate 定义系统初始费用目录，不引用公司的税务资料。
type FeeSettingTemplate struct{ ent.Schema }

func (FeeSettingTemplate) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FeeSettingTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.String("fee_code").NotEmpty().MaxLen(32),
		field.String("name_zh").NotEmpty().MaxLen(64),
		field.String("name_en").Optional().Nillable().MaxLen(128),
		field.String("alias_name").Optional().Nillable().MaxLen(64),
		field.UUID("charge_category_id", uuid.Nil),
		field.String("default_currency").NotEmpty().MinLen(3).MaxLen(3),
		field.UUID("billing_unit_id", uuid.Nil),
		field.UUID("abnormal_case_id", uuid.Nil).Optional().Nillable(),
		field.String("tax_rate").SchemaType(map[string]string{dialect.Postgres: "numeric(5,2)"}),
		field.String("taxable_service_name").NotEmpty().MaxLen(128),
		field.String("taxable_service_short_name").Optional().Nillable().MaxLen(64),
		field.String("taxable_service_goods_code").Optional().Nillable().MaxLen(64),
		field.String("taxable_service_default_tax_rate").SchemaType(map[string]string{dialect.Postgres: "numeric(5,2)"}),
		field.Bool("enabled").Default(true),
		field.Int("sort_order").Default(100),
		searchKeywordsField(),
	}
}

func (FeeSettingTemplate) Hooks() []ent.Hook {
	return []ent.Hook{searchKeywordsHook("name_zh", "name_en", "alias_name")}
}

func (FeeSettingTemplate) Indexes() []ent.Index {
	return []ent.Index{index.Fields("fee_code").Unique(), index.Fields("enabled", "sort_order")}
}
