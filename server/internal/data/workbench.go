package data

import (
	"context"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	bill "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	billline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	adjustment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	rule "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	assignment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionruleassignment"
	nettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	nettingalloc "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	verification "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	allocation "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderabnormalcaseent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderabnormalcase"
	attribution "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderfeesupplementent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeesupplementrequest"
	orderpersonnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
	"github.com/shopspring/decimal"
)

// workbenchRepo 工作台读模型仓储：全部查询只读、限定当前工作区组织与当前用户，
// 并按固定上限有界返回。门禁为假时不查询也不返回提成金额模块；不同组织本位币
// 不合计（Overview 与明细只服务单一当前组织）。本文件不写任何业务数据，也
// 不复用写路径事务。
type workbenchRepo struct{ data *Data }

func NewWorkbenchRepo(data *Data) biz.WorkbenchRepo { return &workbenchRepo{data: data} }

// workbenchStatusAmountRow 是提成/调整按状态分桶的数据库侧聚合行。
type workbenchStatusAmountRow struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
	Total  string `json:"total"`
}

// GetOverview 聚合模块门禁、本人提成状态汇总、近期订单、可靠作业待办与
// 财务/审批摘要。查询顺序：门禁 → 近期订单与待办 → 财务摘要 → 提成汇总
// （仅门禁为真时查询），保证无资格用户不产生提成金额读取。
func (r *workbenchRepo) GetOverview(ctx context.Context, scope biz.WorkbenchScope) (*biz.WorkbenchOverview, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	overview := &biz.WorkbenchOverview{}

	organization, err := client.Organization.Query().
		Where(organizationent.IDEQ(scope.OrganizationID)).
		Select(organizationent.FieldID, organizationent.FieldBaseCurrency).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	if organization.BaseCurrency != nil {
		overview.BaseCurrency = *organization.BaseCurrency
	}

	facts, err := r.workbenchEligibilityFacts(ctx, client, scope)
	if err != nil {
		return nil, err
	}
	eligible, nextEffectiveDate := biz.ResolveWorkbenchEligibility(facts, scope.Today)
	overview.Eligible = eligible
	overview.NextEffectiveDate = nextEffectiveDate

	// 近期订单与作业待办共享同一个「本人协作 ∩ 当前组织」订单 ID 集合，一次
	// 解析后分别按订单主键与 order_id 索引驱动查询。
	myOrderIDs, err := r.workbenchMyOrderIDs(ctx, client, scope)
	if err != nil {
		return nil, err
	}
	recentOrders, err := r.workbenchRecentOrders(ctx, client, scope, myOrderIDs, biz.WorkbenchOverviewRecentOrderLimit)
	if err != nil {
		return nil, err
	}
	overview.RecentOrders = recentOrders

	todos, err := r.workbenchTodoSummary(ctx, client, scope, myOrderIDs)
	if err != nil {
		return nil, err
	}
	overview.Todos = *todos

	// 财务/审批摘要按权限与实时资格返回：无任何财务能力时字段为缺省值，
	// 前端依据 can_read_commission、can_manage_commission 与计数决定是否出现卡片。
	finance, err := r.workbenchFinanceSummary(ctx, client, scope)
	if err != nil {
		return nil, err
	}
	overview.Finance = finance

	// 门禁为假：不查询也不返回本人提成金额模块。
	if eligible {
		summary, summaryErr := r.workbenchCommissionSummary(ctx, client, scope, overview.BaseCurrency)
		if summaryErr != nil {
			return nil, summaryErr
		}
		overview.CommissionSummary = summary
	}
	return overview, nil
}

