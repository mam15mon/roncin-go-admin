package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type FinanceVerificationAllocation struct{ ent.Schema }

func (FinanceVerificationAllocation) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }
func (FinanceVerificationAllocation) Fields() []ent.Field {
	return []ent.Field{field.UUID("verification_id", uuid.Nil).Immutable(), field.UUID("cashflow_id", uuid.Nil).Immutable(), field.UUID("bill_id", uuid.Nil).Immutable(), field.String("cashflow_no").NotEmpty().MaxLen(64).Immutable(), field.String("bill_no").NotEmpty().MaxLen(64).Immutable(), field.String("amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(), field.Bool("active").Default(true)}
}
func (FinanceVerificationAllocation) Edges() []ent.Edge {
	return []ent.Edge{edge.From("verification", FinanceVerification.Type).Ref("allocations").Field("verification_id").Unique().Required().Immutable(), edge.From("cashflow", FinanceCashflow.Type).Ref("verification_allocations").Field("cashflow_id").Unique().Required().Immutable(), edge.From("bill", FinanceBill.Type).Ref("verification_allocations").Field("bill_id").Unique().Required().Immutable()}
}
func (FinanceVerificationAllocation) Indexes() []ent.Index {
	return []ent.Index{index.Fields("verification_id", "active"), index.Fields("cashflow_id", "active"), index.Fields("bill_id", "active")}
}
