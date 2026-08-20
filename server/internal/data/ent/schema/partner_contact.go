package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// PartnerContact 定义往来单位联系人。
type PartnerContact struct{ ent.Schema }

func (PartnerContact) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (PartnerContact) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("partner_id", uuid.Nil),
		field.String("name").NotEmpty().MaxLen(100),
		field.String("phone").Optional().MaxLen(64),
		field.String("email").Optional().MaxLen(254),
		field.String("note").Optional().MaxLen(500),
		field.Bool("is_primary").Default(false),
	}
}

func (PartnerContact) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("partner", Partner.Type).Ref("contacts").Field("partner_id").Unique().Required(),
	}
}

func (PartnerContact) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("partner_id"),
		index.Fields("partner_id").Unique().StorageKey("partner_primary_contact_key").Annotations(entsql.IndexWhere("is_primary")),
	}
}