// workbenchEligibilityFacts 读取门禁原始事实：本人未取消、方案启用且实际区间
// 尚未结束的分配段，以及本人非取消历史提成单与调整单存在性。分配段查询有界
// （单员工可见方案数量受 200 上限约束），历史存在性由数据库 EXISTS 完成。
func (r *workbenchRepo) workbenchEligibilityFacts(ctx context.Context, client *ent.Client, scope biz.WorkbenchScope) (biz.WorkbenchEligibilityFacts, error) {
	facts := biz.WorkbenchEligibilityFacts{}
	segments, err := client.FinanceCommissionRuleAssignment.Query().
		Where(
			assignment.OrganizationIDEQ(scope.OrganizationID),
			assignment.EmployeeIDEQ(scope.UserID),
			assignment.CancelledAtIsNil(),
			// 分配段自身尚未结束：终点为空（正无穷）或不早于业务日期。
			assignment.Or(assignment.EffectiveToIsNil(), assignment.EffectiveToGTE(scope.Today)),
			assignment.HasRuleWith(
				rule.EnabledEQ(true),
				rule.Or(rule.EffectiveToIsNil(), rule.EffectiveToGTE(scope.Today)),
			),
		).
		WithRule().
		Order(assignment.ByEffectiveFrom(), assignment.ByID()).
		Limit(biz.MaxListPageSize).
		All(ctx)
	if err != nil {
		return facts, err
	}
	for _, segment := range segments {
		ruleItem := segment.Edges.Rule
		window, ok := biz.WorkbenchActualAssignmentWindow(
			ruleItem.Enabled,
			optionalStringValue(ruleItem.EffectiveFrom), optionalStringValue(ruleItem.EffectiveTo),
			segment.EffectiveFrom, optionalStringValue(segment.EffectiveTo),
		)
		if ok {
			facts.ActiveWindows = append(facts.ActiveWindows, window)
		}
	}
	facts.HasCommissionHistory, err = client.FinanceCommission.Query().Where(
		commission.OrganizationIDEQ(scope.OrganizationID),
		commission.EmployeeIDEQ(scope.UserID),
		commission.StatusIn(commission.StatusDRAFT, commission.StatusCONFIRMED, commission.StatusPAID),
	).Exist(ctx)
	if err != nil {
		return facts, err
	}
	facts.HasAdjustmentHistory, err = client.FinanceCommissionAdjustment.Query().Where(
		adjustment.OrganizationIDEQ(scope.OrganizationID),
		adjustment.EmployeeIDEQ(scope.UserID),
		adjustment.StatusIn(adjustment.StatusDRAFT, adjustment.StatusCONFIRMED, adjustment.StatusPAID),
	).Exist(ctx)
	return facts, err
}

// workbenchCommissionSummary 按状态分桶统计本人提成与 DECREASE 调整：单组织
// 本位币口径，数据库侧聚合；CANCELLED 不参与任何桶。
func (r *workbenchRepo) workbenchCommissionSummary(ctx context.Context, client *ent.Client, scope biz.WorkbenchScope, baseCurrency string) (*biz.WorkbenchCommissionSummary, error) {
	summary := &biz.WorkbenchCommissionSummary{BaseCurrency: baseCurrency}
	var err error
	personalCommissionPredicates := []predicate.FinanceCommission{
		commission.OrganizationIDEQ(scope.OrganizationID),
		commission.EmployeeIDEQ(scope.UserID),
		commission.StatusNEQ(commission.StatusCANCELLED),
	}
	commissionRows := make([]workbenchStatusAmountRow, 0)
	if err := client.FinanceCommission.Query().Where(personalCommissionPredicates...).
		GroupBy(commission.FieldStatus).
		Aggregate(ent.As(ent.Count(), "count"), ent.As(ent.Sum(commission.FieldCommissionAmount), "total")).
		Scan(ctx, &commissionRows); err != nil {
		return nil, err
	}
	if err := applyWorkbenchStatusAmounts(commissionRows, map[string]func(int, decimal.Decimal){
		string(commission.StatusDRAFT):     func(count int, total decimal.Decimal) { summary.DraftCount, summary.DraftAmount = count, total },
		string(commission.StatusCONFIRMED): func(count int, total decimal.Decimal) { summary.ConfirmedCount, summary.ConfirmedAmount = count, total },
		string(commission.StatusPAID):      func(count int, total decimal.Decimal) { summary.PaidCount, summary.PaidAmount = count, total },
	}); err != nil {
		return nil, err
	}

	yearStart, monthStart := biz.WorkbenchPaidPeriodRange(scope.Now)
	paidWindowPredicates := func(since time.Time) []predicate.FinanceCommission {
		return append(append([]predicate.FinanceCommission{}, personalCommissionPredicates...),
			commission.StatusEQ(commission.StatusPAID),
			commission.PaidAtGTE(since),
		)
	}
	if summary.PaidThisYear, err = workbenchSumCommissions(ctx, client, paidWindowPredicates(yearStart)); err != nil {
		return nil, err
	}
	if summary.PaidThisMonth, err = workbenchSumCommissions(ctx, client, paidWindowPredicates(monthStart)); err != nil {
		return nil, err
	}

	decreasePredicates := []predicate.FinanceCommissionAdjustment{
		adjustment.OrganizationIDEQ(scope.OrganizationID),
		adjustment.EmployeeIDEQ(scope.UserID),
		adjustment.DirectionEQ(adjustment.DirectionDECREASE),
		adjustment.StatusNEQ(adjustment.StatusCANCELLED),
	}
	decreaseRows := make([]workbenchStatusAmountRow, 0)
	if err := client.FinanceCommissionAdjustment.Query().Where(decreasePredicates...).
		GroupBy(adjustment.FieldStatus).
		Aggregate(ent.As(ent.Count(), "count"), ent.As(ent.Sum(adjustment.FieldAmount), "total")).
		Scan(ctx, &decreaseRows); err != nil {
		return nil, err
	}
	if err := applyWorkbenchStatusAmounts(decreaseRows, map[string]func(int, decimal.Decimal){
		string(adjustment.StatusDRAFT): func(count int, total decimal.Decimal) {
			summary.DecreaseDraftCount, summary.DecreaseDraftAmount = count, total
		},
		string(adjustment.StatusCONFIRMED): func(count int, total decimal.Decimal) {
			summary.DecreaseConfirmedCount, summary.DecreaseConfirmedAmount = count, total
		},
		string(adjustment.StatusPAID): func(count int, total decimal.Decimal) {
			summary.DecreasePaidCount, summary.DecreasePaidAmount = count, total
		},
	}); err != nil {
		return nil, err
	}

	estimated, err := r.workbenchEstimatedOpportunity(ctx, client, scope)
	if err != nil {
		return nil, err
	}
	// 无任何可靠机会时不返回估算结构，前端据此显示「暂无法估算」而不是零金额。
	if estimated.OpportunityCount == 0 {
		estimated = nil
	}
	summary.Estimated = estimated
	return summary, nil
}

