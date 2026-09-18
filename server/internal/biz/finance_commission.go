package biz

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	financev1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/shopspring/decimal"
)

var (
	ErrCommissionNotFound             = errors.NotFound("FINANCE_COMMISSION_NOT_FOUND", "提成记录不存在")
	ErrCommissionInvalid              = errors.BadRequest("FINANCE_COMMISSION_INVALID", "提成参数不合法")
	ErrCommissionSource               = errors.Conflict("FINANCE_COMMISSION_SOURCE", "仅有效应收核销可计提，且必须存在可计算的已实现收入")
	ErrCommissionDuplicate            = errors.Conflict("FINANCE_COMMISSION_DUPLICATE", "该核销、员工与人员角色已存在未取消提成记录")
	ErrCommissionTransition           = errors.Conflict(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FINANCE_COMMISSION_TRANSITION), "当前提成状态不允许该操作")
	ErrCommissionRuleNotFound         = errors.NotFound("FINANCE_COMMISSION_RULE_NOT_FOUND", "提成规则不存在")
	ErrCommissionRuleInvalid          = errors.BadRequest("FINANCE_COMMISSION_RULE_INVALID", "提成规则字段不合法")
	ErrCommissionRuleConflict         = errors.Conflict(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FINANCE_COMMISSION_RULE_CONFLICT), "提成规则名称已存在或版本已变化")
	ErrCommissionEmployeeRole         = errors.Conflict("FINANCE_COMMISSION_EMPLOYEE_ROLE", "所选员工未在客户档案中担任规则指定角色")
	ErrCommissionSourceChanged        = errors.Conflict(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FINANCE_COMMISSION_SOURCE_CHANGED), "提成来源数据已变化，请取消当前草稿并重新生成")
	ErrCommissionUnconfirmedFees      = errors.Conflict(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FINANCE_COMMISSION_UNCONFIRMED_FEES), "关联订单仍有草稿费用，请先确认或作废后再确认提成")
	ErrCommissionAdjustmentNotFound   = errors.NotFound("FINANCE_COMMISSION_ADJUSTMENT_NOT_FOUND", "提成调整记录不存在")
	ErrCommissionAdjustmentInvalid    = errors.BadRequest("FINANCE_COMMISSION_ADJUSTMENT_INVALID", "提成调整参数不合法")
	ErrCommissionAdjustmentTransition = errors.Conflict(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FINANCE_COMMISSION_ADJUSTMENT_TRANSITION), "当前提成调整状态不允许该操作")
	ErrCommissionAdjustmentExceeds    = errors.Conflict(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FINANCE_COMMISSION_ADJUSTMENT_EXCEEDS), "冲减后的有效提成金额不能小于零")
	// ErrCommissionAdjustmentCancelNotAllowed 来源专属取消门禁：LOCKED_FEE_SUPPLEMENT
	// 的冲减建议只有 DRAFT 可经通用取消忽略；CONFIRMED/PAID 即使绕过页面直接调用
	// 通用取消接口也稳定拒绝，防止状态洗白。
	ErrCommissionAdjustmentCancelNotAllowed = errors.Conflict(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FINANCE_COMMISSION_ADJUSTMENT_CANCEL_NOT_ALLOWED), "系统冲减建议确认或扣回后不允许取消")
	ErrCommissionExportLimit                = errors.BadRequest("FINANCE_COMMISSION_EXPORT_LIMIT", "提成导出行数超过单次上限，请缩小筛选范围后重试")
	// 锁后费用补录审批对历史提成复算快照的稳定阻断错误码。
	ErrCommissionSnapshotUnavailable           = errors.Conflict("COMMISSION_SNAPSHOT_UNAVAILABLE", "历史提成复算快照不可用，无法处理锁后费用补录")
	ErrCommissionBaseCurrencyMismatch          = errors.Conflict("COMMISSION_BASE_CURRENCY_MISMATCH", "补录费用本位币与历史提成快照本位币不一致，不支持跨本位币换算")
	ErrCommissionCalculationVersionUnsupported = errors.Conflict("COMMISSION_CALCULATION_VERSION_UNSUPPORTED", "提成计算版本不受支持，无法判定应付成本影响")
	// 提成方案员工分配：名单变更、区间唯一与历史只读的稳定领域错误。
	ErrCommissionRuleAssignmentInvalid       = errors.BadRequest("FINANCE_COMMISSION_RULE_ASSIGNMENT_INVALID", "提成方案员工分配参数不合法")
	ErrCommissionRuleEmployeeInvalid         = errors.Conflict("FINANCE_COMMISSION_RULE_EMPLOYEE_INVALID", "所选员工不是当前组织的有效成员")
	ErrCommissionRuleEnableNoEmployee        = errors.Conflict("FINANCE_COMMISSION_RULE_ENABLE_NO_EMPLOYEE", "启用提成方案前必须至少分配一名员工")
	ErrCommissionRuleIntervalOverlap         = errors.Conflict("FINANCE_COMMISSION_RULE_INTERVAL_OVERLAP", "同一员工同一身份在重叠期间只能属于一个启用方案")
	ErrCommissionRuleRetroactive             = errors.Conflict("FINANCE_COMMISSION_RULE_RETROACTIVE", "已生效方案的名单与区间变更只能选择当天或未来的生效日期")
	ErrCommissionRuleLockedParams            = errors.Conflict("FINANCE_COMMISSION_RULE_LOCKED_PARAMS", "方案已生效，人员身份、计提口径、比例和起始日不可原地修改，也不能停用，请复制为新方案")
	ErrCommissionRuleLegacyReadOnly          = errors.Conflict("FINANCE_COMMISSION_RULE_LEGACY_READ_ONLY", "迁移前的历史旧规则为只读，请复制为新方案后再使用")
	ErrCommissionRuleMemberAssignmentBlocked = errors.Conflict("FINANCE_COMMISSION_RULE_MEMBER_ASSIGNMENT", "员工仍有当前或未来的提成方案分配，请先终止分配后再停用成员")
	// ErrCommissionRuleNotResolved 计提自动解析失败：来源归属日期未唯一命中
	// 「已启用方案 ∩ 该员工未取消有效分配」，或员工身份与订单提成归属不匹配。
	// 财务不能把未分配给员工的方案套用到该员工，也不能用不匹配的身份绕过方案。
	ErrCommissionRuleNotResolved = errors.Conflict("FINANCE_COMMISSION_RULE_NOT_RESOLVED", "来源归属日期未唯一命中该员工该身份的有效提成方案与员工分配")
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
	CreateAdjustment(context.Context, uuid.UUID, uuid.UUID, *FinanceCommissionAdjustment, *AuditEvent) (*FinanceCommissionAdjustment, error)
	TransitionAdjustment(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, CommissionStatus, string, *AuditEvent) (*FinanceCommissionAdjustment, error)
}

