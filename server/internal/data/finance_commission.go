package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	bill "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	billline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	adjustment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	commissionline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	rule "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	assignment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionruleassignment"
	nettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	nettingalloc "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	verification "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	allocation "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	attribution "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderfeesupplementent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeesupplementrequest"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
	"github.com/shopspring/decimal"
)

type commissionRepo struct{ data *Data }

func NewCommissionRepo(data *Data) biz.CommissionRepo { return &commissionRepo{data: data} }

func (r *commissionRepo) ListEmployees(ctx context.Context, org uuid.UUID, options biz.SelectorListOptions) (*biz.PagedList[*biz.CommissionEmployeeOption], error) {
	return r.ListEmployeesScoped(ctx, []uuid.UUID{org}, options)
}

func (r *commissionRepo) ListEmployeesScoped(ctx context.Context, organizationIDs []uuid.UUID, options biz.SelectorListOptions) (*biz.PagedList[*biz.CommissionEmployeeOption], error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	predicates := []predicate.User{
		user.EnabledEQ(true),
		user.HasMembershipsWith(membership.OrganizationIDIn(organizationIDs...), membership.EnabledEQ(true)),
	}
	if options.Keyword != "" {
		predicates = append(predicates, user.Or(
			user.UsernameContainsFold(options.Keyword),
			user.DisplayNameContainsFold(options.Keyword),
			user.SearchKeywordsContainsFold(options.Keyword),
		))
	}
	query := client.User.Query().Where(predicates...)
	return paginate(ctx, func(ctx context.Context) (int, error) {
		return query.Clone().Count(ctx)
	}, func(ctx context.Context, offset, limit int) ([]*ent.User, error) {
		return query.Order(user.ByDisplayName(), user.ByUsername(), user.ByID()).Offset(offset).Limit(limit).All(ctx)
	}, options.Page, options.PageSize, infalliblePageConverter(func(item *ent.User) *biz.CommissionEmployeeOption {
		return &biz.CommissionEmployeeOption{ID: item.ID, DisplayName: item.DisplayName}
	}))
}

