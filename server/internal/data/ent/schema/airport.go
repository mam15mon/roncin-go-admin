package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Airport 定义 IATA 与 ICAO 标准标识的机场。
// B 型基线+本地：organization_id 为空表示集团基线行（全网可见），非空表示本组织
// 差异化行；同码时本组织行覆盖基线行（Shadowing）。
type Airport struct{ ent.Schema }

func (Airport) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (Airport) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Optional().Nillable(),
		field.String("iata_code").NotEmpty().MaxLen(3).Immutable(),
		field.String("icao_code").MaxLen(4).Optional().Nillable(),
		field.String("name_zh").MaxLen(200).Optional().Nillable(),
		field.String("name_en").NotEmpty().MaxLen(200),
		field.String("city_name_zh").MaxLen(100).Optional().Nillable(),
		field.String("city_name_en").MaxLen(100).Optional().Nillable(),
		field.String("country_code").NotEmpty().MaxLen(2),
		field.String("source").MaxLen(100).Default("manual"),
		field.String("source_version").MaxLen(100).Optional().Nillable(),
		field.String("source_hash").MaxLen(64).Optional().Nillable(),
		field.Int("sort_order").Default(100),
		field.Bool("enabled").Default(true),
		searchKeywordsField(),
	}
}

func (Airport) Hooks() []ent.Hook {
	return []ent.Hook{searchKeywordsHook("name_zh", "name_en", "city_name_zh", "city_name_en")}
}

func (Airport) Edges() []ent.Edge {
	return []ent.Edge{edge.From("organization", Organization.Type).Ref("airports").Field("organization_id").Unique()}
}

func (Airport) Indexes() []ent.Index {
	return []ent.Index{
		// 基线约束：NULL 行业务码全局唯一；本地约束：同组织内业务码唯一。
		index.Fields("iata_code").Unique().StorageKey("airports_baseline_iata_unique").Annotations(entsql.IndexWhere("organization_id IS NULL")),
		index.Fields("organization_id", "iata_code").Unique().StorageKey("airports_org_iata_unique").Annotations(entsql.IndexWhere("organization_id IS NOT NULL")),
		index.Fields("icao_code").Unique().StorageKey("airports_baseline_icao_unique").Annotations(entsql.IndexWhere("organization_id IS NULL AND icao_code IS NOT NULL")),
		index.Fields("organization_id", "enabled", "sort_order"),
	}
}
