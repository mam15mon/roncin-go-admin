package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
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
		field.String("customer_reference_no").Optional().MaxLen(100),
		field.String("internal_reference_no").Optional().MaxLen(100),
		field.String("shipper_short_name").Optional().MaxLen(200),
		field.String("consignee_short_name").Optional().MaxLen(200),
		field.UUID("carrier_id", uuid.Nil).Optional().Nillable(),
		field.UUID("booking_agent_id", uuid.Nil).Optional().Nillable(),
		field.UUID("foreign_agent_id", uuid.Nil).Optional().Nillable(),
		field.UUID("shipping_agent_id", uuid.Nil).Optional().Nillable(),
		field.String("contract_no").Optional().MaxLen(100),
		field.String("cargo_value").Optional().MaxLen(32),
		field.String("cargo_currency").Optional().MinLen(3).MaxLen(3),
		field.String("insurance_premium").Optional().MaxLen(32),
		field.String("insurance_currency").Optional().MinLen(3).MaxLen(3),
		field.String("un_number").Optional().MaxLen(4),
		field.String("hazard_class").Optional().MaxLen(16),
		field.String("factory_name").Optional().MaxLen(200),
		field.String("cargo_ready_at").Optional().MaxLen(64),
		field.String("loading_terms").Optional().MaxLen(100),
		field.String("declaration_cutoff_at").Optional().MaxLen(64),
		field.String("received_at").Optional().MaxLen(64),
		field.Enum("business_type").Values("SE", "SI", "AE", "AI", "LAND", "RAIL"),
		field.Enum("trade_direction").Values("export", "import"),
		field.Enum("trade_term").Values("EXW", "FCA", "FOB", "CFR", "CIF", "CPT", "CIP", "DAP", "DPU", "DDU", "DDP", "LDP"),
		field.Enum("payment_term").Values("PREPAID", "COLLECT"),
		field.Enum("shipment_type").Values("FCL", "LCL", "BREAK_BULK").Optional().Nillable(),
		field.Enum("container_ownership").Values("COC", "SOC").Optional().Nillable(),
		field.Enum("shipment_mode").Values("TRADITIONAL_FORWARDING", "CROSS_BORDER").Optional().Nillable(),
		field.Enum("flow_status").Values("DRAFT", "BOOKED", "SPACE_ALLOCATED", "TRUCKING_ARRANGED", "DOCUMENT_CUTOFF", "CUSTOMS_DECLARATION_ARRANGED", "DOCUMENT_RELEASED").Default("DRAFT"),
		field.Enum("termination_status").Values("ACTIVE", "TERMINATING", "TERMINATED").Default("ACTIVE"),
		field.Enum("termination_type").Values("CUSTOMER_CANCEL", "CARRIER_CANCEL", "CUSTOMS_RETURN", "OPERATION_CANCEL", "OTHER").Optional().Nillable(),
		field.String("termination_reason").Optional().Nillable().MaxLen(500),
		field.Time("terminated_at").Optional().Nillable(),
		field.UUID("terminated_by", uuid.Nil).Optional().Nillable(),
		field.Enum("closure_status").Values("OPEN", "CLOSED").Default("OPEN"),
		field.String("closure_reason").Optional().Nillable().MaxLen(500),
		field.Time("closed_at").Optional().Nillable(),
		field.UUID("closed_by", uuid.Nil).Optional().Nillable(),
		field.Time("locked_at").Optional().Nillable(),
		field.Bool("is_shared").Default(false),
		field.JSON("tags", []string{}).Default([]string{}),
		field.Uint64("version").Default(1),
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
		field.Float("total_gross_weight_kg").Optional().Nillable().Min(0),
		field.Float("total_volume_cbm").Optional().Nillable().Min(0),
		field.String("total_package_unit").Optional().MaxLen(32),
		field.String("special_requirements").Optional().MaxLen(1000),
		field.String("order_date").Optional().MaxLen(32),
		field.String("notes").Optional().MaxLen(1000),
		field.String("booking_notes").Optional().MaxLen(1000),
		field.String("allocation_notes").Optional().MaxLen(1000),
		field.String("operation_notes").Optional().MaxLen(1000),
	}
}

func (Order) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("orders").Field("organization_id").Unique().Required(),
		edge.From("customer", Partner.Type).Ref("orders").Field("customer_id").Unique().Required(),
		edge.To("lifecycle_events", OrderLifecycleEvent.Type),
		edge.To("service_types", OrderServiceType.Type),
		edge.To("cargo_categories", OrderCargoCategory.Type),
		edge.To("milestones", OrderMilestone.Type),
		edge.To("attachments", OrderAttachment.Type),
		edge.To("personnel", OrderPersonnel.Type),
		edge.To("containers", OrderContainer.Type),
		edge.To("container_requests", OrderContainerRequest.Type),
		edge.To("cargo_items", OrderCargoItem.Type),
		edge.To("shipping_documents", OrderShippingDocument.Type),
		edge.To("release_pods", OrderReleasePod.Type),
		edge.To("abnormal_cases", OrderAbnormalCase.Type),
		edge.To("fees", OrderFee.Type),
		edge.To("finance_bill_lines", FinanceBillLine.Type),
		edge.To("finance_commission_lines", FinanceCommissionLine.Type),
		edge.To("finance_commission_adjustments", FinanceCommissionAdjustment.Type),
		edge.To("commission_attributions", OrderCommissionAttribution.Type),
	}
}

func (Order) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "order_no").Unique(),
		index.Fields("organization_id", "flow_status"),
		index.Fields("organization_id", "termination_status"),
		index.Fields("organization_id", "closure_status"),
		index.Fields("organization_id", "business_type"),
		index.Fields("organization_id", "customer_id"),
		index.Fields("organization_id", "carrier_id"),
		index.Fields("organization_id", "origin_location_id"),
		index.Fields("organization_id", "destination_location_id"),
		index.Fields("organization_id", "locked_at"),
		index.Fields("organization_id", "is_shared"),
		index.Fields("tags").StorageKey("order_tags_gin").Annotations(entsql.IndexType("GIN")),
	}
}