func (u *CommissionUsecase) ListEmployees(ctx context.Context, org uuid.UUID, options SelectorListOptions) (*PagedList[*CommissionEmployeeOption], error) {
	return u.ListEmployeesScoped(ctx, []uuid.UUID{org}, options)
}

func (u *CommissionUsecase) ListEmployeesScoped(ctx context.Context, organizationIDs []uuid.UUID, options SelectorListOptions) (*PagedList[*CommissionEmployeeOption], error) {
	options.Keyword = strings.TrimSpace(options.Keyword)
	if !validCommissionOrganizationIDs(organizationIDs) || !ValidListPagination(options.Page, options.PageSize) || utf8.RuneCountInString(options.Keyword) > 100 {
		return nil, ErrCommissionInvalid
	}
	return u.repo.ListEmployeesScoped(ctx, organizationIDs, options)
}
func (u *CommissionUsecase) ListCandidates(ctx context.Context, org uuid.UUID, f CommissionCandidateFilter) (*CommissionCandidateListResult, error) {
	f.Keyword = strings.TrimSpace(f.Keyword)
	if org == uuid.Nil || !validCommissionSource(f.VerificationID, f.NettingID) || !ValidListPagination(f.Page, f.PageSize) || utf8.RuneCountInString(f.Keyword) > 100 {
		return nil, ErrCommissionInvalid
	}
	return u.repo.ListCandidates(ctx, org, f)
}

// validCommissionSource 校验提成来源二选一：核销与对冲恰好一个非空。
func validCommissionSource(verificationID, nettingID uuid.UUID) bool {
	return (verificationID != uuid.Nil) != (nettingID != uuid.Nil)
}

// ListNettingCandidates 返回可用于计提的已确认对冲单候选（存在应收分摊）。
func (u *CommissionUsecase) ListNettingCandidates(ctx context.Context, org uuid.UUID, f CommissionNettingCandidateFilter) (*CommissionNettingCandidateListResult, error) {
	f.Keyword = strings.TrimSpace(f.Keyword)
	if org == uuid.Nil || !ValidListPagination(f.Page, f.PageSize) || utf8.RuneCountInString(f.Keyword) > 100 {
		return nil, ErrCommissionInvalid
	}
	return u.repo.ListNettingCandidates(ctx, org, f)
}

type CommissionUsecase struct {
	repo       CommissionRepo
	config     *OrderConfigUsecase
	transactor Transactor
}

func NewCommissionUsecase(repo CommissionRepo, config *OrderConfigUsecase, transactor Transactor) *CommissionUsecase {
	return &CommissionUsecase{repo: repo, config: config, transactor: transactor}
}

func (u *CommissionUsecase) List(ctx context.Context, org uuid.UUID, f CommissionFilter) (*CommissionListResult, error) {
	return u.ListScoped(ctx, []uuid.UUID{org}, f)
}
func (u *CommissionUsecase) ListScoped(ctx context.Context, organizationIDs []uuid.UUID, f CommissionFilter) (*CommissionListResult, error) {
	f.Keyword = strings.TrimSpace(f.Keyword)
	f.CommissionDateFrom = strings.TrimSpace(f.CommissionDateFrom)
	f.CommissionDateTo = strings.TrimSpace(f.CommissionDateTo)
	if !validCommissionOrganizationIDs(organizationIDs) || !ValidListPagination(f.Page, f.PageSize) || (f.Status != "" && f.Status != CommissionDraft && f.Status != CommissionConfirmed && f.Status != CommissionPaid && f.Status != CommissionCancelled) || !validFinanceDateRange(f.CommissionDateFrom, f.CommissionDateTo) {
		return nil, ErrCommissionInvalid
	}
	return u.repo.ListScoped(ctx, organizationIDs, f)
}

func validCommissionOrganizationIDs(organizationIDs []uuid.UUID) bool {
	if len(organizationIDs) == 0 {
		return false
	}
	seen := map[uuid.UUID]struct{}{}
	for _, id := range organizationIDs {
		if id == uuid.Nil {
			return false
		}
		if _, exists := seen[id]; exists {
			return false
		}
		seen[id] = struct{}{}
	}
	return true
}

// Export 按与列表一致的筛选、排序和 CNY 口径同步导出提成：先按同一谓词计数，
// 超过单次上限时直接拒绝且不执行数据查询；未超限时按每批最多 200 行分批读取。
// 全部数据读取后、返回响应前写入成功导出审计，审计失败时整体失败，避免
// 数据成功返回却没有审计。
func (u *CommissionUsecase) Export(ctx context.Context, org, actor uuid.UUID, f CommissionFilter) ([]*FinanceCommission, error) {
	return u.ExportScoped(ctx, []uuid.UUID{org}, actor, f)
}

