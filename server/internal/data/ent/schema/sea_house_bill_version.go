package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// SeaHouseBillVersion 记录海运出口 HBL 的不可变快照版本。
type SeaHouseBillVersion struct{ ent.Schema }

func (SeaHouseBillVersion) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }

func (SeaHouseBillVersion) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Checks(map[string]string{
			"sea_house_bill_versions_issuer_check": "((issuer_source = 'SELF_ORGANIZATION' AND issuer_organization_id IS NOT NULL AND issuer_partner_id IS NULL) OR (issuer_source IN ('CUSTOMER_PARTNER', 'OTHER_PARTNER') AND issuer_organization_id IS NULL AND issuer_partner_id IS NOT NULL))",
		}),
	}
}

func (SeaHouseBillVersion) Fields() []ent.Field {
	fields := []ent.Field{
		field.Time("created_at").Immutable().Default(time.Now),
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("house_bill_id", uuid.Nil).Immutable(),
		field.UUID("order_id", uuid.Nil).Immutable(),
		field.UUID("master_bill_id", uuid.Nil).Immutable(),
		field.Uint64("version_no").Immutable(),
		field.Uint64("source_entity_version").Immutable(),
		field.String("house_no").NotEmpty().MaxLen(128).Immutable(),
		field.String("normalized_house_no").NotEmpty().MaxLen(128).Immutable(),
		field.Enum("issuer_source").Values("SELF_ORGANIZATION", "CUSTOMER_PARTNER", "OTHER_PARTNER").Immutable(),
		field.UUID("issuer_organization_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("issuer_partner_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.Enum("status").Values("DRAFT", "CONFIRMED", "RELEASED", "VOIDED", "REPLACED").Immutable(),
		field.String("note").Optional().Nillable().MaxLen(500).Immutable(),
		field.String("content_hash").NotEmpty().MaxLen(64).Immutable(),
		field.Enum("source").Values("ORDER_LOCK", "AMENDMENT", "SWITCH", "VOID").Immutable(),
		field.String("reason").Optional().Nillable().MaxLen(500).Immutable(),
		field.UUID("created_by", uuid.Nil).Optional().Nillable().Immutable(),
	}
	return append(fields, immutableSeaBillContentFields()...)
}

func (SeaHouseBillVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("sea_house_bill_versions").Field("organization_id").Unique().Required().Immutable(),
		edge.From("house_bill", SeaHouseBill.Type).Ref("versions").Field("house_bill_id").Unique().Required().Immutable(),
		edge.From("issuer_organization", Organization.Type).Ref("issued_sea_house_bill_versions").Field("issuer_organization_id").Unique().Immutable(),
		edge.From("issuer_partner", Partner.Type).Ref("sea_house_bill_versions").Field("issuer_partner_id").Unique().Immutable(),
		edge.From("order", Order.Type).Ref("sea_house_bill_versions").Field("order_id").Unique().Required().Immutable(),
		edge.From("master_bill", SeaMasterBill.Type).Ref("house_bill_versions").Field("master_bill_id").Unique().Required().Immutable(),
		edge.From("creator", User.Type).Ref("created_sea_house_bill_versions").Field("created_by").Unique().Immutable(),
		edge.To("lock_snapshots", OrderLockHouseBillSnapshot.Type),
		edge.To("void_events", SeaDocumentVoidEvent.Type),
		edge.To("old_switch_events", SeaHouseBillSwitchEvent.Type),
		edge.To("new_switch_events", SeaHouseBillSwitchEvent.Type),
	}
}

func (SeaHouseBillVersion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("house_bill_id", "version_no").Unique().StorageKey("sea_hbl_version_house_version_no"),
		index.Fields("house_bill_id", "source_entity_version", "content_hash").Unique().StorageKey("sea_hbl_version_source_hash"),
		index.Fields("organization_id", "house_bill_id"),
		index.Fields("organization_id", "order_id"),
	}
}
