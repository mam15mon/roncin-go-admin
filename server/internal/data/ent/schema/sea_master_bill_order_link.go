package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// SeaMasterBillOrderLink 定义海运操作票与共享 MBL 的当前/历史关联关系。
type SeaMasterBillOrderLink struct{ ent.Schema }

func (SeaMasterBillOrderLink) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (SeaMasterBillOrderLink) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.UUID("master_bill_id", uuid.Nil),
		field.UUID("order_id", uuid.Nil),
		field.Enum("status").Values("ACTIVE", "ENDED").Default("ACTIVE"),
		field.Enum("document_structure").Values("UNDETERMINED", "DIRECT", "HOUSE").Default("UNDETERMINED"),
		field.Time("started_at").Default(time.Now),
		field.Time("ended_at").Optional().Nillable(),
		field.String("ended_reason").Optional().Nillable().MaxLen(255),
		field.Uint64("version").Default(1),
		field.Enum("cargo_allocation_status").Values("DRAFT", "CONFIRMED").Default("DRAFT"),
		field.Uint64("cargo_allocation_version").Default(1),
		field.Time("cargo_allocation_confirmed_at").Optional().Nillable(),
		field.UUID("cargo_allocation_confirmed_by", uuid.Nil).Optional().Nillable(),
	}
}

func (SeaMasterBillOrderLink) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("sea_master_bill_order_links").Field("organization_id").Unique().Required(),
		edge.From("master_bill", SeaMasterBill.Type).Ref("order_links").Field("master_bill_id").Unique().Required(),
		edge.From("order", Order.Type).Ref("sea_master_bill_links").Field("order_id").Unique().Required(),
		edge.To("cargo_allocations", SeaCargoAllocation.Type),
		edge.From("cargo_allocation_confirmed_by_user", User.Type).Ref("confirmed_sea_cargo_allocation_links").Field("cargo_allocation_confirmed_by").Unique(),
	}
}

func (SeaMasterBillOrderLink) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "master_bill_id"),
		index.Fields("organization_id", "order_id"),
		index.Fields("order_id").Unique().StorageKey("idx_sea_mbl_order_links_active_order").Annotations(entsql.IndexWhere("status = 'ACTIVE'")),
	}
}
