package biz

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"

	financev1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
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

// BuildOrderListSummaries 为已授权订单页批量构建提成摘要：对每个订单组织单独
// 判定调用者是否持有该组织 system.finance.commission.read，持有则组织级全员
// 状态与汇总，否则 SQL 层固定 employee_id = 调用者。可见模式只在 SQL 条件中
// 生效，不先聚合全员再裁剪；对越权组织返回本人视图（一致的空态语义），不泄露
// 他人记录存在性。返回映射覆盖全部去重后的目标订单，本人/全员无记录时为空态。
func (u *CommissionUsecase) BuildOrderListSummaries(ctx context.Context, caller *Principal, targets []OrderCommissionSummaryTarget) (map[uuid.UUID]*OrderCommissionSummary, error) {
	if caller == nil || caller.UserID == uuid.Nil || len(targets) == 0 || len(targets) > MaxListPageSize {
		return nil, ErrCommissionInvalid
	}
	// 去重目标订单；同一订单只聚合一次，重复目标共享同一份摘要。
	deduped := make(map[uuid.UUID]uuid.UUID, len(targets))
	orderIDsByOrganization := make(map[uuid.UUID][]uuid.UUID)
	organizations := make([]uuid.UUID, 0, 2)
	for _, target := range targets {
		if target.OrderID == uuid.Nil || target.OrganizationID == uuid.Nil {
			return nil, ErrCommissionInvalid
		}
		if _, exists := deduped[target.OrderID]; exists {
			continue
		}
		deduped[target.OrderID] = target.OrganizationID
		existing, seen := orderIDsByOrganization[target.OrganizationID]
		if !seen {
			organizations = append(organizations, target.OrganizationID)
		}
		orderIDsByOrganization[target.OrganizationID] = append(existing, target.OrderID)
	}
	// 逐组织判定可见模式：同一主体在不同组织权限可以不同，切换组织后重新判定。
	scopes := make([]OrderCommissionSummaryScope, 0, len(organizations))
	visibilityByOrganization := make(map[uuid.UUID]OrderCommissionSummaryVisibility, len(organizations))
	for _, organizationID := range organizations {
		visibility := OrderCommissionVisibilityEmployee
		if caller.CanAccessOrganizationForPermission(access.FinanceCommissionRead, organizationID, false) {
			visibility = OrderCommissionVisibilityOrganization
		}
		visibilityByOrganization[organizationID] = visibility
		scope := OrderCommissionSummaryScope{OrganizationID: organizationID, Visibility: visibility, OrderIDs: orderIDsByOrganization[organizationID]}
		if visibility == OrderCommissionVisibilityEmployee {
			scope.EmployeeID = caller.UserID
		}
		scopes = append(scopes, scope)
	}
	summaries, err := u.repo.ListOrderSummaries(ctx, scopes)
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]*OrderCommissionSummary, len(deduped))
	for orderID, organizationID := range deduped {
		summary, exists := summaries[orderID]
		if !exists || summary == nil {
			// 本人/全员无记录：返回一致的空态，而不是缺失，避免调用方把缺失
			// 解释为不可见事实。
			summary = newEmptyOrderCommissionSummary(visibilityByOrganization[organizationID])
		}
		if summary.Visibility != visibilityByOrganization[organizationID] {
			return nil, ErrCommissionInvalid
		}
		result[orderID] = summary
	}
	return result, nil
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

func commissionAudit(org, actor, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &org, UserID: &actor, Action: action, Result: "success", ResourceType: "finance_commission", ResourceID: id.String()}
}
func commissionRuleAudit(org, actor, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &org, UserID: &actor, Action: action, Result: "success", ResourceType: "finance_commission_rule", ResourceID: id.String()}
}

func commissionAdjustmentAudit(org, actor, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &org, UserID: &actor, Action: action, Result: "success", ResourceType: "finance_commission_adjustment", ResourceID: id.String()}
}
