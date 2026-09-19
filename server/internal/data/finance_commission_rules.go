package data

import (
	"context"
	"sort"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	rule "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	assignment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionruleassignment"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

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
