package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// ExchangeRateSetting 是折本币基准汇率主数据。
// B 型基线+本地：organization_id 为空表示集团基线汇率行（全网可见），非空表示
// 本组织行（本币核算优先命中）；汇率以十进制字符串持久化到 PostgreSQL numeric，
// 避免浮点误差。
type ExchangeRateSetting struct{ ent.Schema }

func (ExchangeRateSetting) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (ExchangeRateSetting) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Optional().Nillable(),
		field.String("from_currency").NotEmpty().MinLen(3).MaxLen(3),
		field.String("to_currency").NotEmpty().MinLen(3).MaxLen(3),
		field.Time("effective_from"),
		field.Time("effective_to").Optional().Nillable(),
		field.String("rate").SchemaType(map[string]string{dialect.Postgres: "numeric(18,8)"}),
		// 双轨点差：ar_rate=现汇卖出价（应收汇率），ap_rate=现汇买入价（应付汇率）；
		// rate 保留为基准价（中行折算价口径）。阶段一仅落存储结构，写入与解析在阶段二接入。
		field.String("ar_rate").SchemaType(map[string]string{dialect.Postgres: "numeric(18,8)"}).Optional().Nillable(),
		field.String("ap_rate").SchemaType(map[string]string{dialect.Postgres: "numeric(18,8)"}).Optional().Nillable(),
		field.Bool("is_active").Default(true),
	}
}

func (ExchangeRateSetting) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("from_currency", "to_currency", "effective_from").Unique().
			StorageKey("exchange_rate_setting_baseline_effective_from_unique").
			Annotations(entsql.IndexWhere("organization_id IS NULL")),
		index.Fields("organization_id", "from_currency", "to_currency", "effective_from").Unique().
			StorageKey("exchange_rate_setting_org_effective_from_unique").
			Annotations(entsql.IndexWhere("organization_id IS NOT NULL")),
		index.Fields("organization_id", "from_currency", "to_currency", "is_active").StorageKey("exchange_rate_setting_active_lookup"),
		index.Fields("organization_id", "effective_from", "effective_to").StorageKey("exchange_rate_setting_effective_range"),
	}
}
