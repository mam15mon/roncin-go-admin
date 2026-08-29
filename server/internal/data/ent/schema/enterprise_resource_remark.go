package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// EnterpriseResourceRemark 保存业务备注资源详情。
type EnterpriseResourceRemark struct{ ent.Schema }

func (EnterpriseResourceRemark) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (EnterpriseResourceRemark) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("resource_id", uuid.Nil).Immutable(),
		field.Enum("remark_type").Values("BOOKING", "ALLOCATION", "TRANSPORT", "ORDER", "BILL_OF_LADING", "CUSTOMER", "SUPPLIER", "FOREIGN_AGENT", "QUOTATION", "MANIFEST", "PACKING_LIST", "OPERATION", "COMMISSION", "WAREHOUSE"),
		field.String("content").NotEmpty().MaxLen(4000),
	}
}

func (EnterpriseResourceRemark) Edges() []ent.Edge {
	return []ent.Edge{edge.From("resource", EnterpriseResource.Type).Ref("remark").Field("resource_id").Unique().Required().Immutable()}
}

func (EnterpriseResourceRemark) Indexes() []ent.Index {
	return []ent.Index{index.Fields("resource_id").Unique(), index.Fields("remark_type")}
}
