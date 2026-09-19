package data

import (
	"context"
	"sort"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	bill "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	billline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	applicationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionapplication"
	applicationline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionapplicationline"
	nettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	nettingalloc "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	verification "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	allocation "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	attribution "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	user "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
	"github.com/shopspring/decimal"
)

// financeCommissionApplicationRepo 月度提成申请仓储：候选解析复用既有计提引擎
// （loadCommissionCalculationSource / resolveCommissionRuleForDate /
// calculateCommissionFromSource 同口径），全量扫描按归属月分批有界推进，
// 不套用工作台「预计可计提」的 20 条估算上限；提交/重提在共享事务内完成
// 成员行锁、月度唯一键门禁、来源固定锁序重解析、DRAFT 提成受控复用/创建与
// 明细快照固化，任一来源变更整体回滚。
type financeCommissionApplicationRepo struct{ data *Data }

func NewFinanceCommissionApplicationRepo(data *Data) biz.FinanceCommissionApplicationRepo {
	return &financeCommissionApplicationRepo{data: data}
}

// applicationCandidate 是提交事务与候选列表共用的内部解析结果：金额、指纹为
// 按既有计提口径的精确计算值；existingID 非零表示复用本人未占用的 DRAFT 提成
// （沿用存储金额，不重算历史金额），为零表示提交时需受控创建 DRAFT。
type applicationCandidate struct {
	verificationID, nettingID uuid.UUID
	verificationNo, nettingNo string
	commissionDate            string
	role                      biz.CommissionPersonnelRole
	// 方案快照按提成单事实取值（历史提成允许无方案），随明细固化。
	ruleID           *uuid.UUID
	ruleVersion      uint64
	ruleName         *string
	calculationBasis *string
	baseCurrency     string
	amount           decimal.Decimal
	cnyAmount        decimal.Decimal
	fingerprint      string
	existingID       uuid.UUID
}

func candidateSourceID(item *applicationCandidate) uuid.UUID {
	if item.verificationID != uuid.Nil {
		return item.verificationID
	}
	return item.nettingID
}

// sortApplicationCandidates 按归属日期倒序、身份与来源 ID 升序稳定排序，保证
// 分页列表与摘要分组的确定性。
func sortApplicationCandidates(items []*applicationCandidate) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].commissionDate != items[j].commissionDate {
			return items[i].commissionDate > items[j].commissionDate
		}
		if items[i].role != items[j].role {
			return items[i].role < items[j].role
		}
		return candidateSourceID(items[i]).String() < candidateSourceID(items[j]).String()
	})
}

// applicationCandidateVerificationPredicates 是可申请核销来源的发现谓词：与
// 工作台预计口径同一来源条件（ACTIVE 应收核销 + 活跃分摊 + 本人归属订单），
// 附加归属月窗口；这里的归属过滤只是预筛，精确资格仍由逐来源引擎解析决定。
func applicationCandidateVerificationPredicates(scope biz.WorkbenchScope, dateFrom, dateTo string) []predicate.FinanceVerification {
	attributedOrder := workbenchAttributedOrderPredicate(scope)
	preds := []predicate.FinanceVerification{
		verification.OrganizationIDEQ(scope.OrganizationID),
		verification.StatusEQ(verification.StatusACTIVE),
		verification.DirectionEQ(verification.DirectionRECEIVABLE),
		verification.HasAllocationsWith(allocation.ActiveEQ(true), allocation.HasBillWith(
			bill.HasLinesWith(billline.ActiveEQ(true), billline.HasOrderWith(attributedOrder)),
		)),
	}
	if dateFrom != "" {
		preds = append(preds, verification.VerificationDateGTE(dateFrom))
	}
	if dateTo != "" {
		preds = append(preds, verification.VerificationDateLTE(dateTo))
	}
	return preds
}

// applicationCandidateNettingPredicates 是可申请对冲来源的发现谓词；窗口按
// confirmed_at 的 UTC 日期过滤，与提成归属日期口径一致。
func applicationCandidateNettingPredicates(scope biz.WorkbenchScope, from, to *time.Time) []predicate.FinanceNetting {
	attributedOrder := workbenchAttributedOrderPredicate(scope)
	preds := []predicate.FinanceNetting{
		nettingent.OrganizationIDEQ(scope.OrganizationID),
		nettingent.StatusEQ(nettingent.StatusCONFIRMED),
		nettingent.ConfirmedAtNotNil(),
		nettingent.HasAllocationsWith(
			nettingalloc.ActiveEQ(true),
			nettingalloc.DirectionEQ(nettingalloc.DirectionRECEIVABLE),
			nettingalloc.HasBillWith(bill.HasLinesWith(billline.ActiveEQ(true), billline.HasOrderWith(attributedOrder))),
		),
	}
	if from != nil {
		preds = append(preds, nettingent.ConfirmedAtGTE(*from))
	}
	if to != nil {
		preds = append(preds, nettingent.ConfirmedAtLT(*to))
	}
	return preds
}

// monthWindowList 把 [fromDate, toDate] 拆成逐自然月的闭区间窗口，全量扫描按
// 归属月分批推进，单批内存有界。
func monthWindowList(fromDate, toDate string) [][2]string {
	from, err := time.ParseInLocation("2006-01-02", fromDate, time.UTC)
	if err != nil {
		return nil
	}
	to, err := time.ParseInLocation("2006-01-02", toDate, time.UTC)
	if err != nil || from.After(to) {
		return nil
	}
	windows := make([][2]string, 0, 4)
	cursor := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	for !cursor.After(to) {
		windowEnd := cursor.AddDate(0, 1, -1)
		if windowEnd.After(to) {
			windowEnd = to
		}
		windows = append(windows, [2]string{cursor.Format("2006-01-02"), windowEnd.Format("2006-01-02")})
		cursor = cursor.AddDate(0, 1, 0)
	}
	return windows
}

// applicationSourceRef 是窗口内待处理来源的轻量引用，处理前按来源 ID 升序
// 排序以满足固定锁序。
type applicationSourceRef struct {
	verificationID, nettingID uuid.UUID
}

// earliestApplicationSourceDate 返回本人归属范围内最早来源的归属日期；没有任何
// 候选来源时返回 false。
func earliestApplicationSourceDate(ctx context.Context, client *ent.Client, scope biz.WorkbenchScope) (string, bool, error) {
	oldest, err := client.FinanceVerification.Query().
		Where(applicationCandidateVerificationPredicates(scope, "", "")...).
		Order(verification.ByVerificationDate(), verification.ByID()).
		Select(verification.FieldVerificationDate).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return "", false, err
	}
	earliest := ""
	if err == nil {
		earliest = oldest.VerificationDate
	}
	oldestNetting, err := client.FinanceNetting.Query().
		Where(applicationCandidateNettingPredicates(scope, nil, nil)...).
		Order(nettingent.ByConfirmedAt(), nettingent.ByID()).
		Select(nettingent.FieldConfirmedAt).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return "", false, err
	}
	if err == nil && oldestNetting.ConfirmedAt != nil {
		if nettingDate := oldestNetting.ConfirmedAt.UTC().Format("2006-01-02"); earliest == "" || nettingDate < earliest {
			earliest = nettingDate
		}
	}
	if earliest == "" {
		return "", false, nil
	}
	return earliest, true, nil
}

