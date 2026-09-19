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

// FinanceCommission 保存某员工在指定订单期间内的客户、订单利润和提成汇总快照。
// 来源二选一：核销（verification_id）或对冲（netting_id），恰好一个非空，由服务层校验。
// 身份与生命周期列（来源 ID、员工、方案版本锚点、状态机字段）创建后不可变；
// 计算快照列（方案、金额、指纹、CNY 折算等）允许申请重提事务在来源事实变化时
// 按计提引擎重算结果整体刷新并递增版本——刷新只发生在 DRAFT 且归属申请重提的
// 受控路径，CONFIRMED/PAID 后不再刷新。
type FinanceCommission struct{ ent.Schema }

func (FinanceCommission) Mixin() []ent.Mixin { return []ent.Mixin{IDMixin{}, TimeMixin{}} }

func (FinanceCommission) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.Nil).Immutable(),
		field.String("commission_no").NotEmpty().MaxLen(64).Immutable(),
		field.String("idempotency_key").NotEmpty().MaxLen(128).Immutable(),
		field.UUID("verification_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.String("verification_no").Optional().Nillable().MaxLen(64),
		field.UUID("netting_id", uuid.Nil).Optional().Nillable().Immutable(),
		field.String("netting_no").Optional().Nillable().MaxLen(64),
		field.UUID("employee_id", uuid.Nil).Immutable(),
		field.String("employee_name").NotEmpty().MaxLen(100),
		field.Int("customer_count").NonNegative(),
		field.Int("order_count").NonNegative(),
		field.Int("fee_count").NonNegative(),
		field.UUID("rule_id", uuid.Nil).Optional().Nillable(),
		field.String("rule_name").Optional().Nillable().MaxLen(100),
		field.String("personnel_role").Optional().Nillable().MaxLen(20).Immutable(),
		field.String("calculation_basis").Optional().Nillable().MaxLen(30),
		field.Uint64("rule_version").Default(1),
		field.String("calculation_version").NotEmpty().MaxLen(32).Default("CUSTOMER_REALIZED_PROFIT_V2"),
		field.String("source_fingerprint").MaxLen(64).Default(""),
		field.Enum("status").Values("DRAFT", "CONFIRMED", "PAID", "CANCELLED").Default("DRAFT"),
		field.String("base_currency").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.String("realized_revenue").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}),
		field.String("allocated_cost").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}),
		field.String("realized_profit").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}),
		field.String("commission_base_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}),
		field.String("rate_percent").SchemaType(map[string]string{dialect.Postgres: "numeric(7,4)"}),
		field.String("commission_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}),
		field.String("commission_date").NotEmpty().MinLen(10).MaxLen(10),
		field.String("cny_exchange_rate").SchemaType(map[string]string{dialect.Postgres: "numeric(18,8)"}),
		field.Enum("cny_exchange_rate_source").Values("BASE_CURRENCY", "DERIVED"),
		field.String("cny_exchange_rate_date").NotEmpty().MinLen(10).MaxLen(10),
		field.UUID("cny_exchange_rate_setting_id", uuid.Nil).Optional().Nillable(),
		field.String("cny_commission_amount").SchemaType(map[string]string{dialect.Postgres: "numeric(28,8)"}),
		field.Uint64("adjustment_sequence").Default(0),
		field.String("note").Optional().Nillable().MaxLen(500),
		field.Uint64("version").Default(1),
		field.Time("confirmed_at").Optional().Nillable(),
		field.UUID("confirmed_by", uuid.Nil).Optional().Nillable(),
		field.Time("paid_at").Optional().Nillable(),
		field.UUID("paid_by", uuid.Nil).Optional().Nillable(),
		field.Time("cancelled_at").Optional().Nillable(),
		field.UUID("cancelled_by", uuid.Nil).Optional().Nillable(),
		field.String("cancellation_reason").Optional().Nillable().MaxLen(500),
	}
}

func (FinanceCommission) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).Ref("finance_commissions").Field("organization_id").Unique().Required().Immutable(),
		edge.From("verification", FinanceVerification.Type).Ref("commissions").Field("verification_id").Unique().Immutable(),
		edge.From("netting", FinanceNetting.Type).Ref("commissions").Field("netting_id").Unique().Immutable(),
		edge.From("employee", User.Type).Ref("finance_commissions").Field("employee_id").Unique().Required().Immutable(),
		edge.From("rule", FinanceCommissionRule.Type).Ref("commissions").Field("rule_id").Unique(),
		edge.From("confirmed_by_user", User.Type).Ref("confirmed_finance_commissions").Field("confirmed_by").Unique(),
		edge.From("paid_by_user", User.Type).Ref("paid_finance_commissions").Field("paid_by").Unique(),
		edge.From("cancelled_by_user", User.Type).Ref("cancelled_finance_commissions").Field("cancelled_by").Unique(),
		edge.To("lines", FinanceCommissionLine.Type),
		edge.To("adjustments", FinanceCommissionAdjustment.Type),
		// 月度提成申请明细引用提成事实：外键 NO ACTION，删除提成单不得造成
		// 历史申请断链。
		edge.To("application_lines", FinanceCommissionApplicationLine.Type).Annotations(entsql.OnDelete(entsql.NoAction)),
	}
}

func (FinanceCommission) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "commission_no").Unique(),
		index.Fields("organization_id", "idempotency_key").Unique(),
		// 活跃去重按来源拆分：同来源同员工同角色仅一条非终态提成。
		index.Fields("organization_id", "verification_id", "employee_id", "personnel_role").
			StorageKey("finance_commissions_verification_active_unique").
			Unique().
			Annotations(entsql.IndexWhere("status <> 'CANCELLED' AND verification_id IS NOT NULL")),
		index.Fields("organization_id", "netting_id", "employee_id", "personnel_role").
			StorageKey("finance_commissions_netting_active_unique").
			Unique().
			Annotations(entsql.IndexWhere("status <> 'CANCELLED' AND netting_id IS NOT NULL")),
		index.Fields("verification_id", "employee_id", "status").
			StorageKey("financecommission_source_employee_status").
			Annotations(entsql.IndexWhere("verification_id IS NOT NULL")),
		index.Fields("netting_id", "employee_id", "status").
			StorageKey("financecommission_netting_employee_status").
			Annotations(entsql.IndexWhere("netting_id IS NOT NULL")),
		index.Fields("organization_id", "commission_date"),
		index.Fields("organization_id", "status", "created_at"),
	}
}
