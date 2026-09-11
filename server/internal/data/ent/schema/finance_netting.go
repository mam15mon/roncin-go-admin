package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceNetting 表达同一组织、同一结算单位、同一账单币种的应收应付抵销事实。
// 原始账单继续分别承担发票与毛额审计；对冲单只保存双方抵销金额，不产生混合币种总额。
type FinanceNetting struct{ ent.Schema }

func (FinanceNetting) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceNetting) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.String("netting_no").NotEmpty().MaxLen(64).Immutable(),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.String("request_hash").NotEmpty().MaxLen(64).Immutable(),
		field.UUID("batch_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.Enum("status").Values("DRAFT", "CONFIRMED", "CANCELLED", "REVERSED").Default("DRAFT"),
		field.UUID("settlement_party_id", uuid.Nil).Immutable(),
		field.String("settlement_party_name").NotEmpty().MaxLen(200).Immutable(),
		field.String("currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("base_currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("base_currency_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("note").Optional().Nillable().MaxLen(500),
		field.Uint64("version").Default(1),
		field.Time("confirmed_at").Optional().Nillable(),
		field.UUID("confirmed_by", uuid.Nil).Optional().Nillable(),
		field.Time("cancelled_at").Optional().Nillable(),
		field.UUID("cancelled_by", uuid.Nil).Optional().Nillable(),
		field.String("cancellation_reason").Optional().Nillable().MaxLen(500),
		field.Time("reversed_at").Optional().Nillable(),
		field.UUID("reversed_by", uuid.Nil).Optional().Nillable(),
		field.String("reversal_reason").Optional().Nillable().MaxLen(500),
	}
}

func (FinanceNetting) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("finance_nettings").Field("organization_id").Unique().Required().Immutable(),
		edge.From("settlement_party", Partner.Type).Ref("finance_nettings").Field("settlement_party_id").Unique().Required().Immutable(),
		edge.From("batch", FinanceBillBatch.Type).Ref("nettings").Field("batch_id").Unique().Immutable(),
		edge.From("confirmed_by_user", User.Type).Ref("confirmed_finance_nettings").Field("confirmed_by").Unique(),
		edge.From("cancelled_by_user", User.Type).Ref("cancelled_finance_nettings").Field("cancelled_by").Unique(),
		edge.From("reversed_by_user", User.Type).Ref("reversed_finance_nettings").Field("reversed_by").Unique(),
		edge.To("allocations", FinanceNettingAllocation.Type),
	}
}

func (FinanceNetting) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "netting_no").Unique(),
		index.Fields("organization_id", "idempotency_key").Unique(),
		index.Fields("organization_id", "status", "created_at"),
		index.Fields("settlement_party_id", "currency"),
		index.Fields("batch_id"),
	}
}

func (FinanceNetting) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Checks(map[string]string{
		"financenetting_status_check":             "status IN ('DRAFT', 'CONFIRMED', 'CANCELLED', 'REVERSED')",
		"financenetting_amount_positive":          "amount > 0",
		"financenetting_base_amount_non_negative": "base_currency_amount >= 0",
	})}
}
