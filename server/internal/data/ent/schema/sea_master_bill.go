package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// SeaMasterBill 定义海运出口共享 MBL。
type SeaMasterBill struct{ ent.Schema }

func (SeaMasterBill) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (SeaMasterBill) Fields() []ent.Field {
	fields := []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.UUID("shipping_line_id", uuid.Nil),
		field.String("master_no").NotEmpty().MaxLen(64),
		field.String("normalized_master_no").NotEmpty().MaxLen(64),
		field.Enum("status").Values("DRAFT", "CONFIRMED", "RELEASED", "VOIDED").Default("DRAFT"),
		field.UUID("current_version_id", uuid.Nil).Optional().Nillable(),
		field.Uint64("version").Default(1),
	}
	return append(fields, seaBillContentFields()...)
}

func (SeaMasterBill) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("sea_master_bills").Field("organization_id").Unique().Required(),
		edge.From("shipping_line", ShippingLine.Type).Ref("sea_master_bills").Field("shipping_line_id").Unique().Required(),
		edge.To("order_links", SeaMasterBillOrderLink.Type),
		edge.To("house_bills", SeaHouseBill.Type),
		edge.To("initial_sea_order_split_results", SeaOrderSplitResult.Type),
		edge.To("final_sea_order_split_results", SeaOrderSplitResult.Type),
		edge.To("previous_sea_order_reassignments", SeaOrderReassignmentEvent.Type),
		edge.To("target_sea_order_reassignments", SeaOrderReassignmentEvent.Type),
		edge.To("current_version", SeaMasterBillVersion.Type).Field("current_version_id").Unique(),
		edge.To("versions", SeaMasterBillVersion.Type),
		edge.To("house_bill_versions", SeaHouseBillVersion.Type),
		edge.To("lock_records", OrderLockRecord.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
		edge.To("void_events", SeaDocumentVoidEvent.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
		edge.To("release_pods", OrderReleasePod.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
	}
}

func (SeaMasterBill) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "shipping_line_id", "normalized_master_no").Unique(),
	}
}