// resolveApplicationCandidates 全量解析「截止日以前、当前员工、当前组织、未取消、
// 未被任何申请明细占用」的合格提成候选。lock=false 为只读路径（摘要与候选列表）；
// lock=true 为提交事务路径：成员行锁之后按「来源 → 账单 → 订单 → 费用 → 方案 →
// 归属 → 提成父单」固定锁序重解析，来源变更时由调用方整体回滚。
func (r *financeCommissionApplicationRepo) resolveApplicationCandidates(ctx context.Context, client *ent.Client, scope biz.WorkbenchScope, coverageTo string, lock bool) ([]*applicationCandidate, error) {
	store := commissionStoreFromClient(client)
	// 员工已离职不影响历史资格：解析只依赖来源日期、方案区间与分配区间。
	employeeQuery := store.users.Query().Where(user.IDEQ(scope.UserID))
	if lock {
		employeeQuery.ForUpdate()
	}
	employee, err := employeeQuery.Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return []*applicationCandidate{}, nil
		}
		return nil, err
	}
	earliest, hasAny, err := earliestApplicationSourceDate(ctx, client, scope)
	if err != nil {
		return nil, err
	}
	if !hasAny {
		return []*applicationCandidate{}, nil
	}
	candidates := make([]*applicationCandidate, 0)
	for _, window := range monthWindowList(earliest, coverageTo) {
		batch, windowErr := r.resolveCandidateWindow(ctx, client, store, scope, employee, window[0], window[1], lock)
		if windowErr != nil {
			return nil, windowErr
		}
		candidates = append(candidates, batch...)
	}
	sortApplicationCandidates(candidates)
	return candidates, nil
}

// resolveCandidateWindow 解析单个归属月窗口内的全部来源：核销与对冲来源合并后
// 按来源 ID 升序处理，固定加锁顺序防止并发死锁。
func (r *financeCommissionApplicationRepo) resolveCandidateWindow(ctx context.Context, client *ent.Client, store commissionCalculationStore, scope biz.WorkbenchScope, employee *ent.User, dateFrom, dateTo string, lock bool) ([]*applicationCandidate, error) {
	verificationItems, err := client.FinanceVerification.Query().
		Where(applicationCandidateVerificationPredicates(scope, dateFrom, dateTo)...).
		Select(verification.FieldID).
		All(ctx)
	if err != nil {
		return nil, err
	}
	nettingFrom, nettingTo := nettingUTCTimeWindow(dateFrom, dateTo)
	nettingItems, err := client.FinanceNetting.Query().
		Where(applicationCandidateNettingPredicates(scope, nettingFrom, nettingTo)...).
		Select(nettingent.FieldID).
		All(ctx)
	if err != nil {
		return nil, err
	}
	sources := make([]applicationSourceRef, 0, len(verificationItems)+len(nettingItems))
	for _, item := range verificationItems {
		sources = append(sources, applicationSourceRef{verificationID: item.ID})
	}
	for _, item := range nettingItems {
		sources = append(sources, applicationSourceRef{nettingID: item.ID})
	}
	sort.Slice(sources, func(i, j int) bool { return sourceRefID(sources[i]).String() < sourceRefID(sources[j]).String() })
	candidates := make([]*applicationCandidate, 0)
	for _, ref := range sources {
		batch, sourceErr := r.resolveCandidateSource(ctx, client, store, scope, employee, ref.verificationID, ref.nettingID, lock)
		if sourceErr != nil {
			return nil, sourceErr
		}
		candidates = append(candidates, batch...)
	}
	return candidates, nil
}

func sourceRefID(ref applicationSourceRef) uuid.UUID {
	if ref.verificationID != uuid.Nil {
		return ref.verificationID
	}
	return ref.nettingID
}

// nettingUTCTimeWindow 把归属月闭区间换算为 confirmed_at 的 UTC 半开区间。
func nettingUTCTimeWindow(dateFrom, dateTo string) (*time.Time, *time.Time) {
	from, err := time.ParseInLocation("2006-01-02", dateFrom, time.UTC)
	if err != nil {
		return nil, nil
	}
	to, err := time.ParseInLocation("2006-01-02", dateTo, time.UTC)
	if err != nil {
		return nil, nil
	}
	to = to.AddDate(0, 0, 1)
	return &from, &to
}

// resolveCandidateSource 逐来源解析本人全部身份的候选：引擎加载失败或无本人
// 归属的来源静默跳过（与工作台预计口径一致），其余错误原样外传。
func (r *financeCommissionApplicationRepo) resolveCandidateSource(ctx context.Context, client *ent.Client, store commissionCalculationStore, scope biz.WorkbenchScope, employee *ent.User, verificationID, nettingID uuid.UUID, lock bool) ([]*applicationCandidate, error) {
	source, err := loadCommissionCalculationSource(ctx, store, scope.OrganizationID, verificationID, nettingID, lock)
	if err != nil {
		if isSkippableCommissionError(err) {
			return nil, nil
		}
		return nil, err
	}
	aq := store.attributions.Query().Where(
		attribution.OrganizationIDEQ(scope.OrganizationID),
		attribution.OrderIDIn(source.orderIDs...),
		attribution.EmployeeIDEQ(scope.UserID),
	).Order(attribution.ByAttributedAt(), attribution.ByID())
	if lock {
		aq.ForUpdate()
	}
	attributions, err := aq.All(ctx)
	if err != nil {
		return nil, err
	}
	roleAttributions := make(map[biz.CommissionPersonnelRole][]*ent.OrderCommissionAttribution, 2)
	for _, item := range attributions {
		role := biz.CommissionPersonnelRole(item.PersonnelRole)
		roleAttributions[role] = append(roleAttributions[role], item)
	}
	roles := make([]biz.CommissionPersonnelRole, 0, len(roleAttributions))
	for role := range roleAttributions {
		roles = append(roles, role)
	}
	sort.Slice(roles, func(i, j int) bool { return roles[i] < roles[j] })
	candidates := make([]*applicationCandidate, 0, len(roles))
	for _, role := range roles {
		candidate, roleErr := r.resolveCandidateRole(ctx, client, store, scope, employee, source, role, roleAttributions[role], lock)
		if roleErr != nil {
			return nil, roleErr
		}
		if candidate != nil {
			candidates = append(candidates, candidate)
		}
	}
	return candidates, nil
}

