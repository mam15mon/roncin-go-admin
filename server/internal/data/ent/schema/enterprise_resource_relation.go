package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// EnterpriseResourcePartner 保存资源与企业的多对多关联。
type EnterpriseResourcePartner struct{ ent.Schema }

func (EnterpriseResourcePartner) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (EnterpriseResourcePartner) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("resource_id", uuid.Nil).Immutable(),
		field.UUID("partner_id", uuid.Nil).Immutable(),
		field.Enum("resource_type").Values(enterpriseResourceTypes...).Immutable(),
		field.Bool("is_default").Default(false),
	}
}

func (EnterpriseResourcePartner) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("resource", EnterpriseResource.Type).Ref("partner_links").Field("resource_id").Unique().Required().Immutable(),
		edge.From("partner", Partner.Type).Ref("enterprise_resource_links").Field("partner_id").Unique().Required().Immutable(),
	}
}

func (EnterpriseResourcePartner) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("resource_id", "partner_id").Unique(),
		index.Fields("partner_id", "resource_type"),
	}
}

// EnterpriseResourceAssignee 保存地址资源与内部用户的关联。
type EnterpriseResourceAssignee struct{ ent.Schema }

func (EnterpriseResourceAssignee) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (EnterpriseResourceAssignee) Fields() []ent.Field {
	return []ent.Field{field.UUID("resource_id", uuid.Nil).Immutable(), field.UUID("user_id", uuid.Nil).Immutable()}
}

func (EnterpriseResourceAssignee) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("resource", EnterpriseResource.Type).Ref("assignees").Field("resource_id").Unique().Required().Immutable(),
		edge.From("user", User.Type).Ref("enterprise_resource_assignments").Field("user_id").Unique().Required().Immutable(),
	}
}

func (EnterpriseResourceAssignee) Indexes() []ent.Index {
	return []ent.Index{index.Fields("resource_id", "user_id").Unique(), index.Fields("user_id")}
}

// EnterpriseResourceAddressType 保存地址资源的固定用途。
type EnterpriseResourceAddressType struct{ ent.Schema }

func (EnterpriseResourceAddressType) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (EnterpriseResourceAddressType) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("resource_id", uuid.Nil).Immutable(),
		field.Enum("address_type").Values("CONTAINER_OPERATION", "PICKUP", "DELIVERY").Immutable(),
	}
}

func (EnterpriseResourceAddressType) Edges() []ent.Edge {
	return []ent.Edge{edge.From("resource", EnterpriseResource.Type).Ref("address_types").Field("resource_id").Unique().Required().Immutable()}
}

func (EnterpriseResourceAddressType) Indexes() []ent.Index {
	return []ent.Index{index.Fields("resource_id", "address_type").Unique(), index.Fields("address_type")}
}
