package data

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	applicationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionapplication"
	applicationline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionapplicationline"
	commissionline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	attribution "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	user "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

// Submit 提交本人月度申请（仅用于新建）。事务内顺序：
//  1. 员工 Membership 行 FOR UPDATE（稳定父行锁，与方案写路径同锁序）串行化
//     同一员工的重复点击与并发提交；
//  2. 按月度唯一键检查申请头：当月已存在任何状态的申请头返回稳定冲突并提示
//     到申请详情操作——重提必须走 Resubmit 显式定位原申请，空提交不承担
//     隐式重提语义；
//  3. 不存在时全量重算截止日以前的候选（来源固定锁序），逐候选复用或受控创建
//     DRAFT 提成并固化明细快照与头表汇总；空候选拒绝提交；月度唯一索引兜底
//     并发提交。
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
		exists, queryErr := client.FinanceCommissionApplication.Query().Where(
			applicationent.OrganizationIDEQ(scope.OrganizationID),
			applicationent.EmployeeIDEQ(scope.UserID),
			applicationent.ApplicationMonthEQ(applicationMonth),
		).Exist(txCtx)
		if queryErr != nil {
			return queryErr
		}
		if exists {
			return biz.ErrCommissionApplicationConflict
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

// applicationResubmitOutcome 是显式重提逐笔复核的结果：kept 为保留（含刷新）
// 的明细，removed 为按上游当前事实剔除的明细；refreshed 记录快照被刷新的笔数。
type applicationResubmitOutcome struct {
	kept         []*ent.FinanceCommissionApplicationLine
	removedLines []*ent.FinanceCommissionApplicationLine
	refreshed    int
}

// Resubmit 显式重提本人被驳回的月度申请（design §5.3）。事务内顺序：
//  1. 员工 Membership 行 FOR UPDATE 串行化同一员工的重提与提交竞争；
//  2. 按申请 ID + 组织 + 本人锁定申请头：他人/跨组织申请按不存在处理；
//     非 REJECTED 或 expected_version 不匹配返回「已被处理」稳定冲突；
//  3. 按 commission_id 升序锁定原绑定明细，按来源升序逐笔按上游当前事实重新
//     解析（来源 → 账单 → 订单 → 费用 → 归属 → 提成父单 → 方案固定锁序）：
//     来源仍有效且指纹一致保持不变；指纹变化刷新提成父单与明细快照金额/方案/
//     指纹；来源失效或提成已被取消/确认等失效场景把该明细从本次提交版本剔除；
//  4. 全部明细被剔除时稳定拒绝并整体回滚，申请保持 REJECTED 与原明细集合；
//  5. 成功后剔除失效明细行、重算笔数/合计，状态回 PENDING_REVIEW、版本递增，
//     保留最近一次决策审计（驳回原因对员工持续可见），写 RESUBMIT 审计留痕。
func (r *financeCommissionApplicationRepo) Resubmit(ctx context.Context, scope biz.WorkbenchScope, id uuid.UUID, expectedVersion uint64) (*biz.FinanceCommissionApplication, error) {
	var result *biz.FinanceCommissionApplication
	err := r.data.WithinTransaction(ctx, func(txCtx context.Context) error {
		client, clientErr := r.data.client(txCtx)
		if clientErr != nil {
			return clientErr
		}
		if lockErr := lockApplicationMembership(txCtx, client, scope.OrganizationID, scope.UserID); lockErr != nil {
			return lockErr
		}
		header, err := client.FinanceCommissionApplication.Query().Where(
			applicationent.IDEQ(id),
			applicationent.OrganizationIDEQ(scope.OrganizationID),
			applicationent.EmployeeIDEQ(scope.UserID),
		).ForUpdate().Only(txCtx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionApplicationNotFound, nil)
		}
		if header.Status != applicationent.StatusREJECTED || header.Version != expectedVersion {
			return biz.ErrCommissionApplicationStatusConflict
		}
		lines, err := client.FinanceCommissionApplicationLine.Query().
			Where(applicationline.ApplicationIDEQ(header.ID)).
			Order(applicationline.ByCommissionID()).
			ForUpdate().All(txCtx)
		if err != nil {
			return err
		}
		if len(lines) == 0 {
			return biz.ErrCommissionApplicationSourceConflict
		}
		store := commissionStoreFromClient(client)
		employee, err := store.users.Query().Where(user.IDEQ(scope.UserID)).ForUpdate().Only(txCtx)
		if err != nil {
			return err
		}
		outcome, refreshErr := r.refreshApplicationLines(txCtx, client, store, scope, employee, header, lines)
		if refreshErr != nil {
			return refreshErr
		}
		if len(outcome.kept) == 0 {
			// 全部明细失效：整体回滚，申请保持 REJECTED 与原明细集合。
			return biz.ErrCommissionApplicationNoValidLines
		}
		if len(outcome.removedLines) > 0 {
			removedIDs := make([]uuid.UUID, 0, len(outcome.removedLines))
			for _, line := range outcome.removedLines {
				removedIDs = append(removedIDs, line.ID)
			}
			if _, delErr := client.FinanceCommissionApplicationLine.Delete().
				Where(applicationline.IDIn(removedIDs...)).Exec(txCtx); delErr != nil {
				return delErr
			}
		}
		// 剔除完成后重读剩余明细，重算笔数与本位币/CNY 合计。
		remaining, err := client.FinanceCommissionApplicationLine.Query().
			Where(applicationline.ApplicationIDEQ(header.ID)).
			Order(applicationline.ByCommissionID()).All(txCtx)
		if err != nil {
			return err
		}
		total, totalCNY := decimal.Zero, decimal.Zero
		for _, line := range remaining {
			amount, amountErr := decimalOf(line.CommissionAmount)
			if amountErr != nil {
				return amountErr
			}
			cnyAmount, cnyErr := decimalOf(line.CnyCommissionAmount)
			if cnyErr != nil {
				return cnyErr
			}
			total = total.Add(amount).Round(8)
			totalCNY = totalCNY.Add(cnyAmount).Round(8)
		}
		now := time.Now()
		updated, err := client.FinanceCommissionApplication.UpdateOneID(header.ID).
			SetStatus(applicationent.StatusPENDING_REVIEW).
			SetVersion(header.Version + 1).
			SetCommissionCount(len(remaining)).
			SetTotalCommissionAmount(total.StringFixed(8)).
			SetTotalCnyCommissionAmount(totalCNY.StringFixed(8)).
			SetUpdatedAt(now).
			Save(txCtx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionApplicationNotFound, biz.ErrCommissionApplicationStatusConflict)
		}
		// 决策审计字段不清理：员工重提后仍能看到最近一次驳回原因。
		removedCommissionIDs := make([]string, 0, len(outcome.removedLines))
		for _, line := range outcome.removedLines {
			removedCommissionIDs = append(removedCommissionIDs, line.CommissionID.String())
		}
		audit := biz.CommissionApplicationResubmitAudit(scope.OrganizationID, scope.UserID, header.ID,
			header.ApplicationMonth, header.CoverageTo, len(remaining), outcome.refreshed,
			len(outcome.removedLines), total, removedCommissionIDs)
		if err := writeAudit(txCtx, client.AuditLog, audit); err != nil {
			return err
		}
		result = financeCommissionApplicationToBiz(updated)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// refreshApplicationLines 显式重提的逐笔刷新：按来源升序处理原绑定明细（与
// 提交路径同一锁序与解析引擎），指纹一致的保持不变，指纹变化的刷新提成父单与
// 明细快照，来源失效或提成不再是 DRAFT 的明细剔除。
func (r *financeCommissionApplicationRepo) refreshApplicationLines(ctx context.Context, client *ent.Client, store commissionCalculationStore, scope biz.WorkbenchScope, employee *ent.User, header *ent.FinanceCommissionApplication, lines []*ent.FinanceCommissionApplicationLine) (*applicationResubmitOutcome, error) {
	groups := groupApplicationLinesBySource(lines)
	outcome := &applicationResubmitOutcome{kept: make([]*ent.FinanceCommissionApplicationLine, 0, len(lines))}
	for _, group := range groups {
		source, err := loadCommissionCalculationSource(ctx, store, scope.OrganizationID, group.verificationID, group.nettingID, true)
		if err != nil {
			if isSkippableCommissionError(err) {
				// 来源已失效（核销作废/对冲未确认/无有效分摊）：该来源下全部明细剔除。
				outcome.removedLines = append(outcome.removedLines, group.lines...)
				continue
			}
			return nil, err
		}
		attributions, err := store.attributions.Query().Where(
			attribution.OrganizationIDEQ(scope.OrganizationID),
			attribution.OrderIDIn(source.orderIDs...),
			attribution.EmployeeIDEQ(scope.UserID),
		).Order(attribution.ByAttributedAt(), attribution.ByID()).ForUpdate().All(ctx)
		if err != nil {
			return nil, err
		}
		for _, line := range group.lines {
			role := biz.CommissionPersonnelRole(line.PersonnelRole)
			roleAttributions := make([]*ent.OrderCommissionAttribution, 0, len(attributions))
			for _, item := range attributions {
				if biz.CommissionPersonnelRole(item.PersonnelRole) == role {
					roleAttributions = append(roleAttributions, item)
				}
			}
			// 与提交路径一致：先锁提成父单再解析方案与归属。
			parent, err := client.FinanceCommission.Query().
				Where(commission.IDEQ(line.CommissionID), commission.OrganizationIDEQ(scope.OrganizationID)).
				ForUpdate().Only(ctx)
			if err != nil {
				return nil, mapEntError(err, biz.ErrCommissionApplicationSourceConflict, nil)
			}
			if parent.Status != commission.StatusDRAFT {
				// 提成已被取消/确认等其他路径处理：明细剔除，按既有模型流转。
				outcome.removedLines = append(outcome.removedLines, line)
				continue
			}
			calculation, err := resolveFreshCommissionCalculation(ctx, store, scope, employee, source, role, roleAttributions, true)
			if err != nil {
				if isSkippableCommissionError(err) {
					outcome.removedLines = append(outcome.removedLines, line)
					continue
				}
				return nil, err
			}
			if calculation.SourceFingerprint != "" && parent.SourceFingerprint == calculation.SourceFingerprint && line.SourceFingerprint == calculation.SourceFingerprint {
				outcome.kept = append(outcome.kept, line)
				continue
			}
			if refreshErr := r.refreshCommissionSnapshot(ctx, client, scope, header, parent, line, calculation, source.commissionDate); refreshErr != nil {
				return nil, refreshErr
			}
			outcome.refreshed++
			outcome.kept = append(outcome.kept, line)
		}
	}
	return outcome, nil
}

// applicationLineSourceGroup 是同一来源（核销或对冲）下的申请明细分组：处理前
// 按来源 ID 升序排列满足固定锁序。
type applicationLineSourceGroup struct {
	verificationID, nettingID uuid.UUID
	lines                     []*ent.FinanceCommissionApplicationLine
}

func applicationLineSourceID(line *ent.FinanceCommissionApplicationLine) uuid.UUID {
	if line.VerificationID != nil {
		return *line.VerificationID
	}
	if line.NettingID != nil {
		return *line.NettingID
	}
	return uuid.Nil
}

// groupApplicationLinesBySource 把申请明细按来源分组并按来源 ID 升序返回。
func groupApplicationLinesBySource(lines []*ent.FinanceCommissionApplicationLine) []*applicationLineSourceGroup {
	groupBySource := make(map[uuid.UUID]*applicationLineSourceGroup)
	for _, line := range lines {
		sourceID := applicationLineSourceID(line)
		group, exists := groupBySource[sourceID]
		if !exists {
			if line.VerificationID != nil {
				group = &applicationLineSourceGroup{verificationID: *line.VerificationID}
			} else {
				group = &applicationLineSourceGroup{nettingID: *line.NettingID}
			}
			groupBySource[sourceID] = group
		}
		group.lines = append(group.lines, line)
	}
	sourceIDs := make([]uuid.UUID, 0, len(groupBySource))
	for sourceID := range groupBySource {
		sourceIDs = append(sourceIDs, sourceID)
	}
	sort.Slice(sourceIDs, func(i, j int) bool { return sourceIDs[i].String() < sourceIDs[j].String() })
	groups := make([]*applicationLineSourceGroup, 0, len(sourceIDs))
	for _, sourceID := range sourceIDs {
		groups = append(groups, groupBySource[sourceID])
	}
	return groups
}

// refreshCommissionSnapshot 按重算结果整体刷新提成父单快照与提成行，并同步
// 申请明细快照（design §5.3 重提刷新语义）。仅用于 DRAFT 提成的申请重提受控
// 路径：版本递增并写提成刷新审计（记录金额变化，历史金额由审计留痕）。
func (r *financeCommissionApplicationRepo) refreshCommissionSnapshot(ctx context.Context, client *ent.Client, scope biz.WorkbenchScope, header *ent.FinanceCommissionApplication, parent *ent.FinanceCommission, line *ent.FinanceCommissionApplicationLine, calculation *biz.CommissionCalculation, commissionDate string) error {
	snapshot, err := biz.ResolveCommissionCNYRate(calculation.BaseCurrency, commissionDate, calculation.CommissionAmount)
	if err != nil {
		return err
	}
	update := client.FinanceCommission.UpdateOneID(parent.ID).
		SetEmployeeName(calculation.EmployeeName).
		SetCustomerCount(calculation.CustomerCount).
		SetOrderCount(calculation.OrderCount).
		SetFeeCount(calculation.FeeCount).
		SetRuleID(calculation.RuleID).
		SetNillableRuleName(stringPointerOrNil(calculation.RuleName)).
		SetNillableCalculationBasis(stringPointerOrNil(string(calculation.CalculationBasis))).
		SetRuleVersion(calculation.RuleVersion).
		SetCalculationVersion(calculation.CalculationVersion).
		SetSourceFingerprint(calculation.SourceFingerprint).
		SetRealizedRevenue(calculation.RealizedRevenue.StringFixed(8)).
		SetAllocatedCost(calculation.AllocatedCost.StringFixed(8)).
		SetRealizedProfit(calculation.RealizedProfit.StringFixed(8)).
		SetCommissionBaseAmount(calculation.CommissionBaseAmount.StringFixed(8)).
		SetRatePercent(calculation.RatePercent.StringFixed(4)).
		SetCommissionAmount(calculation.CommissionAmount.StringFixed(8)).
		SetCommissionDate(commissionDate).
		SetCnyExchangeRate(snapshot.ExchangeRate.StringFixed(8)).
		SetCnyExchangeRateSource(commission.CnyExchangeRateSource(snapshot.ExchangeRateSource)).
		SetCnyExchangeRateDate(snapshot.ExchangeRateDate).
		SetNillableCnyExchangeRateSettingID(snapshot.ExchangeRateSettingID).
		SetCnyCommissionAmount(snapshot.CommissionAmount.StringFixed(8)).
		SetVersion(parent.Version + 1)
	if parent.VerificationID != nil {
		update = update.SetVerificationNo(calculation.VerificationNo)
	} else {
		update = update.SetNettingNo(calculation.NettingNo)
	}
	if _, err := update.Save(ctx); err != nil {
		return err
	}
	// 提成行是当前计算事实的逐订单投影：整组替换为重算结果。
	if _, err := client.FinanceCommissionLine.Delete().
		Where(commissionline.CommissionIDEQ(parent.ID)).Exec(ctx); err != nil {
		return err
	}
	lineBuilders, err := financeCommissionLineBuildersFromCalculation(client.FinanceCommissionLine, scope.OrganizationID, parent.ID, calculation)
	if err != nil {
		return err
	}
	if len(lineBuilders) > 0 {
		if _, err := client.FinanceCommissionLine.CreateBulk(lineBuilders...).Save(ctx); err != nil {
			return err
		}
	}
	oldAmount, err := decimalOf(line.CommissionAmount)
	if err != nil {
		return err
	}
	if _, err := client.FinanceCommissionApplicationLine.UpdateOneID(line.ID).
		SetCommissionDate(commissionDate).
		SetNillableRuleID(uuidPointerOrNil(calculation.RuleID)).
		SetRuleVersion(calculation.RuleVersion).
		SetNillableRuleName(stringPointerOrNil(calculation.RuleName)).
		SetNillableCalculationBasis(stringPointerOrNil(string(calculation.CalculationBasis))).
		SetCommissionAmount(calculation.CommissionAmount.StringFixed(8)).
		SetCnyCommissionAmount(snapshot.CommissionAmount.StringFixed(8)).
		SetSourceFingerprint(calculation.SourceFingerprint).
		Save(ctx); err != nil {
		return err
	}
	return writeAudit(ctx, client.AuditLog, &biz.AuditEvent{
		OrganizationID: &scope.OrganizationID,
		UserID:         &scope.UserID,
		Action:         "finance.commission.resubmit_refresh",
		Result:         "success",
		ResourceType:   "finance_commission",
		ResourceID:     parent.ID.String(),
		Details: map[string]string{
			"application_id": header.ID.String(),
			"old_amount":     oldAmount.StringFixed(8),
			"new_amount":     calculation.CommissionAmount.StringFixed(8),
		},
	})
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
