package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// PartnerContract 定义往来单位合同，日期和状态由领域层校验。
type PartnerContract struct{ ent.Schema }

func (PartnerContract) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (PartnerContract) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("partner_id", uuid.Nil),
		field.String("contract_no").NotEmpty().MaxLen(100).Immutable(),
		field.String("name").NotEmpty().MaxLen(200),
		field.Enum("status").Values("pending", "active", "expired", "terminated").Default("pending"),
		field.Time("start_date"),
		field.Time("end_date"),
		field.String("payment_terms").Optional().MaxLen(2000),
		field.String("dispute_resolution").Optional().MaxLen(2000),
		field.String("other_notes").Optional().MaxLen(2000),
	}
}

func (PartnerContract) Edges() []ent.Edge {
	return []ent.Edge{edge.From("partner", Partner.Type).Ref("contracts").Field("partner_id").Unique().Required()}
}

func (PartnerContract) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("partner_id", "contract_no").Unique().StorageKey("partner_contract_no_key"),
		index.Fields("partner_id", "status"),
		index.Fields("partner_id", "start_date", "end_date"),
	}
}
