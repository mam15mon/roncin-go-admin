package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// ExchangeRateTimeStandard 保存每种汇率类型有序启用的取值时间标准。
type ExchangeRateTimeStandard struct{ ent.Schema }

func (ExchangeRateTimeStandard) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (ExchangeRateTimeStandard) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.Enum("rate_type").Values("BASE_CURRENCY", "INVOICE", "SETTLEMENT", "WRITE_OFF", "BILL"),
		field.Enum("time_standard").Values("ETD_ETA_TRAIN_DATE", "BUSINESS_TIME", "BARGE_ETD", "EXPENSE_TIME", "ORDER_CREATED_AT", "BILL_DATE", "INVOICE_DATE", "TRANSACTION_DATE", "WRITE_OFF_TIME"),
		field.Int("sort_order").NonNegative(),
	}
}

func (ExchangeRateTimeStandard) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "rate_type", "time_standard").Unique().StorageKey("exchange_rate_time_standard_unique"),
		index.Fields("organization_id", "rate_type", "sort_order").Unique().StorageKey("exchange_rate_time_standard_sort_unique"),
	}
}