// ListCandidates 按来源单发现计提候选：先按来源订单的提成归属发现「员工 +
// 人员身份」组合，再按来源归属日期解析「已启用方案 ∩ 该员工未取消有效分配」
// 的唯一命中；任一条件不满足的组合不产生候选。同一员工多身份形成独立候选。
func (r *commissionRepo) ListCandidates(ctx context.Context, org uuid.UUID, f biz.CommissionCandidateFilter) (*biz.CommissionCandidateListResult, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	store := commissionStoreFromClient(client)
	source, err := loadCommissionCalculationSource(ctx, store, org, f.VerificationID, f.NettingID, false)
	if err != nil {
		return nil, err
	}
	// 候选发现：来源涉及订单上存在应收费用事实的提成归属，按「员工 + 身份」
	// 去重组合；固定薪员工没有对应身份归属，自然不产生候选。
	attributions, err := client.OrderCommissionAttribution.Query().Where(
		attribution.OrganizationIDEQ(org),
		attribution.OrderIDIn(source.orderIDs...),
		attribution.HasOrderWith(orderent.HasFeesWith(
			fee.StatusIn(fee.StatusCONFIRMED, fee.StatusBILLED),
			fee.DirectionEQ(fee.DirectionRECEIVABLE),
			fee.BaseCurrencyAmountGT("0"),
		)),
	).Order(attribution.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	type candidateCombo struct {
		employeeID    uuid.UUID
		personnelRole biz.CommissionPersonnelRole
		employeeName  string
		attributions  []*ent.OrderCommissionAttribution
	}
	combos := make(map[string]*candidateCombo)
	for _, item := range attributions {
		role := biz.CommissionPersonnelRole(item.PersonnelRole)
		key := item.EmployeeID.String() + "|" + string(role)
		combo, exists := combos[key]
		if !exists {
			combo = &candidateCombo{employeeID: item.EmployeeID, personnelRole: role, employeeName: item.EmployeeName}
			combos[key] = combo
		}
		combo.attributions = append(combo.attributions, item)
	}
	// 逐组合解析唯一方案并试算；解析失败或来源条件不满足的组合静默跳过，
	// 其余错误（解析失败除外）原样外传。
	employees := make(map[uuid.UUID]*ent.User)
	skipped := map[error]bool{biz.ErrCommissionRuleNotResolved: true, biz.ErrCommissionSource: true, biz.ErrCommissionEmployeeRole: true}
	calculations := make([]*biz.CommissionCalculation, 0, len(combos))
	keys := make([]string, 0, len(combos))
	for key := range combos {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		combo := combos[key]
		if f.Keyword != "" && !strings.Contains(strings.ToLower(combo.employeeName), strings.ToLower(f.Keyword)) {
			continue
		}
		employee, exists := employees[combo.employeeID]
		if !exists {
			loaded, loadErr := store.users.Query().Where(user.IDEQ(combo.employeeID)).Only(ctx)
			if loadErr != nil {
				continue
			}
			employees[combo.employeeID] = loaded
			employee = loaded
		}
		ruleItem, assignmentItem, resolveErr := resolveCommissionRuleForDate(ctx, store, org, combo.employeeID, combo.personnelRole, source.commissionDate, false)
		if resolveErr != nil {
			if skipped[resolveErr] {
				continue
			}
			return nil, resolveErr
		}
		calculation, calculateErr := calculateCommissionFromSource(source, ruleItem, assignmentItem, employee, combo.attributions)
		if calculateErr != nil {
			if skipped[calculateErr] {
				continue
			}
			return nil, calculateErr
		}
		calculation.Lines = nil
		calculations = append(calculations, calculation)
	}
	total := len(calculations)
	start := (f.Page - 1) * f.PageSize
	if start > total {
		start = total
	}
	end := start + f.PageSize
	if end > total {
		end = total
	}
	result := &biz.CommissionCandidateListResult{
		Items: calculations[start:end], Total: int64(total), Page: f.Page, PageSize: f.PageSize,
	}
	return result, nil
}

// ListNettingCandidates 返回可计提的对冲单候选：已确认且存在有效应收分摊，
// 按创建时间倒序分页；关键字匹配对冲单号或结算单位名称。
func (r *commissionRepo) ListNettingCandidates(ctx context.Context, org uuid.UUID, f biz.CommissionNettingCandidateFilter) (*biz.CommissionNettingCandidateListResult, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	predicates := []predicate.FinanceNetting{
		nettingent.OrganizationIDEQ(org),
		nettingent.StatusEQ(nettingent.StatusCONFIRMED),
		nettingent.HasAllocationsWith(nettingalloc.ActiveEQ(true), nettingalloc.DirectionEQ(nettingalloc.DirectionRECEIVABLE)),
	}
	if f.Keyword != "" {
		predicates = append(predicates, nettingent.Or(
			nettingent.NettingNoContainsFold(f.Keyword),
			nettingent.SettlementPartyNameContainsFold(f.Keyword),
		))
	}
	query := client.FinanceNetting.Query().Where(predicates...)
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := financeNettingQueryWithRelations(query).
		Order(nettingent.ByCreatedAt(entsql.OrderDesc()), nettingent.ByID(entsql.OrderDesc())).
		Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.CommissionNettingCandidateListResult{
		Items: make([]*biz.FinanceNetting, 0, len(items)), Total: int64(total), Page: f.Page, PageSize: f.PageSize,
	}
	for _, item := range items {
		converted, convertErr := financeNettingToBiz(item)
		if convertErr != nil {
			return nil, convertErr
		}
		result.Items = append(result.Items, converted)
	}
	return result, nil
}

func (r *commissionRepo) ListRules(ctx context.Context, org uuid.UUID, f biz.CommissionRuleFilter) (*biz.CommissionRuleListResult, error) {
	return r.ListRulesScoped(ctx, []uuid.UUID{org}, f)
}
func (r *commissionRepo) ListRulesScoped(ctx context.Context, organizationIDs []uuid.UUID, f biz.CommissionRuleFilter) (*biz.CommissionRuleListResult, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	p := []predicate.FinanceCommissionRule{rule.OrganizationIDIn(organizationIDs...)}
	if f.Keyword != "" {
		// 关键字覆盖方案名称与适用员工（用户名、展示名与已有搜索键）。
		p = append(p, rule.Or(
			rule.NameContainsFold(f.Keyword),
			rule.HasAssignmentsWith(
				assignment.CancelledAtIsNil(),
				assignment.HasEmployeeWith(user.Or(
					user.UsernameContainsFold(f.Keyword),
					user.DisplayNameContainsFold(f.Keyword),
					user.SearchKeywordsContainsFold(f.Keyword),
				)),
			),
		))
	}
	if f.PersonnelRole != "" {
		p = append(p, rule.PersonnelRoleEQ(rule.PersonnelRole(f.PersonnelRole)))
	}
	if f.Enabled != nil {
		p = append(p, rule.EnabledEQ(*f.Enabled))
	}
	if f.EmployeeID != uuid.Nil {
		p = append(p, rule.HasAssignmentsWith(assignment.EmployeeIDEQ(f.EmployeeID), assignment.CancelledAtIsNil()))
	}
	q := commissionRuleQueryWithAssignments(client.FinanceCommissionRule.Query().WithOrganization()).Where(p...)
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	xs, err := q.Order(rule.ByEnabled(entsql.OrderDesc()), rule.ByCreatedAt(entsql.OrderDesc())).Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.CommissionRuleListResult{Items: make([]*biz.FinanceCommissionRule, 0, len(xs)), Total: int64(total)}
	for _, x := range xs {
		item, err := commissionRuleToBiz(x)
		if err != nil {
			return nil, err
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}

func (r *commissionRepo) GetRuleScoped(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*biz.FinanceCommissionRule, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	x, err := commissionRuleQueryWithAssignments(client.FinanceCommissionRule.Query().WithOrganization()).Where(rule.IDEQ(id), rule.OrganizationIDIn(organizationIDs...)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrCommissionRuleNotFound, nil)
	}
	return commissionRuleToBiz(x)
}

// commissionRuleQueryWithAssignments 让方案查询随载未取消分配段（当前/未来名单
// 与已终止历史段）及员工展示名，取消段不参与名单与资格投影。
func commissionRuleQueryWithAssignments(q *ent.FinanceCommissionRuleQuery) *ent.FinanceCommissionRuleQuery {
	return q.WithAssignments(func(aq *ent.FinanceCommissionRuleAssignmentQuery) {
		aq.Where(assignment.CancelledAtIsNil()).WithEmployee().Order(assignment.ByEffectiveFrom(), assignment.ByID())
	})
}

// lockCommissionRuleMemberships 按员工 ID 升序锁定当前组织对应的 Membership
// 父行并确认成员与账号有效。方案与名单的所有写入口共用「Membership → Rule」
// 固定锁序：同一员工的并发名单变更被成员行锁串行化，首次分配也不存在
// 「没有关系行可锁」的竞态。
func lockCommissionRuleMemberships(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, employeeIDs []uuid.UUID) ([]uuid.UUID, error) {
	sorted := uniqueSortedUUIDs(employeeIDs)
	if len(sorted) == 0 {
		return nil, biz.ErrCommissionRuleEmployeeInvalid
	}
	rows, err := tx.Membership.Query().
		Where(membership.OrganizationIDEQ(organizationID), membership.UserIDIn(sorted...)).
		Order(membership.ByUserID(), membership.ByID()).
		ForUpdate().
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(rows) != len(sorted) {
		return nil, biz.ErrCommissionRuleEmployeeInvalid
	}
	for _, row := range rows {
		if !row.Enabled {
			return nil, biz.ErrCommissionRuleEmployeeInvalid
		}
	}
	validUsers, err := tx.User.Query().
		Where(user.IDIn(sorted...), user.EnabledEQ(true)).
		Order(ent.Asc(user.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(validUsers) != len(sorted) {
		return nil, biz.ErrCommissionRuleEmployeeInvalid
	}
	return sorted, nil
}

// ensureCommissionRuleIntervalFree 在锁内校验员工在当前组织同身份的已启用方案
// 上，未取消分配的实际有效区间（方案区间 ∩ 分配区间）与候选区间不重叠；
// excludeAssignmentIDs 供后续变更段时排除自身。
func ensureCommissionRuleIntervalFree(ctx context.Context, tx *ent.Tx, organizationID, employeeID uuid.UUID, personnelRole biz.CommissionPersonnelRole, planFrom, planTo *string, segmentFrom string, segmentTo *string, excludeAssignmentIDs ...uuid.UUID) error {
	query := tx.FinanceCommissionRuleAssignment.Query().Where(
		assignment.OrganizationIDEQ(organizationID),
		assignment.EmployeeIDEQ(employeeID),
		assignment.CancelledAtIsNil(),
		assignment.HasRuleWith(
			rule.OrganizationIDEQ(organizationID),
			rule.EnabledEQ(true),
			rule.PersonnelRoleEQ(rule.PersonnelRole(personnelRole)),
		),
	).WithRule()
	if len(excludeAssignmentIDs) > 0 {
		query = query.Where(assignment.IDNotIn(excludeAssignmentIDs...))
	}
	rows, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		existing := row.Edges.Rule
		if existing == nil {
			continue
		}
		// 实际有效区间 = 方案区间 ∩ 分配区间；两段实际区间相交 ⇔ 方案区间
		// 相交且分配区间相交。
		if !biz.CommissionClosedIntervalsOverlap(optionalStringValue(planFrom), optionalStringValue(planTo), optionalStringValue(existing.EffectiveFrom), optionalStringValue(existing.EffectiveTo)) {
			continue
		}
		if !biz.CommissionClosedIntervalsOverlap(segmentFrom, optionalStringValue(segmentTo), row.EffectiveFrom, optionalStringValue(row.EffectiveTo)) {
			continue
		}
		return biz.ErrCommissionRuleIntervalOverlap
	}
	return nil
}

// commissionRuleEmployees 从分配段领域对象汇总员工 ID（去重升序由锁函数兜底）。
func commissionRuleEmployees(assignments []*biz.FinanceCommissionRuleAssignment) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(assignments))
	for _, item := range assignments {
		ids = append(ids, item.EmployeeID)
	}
	return ids
}

// createCommissionRuleAssignments 批量写入员工分配段；调用方必须已完成区间
// 冲突校验并持有成员行锁。
func createCommissionRuleAssignments(ctx context.Context, tx *ent.Tx, organizationID, ruleID, actorID uuid.UUID, employeeIDs []uuid.UUID, effectiveFrom string, effectiveTo *string) error {
	builders := make([]*ent.FinanceCommissionRuleAssignmentCreate, 0, len(employeeIDs))
	for _, employeeID := range employeeIDs {
		builders = append(builders, tx.FinanceCommissionRuleAssignment.Create().
			SetID(uuid.Must(uuid.NewV7())).
			SetOrganizationID(organizationID).
			SetRuleID(ruleID).
			SetEmployeeID(employeeID).
			SetEffectiveFrom(effectiveFrom).
			SetNillableEffectiveTo(effectiveTo).
			SetCreatedBy(actorID))
	}
	if _, err := tx.FinanceCommissionRuleAssignment.CreateBulk(builders...).Save(ctx); err != nil {
		return mapEntError(err, nil, biz.ErrCommissionRuleConflict)
	}
	return nil
}

// checkCommissionRuleAssignmentsFree 逐员工执行跨方案实际区间重叠校验。
func checkCommissionRuleAssignmentsFree(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, personnelRole biz.CommissionPersonnelRole, planFrom, planTo *string, employeeIDs []uuid.UUID, segmentFrom string, segmentTo *string) error {
	for _, employeeID := range employeeIDs {
		if overlapErr := ensureCommissionRuleIntervalFree(ctx, tx, organizationID, employeeID, personnelRole, planFrom, planTo, segmentFrom, segmentTo); overlapErr != nil {
			return overlapErr
		}
	}
	return nil
}

func (r *commissionRepo) CreateRule(ctx context.Context, org, actor uuid.UUID, item *biz.FinanceCommissionRule, audit *biz.AuditEvent) (*biz.FinanceCommissionRule, error) {
	if item.Enabled && len(item.Assignments) == 0 {
		return nil, biz.ErrCommissionRuleEnableNoEmployee
	}
	var x *ent.FinanceCommissionRule
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 固定锁序：先锁员工成员关系（含成员与账号有效性校验），再创建方案。
		employees, lockErr := lockCommissionRuleMemberships(ctx, tx, org, commissionRuleEmployees(item.Assignments))
		if lockErr != nil {
			return lockErr
		}
		var err error
		x, err = tx.FinanceCommissionRule.Create().SetID(item.ID).SetOrganizationID(org).SetName(item.Name).SetPersonnelRole(rule.PersonnelRole(item.PersonnelRole)).SetCalculationBasis(rule.CalculationBasis(item.CalculationBasis)).SetRatePercent(item.RatePercent.StringFixed(4)).SetNillableEffectiveFrom(item.EffectiveFrom).SetNillableEffectiveTo(item.EffectiveTo).SetEnabled(item.Enabled).SetLegacyReadonly(false).SetNillableNote(item.Note).SetVersion(1).Save(ctx)
		if err != nil {
			return mapEntError(err, nil, biz.ErrCommissionRuleConflict)
		}
		if len(employees) > 0 {
			if item.EffectiveFrom == nil {
				return biz.ErrCommissionRuleInvalid
			}
			if overlapErr := checkCommissionRuleAssignmentsFree(ctx, tx, org, item.PersonnelRole, item.EffectiveFrom, item.EffectiveTo, employees, *item.EffectiveFrom, item.EffectiveTo); overlapErr != nil {
				return overlapErr
			}
			if createErr := createCommissionRuleAssignments(ctx, tx, org, x.ID, actor, employees, *item.EffectiveFrom, item.EffectiveTo); createErr != nil {
				return createErr
			}
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return r.GetRuleScoped(ctx, []uuid.UUID{org}, x.ID)
}

// ensureCommissionRuleParamsLocked 校验已生效方案的锁定参数：人员身份、计提
// 口径、比例和起始日是历史计算依据，禁止原地修改，也不得以停用追溯失效。
func ensureCommissionRuleParamsLocked(x *ent.FinanceCommissionRule, in biz.UpdateCommissionRuleInput) error {
	existingRate, err := decimalOf(x.RatePercent)
	if err != nil {
		return err
	}
	if x.PersonnelRole != rule.PersonnelRole(in.PersonnelRole) ||
		x.CalculationBasis != rule.CalculationBasis(in.CalculationBasis) ||
		!existingRate.Equal(in.RatePercent) ||
		x.EffectiveFrom == nil || in.EffectiveFrom == nil || *x.EffectiveFrom != *in.EffectiveFrom ||
		!in.Enabled {
		return biz.ErrCommissionRuleLockedParams
	}
	return nil
}

func (r *commissionRepo) UpdateRule(ctx context.Context, org uuid.UUID, in biz.UpdateCommissionRuleInput, audit *biz.AuditEvent) (*biz.FinanceCommissionRule, error) {
	var updated *ent.FinanceCommissionRule
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 先按当前分配段预读需要锁定成员关系的员工集合，再按 Membership → Rule
		// 固定锁序锁定成员行与方案行；方案行的权威状态以锁定后重读为准。
		preAssignments, err := tx.FinanceCommissionRuleAssignment.Query().
			Where(assignment.RuleIDEQ(in.ID), assignment.CancelledAtIsNil()).All(ctx)
		if err != nil {
			return err
		}
		if len(preAssignments) > 0 {
			employeeIDs := make([]uuid.UUID, 0, len(preAssignments))
			for _, item := range preAssignments {
				employeeIDs = append(employeeIDs, item.EmployeeID)
			}
			if _, lockErr := lockCommissionRuleMemberships(ctx, tx, org, employeeIDs); lockErr != nil {
				return lockErr
			}
		}
		x, err := tx.FinanceCommissionRule.Query().Where(rule.IDEQ(in.ID), rule.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionRuleNotFound, nil)
		}
		if x.Version != in.ExpectedVersion {
			return biz.ErrCommissionRuleConflict
		}
		if x.LegacyReadonly {
			return biz.ErrCommissionRuleLegacyReadOnly
		}
		today := in.Today
		reached := x.Enabled && x.EffectiveFrom != nil && *x.EffectiveFrom <= today
		switch {
		case reached:
			// 已生效方案：锁定参数，仅名称、备注与不早于当天的终止日可调。
			if lockErr := ensureCommissionRuleParamsLocked(x, in); lockErr != nil {
				return lockErr
			}
			if in.EffectiveTo != nil && *in.EffectiveTo < today {
				return biz.ErrCommissionRuleRetroactive
			}
		default:
			// 未到生效日或停用草稿：允许完整编辑；已产生提成引用时参数同样
			// 锁定，避免历史来源被追溯重算。
			referenced, refErr := tx.FinanceCommission.Query().Where(commission.RuleIDEQ(in.ID)).Exist(ctx)
			if refErr != nil {
				return refErr
			}
			if referenced {
				if lockErr := ensureCommissionRuleParamsLocked(x, in); lockErr != nil {
					return lockErr
				}
			}
			// 启用停用草稿：重新校验名单（成员有效性与区间唯一）后放行。
			if !x.Enabled && in.Enabled {
				if enableErr := enableCommissionRuleDraft(ctx, tx, org, x, in); enableErr != nil {
					return enableErr
				}
			}
		}
		u := tx.FinanceCommissionRule.UpdateOneID(in.ID).SetName(in.Name).SetPersonnelRole(rule.PersonnelRole(in.PersonnelRole)).SetCalculationBasis(rule.CalculationBasis(in.CalculationBasis)).SetRatePercent(in.RatePercent.StringFixed(4)).SetEnabled(in.Enabled).SetVersion(x.Version + 1)
		if in.EffectiveFrom == nil {
			u.ClearEffectiveFrom()
		} else {
			u.SetEffectiveFrom(*in.EffectiveFrom)
		}
		if in.EffectiveTo == nil {
			u.ClearEffectiveTo()
		} else {
			u.SetEffectiveTo(*in.EffectiveTo)
		}
		if in.Note == nil {
			u.ClearNote()
		} else {
			u.SetNote(*in.Note)
		}
		updated, err = u.Save(ctx)
		if err != nil {
			return mapEntError(err, nil, biz.ErrCommissionRuleConflict)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return r.GetRuleScoped(ctx, []uuid.UUID{org}, updated.ID)
}

// enableCommissionRuleDraft 启用停用草稿前的锁内复核：至少一名未取消分配、
// 成员关系与账号有效、且与同身份其他启用方案的实际区间不重叠。
func enableCommissionRuleDraft(ctx context.Context, tx *ent.Tx, org uuid.UUID, x *ent.FinanceCommissionRule, in biz.UpdateCommissionRuleInput) error {
	segments, err := tx.FinanceCommissionRuleAssignment.Query().
		Where(assignment.RuleIDEQ(x.ID), assignment.CancelledAtIsNil()).All(ctx)
	if err != nil {
		return err
	}
	if len(segments) == 0 {
		return biz.ErrCommissionRuleEnableNoEmployee
	}
	employeeIDs := make([]uuid.UUID, 0, len(segments))
	for _, item := range segments {
		employeeIDs = append(employeeIDs, item.EmployeeID)
	}
	if _, lockErr := lockCommissionRuleMemberships(ctx, tx, org, employeeIDs); lockErr != nil {
		return lockErr
	}
	return checkCommissionRuleAssignmentsFree(ctx, tx, org, biz.CommissionPersonnelRole(x.PersonnelRole), in.EffectiveFrom, in.EffectiveTo, uniqueSortedUUIDs(employeeIDs), optionalStringValue(in.EffectiveFrom), in.EffectiveTo)
}

// commissionRuleReachedEffectiveness 判断方案是否已到达生效日：启用且起始日
// 不晚于业务日期。已生效方案进入参数锁定与名单变更日期约束。
func commissionRuleReachedEffectiveness(x *ent.FinanceCommissionRule, today string) bool {
	return x.Enabled && x.EffectiveFrom != nil && *x.EffectiveFrom <= today
}

func (r *commissionRepo) AssignRuleEmployees(ctx context.Context, org, actor uuid.UUID, change biz.CommissionRuleEmployeeChange, audit *biz.AuditEvent) (*biz.FinanceCommissionRule, error) {
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		employees, lockErr := lockCommissionRuleMemberships(ctx, tx, org, change.EmployeeIDs)
		if lockErr != nil {
			return lockErr
		}
		x, loadErr := tx.FinanceCommissionRule.Query().Where(rule.IDEQ(change.RuleID), rule.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if loadErr != nil {
			return mapEntError(loadErr, biz.ErrCommissionRuleNotFound, nil)
		}
		if x.Version != change.ExpectedVersion {
			return biz.ErrCommissionRuleConflict
		}
		if x.LegacyReadonly {
			return biz.ErrCommissionRuleLegacyReadOnly
		}
		today := change.Today
		var segmentFrom string
		if commissionRuleReachedEffectiveness(x, today) {
			// 已生效方案新增成员必须选择当天或未来的变更生效日。
			if change.ChangeEffectiveDate == "" || change.ChangeEffectiveDate < today {
				return biz.ErrCommissionRuleRetroactive
			}
			segmentFrom = change.ChangeEffectiveDate
		} else {
			// 未生效方案可直接调整名单：显式变更日不得回溯，缺省从方案起始日
			//（未来）或当天开始。
			switch {
			case change.ChangeEffectiveDate != "":
				if change.ChangeEffectiveDate < today {
					return biz.ErrCommissionRuleRetroactive
				}
				segmentFrom = change.ChangeEffectiveDate
			case x.EffectiveFrom != nil && *x.EffectiveFrom >= today:
				segmentFrom = *x.EffectiveFrom
			default:
				segmentFrom = today
			}
		}
		if x.EffectiveTo != nil && segmentFrom > *x.EffectiveTo {
			// 方案终止日早于新增起始日：不再接受新增成员。
			return biz.ErrCommissionRuleAssignmentInvalid
		}
		if overlapErr := checkCommissionRuleAssignmentsFree(ctx, tx, org, biz.CommissionPersonnelRole(x.PersonnelRole), x.EffectiveFrom, x.EffectiveTo, employees, segmentFrom, x.EffectiveTo); overlapErr != nil {
			return overlapErr
		}
		if createErr := createCommissionRuleAssignments(ctx, tx, org, x.ID, actor, employees, segmentFrom, x.EffectiveTo); createErr != nil {
			return createErr
		}
		if _, err := tx.FinanceCommissionRule.UpdateOneID(x.ID).SetVersion(x.Version + 1).Save(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return r.GetRuleScoped(ctx, []uuid.UUID{org}, change.RuleID)
}

func (r *commissionRepo) RemoveRuleEmployees(ctx context.Context, org, actor uuid.UUID, change biz.CommissionRuleEmployeeChange, audit *biz.AuditEvent) (*biz.FinanceCommissionRule, error) {
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		employees, lockErr := lockCommissionRuleMemberships(ctx, tx, org, change.EmployeeIDs)
		if lockErr != nil {
			return lockErr
		}
		x, loadErr := tx.FinanceCommissionRule.Query().Where(rule.IDEQ(change.RuleID), rule.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if loadErr != nil {
			return mapEntError(loadErr, biz.ErrCommissionRuleNotFound, nil)
		}
		if x.Version != change.ExpectedVersion {
			return biz.ErrCommissionRuleConflict
		}
		if x.LegacyReadonly {
			return biz.ErrCommissionRuleLegacyReadOnly
		}
		today := change.Today
		changeDate := change.ChangeEffectiveDate
		if changeDate == "" {
			changeDate = today
		}
		if changeDate < today {
			return biz.ErrCommissionRuleRetroactive
		}
		now := time.Now()
		changed := 0
		for _, employeeID := range employees {
			segments, loadErr := tx.FinanceCommissionRuleAssignment.Query().
				Where(assignment.RuleIDEQ(x.ID), assignment.EmployeeIDEQ(employeeID), assignment.CancelledAtIsNil()).
				Order(assignment.ByEffectiveFrom(), assignment.ByID()).
				All(ctx)
			if loadErr != nil {
				return loadErr
			}
			if len(segments) == 0 {
				return biz.ErrCommissionRuleAssignmentInvalid
			}
			for _, seg := range segments {
				switch {
				case seg.TerminatedAt != nil:
					// 已按离开日期终止的历史段不再改写，保留终止审计链。
					continue
				case changeDate <= seg.EffectiveFrom:
					// 变更日不晚于分配起始日：整段尚未生效，标记取消并保留审计，
					// 不写入起止倒置的无效区间。
					if _, updateErr := tx.FinanceCommissionRuleAssignment.UpdateOneID(seg.ID).
						SetCancelledAt(now).SetCancelledBy(actor).Save(ctx); updateErr != nil {
						return updateErr
					}
					changed++
				case seg.EffectiveTo == nil || *seg.EffectiveTo >= changeDate:
					// 以变更日前一日结束该段；已更早结束的段保持不变。
					if _, updateErr := tx.FinanceCommissionRuleAssignment.UpdateOneID(seg.ID).
						SetEffectiveTo(biz.FinanceDateBefore(changeDate)).
						SetTerminatedAt(now).SetTerminatedBy(actor).Save(ctx); updateErr != nil {
						return updateErr
					}
					changed++
				}
			}
		}
		if changed == 0 {
			return biz.ErrCommissionRuleAssignmentInvalid
		}
		if _, err := tx.FinanceCommissionRule.UpdateOneID(x.ID).SetVersion(x.Version + 1).Save(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return r.GetRuleScoped(ctx, []uuid.UUID{org}, change.RuleID)
}

func (r *commissionRepo) CopyRule(ctx context.Context, org, actor uuid.UUID, in biz.CopyCommissionRuleInput, plan *biz.FinanceCommissionRule, sourceAdjustAudit *biz.AuditEvent, createAudit *biz.AuditEvent) (*biz.FinanceCommissionRule, error) {
	if len(plan.Assignments) == 0 {
		return nil, biz.ErrCommissionRuleEnableNoEmployee
	}
	var x *ent.FinanceCommissionRule
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		employees, lockErr := lockCommissionRuleMemberships(ctx, tx, org, commissionRuleEmployees(plan.Assignments))
		if lockErr != nil {
			return lockErr
		}
		source, loadErr := tx.FinanceCommissionRule.Query().
			Where(rule.IDEQ(in.SourceRuleID), rule.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if loadErr != nil {
			return mapEntError(loadErr, biz.ErrCommissionRuleNotFound, nil)
		}
		// 旧方案终止日衔接：新方案生效日前一日结束；旧方案已更早结束或尚无
		// 起始日（迁移前旧规则）则不缩短。调整在事务内先于区间冲突校验生效，
		// 校验查询读到的即是衔接后的区间。
		prevDay := biz.FinanceDateBefore(in.EffectiveFrom)
		endsWithHandover := source.EffectiveFrom != nil && prevDay >= *source.EffectiveFrom &&
			(source.EffectiveTo == nil || *source.EffectiveTo >= in.EffectiveFrom)
		if endsWithHandover {
			if _, updateErr := tx.FinanceCommissionRule.UpdateOneID(source.ID).
				SetEffectiveTo(prevDay).SetVersion(source.Version + 1).Save(ctx); updateErr != nil {
				return updateErr
			}
			if auditErr := writeAudit(ctx, tx.AuditLog, sourceAdjustAudit); auditErr != nil {
				return auditErr
			}
		}
		var createErr error
		x, createErr = tx.FinanceCommissionRule.Create().SetID(plan.ID).SetOrganizationID(org).SetName(plan.Name).SetPersonnelRole(rule.PersonnelRole(plan.PersonnelRole)).SetCalculationBasis(rule.CalculationBasis(plan.CalculationBasis)).SetRatePercent(plan.RatePercent.StringFixed(4)).SetEffectiveFrom(in.EffectiveFrom).SetNillableEffectiveTo(plan.EffectiveTo).SetEnabled(true).SetLegacyReadonly(false).SetNillableNote(plan.Note).SetVersion(1).Save(ctx)
		if createErr != nil {
			return mapEntError(createErr, nil, biz.ErrCommissionRuleConflict)
		}
		if overlapErr := checkCommissionRuleAssignmentsFree(ctx, tx, org, plan.PersonnelRole, plan.EffectiveFrom, plan.EffectiveTo, employees, in.EffectiveFrom, plan.EffectiveTo); overlapErr != nil {
			return overlapErr
		}
		if createErr := createCommissionRuleAssignments(ctx, tx, org, x.ID, actor, employees, in.EffectiveFrom, plan.EffectiveTo); createErr != nil {
			return createErr
		}
		return writeAudit(ctx, tx.AuditLog, createAudit)
	}); err != nil {
		return nil, err
	}
	return r.GetRuleScoped(ctx, []uuid.UUID{org}, x.ID)
}

// commissionListPredicates 构造提成列表筛选谓词，列表与导出复用同一实现。
func commissionListPredicates(org uuid.UUID, f biz.CommissionFilter) []predicate.FinanceCommission {
	return commissionListPredicatesScoped([]uuid.UUID{org}, f)
}
func commissionListPredicatesScoped(organizationIDs []uuid.UUID, f biz.CommissionFilter) []predicate.FinanceCommission {
	p := []predicate.FinanceCommission{commission.OrganizationIDIn(organizationIDs...)}
	if f.Keyword != "" {
		p = append(p, commission.Or(commission.CommissionNoContainsFold(f.Keyword), commission.EmployeeNameContainsFold(f.Keyword), commission.RuleNameContainsFold(f.Keyword)))
	}
	if f.Status != "" {
		p = append(p, commission.StatusEQ(commission.Status(f.Status)))
	}
	if f.CommissionDateFrom != "" {
		p = append(p, commission.CommissionDateGTE(f.CommissionDateFrom))
	}
	if f.CommissionDateTo != "" {
		p = append(p, commission.CommissionDateLTE(f.CommissionDateTo))
	}
	return p
}

func (r *commissionRepo) List(ctx context.Context, org uuid.UUID, f biz.CommissionFilter) (*biz.CommissionListResult, error) {
	return r.ListScoped(ctx, []uuid.UUID{org}, f)
}
func (r *commissionRepo) ListScoped(ctx context.Context, organizationIDs []uuid.UUID, f biz.CommissionFilter) (*biz.CommissionListResult, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	q := client.FinanceCommission.Query().Where(commissionListPredicatesScoped(organizationIDs, f)...)
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	xs, err := q.WithOrganization().WithAdjustments(func(q *ent.FinanceCommissionAdjustmentQuery) {
		q.Order(adjustment.ByCreatedAt())
	}).Order(commission.ByCommissionDate(entsql.OrderDesc()), commission.ByCreatedAt(entsql.OrderDesc()), commission.ByID(entsql.OrderDesc())).Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.CommissionListResult{Items: make([]*biz.FinanceCommission, 0, len(xs)), Total: int64(total)}
	for _, x := range xs {
		item, err := commissionWithLinesToBiz(x)
		if err != nil {
			return nil, err
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}

// Count 按列表同一谓词统计提成总数，供导出上限门禁使用。
func (r *commissionRepo) Count(ctx context.Context, org uuid.UUID, f biz.CommissionFilter) (int64, error) {
	return r.CountScoped(ctx, []uuid.UUID{org}, f)
}

func (r *commissionRepo) CountScoped(ctx context.Context, organizationIDs []uuid.UUID, f biz.CommissionFilter) (int64, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return 0, err
	}
	total, err := client.FinanceCommission.Query().Where(commissionListPredicatesScoped(organizationIDs, f)...).Count(ctx)
	if err != nil {
		return 0, err
	}
	return int64(total), nil
}

// ExportBatch 按列表同谓词与稳定排序（commission_date DESC, created_at DESC,
// id DESC）分页读取一批提成；调整单随行加载，保证 CNY 调整与有效金额的
// 动态口径与列表一致。
func (r *commissionRepo) ExportBatch(ctx context.Context, org uuid.UUID, f biz.CommissionFilter) ([]*biz.FinanceCommission, error) {
	return r.ExportBatchScoped(ctx, []uuid.UUID{org}, f)
}

func (r *commissionRepo) ExportBatchScoped(ctx context.Context, organizationIDs []uuid.UUID, f biz.CommissionFilter) ([]*biz.FinanceCommission, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	xs, err := client.FinanceCommission.Query().Where(commissionListPredicatesScoped(organizationIDs, f)...).WithOrganization().WithAdjustments(func(q *ent.FinanceCommissionAdjustmentQuery) {
		q.Order(adjustment.ByCreatedAt())
	}).Order(commission.ByCommissionDate(entsql.OrderDesc()), commission.ByCreatedAt(entsql.OrderDesc()), commission.ByID(entsql.OrderDesc())).Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*biz.FinanceCommission, 0, len(xs))
	for _, x := range xs {
		item, convertErr := commissionWithLinesToBiz(x)
		if convertErr != nil {
			return nil, convertErr
		}
		items = append(items, item)
	}
	return items, nil
}

// SaveExportAudit 在导出成功返回前持久化业务审计；导出为只读链路，不参与
// 共享事务，审计写入失败时由用例整体失败。
func (r *commissionRepo) SaveExportAudit(ctx context.Context, event *biz.AuditEvent) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	return writeAudit(ctx, client.AuditLog, event)
}

func (r *commissionRepo) GetByKey(ctx context.Context, org uuid.UUID, key string) (*biz.FinanceCommission, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	x, err := client.FinanceCommission.Query().Where(commission.OrganizationIDEQ(org), commission.IdempotencyKeyEQ(key)).WithOrganization().WithLines(func(q *ent.FinanceCommissionLineQuery) {
		q.Order(commissionline.ByOrderNo(), commissionline.ByOrderID())
	}).WithAdjustments(func(q *ent.FinanceCommissionAdjustmentQuery) {
		q.Order(adjustment.ByCreatedAt())
	}).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return commissionWithLinesToBiz(x)
}

type commissionCalculationStore struct {
	verifications *ent.FinanceVerificationClient
	nettings      *ent.FinanceNettingClient
	rules         *ent.FinanceCommissionRuleClient
	users         *ent.UserClient
	bills         *ent.FinanceBillClient
	billLines     *ent.FinanceBillLineClient
	attributions  *ent.OrderCommissionAttributionClient
	fees          *ent.OrderFeeClient
	orders        *ent.OrderClient
}

func commissionStoreFromClient(client *ent.Client) commissionCalculationStore {
	return commissionCalculationStore{verifications: client.FinanceVerification, nettings: client.FinanceNetting, rules: client.FinanceCommissionRule, users: client.User, bills: client.FinanceBill, billLines: client.FinanceBillLine, attributions: client.OrderCommissionAttribution, fees: client.OrderFee, orders: client.Order}
}

func commissionStoreFromTx(tx *ent.Tx) commissionCalculationStore {
	return commissionCalculationStore{verifications: tx.FinanceVerification, nettings: tx.FinanceNetting, rules: tx.FinanceCommissionRule, users: tx.User, bills: tx.FinanceBill, billLines: tx.FinanceBillLine, attributions: tx.OrderCommissionAttribution, fees: tx.OrderFee, orders: tx.Order}
}

func commissionCalculationBillsQuery(store commissionCalculationStore, org uuid.UUID, billIDs []uuid.UUID, lock bool) *ent.FinanceBillQuery {
	bq := store.bills.Query().
		Where(
			bill.IDIn(billIDs...),
			bill.OrganizationIDEQ(org),
			bill.StatusEQ(bill.StatusCONFIRMED),
			bill.DirectionEQ(bill.DirectionRECEIVABLE),
		).
		WithLines().
		Order(bill.ByID())
	if lock {
		bq.ForUpdate()
	}
	return bq
}

func (r *commissionRepo) Preview(ctx context.Context, org, verificationID, nettingID, employeeID uuid.UUID, personnelRole biz.CommissionPersonnelRole) (*biz.CommissionCalculation, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	return calculateCommission(ctx, commissionStoreFromClient(client), org, verificationID, nettingID, employeeID, personnelRole, false)
}

// resolveCommissionRuleForDate 按来源归属日期为「组织 + 员工 + 人员身份」解析
// 唯一有效方案与员工分配：方案必须已启用且归属日期落在方案区间内，且该员工
// 存在未取消分配、归属日期同样落在分配区间内。零命中或多命中（并发或数据
// 异常）均返回 ErrCommissionRuleNotResolved；迁移前停用的旧规则因 enabled=false
// 自然被排除。lock=true 时按方案主键升序 ForUpdate，与方案写入共享串行化点。
func resolveCommissionRuleForDate(ctx context.Context, store commissionCalculationStore, org, employeeID uuid.UUID, personnelRole biz.CommissionPersonnelRole, commissionDate string, lock bool) (*ent.FinanceCommissionRule, *ent.FinanceCommissionRuleAssignment, error) {
	if commissionDate == "" {
		return nil, nil, biz.ErrCommissionRuleNotResolved
	}
	q := store.rules.Query().Where(
		rule.OrganizationIDEQ(org),
		rule.EnabledEQ(true),
		rule.PersonnelRoleEQ(rule.PersonnelRole(personnelRole)),
	).WithAssignments(func(aq *ent.FinanceCommissionRuleAssignmentQuery) {
		aq.Where(assignment.EmployeeIDEQ(employeeID), assignment.CancelledAtIsNil())
	}).Order(rule.ByID())
	if lock {
		q.ForUpdate()
	}
	rules, err := q.All(ctx)
	if err != nil {
		return nil, nil, err
	}
	var (
		hitRule       *ent.FinanceCommissionRule
		hitAssignment *ent.FinanceCommissionRuleAssignment
	)
	for _, ruleItem := range rules {
		if (ruleItem.EffectiveFrom != nil && commissionDate < *ruleItem.EffectiveFrom) || (ruleItem.EffectiveTo != nil && commissionDate > *ruleItem.EffectiveTo) {
			continue
		}
		for _, seg := range ruleItem.Edges.Assignments {
			if commissionDate < seg.EffectiveFrom || (seg.EffectiveTo != nil && commissionDate > *seg.EffectiveTo) {
				continue
			}
			if hitRule != nil {
				// 同员工同身份在同一归属日期命中多个方案：数据异常或并发窗口，
				// 拒绝生成而不是静默选择。
				return nil, nil, biz.ErrCommissionRuleNotResolved
			}
			hitRule, hitAssignment = ruleItem, seg
		}
	}
	if hitRule == nil {
		return nil, nil, biz.ErrCommissionRuleNotResolved
	}
	return hitRule, hitAssignment, nil
}

// GetGenerationContext 读取生成提成所需的来源上下文：归属日期和本位币。
// 核销来源的归属日期为 verification_date；对冲来源为确认日（confirmed_at 的 UTC 日期）。
// CNY 折算汇率按归属日期解析，来源单不再携带汇率快照。
// 事务内首次读取即加 ForUpdate：同一事务内的写入阶段还会对同一来源行 ForUpdate，
// 若先 ForShare 再升级，两个并发创建事务可同持共享锁互等升级形成死锁，因此从入口
// 串行化；普通上下文保持无锁读取。
func (r *commissionRepo) GetGenerationContext(ctx context.Context, org, verificationID, nettingID uuid.UUID) (*biz.CommissionGenerationContext, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	if nettingID != uuid.Nil {
		query := client.FinanceNetting.Query().Where(nettingent.IDEQ(nettingID), nettingent.OrganizationIDEQ(org))
		if _, transactional := transactionFromContext(ctx); transactional {
			query.ForUpdate()
		}
		item, queryErr := query.Only(ctx)
		if queryErr != nil {
			return nil, mapEntError(queryErr, biz.ErrCommissionSource, nil)
		}
		if item.Status != nettingent.StatusCONFIRMED || item.ConfirmedAt == nil {
			return nil, biz.ErrCommissionSource
		}
		return &biz.CommissionGenerationContext{CommissionDate: item.ConfirmedAt.UTC().Format("2006-01-02"), BaseCurrency: item.BaseCurrency}, nil
	}
	query := client.FinanceVerification.Query().Where(verification.IDEQ(verificationID), verification.OrganizationIDEQ(org))
	if _, transactional := transactionFromContext(ctx); transactional {
		query.ForUpdate()
	}
	v, err := query.Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrCommissionSource, nil)
	}
	return &biz.CommissionGenerationContext{CommissionDate: v.VerificationDate, BaseCurrency: v.BaseCurrency}, nil
}

// commissionCalculationSource 是提成计算的来源快照：核销或对冲二选一。
// 两种来源同构地按「分摊金额 / 账单总额 × 账单行本位币」摊入 orderRealized。
// 方案与员工分配不在此结构内：由调用方按「员工 + 人员身份 + 归属日期」解析。
type commissionCalculationSource struct {
	organizationID    uuid.UUID
	verification      *ent.FinanceVerification // 核销来源时非空
	netting           *ent.FinanceNetting      // 对冲来源时非空
	commissionDate    string                   // 归属日期：核销 verification_date / 对冲确认日
	baseCurrency      string
	orderIDs          []uuid.UUID
	orderRealized     map[uuid.UUID]decimal.Decimal
	orderByID         map[uuid.UUID]*ent.Order
	feesByOrder       map[uuid.UUID][]*ent.OrderFee
	billLineBaseByFee map[uuid.UUID]decimal.Decimal
	fingerprintBase   []string
}

// commissionSourceAllocation 是核销/对冲分摊的统一条目，供下游同一聚合管线消费。
type commissionSourceAllocation struct {
	billID      uuid.UUID
	amount      decimal.Decimal
	fingerprint string
}

func calculateCommission(ctx context.Context, store commissionCalculationStore, org, verificationID, nettingID, employeeID uuid.UUID, personnelRole biz.CommissionPersonnelRole, lock bool) (*biz.CommissionCalculation, error) {
	source, err := loadCommissionCalculationSource(ctx, store, org, verificationID, nettingID, lock)
	if err != nil {
		return nil, err
	}
	uq := store.users.Query().Where(user.IDEQ(employeeID))
	if lock {
		uq.ForUpdate()
	}
	employee, err := uq.Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrCommissionInvalid, nil)
	}
	// 员工已离职（账号停用、成员关系解除）不影响解析：历史资格只由来源日期、
	// 方案区间与分配区间决定，不以当前账号状态抹除。
	ruleItem, assignmentItem, err := resolveCommissionRuleForDate(ctx, store, org, employeeID, personnelRole, source.commissionDate, lock)
	if err != nil {
		return nil, err
	}
	aq := store.attributions.Query().Where(
		attribution.OrganizationIDEQ(org),
		attribution.OrderIDIn(source.orderIDs...),
		attribution.EmployeeIDEQ(employeeID),
		attribution.PersonnelRoleEQ(attribution.PersonnelRole(personnelRole)),
	).Order(attribution.ByAttributedAt(), attribution.ByID())
	if lock {
		aq.ForUpdate()
	}
	attributions, err := aq.All(ctx)
	if err != nil {
		return nil, err
	}
	return calculateCommissionFromSource(source, ruleItem, assignmentItem, employee, attributions)
}

