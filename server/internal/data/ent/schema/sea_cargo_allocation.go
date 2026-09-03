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

// SeaCargoAllocation 定义海运出口操作票在活动 MBL 关系下的货物、分单、实际箱定量分配。
type SeaCargoAllocation struct{ ent.Schema }

func (SeaCargoAllocation) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (SeaCargoAllocation) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.UUID("order_id", uuid.Nil),
		field.UUID("master_bill_order_link_id", uuid.Nil),
		field.UUID("cargo_item_id", uuid.Nil),
		field.UUID("house_bill_id", uuid.Nil),
		field.UUID("container_id", uuid.Nil).Optional().Nillable(),
		field.Int("package_count").Positive(),
		field.String("gross_weight_kg").SchemaType(map[string]string{dialect.Postgres: "numeric(18,3)"}),
		field.String("volume_cbm").SchemaType(map[string]string{dialect.Postgres: "numeric(18,6)"}),
	}
}

func (SeaCargoAllocation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("sea_cargo_allocations").Field("organization_id").Unique().Required(),
		edge.From("order", Order.Type).Ref("sea_cargo_allocations").Field("order_id").Unique().Required(),
		edge.From("order_link", SeaMasterBillOrderLink.Type).Ref("cargo_allocations").Field("master_bill_order_link_id").Unique().Required(),
		edge.From("cargo_item", OrderCargoItem.Type).Ref("cargo_allocations").Field("cargo_item_id").Unique().Required(),
		edge.From("house_bill", SeaHouseBill.Type).Ref("cargo_allocations").Field("house_bill_id").Unique().Required(),
		edge.From("container", OrderContainer.Type).Ref("cargo_allocations").Field("container_id").Unique(),
	}
}

func (SeaCargoAllocation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "order_id"),
		index.Fields("master_bill_order_link_id"),
		index.Fields("cargo_item_id"),
		index.Fields("house_bill_id"),
		index.Fields("container_id"),
		index.Fields("master_bill_order_link_id", "cargo_item_id", "house_bill_id").
			Unique().
			StorageKey("idx_sea_cargo_allocations_no_cntr_unique").
			Annotations(entsql.IndexWhere("container_id IS NULL")),
		index.Fields("master_bill_order_link_id", "cargo_item_id", "house_bill_id", "container_id").
			Unique().
			StorageKey("idx_sea_cargo_allocations_cntr_unique").
			Annotations(entsql.IndexWhere("container_id IS NOT NULL")),
	}
}
