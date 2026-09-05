package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OrderReleasePod 定义订单放货凭证。
// 状态仅允许 PENDING→SIGNED→RETURNED 向前流转；签收人与签收时间在签收流转时由会话主体记录。
type OrderReleasePod struct{ ent.Schema }

func (OrderReleasePod) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (OrderReleasePod) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("order_id", uuid.Nil),
		field.UUID("shipping_document_id", uuid.Nil).Optional().Nillable(),
		field.UUID("sea_master_bill_id", uuid.Nil).Optional().Nillable(),
		field.UUID("sea_house_bill_id", uuid.Nil).Optional().Nillable(),
		field.String("release_no").Optional().MaxLen(64),
		field.String("pod_no").Optional().MaxLen(64),
		field.Enum("status").Values("PENDING", "SIGNED", "RETURNED").Default("PENDING"),
		field.Time("signed_at").Optional().Nillable(),
		field.UUID("signed_by", uuid.Nil).Optional().Nillable(),
		field.String("note").Optional().MaxLen(500),
	}
}

func (OrderReleasePod) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).Ref("release_pods").Field("order_id").Unique().Required(),
		edge.From("shipping_document", OrderShippingDocument.Type).Ref("release_pods").Field("shipping_document_id").Unique(),
		edge.From("sea_master_bill", SeaMasterBill.Type).Ref("release_pods").Field("sea_master_bill_id").Unique(),
		edge.From("sea_house_bill", SeaHouseBill.Type).Ref("release_pods").Field("sea_house_bill_id").Unique(),
	}
}

func (OrderReleasePod) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "status"),
		index.Fields("shipping_document_id"),
		index.Fields("sea_master_bill_id"),
		index.Fields("sea_house_bill_id"),
	}
}

func (OrderReleasePod) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Checks(map[string]string{
		"order_release_pods_document_reference_check": "num_nonnulls(shipping_document_id, sea_master_bill_id, sea_house_bill_id) <= 1",
	})}
}
