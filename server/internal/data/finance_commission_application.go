package data

import (
	"context"
	"sort"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	applicationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionapplication"
	applicationline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionapplicationline"
	nettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	verification "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	attribution "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	user "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
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
