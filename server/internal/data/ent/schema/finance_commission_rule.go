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

// FinanceCommissionRule 定义组织内可复用的提成方案：保存人员身份、计提口径、
// 比例、生效区间、启用状态和版本；适用员工通过 FinanceCommissionRuleAssignment
// 有效期分配关联，不在方案字段中保存员工 ID 数组。
type FinanceCommissionRule struct{ ent.Schema }

func (FinanceCommissionRule) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceCommissionRule) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.String("name").NotEmpty().MaxLen(100),
		field.Enum("personnel_role").Values("SALES", "OPERATOR", "CUSTOMER_SERVICE"),
		field.Enum("calculation_basis").Values("REALIZED_PROFIT", "REALIZED_REVENUE"),
		field.String("rate_percent").SchemaType(map[string]string{dialect.Postgres: "numeric(7,4)"}),
		field.String("effective_from").Optional().Nillable().MinLen(10).MaxLen(10),
		field.String("effective_to").Optional().Nillable().MinLen(10).MaxLen(10),
		field.Bool("enabled").Default(true),
		// 迁移停用的无分配旧角色规则为历史只读方案：只能复制为新方案，禁止
		// 重新启用或补挂员工；新建方案恒为 false。
		field.Bool("legacy_readonly").Default(false),
		field.String("note").Optional().Nillable().MaxLen(500),
		field.Uint64("version").Default(1),
	}
}

func (FinanceCommissionRule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("finance_commission_rules").Field("organization_id").Unique().Required().Immutable(),
		edge.To("commissions", FinanceCommission.Type),
		// 员工分配必须保留历史引用：方案不可删除造成孤儿分配，外键显式 NO ACTION。
		edge.To("assignments", FinanceCommissionRuleAssignment.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
	}
}

func (FinanceCommissionRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "name").Unique(),
		index.Fields("organization_id", "enabled", "personnel_role"),
	}
}
