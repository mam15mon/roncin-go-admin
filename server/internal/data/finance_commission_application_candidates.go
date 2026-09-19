package data

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	bill "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	billline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	applicationline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionapplicationline"
	nettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	nettingalloc "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	verification "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	allocation "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	attribution "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	user "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

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