// applyWorkbenchStatusAmounts 把按状态聚合行写入对应分桶。
func applyWorkbenchStatusAmounts(rows []workbenchStatusAmountRow, sink map[string]func(int, decimal.Decimal)) error {
	for _, row := range rows {
		apply, ok := sink[row.Status]
		if !ok {
			continue
		}
		total, err := decimalOf(row.Total)
		if err != nil {
			return err
		}
		apply(row.Count, total)
	}
	return nil
}

// workbenchSumCommissions 按谓词求提成金额合计（本位币口径）。
// 无匹配行时 SQL 聚合返回单行 NULL，按零处理。
func workbenchSumCommissions(ctx context.Context, client *ent.Client, predicates []predicate.FinanceCommission) (decimal.Decimal, error) {
	rows := make([]workbenchStatusAmountRow, 0)
	if err := client.FinanceCommission.Query().Where(predicates...).
		Aggregate(ent.As(ent.Sum(commission.FieldCommissionAmount), "total")).
		Scan(ctx, &rows); err != nil {
		return decimal.Zero, err
	}
	if len(rows) == 0 || rows[0].Total == "" {
		return decimal.Zero, nil
	}
	return decimalOf(rows[0].Total)
}

// workbenchReceivableOrderPredicates 是「本人提成归属 + 有效应收费用」的订单
// 谓词：与计提候选发现保持同一口径（存在本人提成归属，且订单存在 CONFIRMED/
// BILLED 应收费用），复用现有计提来源校验语义。
func workbenchAttributedOrderPredicate(scope biz.WorkbenchScope) predicate.Order {
	return orderent.And(
		orderent.HasCommissionAttributionsWith(attribution.EmployeeIDEQ(scope.UserID)),
		orderent.HasFeesWith(
			fee.StatusIn(fee.StatusCONFIRMED, fee.StatusBILLED),
			fee.DirectionEQ(fee.DirectionRECEIVABLE),
			fee.BaseCurrencyAmountGT("0"),
		),
	)
}

