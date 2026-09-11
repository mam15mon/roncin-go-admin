package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// ExchangeRateSetting 是总部维护的折本币基准汇率主数据。
// 汇率以十进制字符串持久化到 PostgreSQL numeric，避免浮点误差。
type ExchangeRateSetting struct{ ent.Schema }

func (ExchangeRateSetting) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (ExchangeRateSetting) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.String("from_currency").NotEmpty().MinLen(3).MaxLen(3),
		field.String("to_currency").NotEmpty().MinLen(3).MaxLen(3),
		field.Time("effective_from"),
		field.Time("effective_to").Optional().Nillable(),
		field.String("rate").SchemaType(map[string]string{dialect.Postgres: "numeric(18,8)"}),
		field.Bool("is_active").Default(true),
	}
}

func (ExchangeRateSetting) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "from_currency", "to_currency", "effective_from").Unique().StorageKey("exchange_rate_setting_unique_effective_from"),
		index.Fields("organization_id", "from_currency", "to_currency", "is_active").StorageKey("exchange_rate_setting_active_lookup"),
		index.Fields("organization_id", "effective_from", "effective_to").StorageKey("exchange_rate_setting_effective_range"),
	}
}
