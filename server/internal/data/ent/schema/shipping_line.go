package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ShippingLine 定义班轮公司的 SCAC 标准标识。
// A 型全局主数据：SCAC 等客观编码全球唯一，无组织归属。
type ShippingLine struct{ ent.Schema }

func (ShippingLine) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (ShippingLine) Fields() []ent.Field {
	return []ent.Field{
		field.String("scac_code").NotEmpty().MaxLen(4).Immutable(),
		field.String("name_zh").NotEmpty().MaxLen(200),
		field.String("name_en").NotEmpty().MaxLen(200),
		field.String("country_code").NotEmpty().MaxLen(2),
		field.String("tracking_url").MaxLen(500).Optional().Nillable(),
		field.String("alliance").MaxLen(100).Optional().Nillable(),
		field.String("source").MaxLen(100).Default("manual"),
		field.Int("sort_order").Default(100),
		field.Bool("enabled").Default(true),
		searchKeywordsField(),
	}
}

func (ShippingLine) Hooks() []ent.Hook {
	return []ent.Hook{searchKeywordsHook("name_zh", "name_en")}
}

func (ShippingLine) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("container_prefixes", ShippingLineContainerPrefix.Type),
		edge.To("orders", Order.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
		edge.To("sea_transport_executions", SeaTransportExecution.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
		edge.To("sea_transport_execution_versions", SeaTransportExecutionVersion.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
		edge.To("sea_master_bills", SeaMasterBill.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
		edge.To("sea_master_bill_versions", SeaMasterBillVersion.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
	}
}

func (ShippingLine) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("scac_code").Unique(),
		index.Fields("enabled", "sort_order"),
	}
}