// loadCommissionCalculationSource 按来源（核销 ACTIVE+RECEIVABLE / 对冲 CONFIRMED 的
// RECEIVABLE 分摊）加载统一分摊条目，并锁定账单、订单、费用与账单行事实。
// 多行加锁一律按主键稳定排序，固定加锁顺序防止并发提成创建死锁。
func loadCommissionCalculationSource(ctx context.Context, store commissionCalculationStore, org, verificationID, nettingID uuid.UUID, lock bool) (*commissionCalculationSource, error) {
	var (
		fingerprintParts   []string
		commissionDate     string
		allocationEntries  []commissionSourceAllocation
		verificationEntity *ent.FinanceVerification
		nettingEntity      *ent.FinanceNetting
	)
	if nettingID != uuid.Nil {
		nq := store.nettings.Query().Where(nettingent.IDEQ(nettingID), nettingent.OrganizationIDEQ(org)).WithAllocations(func(q *ent.FinanceNettingAllocationQuery) {
			q.Where(nettingalloc.ActiveEQ(true))
		})
		if lock {
			nq.ForUpdate()
		}
		item, err := nq.Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrCommissionSource, nil)
		}
		if item.Status != nettingent.StatusCONFIRMED || item.ConfirmedAt == nil {
			return nil, biz.ErrCommissionSource
		}
		nettingEntity = item
		commissionDate = item.ConfirmedAt.UTC().Format("2006-01-02")
		fingerprintParts = append(fingerprintParts,
			fmt.Sprintf("calculation|%s", biz.CommissionCalculationVersion),
			fmt.Sprintf("src=netting|%s", item.ID),
			fmt.Sprintf("netting|%s|%s|%s|%s|%d", item.ID, item.NettingNo, item.Status, commissionDate, item.Version),
		)
		for _, allocationItem := range item.Edges.Allocations {
			if allocationItem.Direction != nettingalloc.DirectionRECEIVABLE {
				continue
			}
			amount, parseErr := decimalOf(allocationItem.Amount)
			if parseErr != nil {
				return nil, parseErr
			}
			allocationEntries = append(allocationEntries, commissionSourceAllocation{
				billID:      allocationItem.BillID,
				amount:      amount,
				fingerprint: fmt.Sprintf("netting_allocation|%s|%s|%s|%s|%t", allocationItem.ID, allocationItem.BillID, allocationItem.Direction, allocationItem.Amount, allocationItem.Active),
			})
		}
	} else {
		vq := store.verifications.Query().Where(verification.IDEQ(verificationID), verification.OrganizationIDEQ(org)).WithAllocations(func(q *ent.FinanceVerificationAllocationQuery) {
			q.Where(allocation.ActiveEQ(true))
		})
		if lock {
			vq.ForUpdate()
		}
		v, err := vq.Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrCommissionSource, nil)
		}
		if v.Status != verification.StatusACTIVE || v.Direction != verification.DirectionRECEIVABLE {
			return nil, biz.ErrCommissionSource
		}
		verificationEntity = v
		commissionDate = v.VerificationDate
		fingerprintParts = append(fingerprintParts,
			fmt.Sprintf("calculation|%s", biz.CommissionCalculationVersion),
			fmt.Sprintf("src=verification|%s", v.ID),
			fmt.Sprintf("verification|%s|%s|%s|%s|%s|%d", v.ID, v.VerificationNo, v.Status, v.Direction, v.VerificationDate, v.Version),
		)
		for _, item := range v.Edges.Allocations {
			amount, parseErr := decimalOf(item.Amount)
			if parseErr != nil {
				return nil, parseErr
			}
			allocationEntries = append(allocationEntries, commissionSourceAllocation{
				billID:      item.BillID,
				amount:      amount,
				fingerprint: fmt.Sprintf("allocation|%s|%s|%s|%s|%t", item.ID, item.BillID, item.CashflowID, item.Amount, item.Active),
			})
		}
	}
	if len(allocationEntries) == 0 {
		return nil, biz.ErrCommissionSource
	}
	billIDs := make([]uuid.UUID, 0, len(allocationEntries))
	allocationByBill := make(map[uuid.UUID]decimal.Decimal)
	seenBills := make(map[uuid.UUID]struct{})
	for _, item := range allocationEntries {
		if _, exists := seenBills[item.billID]; !exists {
			seenBills[item.billID] = struct{}{}
			billIDs = append(billIDs, item.billID)
		}
		allocationByBill[item.billID] = allocationByBill[item.billID].Add(item.amount)
		fingerprintParts = append(fingerprintParts, item.fingerprint)
	}
	bq := commissionCalculationBillsQuery(store, org, billIDs, lock)
	bills, err := bq.All(ctx)
	if err != nil {
		return nil, err
	}
	if len(bills) != len(billIDs) {
		return nil, biz.ErrCommissionSource
	}
	orderRealized := make(map[uuid.UUID]decimal.Decimal)
	baseCurrency := ""
	for _, billItem := range bills {
		total, parseErr := decimalOf(billItem.TotalAmount)
		if parseErr != nil || !total.IsPositive() {
			return nil, biz.ErrCommissionSource
		}
		ratio := allocationByBill[billItem.ID].Div(total)
		fingerprintParts = append(fingerprintParts, fmt.Sprintf("bill|%s|%s|%s|%s|%s|%d", billItem.ID, billItem.BillNo, billItem.Status, billItem.TotalAmount, billItem.BaseCurrencyAmount, billItem.Version))
		for _, billLine := range billItem.Edges.Lines {
			if !billLine.Active {
				continue
			}
			base, parseErr := decimalOf(billLine.BaseCurrencyAmount)
			if parseErr != nil {
				return nil, parseErr
			}
			if baseCurrency == "" {
				baseCurrency = billLine.BaseCurrency
			} else if baseCurrency != billLine.BaseCurrency {
				return nil, biz.ErrCommissionSource
			}
			orderRealized[billLine.OrderID] = orderRealized[billLine.OrderID].Add(base.Mul(ratio))
			fingerprintParts = append(fingerprintParts, fmt.Sprintf("bill_line|%s|%s|%s|%s|%s|%t", billLine.ID, billLine.OrderFeeID, billLine.OrderID, billLine.BaseCurrencyAmount, billLine.BaseCurrency, billLine.Active))
		}
	}
	orderIDs := make([]uuid.UUID, 0, len(orderRealized))
	for id := range orderRealized {
		orderIDs = append(orderIDs, id)
	}
	if len(orderIDs) == 0 || baseCurrency == "" {
		return nil, biz.ErrCommissionSource
	}
	sort.Slice(orderIDs, func(i, j int) bool { return orderIDs[i].String() < orderIDs[j].String() })
	oq := store.orders.Query().Where(orderent.IDIn(orderIDs...), orderent.OrganizationIDEQ(org)).WithCustomer().Order(orderent.ByID())
	if lock {
		oq.ForUpdate()
	}
	orders, err := oq.All(ctx)
	if err != nil {
		return nil, err
	}
	if len(orders) != len(orderIDs) {
		return nil, biz.ErrCommissionSource
	}
	orderByID := make(map[uuid.UUID]*ent.Order, len(orders))
	for _, item := range orders {
		orderByID[item.ID] = item
		fingerprintParts = append(fingerprintParts, fmt.Sprintf("order|%s|%s|%s|%s|%d", item.ID, item.OrderNo, item.CustomerID, item.OrderDate, item.Version))
	}
	fq := store.fees.Query().Where(fee.OrderIDIn(orderIDs...), fee.StatusIn(fee.StatusCONFIRMED, fee.StatusBILLED)).WithSettlementParty().Order(fee.ByOrderID(), fee.ByCreatedAt(), fee.ByID())
	if lock {
		fq.ForUpdate()
	}
	fees, err := fq.All(ctx)
	if err != nil {
		return nil, err
	}
	feesByOrder := make(map[uuid.UUID][]*ent.OrderFee)
	for _, item := range fees {
		feesByOrder[item.OrderID] = append(feesByOrder[item.OrderID], item)
	}
	// B3：分母聚合拉齐分子口径。分子（已实现收入）按账单行本位币快照计算，
	// 分母（订单总应收/总应付）改用各费用关联账单行的本位币合计；尚未建账的费用
	// 没有账单日快照，回落到费用自身本位币快照，保证分母仍覆盖订单全部费用。
	feeIDs := make([]uuid.UUID, 0, len(fees))
	for _, item := range fees {
		feeIDs = append(feeIDs, item.ID)
	}
	billLineQuery := store.billLines.Query().Where(billline.OrderFeeIDIn(feeIDs...), billline.ActiveEQ(true)).
		// 多行加锁前先按主键排序，固定加锁顺序防止并发提成创建在重叠账单行上死锁。
		Order(billline.ByID())
	if lock {
		billLineQuery.ForUpdate()
	}
	feeBillLines, err := billLineQuery.All(ctx)
	if err != nil {
		return nil, err
	}
	billLineBaseByFee := make(map[uuid.UUID]decimal.Decimal, len(feeBillLines))
	for _, line := range feeBillLines {
		base, parseErr := decimalOf(line.BaseCurrencyAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		billLineBaseByFee[line.OrderFeeID] = billLineBaseByFee[line.OrderFeeID].Add(base)
		fingerprintParts = append(fingerprintParts, fmt.Sprintf("fee_bill_line|%s|%s|%s|%t", line.ID, line.OrderFeeID, line.BaseCurrencyAmount, line.Active))
	}
	return &commissionCalculationSource{
		organizationID: org, verification: verificationEntity, netting: nettingEntity, commissionDate: commissionDate,
		baseCurrency: baseCurrency,
		orderIDs:     orderIDs, orderRealized: orderRealized, orderByID: orderByID, feesByOrder: feesByOrder,
		billLineBaseByFee: billLineBaseByFee,
		fingerprintBase:   fingerprintParts,
	}, nil
}