func (u *CommissionUsecase) ExportScoped(ctx context.Context, organizationIDs []uuid.UUID, actor uuid.UUID, f CommissionFilter) ([]*FinanceCommission, error) {
	f.Keyword = strings.TrimSpace(f.Keyword)
	f.CommissionDateFrom = strings.TrimSpace(f.CommissionDateFrom)
	f.CommissionDateTo = strings.TrimSpace(f.CommissionDateTo)
	if !validCommissionOrganizationIDs(organizationIDs) || actor == uuid.Nil || (f.Status != "" && f.Status != CommissionDraft && f.Status != CommissionConfirmed && f.Status != CommissionPaid && f.Status != CommissionCancelled) || !validFinanceDateRange(f.CommissionDateFrom, f.CommissionDateTo) {
		return nil, ErrCommissionInvalid
	}
	total, err := u.repo.CountScoped(ctx, organizationIDs, f)
	if err != nil {
		return nil, err
	}
	if total > CommissionExportLimit {
		return nil, ErrCommissionExportLimit
	}
	items := make([]*FinanceCommission, 0, total)
	for offset := 0; offset < int(total); offset += commissionExportBatchSize {
		f.Page, f.PageSize = offset/commissionExportBatchSize+1, commissionExportBatchSize
		batch, batchErr := u.repo.ExportBatchScoped(ctx, organizationIDs, f)
		if batchErr != nil {
			return nil, batchErr
		}
		items = append(items, batch...)
	}
	if err := u.repo.SaveExportAudit(ctx, commissionExportAudit(organizationIDs, actor, f, len(items))); err != nil {
		return nil, err
	}
	return items, nil
}

// commissionExportAudit 构造成功导出审计：记录操作人、组织、规范化筛选摘要与
// 最终行数，不保存导出内容。
func commissionExportAudit(organizationIDs []uuid.UUID, actor uuid.UUID, f CommissionFilter, rowCount int) *AuditEvent {
	organizationIDStrings := make([]string, 0, len(organizationIDs))
	for _, organizationID := range organizationIDs {
		organizationIDStrings = append(organizationIDStrings, organizationID.String())
	}
	sort.Strings(organizationIDStrings)
	details := map[string]string{"row_count": strconv.Itoa(rowCount), "organization_count": strconv.Itoa(len(organizationIDStrings)), "organization_ids": strings.Join(organizationIDStrings, ",")}
	var organizationID *uuid.UUID
	if len(organizationIDs) == 1 {
		organizationID = &organizationIDs[0]
	}
	if f.Keyword != "" {
		details["keyword"] = f.Keyword
	}
	if f.Status != "" {
		details["status"] = string(f.Status)
	}
	if f.CommissionDateFrom != "" {
		details["commission_date_from"] = f.CommissionDateFrom
	}
	if f.CommissionDateTo != "" {
		details["commission_date_to"] = f.CommissionDateTo
	}
	return &AuditEvent{OrganizationID: organizationID, UserID: &actor, Action: "finance.commission.export", Result: "success", ResourceType: "finance_commission", Details: details}
}

func (u *CommissionUsecase) Get(ctx context.Context, org, id uuid.UUID) (*FinanceCommission, error) {
	return u.GetScoped(ctx, []uuid.UUID{org}, id)
}
func (u *CommissionUsecase) GetScoped(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*FinanceCommission, error) {
	if !validCommissionOrganizationIDs(organizationIDs) || id == uuid.Nil {
		return nil, ErrCommissionInvalid
	}
	return u.repo.GetScoped(ctx, organizationIDs, id)
}

func (u *CommissionUsecase) GetAdjustmentScoped(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*FinanceCommissionAdjustment, error) {
	if !validCommissionOrganizationIDs(organizationIDs) || id == uuid.Nil {
		return nil, ErrCommissionAdjustmentInvalid
	}
	return u.repo.GetAdjustmentScoped(ctx, organizationIDs, id)
}

// ListAdjustmentsScoped 财务调整列表：服务端分页与筛选校验后委托仓储；权限与
// 组织范围由 Service 按 commission.read 显式解析。
func (u *CommissionUsecase) ListAdjustmentsScoped(ctx context.Context, organizationIDs []uuid.UUID, f CommissionAdjustmentFilter) (*CommissionAdjustmentListResult, error) {
	f.Keyword = strings.TrimSpace(f.Keyword)
	f.SourceType = CommissionAdjustmentSourceType(strings.ToUpper(strings.TrimSpace(string(f.SourceType))))
	if !validCommissionAdjustmentFilter(organizationIDs, f) {
		return nil, ErrCommissionAdjustmentInvalid
	}
	return u.repo.ListAdjustmentsScoped(ctx, organizationIDs, f)
}

// GetMyFeeSupplementAdjustmentSource 员工本人专属冲减来源详情：不要求组织级
// commission.read，授权（employee_id = 当前用户 + 组织成员关系）在领域与仓储
// 查询中执行；参数缺失与他人调整一律按不存在处理，不泄露记录事实。
func (u *CommissionUsecase) GetMyFeeSupplementAdjustmentSource(ctx context.Context, caller *Principal, id uuid.UUID) (*MyFeeSupplementAdjustmentSource, error) {
	if caller == nil || caller.UserID == uuid.Nil || id == uuid.Nil {
		return nil, ErrCommissionAdjustmentNotFound
	}
	return u.repo.GetMyFeeSupplementAdjustmentSource(ctx, caller.UserID, id)
}