// resolveCandidateRole 解析单个「来源 × 身份」候选：已有本人提成事实时按状态
// 分流——CONFIRMED/PAID 与仅有已取消记录的来源不进入待申请集合；DRAFT 校验占用
// 与指纹（提交路径命中返回稳定冲突，读取路径静默排除）；无提成事实的来源由
// 提交路径受控创建 DRAFT。
func (r *financeCommissionApplicationRepo) resolveCandidateRole(ctx context.Context, client *ent.Client, store commissionCalculationStore, scope biz.WorkbenchScope, employee *ent.User, source *commissionCalculationSource, role biz.CommissionPersonnelRole, attributions []*ent.OrderCommissionAttribution, lock bool) (*applicationCandidate, error) {
	cq := client.FinanceCommission.Query().Where(
		commission.OrganizationIDEQ(scope.OrganizationID),
		commission.EmployeeIDEQ(scope.UserID),
		commission.PersonnelRoleEQ(string(role)),
	)
	if source.verification != nil {
		cq = cq.Where(commission.VerificationIDEQ(source.verification.ID))
	} else {
		cq = cq.Where(commission.NettingIDEQ(source.netting.ID))
	}
	if lock {
		cq.ForUpdate()
	}
	history, err := cq.Order(commission.ByCreatedAt(), commission.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	// 部分唯一索引保证非取消提成至多一条；取消与确认/发放同为既定财务事实。
	var active *ent.FinanceCommission
	hasCancelled := false
	for _, item := range history {
		if item.Status == commission.StatusCANCELLED {
			hasCancelled = true
			continue
		}
		active = item
		break
	}
	if active != nil && active.Status != commission.StatusDRAFT {
		// 已确认/已发放的提成不能进入待申请集合。
		return nil, nil
	}
	if active == nil && hasCancelled {
		// 仅剩取消记录的来源：不得通过申请重新生成提成，避免绕过财务取消。
		return nil, nil
	}
	calculation, err := resolveFreshCommissionCalculation(ctx, store, scope, employee, source, role, attributions, lock)
	if err != nil {
		if isSkippableCommissionError(err) {
			return nil, nil
		}
		return nil, err
	}
	verificationID, verificationNo, nettingID, nettingNo := applicationSourceIdentity(source)
	if active != nil {
		occupied, existErr := client.FinanceCommissionApplicationLine.Query().
			Where(applicationline.CommissionIDEQ(active.ID)).Exist(ctx)
		if existErr != nil {
			return nil, existErr
		}
		if occupied {
			if lock {
				// design §3.3：已归属于其他申请的 DRAFT 提交返回明确冲突，
				// 不静默复制或改变历史金额。
				return nil, biz.ErrCommissionApplicationSourceConflict
			}
			return nil, nil
		}
		if active.SourceFingerprint == "" || active.SourceFingerprint != calculation.SourceFingerprint {
			if lock {
				return nil, biz.ErrCommissionApplicationSourceConflict
			}
			return nil, nil
		}
		amount, amountErr := decimalOf(active.CommissionAmount)
		if amountErr != nil {
			return nil, amountErr
		}
		cnyAmount, cnyErr := decimalOf(active.CnyCommissionAmount)
		if cnyErr != nil {
			return nil, cnyErr
		}
		// 复用本人未占用 DRAFT：沿用存储金额与指纹，不重算历史金额。
		return &applicationCandidate{
			verificationID: verificationID, verificationNo: verificationNo,
			nettingID: nettingID, nettingNo: nettingNo,
			commissionDate:   active.CommissionDate,
			role:             role,
			ruleID:           active.RuleID,
			ruleVersion:      active.RuleVersion,
			ruleName:         active.RuleName,
			calculationBasis: active.CalculationBasis,
			baseCurrency:     active.BaseCurrency,
			amount:           amount, cnyAmount: cnyAmount,
			fingerprint: active.SourceFingerprint,
			existingID:  active.ID,
		}, nil
	}
	// 尚无提成事实的合格来源：候选金额为精确计算值，提交时受控创建 DRAFT。
	cnySnapshot, snapshotErr := biz.ResolveCommissionCNYRate(calculation.BaseCurrency, source.commissionDate, calculation.CommissionAmount)
	if snapshotErr != nil {
		return nil, snapshotErr
	}
	return &applicationCandidate{
		verificationID: verificationID, verificationNo: verificationNo,
		nettingID: nettingID, nettingNo: nettingNo,
		commissionDate:   source.commissionDate,
		role:             role,
		ruleID:           uuidPointerOrNil(calculation.RuleID),
		ruleVersion:      calculation.RuleVersion,
		ruleName:         stringPointerOrNil(calculation.RuleName),
		calculationBasis: stringPointerOrNil(string(calculation.CalculationBasis)),
		baseCurrency:     calculation.BaseCurrency,
		amount:           calculation.CommissionAmount,
		cnyAmount:        cnySnapshot.CommissionAmount,
		fingerprint:      calculation.SourceFingerprint,
	}, nil
}

// resolveFreshCommissionCalculation 按既有计提引擎完整重算「来源 × 身份」：
// 方案员工分配唯一命中与订单同身份归属校验同口径，lock=true 时方案与归属行
// 参与事务锁序。
func resolveFreshCommissionCalculation(ctx context.Context, store commissionCalculationStore, scope biz.WorkbenchScope, employee *ent.User, source *commissionCalculationSource, role biz.CommissionPersonnelRole, attributions []*ent.OrderCommissionAttribution, lock bool) (*biz.CommissionCalculation, error) {
	ruleItem, assignmentItem, err := resolveCommissionRuleForDate(ctx, store, scope.OrganizationID, scope.UserID, role, source.commissionDate, lock)
	if err != nil {
		return nil, err
	}
	return calculateCommissionFromSource(source, ruleItem, assignmentItem, employee, attributions)
}

// isSkippableCommissionError 是候选解析中「无可靠计提条件」的静默跳过集合，
// 与工作台预计可计提的口径一致；其余错误原样外传。
func isSkippableCommissionError(err error) bool {
	skippable := map[error]bool{
		biz.ErrCommissionSource:          true,
		biz.ErrCommissionRuleNotResolved: true,
		biz.ErrCommissionEmployeeRole:    true,
	}
	return skippable[err]
}

// applicationSourceIdentity 返回来源二选一的 ID 与单号投影。
func applicationSourceIdentity(source *commissionCalculationSource) (verificationID uuid.UUID, verificationNo string, nettingID uuid.UUID, nettingNo string) {
	if source.verification != nil {
		return source.verification.ID, source.verification.VerificationNo, uuid.Nil, ""
	}
	if source.netting != nil {
		return uuid.Nil, "", source.netting.ID, source.netting.NettingNo
	}
	return uuid.Nil, "", uuid.Nil, ""
}

// ApplicationSummary 返回本人申请摘要段：可申请按归属月分组（精确解析）、
// 本月累计中沿用预计口径的有界扫描、本人历史申请计数与最近申请概要。
func (r *financeCommissionApplicationRepo) ApplicationSummary(ctx context.Context, scope biz.WorkbenchScope, baseCurrency string) (*biz.WorkbenchApplicationSummary, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	_, coverageTo, ok := biz.CommissionApplicationPeriod(scope.Today)
	if !ok {
		return nil, biz.ErrCommissionApplicationInvalid
	}
	candidates, err := r.resolveApplicationCandidates(ctx, client, scope, coverageTo, false)
	if err != nil {
		return nil, err
	}
	summary := &biz.WorkbenchApplicationSummary{BaseCurrency: baseCurrency}
	groupByMonth := make(map[string]*biz.WorkbenchApplyMonthGroup)
	for _, candidate := range candidates {
		month := candidate.commissionDate[:7]
		group, exists := groupByMonth[month]
		if !exists {
			group = &biz.WorkbenchApplyMonthGroup{CommissionMonth: month}
			groupByMonth[month] = group
		}
		group.CommissionCount++
		group.CommissionAmount = group.CommissionAmount.Add(candidate.amount).Round(8)
	}
	months := make([]string, 0, len(groupByMonth))
	for month := range groupByMonth {
		months = append(months, month)
	}
	sort.Strings(months)
	for _, month := range months {
		summary.ApplyGroups = append(summary.ApplyGroups, groupByMonth[month])
	}
	accumulatingCount, accumulatingAmount, err := r.accumulatingCurrentMonthOpportunity(ctx, client, scope)
	if err != nil {
		return nil, err
	}
	summary.AccumulatingCount = accumulatingCount
	summary.AccumulatingAmount = accumulatingAmount
	personalPredicates := []predicate.FinanceCommissionApplication{
		applicationent.OrganizationIDEQ(scope.OrganizationID),
		applicationent.EmployeeIDEQ(scope.UserID),
	}
	if summary.PendingReviewCount, err = client.FinanceCommissionApplication.Query().
		Where(append(append([]predicate.FinanceCommissionApplication{}, personalPredicates...),
			applicationent.StatusEQ(applicationent.StatusPENDING_REVIEW))...).Count(ctx); err != nil {
		return nil, err
	}
	if summary.ApprovedCount, err = client.FinanceCommissionApplication.Query().
		Where(append(append([]predicate.FinanceCommissionApplication{}, personalPredicates...),
			applicationent.StatusEQ(applicationent.StatusAPPROVED))...).Count(ctx); err != nil {
		return nil, err
	}
	latest, err := client.FinanceCommissionApplication.Query().Where(personalPredicates...).
		Order(applicationent.BySubmittedAt(entsql.OrderDesc()), applicationent.ByID(entsql.OrderDesc())).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}
	if err == nil {
		totalAmount, amountErr := decimalOf(latest.TotalCommissionAmount)
		if amountErr != nil {
			return nil, amountErr
		}
		summary.LatestApplication = &biz.WorkbenchApplicationBrief{
			ApplicationID:         latest.ID,
			ApplicationMonth:      latest.ApplicationMonth,
			Status:                biz.CommissionApplicationStatus(latest.Status),
			CommissionCount:       latest.CommissionCount,
			TotalCommissionAmount: totalAmount,
			SubmittedAt:           latest.SubmittedAt,
		}
	}
	return summary, nil
}

