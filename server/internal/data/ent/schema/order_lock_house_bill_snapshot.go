package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderLockHouseBillSnapshot 记录锁定瞬间操作票下所有有效 HBL 版本快照。
type OrderLockHouseBillSnapshot struct{ ent.Schema }

func (OrderLockHouseBillSnapshot) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }

func (OrderLockHouseBillSnapshot) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Immutable().Default(time.Now),
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("lock_record_id", uuid.Nil).Immutable(),
		field.UUID("house_bill_id", uuid.Nil).Immutable(),
		field.UUID("house_bill_version_id", uuid.Nil).Immutable(),
		field.String("house_no_snapshot").NotEmpty().MaxLen(128).Immutable(),
	}
}

func (OrderLockHouseBillSnapshot) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("order_lock_house_bill_snapshots").Field("organization_id").Unique().Required().Immutable(),
		edge.From("lock_record", OrderLockRecord.Type).Ref("house_bill_snapshots").Field("lock_record_id").Unique().Required().Immutable(),
		edge.From("house_bill", SeaHouseBill.Type).Ref("lock_snapshots").Field("house_bill_id").Unique().Required().Immutable(),
		edge.From("house_bill_version", SeaHouseBillVersion.Type).Ref("lock_snapshots").Field("house_bill_version_id").Unique().Required().Immutable(),
	}
}

func (OrderLockHouseBillSnapshot) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("lock_record_id", "house_bill_id").Unique().StorageKey("order_lock_hbl_snapshot_unique"),
		index.Fields("organization_id", "lock_record_id"),
	}
}
