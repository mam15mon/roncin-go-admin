package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// NumberRule 定义组织内某类单据的编号格式。
type NumberRule struct{ ent.Schema }

func (NumberRule) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (NumberRule) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		// coload_house_bill 仅用于读取迁移后保留的历史配置，不再向领域层和 API 暴露。
		field.Enum("document_type").Values("order", "bill", "quotation", "write_off", "receipt_payment", "contract", "internal_reference", "customer_reference", "house_bill", "coload_house_bill", "invoice", "freight_rate"),
		field.String("prefix").MaxLen(32),
		field.Enum("date_format").Values("yyyyMMdd", "yyyyMM", "yyyy", "none").Default("yyyyMMdd"),
		field.Int("sequence_length").Default(4).Min(1).Max(12),
		field.Enum("reset_policy").Values("daily", "monthly", "yearly", "never").Default("daily"),
		field.Bool("enabled").Default(true),
	}
}

func (NumberRule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("number_rules").Field("organization_id").Unique().Required(),
		edge.To("sequences", NumberSequence.Type),
	}
}

func (NumberRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "document_type").Unique(),
		index.Fields("organization_id", "enabled"),
	}
}
