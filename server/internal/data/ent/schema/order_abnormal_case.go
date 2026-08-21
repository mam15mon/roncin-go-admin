package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderAbnormalCase 定义订单异常标记。
// 异常类型引用组织级 abnormal_case 主数据；标记与解决人取会话主体，不接收客户端输入。
type OrderAbnormalCase struct{ ent.Schema }

func (OrderAbnormalCase) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderAbnormalCase) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("order_id", uuid.Nil),
		field.UUID("abnormal_case_id", uuid.Nil),
		field.Enum("status").Values("ACTIVE", "RESOLVED").Default("ACTIVE"),
		field.Time("marked_at").Default(time.Now),
		field.UUID("marked_by", uuid.Nil),
		field.Time("resolved_at").Optional().Nillable(),
		field.UUID("resolved_by", uuid.Nil).Optional().Nillable(),
	}
}

func (OrderAbnormalCase) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).Ref("abnormal_cases").Field("order_id").Unique().Required(),
	}
}

func (OrderAbnormalCase) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "abnormal_case_id").Unique(),
		index.Fields("order_id", "status", "marked_at"),
	}
}
