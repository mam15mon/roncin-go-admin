package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// SeaSharedContainer 定义海运出口跨订单共享集装箱（客户拼货例外）。
type SeaSharedContainer struct{ ent.Schema }

func (SeaSharedContainer) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (SeaSharedContainer) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil),
		field.UUID("transport_execution_id", uuid.Nil),
		field.String("container_no").NotEmpty().MaxLen(64),
		field.UUID("container_spec_id", uuid.Nil),
		field.String("seal_no").Optional().Nillable().MaxLen(64),
		field.Int("package_count").Positive(),
		// 使用字符串承接 PostgreSQL numeric，避免跨订单守恒计算经过 float64 丢失精度。
		field.String("gross_weight_kg").NotEmpty().SchemaType(map[string]string{dialect.Postgres: "numeric(18,3)"}),
		field.String("volume_cbm").NotEmpty().SchemaType(map[string]string{dialect.Postgres: "numeric(18,6)"}),
		field.Enum("status").Values("DRAFT", "CONFIRMED").Default("DRAFT"),
		field.Time("confirmed_at").Optional().Nillable(),
		field.UUID("confirmed_by", uuid.Nil).Optional().Nillable(),
		field.String("note").Optional().Nillable().MaxLen(500),
		field.Uint64("version").Default(1),
	}
}

func (SeaSharedContainer) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("sea_shared_containers").Field("organization_id").Unique().Required(),
		edge.From("transport_execution", SeaTransportExecution.Type).Ref("shared_containers").Field("transport_execution_id").Unique().Required().Annotations(entsql.OnDelete(entsql.NoAction)),
		edge.To("allocations", SeaSharedContainerAllocation.Type),
		edge.From("confirmed_by_user", User.Type).Ref("confirmed_sea_shared_containers").Field("confirmed_by").Unique(),
	}
}

func (SeaSharedContainer) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("transport_execution_id", "container_no").Unique().StorageKey("sea_shared_container_execution_no"),
		index.Fields("organization_id", "transport_execution_id"),
		index.Fields("organization_id", "container_spec_id"),
	}
}
