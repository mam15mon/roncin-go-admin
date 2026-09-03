package schema

import (
	"encoding/json"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// SeaOrderSplitResult 记录拆票事件中涉及的原票或新票结果明细。
type SeaOrderSplitResult struct{ ent.Schema }

func (SeaOrderSplitResult) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}} }

func (SeaOrderSplitResult) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Checks(map[string]string{
			"sea_order_split_results_result_role_check": "result_role IN ('ORIGINAL', 'CREATED')",
		}),
	}
}

func (SeaOrderSplitResult) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Immutable().Default(time.Now),
		field.UUID("split_event_id", uuid.Nil).Immutable(),
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("order_id", uuid.Nil).Immutable(),
		field.String("order_no").NotEmpty().MaxLen(64).Immutable(),
		field.Enum("result_role").Values("ORIGINAL", "CREATED").Immutable(),
		field.Int("sequence").Immutable(),
		field.String("client_result_key").NotEmpty().MaxLen(128).Immutable(),
		field.UUID("initial_master_bill_id", uuid.Nil).Immutable(),
		field.UUID("final_master_bill_id", uuid.Nil).Immutable(),
		field.JSON("result_snapshot", json.RawMessage{}).Immutable(),
	}
}

func (SeaOrderSplitResult) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("split_event", SeaOrderSplitEvent.Type).Ref("results").Field("split_event_id").Unique().Required().Immutable(),
		edge.From("organization", Organization.Type).Ref("sea_order_split_results").Field("organization_id").Unique().Required().Immutable(),
		edge.From("order", Order.Type).Ref("sea_order_split_results").Field("order_id").Unique().Required().Immutable(),
		edge.From("initial_master_bill", SeaMasterBill.Type).Ref("initial_sea_order_split_results").Field("initial_master_bill_id").Unique().Required().Immutable(),
		edge.From("final_master_bill", SeaMasterBill.Type).Ref("final_sea_order_split_results").Field("final_master_bill_id").Unique().Required().Immutable(),
		edge.To("reassignment_events", SeaOrderReassignmentEvent.Type),
	}
}

func (SeaOrderSplitResult) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("split_event_id", "sequence").Unique().StorageKey("sea_order_split_result_sequence"),
		index.Fields("split_event_id", "client_result_key").Unique().StorageKey("sea_order_split_result_client_key"),
		index.Fields("organization_id", "order_id"),
	}
}
