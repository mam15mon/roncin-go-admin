package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// SeaHouseBillSwitchEvent 记录海运出口 HBL 换单（Switch B/L）的不可变事件。
type SeaHouseBillSwitchEvent struct{ ent.Schema }

func (SeaHouseBillSwitchEvent) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }

func (SeaHouseBillSwitchEvent) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Immutable().Default(time.Now),
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("order_id", uuid.Nil).Immutable(),
		field.UUID("master_bill_id", uuid.Nil).Immutable(),
		field.UUID("chain_id", uuid.Nil).Immutable(),
		field.Int("sequence").Immutable().Positive(),
		field.UUID("old_house_bill_id", uuid.Nil).Immutable(),
		field.UUID("old_house_bill_version_id", uuid.Nil).Immutable(),
		field.UUID("new_house_bill_id", uuid.Nil).Immutable(),
		field.UUID("new_house_bill_version_id", uuid.Nil).Immutable(),
		field.String("reason").NotEmpty().MaxLen(500).Immutable(),
		field.String("surrender_info").Optional().Nillable().MaxLen(500).Immutable(),
		field.String("impact_summary").Optional().Nillable().MaxLen(1000).Immutable(),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.String("request_fingerprint").NotEmpty().MaxLen(128).Immutable(),
		field.UUID("created_by", uuid.Nil).Immutable(),
	}
}

func (SeaHouseBillSwitchEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("sea_house_bill_switch_events").Field("organization_id").Unique().Required().Immutable(),
		edge.From("order", Order.Type).Ref("sea_house_bill_switch_events").Field("order_id").Unique().Required().Immutable(),
		edge.From("master_bill", SeaMasterBill.Type).Ref("switch_events").Field("master_bill_id").Unique().Required().Immutable(),
		edge.From("old_house_bill", SeaHouseBill.Type).Ref("old_switch_events").Field("old_house_bill_id").Unique().Required().Immutable(),
		edge.From("old_house_bill_version", SeaHouseBillVersion.Type).Ref("old_switch_events").Field("old_house_bill_version_id").Unique().Required().Immutable(),
		edge.From("new_house_bill", SeaHouseBill.Type).Ref("new_switch_events").Field("new_house_bill_id").Unique().Required().Immutable(),
		edge.From("new_house_bill_version", SeaHouseBillVersion.Type).Ref("new_switch_events").Field("new_house_bill_version_id").Unique().Required().Immutable(),
		edge.From("creator", User.Type).Ref("created_sea_house_bill_switch_events").Field("created_by").Unique().Required().Immutable(),
	}
}

func (SeaHouseBillSwitchEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "idempotency_key").Unique().StorageKey("sea_hbl_switch_idempotency_key"),
		index.Fields("old_house_bill_id").Unique().StorageKey("sea_hbl_switch_old_hbl_unique"),
		index.Fields("chain_id", "sequence"),
		index.Fields("organization_id", "order_id"),
	}
}
