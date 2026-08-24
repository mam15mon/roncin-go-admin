package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// ExchangeRateSetting 是组织级结算汇率主数据。
// 汇率以十进制字符串持久化到 PostgreSQL numeric，避免浮点误差。
type ExchangeRateSetting struct{ ent.Schema }

func (ExchangeRateSetting) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (ExchangeRateSetting) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.Enum("rate_type").Values("SETTLEMENT"),
		field.String("from_currency").NotEmpty().MinLen(3).MaxLen(3),
		field.String("to_currency").NotEmpty().MinLen(3).MaxLen(3),
		field.Enum("time_standard").Values("EXPENSE_DATE"),
		field.String("effective_from").NotEmpty().MinLen(10).MaxLen(10),
		field.String("effective_to").Optional().Nillable().MinLen(10).MaxLen(10),
		field.String("receivable_rate").SchemaType(map[string]string{dialect.Postgres: "numeric(18,8)"}),
		field.String("payable_rate").SchemaType(map[string]string{dialect.Postgres: "numeric(18,8)"}),
		field.Bool("is_active").Default(true),
	}
}

func (ExchangeRateSetting) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "rate_type", "from_currency", "to_currency", "time_standard", "effective_from").Unique(),
		index.Fields("organization_id", "rate_type", "from_currency", "to_currency", "time_standard", "is_active"),
		index.Fields("organization_id", "effective_from", "effective_to"),
	}
}