// workbenchEstimatedOpportunity 汇总「预计可计提」机会：候选来源为本人有提成
// 归属、且来源条件与现有计提引擎一致的有效应收核销/对冲单；逐来源按阶段 C 的
// 解析口径（来源归属日期唯一命中本人对应身份的已启用方案与未取消分配）试算，
// 已存在本人对应身份有效提成的来源不再是机会。扫描有界：超出上限的来源不参与
// 金额合计，只以 HasMore 标记为不完整下界。
func (r *workbenchRepo) workbenchEstimatedOpportunity(ctx context.Context, client *ent.Client, scope biz.WorkbenchScope) (*biz.WorkbenchEstimatedOpportunity, error) {
	result := &biz.WorkbenchEstimatedOpportunity{}
	store := commissionStoreFromClient(client)

	attributedOrder := workbenchAttributedOrderPredicate(scope)
	verificationPredicates := []predicate.FinanceVerification{
		verification.OrganizationIDEQ(scope.OrganizationID),
		verification.StatusEQ(verification.StatusACTIVE),
		verification.DirectionEQ(verification.DirectionRECEIVABLE),
		verification.HasAllocationsWith(allocation.ActiveEQ(true), allocation.HasBillWith(
			bill.HasLinesWith(billline.ActiveEQ(true), billline.HasOrderWith(attributedOrder)),
		)),
	}
	nettingPredicates := []predicate.FinanceNetting{
		nettingent.OrganizationIDEQ(scope.OrganizationID),
		nettingent.StatusEQ(nettingent.StatusCONFIRMED),
		nettingent.ConfirmedAtNotNil(),
		nettingent.HasAllocationsWith(
			nettingalloc.ActiveEQ(true),
			nettingalloc.DirectionEQ(nettingalloc.DirectionRECEIVABLE),
			nettingalloc.HasBillWith(bill.HasLinesWith(billline.ActiveEQ(true), billline.HasOrderWith(attributedOrder))),
		),
	}
	verificationTotal, err := client.FinanceVerification.Query().Where(verificationPredicates...).Count(ctx)
	if err != nil {
		return nil, err
	}
	nettingTotal, err := client.FinanceNetting.Query().Where(nettingPredicates...).Count(ctx)
	if err != nil {
		return nil, err
	}
	verificationItems, err := client.FinanceVerification.Query().Where(verificationPredicates...).
		Order(verification.ByCreatedAt(entsql.OrderDesc()), verification.ByID(entsql.OrderDesc())).
		Limit(biz.WorkbenchEstimatedScanLimit).All(ctx)
	if err != nil {
		return nil, err
	}
	nettingItems, err := client.FinanceNetting.Query().Where(nettingPredicates...).
		Order(nettingent.ByCreatedAt(entsql.OrderDesc()), nettingent.ByID(entsql.OrderDesc())).
		Limit(biz.WorkbenchEstimatedScanLimit).All(ctx)
	if err != nil {
		return nil, err
	}
	result.HasMore = verificationTotal+nettingTotal > len(verificationItems)+len(nettingItems)

	employee, err := store.users.Query().Where(user.IDEQ(scope.UserID)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return result, nil
		}
		return nil, err
	}

	// 逐来源按既有计提口径试算；解析失败、来源不可计算等「无可靠估算条件」的
	// 组合静默跳过，其余错误原样外传。
	skipped := map[error]bool{biz.ErrCommissionRuleNotResolved: true, biz.ErrCommissionSource: true, biz.ErrCommissionEmployeeRole: true}
	process := func(verificationID, nettingID uuid.UUID, commissionDate string) error {
		source, sourceErr := loadCommissionCalculationSource(ctx, store, scope.OrganizationID, verificationID, nettingID, false)
		if sourceErr != nil {
			if skipped[sourceErr] {
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
		roles := make([]biz.CommissionPersonnelRole, 0, 2)
		roleSeen := make(map[biz.CommissionPersonnelRole]struct{}, 2)
		for _, item := range attributions {
			role := biz.CommissionPersonnelRole(item.PersonnelRole)
			if _, exists := roleSeen[role]; exists {
				continue
			}
			roleSeen[role] = struct{}{}
			roles = append(roles, role)
		}
		for _, role := range roles {
			roleAttributions := make([]*ent.OrderCommissionAttribution, 0, len(attributions))
			for _, item := range attributions {
				if biz.CommissionPersonnelRole(item.PersonnelRole) == role {
					roleAttributions = append(roleAttributions, item)
				}
			}
			hasCommission, existErr := workbenchHasActiveCommission(ctx, client, scope, verificationID, nettingID, role)
			if existErr != nil {
				return existErr
			}
			if hasCommission {
				continue
			}
			ruleItem, assignmentItem, resolveErr := resolveCommissionRuleForDate(ctx, store, scope.OrganizationID, scope.UserID, role, commissionDate, false)
			if resolveErr != nil {
				if skipped[resolveErr] {
					continue
				}
				return resolveErr
			}
			calculation, calculateErr := calculateCommissionFromSource(source, ruleItem, assignmentItem, employee, roleAttributions)
			if calculateErr != nil {
				if skipped[calculateErr] {
					continue
				}
				return calculateErr
			}
			result.OpportunityCount++
			result.EstimatedAmount = result.EstimatedAmount.Add(calculation.CommissionAmount).Round(8)
		}
		return nil
	}
	for _, item := range verificationItems {
		if processErr := process(item.ID, uuid.Nil, item.VerificationDate); processErr != nil {
			return nil, processErr
		}
	}
	for _, item := range nettingItems {
		if item.ConfirmedAt == nil {
			continue
		}
		if processErr := process(uuid.Nil, item.ID, item.ConfirmedAt.UTC().Format("2006-01-02")); processErr != nil {
			return nil, processErr
		}
	}
	return result, nil
}

// workbenchHasActiveCommission 判断本人对应身份在该来源上是否已有非取消提成单。
func workbenchHasActiveCommission(ctx context.Context, client *ent.Client, scope biz.WorkbenchScope, verificationID, nettingID uuid.UUID, role biz.CommissionPersonnelRole) (bool, error) {
	predicates := []predicate.FinanceCommission{
		commission.OrganizationIDEQ(scope.OrganizationID),
		commission.EmployeeIDEQ(scope.UserID),
		commission.PersonnelRoleEQ(string(role)),
		commission.StatusNEQ(commission.StatusCANCELLED),
	}
	if verificationID != uuid.Nil {
		predicates = append(predicates, commission.VerificationIDEQ(verificationID))
	} else {
		predicates = append(predicates, commission.NettingIDEQ(nettingID))
	}
	return client.FinanceCommission.Query().Where(predicates...).Exist(ctx)
}

// workbenchMyOrderIDs 解析「本人真实协作 ∩ 当前组织」的订单 ID 集合：先按
// user_id + organization_id 索引取协作人员行的 order_id，再以订单主键批量定位
// 组织内订单。两步均为 order_id / 主键驱动的精确扫描，等价于原先
// `orders ⨝ order_personnels` 半连接的语义；不依赖统计信息即可避免半连接在
// 基数误估（估计数行、实际数百行）下退化成十万次量级的 Join Filter 拒绝。
// 返回 ID 已按协作人员行去重，顺序不稳定，调用方须显式排序。
func (r *workbenchRepo) workbenchMyOrderIDs(ctx context.Context, client *ent.Client, scope biz.WorkbenchScope) ([]uuid.UUID, error) {
	var rawIDs []string
	if err := client.OrderPersonnel.Query().
		Where(
			orderpersonnelent.UserIDEQ(scope.UserID),
			orderpersonnelent.OrganizationIDEQ(scope.OrganizationID),
		).
		Select(orderpersonnelent.FieldOrderID).
		Scan(ctx, &rawIDs); err != nil {
		return nil, err
	}
	if len(rawIDs) == 0 {
		return []uuid.UUID{}, nil
	}
	seen := make(map[uuid.UUID]struct{}, len(rawIDs))
	collaborated := make([]uuid.UUID, 0, len(rawIDs))
	for _, raw := range rawIDs {
		parsed, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			return nil, parseErr
		}
		if _, exists := seen[parsed]; exists {
			continue
		}
		seen[parsed] = struct{}{}
		collaborated = append(collaborated, parsed)
	}
	orders, err := client.Order.Query().
		Where(orderent.IDIn(collaborated...), orderent.OrganizationIDEQ(scope.OrganizationID)).
		Select(orderent.FieldID).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return []uuid.UUID{}, nil
	}
	orderIDs := make([]uuid.UUID, 0, len(orders))
	for _, item := range orders {
		orderIDs = append(orderIDs, item.ID)
	}
	return orderIDs, nil
}

