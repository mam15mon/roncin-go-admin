package schema

import (
	"entgo.io/ent"
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
		field.UUID("issuer_partner_id", uuid.Nil), // 实际签发 MBL 的船公司或上游 NVOCC
		field.UUID("transport_execution_id", uuid.Nil),
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
		edge.From("transport_execution", SeaTransportExecution.Type).Ref("master_bills").Field("transport_execution_id").Unique().Required(),
		edge.To("order_links", SeaMasterBillOrderLink.Type),
		edge.To("house_bills", SeaHouseBill.Type),
		edge.To("initial_sea_order_split_results", SeaOrderSplitResult.Type),
		edge.To("final_sea_order_split_results", SeaOrderSplitResult.Type),
		edge.To("previous_sea_order_reassignments", SeaOrderReassignmentEvent.Type),
		edge.To("target_sea_order_reassignments", SeaOrderReassignmentEvent.Type),
		edge.To("current_version", SeaMasterBillVersion.Type).Field("current_version_id").Unique(),
		edge.To("versions", SeaMasterBillVersion.Type),
		edge.To("house_bill_versions", SeaHouseBillVersion.Type),
		edge.To("lock_records", OrderLockRecord.Type),
		edge.To("void_events", SeaDocumentVoidEvent.Type),
		edge.To("switch_events", SeaHouseBillSwitchEvent.Type),
	}
}

func (SeaMasterBill) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "issuer_partner_id", "normalized_master_no").Unique(),
		index.Fields("organization_id", "transport_execution_id"),
	}
}
