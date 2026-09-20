package biz

import (
	"context"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ErrWorkbenchInvalid 是工作台读接口的统一参数错误：主体、组织、分页、状态或
// 日期过滤不合法时返回，不区分内部细节。
var ErrWorkbenchInvalid = errors.BadRequest("WORKBENCH_INVALID_ARGUMENT", "工作台查询参数不合法")

// 工作台聚合的有界上限：Overview 只携带少量记录，明细统一分页且 pageSize ≤ 200。
const (
	// WorkbenchOverviewRecentOrderLimit 是 Overview 内置的近期订单条数上限。
	WorkbenchOverviewRecentOrderLimit = 5
	// WorkbenchEstimatedScanLimit 是「预计可计提」候选来源的有界扫描上限：
	// 超出部分不参与金额合计，只以 HasMore 标记金额为不完整下界。
	WorkbenchEstimatedScanLimit = 20
	// WorkbenchSupplementScanLimit 是待审批补录申请的有界扫描上限。
	WorkbenchSupplementScanLimit = 50
	// WorkbenchSupplementItemLimit 是 Overview 摘要内返回的补录待办条数上限。
	WorkbenchSupplementItemLimit = 5
)

// WorkbenchScope 是工作台读模型的服务端主体范围：全部字段由 Service 从会话与
// 权限范围解析，不信任任何客户端提交的员工或组织改写参数。
type WorkbenchScope struct {
	// OrganizationID 是当前工作区组织；所有查询固定限定在该组织内。
	OrganizationID uuid.UUID
	// UserID 是当前登录用户；本人接口固定 employee_id/user_id = 该值。
	UserID uuid.UUID
	// Today 是服务端统一财务业务日期（YYYY-MM-DD），用于门禁的「尚未结束」判定。
	Today string
	// Now 是服务端当前时刻，用于本年/本月已发累计窗口（上海业务时区）。
	Now time.Time
	// CommissionReadable 表示当前用户对当前组织具备 system.finance.commission.read。
	CommissionReadable bool
	// CommissionManageable 表示当前用户对当前组织具备 system.finance.commission.manage。
	CommissionManageable bool
	// CanOperateBusiness 决定本人业务申请是否可办理，不影响只读查询。
	CanOperateBusiness bool
	// IsBootstrapAdmin 透传给补录审批实时资格判定。
	IsBootstrapAdmin bool
}

func validWorkbenchScope(scope WorkbenchScope) bool {
	return scope.OrganizationID != uuid.Nil && scope.UserID != uuid.Nil && validFinanceDate(scope.Today) && !scope.Now.IsZero()
}

// WorkbenchAssignmentWindow 是「方案区间 ∩ 员工分配区间」的实际有效闭区间投影；
// EffectiveTo 为空串表示正无穷。
type WorkbenchAssignmentWindow struct {
	EffectiveFrom string
	EffectiveTo   string
}

// WorkbenchActualAssignmentWindow 计算方案与分配的区间交集：任一侧未启用直接
// 失效；起点取两者较晚者（空串视为负无穷），终点取两者较早者（空串视为正无穷），
// 起点晚于终点时区间为空。
func WorkbenchActualAssignmentWindow(ruleEnabled bool, ruleFrom, ruleTo, assignmentFrom, assignmentTo string) (WorkbenchAssignmentWindow, bool) {
	if !ruleEnabled {
		return WorkbenchAssignmentWindow{}, false
	}
	from := assignmentFrom
	if ruleFrom != "" && (from == "" || ruleFrom > from) {
		from = ruleFrom
	}
	to := assignmentTo
	if ruleTo != "" && (to == "" || ruleTo < to) {
		to = ruleTo
	}
	if to != "" && from > to {
		return WorkbenchAssignmentWindow{}, false
	}
	return WorkbenchAssignmentWindow{EffectiveFrom: from, EffectiveTo: to}, true
}

// WorkbenchEligibilityFacts 是仓储门禁查询的原始事实，由 biz 统一判定门禁语义。
type WorkbenchEligibilityFacts struct {
	// ActiveWindows 是未取消且实际区间尚未结束的分配段投影（方案已启用）。
	ActiveWindows []WorkbenchAssignmentWindow
	// HasCommissionHistory 表示存在本人 DRAFT/CONFIRMED/PAID 历史提成单。
	HasCommissionHistory bool
	// HasAdjustmentHistory 表示存在本人 DRAFT/CONFIRMED/PAID 历史提成调整单。
	HasAdjustmentHistory bool
}

// ResolveWorkbenchEligibility 按设计 §4.2 判定提成模块门禁：
// 当前/未来有效方案员工分配（方案启用且实际区间未结束）或本人非取消历史提成/
// 调整记录任一成立即为真；订单协作、提成归属与已取消记录不能开启门禁。
// 第二返回值是最近未来实际生效日（YYYY-MM-DD），供页面说明方案尚未生效。
func ResolveWorkbenchEligibility(facts WorkbenchEligibilityFacts, today string) (bool, string) {
	nextEffectiveDate := ""
	for _, window := range facts.ActiveWindows {
		if window.EffectiveFrom > today && (nextEffectiveDate == "" || window.EffectiveFrom < nextEffectiveDate) {
			nextEffectiveDate = window.EffectiveFrom
		}
	}
	eligible := len(facts.ActiveWindows) > 0 || facts.HasCommissionHistory || facts.HasAdjustmentHistory
	if !eligible {
		nextEffectiveDate = ""
	}
	return eligible, nextEffectiveDate
}

// WorkbenchPaidPeriodRange 返回本年/本月已发累计窗口起点（上海业务时区零点）。
// 数据层以 paid_at >= 起点过滤 PAID 提成单。
func WorkbenchPaidPeriodRange(now time.Time) (yearStart, monthStart time.Time) {
	local := now.In(financeBusinessLocation)
	yearStart = time.Date(local.Year(), 1, 1, 0, 0, 0, 0, financeBusinessLocation)
	monthStart = time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, financeBusinessLocation)
	return yearStart, monthStart
}