// Preview 按核销或对冲来源二选一预览提成：服务端按来源归属日期、员工与人员
// 身份自动解析唯一有效方案与员工分配，并按生成日固化 CNY 快照（原币记账口径）。
func (u *CommissionUsecase) Preview(ctx context.Context, org, verificationID, nettingID, employeeID uuid.UUID, personnelRole CommissionPersonnelRole) (*CommissionCalculation, error) {
	if org == uuid.Nil || employeeID == uuid.Nil || !validCommissionPersonnelRole(personnelRole) || !validCommissionSource(verificationID, nettingID) {
		return nil, ErrCommissionInvalid
	}
	calculation, err := u.repo.Preview(ctx, org, verificationID, nettingID, employeeID, personnelRole)
	if err != nil {
		return nil, err
	}
	generation, err := u.repo.GetGenerationContext(ctx, org, verificationID, nettingID)
	if err != nil {
		return nil, err
	}
	snapshot, err := ResolveCommissionCNYRate(calculation.BaseCurrency, generation.CommissionDate, calculation.CommissionAmount)
	if err != nil {
		return nil, err
	}
	calculation.CNY = snapshot
	return calculation, nil
}

func (u *CommissionUsecase) ListRules(ctx context.Context, org uuid.UUID, f CommissionRuleFilter) (*CommissionRuleListResult, error) {
	return u.ListRulesScoped(ctx, []uuid.UUID{org}, f)
}
func (u *CommissionUsecase) ListRulesScoped(ctx context.Context, organizationIDs []uuid.UUID, f CommissionRuleFilter) (*CommissionRuleListResult, error) {
	f.Keyword = strings.TrimSpace(f.Keyword)
	if !validCommissionOrganizationIDs(organizationIDs) || !ValidListPagination(f.Page, f.PageSize) || utf8.RuneCountInString(f.Keyword) > 100 || (f.PersonnelRole != "" && !validCommissionPersonnelRole(f.PersonnelRole)) {
		return nil, ErrCommissionRuleInvalid
	}
	return u.repo.ListRulesScoped(ctx, organizationIDs, f)
}

func (u *CommissionUsecase) GetRuleScoped(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*FinanceCommissionRule, error) {
	if !validCommissionOrganizationIDs(organizationIDs) || id == uuid.Nil {
		return nil, ErrCommissionRuleInvalid
	}
	return u.repo.GetRuleScoped(ctx, organizationIDs, id)
}

// FinanceBusinessDate 返回服务端统一财务业务日期（YYYY-MM-DD）：与订单列表、
// 汇率督办同口径（Asia/Shanghai），不使用浏览器本地时区。
func FinanceBusinessDate(now time.Time) string {
	return now.In(financeBusinessLocation).Format("2006-01-02")
}

// FinanceDateBefore 返回指定业务日期的前一日（YYYY-MM-DD）；输入必须合法。
func FinanceDateBefore(date string) string {
	parsed, err := time.ParseInLocation("2006-01-02", date, time.UTC)
	if err != nil {
		return date
	}
	return parsed.AddDate(0, 0, -1).Format("2006-01-02")
}

// CommissionClosedIntervalsOverlap 判断两个 YYYY-MM-DD 闭区间是否相交；
// 终点为空串视为正无穷。方案员工分配的实际有效区间唯一性由该纯函数与
// Membership 父行锁共同保证。
func CommissionClosedIntervalsOverlap(fromA, toA, fromB, toB string) bool {
	if toA != "" && fromB > toA {
		return false
	}
	if toB != "" && fromA > toB {
		return false
	}
	return true
}

// financeBusinessLocation 统一财务业务日期时区（Asia/Shanghai，UTC+8 固定偏移）。
var financeBusinessLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

// normalizeCommissionRuleEmployeeIDs 去重并按 UUID 升序整理员工集合；批量
// 多选上限与列表分页上限一致（200），空集合合法（草稿可无员工）。
func normalizeCommissionRuleEmployeeIDs(ids []uuid.UUID) ([]uuid.UUID, bool) {
	if len(ids) > MaxListPageSize {
		return nil, false
	}
	seen := make(map[uuid.UUID]struct{}, len(ids))
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			return nil, false
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result, true
}

func (u *CommissionUsecase) CreateRule(ctx context.Context, org, actor uuid.UUID, in CreateCommissionRuleInput) (*FinanceCommissionRule, error) {
	normalized, err := normalizeCommissionRuleInput(in)
	if err != nil || org == uuid.Nil || actor == uuid.Nil {
		return nil, ErrCommissionRuleInvalid
	}
	if normalized.Enabled && len(normalized.EmployeeIDs) == 0 {
		return nil, ErrCommissionRuleEnableNoEmployee
	}
	rule := &FinanceCommissionRule{ID: uuid.Must(uuid.NewV7()), OrganizationID: org, Name: normalized.Name, PersonnelRole: normalized.PersonnelRole, CalculationBasis: normalized.CalculationBasis, RatePercent: normalized.RatePercent, EffectiveFrom: normalized.EffectiveFrom, EffectiveTo: normalized.EffectiveTo, Note: normalized.Note, Enabled: normalized.Enabled, LegacyReadOnly: false, Version: 1}
	if len(normalized.EmployeeIDs) > 0 {
		// 初始分配：起始日等于方案起始日，终止日跟随方案终止日。
		if rule.EffectiveFrom == nil {
			return nil, ErrCommissionRuleInvalid
		}
		rule.Assignments = make([]*FinanceCommissionRuleAssignment, 0, len(normalized.EmployeeIDs))
		for _, employeeID := range normalized.EmployeeIDs {
			rule.Assignments = append(rule.Assignments, &FinanceCommissionRuleAssignment{
				ID: uuid.Must(uuid.NewV7()), OrganizationID: org, RuleID: rule.ID,
				EmployeeID: employeeID, EffectiveFrom: *rule.EffectiveFrom, EffectiveTo: rule.EffectiveTo, CreatedBy: actor,
			})
		}
	}
	return u.repo.CreateRule(ctx, org, actor, rule, commissionRuleAudit(org, actor, rule.ID, "finance.commission_rule.create"))
}

