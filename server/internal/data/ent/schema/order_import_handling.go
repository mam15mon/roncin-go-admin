package schema

import (
	"encoding/json"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderImportHandling 保存本票进口承接服务当前事实。
type OrderImportHandling struct{ ent.Schema }

func (OrderImportHandling) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }
func (OrderImportHandling) Fields() []ent.Field {
	return []ent.Field{field.UUID("order_id", uuid.Nil).Immutable(), field.JSON("state", json.RawMessage{})}
}
func (OrderImportHandling) Edges() []ent.Edge {
	return []ent.Edge{edge.To("order", Order.Type).Field("order_id").Unique().Required().Immutable().Annotations(entsql.OnDelete(entsql.NoAction))}
}
func (OrderImportHandling) Indexes() []ent.Index {
	return []ent.Index{index.Fields("order_id").Unique()}
}
