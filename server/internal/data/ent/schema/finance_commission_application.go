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

// FinanceCommissionApplication 保存员工在当前组织内按提交自然月形成的月度提成
// 汇总申请头：员工一次提交覆盖截至上一自然月末（coverage_to）的全部合格未申请
// 提成，财务对整张申请执行批准或驳回。
// 状态机：PENDING_REVIEW（财务待审）→ APPROVED / REJECTED；驳回后只能在本申请
// 上重新提交，不产生替代申请。唯一索引「组织 + 员工 + 申请月」含被驳回行，
// 结构上禁止同一组织、同一员工、同一提交月出现第二张申请（含并行或替代申请）。
// 申请头与明细为提交当时的不可变快照：金额、覆盖月份与明细集合只能随整单重提
// 重新形成，不做行级修改；提交人是永久审计事实（NO ACTION），决策人是可逆审计
// 引用（SET NULL）。
type FinanceCommissionApplication struct{ ent.Schema }

func (FinanceCommissionApplication) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceCommissionApplication) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Checks(map[string]string{
			// 申请状态取值；PENDING_REVIEW 只能进入 APPROVED 或 REJECTED 由
			// 状态机与仓储校验保证。
			"finance_commission_applications_status_check": "status IN ('PENDING_REVIEW', 'REJECTED', 'APPROVED')",
			// 汇总口径非负：申请只纳入合格提成事实，笔数与本位币/CNY 合计
			// 不允许出现负值。
			"finance_commission_applications_commission_count_non_negative":       "commission_count >= 0",
			"finance_commission_applications_total_commission_amount_non_negative": "total_commission_amount >= 0",
			"finance_commission_applications_total_cny_commission_amount_non_negative": "total_cny_commission_amount >= 0",
		}),
	}
}

func (FinanceCommissionApplication) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		// 员工是申请主体：与组织、申请月共同构成月度唯一键。
		field.UUID("employee_id", uuid.Nil).Immutable(),
		// 提交发生的自然月 YYYY-MM：月度唯一键组成部分，提交后不可变。
		field.String("application_month").NotEmpty().MinLen(7).MaxLen(7).Immutable(),
		// 覆盖截止日 YYYY-MM-DD：固定为提交月份前一自然月最后一天；晚于该日
		// 才满足计提条件的提成顺延到下一次申请，不回填本申请。
		field.String("coverage_to").NotEmpty().MinLen(10).MaxLen(10).Immutable(),
		field.Enum("status").Values("PENDING_REVIEW", "REJECTED", "APPROVED").Default("PENDING_REVIEW"),
		// 乐观锁版本：每次提交/重提、批准或驳回递增；财务审批携带
		// expected_version 防并发覆盖。
		field.Uint64("version").Default(1),
		// 提交时固化的汇总快照：明细笔数与申请头本位币口径合计；CNY 合计与
		// 提成单导出口径一致，便于财务汇总。
		field.Int("commission_count").NonNegative().Immutable(),
		field.String("base_currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("total_commission_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		field.String("total_cny_commission_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}).Immutable(),
		// 提交审计：申请行在提交事务内创建，提交人与时间为永久事实。
		field.Time("submitted_at").Immutable(),
		field.UUID("submitted_by", uuid.Nil).Immutable(),
		// 决策审计：批准或驳回的决策人/时间；驳回原因必填由业务层校验，列可空。
		field.Time("decided_at").Optional().Nillable(),
		field.UUID("decided_by", uuid.Nil).Optional().Nillable(),
		field.String("decision_reason").Optional().Nillable().MaxLen(500),
	}
}

func (FinanceCommissionApplication) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("finance_commission_applications").Field("organization_id").Unique().Required().Immutable(),
		edge.From("employee", User.Type).Ref("finance_commission_applications").Field("employee_id").Unique().Required().Immutable(),
		edge.From("submitted_by_user", User.Type).Ref("submitted_finance_commission_applications").Field("submitted_by").Unique().Required().Immutable(),
		// 决策人是可逆审计引用，删除用户时置空（SET NULL）即可。
		edge.From("decided_by_user", User.Type).Ref("decided_finance_commission_applications").Field("decided_by").Unique(),
		// 申请明细保留历史引用：删除申请不得级联清空明细快照，外键显式 NO ACTION。
		edge.To("lines", FinanceCommissionApplicationLine.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
	}
}

func (FinanceCommissionApplication) Indexes() []ent.Index {
	return []ent.Index{
		// 一人一组织一提交月一行（含被驳回行）：结构上禁止替代申请与并行申请，
		// 驳回后只能在本申请上重提。
		index.Fields("organization_id", "employee_id", "application_month").Unique(),
		// 本人工作台与财务列表按员工 + 状态定位申请批次。
		index.Fields("organization_id", "employee_id", "status"),
		// 财务按提交月过滤申请批次。
		index.Fields("organization_id", "application_month"),
	}
}
