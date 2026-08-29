package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

var enterpriseResourceTypes = []string{
	"ADDRESS", "REMARK", "IMAGE", "TAG", "SHIPPER", "CONSIGNEE", "NOTIFY_PARTY",
	"ENGLISH_CARGO_NAME", "HS_CODE", "MARKS",
}

// EnterpriseResource 保存组织级可复用资源的公共字段。
type EnterpriseResource struct{ ent.Schema }

func (EnterpriseResource) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (EnterpriseResource) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.Enum("resource_type").Values(enterpriseResourceTypes...).Immutable(),
		field.String("short_name").NotEmpty().MaxLen(200),
		field.Bool("enabled").Default(true),
		field.Int("sort_order").Default(0).NonNegative(),
		field.UUID("created_by", uuid.Nil).Optional().Nillable().Immutable(),
		field.UUID("updated_by", uuid.Nil).Optional().Nillable(),
		searchKeywordsField(),
	}
}

func (EnterpriseResource) Hooks() []ent.Hook { return []ent.Hook{searchKeywordsHook("short_name")} }

func (EnterpriseResource) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("enterprise_resources").Field("organization_id").Unique().Required().Immutable(),
		edge.From("creator", User.Type).Ref("created_enterprise_resources").Field("created_by").Unique().Immutable(),
		edge.From("updater", User.Type).Ref("updated_enterprise_resources").Field("updated_by").Unique(),
		edge.To("address", EnterpriseResourceAddress.Type).Unique(),
		edge.To("remark", EnterpriseResourceRemark.Type).Unique(),
		edge.To("image", EnterpriseResourceImage.Type).Unique(),
		edge.To("party", EnterpriseResourceParty.Type).Unique(),
		edge.To("shipping_text", EnterpriseResourceShippingText.Type).Unique(),
		edge.To("tag", EnterpriseTag.Type).Unique(),
		edge.To("partner_links", EnterpriseResourcePartner.Type),
		edge.To("assignees", EnterpriseResourceAssignee.Type),
		edge.To("address_types", EnterpriseResourceAddressType.Type),
		edge.To("order_tag_links", OrderEnterpriseTag.Type),
		edge.To("order_fee_tag_links", OrderFeeEnterpriseTag.Type),
		edge.To("finance_bill_tag_links", FinanceBillEnterpriseTag.Type),
	}
}

func (EnterpriseResource) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "resource_type", "enabled", "sort_order"),
		index.Fields("organization_id", "updated_at"),
	}
}