// accumulatingCurrentMonthOpportunity 汇总当前自然月的「本月累计中」概要：
// 沿用工作台预计可计提口径（同一来源条件与解析引擎、有界扫描、跳过已有有效
// 提成的来源），只统计归属日期落在当前自然月内的来源。
func (r *financeCommissionApplicationRepo) accumulatingCurrentMonthOpportunity(ctx context.Context, client *ent.Client, scope biz.WorkbenchScope) (int, decimal.Decimal, error) {
	store := commissionStoreFromClient(client)
	employee, err := store.users.Query().Where(user.IDEQ(scope.UserID)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, decimal.Zero, nil
		}
		return 0, decimal.Zero, err
	}
	monthStart := scope.Today[:8] + "01"
	nettingFrom, nettingTo := nettingUTCTimeWindow(monthStart, scope.Today)
	verificationItems, err := client.FinanceVerification.Query().
		Where(applicationCandidateVerificationPredicates(scope, monthStart, scope.Today)...).
		Order(verification.ByCreatedAt(entsql.OrderDesc()), verification.ByID(entsql.OrderDesc())).
		Limit(biz.WorkbenchEstimatedScanLimit).All(ctx)
	if err != nil {
		return 0, decimal.Zero, err
	}
	nettingItems, err := client.FinanceNetting.Query().
		Where(applicationCandidateNettingPredicates(scope, nettingFrom, nettingTo)...).
		Order(nettingent.ByCreatedAt(entsql.OrderDesc()), nettingent.ByID(entsql.OrderDesc())).
		Limit(biz.WorkbenchEstimatedScanLimit).All(ctx)
	if err != nil {
		return 0, decimal.Zero, err
	}
	count := 0
	amount := decimal.Zero
	process := func(verificationID, nettingID uuid.UUID) error {
		source, sourceErr := loadCommissionCalculationSource(ctx, store, scope.OrganizationID, verificationID, nettingID, false)
		if sourceErr != nil {
			if isSkippableCommissionError(sourceErr) {
				return nil
			}
			return sourceErr
		}
		attributions, attributionErr := store.attributions.Query().Where(
			attribution.OrganizationIDEQ(scope.OrganizationID),
			attribution.OrderIDIn(source.orderIDs...),
			attribution.EmployeeIDEQ(scope.UserID),
		).Order(attribution.ByID()).All(ctx)
		if attributionErr != nil {
			return attributionErr
		}
		roleSeen := make(map[biz.CommissionPersonnelRole]struct{}, 2)
		for _, item := range attributions {
			role := biz.CommissionPersonnelRole(item.PersonnelRole)
			if _, exists := roleSeen[role]; exists {
				continue
			}
			roleSeen[role] = struct{}{}
			roleAttributions := make([]*ent.OrderCommissionAttribution, 0, len(attributions))
			for _, attributionItem := range attributions {
				if biz.CommissionPersonnelRole(attributionItem.PersonnelRole) == role {
					roleAttributions = append(roleAttributions, attributionItem)
				}
			}
			hasCommission, existErr := workbenchHasActiveCommission(ctx, client, scope, verificationID, nettingID, role)
			if existErr != nil {
				return existErr
			}
			if hasCommission {
				continue
			}
			calculation, calculateErr := resolveFreshCommissionCalculation(ctx, store, scope, employee, source, role, roleAttributions, false)
			if calculateErr != nil {
				if isSkippableCommissionError(calculateErr) {
					continue
				}
				return calculateErr
			}
			count++
			amount = amount.Add(calculation.CommissionAmount).Round(8)
		}
		return nil
	}
	for _, item := range verificationItems {
		if processErr := process(item.ID, uuid.Nil); processErr != nil {
			return 0, decimal.Zero, processErr
		}
	}
	for _, item := range nettingItems {
		if item.ConfirmedAt == nil {
			continue
		}
		if processErr := process(uuid.Nil, item.ID); processErr != nil {
			return 0, decimal.Zero, processErr
		}
	}
	return count, amount, nil
}

// ListMyCandidates 返回本人可申请候选分页：全量精确解析后按归属月过滤并内存
// 分页（解析本身必须覆盖完整集合，分页只裁剪返回视图）。
func (r *financeCommissionApplicationRepo) ListMyCandidates(ctx context.Context, scope biz.WorkbenchScope, filter biz.WorkbenchApplicationCandidateFilter) (*biz.PagedList[*biz.WorkbenchApplicationCandidate], error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	_, coverageTo, ok := biz.CommissionApplicationPeriod(scope.Today)
	if !ok {
		return nil, biz.ErrCommissionApplicationInvalid
	}
	candidates, err := r.resolveApplicationCandidates(ctx, client, scope, coverageTo, false)
	if err != nil {
		return nil, err
	}
	filtered := make([]*applicationCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if filter.CommissionMonth != "" && candidate.commissionDate[:7] != filter.CommissionMonth {
			continue
		}
		filtered = append(filtered, candidate)
	}
	start := (filter.Page - 1) * filter.PageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + filter.PageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	items := make([]*biz.WorkbenchApplicationCandidate, 0, end-start)
	for _, candidate := range filtered[start:end] {
		candidateRuleID := uuid.Nil
		if candidate.ruleID != nil {
			candidateRuleID = *candidate.ruleID
		}
		candidateRuleName := ""
		if candidate.ruleName != nil {
			candidateRuleName = *candidate.ruleName
		}
		candidateBasis := ""
		if candidate.calculationBasis != nil {
			candidateBasis = *candidate.calculationBasis
		}
		items = append(items, &biz.WorkbenchApplicationCandidate{
			VerificationID:      candidate.verificationID,
			VerificationNo:      candidate.verificationNo,
			NettingID:           candidate.nettingID,
			NettingNo:           candidate.nettingNo,
			CommissionDate:      candidate.commissionDate,
			PersonnelRole:       candidate.role,
			RuleID:              candidateRuleID,
			RuleName:            candidateRuleName,
			CalculationBasis:    biz.CommissionCalculationBasis(candidateBasis),
			BaseCurrency:        candidate.baseCurrency,
			CommissionAmount:    candidate.amount,
			CNYCommissionAmount: candidate.cnyAmount,
		})
	}
	return &biz.PagedList[*biz.WorkbenchApplicationCandidate]{Items: items, Total: len(filtered), Page: filter.Page, PageSize: filter.PageSize}, nil
}

// ListMyApplications 返回本人申请历史分页：组织 + 员工固定在查询条件中，
// 按提交时间倒序稳定排序。
func (r *financeCommissionApplicationRepo) ListMyApplications(ctx context.Context, scope biz.WorkbenchScope, filter biz.WorkbenchApplicationFilter) (*biz.PagedList[*biz.FinanceCommissionApplication], error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	predicates := []predicate.FinanceCommissionApplication{
		applicationent.OrganizationIDEQ(scope.OrganizationID),
		applicationent.EmployeeIDEQ(scope.UserID),
	}
	if filter.Status != "" {
		predicates = append(predicates, applicationent.StatusEQ(applicationent.Status(filter.Status)))
	}
	query := client.FinanceCommissionApplication.Query().Where(predicates...)
	return paginate(ctx, func(ctx context.Context) (int, error) {
		return query.Clone().Count(ctx)
	}, func(ctx context.Context, offset, limit int) ([]*ent.FinanceCommissionApplication, error) {
		return query.Order(
			applicationent.BySubmittedAt(entsql.OrderDesc()),
			applicationent.ByID(entsql.OrderDesc()),
		).Offset(offset).Limit(limit).All(ctx)
	}, filter.Page, filter.PageSize, infalliblePageConverter(financeCommissionApplicationToBiz))
}