// workbenchRecentOrders 查询本人真实协作的近期海运出口订单（OrderPersonnel 归属）。
func (r *workbenchRepo) workbenchRecentOrders(ctx context.Context, client *ent.Client, scope biz.WorkbenchScope, myOrderIDs []uuid.UUID, limit int) ([]*biz.WorkbenchRecentOrder, error) {
	if len(myOrderIDs) == 0 {
		return []*biz.WorkbenchRecentOrder{}, nil
	}
	items, err := r.workbenchRecentOrderQuery(client, scope, myOrderIDs).Limit(limit).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.WorkbenchRecentOrder, 0, len(items))
	for _, item := range items {
		result = append(result, workbenchRecentOrderToBiz(item))
	}
	return result, nil
}

// workbenchRecentOrderQuery 是本人海运出口订单的基础查询：本人协作订单 ID 集合
// 内的当前组织 SE 订单；按创建时间倒序稳定排序。
func (r *workbenchRepo) workbenchRecentOrderQuery(client *ent.Client, scope biz.WorkbenchScope, myOrderIDs []uuid.UUID) *ent.OrderQuery {
	return client.Order.Query().Where(
		orderent.IDIn(myOrderIDs...),
		orderent.OrganizationIDEQ(scope.OrganizationID),
		orderent.BusinessTypeEQ(orderent.BusinessTypeSE),
	).WithCustomer(func(query *ent.PartnerQuery) {
		query.Select(partnerent.FieldID, partnerent.FieldLegalName)
	}).Order(orderent.ByCreatedAt(entsql.OrderDesc()), orderent.ByID(entsql.OrderDesc()))
}

func workbenchRecentOrderToBiz(item *ent.Order) *biz.WorkbenchRecentOrder {
	customerName := ""
	if item.Edges.Customer != nil {
		customerName = item.Edges.Customer.LegalName
	}
	return &biz.WorkbenchRecentOrder{
		OrderID:           item.ID,
		OrderNo:           item.OrderNo,
		BusinessType:      string(item.BusinessType),
		CustomerName:      customerName,
		FlowStatus:        string(item.FlowStatus),
		TerminationStatus: string(item.TerminationStatus),
		OrderDate:         item.OrderDate,
		CreatedAt:         item.CreatedAt,
	}
}

