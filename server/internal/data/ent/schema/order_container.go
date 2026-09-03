package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderContainer 定义订单集装箱。
// 箱型引用组织级 container_spec 主数据，装载状态与跟踪事件建模时再引入。
type OrderContainer struct{ ent.Schema }

func (OrderContainer) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderContainer) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.UUID("order_id", uuid.Nil),
		field.String("container_no").NotEmpty().MaxLen(64),
		field.UUID("container_spec_id", uuid.Nil),
		field.Int("package_count").Positive(),
		field.String("seal_no").Optional().MaxLen(64),
		field.Float("gross_weight_kg").Positive().SchemaType(map[string]string{dialect.Postgres: "numeric(18,3)"}),
		field.Float("volume_cbm").Positive().SchemaType(map[string]string{dialect.Postgres: "numeric(18,6)"}),
		field.String("note").Optional().MaxLen(500),
		field.Uint64("version").Default(1),
	}
}

func (OrderContainer) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("order_containers").Field("organization_id").Unique().Required(),
		edge.From("order", Order.Type).Ref("containers").Field("order_id").Unique().Required(),
		edge.To("cargo_allocations", SeaCargoAllocation.Type),
	}
}

func (OrderContainer) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "container_no").Unique(),
		index.Fields("organization_id"),
		index.Fields("order_id"),
		index.Fields("container_spec_id"),
	}
}
