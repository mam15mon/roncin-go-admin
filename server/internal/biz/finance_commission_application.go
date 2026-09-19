package biz

import (
	"context"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// 月度提成申请的稳定领域错误：提交/重提、候选读取与本人详情的统一语义。
var (
	// ErrCommissionApplicationNotFound 本人详情查询未命中本人申请：他人或跨组织
	// 申请一律按不存在处理，不泄露记录事实。
	ErrCommissionApplicationNotFound = errors.NotFound("FINANCE_COMMISSION_APPLICATION_NOT_FOUND", "提成申请不存在")
	ErrCommissionApplicationInvalid  = errors.BadRequest("FINANCE_COMMISSION_APPLICATION_INVALID", "提成申请参数不合法")
	// ErrCommissionApplicationEmployeeInvalid 提交主体的组织成员关系缺失或停用。
	ErrCommissionApplicationEmployeeInvalid = errors.Conflict("FINANCE_COMMISSION_APPLICATION_EMPLOYEE_INVALID", "当前用户不是当前组织的有效成员")
	// ErrCommissionApplicationEmpty 截止日以前没有任何合格未申请提成：拒绝提交，
	// 不创建空申请。
	ErrCommissionApplicationEmpty = errors.Conflict("FINANCE_COMMISSION_APPLICATION_EMPTY", "截至上一自然月末暂无可申请的合格提成")
	// ErrCommissionApplicationConflict 月度唯一键下的状态门禁：申请已在
	// PENDING_REVIEW 或 APPROVED 时重复/并发提交稳定拒绝。
	ErrCommissionApplicationConflict = errors.Conflict("FINANCE_COMMISSION_APPLICATION_CONFLICT", "该月申请已提交或已批准，不能重复提交")
	// ErrCommissionApplicationSourceConflict 提交事务内的来源变更冲突（design §3.3）：
	// 已占用提成、指纹失效的 DRAFT、重提时明细失效等任一命中即整体回滚。
	ErrCommissionApplicationSourceConflict = errors.Conflict("FINANCE_COMMISSION_APPLICATION_SOURCE_CONFLICT", "部分提成事实已变化或已被占用，请刷新后重试")
)

// CommissionApplicationStatus 是月度提成申请头状态，与申请头 CHECK 同源。
type CommissionApplicationStatus string

const (
	CommissionApplicationPendingReview CommissionApplicationStatus = "PENDING_REVIEW"
	CommissionApplicationRejected      CommissionApplicationStatus = "REJECTED"
	CommissionApplicationApproved      CommissionApplicationStatus = "APPROVED"
)

// FinanceCommissionApplication 是员工在组织内按提交自然月形成的申请头投影；
// 金额与覆盖月份为提交当时快照，驳回后原申请重提沿用同一 ID 与申请月份。
type FinanceCommissionApplication struct {
	ID, OrganizationID, EmployeeID                  uuid.UUID
	ApplicationMonth, CoverageTo                    string
	Status                                          CommissionApplicationStatus
	Version                                         uint64
	CommissionCount                                 int
	BaseCurrency                                    string
	TotalCommissionAmount, TotalCNYCommissionAmount decimal.Decimal
	SubmittedAt                                     time.Time
	SubmittedBy                                     uuid.UUID
	DecidedAt                                       *time.Time
	DecidedBy                                       *uuid.UUID
	DecisionReason                                  *string
	CreatedAt, UpdatedAt                            time.Time
}

// FinanceCommissionApplicationLine 是申请明细的提成事实快照投影；commission_id
// 全局唯一——一个提成事实至多进入一张申请，被驳回也不回流公共池。
type FinanceCommissionApplicationLine struct {
	ID, OrganizationID, EmployeeID, ApplicationID, CommissionID uuid.UUID
	CommissionDate                                              string
	VerificationID                                              *uuid.UUID
	VerificationNo                                              *string
	NettingID                                                   *uuid.UUID
	NettingNo                                                   *string
	PersonnelRole                                               CommissionPersonnelRole
	RuleID                                                      *uuid.UUID
	RuleVersion                                                 uint64
	RuleName                                                    *string
	CalculationBasis                                            *string
	BaseCurrency                                                string
	CommissionAmount, CNYCommissionAmount                       decimal.Decimal
	SourceFingerprint                                           string
	CreatedAt                                                   time.Time
}

// FinanceCommissionApplicationDetail 是本人/财务申请详情：申请头与明细快照。
type FinanceCommissionApplicationDetail struct {
	Application *FinanceCommissionApplication
	Lines       []*FinanceCommissionApplicationLine
}

// WorkbenchApplicationCandidate 是单条可申请提成候选：以核销/对冲来源为身份
// （恰好一个非空），金额为按现有计提口径的服务端计算值；尚未生成提成事实的
// 候选以提交申请时受控创建为准，候选本身不携带 commission_id。
type WorkbenchApplicationCandidate struct {
	VerificationID, NettingID uuid.UUID
	VerificationNo, NettingNo string
	// CommissionDate 是提成归属日期（YYYY-MM-DD）：核销日期或对冲确认日期。
	CommissionDate      string
	PersonnelRole       CommissionPersonnelRole
	RuleID              uuid.UUID
	RuleName            string
	CalculationBasis    CommissionCalculationBasis
	BaseCurrency        string
	CommissionAmount    decimal.Decimal
	CNYCommissionAmount decimal.Decimal
}

// WorkbenchApplicationSummary 是工作台申请摘要段：可申请=截至上一自然月末仍未
// 进入任何申请的合格提成（按提成归属月分组、精确解析）；本月累计中=当前自然月
// 尚未结束、只能累计展示的合格提成（沿用预计口径）；审批中/已批准为本人全部
// 历史申请计数；最近申请概要尚无申请时缺省。
type WorkbenchApplicationSummary struct {
	BaseCurrency       string
	ApplyGroups        []*WorkbenchApplyMonthGroup
	AccumulatingCount  int
	AccumulatingAmount decimal.Decimal
	PendingReviewCount int
	ApprovedCount      int
	LatestApplication  *WorkbenchApplicationBrief
}

// WorkbenchApplyMonthGroup 是单个提成归属月的可申请汇总：金额为本位币合计。
type WorkbenchApplyMonthGroup struct {
	CommissionMonth  string
	CommissionCount  int
	CommissionAmount decimal.Decimal
}

// WorkbenchApplicationBrief 是最近一张申请的最小概要，用于首页入口展示。
type WorkbenchApplicationBrief struct {
	ApplicationID         uuid.UUID
	ApplicationMonth      string
	Status                CommissionApplicationStatus
	CommissionCount       int
	TotalCommissionAmount decimal.Decimal
	SubmittedAt           time.Time
}

// WorkbenchApplicationCandidateFilter 是本人可申请候选的服务端分页筛选；
// CommissionMonth 按提成归属月（YYYY-MM）过滤，缺省返回全部可申请候选。
type WorkbenchApplicationCandidateFilter struct {
	Page, PageSize  int
	CommissionMonth string
}

// WorkbenchApplicationFilter 是本人申请历史的服务端分页筛选。
type WorkbenchApplicationFilter struct {
	Page, PageSize int
	Status         CommissionApplicationStatus
}

// CommissionApplicationPeriod 由服务端财务业务日期推导提交月与覆盖截止日：
// application_month 为提交发生的自然月（YYYY-MM），coverage_to 固定为前一自然月
// 最后一天（YYYY-MM-DD）。当前自然月来源因晚于 coverage_to 只能进入累计中，
// 结构上拒绝当月数据提前申请。
func CommissionApplicationPeriod(today string) (applicationMonth, coverageTo string, valid bool) {
	if !validFinanceDate(today) {
		return "", "", false
	}
	parsed, err := time.ParseInLocation("2006-01-02", today, time.UTC)
	if err != nil {
		return "", "", false
	}
	monthStart := time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC)
	return monthStart.Format("2006-01"), monthStart.AddDate(0, 0, -1).Format("2006-01-02"), true
}

