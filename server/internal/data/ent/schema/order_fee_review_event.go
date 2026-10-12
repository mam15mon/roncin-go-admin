package schema

import (
	"encoding/json"
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderFeeReviewEvent 只追加的整票费用审核事实；快照不随费用删除或账单关联变更。
type OrderFeeReviewEvent struct{ ent.Schema }

func (OrderFeeReviewEvent) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }
func (OrderFeeReviewEvent) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("order_id", uuid.Nil).Immutable(),
		field.Enum("action").Values("SUBMIT", "WITHDRAW", "APPROVE", "REJECT", "REVOKE", "SUPPLEMENT_APPROVE").Immutable(),
		field.UUID("submission_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("actor_id", uuid.Nil).Immutable(),
		field.String("actor_name").Default("").Immutable(),
		field.JSON("authorization_snapshot", map[string]string{}).Immutable(),
		field.String("comment").Default("").MaxLen(1000).Immutable(),
		field.Uint64("content_version").Immutable(),
		field.String("content_fingerprint").NotEmpty().MaxLen(64).Immutable(),
		field.JSON("snapshot", json.RawMessage{}).Immutable(),
		field.UUID("supplement_request_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.String("request_fingerprint").NotEmpty().MaxLen(64).Immutable(),
	}
}
func (OrderFeeReviewEvent) Edges() []ent.Edge {
	return []ent.Edge{edge.From("order", Order.Type).Ref("fee_review_events").Field("order_id").Unique().Required().Immutable()}
}
func (OrderFeeReviewEvent) Indexes() []ent.Index {
	return []ent.Index{index.Fields("order_id", "idempotency_key").Unique(), index.Fields("organization_id", "order_id", "created_at")}
}
func (OrderFeeReviewEvent) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Checks(map[string]string{"order_fee_review_events_action_check": "action IN ('SUBMIT','WITHDRAW','APPROVE','REJECT','REVOKE','SUPPLEMENT_APPROVE')"})}
}