// calculateCommissionFromSource 基于来源快照、已解析的方案与员工分配、员工及其
// 提成归属逐订单计算提成。指纹只覆盖影响来源日期计算资格的稳定标识：方案参数
// （身份、口径、比例、起始日）与分配段身份。方案版本、终止日与名单后续变更
// （未来生效）不改变来源日期的资格，不参与指纹，保证历史快照确认不被重算。
func calculateCommissionFromSource(source *commissionCalculationSource, ruleItem *ent.FinanceCommissionRule, assignmentItem *ent.FinanceCommissionRuleAssignment, employee *ent.User, attributions []*ent.OrderCommissionAttribution) (*biz.CommissionCalculation, error) {
	if ruleItem == nil || assignmentItem == nil || !ruleItem.Enabled {
		return nil, biz.ErrCommissionRuleNotResolved
	}
	rate, err := decimalOf(ruleItem.RatePercent)
	if err != nil {
		return nil, err
	}
	fingerprintParts := append([]string(nil), source.fingerprintBase...)
	fingerprintParts = append(fingerprintParts,
		fmt.Sprintf("rule|%s|%s|%s|%s|%s", ruleItem.ID, ruleItem.PersonnelRole, ruleItem.CalculationBasis, ruleItem.RatePercent, optionalStringValue(ruleItem.EffectiveFrom)),
		fmt.Sprintf("rule_assignment|%s|%s|%s", assignmentItem.ID, assignmentItem.EmployeeID, assignmentItem.EffectiveFrom),
		fmt.Sprintf("employee|%s|%s|%t", employee.ID, employee.DisplayName, employee.Enabled),
	)
	attributionByOrder := make(map[uuid.UUID]*ent.OrderCommissionAttribution, len(attributions))
	for _, item := range attributions {
		attributionByOrder[item.OrderID] = item
		fingerprintParts = append(fingerprintParts, fmt.Sprintf("order_attribution|%s|%s|%s|%s|%s|%s|%s|%s", item.ID, item.OrderID, item.CustomerID, item.EmployeeID, item.PersonnelRole, item.SourceAssignmentID, item.EmployeeName, item.AttributedAt.UTC().Format(time.RFC3339Nano)))
	}
	eligibleOrderIDs := make([]uuid.UUID, 0, len(source.orderIDs))
	for _, id := range source.orderIDs {
		if _, eligible := attributionByOrder[id]; eligible {
			eligibleOrderIDs = append(eligibleOrderIDs, id)
		}
	}
	if len(eligibleOrderIDs) == 0 {
		return nil, biz.ErrCommissionEmployeeRole
	}
	result := &biz.CommissionCalculation{
		EmployeeID: employee.ID, EmployeeName: attributions[0].EmployeeName,
		RuleID: ruleItem.ID, RuleName: ruleItem.Name, PersonnelRole: biz.CommissionPersonnelRole(ruleItem.PersonnelRole),
		CalculationBasis: biz.CommissionCalculationBasis(ruleItem.CalculationBasis), RuleVersion: ruleItem.Version,
		CalculationVersion: biz.CommissionCalculationVersion, BaseCurrency: source.baseCurrency, RatePercent: rate,
		Lines: make([]*biz.FinanceCommissionLine, 0, len(eligibleOrderIDs)),
	}
	if source.verification != nil {
		result.VerificationID = source.verification.ID
		result.VerificationNo = source.verification.VerificationNo
	}
	if source.netting != nil {
		result.NettingID = source.netting.ID
		result.NettingNo = source.netting.NettingNo
	}
	customersWithFees := make(map[uuid.UUID]struct{})
	for _, orderID := range eligibleOrderIDs {
		orderItem := source.orderByID[orderID]
		orderFees := source.feesByOrder[orderItem.ID]
		customer, edgeErr := orderItem.Edges.CustomerOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		attributionItem := attributionByOrder[orderItem.ID]
		line := &biz.FinanceCommissionLine{
			OrderID: orderItem.ID, OrderNo: orderItem.OrderNo, OrderDate: orderItem.OrderDate,
			CustomerID: customer.ID, CustomerCode: partnerCodeValue(customer.Code), CustomerName: customer.LegalName,
			CustomerAssignmentID: attributionItem.SourceAssignmentID, CustomerAssignmentOrganizationID: attributionItem.OrganizationID, CustomerAssignedAt: attributionItem.AttributedAt,
			EmployeeID: employee.ID, EmployeeName: attributionItem.EmployeeName, PersonnelRole: result.PersonnelRole,
			CalculationBasis: result.CalculationBasis, RatePercent: rate, Fees: make([]*biz.CommissionFeeDetail, 0, len(orderFees)),
		}
		for _, feeItem := range orderFees {
			party, partyErr := feeItem.Edges.SettlementPartyOrErr()
			if partyErr != nil {
				return nil, partyErr
			}
			totalAmount, parseErr := decimalOf(feeItem.TotalAmount)
			if parseErr != nil {
				return nil, parseErr
			}
			exchangeRate, parseErr := decimalOf(feeItem.ExchangeRate)
			if parseErr != nil {
				return nil, parseErr
			}
			baseAmount, parseErr := decimalOf(feeItem.BaseCurrencyAmount)
			if parseErr != nil {
				return nil, parseErr
			}
			if result.BaseCurrency != feeItem.BaseCurrency {
				return nil, biz.ErrCommissionSource
			}
			line.BaseCurrency = result.BaseCurrency
			// 总应收/总应付分母优先使用关联账单行的本位币快照（账单日口径），
			// 未建账费用回落到费用自身快照，避免分子分母汇率基准混用。
			aggregateBase := baseAmount
			if billedBase, billed := source.billLineBaseByFee[feeItem.ID]; billed {
				aggregateBase = billedBase
			}
			if feeItem.Direction == fee.DirectionRECEIVABLE {
				line.RealizedRevenue = line.RealizedRevenue.Add(aggregateBase)
			} else {
				line.AllocatedCost = line.AllocatedCost.Add(aggregateBase)
			}
			line.Fees = append(line.Fees, &biz.CommissionFeeDetail{
				FeeID: feeItem.ID, SettlementPartyID: feeItem.SettlementPartyID, Direction: string(feeItem.Direction),
				FeeCode: feeItem.FeeCode, FeeName: feeItem.FeeName, SettlementPartyName: party.LegalName,
				Currency: feeItem.Currency, TotalAmount: totalAmount, ExchangeRate: exchangeRate,
				BaseCurrency: feeItem.BaseCurrency, BaseCurrencyAmount: baseAmount, ExpenseDate: feeItem.ExpenseDate, Status: string(feeItem.Status),
			})
			fingerprintParts = append(fingerprintParts, fmt.Sprintf("fee|%s|%s|%s|%s|%s|%s|%s|%s|%d", feeItem.ID, feeItem.OrderID, feeItem.Direction, feeItem.Status, feeItem.TotalAmount, feeItem.ExchangeRate, feeItem.BaseCurrencyAmount, feeItem.BaseCurrency, feeItem.Version))
		}
		line.FeeCount = len(line.Fees)
		totalReceivable := line.RealizedRevenue.Round(8)
		totalPayable := line.AllocatedCost.Round(8)
		realized := source.orderRealized[orderID].Round(8)
		cost, profit, commissionBase, amount, calculateErr := biz.CalculateCommissionLine(realized, totalReceivable, totalPayable, rate, result.CalculationBasis)
		if calculateErr != nil {
			return nil, calculateErr
		}
		line.RealizedRevenue, line.AllocatedCost, line.RealizedProfit = realized, cost, profit
		// 历史分母快照：新生成提成行固定写 READY + NATIVE，作为锁后费用补录
		// 审批的边际复算输入；分母为该行确认时冻结的历史总应收/总应付。
		line.TotalReceivableSnapshot, line.TotalPayableSnapshot = totalReceivable, totalPayable
		line.SnapshotStatus, line.SnapshotSource = biz.CommissionLineSnapshotReady, biz.CommissionLineSnapshotNative
		line.CommissionBaseAmount, line.CommissionAmount = commissionBase, amount
		result.Lines = append(result.Lines, line)
		customersWithFees[orderItem.CustomerID] = struct{}{}
		result.RealizedRevenue = result.RealizedRevenue.Add(realized)
		result.AllocatedCost = result.AllocatedCost.Add(cost)
		result.RealizedProfit = result.RealizedProfit.Add(profit)
		result.CommissionBaseAmount = result.CommissionBaseAmount.Add(commissionBase)
		result.CommissionAmount = result.CommissionAmount.Add(amount)
		result.FeeCount += line.FeeCount
		fingerprintParts = append(fingerprintParts, fmt.Sprintf("result_line|%s|%s|%s|%s|%s", orderItem.ID, realized.StringFixed(8), cost.StringFixed(8), profit.StringFixed(8), amount.StringFixed(8)))
	}
	if len(result.Lines) == 0 || !result.RealizedRevenue.IsPositive() {
		return nil, biz.ErrCommissionSource
	}
	result.CustomerCount, result.OrderCount = len(customersWithFees), len(result.Lines)
	result.RealizedRevenue = result.RealizedRevenue.Round(8)
	result.AllocatedCost = result.AllocatedCost.Round(8)
	result.RealizedProfit = result.RealizedProfit.Round(8)
	result.CommissionBaseAmount = result.CommissionBaseAmount.Round(8)
	result.CommissionAmount = result.CommissionAmount.Round(8)
	sort.Strings(fingerprintParts)
	digest := sha256.Sum256([]byte(strings.Join(fingerprintParts, "\n")))
	result.SourceFingerprint = hex.EncodeToString(digest[:])
	return result, nil
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// Create 在事务内锁定来源并写入提成草稿与 CNY 快照，只负责写入并返回错误；
// 完整业务响应由用例在共享事务提交后通过普通上下文重读。方案与员工分配在
// 事务内按来源归属日期重新解析，不信任预览结果。
func (r *commissionRepo) Create(ctx context.Context, org uuid.UUID, c *biz.FinanceCommission, snapshot *biz.CommissionCNYSnapshot, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		calculation, err := calculateCommission(ctx, commissionStoreFromTx(tx), org, c.VerificationID, c.NettingID, c.EmployeeID, c.PersonnelRole, true)
		if err != nil {
			return err
		}
		// 原始 CNY 提成金额依赖锁内计算出的提成金额，按 biz 纯函数固化到快照。
		snapshot.ApplyCommissionAmount(calculation.CommissionAmount)
		// 活跃去重按来源路由：同来源同员工同角色仅允许一条非终态提成。
		duplicatePredicates := []predicate.FinanceCommission{
			commission.OrganizationIDEQ(org),
			commission.EmployeeIDEQ(c.EmployeeID),
			commission.PersonnelRoleEQ(string(calculation.PersonnelRole)),
			commission.StatusNEQ(commission.StatusCANCELLED),
		}
		if c.VerificationID != uuid.Nil {
			duplicatePredicates = append(duplicatePredicates, commission.VerificationIDEQ(c.VerificationID))
		} else {
			duplicatePredicates = append(duplicatePredicates, commission.NettingIDEQ(c.NettingID))
		}
		hasActive, err := tx.FinanceCommission.Query().Where(duplicatePredicates...).Exist(ctx)
		if err != nil {
			return err
		}
		if hasActive {
			return biz.ErrCommissionDuplicate
		}
		c.VerificationNo, c.NettingNo, c.EmployeeName, c.RuleName = calculation.VerificationNo, calculation.NettingNo, calculation.EmployeeName, calculation.RuleName
		c.PersonnelRole, c.CalculationBasis = calculation.PersonnelRole, calculation.CalculationBasis
		c.RuleID, c.RuleVersion, c.CalculationVersion, c.SourceFingerprint = calculation.RuleID, calculation.RuleVersion, calculation.CalculationVersion, calculation.SourceFingerprint
		c.BaseCurrency, c.RatePercent = calculation.BaseCurrency, calculation.RatePercent
		c.CustomerCount, c.OrderCount, c.FeeCount = calculation.CustomerCount, calculation.OrderCount, calculation.FeeCount
		c.RealizedRevenue, c.AllocatedCost, c.RealizedProfit = calculation.RealizedRevenue, calculation.AllocatedCost, calculation.RealizedProfit
		c.CommissionBaseAmount, c.CommissionAmount = calculation.CommissionBaseAmount, calculation.CommissionAmount
		create := tx.FinanceCommission.Create().SetID(c.ID).SetOrganizationID(org).SetCommissionNo(c.CommissionNo).SetIdempotencyKey(c.IdempotencyKey).SetEmployeeID(c.EmployeeID).SetEmployeeName(c.EmployeeName).SetCustomerCount(c.CustomerCount).SetOrderCount(c.OrderCount).SetFeeCount(c.FeeCount).SetRuleID(c.RuleID).SetRuleName(c.RuleName).SetPersonnelRole(string(c.PersonnelRole)).SetCalculationBasis(string(c.CalculationBasis)).SetRuleVersion(c.RuleVersion).SetCalculationVersion(c.CalculationVersion).SetSourceFingerprint(c.SourceFingerprint).SetStatus(commission.StatusDRAFT).SetBaseCurrency(c.BaseCurrency).SetRealizedRevenue(c.RealizedRevenue.StringFixed(8)).SetAllocatedCost(c.AllocatedCost.StringFixed(8)).SetRealizedProfit(c.RealizedProfit.StringFixed(8)).SetCommissionBaseAmount(c.CommissionBaseAmount.StringFixed(8)).SetRatePercent(c.RatePercent.StringFixed(4)).SetCommissionAmount(c.CommissionAmount.StringFixed(8)).SetCommissionDate(snapshot.CommissionDate).SetCnyExchangeRate(snapshot.ExchangeRate.StringFixed(8)).SetCnyExchangeRateSource(commission.CnyExchangeRateSource(snapshot.ExchangeRateSource)).SetCnyExchangeRateDate(snapshot.ExchangeRateDate).SetNillableCnyExchangeRateSettingID(snapshot.ExchangeRateSettingID).SetCnyCommissionAmount(snapshot.CommissionAmount.StringFixed(8)).SetNillableNote(c.Note).SetVersion(1)
		// 来源二选一落库：空来源显式置 NULL，保证部分唯一索引语义正确。
		if c.VerificationID != uuid.Nil {
			create = create.SetVerificationID(c.VerificationID).SetVerificationNo(c.VerificationNo)
		}
		if c.NettingID != uuid.Nil {
			create = create.SetNettingID(c.NettingID).SetNettingNo(c.NettingNo)
		}
		if _, err = create.Save(ctx); err != nil {
			return mapEntError(err, nil, biz.ErrCommissionDuplicate)
		}
		lineBuilders := make([]*ent.FinanceCommissionLineCreate, 0, len(calculation.Lines))
		for _, line := range calculation.Lines {
			line.ID = uuid.Must(uuid.NewV7())
			line.OrganizationID = org
			line.CommissionID = c.ID
			feeSnapshot, marshalErr := json.Marshal(line.Fees)
			if marshalErr != nil {
				return marshalErr
			}
			lineBuilders = append(lineBuilders, tx.FinanceCommissionLine.Create().SetID(line.ID).SetOrganizationID(org).SetCommissionID(c.ID).SetOrderID(line.OrderID).SetOrderNo(line.OrderNo).SetOrderDate(line.OrderDate).SetCustomerID(line.CustomerID).SetCustomerCode(line.CustomerCode).SetCustomerName(line.CustomerName).SetPersonnelAssignmentID(line.CustomerAssignmentID).SetPersonnelOrganizationID(line.CustomerAssignmentOrganizationID).SetPersonnelAssignedAt(line.CustomerAssignedAt).SetFeeCount(line.FeeCount).SetFeeSnapshot(string(feeSnapshot)).SetEmployeeID(line.EmployeeID).SetEmployeeName(line.EmployeeName).SetPersonnelRole(string(line.PersonnelRole)).SetCalculationBasis(string(line.CalculationBasis)).SetBaseCurrency(line.BaseCurrency).SetRealizedRevenue(line.RealizedRevenue.StringFixed(8)).SetAllocatedCost(line.AllocatedCost.StringFixed(8)).SetRealizedProfit(line.RealizedProfit.StringFixed(8)).SetCommissionBaseAmount(line.CommissionBaseAmount.StringFixed(8)).SetRatePercent(line.RatePercent.StringFixed(4)).SetCommissionAmount(line.CommissionAmount.StringFixed(8)).SetTotalReceivableSnapshot(line.TotalReceivableSnapshot.StringFixed(8)).SetTotalPayableSnapshot(line.TotalPayableSnapshot.StringFixed(8)).SetSnapshotStatus(commissionline.SnapshotStatusREADY).SetSnapshotSource(commissionline.SnapshotSourceNATIVE))
		}
		if _, err = tx.FinanceCommissionLine.CreateBulk(lineBuilders...).Save(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

func (r *commissionRepo) Transition(ctx context.Context, org, id, actor uuid.UUID, version uint64, target biz.CommissionStatus, reason string, audit *biz.AuditEvent) (*biz.FinanceCommission, error) {
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if target == biz.CommissionConfirmed {
			// CONFIRMED 分支先按既有「来源 → 账单 → 订单 → 费用」锁序完成指纹
			// 复算，避免引入与提成创建路径相反的订单/账单加锁顺序。方案与员工
			// 分配按快照的员工、身份与归属日期重新解析：已生效方案的历史快照
			// 不被名单或方案后续变化重算，但解析失败或指纹漂移会拒绝确认。
			snapshot, lookupErr := tx.FinanceCommission.Query().Where(commission.IDEQ(id), commission.OrganizationIDEQ(org)).Only(ctx)
			if lookupErr != nil {
				return mapEntError(lookupErr, biz.ErrCommissionNotFound, nil)
			}
			current, calculateErr := calculateCommission(ctx, commissionStoreFromTx(tx), org, valueOrNilUUID(snapshot.VerificationID), valueOrNilUUID(snapshot.NettingID), snapshot.EmployeeID, derefCommissionPersonnelRole(snapshot.PersonnelRole), true)
			if calculateErr != nil {
				return biz.ErrCommissionSourceChanged
			}
			if snapshot.SourceFingerprint == "" || snapshot.SourceFingerprint != current.SourceFingerprint {
				return biz.ErrCommissionSourceChanged
			}
		}
		// 提成状态迁移会改变订单财务锁证据集合；在修改提成行之前统一按 UUID
		// 升序取得受影响 Order 行锁，保持 Order → 提成父单 → 调整的固定锁序，
		// 避免补录审批复核证据后被并发改写。
		preLines, preLineErr := tx.FinanceCommissionLine.Query().Where(commissionline.CommissionIDEQ(id)).All(ctx)
		if preLineErr != nil {
			return preLineErr
		}
		if lockErr := lockOrdersSortedForFinance(ctx, tx, orderUUIDsFromLines(preLines)); lockErr != nil {
			return lockErr
		}
		if target == biz.CommissionConfirmed {
			orderIDs, orderIDErr := tx.FinanceCommissionLine.Query().Where(commissionline.CommissionIDEQ(id)).All(ctx)
			if orderIDErr != nil {
				return orderIDErr
			}
			lineOrderIDs := orderUUIDsFromLines(orderIDs)
			hasDraftFees, draftErr := tx.OrderFee.Query().Where(fee.OrderIDIn(lineOrderIDs...), fee.StatusEQ(fee.StatusDRAFT)).Exist(ctx)
			if draftErr != nil {
				return draftErr
			}
			if hasDraftFees {
				return biz.ErrCommissionUnconfirmedFees
			}
		}
		x, err := tx.FinanceCommission.Query().Where(commission.IDEQ(id), commission.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionNotFound, nil)
		}
		if x.Version != version {
			return biz.ErrCommissionTransition
		}
		now := time.Now()
		update := tx.FinanceCommission.UpdateOneID(id).SetVersion(version + 1)
		switch target {
		case biz.CommissionConfirmed:
			if x.Status != commission.StatusDRAFT {
				return biz.ErrCommissionTransition
			}
			update.SetStatus(commission.StatusCONFIRMED).SetConfirmedAt(now).SetConfirmedBy(actor)
		case biz.CommissionPaid:
			if x.Status != commission.StatusCONFIRMED {
				return biz.ErrCommissionTransition
			}
			update.SetStatus(commission.StatusPAID).SetPaidAt(now).SetPaidBy(actor)
		case biz.CommissionCancelled:
			if x.Status != commission.StatusDRAFT && x.Status != commission.StatusCONFIRMED {
				return biz.ErrCommissionTransition
			}
			hasAdjustments, adjustmentErr := tx.FinanceCommissionAdjustment.Query().Where(adjustment.CommissionIDEQ(id), adjustment.StatusNEQ(adjustment.StatusCANCELLED)).Exist(ctx)
			if adjustmentErr != nil {
				return adjustmentErr
			}
			if hasAdjustments {
				return biz.ErrCommissionTransition
			}
			update.SetStatus(commission.StatusCANCELLED).SetCancelledAt(now).SetCancelledBy(actor).SetCancellationReason(reason)
		default:
			return biz.ErrCommissionInvalid
		}
		if _, err = update.Save(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return r.Get(ctx, org, id)
}

func valueOrNilUUID(value *uuid.UUID) uuid.UUID {
	if value == nil {
		return uuid.Nil
	}
	return *value
}

// derefCommissionPersonnelRole 读取提成快照中的人员身份；存量行身份为空时返回
// 非法值，由后续解析稳定拒绝。
func derefCommissionPersonnelRole(value *string) biz.CommissionPersonnelRole {
	if value == nil {
		return ""
	}
	return biz.CommissionPersonnelRole(*value)
}

// orderUUIDsFromLines 汇总提成行的订单 ID。
func orderUUIDsFromLines(lines []*ent.FinanceCommissionLine) []uuid.UUID {
	orderIDs := make([]uuid.UUID, 0, len(lines))
	for _, line := range lines {
		orderIDs = append(orderIDs, line.OrderID)
	}
	return orderIDs
}

// lockOrdersSortedForFinance 按 UUID 升序锁定受影响的订单行：所有会改变财务锁
// 净额或证据集合的提成/调整状态迁移统一保持 Order → 提成父单 → 调整的固定锁
// 序，防止补录审批复核证据后被并发改写。取得 Order 锁仅做线性化互斥。
func lockOrdersSortedForFinance(ctx context.Context, tx *ent.Tx, orderIDs []uuid.UUID) error {
	if len(orderIDs) == 0 {
		return nil
	}
	sorted := uniqueSortedUUIDs(orderIDs)
	_, err := tx.Order.Query().Where(orderent.IDIn(sorted...)).Order(ent.Asc(orderent.FieldID)).ForUpdate().All(ctx)
	return err
}

func (r *commissionRepo) Get(ctx context.Context, org, id uuid.UUID) (*biz.FinanceCommission, error) {
	return r.GetScoped(ctx, []uuid.UUID{org}, id)
}
func (r *commissionRepo) GetScoped(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*biz.FinanceCommission, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	x, err := client.FinanceCommission.Query().Where(commission.IDEQ(id), commission.OrganizationIDIn(organizationIDs...)).WithOrganization().WithLines(func(q *ent.FinanceCommissionLineQuery) {
		q.Order(commissionline.ByOrderNo(), commissionline.ByOrderID())
	}).WithAdjustments(func(q *ent.FinanceCommissionAdjustmentQuery) {
		q.Order(adjustment.ByCreatedAt())
	}).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrCommissionNotFound, nil)
	}
	return commissionWithLinesToBiz(x)
}

func commissionWithLinesToBiz(x *ent.FinanceCommission) (*biz.FinanceCommission, error) {
	result, err := commissionToBiz(x)
	if err != nil {
		return nil, err
	}
	result.Lines = make([]*biz.FinanceCommissionLine, 0, len(x.Edges.Lines))
	for _, line := range x.Edges.Lines {
		converted, convertErr := commissionLineToBiz(line)
		if convertErr != nil {
			return nil, convertErr
		}
		result.Lines = append(result.Lines, converted)
	}
	result.Adjustments = make([]*biz.FinanceCommissionAdjustment, 0, len(x.Edges.Adjustments))
	for _, item := range x.Edges.Adjustments {
		converted, convertErr := commissionAdjustmentToBiz(item)
		if convertErr != nil {
			return nil, convertErr
		}
		result.Adjustments = append(result.Adjustments, converted)
		if converted.Status == biz.CommissionConfirmed || converted.Status == biz.CommissionPaid {
			if converted.Direction == biz.CommissionAdjustmentDecrease {
				result.AdjustmentAmount = result.AdjustmentAmount.Sub(converted.Amount)
			} else {
				result.AdjustmentAmount = result.AdjustmentAmount.Add(converted.Amount)
			}
		}
	}
	result.AdjustmentAmount = result.AdjustmentAmount.Round(8)
	result.EffectiveCommissionAmount = result.CommissionAmount.Add(result.AdjustmentAmount).Round(8)
	// CNY 调整金额动态继承主单汇率快照折算，草稿与已取消调整不计入。
	result.CNYAdjustmentAmount = result.AdjustmentAmount.Mul(result.CNYExchangeRate).Round(8)
	result.CNYEffectiveCommissionAmount = result.CNYCommissionAmount.Add(result.CNYAdjustmentAmount).Round(8)
	return result, nil
}

func commissionToBiz(x *ent.FinanceCommission) (*biz.FinanceCommission, error) {
	revenue, err := decimalOf(x.RealizedRevenue)
	if err != nil {
		return nil, err
	}
	cost, err := decimalOf(x.AllocatedCost)
	if err != nil {
		return nil, err
	}
	profit, err := decimalOf(x.RealizedProfit)
	if err != nil {
		return nil, err
	}
	rate, err := decimalOf(x.RatePercent)
	if err != nil {
		return nil, err
	}
	amount, err := decimalOf(x.CommissionAmount)
	if err != nil {
		return nil, err
	}
	commissionBase, err := decimalOf(x.CommissionBaseAmount)
	if err != nil {
		return nil, err
	}
	cnyRate, err := decimalOf(x.CnyExchangeRate)
	if err != nil {
		return nil, err
	}
	cnyAmount, err := decimalOf(x.CnyCommissionAmount)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceCommission{ID: x.ID, OrganizationID: x.OrganizationID, CommissionNo: x.CommissionNo, IdempotencyKey: x.IdempotencyKey, EmployeeID: x.EmployeeID, EmployeeName: x.EmployeeName, CustomerCount: x.CustomerCount, OrderCount: x.OrderCount, FeeCount: x.FeeCount, Status: biz.CommissionStatus(x.Status), BaseCurrency: x.BaseCurrency, RealizedRevenue: revenue, AllocatedCost: cost, RealizedProfit: profit, CommissionBaseAmount: commissionBase, RatePercent: rate, CommissionAmount: amount, EffectiveCommissionAmount: amount, CommissionDate: x.CommissionDate, CNYExchangeRate: cnyRate, CNYExchangeRateSource: string(x.CnyExchangeRateSource), CNYExchangeRateDate: x.CnyExchangeRateDate, CNYExchangeRateSettingID: x.CnyExchangeRateSettingID, CNYCommissionAmount: cnyAmount, Note: x.Note, Version: x.Version, RuleVersion: x.RuleVersion, CalculationVersion: x.CalculationVersion, SourceFingerprint: x.SourceFingerprint, ConfirmedAt: x.ConfirmedAt, ConfirmedBy: x.ConfirmedBy, PaidAt: x.PaidAt, PaidBy: x.PaidBy, CancelledAt: x.CancelledAt, CancelledBy: x.CancelledBy, CancellationReason: x.CancellationReason, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt}
	if x.VerificationID != nil {
		result.VerificationID = *x.VerificationID
		result.VerificationNo = optionalStringValue(x.VerificationNo)
	}
	if x.NettingID != nil {
		result.NettingID = *x.NettingID
		result.NettingNo = optionalStringValue(x.NettingNo)
	}
	if x.Edges.Organization != nil {
		result.OrganizationName = x.Edges.Organization.Name
	}
	if x.RuleID != nil {
		result.RuleID = *x.RuleID
	}
	if x.RuleName != nil {
		result.RuleName = *x.RuleName
	}
	if x.PersonnelRole != nil {
		result.PersonnelRole = biz.CommissionPersonnelRole(*x.PersonnelRole)
	}
	if x.CalculationBasis != nil {
		result.CalculationBasis = biz.CommissionCalculationBasis(*x.CalculationBasis)
	}
	return result, nil
}

func commissionLineToBiz(x *ent.FinanceCommissionLine) (*biz.FinanceCommissionLine, error) {
	revenue, err := decimalOf(x.RealizedRevenue)
	if err != nil {
		return nil, err
	}
	cost, err := decimalOf(x.AllocatedCost)
	if err != nil {
		return nil, err
	}
	profit, err := decimalOf(x.RealizedProfit)
	if err != nil {
		return nil, err
	}
	rate, err := decimalOf(x.RatePercent)
	if err != nil {
		return nil, err
	}
	amount, err := decimalOf(x.CommissionAmount)
	if err != nil {
		return nil, err
	}
	commissionBase, err := decimalOf(x.CommissionBaseAmount)
	if err != nil {
		return nil, err
	}
	fees := make([]*biz.CommissionFeeDetail, 0, x.FeeCount)
	if err = json.Unmarshal([]byte(x.FeeSnapshot), &fees); err != nil {
		return nil, err
	}
	result := &biz.FinanceCommissionLine{ID: x.ID, OrganizationID: x.OrganizationID, CommissionID: x.CommissionID, OrderID: x.OrderID, OrderNo: x.OrderNo, OrderDate: x.OrderDate, CustomerID: x.CustomerID, CustomerCode: x.CustomerCode, CustomerName: x.CustomerName, CustomerAssignmentID: x.PersonnelAssignmentID, CustomerAssignmentOrganizationID: x.PersonnelOrganizationID, CustomerAssignedAt: x.PersonnelAssignedAt, EmployeeID: x.EmployeeID, EmployeeName: x.EmployeeName, PersonnelRole: biz.CommissionPersonnelRole(x.PersonnelRole), CalculationBasis: biz.CommissionCalculationBasis(x.CalculationBasis), BaseCurrency: x.BaseCurrency, RealizedRevenue: revenue, AllocatedCost: cost, RealizedProfit: profit, CommissionBaseAmount: commissionBase, RatePercent: rate, CommissionAmount: amount, FeeCount: x.FeeCount, Fees: fees, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt, SnapshotStatus: snapshotStatusOrEmpty(x.SnapshotStatus), SnapshotSource: snapshotSourceOrEmpty(x.SnapshotSource)}
	if x.TotalReceivableSnapshot != nil {
		receivable, parseErr := decimalOf(*x.TotalReceivableSnapshot)
		if parseErr != nil {
			return nil, parseErr
		}
		result.TotalReceivableSnapshot = receivable
	}
	if x.TotalPayableSnapshot != nil {
		payable, parseErr := decimalOf(*x.TotalPayableSnapshot)
		if parseErr != nil {
			return nil, parseErr
		}
		result.TotalPayableSnapshot = payable
	}
	return result, nil
}

// snapshotStatusOrEmpty / snapshotSourceOrEmpty 把可空快照枚举转换为领域字符串；
// 全空存量行为空串，调用方必须按「正向判定 == READY」处理，不得放行全空行。
func snapshotStatusOrEmpty(value *commissionline.SnapshotStatus) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func snapshotSourceOrEmpty(value *commissionline.SnapshotSource) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func (r *commissionRepo) GetAdjustmentByKey(ctx context.Context, org uuid.UUID, key string) (*biz.FinanceCommissionAdjustment, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	x, err := client.FinanceCommissionAdjustment.Query().WithOrganization().Where(adjustment.OrganizationIDEQ(org), adjustment.IdempotencyKeyEQ(key)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return commissionAdjustmentToBiz(x)
}

func (r *commissionRepo) GetAdjustmentScoped(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*biz.FinanceCommissionAdjustment, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	x, err := client.FinanceCommissionAdjustment.Query().Where(adjustment.IDEQ(id), adjustment.OrganizationIDIn(organizationIDs...)).WithOrganization().Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrCommissionAdjustmentNotFound, nil)
	}
	return commissionAdjustmentToBiz(x)
}

