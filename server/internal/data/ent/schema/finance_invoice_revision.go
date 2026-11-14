package schema

import (
	"encoding/json"
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceInvoiceRevision 是发票修订的前后快照，仅追加，不覆盖历史。
type FinanceInvoiceRevision struct{ ent.Schema }

func (FinanceInvoiceRevision) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }
func (FinanceInvoiceRevision) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("invoice_id", uuid.Nil).Immutable(), field.UUID("actor_id", uuid.Nil).Immutable(),
		field.UUID("command_id", uuid.Nil).Immutable(),
		field.Uint64("revision_number").Immutable(),
		field.String("reason").NotEmpty().MaxRuneLen(500).Immutable(),
		field.JSON("before_snapshot", json.RawMessage{}).Immutable(), field.JSON("after_snapshot", json.RawMessage{}).Immutable(),
	}
}
func (FinanceInvoiceRevision) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("invoice", FinanceInvoice.Type).Ref("revisions").Field("invoice_id").Unique().Required().Immutable(),
	}
}
func (FinanceInvoiceRevision) Indexes() []ent.Index {
	return []ent.Index{index.Fields("invoice_id", "revision_number").Unique()}
}