// ValidCommissionApplicationMonth 校验 YYYY-MM 归属月过滤参数。
func ValidCommissionApplicationMonth(value string) bool {
	parsed, err := time.ParseInLocation("2006-01", value, time.UTC)
	if err != nil {
		return false
	}
	return parsed.Format("2006-01") == value
}

func validCommissionApplicationStatus(status CommissionApplicationStatus) bool {
	switch status {
	case CommissionApplicationPendingReview, CommissionApplicationRejected, CommissionApplicationApproved:
		return true
	default:
		return false
	}
}

// FinanceCommissionApplicationRepo 是月度提成申请仓储：候选解析与提交事务的
// 实现必须复用既有计提引擎口径（方案员工分配唯一命中、订单同身份归属、费用与
// 回款校验），并区分「可申请（精确全量）」与「本月累计中（有界预计）」两套语义。
type FinanceCommissionApplicationRepo interface {
	// ApplicationSummary 返回本人申请摘要段：可申请按归属月分组的精确笔数/金额、
	// 当前月累计概要与本人历史申请计数及最近申请。
	ApplicationSummary(ctx context.Context, scope WorkbenchScope, baseCurrency string) (*WorkbenchApplicationSummary, error)
	// ListMyCandidates 全量解析本人截至 coverage_to 的可申请候选并按归属月过滤、
	// 服务端分页；不套用工作台预计估算的有界上限。
	ListMyCandidates(ctx context.Context, scope WorkbenchScope, filter WorkbenchApplicationCandidateFilter) (*PagedList[*WorkbenchApplicationCandidate], error)
	// Submit 提交或重提本人月度申请：会话固定本人与当前组织，服务端以财务业务
	// 日期推导提交月与覆盖截止日，在共享事务内全量重算候选并固化申请头与明细。
	Submit(ctx context.Context, scope WorkbenchScope) (*FinanceCommissionApplication, error)
	// ListMyApplications 返回本人申请历史分页。
	ListMyApplications(ctx context.Context, scope WorkbenchScope, filter WorkbenchApplicationFilter) (*PagedList[*FinanceCommissionApplication], error)
	// GetMyApplication 返回本人单张申请详情；他人或跨组织申请按不存在处理。
	GetMyApplication(ctx context.Context, scope WorkbenchScope, id uuid.UUID) (*FinanceCommissionApplicationDetail, error)
}