// workbenchTodoSummary 统计本人协作订单上的可靠作业待办：草稿费用与未解决异常。
// 两条计数均由 order_id + status 复合索引驱动，语义等价于「订单在当前组织且
// 本人存在于协作人员」，order_id 集合由 workbenchMyOrderIDs 统一解析。
func (r *workbenchRepo) workbenchTodoSummary(ctx context.Context, client *ent.Client, scope biz.WorkbenchScope, myOrderIDs []uuid.UUID) (*biz.WorkbenchTodoSummary, error) {
	summary := &biz.WorkbenchTodoSummary{}
	if len(myOrderIDs) == 0 {
		return summary, nil
	}
	draftFeeCount, err := client.OrderFee.Query().Where(
		fee.StatusEQ(fee.StatusDRAFT),
		fee.OrderIDIn(myOrderIDs...),
	).Count(ctx)
	if err != nil {
		return nil, err
	}
	summary.DraftFeeCount = draftFeeCount
	openAbnormalCount, err := client.OrderAbnormalCase.Query().Where(
		orderabnormalcaseent.StatusEQ(orderabnormalcaseent.StatusACTIVE),
		orderabnormalcaseent.OrderIDIn(myOrderIDs...),
	).Count(ctx)
	if err != nil {
		return nil, err
	}
	summary.OpenAbnormalCount = openAbnormalCount
	return summary, nil
}

// workbenchFinanceSummary 聚合财务提成/冲减摘要与补录审批待办：
//   - 提成/冲减计数仅在当前用户对当前组织具备 commission.read 时查询；
//   - 补录审批只聚合当前用户实时具备直接解锁资格的 PENDING 申请，复用补录仓储
//     同一资格谓词，不引入「财务角色即审批人」的第二套规则；
//   - 扫描有界：PENDING 候选超过上限时计数仅为下界并置 truncated。
func (r *workbenchRepo) workbenchFinanceSummary(ctx context.Context, client *ent.Client, scope biz.WorkbenchScope) (*biz.WorkbenchFinanceSummary, error) {
	summary := &biz.WorkbenchFinanceSummary{
		CanReadCommission:   scope.CommissionReadable,
		CanManageCommission: scope.CommissionManageable,
	}
	if scope.CommissionReadable {
		confirmedCount, err := client.FinanceCommission.Query().Where(
			commission.OrganizationIDEQ(scope.OrganizationID),
			commission.StatusEQ(commission.StatusCONFIRMED),
		).Count(ctx)
		if err != nil {
			return nil, err
		}
		summary.ConfirmedCommissionCount = confirmedCount
		pendingDecreaseCount, err := client.FinanceCommissionAdjustment.Query().Where(
			adjustment.OrganizationIDEQ(scope.OrganizationID),
			adjustment.DirectionEQ(adjustment.DirectionDECREASE),
			adjustment.StatusEQ(adjustment.StatusDRAFT),
			adjustment.SourceTypeEQ(adjustment.SourceTypeLOCKED_FEE_SUPPLEMENT),
		).Count(ctx)
		if err != nil {
			return nil, err
		}
		summary.PendingDecreaseCount = pendingDecreaseCount
	}

	requests, err := client.OrderFeeSupplementRequest.Query().Where(
		orderfeesupplementent.OrganizationIDEQ(scope.OrganizationID),
		orderfeesupplementent.StatusEQ(orderfeesupplementent.StatusPENDING),
	).WithOrder().WithRequestedByUser(func(query *ent.UserQuery) {
		query.Select(user.FieldID, user.FieldDisplayName)
	}).Order(orderfeesupplementent.ByCreatedAt(entsql.OrderDesc()), orderfeesupplementent.ByID(entsql.OrderDesc())).
		Limit(biz.WorkbenchSupplementScanLimit + 1).All(ctx)
	if err != nil {
		return nil, err
	}
	truncated := len(requests) > biz.WorkbenchSupplementScanLimit
	if truncated {
		requests = requests[:biz.WorkbenchSupplementScanLimit]
	}
	for _, requestItem := range requests {
		if requestItem.Edges.Order == nil {
			continue
		}
		businessType, typeErr := orderAccessBusinessType(requestItem.Edges.Order.BusinessType)
		if typeErr != nil {
			// 业务类型不受支持视为不具备资格，与审批路径一致按无资格跳过。
			continue
		}
		qualified, grantErr := hasRealtimeLockGrantWithClient(ctx, client, scope.OrganizationID, scope.UserID, businessType, scope.IsBootstrapAdmin)
		if grantErr != nil {
			return nil, grantErr
		}
		if !qualified {
			continue
		}
		summary.PendingSupplementApprovalCount++
		if len(summary.PendingSupplementApprovals) < biz.WorkbenchSupplementItemLimit {
			amount, amountErr := decimalOf(requestItem.TotalAmount)
			if amountErr != nil {
				return nil, amountErr
			}
			summary.PendingSupplementApprovals = append(summary.PendingSupplementApprovals, &biz.WorkbenchSupplementApprovalItem{
				RequestID:       requestItem.ID,
				OrderID:         requestItem.OrderID,
				OrderNo:         requestItem.Edges.Order.OrderNo,
				FeeCode:         requestItem.FeeCode,
				FeeName:         requestItem.FeeName,
				Currency:        requestItem.Currency,
				Amount:          amount,
				Reason:          requestItem.Reason,
				RequestedByName: workbenchDisplayName(requestItem.Edges.RequestedByUser),
				RequestedAt:     requestItem.CreatedAt,
			})
		}
	}
	summary.SupplementApprovalsTruncated = truncated
	return summary, nil
}

