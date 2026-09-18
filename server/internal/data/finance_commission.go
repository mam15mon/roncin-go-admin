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
	nettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	nettingalloc "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	verification "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	allocation "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	attribution "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
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

func (r *commissionRepo) ListCandidates(ctx context.Context, org uuid.UUID, f biz.CommissionCandidateFilter) (*biz.CommissionCandidateListResult, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	store := commissionStoreFromClient(client)
	source, err := loadCommissionCalculationSource(ctx, store, org, f.VerificationID, uuid.Nil, f.RuleID, false)
	if err != nil {
		return nil, err
	}
	employeePredicates := commissionCandidateEmployeePredicates(org, source, f.Keyword)
	employeeQuery := client.User.Query().Where(employeePredicates...)
	total, err := employeeQuery.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	employees, err := employeeQuery.
		Order(user.ByDisplayName(), user.ByUsername(), user.ByID()).
		Offset((f.Page - 1) * f.PageSize).
		Limit(f.PageSize).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.CommissionCandidateListResult{
		Items: make([]*biz.CommissionCalculation, 0, len(employees)), Total: int64(total), Page: f.Page, PageSize: f.PageSize,
	}
	if len(employees) == 0 {
		return result, nil
	}
	employeeIDs := make([]uuid.UUID, 0, len(employees))
	for _, employee := range employees {
		employeeIDs = append(employeeIDs, employee.ID)
	}
	attributions, err := client.OrderCommissionAttribution.Query().Where(
		attribution.OrganizationIDEQ(org),
		attribution.OrderIDIn(source.orderIDs...),
		attribution.EmployeeIDIn(employeeIDs...),
		attribution.PersonnelRoleEQ(attribution.PersonnelRole(source.rule.PersonnelRole)),
	).Order(attribution.ByAttributedAt(), attribution.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	attributionsByEmployee := make(map[uuid.UUID][]*ent.OrderCommissionAttribution, len(employees))
	for _, item := range attributions {
		attributionsByEmployee[item.EmployeeID] = append(attributionsByEmployee[item.EmployeeID], item)
	}
	for _, employee := range employees {
		calculation, calculateErr := calculateCommissionFromSource(source, employee, attributionsByEmployee[employee.ID])
		if calculateErr != nil {
			return nil, calculateErr
		}
		calculation.Lines = nil
		result.Items = append(result.Items, calculation)
	}
	return result, nil
}

func commissionCandidateEmployeePredicates(org uuid.UUID, source *commissionCalculationSource, keyword string) []predicate.User {
	attributionPredicates := []predicate.OrderCommissionAttribution{
		attribution.OrganizationIDEQ(org),
		attribution.OrderIDIn(source.orderIDs...),
		attribution.PersonnelRoleEQ(attribution.PersonnelRole(source.rule.PersonnelRole)),
		attribution.HasOrderWith(orderent.HasFeesWith(
			fee.StatusIn(fee.StatusCONFIRMED, fee.StatusBILLED),
			fee.DirectionEQ(fee.DirectionRECEIVABLE),
			fee.BaseCurrencyAmountGT("0"),
		)),
	}
	employeePredicates := []predicate.User{
		user.EnabledEQ(true),
		user.HasOrderCommissionAttributionsWith(attributionPredicates...),
	}
	if keyword != "" {
		employeePredicates = append(employeePredicates, user.Or(
			user.UsernameContainsFold(keyword),
			user.DisplayNameContainsFold(keyword),
			user.SearchKeywordsContainsFold(keyword),
		))
	}
	return employeePredicates
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
		p = append(p, rule.NameContainsFold(f.Keyword))
	}
	if f.PersonnelRole != "" {
		p = append(p, rule.PersonnelRoleEQ(rule.PersonnelRole(f.PersonnelRole)))
	}
	if f.Enabled != nil {
		p = append(p, rule.EnabledEQ(*f.Enabled))
	}
	q := client.FinanceCommissionRule.Query().WithOrganization().Where(p...)
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
	x, err := client.FinanceCommissionRule.Query().Where(rule.IDEQ(id), rule.OrganizationIDIn(organizationIDs...)).WithOrganization().Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrCommissionRuleNotFound, nil)
	}
	return commissionRuleToBiz(x)
}
func (r *commissionRepo) CreateRule(ctx context.Context, org uuid.UUID, item *biz.FinanceCommissionRule, audit *biz.AuditEvent) (*biz.FinanceCommissionRule, error) {
	var x *ent.FinanceCommissionRule
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		var err error
		x, err = tx.FinanceCommissionRule.Create().SetID(item.ID).SetOrganizationID(org).SetName(item.Name).SetPersonnelRole(rule.PersonnelRole(item.PersonnelRole)).SetCalculationBasis(rule.CalculationBasis(item.CalculationBasis)).SetRatePercent(item.RatePercent.StringFixed(4)).SetNillableEffectiveFrom(item.EffectiveFrom).SetNillableEffectiveTo(item.EffectiveTo).SetEnabled(item.Enabled).SetNillableNote(item.Note).SetVersion(1).Save(ctx)
		if err != nil {
			return mapEntError(err, nil, biz.ErrCommissionRuleConflict)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return r.GetRuleScoped(ctx, []uuid.UUID{org}, x.ID)
}
func (r *commissionRepo) UpdateRule(ctx context.Context, org uuid.UUID, in biz.UpdateCommissionRuleInput, audit *biz.AuditEvent) (*biz.FinanceCommissionRule, error) {
	var updated *ent.FinanceCommissionRule
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		x, err := tx.FinanceCommissionRule.Query().Where(rule.IDEQ(in.ID), rule.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionRuleNotFound, nil)
		}
		if x.Version != in.ExpectedVersion {
			return biz.ErrCommissionRuleConflict
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

func (r *commissionRepo) Preview(ctx context.Context, org, verificationID, nettingID, employeeID, ruleID uuid.UUID) (*biz.CommissionCalculation, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	return calculateCommission(ctx, commissionStoreFromClient(client), org, verificationID, nettingID, employeeID, ruleID, false)
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
type commissionCalculationSource struct {
	organizationID    uuid.UUID
	verification      *ent.FinanceVerification // 核销来源时非空
	netting           *ent.FinanceNetting      // 对冲来源时非空
	commissionDate    string                   // 归属日期：核销 verification_date / 对冲确认日
	rule              *ent.FinanceCommissionRule
	rate              decimal.Decimal
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

func calculateCommission(ctx context.Context, store commissionCalculationStore, org, verificationID, nettingID, employeeID, ruleID uuid.UUID, lock bool) (*biz.CommissionCalculation, error) {
	source, err := loadCommissionCalculationSource(ctx, store, org, verificationID, nettingID, ruleID, lock)
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
	aq := store.attributions.Query().Where(
		attribution.OrganizationIDEQ(org),
		attribution.OrderIDIn(source.orderIDs...),
		attribution.EmployeeIDEQ(employeeID),
		attribution.PersonnelRoleEQ(attribution.PersonnelRole(source.rule.PersonnelRole)),
	).Order(attribution.ByAttributedAt(), attribution.ByID())
	if lock {
		aq.ForUpdate()
	}
	attributions, err := aq.All(ctx)
	if err != nil {
		return nil, err
	}
	return calculateCommissionFromSource(source, employee, attributions)
}

// loadCommissionCalculationSource 按来源（核销 ACTIVE+RECEIVABLE / 对冲 CONFIRMED 的
// RECEIVABLE 分摊）加载统一分摊条目，并锁定规则、账单、订单、费用与账单行事实。
// 多行加锁一律按主键稳定排序，固定加锁顺序防止并发提成创建死锁。
func loadCommissionCalculationSource(ctx context.Context, store commissionCalculationStore, org, verificationID, nettingID, ruleID uuid.UUID, lock bool) (*commissionCalculationSource, error) {
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
	rq := store.rules.Query().Where(rule.IDEQ(ruleID), rule.OrganizationIDEQ(org))
	if lock {
		rq.ForUpdate()
	}
	ruleItem, err := rq.Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrCommissionRuleNotFound, nil)
	}
	if !ruleItem.Enabled || (ruleItem.EffectiveFrom != nil && commissionDate < *ruleItem.EffectiveFrom) || (ruleItem.EffectiveTo != nil && commissionDate > *ruleItem.EffectiveTo) {
		return nil, biz.ErrCommissionRuleInvalid
	}
	rate, err := decimalOf(ruleItem.RatePercent)
	if err != nil {
		return nil, err
	}
	fingerprintParts = append(fingerprintParts, fmt.Sprintf("rule|%s|%s|%s|%s|%s|%d|%t|%s|%s", ruleItem.ID, ruleItem.Name, ruleItem.PersonnelRole, ruleItem.CalculationBasis, ruleItem.RatePercent, ruleItem.Version, ruleItem.Enabled, optionalStringValue(ruleItem.EffectiveFrom), optionalStringValue(ruleItem.EffectiveTo)))
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
		rule: ruleItem, rate: rate, baseCurrency: baseCurrency,
		orderIDs: orderIDs, orderRealized: orderRealized, orderByID: orderByID, feesByOrder: feesByOrder,
		billLineBaseByFee: billLineBaseByFee,
		fingerprintBase:   fingerprintParts,
	}, nil
}

func calculateCommissionFromSource(source *commissionCalculationSource, employee *ent.User, attributions []*ent.OrderCommissionAttribution) (*biz.CommissionCalculation, error) {
	if len(attributions) == 0 {
		return nil, biz.ErrCommissionEmployeeRole
	}
	fingerprintParts := append([]string(nil), source.fingerprintBase...)
	fingerprintParts = append(fingerprintParts, fmt.Sprintf("employee|%s|%s|%t", employee.ID, employee.DisplayName, employee.Enabled))
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
		RuleID: source.rule.ID, RuleName: source.rule.Name, PersonnelRole: biz.CommissionPersonnelRole(source.rule.PersonnelRole),
		CalculationBasis: biz.CommissionCalculationBasis(source.rule.CalculationBasis), RuleVersion: source.rule.Version,
		CalculationVersion: biz.CommissionCalculationVersion, BaseCurrency: source.baseCurrency, RatePercent: source.rate,
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
			CalculationBasis: result.CalculationBasis, RatePercent: source.rate, Fees: make([]*biz.CommissionFeeDetail, 0, len(orderFees)),
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
		cost, profit, commissionBase, amount, calculateErr := biz.CalculateCommissionLine(realized, totalReceivable, totalPayable, source.rate, result.CalculationBasis)
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
// 完整业务响应由用例在共享事务提交后通过普通上下文重读。
func (r *commissionRepo) Create(ctx context.Context, org uuid.UUID, c *biz.FinanceCommission, snapshot *biz.CommissionCNYSnapshot, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		calculation, err := calculateCommission(ctx, commissionStoreFromTx(tx), org, c.VerificationID, c.NettingID, c.EmployeeID, c.RuleID, true)
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
		c.RuleVersion, c.CalculationVersion, c.SourceFingerprint = calculation.RuleVersion, calculation.CalculationVersion, calculation.SourceFingerprint
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
			// 复算，避免引入与提成创建路径相反的订单/账单加锁顺序。
			snapshot, lookupErr := tx.FinanceCommission.Query().Where(commission.IDEQ(id), commission.OrganizationIDEQ(org)).Only(ctx)
			if lookupErr != nil {
				return mapEntError(lookupErr, biz.ErrCommissionNotFound, nil)
			}
			current, calculateErr := calculateCommission(ctx, commissionStoreFromTx(tx), org, valueOrNilUUID(snapshot.VerificationID), valueOrNilUUID(snapshot.NettingID), snapshot.EmployeeID, valueOrNilUUID(snapshot.RuleID), true)
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
				currentAmount, parseErr := decimalOf(x.Amount)
				if parseErr != nil {
					return parseErr
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
			if x.Status != adjustment.StatusDRAFT && x.Status != adjustment.StatusCONFIRMED {
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
	result := &biz.FinanceCommissionRule{ID: x.ID, OrganizationID: x.OrganizationID, Name: x.Name, PersonnelRole: biz.CommissionPersonnelRole(x.PersonnelRole), CalculationBasis: biz.CommissionCalculationBasis(x.CalculationBasis), RatePercent: rate, EffectiveFrom: x.EffectiveFrom, EffectiveTo: x.EffectiveTo, Enabled: x.Enabled, Note: x.Note, Version: x.Version, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt}
	if x.Edges.Organization != nil {
		result.OrganizationName = x.Edges.Organization.Name
	}
	return result, nil
}

var _ biz.CommissionRepo = (*commissionRepo)(nil)
