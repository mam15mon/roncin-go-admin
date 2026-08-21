package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Order 定义订单核心业务聚合。
type Order struct{ ent.Schema }

func (Order) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (Order) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.String("order_no").NotEmpty().MaxLen(64).Immutable(),
		field.UUID("customer_id", uuid.Nil),
		field.UUID("carrier_id", uuid.Nil).Optional().Nillable(),
		field.UUID("booking_agent_id", uuid.Nil).Optional().Nillable(),
		field.Enum("business_type").Values("SE", "SI", "AE", "AI", "LAND", "RAIL"),
		field.Enum("trade_direction").Values("export", "import"),
		field.Enum("trade_term").Values("EXW", "FCA", "FOB", "CFR", "CIF", "CPT", "CIP", "DAP", "DPU", "DDU", "DDP", "LDP"),
		field.Enum("payment_term").Values("PREPAID", "COLLECT"),
		field.Enum("shipment_type").Values("FCL", "LCL", "BREAK_BULK").Optional().Nillable(),
		field.Enum("container_ownership").Values("COC", "SOC").Optional().Nillable(),
		field.Enum("shipment_mode").Values("CONSOLIDATION", "CROSS_BORDER").Optional().Nillable(),
		field.String("status").NotEmpty().MaxLen(64).Default("DRAFT"),
		field.UUID("status_template_id", uuid.Nil),
		field.UUID("origin_location_id", uuid.Nil).Optional().Nillable(),
		field.UUID("destination_location_id", uuid.Nil).Optional().Nillable(),
		field.UUID("discharge_location_id", uuid.Nil).Optional().Nillable(),
		field.UUID("transit_location_id", uuid.Nil).Optional().Nillable(),
		field.String("vessel_voyage").Optional().MaxLen(100),
		field.String("etd").Optional().MaxLen(64),
		field.String("eta").Optional().MaxLen(64),
		field.String("si_cutoff").Optional().MaxLen(64),
		field.String("doc_cutoff").Optional().MaxLen(64),
		field.String("customs_cutoff").Optional().MaxLen(64),
		field.String("vgm_cutoff").Optional().MaxLen(64),
		field.String("goods_description").Optional().MaxLen(1000),
		field.Int("total_packages").Optional().Nillable(),
		field.String("total_package_unit").Optional().MaxLen(32),
		field.String("special_requirements").Optional().MaxLen(1000),
		field.String("order_date").Optional().MaxLen(32),
		field.String("notes").Optional().MaxLen(1000),
	}
}

func (Order) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("orders").Field("organization_id").Unique().Required(),
		edge.From("customer", Partner.Type).Ref("orders").Field("customer_id").Unique().Required(),
		edge.From("status_template", StatusTemplate.Type).Ref("orders").Field("status_template_id").Unique().Required(),
		edge.To("status_logs", OrderStatusLog.Type),
		edge.To("service_types", OrderServiceType.Type),
		edge.To("cargo_categories", OrderCargoCategory.Type),
		edge.To("milestones", OrderMilestone.Type),
		edge.To("attachments", OrderAttachment.Type),
		edge.To("personnel", OrderPersonnel.Type),
		edge.To("containers", OrderContainer.Type),
		edge.To("cargo_items", OrderCargoItem.Type),
		edge.To("shipping_documents", OrderShippingDocument.Type),
	}
}

func (Order) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "order_no").Unique(),
		index.Fields("organization_id", "status"),
		index.Fields("organization_id", "business_type"),
		index.Fields("organization_id", "customer_id"),
	}
}
