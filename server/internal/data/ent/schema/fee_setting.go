package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FeeSetting 定义组织级费用设置主数据。
type FeeSetting struct{ ent.Schema }

func (FeeSetting) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FeeSetting) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.String("fee_code").NotEmpty().MaxLen(32),
		field.String("name_zh").NotEmpty().MaxLen(64),
		field.String("name_en").Optional().Nillable().MaxLen(128),
		field.String("alias_name").Optional().Nillable().MaxLen(64),
		field.UUID("service_type_id", uuid.Nil).Optional().Nillable(),
		field.String("default_currency").NotEmpty().MinLen(3).MaxLen(3),
		field.UUID("billing_unit_id", uuid.Nil),
		field.UUID("abnormal_case_id", uuid.Nil).Optional().Nillable(),
		field.String("tax_rate").SchemaType(map[string]string{dialect.Postgres: "numeric(5,2)"}),
		field.UUID("taxable_service_id", uuid.Nil),
		field.Bool("enabled").Default(true),
		field.Int("sort_order").Default(100),
	}
}

func (FeeSetting) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("fee_settings").Field("organization_id").Unique().Required(),
		edge.From("service_type", MasterDataItem.Type).Ref("service_type_fee_settings").Field("service_type_id").Unique(),
		edge.From("billing_unit", BillingUnit.Type).Ref("fee_settings").Field("billing_unit_id").Unique().Required(),
		edge.From("abnormal_case", MasterDataItem.Type).Ref("abnormal_case_fee_settings").Field("abnormal_case_id").Unique(),
		edge.From("taxable_service", TaxableService.Type).Ref("fee_settings").Field("taxable_service_id").Unique().Required(),
		edge.To("order_fees", OrderFee.Type),
	}
}

func (FeeSetting) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "fee_code").Unique(),
		index.Fields("organization_id", "enabled", "sort_order"),
		index.Fields("organization_id", "service_type_id", "abnormal_case_id"),
	}
}