// WorkbenchOverview 是 Overview 接口的领域投影。
type WorkbenchOverview struct {
	Eligible          bool
	NextEffectiveDate string
	BaseCurrency      string
	// CommissionSummary 仅在门禁为真时由仓储填充；为假时保持 nil，
	// 服务端不返回提成金额模块。
	CommissionSummary *WorkbenchCommissionSummary
	RecentOrders      []*WorkbenchRecentOrder
	Todos             WorkbenchTodoSummary
	// Finance 仅在当前用户对当前组织具备财务提成读取权限时由仓储填充。
	Finance *WorkbenchFinanceSummary
}

// WorkbenchCommissionSummary 汇总本人提成与冲减分桶：全部金额为当前组织本位币，
// 不同组织本位币不合计；CANCELLED 不计入任何有效金额。
type WorkbenchCommissionSummary struct {
	BaseCurrency string
	// DRAFT=待财务确认；CONFIRMED=已确认待发；PAID=已发放。
	DraftCount      int
	DraftAmount     decimal.Decimal
	ConfirmedCount  int
	ConfirmedAmount decimal.Decimal
	PaidCount       int
	PaidAmount      decimal.Decimal
	PaidThisYear    decimal.Decimal
	PaidThisMonth   decimal.Decimal
	// 冲减三阶段：DRAFT=待处理、尚未扣回；CONFIRMED=已确认、尚未实际扣回；
	// PAID=已扣回。仅统计 DECREASE 调整单。
	DecreaseDraftCount      int
	DecreaseDraftAmount     decimal.Decimal
	DecreaseConfirmedCount  int
	DecreaseConfirmedAmount decimal.Decimal
	DecreasePaidCount       int
	DecreasePaidAmount      decimal.Decimal
	// Estimated 是预计可计提机会摘要；无可靠估算时为 nil。
	Estimated *WorkbenchEstimatedOpportunity
}

