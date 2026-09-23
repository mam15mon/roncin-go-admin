package data

import (
	"context"
	"sort"
	"strings"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	nettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	nettingalloc "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	attribution "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

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
			fee.StatusIn(fee.StatusUNBILLED, fee.StatusBILLED),
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
