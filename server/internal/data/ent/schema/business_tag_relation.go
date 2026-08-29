package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderEnterpriseTag 保存订单与组织标签的多对多关联。
type OrderEnterpriseTag struct{ ent.Schema }

func (OrderEnterpriseTag) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderEnterpriseTag) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("order_id", uuid.Nil).Immutable(),
		field.UUID("tag_resource_id", uuid.Nil).Immutable(),
	}
}

func (OrderEnterpriseTag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("order_enterprise_tags").Field("organization_id").Unique().Required().Immutable(),
		edge.From("order", Order.Type).Ref("enterprise_tag_links").Field("order_id").Unique().Required().Immutable(),
		edge.From("tag_resource", EnterpriseResource.Type).Ref("order_tag_links").Field("tag_resource_id").Unique().Required().Immutable(),
	}
}

func (OrderEnterpriseTag) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "tag_resource_id").Unique(),
		index.Fields("organization_id", "tag_resource_id"),
		index.Fields("order_id"),
	}
}

// OrderFeeEnterpriseTag 保存订单费用与组织标签的多对多关联。
type OrderFeeEnterpriseTag struct{ ent.Schema }

func (OrderFeeEnterpriseTag) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderFeeEnterpriseTag) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("order_fee_id", uuid.Nil).Immutable(),
		field.UUID("tag_resource_id", uuid.Nil).Immutable(),
	}
}

func (OrderFeeEnterpriseTag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("order_fee_enterprise_tags").Field("organization_id").Unique().Required().Immutable(),
		edge.From("order_fee", OrderFee.Type).Ref("enterprise_tag_links").Field("order_fee_id").Unique().Required().Immutable(),
		edge.From("tag_resource", EnterpriseResource.Type).Ref("order_fee_tag_links").Field("tag_resource_id").Unique().Required().Immutable(),
	}
}

func (OrderFeeEnterpriseTag) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_fee_id", "tag_resource_id").Unique(),
		index.Fields("organization_id", "tag_resource_id"),
		index.Fields("order_fee_id"),
	}
}

// FinanceBillEnterpriseTag 保存财务账单与组织标签的多对多关联。
type FinanceBillEnterpriseTag struct{ ent.Schema }

func (FinanceBillEnterpriseTag) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceBillEnterpriseTag) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("finance_bill_id", uuid.Nil).Immutable(),
		field.UUID("tag_resource_id", uuid.Nil).Immutable(),
	}
}

func (FinanceBillEnterpriseTag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("finance_bill_enterprise_tags").Field("organization_id").Unique().Required().Immutable(),
		edge.From("finance_bill", FinanceBill.Type).Ref("enterprise_tag_links").Field("finance_bill_id").Unique().Required().Immutable(),
		edge.From("tag_resource", EnterpriseResource.Type).Ref("finance_bill_tag_links").Field("tag_resource_id").Unique().Required().Immutable(),
	}
}

func (FinanceBillEnterpriseTag) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("finance_bill_id", "tag_resource_id").Unique(),
		index.Fields("organization_id", "tag_resource_id"),
		index.Fields("finance_bill_id"),
	}
}
