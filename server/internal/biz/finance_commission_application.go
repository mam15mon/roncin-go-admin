package biz

import (
	"context"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

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
	// ErrCommissionApplicationConflict 月度唯一键下的状态门禁：当月已存在任何
	// 状态的申请头时空提交稳定拒绝，重提必须走显式申请 ID 的 Resubmit 路径；
	// 并发提交唯一索引兜底同样映射到该错误。
	ErrCommissionApplicationConflict = errors.Conflict("FINANCE_COMMISSION_APPLICATION_CONFLICT", "该月申请已存在，请在申请详情中查看进度或重新提交")
	// ErrCommissionApplicationSourceConflict 提交事务内的来源变更冲突（design §3.3）：
	// 已占用提成、指纹失效的 DRAFT、重提时明细失效等任一命中即整体回滚。
	ErrCommissionApplicationSourceConflict = errors.Conflict("FINANCE_COMMISSION_APPLICATION_SOURCE_CONFLICT", "部分提成事实已变化或已被占用，请刷新后重试")
	// ErrCommissionApplicationStatusConflict 审批门禁（design §5.2/§5.3）：申请头
	// expected_version 与 PENDING_REVIEW 双重校验不通过——并发批准/驳回/重提只有
	// 一个事务成功，后到者按「已被处理」稳定拒绝，不做部分批准。
	ErrCommissionApplicationStatusConflict = errors.Conflict("FINANCE_COMMISSION_APPLICATION_STATUS_CONFLICT", "该申请已被处理，请刷新后重试")
	// ErrCommissionApplicationNoValidLines 显式重提门禁：原绑定明细按上游当前
	// 事实逐笔复核后全部失效（来源失效或提成已被取消/冲销），无有效明细可提交，
	// 整体回滚且申请保持 REJECTED。
	ErrCommissionApplicationNoValidLines = errors.Conflict("FINANCE_COMMISSION_APPLICATION_NO_VALID_LINES", "申请内已无有效明细，无法重新提交")
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
// EmployeeName/OrganizationName 为财务列表/详情的展示投影，按 ID 服务端解析，
// 员工本人读取路径不填充。
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
	EmployeeName, OrganizationName                  string
}