// WorkbenchEstimatedOpportunity 是预计可计提机会摘要：金额不是应发承诺，
// HasMore 为真时金额只是有界扫描得到的不完整下界。
type WorkbenchEstimatedOpportunity struct {
	OpportunityCount int
	EstimatedAmount  decimal.Decimal
	HasMore          bool
}

// WorkbenchRecentOrder 是本人真实协作的近期海运出口订单投影。
type WorkbenchRecentOrder struct {
	OrderID           uuid.UUID
	OrderNo           string
	BusinessType      string
	CustomerName      string
	FlowStatus        string
	TerminationStatus string
	OrderDate         string
	CreatedAt         time.Time
}

// WorkbenchTodoSummary 只聚合现有事实可准确判定的作业待办。
type WorkbenchTodoSummary struct {
	// DraftFeeCount 是本人协作订单上状态为 DRAFT 的费用数量。
	DraftFeeCount int
	// OpenAbnormalCount 是本人协作订单上未解决异常的数量。
	OpenAbnormalCount int
}

// WorkbenchFinanceSummary 是财务/审批摘要：只有计数、少量记录与权限标记，
// 实际动作仍走原业务接口并重新鉴权。
type WorkbenchFinanceSummary struct {
	CanReadCommission              bool
	CanManageCommission            bool
	ConfirmedCommissionCount       int
	PendingDecreaseCount           int
	PendingSupplementApprovalCount int
	PendingSupplementApprovals     []*WorkbenchSupplementApprovalItem
	SupplementApprovalsTruncated   bool
}

// WorkbenchSupplementApprovalItem 是当前用户实时有权审批的 PENDING 补录申请。
type WorkbenchSupplementApprovalItem struct {
	RequestID       uuid.UUID
	OrderID         uuid.UUID
	OrderNo         string
	FeeCode         string
	FeeName         string
	Currency        string
	Amount          decimal.Decimal
	Reason          string
	RequestedByName string
	RequestedAt     time.Time
}

// WorkbenchMyCommission 是本人提成单下钻投影，附带本人调整明细。
type WorkbenchMyCommission struct {
	ID               uuid.UUID
	CommissionNo     string
	Status           CommissionStatus
	PersonnelRole    CommissionPersonnelRole
	RuleName         *string
	CalculationBasis *string
	BaseCurrency     string
	CommissionAmount decimal.Decimal
	CommissionDate   string
	VerificationNo   *string
	NettingNo        *string
	CreatedAt        time.Time
	Adjustments      []*WorkbenchMyCommissionAdjustment
}

// WorkbenchMyCommissionAdjustment 是本人调整单最小投影。
type WorkbenchMyCommissionAdjustment struct {
	ID           uuid.UUID
	AdjustmentNo string
	Direction    CommissionAdjustmentDirection
	Status       CommissionStatus
	Amount       decimal.Decimal
	Reason       string
	CreatedAt    time.Time
}

// WorkbenchMyReceivable 是与本人提成归属相关的已确认应收未结项投影；
// 金额为账单原币，不同币种不合计。
type WorkbenchMyReceivable struct {
	BillID              uuid.UUID
	BillNo              string
	SettlementPartyName string
	Currency            string
	TotalAmount         decimal.Decimal
	UnverifiedAmount    decimal.Decimal
	BillDate            *string
	DueDate             *string
	OverdueDays         int32
	ConfirmedAt         time.Time
}

// WorkbenchCommissionFilter 是本人提成明细的服务端分页筛选。
type WorkbenchCommissionFilter struct {
	Page, PageSize     int
	Status             CommissionStatus
	CommissionDateFrom string
	CommissionDateTo   string
}

