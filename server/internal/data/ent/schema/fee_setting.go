package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FeeSetting 定义费用设置主数据。
// B 型基线+本地：organization_id 为空表示总部公共科目行（全网可见），非空表示
// 分公司本地明细科目行（同码时本地行覆盖基线行）；本地行必须挂载 A 型费用大类。
type FeeSetting struct{ ent.Schema }

func (FeeSetting) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FeeSetting) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Optional().Nillable(),
		field.String("fee_code").NotEmpty().MaxLen(32),
		field.String("name_zh").NotEmpty().MaxLen(64),
		field.String("name_en").Optional().Nillable().MaxLen(128),
		field.String("alias_name").Optional().Nillable().MaxLen(64),
		field.UUID("charge_category_id", uuid.Nil),
		field.String("default_currency").NotEmpty().MinLen(3).MaxLen(3),
		field.UUID("billing_unit_id", uuid.Nil),
		field.UUID("abnormal_case_id", uuid.Nil).Optional().Nillable(),
		field.String("tax_rate").SchemaType(map[string]string{dialect.Postgres: "numeric(5,2)"}),
		field.UUID("taxable_service_id", uuid.Nil),
		field.Bool("enabled").Default(true),
		field.Int("sort_order").Default(100),
		searchKeywordsField(),
	}
}

func (FeeSetting) Hooks() []ent.Hook {
	return []ent.Hook{searchKeywordsHook("name_zh", "name_en", "alias_name")}
}

func (FeeSetting) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("fee_settings").Field("organization_id").Unique(),
		edge.From("charge_category", MasterDataItem.Type).Ref("charge_category_fee_settings").Field("charge_category_id").Unique().Required(),
		edge.From("billing_unit", BillingUnit.Type).Ref("fee_settings").Field("billing_unit_id").Unique().Required(),
		edge.From("abnormal_case", MasterDataItem.Type).Ref("abnormal_case_fee_settings").Field("abnormal_case_id").Unique(),
		edge.From("taxable_service", TaxableService.Type).Ref("fee_settings").Field("taxable_service_id").Unique().Required(),
		edge.To("order_fees", OrderFee.Type),
	}
}

func (FeeSetting) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("fee_code").Unique().StorageKey("fee_settings_baseline_code_unique").Annotations(entsql.IndexWhere("organization_id IS NULL")),
		index.Fields("organization_id", "fee_code").Unique().StorageKey("fee_settings_org_code_unique").Annotations(entsql.IndexWhere("organization_id IS NOT NULL")),
		index.Fields("organization_id", "enabled", "sort_order"),
		index.Fields("organization_id", "charge_category_id", "abnormal_case_id"),
	}
}
