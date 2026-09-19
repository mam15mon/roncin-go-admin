package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// FinanceCommissionApplicationLine 保存月度提成申请明细：申请提交时逐笔固化的
// 提成事实快照（归属日期、来源核销/对冲、人员身份、方案与金额），全部字段创建
// 后不可变，申请批准后快照不可修改。
// 组织与员工冗余列与申请头一致，用于隔离查询与索引；提成归属日期与来源、方案、
// 金额快照按提交当时的事实固化，迟到或后续变化的历史来源不回填已提交申请。
// 外键全部 NO ACTION：删除组织、员工、申请头或提成单都不得造成历史申请断链；
// commission_id 全局唯一——一个提成事实至多进入一张申请，被驳回也不回流公共池。
type FinanceCommissionApplicationLine struct{ ent.Schema }

func (FinanceCommissionApplicationLine) Mixin() []ent.Mixin {
	return []ent.Mixin{IDMixin{}, TimeMixin{}}
}

func (FinanceCommissionApplicationLine) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Checks(map[string]string{
			// 明细金额快照非负：只纳入合格提成事实，与提成单金额口径一致。
			"finance_commission_application_lines_commission_amount_non_negative":       "commission_amount >= 0",
			"finance_commission_application_lines_cny_commission_amount_non_negative": "cny_commission_amount >= 0",
		}),
	}
}

func (FinanceCommissionApplicationLine) Fields() []ent.Field {
	return []ent.Field{
		// 组织与员工冗余列：与申请头一致，支撑本人/财务隔离查询与索引。
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.UUID("employee_id", uuid.Nil).Immutable(),
		field.UUID("application_id", uuid.Nil).Immutable(),
		// 一个提成事实至多进入一张申请：全局唯一索引兜底，被驳回仍占用原申请。
		field.UUID("commission_id", uuid.Nil).Immutable(),
		// 提成归属日期 YYYY-MM-DD：核销日期或对冲确认日期，与提成单口径一致。
		field.String("commission_date").NotEmpty().MinLen(10).MaxLen(10).Immutable(),
		// 来源核销/对冲快照：恰好一个非空由提成单创建路径保证，此处按事实固化。
		field.UUID("verification_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.String("verification_no").Optional().Nillable().MaxLen(64).Immutable(),
		field.UUID("netting_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.String("netting_no").Optional().Nillable().MaxLen(64).Immutable(),
		// 人员身份与方案快照：方案取值随提成单固化，自定义历史提成允许为空。
		field.String("personnel_role").NotEmpty().MaxLen(20).Immutable(),
		field.UUID("rule_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.Uint64("rule_version").Default(1).Immutable(),
		field.String("rule_name").Optional().Nillable().MaxLen(100).Immutable(),
		field.String("calculation_basis").Optional().Nillable().MaxLen(30).Immutable(),
		// 金额快照：组织本位币口径金额与 CNY 折算口径，随提交当时事实固化。
		field.String("base_currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("commission_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("cny_commission_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		// 提交当时来源指纹：批准复核据此确认来源仍为同一员工/身份/方案事实。
		field.String("source_fingerprint").MaxLen(64).Default("").Immutable(),
	}
}

func (FinanceCommissionApplicationLine) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("finance_commission_application_lines").Field("organization_id").Unique().Required().Immutable(),
		edge.From("employee", User.Type).Ref("finance_commission_application_lines").Field("employee_id").Unique().Required().Immutable(),
		edge.From("application", FinanceCommissionApplication.Type).Ref("lines").Field("application_id").Unique().Required().Immutable(),
		edge.From("commission", FinanceCommission.Type).Ref("application_lines").Field("commission_id").Unique().Required().Immutable(),
	}
}

func (FinanceCommissionApplicationLine) Indexes() []ent.Index {
	return []ent.Index{
		// 申请内明细与提成事实一一对应；application_id 前导同时支撑申请详情
		// 明细下钻查询，无需单独的 application_id 索引。
		index.Fields("application_id", "commission_id").Unique(),
		// 一个提成事实至多进入一张申请：跨申请、跨月份全局唯一，被驳回也不
		// 回流公共池。
		index.Fields("commission_id").Unique(),
		// 本人跨申请明细查询与组织隔离过滤。
		index.Fields("organization_id", "employee_id"),
	}
}
