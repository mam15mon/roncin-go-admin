package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// TaxableService 定义组织级货物或应税劳务名称目录。
type TaxableService struct{ ent.Schema }

func (TaxableService) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (TaxableService) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.String("name").NotEmpty().MaxLen(128),
		field.String("short_name").Optional().Nillable().MaxLen(64),
		field.String("goods_code").Optional().Nillable().MaxLen(64),
		field.String("default_tax_rate").SchemaType(map[string]string{dialect.Postgres: "numeric(5,2)"}),
		field.Bool("enabled").Default(true),
	}
}

func (TaxableService) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("taxable_services").Field("organization_id").Unique().Required(),
		edge.To("fee_settings", FeeSetting.Type),
	}
}

func (TaxableService) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "name").Unique(),
		index.Fields("organization_id", "enabled", "name"),
	}
}
