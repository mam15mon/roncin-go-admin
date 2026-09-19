package biz

import (
	"context"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CommissionStatus string
type CommissionPersonnelRole string
type CommissionCalculationBasis string
type CommissionAdjustmentDirection string
type CommissionAdjustmentSourceType string

const (
	CommissionDraft                                CommissionStatus               = "DRAFT"
	CommissionConfirmed                            CommissionStatus               = "CONFIRMED"
	CommissionPaid                                 CommissionStatus               = "PAID"
	CommissionCancelled                            CommissionStatus               = "CANCELLED"
	CommissionRoleSales                            CommissionPersonnelRole        = "SALES"
	CommissionRoleOperator                         CommissionPersonnelRole        = "OPERATOR"
	CommissionRoleCustomerService                  CommissionPersonnelRole        = "CUSTOMER_SERVICE"
	CommissionBasisRealizedProfit                  CommissionCalculationBasis     = "REALIZED_PROFIT"
	CommissionBasisRealizedRevenue                 CommissionCalculationBasis     = "REALIZED_REVENUE"
	CommissionCalculationVersion                                                  = "CUSTOMER_REALIZED_PROFIT_V3"
	CommissionAdjustmentIncrease                   CommissionAdjustmentDirection  = "INCREASE"
	CommissionAdjustmentDecrease                   CommissionAdjustmentDirection  = "DECREASE"
	CommissionAdjustmentSourceManual               CommissionAdjustmentSourceType = "MANUAL"
	CommissionAdjustmentSourceVerificationReversal CommissionAdjustmentSourceType = "VERIFICATION_REVERSAL"
	CommissionAdjustmentSourceNettingReversal      CommissionAdjustmentSourceType = "NETTING_REVERSAL"
	// CommissionAdjustmentSourceLockedFeeSupplement 锁后费用补录的系统冲减建议
	// 来源，必须携带补录申请关联并由数据库 CHECK 强制双向对应。
	CommissionAdjustmentSourceLockedFeeSupplement CommissionAdjustmentSourceType = "LOCKED_FEE_SUPPLEMENT"
	cnyCurrency                                                                  = "CNY"
)

// FinanceCommissionAdjustment 以独立单据记录原始提成确认后的增提或冲减。
type FinanceCommissionAdjustment struct {
	ID, OrganizationID, CommissionID, OrderID, EmployeeID             uuid.UUID
	AdjustmentNo, IdempotencyKey, CommissionNo, OrderNo, EmployeeName string
	OrganizationName                                                  string
	Direction                                                         CommissionAdjustmentDirection
	SourceType                                                        CommissionAdjustmentSourceType
	SourceVerificationID                                              *uuid.UUID
	Status                                                            CommissionStatus
	BaseCurrency                                                      string
	Amount                                                            decimal.Decimal
	Reason                                                            string
	Note                                                              *string
	Version                                                           uint64
	ConfirmedAt, PaidAt, CancelledAt                                  *time.Time
	ConfirmedBy, PaidBy, CancelledBy                                  *uuid.UUID
	CancellationReason                                                *string
	CreatedAt, UpdatedAt                                              time.Time
}

// CommissionFeeDetail 是逐订单提成中的费用明细快照。
type CommissionFeeDetail struct {
	FeeID, SettlementPartyID                         uuid.UUID
	Direction, FeeCode, FeeName, SettlementPartyName string
	Currency, BaseCurrency, ExpenseDate, Status      string
	TotalAmount, ExchangeRate, BaseCurrencyAmount    decimal.Decimal
}

// FinanceCommissionLine 保存逐订单提成计算快照，保证客户归属与费用构成可追溯。
// TotalReceivableSnapshot / TotalPayableSnapshot 与 SnapshotStatus / SnapshotSource
// 是锁后费用补录审批的历史复算输入：新生成行固定写 READY + NATIVE，存量行由
// 迁移回填或标记 UNAVAILABLE；全空（SnapshotStatus 为空）不得参与复算。
type FinanceCommissionLine struct {
	ID, OrganizationID, CommissionID, OrderID, CustomerID, CustomerAssignmentID, CustomerAssignmentOrganizationID, EmployeeID uuid.UUID
	OrderNo, OrderDate, CustomerCode, CustomerName, EmployeeName, BaseCurrency                                                string
	PersonnelRole                                                                                                             CommissionPersonnelRole
	CalculationBasis                                                                                                          CommissionCalculationBasis
	RealizedRevenue, AllocatedCost, RealizedProfit, CommissionBaseAmount                                                      decimal.Decimal
	RatePercent, CommissionAmount                                                                                             decimal.Decimal
	TotalReceivableSnapshot, TotalPayableSnapshot                                                                             decimal.Decimal
	SnapshotStatus, SnapshotSource                                                                                            string
	CustomerAssignedAt, CreatedAt, UpdatedAt                                                                                  time.Time
	FeeCount                                                                                                                  int
	Fees                                                                                                                      []*CommissionFeeDetail
}

// CommissionLineSnapshotStatus / SnapshotSource 的取值，与数据库 CHECK 同源。
const (
	CommissionLineSnapshotReady       = "READY"
	CommissionLineSnapshotUnavailable = "UNAVAILABLE"
	CommissionLineSnapshotNative      = "NATIVE"
	CommissionLineSnapshotMigrated    = "MIGRATED"
)

// CommissionCNYRateSource 标识提成 CNY 折算率的来源口径。
const (
	CommissionCNYRateSourceBaseCurrency = "BASE_CURRENCY"
	CommissionCNYRateSourceDerived      = "DERIVED"
)

// CommissionCNYSnapshot 是提成生成时固化的 CNY 折算快照与归属日期，写入后不可变。
type CommissionCNYSnapshot struct {
	CommissionDate        string          // 归属日期，等于核销单 verification_date
	ExchangeRate          decimal.Decimal // 本位币折算到 CNY 的汇率（原币记账口径恒为 1）
	ExchangeRateSource    string          // BASE_CURRENCY（原币记账恒等口径）
	ExchangeRateDate      string          // 来源汇率日期
	ExchangeRateSettingID *uuid.UUID      // 被反算的原始 CNY→本位币汇率配置（原币记账后恒空）
	CommissionAmount      decimal.Decimal // 原始提成 CNY 快照
}

// ApplyCommissionAmount 按主单提成金额折算并固化原始 CNY 提成金额。
// 提成金额允许为零（亏损订单按零计提），汇率已在构造时校验为正。
func (s *CommissionCNYSnapshot) ApplyCommissionAmount(commissionAmount decimal.Decimal) {
	if s == nil {
		return
	}
	s.CommissionAmount = commissionAmount.Mul(s.ExchangeRate).Round(8)
}

// ResolveCommissionCNYRate 是预览与创建共用的 CNY 快照纯计算函数（原币记账口径）：
// 提成金额已按组织本位币核算，CNY 快照固化恒等口径（汇率 1、来源 BASE_CURRENCY），
// 不再经总部基线汇率二次折算——ResolveBaseRate 已退役，跨组织提成与往来按单据
// 原币记账（PRD R3.5），各组织本币管理口径如需折算用本组织汇率自行处理。
func ResolveCommissionCNYRate(baseCurrency, commissionDate string, commissionAmount decimal.Decimal) (*CommissionCNYSnapshot, error) {
	if baseCurrency == "" || !validFinanceDate(commissionDate) {
		return nil, ErrCommissionInvalid
	}
	return &CommissionCNYSnapshot{
		CommissionDate:     commissionDate,
		ExchangeRate:       decimal.NewFromInt(1),
		ExchangeRateSource: CommissionCNYRateSourceBaseCurrency,
		ExchangeRateDate:   commissionDate,
		CommissionAmount:   commissionAmount.Round(8),
	}, nil
}

// CommissionCalculation 是预览和创建提成共用的计算结果，不包含持久化状态。
// 来源二选一：VerificationID 或 NettingID 恰好一个非空（uuid.Nil 表示未提供）。
type CommissionCalculation struct {
	VerificationID, NettingID, EmployeeID, RuleID     uuid.UUID
	VerificationNo, NettingNo, EmployeeName, RuleName string
	BaseCurrency                                      string
	PersonnelRole                                     CommissionPersonnelRole
	CalculationBasis                                  CommissionCalculationBasis
	RuleVersion                                       uint64
	CalculationVersion, SourceFingerprint             string
	CustomerCount, OrderCount, FeeCount               int
	RealizedRevenue, AllocatedCost, RealizedProfit    decimal.Decimal
	CommissionBaseAmount                              decimal.Decimal
	RatePercent, CommissionAmount                     decimal.Decimal
	Lines                                             []*FinanceCommissionLine
	CNY                                               *CommissionCNYSnapshot
}

// FinanceCommission 保存提成快照；来源二选一：VerificationID/VerificationNo 或
// NettingID/NettingNo，恰好一组非空（uuid.Nil/空串表示未提供）。
type FinanceCommission struct {
	ID, OrganizationID, VerificationID, NettingID, EmployeeID, RuleID     uuid.UUID
	CommissionNo, IdempotencyKey, VerificationNo, NettingNo, EmployeeName string
	OrganizationName                                                      string
	RuleName                                                              string
	PersonnelRole                                                         CommissionPersonnelRole
	CalculationBasis                                                      CommissionCalculationBasis
	RuleVersion                                                           uint64
	CalculationVersion, SourceFingerprint                                 string
	Status                                                                CommissionStatus
	BaseCurrency                                                          string
	CustomerCount, OrderCount, FeeCount                                   int
	RealizedRevenue, AllocatedCost, RealizedProfit                        decimal.Decimal
	CommissionBaseAmount                                                  decimal.Decimal
	RatePercent, CommissionAmount                                         decimal.Decimal
	CommissionDate                                                        string
	CNYExchangeRate                                                       decimal.Decimal
	CNYExchangeRateSource, CNYExchangeRateDate                            string
	CNYExchangeRateSettingID                                              *uuid.UUID
	CNYCommissionAmount                                                   decimal.Decimal
	Note                                                                  *string
	Version                                                               uint64
	ConfirmedAt, PaidAt, CancelledAt                                      *time.Time
	ConfirmedBy, PaidBy, CancelledBy                                      *uuid.UUID
	CancellationReason                                                    *string
	CreatedAt, UpdatedAt                                                  time.Time
	Lines                                                                 []*FinanceCommissionLine
	Adjustments                                                           []*FinanceCommissionAdjustment
	AdjustmentAmount, EffectiveCommissionAmount                           decimal.Decimal
	CNYAdjustmentAmount, CNYEffectiveCommissionAmount                     decimal.Decimal
}

type CommissionFilter struct {
	Page, PageSize     int
	Keyword            string
	Status             CommissionStatus
	CommissionDateFrom string
	CommissionDateTo   string
}

// CommissionExportLimit 是单次同步提成导出的最大行数；超过时拒绝导出并提示
// 缩小筛选范围，不截断结果，也不只导出前 N 行。
const CommissionExportLimit = 10000

// commissionExportBatchSize 是导出分批读取的每批行数，与列表分页上限一致。
const commissionExportBatchSize = MaxListPageSize

type CommissionListResult struct {
	Items []*FinanceCommission
	Total int64
}
type CommissionEmployeeOption struct {
	ID          uuid.UUID
	DisplayName string
}

// CommissionReversalLine 是核销撤销时逐订单的有效提成输入与冲减结果。
type CommissionReversalLine struct {
	OrderID uuid.UUID
	Amount  decimal.Decimal
}

// CommissionReversalPlan 描述核销撤销对提成单和调整单的原子处理计划。
type CommissionReversalPlan struct {
	CancelCommission    bool
	CancelAdjustmentIDs []uuid.UUID
	Recoveries          []CommissionReversalLine
}

// PlanCommissionReversal 保留已支付历史，取消未支付权益，并把剩余有效提成冲减至零。
func PlanCommissionReversal(status CommissionStatus, baseAmount decimal.Decimal, lines []CommissionReversalLine, adjustments []*FinanceCommissionAdjustment) (*CommissionReversalPlan, error) {
	plan := &CommissionReversalPlan{}
	hasPaidExposure := status == CommissionPaid
	for _, item := range adjustments {
		if item.Status == CommissionPaid {
			hasPaidExposure = true
		}
	}
	if !hasPaidExposure {
		plan.CancelCommission = true
		for _, item := range adjustments {
			if item.Status == CommissionDraft || item.Status == CommissionConfirmed {
				plan.CancelAdjustmentIDs = append(plan.CancelAdjustmentIDs, item.ID)
			}
		}
		return plan, nil
	}
	for _, item := range adjustments {
		if item.Status == CommissionDraft {
			plan.CancelAdjustmentIDs = append(plan.CancelAdjustmentIDs, item.ID)
		}
	}
	if len(lines) == 0 {
		return nil, ErrCommissionSource
	}
	orderEffective := make(map[uuid.UUID]decimal.Decimal, len(lines))
	for _, line := range lines {
		orderEffective[line.OrderID] = orderEffective[line.OrderID].Add(line.Amount)
	}
	effective := baseAmount
	for _, item := range adjustments {
		if item.Status != CommissionConfirmed && item.Status != CommissionPaid {
			continue
		}
		if item.Direction == CommissionAdjustmentDecrease {
			effective = effective.Sub(item.Amount)
			orderEffective[item.OrderID] = orderEffective[item.OrderID].Sub(item.Amount)
		} else {
			effective = effective.Add(item.Amount)
			orderEffective[item.OrderID] = orderEffective[item.OrderID].Add(item.Amount)
		}
	}
	remaining := effective.Round(8)
	if !remaining.IsPositive() {
		return plan, nil
	}
	for _, line := range lines {
		available := orderEffective[line.OrderID].Round(8)
		if !available.IsPositive() || !remaining.IsPositive() {
			continue
		}
		amount := decimal.Min(available, remaining).Round(8)
		plan.Recoveries = append(plan.Recoveries, CommissionReversalLine{OrderID: line.OrderID, Amount: amount})
		remaining = remaining.Sub(amount).Round(8)
	}
	if remaining.IsPositive() {
		return nil, ErrCommissionSource
	}
	return plan, nil
}

// CommissionCandidateFilter 按来源单（核销/对冲二选一）发现计提候选：服务端
// 从来源订单提成归属中发现「员工 + 人员身份」组合，再按来源归属日期解析唯一
// 有效方案与员工分配；任一条件不满足的组合不产生候选。
type CommissionCandidateFilter struct {
	Page, PageSize int
	Keyword        string
	VerificationID uuid.UUID
	NettingID      uuid.UUID
}
type CommissionCandidateListResult struct {
	Items          []*CommissionCalculation
	Total          int64
	Page, PageSize int
}
type CommissionNettingCandidateFilter struct {
	Page, PageSize int
	Keyword        string
}
type CommissionNettingCandidateListResult struct {
	Items          []*FinanceNetting
	Total          int64
	Page, PageSize int
}

// FinanceCommissionRuleAssignment 是方案与员工之间保留审计的有效期分配段：
// 起始日必填且创建后不可变，终止日为空表示尚未终止；未生效分配以 CancelledAt
// 撤销，已生效分配以 EffectiveTo 终止并记录 TerminatedAt，不物理删除。
// EmployeeName 是名单投影用的员工展示名（用户不可考时为空）。
type FinanceCommissionRuleAssignment struct {
	ID, OrganizationID, RuleID, EmployeeID uuid.UUID
	EmployeeName                           string
	EffectiveFrom                          string
	EffectiveTo                            *string
	CancelledAt, TerminatedAt              *time.Time
	CancelledBy, TerminatedBy              *uuid.UUID
	CreatedBy                              uuid.UUID
	CreatedAt, UpdatedAt                   time.Time
}

type FinanceCommissionRule struct {
	ID, OrganizationID               uuid.UUID
	Name                             string
	OrganizationName                 string
	PersonnelRole                    CommissionPersonnelRole
	CalculationBasis                 CommissionCalculationBasis
	RatePercent                      decimal.Decimal
	EffectiveFrom, EffectiveTo, Note *string
	Enabled                          bool
	// LegacyReadOnly 标记迁移停用的无分配旧角色规则：历史只读，只能复制为
	// 新方案，禁止重新启用或补挂员工。
	LegacyReadOnly bool
	// Assignments 是未取消分配段投影（当前/未来名单与已终止历史段），
	// 由仓储随方案加载；多选只是批量写入的表现，不在方案上存员工 ID 数组。
	Assignments          []*FinanceCommissionRuleAssignment
	Version              uint64
	CreatedAt, UpdatedAt time.Time
}
type CommissionRuleFilter struct {
	Page, PageSize int
	Keyword        string
	PersonnelRole  CommissionPersonnelRole
	Enabled        *bool
	// EmployeeID 按适用员工过滤：只返回该员工存在未取消分配的方案。
	EmployeeID uuid.UUID
}
type CommissionRuleListResult struct {
	Items []*FinanceCommissionRule
	Total int64
}
type CreateCommissionRuleInput struct {
	Name                             string
	PersonnelRole                    CommissionPersonnelRole
	CalculationBasis                 CommissionCalculationBasis
	RatePercent                      decimal.Decimal
	EffectiveFrom, EffectiveTo, Note *string
	Enabled                          bool
	// EmployeeIDs 是初始适用员工：启用方案至少一名；草稿可空。仓储在事务内
	// 锁定成员关系后写入初始分配段（起始日等于方案起始日，终止日跟随方案）。
	EmployeeIDs []uuid.UUID
	// Today 是服务端统一财务业务日期（YYYY-MM-DD）；为空时由用例按当前时刻
	// 换算，保证「当天或未来」判定不依赖浏览器时区。
	Today string
}
type UpdateCommissionRuleInput struct {
	ID uuid.UUID
	CreateCommissionRuleInput
	ExpectedVersion uint64
}

// CommissionRuleEmployeeChange 描述方案名单的批量增删：ExpectedVersion 防并发
// 覆盖；ChangeEffectiveDate 对已生效方案必填且只允许当天或未来，未生效方案可
// 省略（新增默认从方案起始日或当天开始，移除默认从当天起算）。
type CommissionRuleEmployeeChange struct {
	RuleID              uuid.UUID
	EmployeeIDs         []uuid.UUID
	ExpectedVersion     uint64
	ChangeEffectiveDate string
	Today               string
}

// CopyCommissionRuleInput 描述【复制为新方案】：新方案以当天或未来日期生效，
// 可同时调整身份、口径、比例与名单；源方案终止日在同一事务内衔接为新方案
// 生效日前一日，保证新旧区间无缝不重叠。
type CopyCommissionRuleInput struct {
	SourceRuleID      uuid.UUID
	Name              string
	PersonnelRole     CommissionPersonnelRole
	CalculationBasis  CommissionCalculationBasis
	RatePercent       decimal.Decimal
	EffectiveFrom     string
	EffectiveTo, Note *string
	EmployeeIDs       []uuid.UUID
	Today             string
}

// CreateCommissionInput 的来源二选一：VerificationID 与 NettingID 恰好一个非空。
// 不再接受客户端指定规则：PersonnelRole 与员工一起由服务端按来源归属日期解析
// 唯一有效方案与员工分配，实际命中的规则快照写入提成单。
type CreateCommissionInput struct {
	VerificationID, NettingID, EmployeeID uuid.UUID
	PersonnelRole                         CommissionPersonnelRole
	Note                                  *string
	IdempotencyKey                        string
}

type CreateCommissionAdjustmentInput struct {
	CommissionID, OrderID uuid.UUID
	Direction             CommissionAdjustmentDirection
	Amount                decimal.Decimal
	Reason                string
	Note                  *string
	IdempotencyKey        string
}

// CommissionAdjustmentFilter 是财务调整列表的服务端分页筛选：状态、来源、员工
// 与订单号/提成号/调整号关键字；组织范围由调用方按权限显式传入。
type CommissionAdjustmentFilter struct {
	Page, PageSize int
	Keyword        string
	Status         CommissionStatus
	SourceType     CommissionAdjustmentSourceType
	EmployeeID     uuid.UUID
}

// CommissionAdjustmentListResult 是财务调整列表的分页结果，默认按 created_at 倒序。
type CommissionAdjustmentListResult struct {
	Items          []*FinanceCommissionAdjustment
	Total          int64
	Page, PageSize int
}

// MyFeeSupplementAdjustmentSource 是员工本人专属的冲减来源最小详情：只暴露订单号、
// 原提成号、补录费用摘要、建议金额、调整状态与生成时间，不包含父单其他订单行、
// 其他员工数据或财务明细。
type MyFeeSupplementAdjustmentSource struct {
	AdjustmentID          uuid.UUID
	AdjustmentNo          string
	OrderNo               string
	CommissionNo          string
	Status                CommissionStatus
	SuggestedAmount       decimal.Decimal
	BaseCurrency          string
	CreatedAt             time.Time
	FeeCode               string
	FeeName               string
	FeeCurrency           string
	FeeTotalAmount        decimal.Decimal
	FeeBaseCurrency       string
	FeeBaseCurrencyAmount decimal.Decimal
	FeeExpenseDate        string
	SupplementReason      string
}

// OrderCommissionSummaryVisibility 标识订单提成摘要的可见模式：EMPLOYEE 仅含
// 本人提成事实（SQL 层固定 employee_id = 调用者），ORGANIZATION 为组织级全员
// 状态与汇总。判定真相源是目标组织的 system.finance.commission.read 权限范围，
// 用户角色或管理员身份本身不替代该权限。
type OrderCommissionSummaryVisibility string

const (
	OrderCommissionVisibilityEmployee     OrderCommissionSummaryVisibility = "EMPLOYEE"
	OrderCommissionVisibilityOrganization OrderCommissionSummaryVisibility = "ORGANIZATION"
)

// OrderCommissionSummaryTarget 是订单页内一个需要附加提成摘要的订单；订单必须
// 已通过订单读取授权，由调用方（订单列表服务）传入。
type OrderCommissionSummaryTarget struct {
	OrderID        uuid.UUID
	OrganizationID uuid.UUID
}

// OrderCommissionSummaryScope 描述同一组织、同一可见模式下一组订单的批量聚合
// 范围。EMPLOYEE 模式的 EmployeeID 必填：隐私裁剪发生在 SQL 查询条件内，禁止
// 先聚合全员再在内存或前端裁剪。
type OrderCommissionSummaryScope struct {
	OrganizationID uuid.UUID
	Visibility     OrderCommissionSummaryVisibility
	OrderIDs       []uuid.UUID
	EmployeeID     uuid.UUID
}

// OrderCommissionSummary 是订单列表提成摘要的领域投影：可并存事实集合，允许
// 同票多员工、多来源、分期回款与不同状态同时存在，不构造整票互斥状态。
// 金额是当前可见范围内的汇总（订单组织本位币），取消记录不参与有效汇总；
// Paid 只表示对应提成单已发，PendingDecrease 只表示 DRAFT 冲减建议尚未处理，
// 预计机会是尚未生成有效基础提成单的估算来源数量，均不做应发承诺。
type OrderCommissionSummary struct {
	Visibility   OrderCommissionSummaryVisibility
	BaseCurrency string

	HasExpectedOpportunity   bool
	ExpectedOpportunityCount int

	HasDraftCommission    bool
	DraftCommissionCount  int
	DraftCommissionAmount decimal.Decimal

	HasConfirmedCommission    bool
	ConfirmedCommissionCount  int
	ConfirmedCommissionAmount decimal.Decimal

	HasPaidCommission    bool
	PaidCommissionCount  int
	PaidCommissionAmount decimal.Decimal

	HasPendingDecrease    bool
	PendingDecreaseCount  int
	PendingDecreaseAmount decimal.Decimal
}

// newEmptyOrderCommissionSummary 返回指定可见模式的本人/全员空态摘要。
func newEmptyOrderCommissionSummary(visibility OrderCommissionSummaryVisibility) *OrderCommissionSummary {
	return &OrderCommissionSummary{Visibility: visibility}
}

// validCommissionAdjustmentSourceType 判断来源类型取值是否已登记。
func validCommissionAdjustmentSourceType(source CommissionAdjustmentSourceType) bool {
	switch source {
	case CommissionAdjustmentSourceManual,
		CommissionAdjustmentSourceVerificationReversal,
		CommissionAdjustmentSourceNettingReversal,
		CommissionAdjustmentSourceLockedFeeSupplement:
		return true
	default:
		return false
	}
}

// validCommissionAdjustmentFilter 校验调整列表筛选：组织范围、分页、状态、来源、
// 员工与关键字长度。
func validCommissionAdjustmentFilter(organizationIDs []uuid.UUID, f CommissionAdjustmentFilter) bool {
	if !validCommissionOrganizationIDs(organizationIDs) || !ValidListPagination(f.Page, f.PageSize) || utf8.RuneCountInString(f.Keyword) > 100 {
		return false
	}
	if f.Status != "" && f.Status != CommissionDraft && f.Status != CommissionConfirmed && f.Status != CommissionPaid && f.Status != CommissionCancelled {
		return false
	}
	if f.SourceType != "" && !validCommissionAdjustmentSourceType(f.SourceType) {
		return false
	}
	return true
}

// CommissionGenerationContext 是生成提成前从核销单读取的 CNY 折算上下文。
type CommissionGenerationContext struct {
	CommissionDate string // 归属日期，等于核销单 verification_date，同时作为 CNY 汇率解析日
	BaseCurrency   string // 核销单本位币
}

type CommissionRepo interface {
	List(context.Context, uuid.UUID, CommissionFilter) (*CommissionListResult, error)
	ListScoped(context.Context, []uuid.UUID, CommissionFilter) (*CommissionListResult, error)
	Count(context.Context, uuid.UUID, CommissionFilter) (int64, error)
	CountScoped(context.Context, []uuid.UUID, CommissionFilter) (int64, error)
	ExportBatch(context.Context, uuid.UUID, CommissionFilter) ([]*FinanceCommission, error)
	ExportBatchScoped(context.Context, []uuid.UUID, CommissionFilter) ([]*FinanceCommission, error)
	SaveExportAudit(context.Context, *AuditEvent) error
	Get(context.Context, uuid.UUID, uuid.UUID) (*FinanceCommission, error)
	GetScoped(context.Context, []uuid.UUID, uuid.UUID) (*FinanceCommission, error)
	ListEmployees(context.Context, uuid.UUID, SelectorListOptions) (*PagedList[*CommissionEmployeeOption], error)
	ListEmployeesScoped(context.Context, []uuid.UUID, SelectorListOptions) (*PagedList[*CommissionEmployeeOption], error)
	ListCandidates(context.Context, uuid.UUID, CommissionCandidateFilter) (*CommissionCandidateListResult, error)
	ListNettingCandidates(context.Context, uuid.UUID, CommissionNettingCandidateFilter) (*CommissionNettingCandidateListResult, error)
	// Preview 按来源二选一与「员工 + 人员身份」自动解析唯一有效方案后预览提成。
	Preview(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, CommissionPersonnelRole) (*CommissionCalculation, error)
	GetGenerationContext(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*CommissionGenerationContext, error)
	ListRules(context.Context, uuid.UUID, CommissionRuleFilter) (*CommissionRuleListResult, error)
	ListRulesScoped(context.Context, []uuid.UUID, CommissionRuleFilter) (*CommissionRuleListResult, error)
	GetRuleScoped(context.Context, []uuid.UUID, uuid.UUID) (*FinanceCommissionRule, error)
	// CreateRule 在事务内按 Membership → Rule 固定锁序先锁员工成员关系，
	// 再创建方案与初始分配段，并在锁内校验跨方案实际区间重叠。
	CreateRule(context.Context, uuid.UUID, uuid.UUID, *FinanceCommissionRule, *AuditEvent) (*FinanceCommissionRule, error)
	UpdateRule(context.Context, uuid.UUID, UpdateCommissionRuleInput, *AuditEvent) (*FinanceCommissionRule, error)
	// AssignRuleEmployees / RemoveRuleEmployees 在同一锁序下批量新增或终止
	// 员工分配：已生效名单只允许当天或未来的变更生效日，不物理删除历史分配，
	// 写入成功后递增方案版本。
	AssignRuleEmployees(context.Context, uuid.UUID, uuid.UUID, CommissionRuleEmployeeChange, *AuditEvent) (*FinanceCommissionRule, error)
	RemoveRuleEmployees(context.Context, uuid.UUID, uuid.UUID, CommissionRuleEmployeeChange, *AuditEvent) (*FinanceCommissionRule, error)
	// CopyRule 实现【复制为新方案】：新方案与初始分配在锁定源方案后创建，
	// 源方案终止日衔接为新方案生效日前一并递增版本。
	CopyRule(context.Context, uuid.UUID, uuid.UUID, CopyCommissionRuleInput, *FinanceCommissionRule, *AuditEvent, *AuditEvent) (*FinanceCommissionRule, error)
	GetByKey(context.Context, uuid.UUID, string) (*FinanceCommission, error)
	Create(context.Context, uuid.UUID, *FinanceCommission, *CommissionCNYSnapshot, *AuditEvent) error
	Transition(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, CommissionStatus, string, *AuditEvent) (*FinanceCommission, error)
	GetAdjustmentByKey(context.Context, uuid.UUID, string) (*FinanceCommissionAdjustment, error)
	GetAdjustmentScoped(context.Context, []uuid.UUID, uuid.UUID) (*FinanceCommissionAdjustment, error)
	// ListAdjustmentsScoped 服务端分页读取组织范围内的提成调整，默认 created_at
	// 倒序；筛选与关键字匹配在数据库内完成。
	ListAdjustmentsScoped(context.Context, []uuid.UUID, CommissionAdjustmentFilter) (*CommissionAdjustmentListResult, error)
	// GetMyFeeSupplementAdjustmentSource 员工本人专属冲减来源最小详情：仓储查询
	// 同时限定调整 ID、employee_id = 当前用户、LOCKED_FEE_SUPPLEMENT 来源和组织
	// 成员关系；任一不满足时按不存在处理，不泄露记录事实。
	GetMyFeeSupplementAdjustmentSource(context.Context, uuid.UUID, uuid.UUID) (*MyFeeSupplementAdjustmentSource, error)
	// ListOrderSummaries 按组织与可见模式分组的批量聚合查询：一次处理整页订单
	// ID，不逐行查询。EMPLOYEE 作用域必须在 SQL 条件内固定 employee_id，普通
	// 员工响应不得包含他人数量、姓名、状态、金额或整票是否有提成等旁路字段。
	ListOrderSummaries(context.Context, []OrderCommissionSummaryScope) (map[uuid.UUID]*OrderCommissionSummary, error)
	CreateAdjustment(context.Context, uuid.UUID, uuid.UUID, *FinanceCommissionAdjustment, *AuditEvent) (*FinanceCommissionAdjustment, error)
	TransitionAdjustment(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, CommissionStatus, string, *AuditEvent) (*FinanceCommissionAdjustment, error)
}
