package schema

import (
	"encoding/json"
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"time"
)

// OrderImportHandlingEvent 保存本票进口承接服务不可变变更历史。
type OrderImportHandlingEvent struct{ ent.Schema }

func (OrderImportHandlingEvent) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }
func (OrderImportHandlingEvent) Fields() []ent.Field {
	return []ent.Field{field.UUID("order_id", uuid.Nil).Immutable(), field.UUID("actor_id", uuid.Nil).Immutable(), field.Time("occurred_at").Default(time.Now).Immutable(), field.String("reason").Default("").MaxRuneLen(500).Immutable(), field.JSON("before", json.RawMessage{}).Immutable(), field.JSON("after", json.RawMessage{}).Immutable()}
}
func (OrderImportHandlingEvent) Edges() []ent.Edge {
	return []ent.Edge{edge.To("actor", User.Type).Field("actor_id").Unique().Required().Immutable().Annotations(entsql.OnDelete(entsql.NoAction)), edge.To("order", Order.Type).Field("order_id").Unique().Required().Immutable().Annotations(entsql.OnDelete(entsql.NoAction))}
}
func (OrderImportHandlingEvent) Indexes() []ent.Index {
	return []ent.Index{index.Fields("order_id", "occurred_at")}
}