func workbenchDisplayName(item *ent.User) string {
	if item == nil {
		return ""
	}
	return item.DisplayName
}

// ListMyCommissions 返回本人提成单与调整明细分页：组织 + 本人固定在查询条件中，
// 状态与归属日期过滤在数据库侧完成；按归属日期与创建时间倒序稳定排序。
func (r *workbenchRepo) ListMyCommissions(ctx context.Context, scope biz.WorkbenchScope, f biz.WorkbenchCommissionFilter) (*biz.PagedList[*biz.WorkbenchMyCommission], error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	predicates := []predicate.FinanceCommission{
		commission.OrganizationIDEQ(scope.OrganizationID),
		commission.EmployeeIDEQ(scope.UserID),
	}
	if f.Status != "" {
		predicates = append(predicates, commission.StatusEQ(commission.Status(f.Status)))
	}
	if f.CommissionDateFrom != "" {
		predicates = append(predicates, commission.CommissionDateGTE(f.CommissionDateFrom))
	}
	if f.CommissionDateTo != "" {
		predicates = append(predicates, commission.CommissionDateLTE(f.CommissionDateTo))
	}
	query := client.FinanceCommission.Query().Where(predicates...).WithAdjustments(func(adjustmentQuery *ent.FinanceCommissionAdjustmentQuery) {
		adjustmentQuery.Order(adjustment.ByCreatedAt(entsql.OrderDesc()), adjustment.ByID(entsql.OrderDesc()))
	})
	return paginate(ctx, func(ctx context.Context) (int, error) {
		return query.Clone().Count(ctx)
	}, func(ctx context.Context, offset, limit int) ([]*ent.FinanceCommission, error) {
		return query.Order(
			commission.ByCommissionDate(entsql.OrderDesc()),
			commission.ByCreatedAt(entsql.OrderDesc()),
			commission.ByID(entsql.OrderDesc()),
		).Offset(offset).Limit(limit).All(ctx)
	}, f.Page, f.PageSize, workbenchMyCommissionToBiz)
}

func workbenchMyCommissionToBiz(item *ent.FinanceCommission) (*biz.WorkbenchMyCommission, error) {
	amount, err := decimalOf(item.CommissionAmount)
	if err != nil {
		return nil, err
	}
	personnelRole := ""
	if item.PersonnelRole != nil {
		personnelRole = *item.PersonnelRole
	}
	result := &biz.WorkbenchMyCommission{
		ID:               item.ID,
		CommissionNo:     item.CommissionNo,
		Status:           biz.CommissionStatus(item.Status),
		PersonnelRole:    biz.CommissionPersonnelRole(personnelRole),
		RuleName:         item.RuleName,
		CalculationBasis: item.CalculationBasis,
		BaseCurrency:     item.BaseCurrency,
		CommissionAmount: amount,
		CommissionDate:   item.CommissionDate,
		VerificationNo:   item.VerificationNo,
		NettingNo:        item.NettingNo,
		CreatedAt:        item.CreatedAt,
		Adjustments:      make([]*biz.WorkbenchMyCommissionAdjustment, 0, len(item.Edges.Adjustments)),
	}
	for _, adjustmentItem := range item.Edges.Adjustments {
		adjustmentAmount, err := decimalOf(adjustmentItem.Amount)
		if err != nil {
			return nil, err
		}
		result.Adjustments = append(result.Adjustments, &biz.WorkbenchMyCommissionAdjustment{
			ID:           adjustmentItem.ID,
			AdjustmentNo: adjustmentItem.AdjustmentNo,
			Direction:    biz.CommissionAdjustmentDirection(adjustmentItem.Direction),
			Status:       biz.CommissionStatus(adjustmentItem.Status),
			Amount:       adjustmentAmount,
			Reason:       adjustmentItem.Reason,
			CreatedAt:    adjustmentItem.CreatedAt,
		})
	}
	return result, nil
}

