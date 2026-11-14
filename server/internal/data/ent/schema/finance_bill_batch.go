package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceBillBatch 只保存一次批量建账的身份：拆单策略、幂等键与请求摘要。
// 普通模式逐组独立提交、可部分成功，费用数、账单数与本位币合计在读取时按账单聚合。
type FinanceBillBatch struct{ ent.Schema }

func (FinanceBillBatch) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceBillBatch) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.String("batch_no").NotEmpty().MaxLen(64).Immutable(),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.String("request_hash").NotEmpty().MinLen(64).MaxLen(64).Immutable(),
		field.Bool("split_by_order").Default(false).Immutable(),
		field.Bool("split_by_tax_rate").Default(false).Immutable(),
		field.Enum("grouping_mode").Values("NORMAL", "NETTING").Default("NORMAL").Immutable(),
		field.UUID("created_by", uuid.Nil).Immutable(),
	}
}

func (FinanceBillBatch) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("finance_bill_batches").Field("organization_id").Unique().Required().Immutable(),
		edge.From("creator", User.Type).Ref("created_finance_bill_batches").Field("created_by").Unique().Required().Immutable(),
		edge.To("bills", FinanceBill.Type),
		edge.To("nettings", FinanceNetting.Type),
	}
}

func (FinanceBillBatch) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "batch_no").Unique(),
		index.Fields("organization_id", "idempotency_key").Unique(),
		index.Fields("organization_id", "created_at"),
	}
}

func (FinanceBillBatch) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Checks(map[string]string{"finance_bill_batches_grouping_mode_check": "grouping_mode IN ('NORMAL', 'NETTING')"})}
}
