package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderCargoItem 定义订单货物明细行。
// 计费重、AE 仓储执行字段和申报状态按 P8/M-006 决策后置，不在首版伪造空列。
type OrderCargoItem struct{ ent.Schema }

func (OrderCargoItem) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderCargoItem) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("order_id", uuid.Nil),
		field.String("cargo_name").NotEmpty().MaxLen(200),
		field.Int("package_count").Positive(),
		field.Float("gross_weight_kg").Positive(),
		field.Float("volume_cbm").Positive(),
		field.Float("net_weight_kg").Optional().Positive(),
		field.String("note").Optional().MaxLen(500),
	}
}

func (OrderCargoItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).Ref("cargo_items").Field("order_id").Unique().Required(),
	}
}

func (OrderCargoItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id"),
	}
}