// commissionAdjustmentListPredicates 构造调整列表筛选谓词：关键字覆盖订单号、
// 提成号与调整号；状态、来源与员工过滤显式命中。
func commissionAdjustmentListPredicates(organizationIDs []uuid.UUID, f biz.CommissionAdjustmentFilter) []predicate.FinanceCommissionAdjustment {
	p := []predicate.FinanceCommissionAdjustment{adjustment.OrganizationIDIn(organizationIDs...)}
	if f.Keyword != "" {
		p = append(p, adjustment.Or(
			adjustment.OrderNoContainsFold(f.Keyword),
			adjustment.CommissionNoContainsFold(f.Keyword),
			adjustment.AdjustmentNoContainsFold(f.Keyword),
		))
	}
	if f.Status != "" {
		p = append(p, adjustment.StatusEQ(adjustment.Status(f.Status)))
	}
	if f.SourceType != "" {
		p = append(p, adjustment.SourceTypeEQ(adjustment.SourceType(f.SourceType)))
	}
	if f.EmployeeID != uuid.Nil {
		p = append(p, adjustment.EmployeeIDEQ(f.EmployeeID))
	}
	return p
}

// ListAdjustmentsScoped 服务端分页读取提成调整，默认 created_at 倒序、主键倒序
// 兜底稳定排序；组织范围显式传入，由调用方按 commission.read 权限解析。
func (r *commissionRepo) ListAdjustmentsScoped(ctx context.Context, organizationIDs []uuid.UUID, f biz.CommissionAdjustmentFilter) (*biz.CommissionAdjustmentListResult, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	q := client.FinanceCommissionAdjustment.Query().Where(commissionAdjustmentListPredicates(organizationIDs, f)...)
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	xs, err := q.WithOrganization().
		Order(adjustment.ByCreatedAt(entsql.OrderDesc()), adjustment.ByID(entsql.OrderDesc())).
		Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.CommissionAdjustmentListResult{Items: make([]*biz.FinanceCommissionAdjustment, 0, len(xs)), Total: int64(total), Page: f.Page, PageSize: f.PageSize}
	for _, x := range xs {
		converted, convertErr := commissionAdjustmentToBiz(x)
		if convertErr != nil {
			return nil, convertErr
		}
		result.Items = append(result.Items, converted)
	}
	return result, nil
}

