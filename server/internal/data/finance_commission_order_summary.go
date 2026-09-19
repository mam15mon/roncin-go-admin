package data

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

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
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	attribution "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
)

// ListOrderSummaries 按组织与可见模式分组的批量聚合查询：一次处理整页订单
// ID，不逐行查询。EMPLOYEE 作用域的本人过滤在 SQL 条件内生效，普通员工的
// 聚合结果只可能包含本人事实；取消的提成与调整不参与任何事实桶。
func (r *commissionRepo) ListOrderSummaries(ctx context.Context, scopes []biz.OrderCommissionSummaryScope) (map[uuid.UUID]*biz.OrderCommissionSummary, error) {
	result := make(map[uuid.UUID]*biz.OrderCommissionSummary)
	if len(scopes) == 0 {
		return result, nil
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	for _, scope := range scopes {
		if err = appendOrderCommissionSummaries(ctx, client, scope, result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// orderSummaryBucket 是订单摘要单个事实桶的行数与金额聚合。
type orderSummaryBucket struct {
	count  int
	amount decimal.Decimal
}

func (b *orderSummaryBucket) add(amount decimal.Decimal) {
	b.count++
	b.amount = b.amount.Add(amount)
}

// orderSummaryAggregate 是单个订单在当前可见范围内的全部提成事实聚合。
type orderSummaryAggregate struct {
	currency                                string
	draft, confirmed, paid, pendingDecrease orderSummaryBucket
	opportunitySources                      map[uuid.UUID]struct{}
}

func orderSummaryAggregateFor(aggregates map[uuid.UUID]*orderSummaryAggregate, orderID uuid.UUID) *orderSummaryAggregate {
	aggregate, exists := aggregates[orderID]
	if !exists {
		aggregate = &orderSummaryAggregate{opportunitySources: make(map[uuid.UUID]struct{})}
		aggregates[orderID] = aggregate
	}
	return aggregate
}

// setCurrency 锁定订单金额口径币种；同一订单出现不同币种属于数据异常，禁止
// 裸相加，直接失败而不是静默合并。
func (a *orderSummaryAggregate) setCurrency(currency string) error {
	if a.currency == "" {
		a.currency = currency
		return nil
	}
	if a.currency != currency {
		return biz.ErrCommissionInvalid
	}
	return nil
}

func (a *orderSummaryAggregate) markOpportunity(sourceID uuid.UUID) {
	if a.opportunitySources == nil {
		a.opportunitySources = make(map[uuid.UUID]struct{})
	}
	a.opportunitySources[sourceID] = struct{}{}
}

func (a *orderSummaryAggregate) toSummary(visibility biz.OrderCommissionSummaryVisibility) *biz.OrderCommissionSummary {
	summary := &biz.OrderCommissionSummary{Visibility: visibility, BaseCurrency: a.currency}
	summary.HasExpectedOpportunity = len(a.opportunitySources) > 0
	summary.ExpectedOpportunityCount = len(a.opportunitySources)
	summary.HasDraftCommission = a.draft.count > 0
	summary.DraftCommissionCount = a.draft.count
	summary.DraftCommissionAmount = a.draft.amount.Round(8)
	summary.HasConfirmedCommission = a.confirmed.count > 0
	summary.ConfirmedCommissionCount = a.confirmed.count
	summary.ConfirmedCommissionAmount = a.confirmed.amount.Round(8)
	summary.HasPaidCommission = a.paid.count > 0
	summary.PaidCommissionCount = a.paid.count
	summary.PaidCommissionAmount = a.paid.amount.Round(8)
	summary.HasPendingDecrease = a.pendingDecrease.count > 0
	summary.PendingDecreaseCount = a.pendingDecrease.count
	summary.PendingDecreaseAmount = a.pendingDecrease.amount.Round(8)
	return summary
}

// appendOrderCommissionSummaries 聚合同一组织同一可见模式下一组订单的提成
// 事实：提成单按订单行的父单状态分桶（CANCELLED 不参与）、冲减只统计 DRAFT
// 的 DECREASE 建议（DRAFT ≠ 已扣回）、预计机会复用阶段 C 解析口径。
func appendOrderCommissionSummaries(ctx context.Context, client *ent.Client, scope biz.OrderCommissionSummaryScope, result map[uuid.UUID]*biz.OrderCommissionSummary) error {
	if scope.OrganizationID == uuid.Nil || len(scope.OrderIDs) == 0 {
		return biz.ErrCommissionInvalid
	}
	if scope.Visibility != biz.OrderCommissionVisibilityEmployee && scope.Visibility != biz.OrderCommissionVisibilityOrganization {
		return biz.ErrCommissionInvalid
	}
	employeeOnly := scope.Visibility == biz.OrderCommissionVisibilityEmployee
	if employeeOnly && scope.EmployeeID == uuid.Nil {
		return biz.ErrCommissionInvalid
	}
	aggregates := make(map[uuid.UUID]*orderSummaryAggregate, len(scope.OrderIDs))

	// 1. 提成单事实：先按页面订单定位提成行（EMPLOYEE 视图叠加本人谓词，
	//    本人过滤始终在 SQL 条件内，聚合结果从源头就不包含同事数据），再以
	//    父提成单主键批量取当前组织内非取消状态，Go 侧取交集。与单条 JOIN
	//    语义完全一致；拆分后两条查询分别由 (order_id, employee_id) 复合索引
	//    与父单主键驱动，避免统计信息缺失时 employee_id 位图命中组织全量行、
	//    再与父提成单做无哈希 Join Filter 的退化计划。取消的提成与调整不参与
	//    任何事实桶。
	linePredicates := []predicate.FinanceCommissionLine{
		commissionline.OrderIDIn(scope.OrderIDs...),
	}
	if employeeOnly {
		linePredicates = append(linePredicates, commissionline.EmployeeIDEQ(scope.EmployeeID))
	}
	lines, err := client.FinanceCommissionLine.Query().Where(linePredicates...).All(ctx)
	if err != nil {
		return err
	}
	parentStatus := map[uuid.UUID]string{}
	if len(lines) > 0 {
		commissionIDs := make([]uuid.UUID, 0, len(lines))
		seenCommission := make(map[uuid.UUID]struct{}, len(lines))
		for _, line := range lines {
			if _, exists := seenCommission[line.CommissionID]; exists {
				continue
			}
			seenCommission[line.CommissionID] = struct{}{}
			commissionIDs = append(commissionIDs, line.CommissionID)
		}
		parents, parentErr := client.FinanceCommission.Query().Where(
			commission.OrganizationIDEQ(scope.OrganizationID),
			commission.StatusNEQ(commission.StatusCANCELLED),
			commission.IDIn(commissionIDs...),
		).All(ctx)
		if parentErr != nil {
			return parentErr
		}
		for _, parent := range parents {
			parentStatus[parent.ID] = string(parent.Status)
		}
	}
	for _, line := range lines {
		status, exists := parentStatus[line.CommissionID]
		if !exists {
			continue
		}
		amount, parseErr := decimalOf(line.CommissionAmount)
		if parseErr != nil {
			return parseErr
		}
		aggregate := orderSummaryAggregateFor(aggregates, line.OrderID)
		if currencyErr := aggregate.setCurrency(line.BaseCurrency); currencyErr != nil {
			return currencyErr
		}
		switch biz.CommissionStatus(status) {
		case biz.CommissionDraft:
			aggregate.draft.add(amount)
		case biz.CommissionConfirmed:
			aggregate.confirmed.add(amount)
		case biz.CommissionPaid:
			aggregate.paid.add(amount)
		}
	}

	// 2. 待处理冲减事实：DRAFT + DECREASE 调整单；CONFIRMED 是已确认尚未扣回、
	// PAID 才是已扣回，均不进入该桶。
	adjustmentPredicates := []predicate.FinanceCommissionAdjustment{
		adjustment.OrganizationIDEQ(scope.OrganizationID),
		adjustment.OrderIDIn(scope.OrderIDs...),
		adjustment.StatusEQ(adjustment.StatusDRAFT),
		adjustment.DirectionEQ(adjustment.DirectionDECREASE),
	}
	if employeeOnly {
		adjustmentPredicates = append(adjustmentPredicates, adjustment.EmployeeIDEQ(scope.EmployeeID))
	}
	adjustments, err := client.FinanceCommissionAdjustment.Query().Where(adjustmentPredicates...).All(ctx)
	if err != nil {
		return err
	}
	for _, item := range adjustments {
		amount, parseErr := decimalOf(item.Amount)
		if parseErr != nil {
			return parseErr
		}
		aggregate := orderSummaryAggregateFor(aggregates, item.OrderID)
		if currencyErr := aggregate.setCurrency(item.BaseCurrency); currencyErr != nil {
			return currencyErr
		}
		aggregate.pendingDecrease.add(amount)
	}

	if err = appendOrderSummaryOpportunities(ctx, client, scope, aggregates); err != nil {
		return err
	}
	for orderID, aggregate := range aggregates {
		result[orderID] = aggregate.toSummary(scope.Visibility)
	}
	return nil
}

// orderSummaryOpportunitySource 是参与预计机会判定的计提来源：核销或对冲二选一，
// 携带来源归属日期与按账单汇总的有效分摊金额。
type orderSummaryOpportunitySource struct {
	id             uuid.UUID
	commissionDate string
	billAmounts    map[uuid.UUID]decimal.Decimal
	orderRealized  map[uuid.UUID]decimal.Decimal
}

// appendOrderSummaryOpportunities 复用阶段 C 计提解析口径批量判断预计机会：
// 来源（ACTIVE 应收核销 / CONFIRMED 对冲的 RECEIVABLE 有效分摊）经 CONFIRMED
// 应收账单的有效行摊入页面订单；订单上须存在「员工 + 人员身份」提成归属，且
// 订单有已确认/已开票的正数应收费用；来源归属日期须唯一命中「已启用方案 ∩
// 未取消员工分配」；该「来源 + 员工 + 身份」尚无非取消基础提成单。机会数量
// 按去重来源单计数，只给事实不给估算金额。
func appendOrderSummaryOpportunities(ctx context.Context, client *ent.Client, scope biz.OrderCommissionSummaryScope, aggregates map[uuid.UUID]*orderSummaryAggregate) error {
	employeeOnly := scope.Visibility == biz.OrderCommissionVisibilityEmployee
	attributionPredicates := []predicate.OrderCommissionAttribution{
		attribution.OrganizationIDEQ(scope.OrganizationID),
		attribution.OrderIDIn(scope.OrderIDs...),
		// 与候选发现同口径：订单须存在已确认/已开票的正数应收费用。
		attribution.HasOrderWith(orderent.HasFeesWith(
			fee.StatusIn(fee.StatusCONFIRMED, fee.StatusBILLED),
			fee.DirectionEQ(fee.DirectionRECEIVABLE),
			fee.BaseCurrencyAmountGT("0"),
		)),
	}
	if employeeOnly {
		attributionPredicates = append(attributionPredicates, attribution.EmployeeIDEQ(scope.EmployeeID))
	}
	attributions, err := client.OrderCommissionAttribution.Query().Where(attributionPredicates...).All(ctx)
	if err != nil {
		return err
	}
	if len(attributions) == 0 {
		return nil
	}

	// 来源摊入的账单集合：CONFIRMED 应收账单且包含页面订单的有效行；只加载
	// 页面订单的行即可得到这些订单的已实现分摊。
	billRows, err := client.FinanceBill.Query().Where(
		bill.OrganizationIDEQ(scope.OrganizationID),
		bill.StatusEQ(bill.StatusCONFIRMED),
		bill.DirectionEQ(bill.DirectionRECEIVABLE),
		bill.HasLinesWith(billline.OrderIDIn(scope.OrderIDs...), billline.ActiveEQ(true)),
	).WithLines(func(q *ent.FinanceBillLineQuery) {
		q.Where(billline.OrderIDIn(scope.OrderIDs...), billline.ActiveEQ(true))
	}).All(ctx)
	if err != nil {
		return err
	}
	billTotalByID := make(map[uuid.UUID]decimal.Decimal, len(billRows))
	billLinesByBill := make(map[uuid.UUID][]*ent.FinanceBillLine, len(billRows))
	billIDs := make([]uuid.UUID, 0, len(billRows))
	for _, billItem := range billRows {
		total, parseErr := decimalOf(billItem.TotalAmount)
		if parseErr != nil {
			return parseErr
		}
		if !total.IsPositive() {
			continue
		}
		billIDs = append(billIDs, billItem.ID)
		billTotalByID[billItem.ID] = total
		billLinesByBill[billItem.ID] = billItem.Edges.Lines
	}
	if len(billIDs) == 0 {
		return nil
	}

	sources := make([]*orderSummaryOpportunitySource, 0)
	verificationIDs := make([]uuid.UUID, 0)
	nettingIDs := make([]uuid.UUID, 0)
	verifications, err := client.FinanceVerification.Query().Where(
		verification.OrganizationIDEQ(scope.OrganizationID),
		verification.StatusEQ(verification.StatusACTIVE),
		verification.DirectionEQ(verification.DirectionRECEIVABLE),
		verification.HasAllocationsWith(allocation.ActiveEQ(true), allocation.BillIDIn(billIDs...)),
	).WithAllocations(func(q *ent.FinanceVerificationAllocationQuery) {
		q.Where(allocation.ActiveEQ(true))
	}).All(ctx)
	if err != nil {
		return err
	}
	for _, item := range verifications {
		billAmounts := make(map[uuid.UUID]decimal.Decimal)
		for _, alloc := range item.Edges.Allocations {
			if !alloc.Active {
				continue
			}
			if _, target := billTotalByID[alloc.BillID]; !target {
				continue
			}
			amount, parseErr := decimalOf(alloc.Amount)
			if parseErr != nil {
				return parseErr
			}
			billAmounts[alloc.BillID] = billAmounts[alloc.BillID].Add(amount)
		}
		if len(billAmounts) == 0 {
			continue
		}
		sources = append(sources, &orderSummaryOpportunitySource{id: item.ID, commissionDate: item.VerificationDate, billAmounts: billAmounts})
		verificationIDs = append(verificationIDs, item.ID)
	}
	nettings, err := client.FinanceNetting.Query().Where(
		nettingent.OrganizationIDEQ(scope.OrganizationID),
		nettingent.StatusEQ(nettingent.StatusCONFIRMED),
		nettingent.HasAllocationsWith(nettingalloc.ActiveEQ(true), nettingalloc.DirectionEQ(nettingalloc.DirectionRECEIVABLE), nettingalloc.BillIDIn(billIDs...)),
	).WithAllocations(func(q *ent.FinanceNettingAllocationQuery) {
		q.Where(nettingalloc.ActiveEQ(true))
	}).All(ctx)
	if err != nil {
		return err
	}
	for _, item := range nettings {
		// 对冲来源归属日期与生成上下文同口径：确认时间的 UTC 日期。
		commissionDate := ""
		if item.ConfirmedAt != nil {
			commissionDate = item.ConfirmedAt.UTC().Format("2006-01-02")
		}
		billAmounts := make(map[uuid.UUID]decimal.Decimal)
		for _, alloc := range item.Edges.Allocations {
			if !alloc.Active || alloc.Direction != nettingalloc.DirectionRECEIVABLE {
				continue
			}
			if _, target := billTotalByID[alloc.BillID]; !target {
				continue
			}
			amount, parseErr := decimalOf(alloc.Amount)
			if parseErr != nil {
				return parseErr
			}
			billAmounts[alloc.BillID] = billAmounts[alloc.BillID].Add(amount)
		}
		if len(billAmounts) == 0 {
			continue
		}
		sources = append(sources, &orderSummaryOpportunitySource{id: item.ID, commissionDate: commissionDate, billAmounts: billAmounts})
		nettingIDs = append(nettingIDs, item.ID)
	}
	if len(sources) == 0 {
		return nil
	}
	// 与提成计算同口径：按「分摊金额 / 账单总额」把账单行本位币摊入订单。
	for _, source := range sources {
		source.orderRealized = make(map[uuid.UUID]decimal.Decimal, len(scope.OrderIDs))
		for billID, allocated := range source.billAmounts {
			ratio := allocated.Div(billTotalByID[billID])
			for _, line := range billLinesByBill[billID] {
				base, parseErr := decimalOf(line.BaseCurrencyAmount)
				if parseErr != nil {
					return parseErr
				}
				source.orderRealized[line.OrderID] = source.orderRealized[line.OrderID].Add(base.Mul(ratio))
			}
		}
	}

	// 员工 × 身份组合去重；批量加载涉及身份的已启用方案与未取消分配段，
	// 避免逐组合查询。
	type opportunityCombo struct {
		orderID    uuid.UUID
		employeeID uuid.UUID
		role       biz.CommissionPersonnelRole
	}
	employeeSet := make(map[uuid.UUID]struct{})
	roleSet := make(map[biz.CommissionPersonnelRole]struct{})
	combos := make([]opportunityCombo, 0, len(attributions))
	for _, item := range attributions {
		role := biz.CommissionPersonnelRole(item.PersonnelRole)
		employeeSet[item.EmployeeID] = struct{}{}
		roleSet[role] = struct{}{}
		combos = append(combos, opportunityCombo{orderID: item.OrderID, employeeID: item.EmployeeID, role: role})
	}
	sort.Slice(combos, func(i, j int) bool {
		if combos[i].orderID != combos[j].orderID {
			return combos[i].orderID.String() < combos[j].orderID.String()
		}
		if combos[i].employeeID != combos[j].employeeID {
			return combos[i].employeeID.String() < combos[j].employeeID.String()
		}
		return combos[i].role < combos[j].role
	})
	employeeIDs := make([]uuid.UUID, 0, len(employeeSet))
	for employeeID := range employeeSet {
		employeeIDs = append(employeeIDs, employeeID)
	}
	employeeIDs = uniqueSortedUUIDs(employeeIDs)
	rulesByRole := make(map[biz.CommissionPersonnelRole][]*ent.FinanceCommissionRule, len(roleSet))
	for role := range roleSet {
		ruleRows, ruleErr := client.FinanceCommissionRule.Query().Where(
			rule.OrganizationIDEQ(scope.OrganizationID),
			rule.EnabledEQ(true),
			rule.PersonnelRoleEQ(rule.PersonnelRole(role)),
		).WithAssignments(func(q *ent.FinanceCommissionRuleAssignmentQuery) {
			q.Where(assignment.CancelledAtIsNil(), assignment.EmployeeIDIn(employeeIDs...))
		}).All(ctx)
		if ruleErr != nil {
			return ruleErr
		}
		rulesByRole[role] = ruleRows
	}

	// 已有非取消基础提成单的「来源 + 员工 + 身份」不再是预计机会；一次批量
	// 查询后在内存匹配，避免逐组合查询。
	type activeCommissionKey struct {
		sourceID   uuid.UUID
		employeeID uuid.UUID
		role       string
	}
	activeCommissions := make(map[activeCommissionKey]struct{})
	sourcePredicates := make([]predicate.FinanceCommission, 0, 2)
	if len(verificationIDs) > 0 {
		sourcePredicates = append(sourcePredicates, commission.VerificationIDIn(verificationIDs...))
	}
	if len(nettingIDs) > 0 {
		sourcePredicates = append(sourcePredicates, commission.NettingIDIn(nettingIDs...))
	}
	if len(sourcePredicates) > 0 {
		activeRows, activeErr := client.FinanceCommission.Query().Where(
			commission.OrganizationIDEQ(scope.OrganizationID),
			commission.StatusNEQ(commission.StatusCANCELLED),
			commission.EmployeeIDIn(employeeIDs...),
			commission.Or(sourcePredicates...),
		).All(ctx)
		if activeErr != nil {
			return activeErr
		}
		for _, row := range activeRows {
			var sourceID uuid.UUID
			if row.VerificationID != nil {
				sourceID = *row.VerificationID
			} else if row.NettingID != nil {
				sourceID = *row.NettingID
			} else {
				continue
			}
			role := ""
			if row.PersonnelRole != nil {
				role = *row.PersonnelRole
			}
			activeCommissions[activeCommissionKey{sourceID: sourceID, employeeID: row.EmployeeID, role: role}] = struct{}{}
		}
	}

	for _, combo := range combos {
		for _, source := range sources {
			if !source.orderRealized[combo.orderID].IsPositive() {
				continue
			}
			if _, exists := activeCommissions[activeCommissionKey{sourceID: source.id, employeeID: combo.employeeID, role: string(combo.role)}]; exists {
				continue
			}
			if !resolveOrderSummaryRule(rulesByRole[combo.role], combo.employeeID, source.commissionDate) {
				continue
			}
			orderSummaryAggregateFor(aggregates, combo.orderID).markOpportunity(source.id)
		}
	}
	return nil
}

// resolveOrderSummaryRule 按来源归属日期在内存中判定「已启用方案 ∩ 未取消员工
// 分配」是否唯一命中；闭区间判定与多命中拒绝口径与 resolveCommissionRuleForDate
// 一致。规则已按组织 + 身份 + 启用过滤，分配段已按未取消过滤。
func resolveOrderSummaryRule(rules []*ent.FinanceCommissionRule, employeeID uuid.UUID, commissionDate string) bool {
	if commissionDate == "" {
		return false
	}
	hits := 0
	for _, ruleItem := range rules {
		if (ruleItem.EffectiveFrom != nil && commissionDate < *ruleItem.EffectiveFrom) || (ruleItem.EffectiveTo != nil && commissionDate > *ruleItem.EffectiveTo) {
			continue
		}
		for _, seg := range ruleItem.Edges.Assignments {
			if seg.EmployeeID != employeeID {
				continue
			}
			if commissionDate < seg.EffectiveFrom || (seg.EffectiveTo != nil && commissionDate > *seg.EffectiveTo) {
				continue
			}
			hits++
			if hits > 1 {
				// 同员工同身份在同一归属日期命中多个方案：与计提解析一致，
				// 不视为可靠的预计机会。
				return false
			}
		}
	}
	return hits == 1
}