func (u *CommissionUsecase) UpdateRule(ctx context.Context, org, actor uuid.UUID, in UpdateCommissionRuleInput) (*FinanceCommissionRule, error) {
	normalized, err := normalizeCommissionRuleInput(in.CreateCommissionRuleInput)
	if err != nil || org == uuid.Nil || actor == uuid.Nil || in.ID == uuid.Nil || in.ExpectedVersion == 0 {
		return nil, ErrCommissionRuleInvalid
	}
	in.CreateCommissionRuleInput = normalized
	// 名单变更只经 Assign/RemoveRuleEmployees 专属路径，更新入口忽略员工集合。
	in.CreateCommissionRuleInput.EmployeeIDs = nil
	return u.repo.UpdateRule(ctx, org, in, commissionRuleAudit(org, actor, in.ID, "finance.commission_rule.update"))
}

func (u *CommissionUsecase) AssignRuleEmployees(ctx context.Context, org, actor uuid.UUID, in CommissionRuleEmployeeChange) (*FinanceCommissionRule, error) {
	change, err := validCommissionRuleEmployeeChange(in)
	if err != nil {
		return nil, err
	}
	audit := commissionRuleEmployeeChangeAudit(org, actor, change, "finance.commission_rule.assign_employees")
	return u.repo.AssignRuleEmployees(ctx, org, actor, change, audit)
}

func (u *CommissionUsecase) RemoveRuleEmployees(ctx context.Context, org, actor uuid.UUID, in CommissionRuleEmployeeChange) (*FinanceCommissionRule, error) {
	change, err := validCommissionRuleEmployeeChange(in)
	if err != nil {
		return nil, err
	}
	audit := commissionRuleEmployeeChangeAudit(org, actor, change, "finance.commission_rule.remove_employees")
	return u.repo.RemoveRuleEmployees(ctx, org, actor, change, audit)
}

func (u *CommissionUsecase) CopyRule(ctx context.Context, org, actor uuid.UUID, in CopyCommissionRuleInput) (*FinanceCommissionRule, error) {
	normalized, err := normalizeCopyCommissionRuleInput(in)
	if err != nil || org == uuid.Nil || actor == uuid.Nil {
		return nil, ErrCommissionRuleInvalid
	}
	plan := &FinanceCommissionRule{
		ID: uuid.Must(uuid.NewV7()), OrganizationID: org, Name: normalized.Name,
		PersonnelRole: normalized.PersonnelRole, CalculationBasis: normalized.CalculationBasis,
		RatePercent: normalized.RatePercent, EffectiveFrom: &normalized.EffectiveFrom,
		EffectiveTo: normalized.EffectiveTo, Note: normalized.Note, Enabled: true, LegacyReadOnly: false, Version: 1,
	}
	plan.Assignments = make([]*FinanceCommissionRuleAssignment, 0, len(normalized.EmployeeIDs))
	for _, employeeID := range normalized.EmployeeIDs {
		plan.Assignments = append(plan.Assignments, &FinanceCommissionRuleAssignment{
			ID: uuid.Must(uuid.NewV7()), OrganizationID: org, RuleID: plan.ID,
			EmployeeID: employeeID, EffectiveFrom: normalized.EffectiveFrom, EffectiveTo: normalized.EffectiveTo, CreatedBy: actor,
		})
	}
	sourceAdjustAudit := commissionRuleAudit(org, actor, normalized.SourceRuleID, "finance.commission_rule.copy_terminate_source")
	sourceAdjustAudit.Details = map[string]string{
		"copy.effective_from": normalized.EffectiveFrom,
		"copy.rule_id":        plan.ID.String(),
	}
	createAudit := commissionRuleAudit(org, actor, plan.ID, "finance.commission_rule.copy")
	createAudit.Details = map[string]string{
		"copy.source_rule_id": normalized.SourceRuleID.String(),
		"employee_ids":        commissionEmployeeIDList(normalized.EmployeeIDs),
	}
	return u.repo.CopyRule(ctx, org, actor, normalized, plan, sourceAdjustAudit, createAudit)
}

// validCommissionRuleEmployeeChange 归一化名单增删输入：员工集合非空且不超过
// 分页上限，变更生效日期必须是合法财务日期。
func validCommissionRuleEmployeeChange(in CommissionRuleEmployeeChange) (CommissionRuleEmployeeChange, error) {
	in.Today = strings.TrimSpace(in.Today)
	if in.Today == "" {
		in.Today = FinanceBusinessDate(time.Now())
	}
	in.ChangeEffectiveDate = strings.TrimSpace(in.ChangeEffectiveDate)
	employees, ok := normalizeCommissionRuleEmployeeIDs(in.EmployeeIDs)
	if !ok {
		return CommissionRuleEmployeeChange{}, ErrCommissionRuleAssignmentInvalid
	}
	in.EmployeeIDs = employees
	if in.RuleID == uuid.Nil || in.ExpectedVersion == 0 || len(in.EmployeeIDs) == 0 {
		return CommissionRuleEmployeeChange{}, ErrCommissionRuleAssignmentInvalid
	}
	if in.ChangeEffectiveDate != "" && !validFinanceDate(in.ChangeEffectiveDate) {
		return CommissionRuleEmployeeChange{}, ErrCommissionRuleAssignmentInvalid
	}
	return in, nil
}

