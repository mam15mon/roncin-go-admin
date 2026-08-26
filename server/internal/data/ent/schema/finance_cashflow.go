package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceCashflow 记录企业真实收款或付款流水，核销前不改变账单结清状态。
type FinanceCashflow struct{ ent.Schema }

func (FinanceCashflow) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }
func (FinanceCashflow) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(), field.String("flow_no").NotEmpty().MaxLen(64).Immutable(), field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.Enum("direction").Values("RECEIVABLE", "PAYABLE").Immutable(), field.Enum("status").Values("DRAFT", "CONFIRMED", "CANCELLED").Default("DRAFT"),
		field.UUID("settlement_party_id", uuid.Nil).Immutable(), field.String("settlement_party_name").NotEmpty().MaxLen(200).Immutable(), field.String("currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(), field.String("exchange_rate").SchemaType(map[string]string{dialect.Postgres: "numeric(18,8)"}).Immutable(), field.String("base_currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(), field.String("base_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("transaction_date").NotEmpty().MinLen(10).MaxLen(10), field.String("our_account").NotEmpty().MaxLen(200), field.String("counterparty_account").Optional().Nillable().MaxLen(200), field.String("payment_method").NotEmpty().MaxLen(50), field.String("bank_reference_no").Optional().Nillable().MaxLen(100), field.String("note").Optional().Nillable().MaxLen(500), field.Uint64("version").Default(1),
		field.Time("confirmed_at").Optional().Nillable(), field.UUID("confirmed_by", uuid.Nil).Optional().Nillable(), field.Time("cancelled_at").Optional().Nillable(), field.UUID("cancelled_by", uuid.Nil).Optional().Nillable(), field.String("cancellation_reason").Optional().Nillable().MaxLen(500),
	}
}
func (FinanceCashflow) Edges() []ent.Edge {
	return []ent.Edge{edge.From("organization", Organization.Type).Ref("finance_cashflows").Field("organization_id").Unique().Required().Immutable(), edge.From("settlement_party", Partner.Type).Ref("finance_cashflows").Field("settlement_party_id").Unique().Required().Immutable(), edge.From("confirmed_by_user", User.Type).Ref("confirmed_finance_cashflows").Field("confirmed_by").Unique(), edge.From("cancelled_by_user", User.Type).Ref("cancelled_finance_cashflows").Field("cancelled_by").Unique(), edge.To("verification_allocations", FinanceVerificationAllocation.Type)}
}
func (FinanceCashflow) Indexes() []ent.Index {
	return []ent.Index{index.Fields("organization_id", "flow_no").Unique(), index.Fields("organization_id", "idempotency_key").Unique(), index.Fields("organization_id", "status", "transaction_date"), index.Fields("settlement_party_id", "direction", "currency"), index.Fields("bank_reference_no")}
}