// GetMyFeeSupplementAdjustmentSource 员工本人专属冲减来源最小详情。查询谓词同时
// 限定调整 ID、employee_id = 当前用户、LOCKED_FEE_SUPPLEMENT 来源，以及「调整
// 所属组织启用且当前用户存在启用成员关系」；任一不满足统一映射为调整不存在，
// 不泄露他人调整的记录事实。
func (r *commissionRepo) GetMyFeeSupplementAdjustmentSource(ctx context.Context, userID, id uuid.UUID) (*biz.MyFeeSupplementAdjustmentSource, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	x, err := client.FinanceCommissionAdjustment.Query().Where(
		adjustment.IDEQ(id),
		adjustment.EmployeeIDEQ(userID),
		adjustment.SourceTypeEQ(adjustment.SourceTypeLOCKED_FEE_SUPPLEMENT),
		adjustment.HasOrganizationWith(
			organizationent.EnabledEQ(true),
			organizationent.HasMembershipsWith(membership.UserIDEQ(userID), membership.EnabledEQ(true)),
		),
	).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrCommissionAdjustmentNotFound, nil)
	}
	requestID := uuid.Nil
	if x.SourceFeeSupplementRequestID != nil {
		requestID = *x.SourceFeeSupplementRequestID
	}
	if requestID == uuid.Nil {
		return nil, biz.ErrCommissionAdjustmentNotFound
	}
	request, requestErr := client.OrderFeeSupplementRequest.Query().
		Where(orderfeesupplementent.IDEQ(requestID)).
		Only(ctx)
	if requestErr != nil {
		return nil, biz.ErrCommissionAdjustmentNotFound
	}
	amount, parseErr := decimalOf(x.Amount)
	if parseErr != nil {
		return nil, parseErr
	}
	feeTotal, parseErr := decimalOf(request.TotalAmount)
	if parseErr != nil {
		return nil, parseErr
	}
	feeBase, parseErr := decimalOf(request.BaseCurrencyAmount)
	if parseErr != nil {
		return nil, parseErr
	}
	return &biz.MyFeeSupplementAdjustmentSource{
		AdjustmentID: x.ID, AdjustmentNo: x.AdjustmentNo, OrderNo: x.OrderNo, CommissionNo: x.CommissionNo,
		Status: biz.CommissionStatus(x.Status), SuggestedAmount: amount, BaseCurrency: x.BaseCurrency, CreatedAt: x.CreatedAt,
		FeeCode: request.FeeCode, FeeName: request.FeeName, FeeCurrency: request.Currency, FeeTotalAmount: feeTotal,
		FeeBaseCurrency: request.BaseCurrency, FeeBaseCurrencyAmount: feeBase, FeeExpenseDate: request.ExpenseDate,
		SupplementReason: request.Reason,
	}, nil
}

