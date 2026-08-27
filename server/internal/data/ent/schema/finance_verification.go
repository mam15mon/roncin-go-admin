package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type FinanceVerification struct{ ent.Schema }

func (FinanceVerification) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }
func (FinanceVerification) Fields() []ent.Field {
	return []ent.Field{field.UUID("organization_id", uuid.Nil).Immutable(), field.String("verification_no").NotEmpty().MaxLen(64).Immutable(), field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(), field.Enum("status").Values("ACTIVE", "REVERSED").Default("ACTIVE"), field.Enum("direction").Values("RECEIVABLE", "PAYABLE").Immutable(), field.UUID("settlement_party_id", uuid.Nil).Immutable(), field.String("settlement_party_name").NotEmpty().MaxLen(200).Immutable(), field.String("currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(), field.String("amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(), field.String("base_currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(), field.String("exchange_rate").SchemaType(map[string]string{dialect.Postgres: "numeric(18,8)"}).Immutable(), field.Enum("exchange_rate_source").Values("SYSTEM", "BASE_CURRENCY", "INHERITED_BASE_CURRENCY", "MANUAL", "DERIVED").Immutable(), field.String("exchange_rate_date").NotEmpty().MinLen(10).MaxLen(10).Immutable(), field.UUID("exchange_rate_setting_id", uuid.Nil).Optional().Nillable().Immutable(), field.String("base_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(), field.String("bill_base_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(), field.String("cashflow_base_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(), field.String("exchange_gain_loss").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(), field.String("verification_date").NotEmpty().MinLen(10).MaxLen(10).Immutable(), field.String("note").Optional().Nillable().MaxLen(500), field.Uint64("version").Default(1), field.Time("reversed_at").Optional().Nillable(), field.UUID("reversed_by", uuid.Nil).Optional().Nillable(), field.String("reversal_reason").Optional().Nillable().MaxLen(500)}
}
func (FinanceVerification) Edges() []ent.Edge {
	return []ent.Edge{edge.From("organization", Organization.Type).Ref("finance_verifications").Field("organization_id").Unique().Required().Immutable(), edge.From("settlement_party", Partner.Type).Ref("finance_verifications").Field("settlement_party_id").Unique().Required().Immutable(), edge.From("reversed_by_user", User.Type).Ref("reversed_finance_verifications").Field("reversed_by").Unique(), edge.To("allocations", FinanceVerificationAllocation.Type), edge.To("commissions", FinanceCommission.Type)}
}
func (FinanceVerification) Indexes() []ent.Index {
	return []ent.Index{index.Fields("organization_id", "verification_no").Unique(), index.Fields("organization_id", "idempotency_key").Unique(), index.Fields("organization_id", "status", "verification_date")}
}