// GetMyApplication 返回本人单张申请详情：查询同时限定组织与本人，他人或跨组织
// 申请按不存在处理，不泄露记录事实。
func (r *financeCommissionApplicationRepo) GetMyApplication(ctx context.Context, scope biz.WorkbenchScope, id uuid.UUID) (*biz.FinanceCommissionApplicationDetail, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	header, err := client.FinanceCommissionApplication.Query().Where(
		applicationent.IDEQ(id),
		applicationent.OrganizationIDEQ(scope.OrganizationID),
		applicationent.EmployeeIDEQ(scope.UserID),
	).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrCommissionApplicationNotFound, nil)
	}
	lines, err := client.FinanceCommissionApplicationLine.Query().
		Where(applicationline.ApplicationIDEQ(id)).
		Order(applicationline.ByCommissionDate(), applicationline.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	detail := &biz.FinanceCommissionApplicationDetail{Application: financeCommissionApplicationToBiz(header), Lines: make([]*biz.FinanceCommissionApplicationLine, 0, len(lines))}
	for _, line := range lines {
		item, convertErr := financeCommissionApplicationLineToBiz(line)
		if convertErr != nil {
			return nil, convertErr
		}
		detail.Lines = append(detail.Lines, item)
	}
	return detail, nil
}

// fillApplicationDisplayNames 批量填充申请头的员工与组织展示名：页内去重后
// 一次查询，不逐行回表。
func fillApplicationDisplayNames(ctx context.Context, client *ent.Client, items []*biz.FinanceCommissionApplication) error {
	if len(items) == 0 {
		return nil
	}
	employeeIDs := make([]uuid.UUID, 0, len(items))
	organizationIDs := make([]uuid.UUID, 0, len(items))
	seenEmployee := make(map[uuid.UUID]struct{}, len(items))
	seenOrganization := make(map[uuid.UUID]struct{}, len(items))
	for _, item := range items {
		if _, exists := seenEmployee[item.EmployeeID]; !exists {
			seenEmployee[item.EmployeeID] = struct{}{}
			employeeIDs = append(employeeIDs, item.EmployeeID)
		}
		if _, exists := seenOrganization[item.OrganizationID]; !exists {
			seenOrganization[item.OrganizationID] = struct{}{}
			organizationIDs = append(organizationIDs, item.OrganizationID)
		}
	}
	employees, err := client.User.Query().Where(user.IDIn(employeeIDs...)).Select(user.FieldID, user.FieldDisplayName).All(ctx)
	if err != nil {
		return err
	}
	nameByEmployee := make(map[uuid.UUID]string, len(employees))
	for _, item := range employees {
		nameByEmployee[item.ID] = item.DisplayName
	}
	organizations, err := client.Organization.Query().Where(organizationent.IDIn(organizationIDs...)).Select(organizationent.FieldID, organizationent.FieldName).All(ctx)
	if err != nil {
		return err
	}
	nameByOrganization := make(map[uuid.UUID]string, len(organizations))
	for _, item := range organizations {
		nameByOrganization[item.ID] = item.Name
	}
	for _, item := range items {
		item.EmployeeName = nameByEmployee[item.EmployeeID]
		item.OrganizationName = nameByOrganization[item.OrganizationID]
	}
	return nil
}

// applicationCommissionNumbers 按提成 ID 批量解析提成单号，供财务明细下钻。
func applicationCommissionNumbers(ctx context.Context, client *ent.Client, commissionIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	if len(commissionIDs) == 0 {
		return map[uuid.UUID]string{}, nil
	}
	rows, err := client.FinanceCommission.Query().
		Where(commission.IDIn(commissionIDs...)).
		Select(commission.FieldID, commission.FieldCommissionNo).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]string, len(rows))
	for _, row := range rows {
		result[row.ID] = row.CommissionNo
	}
	return result, nil
}

// ListForOrganization 财务申请列表：组织范围显式来自 read 权限解析（跨组织
// 查询必须显式带组织过滤），按员工/状态/提交月过滤并服务端分页，按提交时间
// 倒序稳定排序；员工与组织展示名按页批量解析。
func (r *financeCommissionApplicationRepo) ListForOrganization(ctx context.Context, organizationIDs []uuid.UUID, filter biz.CommissionApplicationFinanceFilter) (*biz.PagedList[*biz.FinanceCommissionApplication], error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	predicates := []predicate.FinanceCommissionApplication{applicationent.OrganizationIDIn(organizationIDs...)}
	if filter.EmployeeID != uuid.Nil {
		predicates = append(predicates, applicationent.EmployeeIDEQ(filter.EmployeeID))
	}
	if filter.Status != "" {
		predicates = append(predicates, applicationent.StatusEQ(applicationent.Status(filter.Status)))
	}
	if filter.ApplicationMonth != "" {
		predicates = append(predicates, applicationent.ApplicationMonthEQ(filter.ApplicationMonth))
	}
	query := client.FinanceCommissionApplication.Query().Where(predicates...)
	result, err := paginate(ctx, func(ctx context.Context) (int, error) {
		return query.Clone().Count(ctx)
	}, func(ctx context.Context, offset, limit int) ([]*ent.FinanceCommissionApplication, error) {
		return query.Order(
			applicationent.BySubmittedAt(entsql.OrderDesc()),
			applicationent.ByID(entsql.OrderDesc()),
		).Offset(offset).Limit(limit).All(ctx)
	}, filter.Page, filter.PageSize, infalliblePageConverter(financeCommissionApplicationToBiz))
	if err != nil {
		return nil, err
	}
	if err := fillApplicationDisplayNames(ctx, client, result.Items); err != nil {
		return nil, err
	}
	return result, nil
}

// GetForOrganization 财务申请详情：申请头审计字段、明细快照与提成单号投影；
// 查询同时限定组织范围，跨组织申请按不存在处理，不泄露记录事实。
func (r *financeCommissionApplicationRepo) GetForOrganization(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*biz.FinanceCommissionApplicationDetail, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	header, err := client.FinanceCommissionApplication.Query().Where(
		applicationent.IDEQ(id),
		applicationent.OrganizationIDIn(organizationIDs...),
	).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrCommissionApplicationNotFound, nil)
	}
	lines, err := client.FinanceCommissionApplicationLine.Query().
		Where(applicationline.ApplicationIDEQ(id)).
		Order(applicationline.ByCommissionDate(), applicationline.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	commissionIDs := make([]uuid.UUID, 0, len(lines))
	for _, line := range lines {
		commissionIDs = append(commissionIDs, line.CommissionID)
	}
	commissionNoByID, err := applicationCommissionNumbers(ctx, client, commissionIDs)
	if err != nil {
		return nil, err
	}
	detail := &biz.FinanceCommissionApplicationDetail{Application: financeCommissionApplicationToBiz(header), Lines: make([]*biz.FinanceCommissionApplicationLine, 0, len(lines))}
	for _, line := range lines {
		item, convertErr := financeCommissionApplicationLineToBiz(line)
		if convertErr != nil {
			return nil, convertErr
		}
		item.CommissionNo = commissionNoByID[line.CommissionID]
		detail.Lines = append(detail.Lines, item)
	}
	if err := fillApplicationDisplayNames(ctx, client, []*biz.FinanceCommissionApplication{detail.Application}); err != nil {
		return nil, err
	}
	return detail, nil
}