// normalizeCopyCommissionRuleInput 归一化【复制为新方案】输入：新方案必须以
// 当天或未来日期生效、至少一名真实员工，计算参数全部显式提供。
func normalizeCopyCommissionRuleInput(in CopyCommissionRuleInput) (CopyCommissionRuleInput, error) {
	in.Today = strings.TrimSpace(in.Today)
	if in.Today == "" {
		in.Today = FinanceBusinessDate(time.Now())
	}
	in.Name = strings.TrimSpace(in.Name)
	in.EffectiveFrom = strings.TrimSpace(in.EffectiveFrom)
	in.EffectiveTo = normalizedOptionalFinanceString(in.EffectiveTo)
	in.Note = normalizedOptionalFinanceString(in.Note)
	employees, ok := normalizeCommissionRuleEmployeeIDs(in.EmployeeIDs)
	if !ok || len(employees) == 0 {
		return CopyCommissionRuleInput{}, ErrCommissionRuleInvalid
	}
	in.EmployeeIDs = employees
	if in.SourceRuleID == uuid.Nil || in.Name == "" || utf8.RuneCountInString(in.Name) > 100 ||
		!validCommissionPersonnelRole(in.PersonnelRole) ||
		(in.CalculationBasis != CommissionBasisRealizedProfit && in.CalculationBasis != CommissionBasisRealizedRevenue) ||
		!in.RatePercent.IsPositive() || in.RatePercent.GreaterThan(decimal.NewFromInt(100)) ||
		!validFinanceDate(in.EffectiveFrom) || in.EffectiveFrom < in.Today ||
		(in.EffectiveTo != nil && (!validFinanceDate(*in.EffectiveTo) || in.EffectiveFrom > *in.EffectiveTo)) ||
		(in.Note != nil && utf8.RuneCountInString(*in.Note) > 500) {
		return CopyCommissionRuleInput{}, ErrCommissionRuleInvalid
	}
	return in, nil
}

// NewCommissionRuleMemberAssignmentBlocked 构造成员停用阻断错误：列出需先以
// 当天或未来日期终止分配的方案名，管理员先处理方案名单再停用成员。
func NewCommissionRuleMemberAssignmentBlocked(ruleNames []string) error {
	if len(ruleNames) == 0 {
		return ErrCommissionRuleMemberAssignmentBlocked
	}
	return errors.Conflict("FINANCE_COMMISSION_RULE_MEMBER_ASSIGNMENT", fmt.Sprintf("员工仍有当前或未来的提成方案分配（%s），请先以当天或未来的日期终止分配后再停用成员", strings.Join(ruleNames, "、")))
}

func commissionEmployeeIDList(ids []uuid.UUID) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, id.String())
	}
	return strings.Join(parts, ",")
}

func commissionRuleEmployeeChangeAudit(org, actor uuid.UUID, change CommissionRuleEmployeeChange, action string) *AuditEvent {
	event := commissionRuleAudit(org, actor, change.RuleID, action)
	details := map[string]string{
		"employee_ids":     commissionEmployeeIDList(change.EmployeeIDs),
		"expected_version": strconv.FormatUint(change.ExpectedVersion, 10),
		"today":            change.Today,
	}
	if change.ChangeEffectiveDate != "" {
		details["change_effective_date"] = change.ChangeEffectiveDate
	}
	event.Details = details
	return event
}

func normalizeCommissionRuleInput(in CreateCommissionRuleInput) (CreateCommissionRuleInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.EffectiveFrom = normalizedOptionalFinanceString(in.EffectiveFrom)
	in.EffectiveTo = normalizedOptionalFinanceString(in.EffectiveTo)
	in.Note = normalizedOptionalFinanceString(in.Note)
	in.Today = strings.TrimSpace(in.Today)
	if in.Today == "" {
		in.Today = FinanceBusinessDate(time.Now())
	}
	employees, ok := normalizeCommissionRuleEmployeeIDs(in.EmployeeIDs)
	if !ok {
		return CreateCommissionRuleInput{}, ErrCommissionRuleInvalid
	}
	in.EmployeeIDs = employees
	if in.Name == "" || utf8.RuneCountInString(in.Name) > 100 || !validCommissionPersonnelRole(in.PersonnelRole) || (in.CalculationBasis != CommissionBasisRealizedProfit && in.CalculationBasis != CommissionBasisRealizedRevenue) || !in.RatePercent.IsPositive() || in.RatePercent.GreaterThan(decimal.NewFromInt(100)) || (in.EffectiveFrom != nil && !validFinanceDate(*in.EffectiveFrom)) || (in.EffectiveTo != nil && !validFinanceDate(*in.EffectiveTo)) || (in.EffectiveFrom != nil && in.EffectiveTo != nil && *in.EffectiveFrom > *in.EffectiveTo) || (in.Note != nil && utf8.RuneCountInString(*in.Note) > 500) {
		return CreateCommissionRuleInput{}, ErrCommissionRuleInvalid
	}
	// 新启用方案必须有当天或未来的生效起始日；未启用草稿可暂不设起始日。
	if in.Enabled && (in.EffectiveFrom == nil || *in.EffectiveFrom < in.Today) {
		return CreateCommissionRuleInput{}, ErrCommissionRuleInvalid
	}
	return in, nil
}

func validCommissionPersonnelRole(role CommissionPersonnelRole) bool {
	return role == CommissionRoleSales || role == CommissionRoleOperator || role == CommissionRoleCustomerService
}