// FinanceCommissionApplicationLine 是申请明细的提成事实快照投影；commission_id
// 全局唯一——一个提成事实至多进入一张申请，被驳回也不回流公共池。CommissionNo
// 为财务下钻按提成单解析的展示投影，员工本人读取路径不填充。
type FinanceCommissionApplicationLine struct {
	ID, OrganizationID, EmployeeID, ApplicationID, CommissionID uuid.UUID
	CommissionNo                                                string
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

// CommissionApplicationFinanceFilter 是财务申请列表的服务端分页筛选：组织范围
// 由 Service 按 system.finance.commission.read 显式解析后传入，跨组织查询必须
// 显式携带组织；员工、状态与提交月为可选过滤。
type CommissionApplicationFinanceFilter struct {
	Page, PageSize   int
	EmployeeID       uuid.UUID
	Status           CommissionApplicationStatus
	ApplicationMonth string
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
	// Submit 提交本人月度申请（仅用于新建）：会话固定本人与当前组织，服务端以
	// 财务业务日期推导提交月与覆盖截止日，在共享事务内全量重算候选并固化申请头
	// 与明细；当月已存在任何状态的申请头时稳定冲突，不承担重提语义。
	Submit(ctx context.Context, scope WorkbenchScope) (*FinanceCommissionApplication, error)
	// Resubmit 显式重提本人被驳回的月度申请（design §5.3）：application_id +
	// expected_version 定位原申请，会话固定本人与当前组织；事务内按上游当前
	// 事实逐笔刷新明细快照金额/方案/指纹（来源失效或提成已被取消/冲销的明细
	// 剔除并留审计），重算笔数/合计，版本递增并回到 PENDING_REVIEW。
	Resubmit(ctx context.Context, scope WorkbenchScope, id uuid.UUID, expectedVersion uint64) (*FinanceCommissionApplication, error)
	// ListMyApplications 返回本人申请历史分页。
	ListMyApplications(ctx context.Context, scope WorkbenchScope, filter WorkbenchApplicationFilter) (*PagedList[*FinanceCommissionApplication], error)
	// GetMyApplication 返回本人单张申请详情；他人或跨组织申请按不存在处理。
	GetMyApplication(ctx context.Context, scope WorkbenchScope, id uuid.UUID) (*FinanceCommissionApplicationDetail, error)
	// ListForOrganization 财务申请列表：组织范围显式来自 read 权限解析，按
	// 员工/状态/提交月过滤并服务端分页；填充员工与组织展示名。
	ListForOrganization(ctx context.Context, organizationIDs []uuid.UUID, filter CommissionApplicationFinanceFilter) (*PagedList[*FinanceCommissionApplication], error)
	// GetForOrganization 财务申请详情：申请头审计字段、明细快照与提成单号投影；
	// 跨组织申请按不存在处理。
	GetForOrganization(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*FinanceCommissionApplicationDetail, error)
	// Approve 整单批准（design §5.2）：单个共享事务内锁定申请头（expected_version
	// + PENDING_REVIEW），按明细提成 ID 排序复核指纹与阻断事实后整批确认 DRAFT
	// 提成；任一明细失效整体回滚，不做部分批准。
	Approve(ctx context.Context, organizationIDs []uuid.UUID, decisionMaker, id uuid.UUID, expectedVersion uint64) (*FinanceCommissionApplication, error)
	// Reject 整单驳回（design §5.3）：原因必填；只改变申请头状态与决策审计并
	// 递增版本，明细与提成单保持不变，员工可在原申请上重提。
	Reject(ctx context.Context, organizationIDs []uuid.UUID, decisionMaker, id uuid.UUID, expectedVersion uint64, reason string) (*FinanceCommissionApplication, error)
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

// Submit 提交本人月度申请：无业务参数，主体与组织固定取自会话；仅用于新建，
// 当月已存在申请头时稳定冲突。
func (u *FinanceCommissionApplicationUsecase) Submit(ctx context.Context, scope WorkbenchScope) (*FinanceCommissionApplication, error) {
	if !validWorkbenchScope(scope) {
		return nil, ErrWorkbenchInvalid
	}
	if !scope.CanOperateBusiness {
		return nil, ErrOperatingCompanyRequired
	}
	return u.repo.Submit(ctx, scope)
}

// Resubmit 显式重提本人被驳回的月度申请：application_id + expected_version
// 定位原申请，主体与组织固定取自会话；刷新快照、剔除失效明细与版本递增由
// 仓储事务实现承担。
func (u *FinanceCommissionApplicationUsecase) Resubmit(ctx context.Context, scope WorkbenchScope, id uuid.UUID, expectedVersion uint64) (*FinanceCommissionApplication, error) {
	if !validWorkbenchScope(scope) || id == uuid.Nil || expectedVersion == 0 {
		return nil, ErrCommissionApplicationInvalid
	}
	if !scope.CanOperateBusiness {
		return nil, ErrOperatingCompanyRequired
	}
	return u.repo.Resubmit(ctx, scope, id, expectedVersion)
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

// ListForOrganization 财务申请列表：组织范围必须显式来自 read 权限解析，
// 分页、状态与提交月过滤在领域入口统一校验。
func (u *FinanceCommissionApplicationUsecase) ListForOrganization(ctx context.Context, organizationIDs []uuid.UUID, filter CommissionApplicationFinanceFilter) (*PagedList[*FinanceCommissionApplication], error) {
	if !validCommissionOrganizationIDs(organizationIDs) || !ValidListPagination(filter.Page, filter.PageSize) ||
		(filter.Status != "" && !validCommissionApplicationStatus(filter.Status)) ||
		(filter.ApplicationMonth != "" && !ValidCommissionApplicationMonth(filter.ApplicationMonth)) {
		return nil, ErrCommissionApplicationInvalid
	}
	return u.repo.ListForOrganization(ctx, organizationIDs, filter)
}

// GetForOrganization 财务申请详情：组织范围显式来自 read 权限解析，跨组织
// 申请按不存在处理，不泄露记录事实。
func (u *FinanceCommissionApplicationUsecase) GetForOrganization(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*FinanceCommissionApplicationDetail, error) {
	if !validCommissionOrganizationIDs(organizationIDs) || id == uuid.Nil {
		return nil, ErrCommissionApplicationNotFound
	}
	return u.repo.GetForOrganization(ctx, organizationIDs, id)
}

// Approve 整单批准：申请头 expected_version 防并发覆盖，决策人取自当前会话；
// 锁序、指纹复核、未建账费用阻断与整批状态迁移由仓储事务实现承担。
func (u *FinanceCommissionApplicationUsecase) Approve(ctx context.Context, organizationIDs []uuid.UUID, decisionMaker, id uuid.UUID, expectedVersion uint64) (*FinanceCommissionApplication, error) {
	if !validCommissionOrganizationIDs(organizationIDs) || decisionMaker == uuid.Nil || id == uuid.Nil || expectedVersion == 0 {
		return nil, ErrCommissionApplicationInvalid
	}
	return u.repo.Approve(ctx, organizationIDs, decisionMaker, id, expectedVersion)
}

// Reject 整单驳回：原因必填（空原因或超长原因稳定拒绝），决策人取自当前会话。
func (u *FinanceCommissionApplicationUsecase) Reject(ctx context.Context, organizationIDs []uuid.UUID, decisionMaker, id uuid.UUID, expectedVersion uint64, reason string) (*FinanceCommissionApplication, error) {
	reason = strings.TrimSpace(reason)
	if !validCommissionOrganizationIDs(organizationIDs) || decisionMaker == uuid.Nil || id == uuid.Nil ||
		expectedVersion == 0 || reason == "" || utf8.RuneCountInString(reason) > 500 {
		return nil, ErrCommissionApplicationInvalid
	}
	return u.repo.Reject(ctx, organizationIDs, decisionMaker, id, expectedVersion, reason)
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

// CommissionApplicationResubmitAudit 构造显式重提审计：在提交审计形状之上记录
// 本次重提的快照刷新结果——刷新明细数、剔除明细数与被剔除的提成事实 ID 集合
// （审计留痕；被剔除明细的提成按既有冲销/调整模型处理）。不记录完整收入明细。
func CommissionApplicationResubmitAudit(org, actor, applicationID uuid.UUID, month, coverageTo string, count, refreshed, removed int, total decimal.Decimal, removedCommissionIDs []string) *AuditEvent {
	audit := CommissionApplicationSubmitAudit(org, actor, applicationID, CommissionApplicationSubmitAction(true), month, coverageTo, count, total)
	audit.Details["refreshed_count"] = strconv.Itoa(refreshed)
	audit.Details["removed_count"] = strconv.Itoa(removed)
	if len(removedCommissionIDs) > 0 {
		audit.Details["removed_commission_ids"] = strings.Join(removedCommissionIDs, ",")
	}
	return audit
}

// CommissionApplicationSubmitAction 返回提交或重提的审计动作名。
func CommissionApplicationSubmitAction(resubmit bool) string {
	if resubmit {
		return "finance.commission_application.resubmit"
	}
	return "finance.commission_application.submit"
}

// CommissionApplicationDecisionAction 返回财务整单批准或驳回的审计动作名。
func CommissionApplicationDecisionAction(approved bool) string {
	if approved {
		return "finance.commission_application.approve"
	}
	return "finance.commission_application.reject"
}

// CommissionApplicationDecisionAudit 构造财务整单批准/驳回审计：记录决策人、
// 申请员工、申请月、覆盖截止日、笔数、金额快照与决策后版本；驳回携带必填原因。
// 不记录完整收入明细。
func CommissionApplicationDecisionAudit(org, decisionMaker, applicationID, employeeID uuid.UUID, action, month, coverageTo string, count int, total decimal.Decimal, version uint64, reason string) *AuditEvent {
	details := map[string]string{
		"employee_id":       employeeID.String(),
		"application_month": month,
		"coverage_to":       coverageTo,
		"commission_count":  strconv.Itoa(count),
		"total_amount":      total.StringFixed(8),
		"version":           strconv.FormatUint(version, 10),
	}
	if reason != "" {
		details["reason"] = reason
	}
	return &AuditEvent{
		OrganizationID: &org,
		UserID:         &decisionMaker,
		Action:         action,
		Result:         "success",
		ResourceType:   "finance_commission_application",
		ResourceID:     applicationID.String(),
		Details:        details,
	}
}