// lockApplicationHeaderForDecision 是财务批准/驳回共用的申请头锁定与门禁：
// FOR UPDATE 串行化并发决策，expected_version 与 PENDING_REVIEW 双重校验，
// 不匹配返回「该申请已被处理」稳定冲突。
func lockApplicationHeaderForDecision(ctx context.Context, client *ent.Client, organizationIDs []uuid.UUID, id uuid.UUID, expectedVersion uint64) (*ent.FinanceCommissionApplication, error) {
	header, err := client.FinanceCommissionApplication.Query().Where(
		applicationent.IDEQ(id),
		applicationent.OrganizationIDIn(organizationIDs...),
	).ForUpdate().Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrCommissionApplicationNotFound, nil)
	}
	if header.Version != expectedVersion || header.Status != applicationent.StatusPENDING_REVIEW {
		return nil, biz.ErrCommissionApplicationStatusConflict
	}
	return header, nil
}

// Approve 整单批准月度申请（design §5.2）。单个共享事务内顺序：
//  1. 锁定申请头并校验 expected_version 与 PENDING_REVIEW——并发批准/驳回/重提
//     只有一个事务成功，其余稳定冲突；
//  2. 按 commission_id 升序锁定申请明细，交由共享事务内的整批确认辅助
//     （confirmCommissionsForApplicationApproval）按「来源订单 → 提成父单」
//     固定锁序逐笔重算指纹、阻断草稿费用并把 DRAFT 提成整批转为 CONFIRMED；
//  3. 更新申请头为 APPROVED、记录决策人/时间、递增版本并写决策审计。
//     任一步失败整体回滚，不产生部分批准。
func (r *financeCommissionApplicationRepo) Approve(ctx context.Context, organizationIDs []uuid.UUID, decisionMaker, id uuid.UUID, expectedVersion uint64) (*biz.FinanceCommissionApplication, error) {
	var result *biz.FinanceCommissionApplication
	err := r.data.WithinTransaction(ctx, func(txCtx context.Context) error {
		client, clientErr := r.data.client(txCtx)
		if clientErr != nil {
			return clientErr
		}
		header, lockErr := lockApplicationHeaderForDecision(txCtx, client, organizationIDs, id, expectedVersion)
		if lockErr != nil {
			return lockErr
		}
		lines, lineErr := client.FinanceCommissionApplicationLine.Query().
			Where(applicationline.ApplicationIDEQ(header.ID)).
			Order(applicationline.ByCommissionID()).
			ForUpdate().All(txCtx)
		if lineErr != nil {
			return lineErr
		}
		if len(lines) == 0 {
			return biz.ErrCommissionApplicationSourceConflict
		}
		commissionIDs := make([]uuid.UUID, 0, len(lines))
		for _, line := range lines {
			commissionIDs = append(commissionIDs, line.CommissionID)
		}
		tx, txOK := txFromContext(txCtx)
		if !txOK {
			return biz.ErrCommissionApplicationInvalid
		}
		if confirmErr := confirmCommissionsForApplicationApproval(txCtx, tx, header.OrganizationID, decisionMaker, commissionIDs); confirmErr != nil {
			return confirmErr
		}
		total, amountErr := decimalOf(header.TotalCommissionAmount)
		if amountErr != nil {
			return amountErr
		}
		now := time.Now()
		updated, updateErr := client.FinanceCommissionApplication.UpdateOneID(header.ID).
			SetStatus(applicationent.StatusAPPROVED).
			SetVersion(header.Version + 1).
			SetDecidedAt(now).
			SetDecidedBy(decisionMaker).
			SetUpdatedAt(now).
			Save(txCtx)
		if updateErr != nil {
			return mapEntError(updateErr, biz.ErrCommissionApplicationNotFound, biz.ErrCommissionApplicationStatusConflict)
		}
		audit := biz.CommissionApplicationDecisionAudit(header.OrganizationID, decisionMaker, header.ID, header.EmployeeID,
			biz.CommissionApplicationDecisionAction(true), header.ApplicationMonth, header.CoverageTo, header.CommissionCount,
			total, header.Version+1, "")
		if auditErr := writeAudit(txCtx, client.AuditLog, audit); auditErr != nil {
			return auditErr
		}
		result = financeCommissionApplicationToBiz(updated)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Reject 整单驳回月度申请（design §5.3）：与批准共用申请头锁定与版本门禁；
// 只把申请转为 REJECTED、记录决策审计并递增版本，明细与提成单保持不变
// （提成仍为 DRAFT，员工重提路径承接）；驳回原因必填由领域入口校验。
func (r *financeCommissionApplicationRepo) Reject(ctx context.Context, organizationIDs []uuid.UUID, decisionMaker, id uuid.UUID, expectedVersion uint64, reason string) (*biz.FinanceCommissionApplication, error) {
	var result *biz.FinanceCommissionApplication
	err := r.data.WithinTransaction(ctx, func(txCtx context.Context) error {
		client, clientErr := r.data.client(txCtx)
		if clientErr != nil {
			return clientErr
		}
		header, lockErr := lockApplicationHeaderForDecision(txCtx, client, organizationIDs, id, expectedVersion)
		if lockErr != nil {
			return lockErr
		}
		total, amountErr := decimalOf(header.TotalCommissionAmount)
		if amountErr != nil {
			return amountErr
		}
		now := time.Now()
		updated, updateErr := client.FinanceCommissionApplication.UpdateOneID(header.ID).
			SetStatus(applicationent.StatusREJECTED).
			SetVersion(header.Version + 1).
			SetDecidedAt(now).
			SetDecidedBy(decisionMaker).
			SetDecisionReason(reason).
			SetUpdatedAt(now).
			Save(txCtx)
		if updateErr != nil {
			return mapEntError(updateErr, biz.ErrCommissionApplicationNotFound, biz.ErrCommissionApplicationStatusConflict)
		}
		audit := biz.CommissionApplicationDecisionAudit(header.OrganizationID, decisionMaker, header.ID, header.EmployeeID,
			biz.CommissionApplicationDecisionAction(false), header.ApplicationMonth, header.CoverageTo, header.CommissionCount,
			total, header.Version+1, reason)
		if auditErr := writeAudit(txCtx, client.AuditLog, audit); auditErr != nil {
			return auditErr
		}
		result = financeCommissionApplicationToBiz(updated)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Submit 提交或重提本人月度申请。事务内顺序：
//  1. 员工 Membership 行 FOR UPDATE（稳定父行锁，与方案写路径同锁序）串行化
//     同一员工的重复点击与并发提交；
//  2. 按月度唯一键锁定申请头：PENDING_REVIEW/APPROVED 稳定冲突；REJECTED 走
//     原申请重提（沿用原 ID 与申请月份，禁止替代申请）；不存在则新建；
//  3. 全量重算截止日以前的候选（来源固定锁序），逐候选复用或受控创建 DRAFT
//     提成并固化明细快照与头表汇总；空候选拒绝提交，任一来源变更整体回滚。
func (r *financeCommissionApplicationRepo) Submit(ctx context.Context, scope biz.WorkbenchScope) (*biz.FinanceCommissionApplication, error) {
	applicationMonth, coverageTo, ok := biz.CommissionApplicationPeriod(scope.Today)
	if !ok {
		return nil, biz.ErrCommissionApplicationInvalid
	}
	var result *biz.FinanceCommissionApplication
	err := r.data.WithinTransaction(ctx, func(txCtx context.Context) error {
		client, clientErr := r.data.client(txCtx)
		if clientErr != nil {
			return clientErr
		}
		if lockErr := lockApplicationMembership(txCtx, client, scope.OrganizationID, scope.UserID); lockErr != nil {
			return lockErr
		}
		existing, queryErr := client.FinanceCommissionApplication.Query().Where(
			applicationent.OrganizationIDEQ(scope.OrganizationID),
			applicationent.EmployeeIDEQ(scope.UserID),
			applicationent.ApplicationMonthEQ(applicationMonth),
		).ForUpdate().Only(txCtx)
		if queryErr != nil && !ent.IsNotFound(queryErr) {
			return queryErr
		}
		if existing != nil {
			if existing.Status != applicationent.StatusREJECTED {
				return biz.ErrCommissionApplicationConflict
			}
			header, resubmitErr := r.resubmitApplication(txCtx, client, existing, scope, applicationMonth, coverageTo)
			if resubmitErr != nil {
				return resubmitErr
			}
			result = header
			return nil
		}
		header, createErr := r.createApplication(txCtx, client, scope, applicationMonth, coverageTo)
		if createErr != nil {
			return createErr
		}
		result = header
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// lockApplicationMembership 按当前组织锁定员工成员关系父行并确认有效：与方案
// 写路径共用「Membership → …」锁序，同一员工的并发申请提交在此串行化。
func lockApplicationMembership(ctx context.Context, client *ent.Client, organizationID, userID uuid.UUID) error {
	rows, err := client.Membership.Query().
		Where(membership.OrganizationIDEQ(organizationID), membership.UserIDEQ(userID)).
		Order(membership.ByID()).
		ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return biz.ErrCommissionApplicationEmployeeInvalid
	}
	for _, row := range rows {
		if !row.Enabled {
			return biz.ErrCommissionApplicationEmployeeInvalid
		}
	}
	return nil
}

// resubmitApplication 在原申请上重提：不改变原绑定明细集合（design §3.2 不回流
// 公共池），不新增迟到来源；逐明细校验提成仍为 DRAFT 且快照指纹一致，任一失效
// 返回稳定冲突。重提回到 PENDING_REVIEW、清空上一版决策审计并递增版本。
func (r *financeCommissionApplicationRepo) resubmitApplication(ctx context.Context, client *ent.Client, existing *ent.FinanceCommissionApplication, scope biz.WorkbenchScope, applicationMonth, coverageTo string) (*biz.FinanceCommissionApplication, error) {
	lines, err := client.FinanceCommissionApplicationLine.Query().
		Where(applicationline.ApplicationIDEQ(existing.ID)).
		Order(applicationline.ByCommissionID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, biz.ErrCommissionApplicationSourceConflict
	}
	commissionIDs := make([]uuid.UUID, 0, len(lines))
	total := decimal.Zero
	for _, line := range lines {
		commissionIDs = append(commissionIDs, line.CommissionID)
	}
	// 已按 commission_id 升序锁定提成父单：与提成状态迁移的父子锁序一致。
	parents, err := client.FinanceCommission.Query().
		Where(commission.IDIn(commissionIDs...)).
		Order(ent.Asc(commission.FieldID)).
		ForUpdate().All(ctx)
	if err != nil {
		return nil, err
	}
	parentByID := make(map[uuid.UUID]*ent.FinanceCommission, len(parents))
	for _, parent := range parents {
		parentByID[parent.ID] = parent
	}
	for _, line := range lines {
		amount, amountErr := decimalOf(line.CommissionAmount)
		if amountErr != nil {
			return nil, amountErr
		}
		total = total.Add(amount).Round(8)
		parent, ok := parentByID[line.CommissionID]
		if !ok {
			return nil, biz.ErrCommissionApplicationSourceConflict
		}
		if parent.Status != commission.StatusDRAFT {
			return nil, biz.ErrCommissionApplicationSourceConflict
		}
		if parent.SourceFingerprint == "" || parent.SourceFingerprint != line.SourceFingerprint {
			return nil, biz.ErrCommissionApplicationSourceConflict
		}
	}
	now := time.Now()
	updated, err := client.FinanceCommissionApplication.UpdateOneID(existing.ID).
		SetStatus(applicationent.StatusPENDING_REVIEW).
		SetVersion(existing.Version + 1).
		ClearDecidedAt().
		ClearDecidedBy().
		ClearDecisionReason().
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrCommissionApplicationNotFound, biz.ErrCommissionApplicationConflict)
	}
	audit := biz.CommissionApplicationSubmitAudit(scope.OrganizationID, scope.UserID, existing.ID,
		biz.CommissionApplicationSubmitAction(true), applicationMonth, coverageTo, len(lines), total)
	if err := writeAudit(ctx, client.AuditLog, audit); err != nil {
		return nil, err
	}
	return financeCommissionApplicationToBiz(updated), nil
}

// createApplication 新建月度申请：全量重算候选，逐候选复用或受控创建 DRAFT
// 提成，写明细快照、头表汇总与审计。候选为空拒绝提交；唯一索引兜底并发提交。
func (r *financeCommissionApplicationRepo) createApplication(ctx context.Context, client *ent.Client, scope biz.WorkbenchScope, applicationMonth, coverageTo string) (*biz.FinanceCommissionApplication, error) {
	candidates, err := r.resolveApplicationCandidates(ctx, client, scope, coverageTo, true)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, biz.ErrCommissionApplicationEmpty
	}
	orgRow, err := client.Organization.Query().
		Where(organizationent.IDEQ(scope.OrganizationID)).
		Select(organizationent.FieldID, organizationent.FieldBaseCurrency).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	baseCurrency := optionalStringValue(orgRow.BaseCurrency)
	if baseCurrency == "" {
		return nil, biz.ErrCommissionApplicationInvalid
	}
	now := time.Now()
	headerID := uuid.Must(uuid.NewV7())
	total, totalCNY := decimal.Zero, decimal.Zero
	builders := make([]*ent.FinanceCommissionApplicationLineCreate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.baseCurrency != baseCurrency {
			return nil, biz.ErrCommissionApplicationSourceConflict
		}
		commissionID, amount, cnyAmount, ensureErr := r.ensureCandidateCommission(ctx, scope, candidate)
		if ensureErr != nil {
			return nil, ensureErr
		}
		total = total.Add(amount).Round(8)
		totalCNY = totalCNY.Add(cnyAmount).Round(8)
		builders = append(builders, client.FinanceCommissionApplicationLine.Create().
			SetID(uuid.Must(uuid.NewV7())).
			SetOrganizationID(scope.OrganizationID).
			SetEmployeeID(scope.UserID).
			SetApplicationID(headerID).
			SetCommissionID(commissionID).
			SetCommissionDate(candidate.commissionDate).
			SetNillableVerificationID(uuidPointerOrNil(candidate.verificationID)).
			SetNillableVerificationNo(stringPointerOrNil(candidate.verificationNo)).
			SetNillableNettingID(uuidPointerOrNil(candidate.nettingID)).
			SetNillableNettingNo(stringPointerOrNil(candidate.nettingNo)).
			SetPersonnelRole(string(candidate.role)).
			SetNillableRuleID(candidate.ruleID).
			SetRuleVersion(candidate.ruleVersion).
			SetNillableRuleName(candidate.ruleName).
			SetNillableCalculationBasis(candidate.calculationBasis).
			SetBaseCurrency(candidate.baseCurrency).
			SetCommissionAmount(amount.StringFixed(8)).
			SetCnyCommissionAmount(cnyAmount.StringFixed(8)).
			SetSourceFingerprint(candidate.fingerprint))
	}
	header, err := client.FinanceCommissionApplication.Create().
		SetID(headerID).
		SetOrganizationID(scope.OrganizationID).
		SetEmployeeID(scope.UserID).
		SetApplicationMonth(applicationMonth).
		SetCoverageTo(coverageTo).
		SetStatus(applicationent.StatusPENDING_REVIEW).
		SetVersion(1).
		SetCommissionCount(len(candidates)).
		SetBaseCurrency(baseCurrency).
		SetTotalCommissionAmount(total.StringFixed(8)).
		SetTotalCnyCommissionAmount(totalCNY.StringFixed(8)).
		SetSubmittedAt(now).
		SetSubmittedBy(scope.UserID).
		Save(ctx)
	if err != nil {
		// 月度唯一索引兜底：并发提交只有一个事务成功。
		return nil, mapEntConstraints(err,
			entConstraintMapping{name: "financecommissionapplication_organization_id_employee_id_application_month", domainErr: biz.ErrCommissionApplicationConflict},
			entConstraintMapping{name: "financecommissionapplicationline_commission_id", domainErr: biz.ErrCommissionApplicationSourceConflict},
		)
	}
	if _, err := client.FinanceCommissionApplicationLine.CreateBulk(builders...).Save(ctx); err != nil {
		return nil, mapEntConstraints(err,
			entConstraintMapping{name: "financecommissionapplicationline_commission_id", domainErr: biz.ErrCommissionApplicationSourceConflict},
		)
	}
	audit := biz.CommissionApplicationSubmitAudit(scope.OrganizationID, scope.UserID, headerID,
		biz.CommissionApplicationSubmitAction(false), applicationMonth, coverageTo, len(candidates), total)
	if err := writeAudit(ctx, client.AuditLog, audit); err != nil {
		return nil, err
	}
	return financeCommissionApplicationToBiz(header), nil
}

// ensureCandidateCommission 返回候选对应的提成事实 ID 与快照金额：已有本人未
// 占用 DRAFT 直接复用（沿用存储金额），否则经内部受控路径创建 DRAFT——复用
// CommissionUsecase.Create 内核（编号分配 → CNY 快照固化 → commissionRepo.Create
// 事务内重解析、活跃去重与草稿写入），嵌套事务合并到外层提交事务。
func (r *financeCommissionApplicationRepo) ensureCandidateCommission(ctx context.Context, scope biz.WorkbenchScope, candidate *applicationCandidate) (uuid.UUID, decimal.Decimal, decimal.Decimal, error) {
	if candidate.existingID != uuid.Nil {
		return candidate.existingID, candidate.amount, candidate.cnyAmount, nil
	}
	tx, ok := txFromContext(ctx)
	if !ok {
		return uuid.Nil, decimal.Zero, decimal.Zero, biz.ErrCommissionApplicationInvalid
	}
	now := time.Now()
	numberRule, sequence, err := allocateNumberInTx(ctx, tx, scope.OrganizationID, biz.DocumentTypeCommission, now)
	if err != nil {
		return uuid.Nil, decimal.Zero, decimal.Zero, err
	}
	commissionNo, err := biz.FormatAllocatedNumber(now, numberRule, sequence, "")
	if err != nil {
		return uuid.Nil, decimal.Zero, decimal.Zero, err
	}
	snapshot, err := biz.ResolveCommissionCNYRate(candidate.baseCurrency, candidate.commissionDate, decimal.Zero)
	if err != nil {
		return uuid.Nil, decimal.Zero, decimal.Zero, err
	}
	created := &biz.FinanceCommission{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: scope.OrganizationID,
		CommissionNo:   commissionNo,
		IdempotencyKey: "fca-" + uuid.Must(uuid.NewV7()).String(),
		VerificationID: candidate.verificationID,
		NettingID:      candidate.nettingID,
		EmployeeID:     scope.UserID,
		PersonnelRole:  candidate.role,
		Status:         biz.CommissionDraft,
		Version:        1,
	}
	// 受控创建复用 CommissionUsecase.Create 的写入内核；审计动作与明细与财务
	// 手工创建提成保持同一形状（biz.commissionCreateAudit 为包内私有，此处按
	// 同一动作名与字段构造）。
	audit := &biz.AuditEvent{
		OrganizationID: &scope.OrganizationID,
		UserID:         &scope.UserID,
		Action:         "finance.commission.create",
		Result:         "success",
		ResourceType:   "finance_commission",
		ResourceID:     created.ID.String(),
		Details: map[string]string{
			"commission_date":   snapshot.CommissionDate,
			"cny.exchange_rate": snapshot.ExchangeRate.StringFixed(8),
			"cny.rate_date":     snapshot.ExchangeRateDate,
			"cny.source":        snapshot.ExchangeRateSource,
		},
	}
	if err := (&commissionRepo{data: r.data}).Create(ctx, scope.OrganizationID, created, snapshot, audit); err != nil {
		return uuid.Nil, decimal.Zero, decimal.Zero, err
	}
	return created.ID, created.CommissionAmount, snapshot.CommissionAmount, nil
}

// txFromContext 取出共享事务句柄，供编号分配等需要 *ent.Tx 的内核复用；
// 仅在 WithinTransaction 建立的共享事务内可用。
func txFromContext(ctx context.Context) (*ent.Tx, bool) {
	transaction, ok := transactionFromContext(ctx)
	if !ok || !transaction.active.Load() {
		return nil, false
	}
	return transaction.tx, true
}

func uuidPointerOrNil(value uuid.UUID) *uuid.UUID {
	if value == uuid.Nil {
		return nil
	}
	return &value
}

func stringPointerOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func financeCommissionApplicationToBiz(item *ent.FinanceCommissionApplication) *biz.FinanceCommissionApplication {
	total, _ := decimalOf(item.TotalCommissionAmount)
	totalCNY, _ := decimalOf(item.TotalCnyCommissionAmount)
	return &biz.FinanceCommissionApplication{
		ID:                       item.ID,
		OrganizationID:           item.OrganizationID,
		EmployeeID:               item.EmployeeID,
		ApplicationMonth:         item.ApplicationMonth,
		CoverageTo:               item.CoverageTo,
		Status:                   biz.CommissionApplicationStatus(item.Status),
		Version:                  item.Version,
		CommissionCount:          item.CommissionCount,
		BaseCurrency:             item.BaseCurrency,
		TotalCommissionAmount:    total,
		TotalCNYCommissionAmount: totalCNY,
		SubmittedAt:              item.SubmittedAt,
		SubmittedBy:              item.SubmittedBy,
		DecidedAt:                item.DecidedAt,
		DecidedBy:                item.DecidedBy,
		DecisionReason:           item.DecisionReason,
		CreatedAt:                item.CreatedAt,
		UpdatedAt:                item.UpdatedAt,
	}
}

func financeCommissionApplicationLineToBiz(item *ent.FinanceCommissionApplicationLine) (*biz.FinanceCommissionApplicationLine, error) {
	amount, err := decimalOf(item.CommissionAmount)
	if err != nil {
		return nil, err
	}
	cnyAmount, err := decimalOf(item.CnyCommissionAmount)
	if err != nil {
		return nil, err
	}
	personnelRole := ""
	if item.PersonnelRole != "" {
		personnelRole = item.PersonnelRole
	}
	return &biz.FinanceCommissionApplicationLine{
		ID:                  item.ID,
		OrganizationID:      item.OrganizationID,
		EmployeeID:          item.EmployeeID,
		ApplicationID:       item.ApplicationID,
		CommissionID:        item.CommissionID,
		CommissionDate:      item.CommissionDate,
		VerificationID:      item.VerificationID,
		VerificationNo:      item.VerificationNo,
		NettingID:           item.NettingID,
		NettingNo:           item.NettingNo,
		PersonnelRole:       biz.CommissionPersonnelRole(personnelRole),
		RuleID:              item.RuleID,
		RuleVersion:         item.RuleVersion,
		RuleName:            item.RuleName,
		CalculationBasis:    item.CalculationBasis,
		BaseCurrency:        item.BaseCurrency,
		CommissionAmount:    amount,
		CNYCommissionAmount: cnyAmount,
		SourceFingerprint:   item.SourceFingerprint,
		CreatedAt:           item.CreatedAt,
	}, nil
}
