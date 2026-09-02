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
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.UUID("issuer_partner_id", uuid.Nil), // 实际签发 MBL 的船公司或上游 NVOCC
		field.UUID("transport_execution_id", uuid.Nil),
		field.String("master_no").NotEmpty().MaxLen(64),
		field.String("normalized_master_no").NotEmpty().MaxLen(64),
		field.Enum("status").Values("DRAFT", "CONFIRMED", "RELEASED").Default("DRAFT"),
		field.Uint64("version").Default(1),
	}
}

func (SeaMasterBill) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("sea_master_bills").Field("organization_id").Unique().Required(),
		edge.From("transport_execution", SeaTransportExecution.Type).Ref("master_bills").Field("transport_execution_id").Unique().Required(),
		edge.To("order_links", SeaMasterBillOrderLink.Type),
	}
}

func (SeaMasterBill) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "issuer_partner_id", "normalized_master_no").Unique(),
		index.Fields("organization_id", "transport_execution_id"),
	}
}
