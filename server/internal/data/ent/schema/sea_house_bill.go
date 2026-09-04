package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// SeaHouseBill 定义海运出口真实 HBL 分单。
type SeaHouseBill struct{ ent.Schema }

func (SeaHouseBill) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (SeaHouseBill) Fields() []ent.Field {
	fields := []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.UUID("order_id", uuid.Nil),
		field.UUID("master_bill_id", uuid.Nil),
		field.String("house_no").NotEmpty().MaxLen(128),
		field.String("normalized_house_no").NotEmpty().MaxLen(128),
		field.Enum("issuer_source").Values("SELF_ORGANIZATION", "CUSTOMER_PARTNER", "OTHER_PARTNER"),
		field.UUID("issuer_organization_id", uuid.Nil).Optional().Nillable(),
		field.UUID("issuer_partner_id", uuid.Nil).Optional().Nillable(),
		field.Enum("status").Values("DRAFT", "CONFIRMED", "RELEASED", "VOIDED", "REPLACED").Default("DRAFT"),
		field.UUID("current_version_id", uuid.Nil).Optional().Nillable(),
		field.Uint64("version").Default(1),
		field.String("note").Optional().Nillable().MaxLen(500),
	}
	return append(fields, seaBillContentFields()...)
}

func (SeaHouseBill) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("sea_house_bills").Field("organization_id").Unique().Required(),
		edge.From("order", Order.Type).Ref("sea_house_bills").Field("order_id").Unique().Required(),
		edge.From("master_bill", SeaMasterBill.Type).Ref("house_bills").Field("master_bill_id").Unique().Required(),
		edge.From("issuer_organization", Organization.Type).Ref("issued_sea_house_bills").Field("issuer_organization_id").Unique(),
		edge.From("issuer_partner", Partner.Type).Ref("issued_sea_house_bills").Field("issuer_partner_id").Unique(),
		edge.To("cargo_allocations", SeaCargoAllocation.Type),
		edge.To("current_version", SeaHouseBillVersion.Type).Field("current_version_id").Unique(),
		edge.To("versions", SeaHouseBillVersion.Type),
		edge.To("lock_snapshots", OrderLockHouseBillSnapshot.Type),
		edge.To("void_events", SeaDocumentVoidEvent.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
		edge.To("old_switch_events", SeaHouseBillSwitchEvent.Type),
		edge.To("new_switch_events", SeaHouseBillSwitchEvent.Type),
	}
}

func (SeaHouseBill) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "order_id"),
		index.Fields("organization_id", "master_bill_id"),
		index.Fields("organization_id", "issuer_organization_id", "normalized_house_no").
			Unique().
			StorageKey("idx_sea_house_bills_self_org_unique").
			Annotations(entsql.IndexWhere("issuer_source = 'SELF_ORGANIZATION'")),
		index.Fields("organization_id", "issuer_partner_id", "normalized_house_no").
			Unique().
			StorageKey("idx_sea_house_bills_partner_unique").
			Annotations(entsql.IndexWhere("issuer_source IN ('CUSTOMER_PARTNER', 'OTHER_PARTNER')")),
	}
}