// CalculateCommissionAmount 根据规则口径计算提成，亏损时按零基数计提。
func CalculateCommissionAmount(realizedRevenue, realizedProfit, ratePercent decimal.Decimal, basis CommissionCalculationBasis) (decimal.Decimal, decimal.Decimal, error) {
	base := realizedProfit
	if basis == CommissionBasisRealizedRevenue {
		base = realizedRevenue
	} else if basis != CommissionBasisRealizedProfit {
		return decimal.Zero, decimal.Zero, ErrCommissionRuleInvalid
	}
	if base.IsNegative() {
		base = decimal.Zero
	}
	return base.Round(8), base.Mul(ratePercent).Div(decimal.NewFromInt(100)).Round(8), nil
}

// CalculateCommissionLine 按单个订单计算本次核销收入对应的成本、已实现毛利和提成。
func CalculateCommissionLine(realizedRevenue, totalReceivable, totalPayable, ratePercent decimal.Decimal, basis CommissionCalculationBasis) (decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal, error) {
	if !realizedRevenue.IsPositive() || !totalReceivable.IsPositive() || totalPayable.IsNegative() {
		return decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, ErrCommissionSource
	}
	allocatedCost := realizedRevenue.Mul(totalPayable).Div(totalReceivable).Round(8)
	realizedProfit := realizedRevenue.Sub(allocatedCost).Round(8)
	base, amount, err := CalculateCommissionAmount(realizedRevenue, realizedProfit, ratePercent, basis)
	if err != nil {
		return decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, err
	}
	return allocatedCost, realizedProfit, base, amount, nil
}

// sameCommissionCreateIntent 判断幂等重放请求与已存在提成是否语义一致：
// 来源（核销/对冲二选一）、员工、人员身份与备注都必须相同；规则由服务端按
// 来源日期解析，不属于客户端意图的一部分。
func sameCommissionCreateIntent(old *FinanceCommission, in CreateCommissionInput) bool {
	return old.VerificationID == in.VerificationID && old.NettingID == in.NettingID && old.EmployeeID == in.EmployeeID && old.PersonnelRole == in.PersonnelRole && stringPointersEqual(old.Note, in.Note)
}

func (u *CommissionUsecase) Create(ctx context.Context, org, actor uuid.UUID, in CreateCommissionInput) (*FinanceCommission, error) {
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.Note = normalizedOptionalFinanceString(in.Note)
	if org == uuid.Nil || actor == uuid.Nil || !validCommissionSource(in.VerificationID, in.NettingID) || in.EmployeeID == uuid.Nil || !validCommissionPersonnelRole(in.PersonnelRole) || in.IdempotencyKey == "" || utf8.RuneCountInString(in.IdempotencyKey) > 128 || (in.Note != nil && utf8.RuneCountInString(*in.Note) > 500) {
		return nil, ErrCommissionInvalid
	}
	if u.transactor == nil {
		return nil, ErrCommissionInvalid
	}
	if old, err := u.repo.GetByKey(ctx, org, in.IdempotencyKey); err != nil {
		return nil, err
	} else if old != nil {
		if !sameCommissionCreateIntent(old, in) {
			return nil, ErrCommissionDuplicate
		}
		return old, nil
	}
	id := uuid.Must(uuid.NewV7())
	commissionNo, err := u.config.NextNumber(ctx, org, DocumentTypeCommission)
	if err != nil {
		return nil, err
	}
	c := &FinanceCommission{ID: id, OrganizationID: org, CommissionNo: commissionNo, IdempotencyKey: in.IdempotencyKey, VerificationID: in.VerificationID, NettingID: in.NettingID, EmployeeID: in.EmployeeID, PersonnelRole: in.PersonnelRole, Status: CommissionDraft, Note: in.Note, Version: 1}
	// 生成上下文读取与提成写入在同一共享事务内完成；CNY 快照不依赖预览结果，
	// 按事务内固化（原币记账恒等口径，无需再解析外部汇率）。
	err = u.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		generation, transactionErr := u.repo.GetGenerationContext(txCtx, org, in.VerificationID, in.NettingID)
		if transactionErr != nil {
			return transactionErr
		}
		snapshot, transactionErr := ResolveCommissionCNYRate(generation.BaseCurrency, generation.CommissionDate, decimal.Zero)
		if transactionErr != nil {
			return transactionErr
		}
		return u.repo.Create(txCtx, org, c, snapshot, commissionCreateAudit(org, actor, id, snapshot))
	})
	if err == nil {
		return u.repo.Get(ctx, org, c.ID)
	}
	// 并发重试可能在预查后命中幂等唯一索引；仅在请求语义一致时重放原结果。
	old, lookupErr := u.repo.GetByKey(ctx, org, in.IdempotencyKey)
	if lookupErr == nil && old != nil && sameCommissionCreateIntent(old, in) {
		return old, nil
	}
	return nil, err
}

func commissionCreateAudit(org, actor, id uuid.UUID, snapshot *CommissionCNYSnapshot) *AuditEvent {
	event := commissionAudit(org, actor, id, "finance.commission.create")
	details := map[string]string{
		"commission_date":   snapshot.CommissionDate,
		"cny.exchange_rate": snapshot.ExchangeRate.StringFixed(8),
		"cny.rate_date":     snapshot.ExchangeRateDate,
		"cny.source":        snapshot.ExchangeRateSource,
	}
	if snapshot.ExchangeRateSettingID != nil {
		details["cny.setting_id"] = snapshot.ExchangeRateSettingID.String()
	}
	event.Details = details
	return event
}

func (u *CommissionUsecase) Confirm(ctx context.Context, org, actor, id uuid.UUID, version uint64) (*FinanceCommission, error) {
	return u.transition(ctx, org, actor, id, version, CommissionConfirmed, "")
}
func (u *CommissionUsecase) MarkPaid(ctx context.Context, org, actor, id uuid.UUID, version uint64) (*FinanceCommission, error) {
	return u.transition(ctx, org, actor, id, version, CommissionPaid, "")
}
func (u *CommissionUsecase) Cancel(ctx context.Context, org, actor, id uuid.UUID, version uint64, reason string) (*FinanceCommission, error) {
	return u.transition(ctx, org, actor, id, version, CommissionCancelled, strings.TrimSpace(reason))
}