// ListMyReceivables 返回与本人提成归属相关的已确认应收未结项分页：
// 未结谓词与账单台账共用同一 SQL 口径；未结余额、逾期天数经与账单列表相同的
// 富化方法计算，保证「工作台显示未结清、台账可核销」两侧一致。
func (r *workbenchRepo) ListMyReceivables(ctx context.Context, scope biz.WorkbenchScope, f biz.WorkbenchReceivableFilter) (*biz.PagedList[*biz.WorkbenchMyReceivable], error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	predicates := []predicate.FinanceBill{
		bill.OrganizationIDEQ(scope.OrganizationID),
		bill.DirectionEQ(bill.DirectionRECEIVABLE),
		bill.StatusEQ(bill.StatusCONFIRMED),
		billUnsettledPredicate(),
		bill.HasLinesWith(billline.ActiveEQ(true), billline.HasOrderWith(
			orderent.HasCommissionAttributionsWith(attribution.EmployeeIDEQ(scope.UserID)),
		)),
	}
	query := client.FinanceBill.Query().Where(predicates...)
	return paginate(ctx, func(ctx context.Context) (int, error) {
		return query.Clone().Count(ctx)
	}, func(ctx context.Context, offset, limit int) ([]*ent.FinanceBill, error) {
		return query.Order(
			bill.ByBillDate(entsql.OrderDesc()),
			bill.ByCreatedAt(entsql.OrderDesc()),
			bill.ByID(entsql.OrderDesc()),
		).Offset(offset).Limit(limit).All(ctx)
	}, f.Page, f.PageSize, func(item *ent.FinanceBill) (*biz.WorkbenchMyReceivable, error) {
		return r.workbenchMyReceivableToBiz(ctx, item, scope)
	})
}

// workbenchMyReceivableToBiz 单条转换并复用账单仓储的核销/对冲富化口径：
// 未结余额 = 总额 - 有效核销 - 有效对冲（负值钳零），逾期天数按业务日期计算。
func (r *workbenchRepo) workbenchMyReceivableToBiz(ctx context.Context, item *ent.FinanceBill, scope biz.WorkbenchScope) (*biz.WorkbenchMyReceivable, error) {
	totalAmount, err := decimalOf(item.TotalAmount)
	if err != nil {
		return nil, err
	}
	projected := &biz.FinanceBill{
		ID:                  item.ID,
		BillNo:              item.BillNo,
		Direction:           biz.OrderFeeDirection(item.Direction),
		Status:              biz.FinanceBillStatus(item.Status),
		SettlementPartyName: item.SettlementPartyName,
		Currency:            item.Currency,
		TotalAmount:         totalAmount,
		BillDate:            item.BillDate,
		DueDate:             item.DueDate,
		ConfirmedAt:         item.ConfirmedAt,
	}
	if enrichErr := (&financeBillRepo{data: r.data}).enrichVerificationAmounts(ctx, []*biz.FinanceBill{projected}, scope.Today); enrichErr != nil {
		return nil, enrichErr
	}
	confirmedAt := time.Time{}
	if item.ConfirmedAt != nil {
		confirmedAt = *item.ConfirmedAt
	}
	return &biz.WorkbenchMyReceivable{
		BillID:              item.ID,
		BillNo:              item.BillNo,
		SettlementPartyName: item.SettlementPartyName,
		Currency:            item.Currency,
		TotalAmount:         projected.TotalAmount,
		UnverifiedAmount:    projected.UnverifiedAmount,
		BillDate:            &item.BillDate,
		DueDate:             item.DueDate,
		OverdueDays:         projected.OverdueDays,
		ConfirmedAt:         confirmedAt,
	}, nil
}

// ListMyRecentOrders 返回本人真实协作的近期海运出口订单分页。
func (r *workbenchRepo) ListMyRecentOrders(ctx context.Context, scope biz.WorkbenchScope, f biz.WorkbenchOrderFilter) (*biz.PagedList[*biz.WorkbenchRecentOrder], error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	myOrderIDs, err := r.workbenchMyOrderIDs(ctx, client, scope)
	if err != nil {
		return nil, err
	}
	if len(myOrderIDs) == 0 {
		return &biz.PagedList[*biz.WorkbenchRecentOrder]{Items: []*biz.WorkbenchRecentOrder{}, Page: f.Page, PageSize: f.PageSize}, nil
	}
	query := r.workbenchRecentOrderQuery(client, scope, myOrderIDs)
	return paginate(ctx, func(ctx context.Context) (int, error) {
		return client.Order.Query().Where(
			orderent.IDIn(myOrderIDs...),
			orderent.OrganizationIDEQ(scope.OrganizationID),
			orderent.BusinessTypeEQ(orderent.BusinessTypeSE),
		).Count(ctx)
	}, func(ctx context.Context, offset, limit int) ([]*ent.Order, error) {
		return query.Offset(offset).Limit(limit).All(ctx)
	}, f.Page, f.PageSize, infalliblePageConverter(workbenchRecentOrderToBiz))
}
