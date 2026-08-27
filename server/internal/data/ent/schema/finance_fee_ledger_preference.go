package schema

import (
	"encoding/json"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceFeeLedgerPreference 保存用户在费用明细台账中的个性化列表配置。
type FinanceFeeLedgerPreference struct{ ent.Schema }

func (FinanceFeeLedgerPreference) Mixin() []ent.Mixin {
	return []ent.Mixin{IDMixin{}, TimeMixin{}}
}

func (FinanceFeeLedgerPreference) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("user_id", uuid.Nil).Immutable(),
		field.JSON("columns", json.RawMessage{}),
		field.Int("page_size"),
		field.String("sort_field").MaxLen(64).Optional().Nillable(),
		field.Enum("sort_direction").Values("ASC", "DESC").Optional().Nillable(),
		field.JSON("row_colors", json.RawMessage{}),
		field.Uint64("version").Default(1),
	}
}

func (FinanceFeeLedgerPreference) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("finance_fee_ledger_preferences").Field("organization_id").Unique().Required().Immutable(),
		edge.From("user", User.Type).Ref("finance_fee_ledger_preferences").Field("user_id").Unique().Required().Immutable(),
	}
}

func (FinanceFeeLedgerPreference) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "user_id").Unique(),
		index.Fields("user_id"),
	}
}
