package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

func seaBillContentFields() []ent.Field {
	return []ent.Field{
		field.Text("shipper_text").Optional().Nillable(),
		field.Text("consignee_text").Optional().Nillable(),
		field.Text("notify_party_text").Optional().Nillable(),
		field.Text("second_notify_party_text").Optional().Nillable(),
		field.Text("marks_text").Optional().Nillable(),
		field.Text("goods_description_text").Optional().Nillable(),
		field.Int("package_count").Optional().Nillable().NonNegative(),
		field.String("package_unit").Optional().Nillable().MaxLen(64),
		field.Float("gross_weight_kg").Optional().Nillable().Min(0),
		field.Float("volume_cbm").Optional().Nillable().Min(0),
		field.String("freight_terms").Optional().Nillable().MaxLen(64),
		field.String("transport_terms").Optional().Nillable().MaxLen(64),
		field.String("bill_form").Optional().Nillable().MaxLen(64),
		field.String("release_type").Optional().Nillable().MaxLen(64),
		field.Text("clauses").Optional().Nillable(),
	}
}

// immutableSeaBillContentFields 用于历史版本实体，确保提单正文只能在创建时写入。
// 不复用可变实体的字段定义，避免 Ent 为版本实体生成 Update setter。
func immutableSeaBillContentFields() []ent.Field {
	return []ent.Field{
		field.Text("shipper_text").Optional().Nillable().Immutable(),
		field.Text("consignee_text").Optional().Nillable().Immutable(),
		field.Text("notify_party_text").Optional().Nillable().Immutable(),
		field.Text("second_notify_party_text").Optional().Nillable().Immutable(),
		field.Text("marks_text").Optional().Nillable().Immutable(),
		field.Text("goods_description_text").Optional().Nillable().Immutable(),
		field.Int("package_count").Optional().Nillable().NonNegative().Immutable(),
		field.String("package_unit").Optional().Nillable().MaxLen(64).Immutable(),
		field.Float("gross_weight_kg").Optional().Nillable().Min(0).Immutable(),
		field.Float("volume_cbm").Optional().Nillable().Min(0).Immutable(),
		field.String("freight_terms").Optional().Nillable().MaxLen(64).Immutable(),
		field.String("transport_terms").Optional().Nillable().MaxLen(64).Immutable(),
		field.String("bill_form").Optional().Nillable().MaxLen(64).Immutable(),
		field.String("release_type").Optional().Nillable().MaxLen(64).Immutable(),
		field.Text("clauses").Optional().Nillable().Immutable(),
	}
}