// WorkbenchOrderFilter 是本人近期订单的服务端分页参数。
type WorkbenchOrderFilter struct {
	Page, PageSize int
}

// WorkbenchReceivableFilter 是本人应收未结项的服务端分页参数。
type WorkbenchReceivableFilter struct {
	Page, PageSize int
}

// WorkbenchRepo 是工作台读模型的仓储接口；实现必须保证：
// 门禁为假时不查询也不返回提成金额模块；不同组织本位币不合计；
// 全部查询只读且按上限有界。
type WorkbenchRepo interface {
	GetOverview(context.Context, WorkbenchScope) (*WorkbenchOverview, error)
	ListMyCommissions(context.Context, WorkbenchScope, WorkbenchCommissionFilter) (*PagedList[*WorkbenchMyCommission], error)
	ListMyReceivables(context.Context, WorkbenchScope, WorkbenchReceivableFilter) (*PagedList[*WorkbenchMyReceivable], error)
	ListMyRecentOrders(context.Context, WorkbenchScope, WorkbenchOrderFilter) (*PagedList[*WorkbenchRecentOrder], error)
}

// WorkbenchUsecase 工作台读模型用例：只做主体校验、过滤校验与只读委托，
// 不承担写路径。
type WorkbenchUsecase struct {
	repo WorkbenchRepo
}

func NewWorkbenchUsecase(repo WorkbenchRepo) *WorkbenchUsecase {
	return &WorkbenchUsecase{repo: repo}
}

// GetOverview 返回当前主体在当前工作区组织的工作台聚合。
func (u *WorkbenchUsecase) GetOverview(ctx context.Context, scope WorkbenchScope) (*WorkbenchOverview, error) {
	if !validWorkbenchScope(scope) {
		return nil, ErrWorkbenchInvalid
	}
	return u.repo.GetOverview(ctx, scope)
}

// ListMyCommissions 返回本人提成单与调整明细分页：状态限定提成四态，日期为
// 归属日期闭区间过滤。
func (u *WorkbenchUsecase) ListMyCommissions(ctx context.Context, scope WorkbenchScope, f WorkbenchCommissionFilter) (*PagedList[*WorkbenchMyCommission], error) {
	f.CommissionDateFrom = strings.TrimSpace(f.CommissionDateFrom)
	f.CommissionDateTo = strings.TrimSpace(f.CommissionDateTo)
	if !validWorkbenchScope(scope) || !ValidListPagination(f.Page, f.PageSize) ||
		(f.Status != "" && f.Status != CommissionDraft && f.Status != CommissionConfirmed && f.Status != CommissionPaid && f.Status != CommissionCancelled) ||
		!validFinanceDateRange(f.CommissionDateFrom, f.CommissionDateTo) {
		return nil, ErrWorkbenchInvalid
	}
	return u.repo.ListMyCommissions(ctx, scope, f)
}

// ListMyReceivables 返回与本人提成归属相关的已确认应收未结项分页。
func (u *WorkbenchUsecase) ListMyReceivables(ctx context.Context, scope WorkbenchScope, f WorkbenchReceivableFilter) (*PagedList[*WorkbenchMyReceivable], error) {
	if !validWorkbenchScope(scope) || !ValidListPagination(f.Page, f.PageSize) {
		return nil, ErrWorkbenchInvalid
	}
	return u.repo.ListMyReceivables(ctx, scope, f)
}

// ListMyRecentOrders 返回本人真实协作的近期海运出口订单分页。
func (u *WorkbenchUsecase) ListMyRecentOrders(ctx context.Context, scope WorkbenchScope, f WorkbenchOrderFilter) (*PagedList[*WorkbenchRecentOrder], error) {
	if !validWorkbenchScope(scope) || !ValidListPagination(f.Page, f.PageSize) {
		return nil, ErrWorkbenchInvalid
	}
	return u.repo.ListMyRecentOrders(ctx, scope, f)
}
