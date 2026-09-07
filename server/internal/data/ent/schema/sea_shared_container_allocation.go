package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// SeaSharedContainerAllocation 定义跨订单共享集装箱下的逐票（HOUSE Order / HBL / CargoItem）定量分配。
type SeaSharedContainerAllocation struct{ ent.Schema }

func (SeaSharedContainerAllocation) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (SeaSharedContainerAllocation) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.UUID("shared_container_id", uuid.Nil),
		field.UUID("order_id", uuid.Nil),
		field.UUID("house_bill_id", uuid.Nil),
		field.UUID("cargo_item_id", uuid.Nil),
		field.Int("package_count").Positive(),
		field.Float("gross_weight_kg").Positive().SchemaType(map[string]string{dialect.Postgres: "numeric(18,3)"}),
		field.Float("volume_cbm").Positive().SchemaType(map[string]string{dialect.Postgres: "numeric(18,6)"}),
		field.Uint64("version").Default(1),
	}
}

func (SeaSharedContainerAllocation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("sea_shared_container_allocations").Field("organization_id").Unique().Required(),
		edge.From("shared_container", SeaSharedContainer.Type).Ref("allocations").Field("shared_container_id").Unique().Required(),
		edge.From("order", Order.Type).Ref("sea_shared_container_allocations").Field("order_id").Unique().Required(),
		edge.From("house_bill", SeaHouseBill.Type).Ref("shared_container_allocations").Field("house_bill_id").Unique().Required(),
		edge.From("cargo_item", OrderCargoItem.Type).Ref("shared_container_allocations").Field("cargo_item_id").Unique().Required(),
	}
}

func (SeaSharedContainerAllocation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("shared_container_id", "cargo_item_id").Unique().StorageKey("sea_shared_cntr_alloc_unique"),
		index.Fields("organization_id", "shared_container_id"),
		index.Fields("organization_id", "order_id"),
		index.Fields("organization_id", "house_bill_id"),
	}
}
