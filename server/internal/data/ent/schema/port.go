package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Port 定义使用 UN/LOCODE 标识的海运港口。
// B 型基线+本地：organization_id 为空表示集团基线行（全网可见），非空表示本组织
// 差异化行；同码时本组织行覆盖基线行（Shadowing）。
type Port struct{ ent.Schema }

func (Port) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (Port) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Optional().Nillable(),
		field.String("un_locode").NotEmpty().MaxLen(5).Immutable(),
		field.String("name_zh").MaxLen(200).Optional().Nillable(),
		field.String("name_en").NotEmpty().MaxLen(200),
		field.String("country_code").NotEmpty().MaxLen(2),
		field.JSON("transport_modes", []string{}),
		field.String("source").MaxLen(100).Default("manual"),
		field.String("source_version").MaxLen(100).Optional().Nillable(),
		field.String("source_hash").MaxLen(64).Optional().Nillable(),
		field.Int("sort_order").Default(100),
		field.Bool("enabled").Default(true),
		searchKeywordsField(),
	}
}

func (Port) Hooks() []ent.Hook { return []ent.Hook{searchKeywordsHook("name_zh", "name_en")} }

func (Port) Edges() []ent.Edge {
	return []ent.Edge{edge.From("organization", Organization.Type).Ref("ports").Field("organization_id").Unique()}
}

func (Port) Indexes() []ent.Index {
	return []ent.Index{
		// 基线约束：NULL 行业务码全局唯一；本地约束：同组织内业务码唯一。
		index.Fields("un_locode").Unique().StorageKey("ports_baseline_locode_unique").Annotations(entsql.IndexWhere("organization_id IS NULL")),
		index.Fields("organization_id", "un_locode").Unique().StorageKey("ports_org_locode_unique").Annotations(entsql.IndexWhere("organization_id IS NOT NULL")),
		index.Fields("organization_id", "enabled", "sort_order"),
	}
}
