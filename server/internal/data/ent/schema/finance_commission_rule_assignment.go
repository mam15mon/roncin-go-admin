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

// FinanceCommissionRuleAssignment 保存提成方案与员工的有效期分配关系：
// 一个方案可分配多名员工，员工退出方案或离开组织不删除历史分配——未生效分配
// 以 cancelled_at/cancelled_by 撤销，已生效分配以 effective_to 终止并记录
// terminated_at/terminated_by；实际有效区间为方案区间与分配区间的闭区间交集。
// 分配起始日必填且创建后不可变，与「组织 + 方案 + 员工 + 起始日」唯一索引共同
// 构成分配段的身份；方案、员工、组织与创建人外键一律 NO ACTION，禁止删除
// 造成历史断链。
type FinanceCommissionRuleAssignment struct{ ent.Schema }

func (FinanceCommissionRuleAssignment) Mixin() []ent.Mixin {
	return []ent.Mixin{IDMixin{}, TimeMixin{}}
}

func (FinanceCommissionRuleAssignment) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Checks(map[string]string{
			// 有效期闭区间合法性：终止日为空视为正无穷，存在时不得早于起始日，
			// 禁止写入起止倒置的无效区间。
			"finance_commission_rule_assignments_effective_period_check": "effective_to IS NULL OR effective_from <= effective_to",
			// 撤销审计一致性：撤销时间与撤销操作者必须成对出现。
			"finance_commission_rule_assignments_cancel_audit_check": "(cancelled_at IS NULL AND cancelled_by IS NULL) OR (cancelled_at IS NOT NULL AND cancelled_by IS NOT NULL)",
			// 终止审计一致性：终止时间与终止操作者必须成对出现。
			"finance_commission_rule_assignments_terminate_audit_check": "(terminated_at IS NULL AND terminated_by IS NULL) OR (terminated_at IS NOT NULL AND terminated_by IS NOT NULL)",
		}),
	}
}

func (FinanceCommissionRuleAssignment) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("rule_id", uuid.Nil).Immutable(),
		field.UUID("employee_id", uuid.Nil).Immutable(),
		// 分配段起始日必填且不可变：YYYY-MM-DD 闭区间口径，与方案区间取交集后
		// 参与唯一性判断。
		field.String("effective_from").NotEmpty().MinLen(10).MaxLen(10).Immutable(),
		// 终止日可空：为空表示尚未终止（正无穷）；移除已生效分配时写入变更日前一日。
		field.String("effective_to").Optional().Nillable().MinLen(10).MaxLen(10),
		// 未生效分配的撤销标记：不引入状态枚举，保留审计。
		field.Time("cancelled_at").Optional().Nillable(),
		field.UUID("cancelled_by", uuid.Nil).Optional().Nillable(),
		// 已生效分配的终止审计：记录以 effective_to 退出方案的操作者与时间。
		field.Time("terminated_at").Optional().Nillable(),
		field.UUID("terminated_by", uuid.Nil).Optional().Nillable(),
		// 创建人是永久审计事实，删除用户不得清空分配历史。
		field.UUID("created_by", uuid.Nil).Immutable(),
	}
}

func (FinanceCommissionRuleAssignment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("finance_commission_rule_assignments").Field("organization_id").Unique().Required().Immutable(),
		edge.From("rule", FinanceCommissionRule.Type).Ref("assignments").Field("rule_id").Unique().Required().Immutable(),
		edge.From("employee", User.Type).Ref("finance_commission_rule_assignments").Field("employee_id").Unique().Required().Immutable(),
		edge.From("created_by_user", User.Type).Ref("created_finance_commission_rule_assignments").Field("created_by").Unique().Required().Immutable(),
		// 撤销/终止操作者是可逆审计引用，删除用户时置空（SET NULL）即可。
		edge.From("cancelled_by_user", User.Type).Ref("cancelled_finance_commission_rule_assignments").Field("cancelled_by").Unique(),
		edge.From("terminated_by_user", User.Type).Ref("terminated_finance_commission_rule_assignments").Field("terminated_by").Unique(),
	}
}

func (FinanceCommissionRuleAssignment) Indexes() []ent.Index {
	return []ent.Index{
		// 同一方案、员工和分配起始日唯一：允许员工退出后在另一不重叠区间重新
		// 加入同一方案，但禁止重复登记同一区间的起点。
		index.Fields("organization_id", "rule_id", "employee_id", "effective_from").Unique(),
		// 候选与门禁查询：按组织 + 员工定位当前/未来分配（成员停用前检查、
		// 计提候选解析与工作台资格判定）。
		index.Fields("organization_id", "employee_id", "effective_to"),
		// 方案名单查询：按方案定位未结束的分配段。
		index.Fields("rule_id", "effective_to"),
	}
}
