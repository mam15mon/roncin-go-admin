package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceInvoiceCommand 固化整次登记或修订的请求指纹和结果身份，修订票据后原请求仍可正确回放。
type FinanceInvoiceCommand struct{ ent.Schema }

func (FinanceInvoiceCommand) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }
func (FinanceInvoiceCommand) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(), field.UUID("actor_id", uuid.Nil).Immutable(),
		field.Enum("kind").Values("REGISTER", "REVISE").Immutable(),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.String("request_fingerprint").NotEmpty().MaxLen(64).Immutable(),
		field.JSON("invoice_ids", []uuid.UUID{}).Default([]uuid.UUID{}), field.Bool("completed").Default(false),
	}
}
func (FinanceInvoiceCommand) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("finance_invoice_commands").Field("organization_id").Unique().Required().Immutable(),
	}
}
func (FinanceInvoiceCommand) Indexes() []ent.Index {
	return []ent.Index{index.Fields("organization_id", "idempotency_key").Unique().StorageKey("finance_invoice_command_org_key")}
}