func (r *commissionRepo) CreateAdjustment(ctx context.Context, org, actor uuid.UUID, item *biz.FinanceCommissionAdjustment, audit *biz.AuditEvent) (*biz.FinanceCommissionAdjustment, error) {
	var created *ent.FinanceCommissionAdjustment
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		parent, err := tx.FinanceCommission.Query().Where(commission.IDEQ(item.CommissionID), commission.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionNotFound, nil)
		}
		if parent.Status != commission.StatusCONFIRMED && parent.Status != commission.StatusPAID {
			return biz.ErrCommissionAdjustmentTransition
		}
		line, err := tx.FinanceCommissionLine.Query().Where(commissionline.CommissionIDEQ(parent.ID), commissionline.OrderIDEQ(item.OrderID)).Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionAdjustmentInvalid, nil)
		}
		sequence := parent.AdjustmentSequence + 1
		item.AdjustmentNo = fmt.Sprintf("%s-ADJ%03d", parent.CommissionNo, sequence)
		item.CommissionNo, item.OrderNo = parent.CommissionNo, line.OrderNo
		item.EmployeeID, item.EmployeeName = parent.EmployeeID, parent.EmployeeName
		item.BaseCurrency = parent.BaseCurrency
		created, err = tx.FinanceCommissionAdjustment.Create().
			SetID(item.ID).SetOrganizationID(org).SetCommissionID(parent.ID).SetOrderID(line.OrderID).
			SetAdjustmentNo(item.AdjustmentNo).SetIdempotencyKey(item.IdempotencyKey).
			SetCommissionNo(parent.CommissionNo).SetOrderNo(line.OrderNo).
			SetEmployeeID(parent.EmployeeID).SetEmployeeName(parent.EmployeeName).
			SetSourceType(adjustment.SourceType(item.SourceType)).SetDirection(adjustment.Direction(item.Direction)).SetStatus(adjustment.StatusDRAFT).
			SetBaseCurrency(parent.BaseCurrency).SetAmount(item.Amount.StringFixed(8)).SetReason(item.Reason).
			SetNillableNote(item.Note).SetVersion(1).Save(ctx)
		if err != nil {
			return mapEntError(err, nil, biz.ErrCommissionAdjustmentInvalid)
		}
		if _, err = tx.FinanceCommission.UpdateOne(parent).SetAdjustmentSequence(sequence).Save(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return commissionAdjustmentToBiz(created)
}

