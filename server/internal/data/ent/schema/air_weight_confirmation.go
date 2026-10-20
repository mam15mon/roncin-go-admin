package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// AirWeightConfirmation 保存不可变的空运计费重确认及可复现输入；不自动改写订单或费用。
type AirWeightConfirmation struct{ ent.Schema }

func (AirWeightConfirmation) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }
func (AirWeightConfirmation) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.Enum("scope").Values("ORDER", "MASTER").Immutable(),
		field.UUID("target_id", uuid.Nil).Immutable(),
		field.String("source").Default("ACTUAL").Immutable(),
		field.Uint64("version").Immutable(),
		field.String("input_fingerprint").MaxLen(64).Immutable(),
		field.JSON("snapshot", map[string]string{}).Immutable(),
		field.String("confirmed_weight_kg").SchemaType(map[string]string{dialect.Postgres: "numeric(18,4)"}).Immutable(),
		field.String("reason").MaxLen(500).Default("").Immutable(),
		field.UUID("confirmed_by", uuid.Nil).Immutable(),
	}
}
func (AirWeightConfirmation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "scope", "target_id", "source", "version").Unique().StorageKey("airweightconfirmation_org_scope_target_source_version"),
		index.Fields("organization_id", "scope", "target_id", "source", "created_at").StorageKey("airweightconfirmation_org_scope_target_source_created_at"),
	}
}

func (AirWeightConfirmation) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Checks(map[string]string{
		"air_weight_confirmation_scope_check":    "scope IN ('ORDER','MASTER')",
		"air_weight_confirmation_source_check":   "source IN ('PLANNED','ACTUAL')",
		"air_weight_confirmation_positive_check": "confirmed_weight_kg > 0 AND version > 0",
	})}
}