// FinanceCommissionApplicationUsecase 月度提成申请用例：只做主体与参数校验后
// 委托仓储；提交事务（成员行锁、月度唯一键、来源固定锁序、DRAFT 受控创建与
// 明细快照固化）全部由仓储实现承担。
type FinanceCommissionApplicationUsecase struct {
	repo FinanceCommissionApplicationRepo
}

func NewFinanceCommissionApplicationUsecase(repo FinanceCommissionApplicationRepo) *FinanceCommissionApplicationUsecase {
	return &FinanceCommissionApplicationUsecase{repo: repo}
}

// ApplicationSummary 返回当前主体在当前工作区组织的申请摘要段。
func (u *FinanceCommissionApplicationUsecase) ApplicationSummary(ctx context.Context, scope WorkbenchScope, baseCurrency string) (*WorkbenchApplicationSummary, error) {
	if !validWorkbenchScope(scope) {
		return nil, ErrWorkbenchInvalid
	}
	return u.repo.ApplicationSummary(ctx, scope, baseCurrency)
}

// ListMyCandidates 返回本人可申请候选分页：归属月过滤必须为 YYYY-MM。
func (u *FinanceCommissionApplicationUsecase) ListMyCandidates(ctx context.Context, scope WorkbenchScope, filter WorkbenchApplicationCandidateFilter) (*PagedList[*WorkbenchApplicationCandidate], error) {
	filter.CommissionMonth = strings.TrimSpace(filter.CommissionMonth)
	if !validWorkbenchScope(scope) || !ValidListPagination(filter.Page, filter.PageSize) ||
		(filter.CommissionMonth != "" && !ValidCommissionApplicationMonth(filter.CommissionMonth)) {
		return nil, ErrWorkbenchInvalid
	}
	return u.repo.ListMyCandidates(ctx, scope, filter)
}

// Submit 提交或重提本人月度申请：无业务参数，主体与组织固定取自会话。
func (u *FinanceCommissionApplicationUsecase) Submit(ctx context.Context, scope WorkbenchScope) (*FinanceCommissionApplication, error) {
	if !validWorkbenchScope(scope) {
		return nil, ErrWorkbenchInvalid
	}
	return u.repo.Submit(ctx, scope)
}

// ListMyApplications 返回本人申请历史分页。
func (u *FinanceCommissionApplicationUsecase) ListMyApplications(ctx context.Context, scope WorkbenchScope, filter WorkbenchApplicationFilter) (*PagedList[*FinanceCommissionApplication], error) {
	if !validWorkbenchScope(scope) || !ValidListPagination(filter.Page, filter.PageSize) ||
		(filter.Status != "" && !validCommissionApplicationStatus(filter.Status)) {
		return nil, ErrWorkbenchInvalid
	}
	return u.repo.ListMyApplications(ctx, scope, filter)
}

// GetMyApplication 返回本人单张申请详情；参数缺失或他人申请按不存在处理。
func (u *FinanceCommissionApplicationUsecase) GetMyApplication(ctx context.Context, scope WorkbenchScope, id uuid.UUID) (*FinanceCommissionApplicationDetail, error) {
	if !validWorkbenchScope(scope) || id == uuid.Nil {
		return nil, ErrCommissionApplicationNotFound
	}
	return u.repo.GetMyApplication(ctx, scope, id)
}

// CommissionApplicationSubmitAudit 构造提交/重提审计：记录员工、申请月、覆盖
// 截止日、明细笔数与金额快照，不记录完整收入明细。
func CommissionApplicationSubmitAudit(org, actor, applicationID uuid.UUID, action string, month, coverageTo string, count int, total decimal.Decimal) *AuditEvent {
	return &AuditEvent{
		OrganizationID: &org,
		UserID:         &actor,
		Action:         action,
		Result:         "success",
		ResourceType:   "finance_commission_application",
		ResourceID:     applicationID.String(),
		Details: map[string]string{
			"employee_id":       actor.String(),
			"application_month": month,
			"coverage_to":       coverageTo,
			"commission_count":  decimal.NewFromInt(int64(count)).String(),
			"total_amount":      total.StringFixed(8),
		},
	}
}

// CommissionApplicationSubmitAction 返回提交或重提的审计动作名。
func CommissionApplicationSubmitAction(resubmit bool) string {
	if resubmit {
		return "finance.commission_application.resubmit"
	}
	return "finance.commission_application.submit"
}