func (u *CommissionUsecase) CreateAdjustment(ctx context.Context, org, actor uuid.UUID, in CreateCommissionAdjustmentInput) (*FinanceCommissionAdjustment, error) {
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.Reason = strings.TrimSpace(in.Reason)
	in.Note = normalizedOptionalFinanceString(in.Note)
	if org == uuid.Nil || actor == uuid.Nil || in.CommissionID == uuid.Nil || in.OrderID == uuid.Nil ||
		(in.Direction != CommissionAdjustmentIncrease && in.Direction != CommissionAdjustmentDecrease) ||
		!in.Amount.IsPositive() || !totalAmountPattern.MatchString(in.Amount.String()) || in.Reason == "" ||
		utf8.RuneCountInString(in.Reason) > 500 || in.IdempotencyKey == "" || utf8.RuneCountInString(in.IdempotencyKey) > 128 ||
		(in.Note != nil && utf8.RuneCountInString(*in.Note) > 500) {
		return nil, ErrCommissionAdjustmentInvalid
	}
	if old, err := u.repo.GetAdjustmentByKey(ctx, org, in.IdempotencyKey); err != nil {
		return nil, err
	} else if old != nil {
		if !sameCommissionAdjustmentIntent(old, in) {
			return nil, ErrCommissionAdjustmentInvalid
		}
		return old, nil
	}
	item := &FinanceCommissionAdjustment{
		ID: uuid.Must(uuid.NewV7()), OrganizationID: org, CommissionID: in.CommissionID, OrderID: in.OrderID,
		IdempotencyKey: in.IdempotencyKey, Direction: in.Direction, SourceType: CommissionAdjustmentSourceManual, Status: CommissionDraft,
		Amount: in.Amount.Round(8), Reason: in.Reason, Note: in.Note, Version: 1,
	}
	created, err := u.repo.CreateAdjustment(ctx, org, actor, item, commissionAdjustmentAudit(org, actor, item.ID, "finance.commission_adjustment.create"))
	if err == nil {
		return created, nil
	}
	old, lookupErr := u.repo.GetAdjustmentByKey(ctx, org, in.IdempotencyKey)
	if lookupErr == nil && old != nil && sameCommissionAdjustmentIntent(old, in) {
		return old, nil
	}
	return nil, err
}

func sameCommissionAdjustmentIntent(old *FinanceCommissionAdjustment, in CreateCommissionAdjustmentInput) bool {
	return old.CommissionID == in.CommissionID && old.OrderID == in.OrderID && old.Direction == in.Direction &&
		old.Amount.Equal(in.Amount.Round(8)) && old.Reason == in.Reason && stringPointersEqual(old.Note, in.Note)
}

func (u *CommissionUsecase) ConfirmAdjustment(ctx context.Context, org, actor, id uuid.UUID, version uint64) (*FinanceCommissionAdjustment, error) {
	return u.transitionAdjustment(ctx, org, actor, id, version, CommissionConfirmed, "")
}

func (u *CommissionUsecase) MarkAdjustmentPaid(ctx context.Context, org, actor, id uuid.UUID, version uint64) (*FinanceCommissionAdjustment, error) {
	return u.transitionAdjustment(ctx, org, actor, id, version, CommissionPaid, "")
}

func (u *CommissionUsecase) CancelAdjustment(ctx context.Context, org, actor, id uuid.UUID, version uint64, reason string) (*FinanceCommissionAdjustment, error) {
	return u.transitionAdjustment(ctx, org, actor, id, version, CommissionCancelled, strings.TrimSpace(reason))
}

func (u *CommissionUsecase) transitionAdjustment(ctx context.Context, org, actor, id uuid.UUID, version uint64, target CommissionStatus, reason string) (*FinanceCommissionAdjustment, error) {
	if org == uuid.Nil || actor == uuid.Nil || id == uuid.Nil || version == 0 ||
		(target == CommissionCancelled && (reason == "" || utf8.RuneCountInString(reason) > 500)) {
		return nil, ErrCommissionAdjustmentInvalid
	}
	return u.repo.TransitionAdjustment(ctx, org, id, actor, version, target, reason, commissionAdjustmentAudit(org, actor, id, "finance.commission_adjustment."+strings.ToLower(string(target))))
}
func (u *CommissionUsecase) transition(ctx context.Context, org, actor, id uuid.UUID, version uint64, target CommissionStatus, reason string) (*FinanceCommission, error) {
	if org == uuid.Nil || actor == uuid.Nil || id == uuid.Nil || version == 0 || (target == CommissionCancelled && (reason == "" || utf8.RuneCountInString(reason) > 500)) {
		return nil, ErrCommissionInvalid
	}
	return u.repo.Transition(ctx, org, id, actor, version, target, reason, commissionAudit(org, actor, id, "finance.commission."+strings.ToLower(string(target))))
}
func commissionAudit(org, actor, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &org, UserID: &actor, Action: action, Result: "success", ResourceType: "finance_commission", ResourceID: id.String()}
}
func commissionRuleAudit(org, actor, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &org, UserID: &actor, Action: action, Result: "success", ResourceType: "finance_commission_rule", ResourceID: id.String()}
}

func commissionAdjustmentAudit(org, actor, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &org, UserID: &actor, Action: action, Result: "success", ResourceType: "finance_commission_adjustment", ResourceID: id.String()}
}