func (r *commissionRepo) TransitionAdjustment(ctx context.Context, org, id, actor uuid.UUID, version uint64, target biz.CommissionStatus, reason string, audit *biz.AuditEvent) (*biz.FinanceCommissionAdjustment, error) {
	var updated *ent.FinanceCommissionAdjustment
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 调整状态迁移会改变订单财务锁证据集合与双层余额；先只读定位目标调整，
		// 再按 Order → 提成父单 → 调整固定锁序取得行锁，避免补录审批复核证据后
		// 被并发改写。
		pre, preErr := tx.FinanceCommissionAdjustment.Query().Where(adjustment.IDEQ(id), adjustment.OrganizationIDEQ(org)).Only(ctx)
		if preErr != nil {
			return mapEntError(preErr, biz.ErrCommissionAdjustmentNotFound, nil)
		}
		if lockErr := lockOrdersSortedForFinance(ctx, tx, []uuid.UUID{pre.OrderID}); lockErr != nil {
			return lockErr
		}
		parent, err := tx.FinanceCommission.Query().Where(commission.IDEQ(pre.CommissionID), commission.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionNotFound, nil)
		}
		x, err := tx.FinanceCommissionAdjustment.Query().Where(adjustment.IDEQ(id), adjustment.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionAdjustmentNotFound, nil)
		}
		if x.Version != version {
			return biz.ErrCommissionAdjustmentTransition
		}
		now := time.Now().UTC()
		update := tx.FinanceCommissionAdjustment.UpdateOne(x).SetVersion(version + 1)
		switch target {
		case biz.CommissionConfirmed:
			if x.Status != adjustment.StatusDRAFT || (parent.Status != commission.StatusCONFIRMED && parent.Status != commission.StatusPAID) {
				return biz.ErrCommissionAdjustmentTransition
			}
			if x.Direction == adjustment.DirectionDECREASE {
				currentAmount, parseErr := decimalOf(x.Amount)
				if parseErr != nil {
					return parseErr
				}
				// 订单行层有效余额：以 FinanceCommissionLine.commission_amount 为起点，
				// 只汇总同 commission_id + order_id 的 CONFIRMED/PAID 符号化调整
				//（不含其他 DRAFT）。确认在锁内重算，不静默缩小建议金额；
				// 同父单其他订单行的正余额不得替当前订单行兜底。
				line, lineErr := tx.FinanceCommissionLine.Query().
					Where(commissionline.CommissionIDEQ(parent.ID), commissionline.OrderIDEQ(x.OrderID)).
					Only(ctx)
				if lineErr != nil {
					return mapEntError(lineErr, biz.ErrCommissionAdjustmentInvalid, nil)
				}
				lineEffective, parseErr := decimalOf(line.CommissionAmount)
				if parseErr != nil {
					return parseErr
				}
				lineAdjustments, queryErr := tx.FinanceCommissionAdjustment.Query().Where(
					adjustment.CommissionIDEQ(parent.ID),
					adjustment.OrderIDEQ(x.OrderID),
					adjustment.IDNEQ(x.ID),
					adjustment.StatusIn(adjustment.StatusCONFIRMED, adjustment.StatusPAID),
				).All(ctx)
				if queryErr != nil {
					return queryErr
				}
				for _, old := range lineAdjustments {
					amount, amountErr := decimalOf(old.Amount)
					if amountErr != nil {
						return amountErr
					}
					if old.Direction == adjustment.DirectionDECREASE {
						lineEffective = lineEffective.Sub(amount)
					} else {
						lineEffective = lineEffective.Add(amount)
					}
				}
				if lineEffective.Sub(currentAmount).IsNegative() {
					return biz.ErrCommissionAdjustmentExceeds
				}
				// 父单层有效余额：以 FinanceCommission.commission_amount 为起点，
				// 汇总父单全部 CONFIRMED/PAID 符号化调整（不含其他 DRAFT）。
				active, queryErr := tx.FinanceCommissionAdjustment.Query().Where(
					adjustment.CommissionIDEQ(parent.ID), adjustment.IDNEQ(x.ID),
					adjustment.StatusIn(adjustment.StatusCONFIRMED, adjustment.StatusPAID),
				).All(ctx)
				if queryErr != nil {
					return queryErr
				}
				effective, parseErr := decimalOf(parent.CommissionAmount)
				if parseErr != nil {
					return parseErr
				}
				for _, old := range active {
					amount, amountErr := decimalOf(old.Amount)
					if amountErr != nil {
						return amountErr
					}
					if old.Direction == adjustment.DirectionDECREASE {
						effective = effective.Sub(amount)
					} else {
						effective = effective.Add(amount)
					}
				}
				if effective.Sub(currentAmount).IsNegative() {
					return biz.ErrCommissionAdjustmentExceeds
				}
			}
			update.SetStatus(adjustment.StatusCONFIRMED).SetConfirmedAt(now).SetConfirmedBy(actor)
		case biz.CommissionPaid:
			if x.Status != adjustment.StatusCONFIRMED {
				return biz.ErrCommissionAdjustmentTransition
			}
			update.SetStatus(adjustment.StatusPAID).SetPaidAt(now).SetPaidBy(actor)
		case biz.CommissionCancelled:
			if x.SourceType == adjustment.SourceTypeLOCKED_FEE_SUPPLEMENT {
				// 来源专属取消门禁：锁后费用补录冲减建议只有 DRAFT 可经通用取消
				// 接口忽略；CONFIRMED/PAID 即使绕过页面直接调用也稳定拒绝，
				// 防止经通用取消把已确认冲减洗白为已取消。
				if x.Status != adjustment.StatusDRAFT {
					return biz.ErrCommissionAdjustmentCancelNotAllowed
				}
			} else if x.Status != adjustment.StatusDRAFT && x.Status != adjustment.StatusCONFIRMED {
				return biz.ErrCommissionAdjustmentTransition
			}
			update.SetStatus(adjustment.StatusCANCELLED).SetCancelledAt(now).SetCancelledBy(actor).SetCancellationReason(reason)
		default:
			return biz.ErrCommissionAdjustmentInvalid
		}
		updated, err = update.Save(ctx)
		if err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return commissionAdjustmentToBiz(updated)
}

func commissionAdjustmentToBiz(x *ent.FinanceCommissionAdjustment) (*biz.FinanceCommissionAdjustment, error) {
	amount, err := decimalOf(x.Amount)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceCommissionAdjustment{
		ID: x.ID, OrganizationID: x.OrganizationID, CommissionID: x.CommissionID, OrderID: x.OrderID,
		AdjustmentNo: x.AdjustmentNo, IdempotencyKey: x.IdempotencyKey, CommissionNo: x.CommissionNo,
		OrderNo: x.OrderNo, EmployeeID: x.EmployeeID, EmployeeName: x.EmployeeName,
		Direction: biz.CommissionAdjustmentDirection(x.Direction), SourceType: biz.CommissionAdjustmentSourceType(x.SourceType), SourceVerificationID: x.SourceVerificationID, Status: biz.CommissionStatus(x.Status),
		BaseCurrency: x.BaseCurrency, Amount: amount, Reason: x.Reason, Note: x.Note, Version: x.Version,
		ConfirmedAt: x.ConfirmedAt, ConfirmedBy: x.ConfirmedBy, PaidAt: x.PaidAt, PaidBy: x.PaidBy, CancelledAt: x.CancelledAt, CancelledBy: x.CancelledBy,
		CancellationReason: x.CancellationReason, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt,
	}
	if x.Edges.Organization != nil {
		result.OrganizationName = x.Edges.Organization.Name
	}
	return result, nil
}

func commissionRuleToBiz(x *ent.FinanceCommissionRule) (*biz.FinanceCommissionRule, error) {
	rate, err := decimalOf(x.RatePercent)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceCommissionRule{ID: x.ID, OrganizationID: x.OrganizationID, Name: x.Name, PersonnelRole: biz.CommissionPersonnelRole(x.PersonnelRole), CalculationBasis: biz.CommissionCalculationBasis(x.CalculationBasis), RatePercent: rate, EffectiveFrom: x.EffectiveFrom, EffectiveTo: x.EffectiveTo, Enabled: x.Enabled, LegacyReadOnly: x.LegacyReadonly, Note: x.Note, Version: x.Version, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt, Assignments: make([]*biz.FinanceCommissionRuleAssignment, 0, len(x.Edges.Assignments))}
	if x.Edges.Organization != nil {
		result.OrganizationName = x.Edges.Organization.Name
	}
	for _, item := range x.Edges.Assignments {
		result.Assignments = append(result.Assignments, commissionRuleAssignmentToBiz(item))
	}
	return result, nil
}

func commissionRuleAssignmentToBiz(x *ent.FinanceCommissionRuleAssignment) *biz.FinanceCommissionRuleAssignment {
	result := &biz.FinanceCommissionRuleAssignment{
		ID: x.ID, OrganizationID: x.OrganizationID, RuleID: x.RuleID, EmployeeID: x.EmployeeID,
		EffectiveFrom: x.EffectiveFrom, EffectiveTo: x.EffectiveTo,
		CancelledAt: x.CancelledAt, CancelledBy: x.CancelledBy,
		TerminatedAt: x.TerminatedAt, TerminatedBy: x.TerminatedBy,
		CreatedBy: x.CreatedBy, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt,
	}
	if x.Edges.Employee != nil {
		result.EmployeeName = x.Edges.Employee.DisplayName
	}
	return result
}

// activeCommissionAssignmentRuleNames 返回员工在组织内当前或未来仍有效的方案
// 分配所属方案名（已启用方案、未取消分配、实际区间未结束），供成员停用路径
// 阻断并提示需先以当天或未来离开日期处理的方案。
func activeCommissionAssignmentRuleNames(ctx context.Context, tx *ent.Tx, organizationID, userID uuid.UUID, today string) ([]string, error) {
	rows, err := tx.FinanceCommissionRuleAssignment.Query().
		Where(
			assignment.OrganizationIDEQ(organizationID),
			assignment.EmployeeIDEQ(userID),
			assignment.CancelledAtIsNil(),
			assignment.HasRuleWith(rule.EnabledEQ(true)),
		).
		WithRule().
		All(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		ruleItem := row.Edges.Rule
		if ruleItem == nil {
			continue
		}
		// 管理员已在方案管理中以离开日期终止的分配（含未来离开日）视为已处理，
		// 不再阻断停用；未终止且实际区间未结束的分配必须先处理。
		if row.TerminatedAt != nil {
			continue
		}
		// 实际有效区间 = 方案区间 ∩ 分配区间；终点为空视为正无穷。
		actualFrom := row.EffectiveFrom
		if ruleItem.EffectiveFrom != nil && *ruleItem.EffectiveFrom > actualFrom {
			actualFrom = *ruleItem.EffectiveFrom
		}
		actualTo := row.EffectiveTo
		if ruleItem.EffectiveTo != nil && (actualTo == nil || *ruleItem.EffectiveTo < *actualTo) {
			actualTo = ruleItem.EffectiveTo
		}
		if actualTo != nil && (*actualTo < actualFrom || *actualTo < today) {
			continue
		}
		if _, exists := seen[ruleItem.Name]; exists {
			continue
		}
		seen[ruleItem.Name] = struct{}{}
		names = append(names, ruleItem.Name)
	}
	sort.Strings(names)
	return names, nil
}

var _ biz.CommissionRepo = (*commissionRepo)(nil)
