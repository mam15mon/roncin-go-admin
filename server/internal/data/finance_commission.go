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
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	adjustment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	commissionline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	rule "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
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
	predicates := []predicate.User{
		user.EnabledEQ(true),
		user.HasMembershipsWith(membership.OrganizationIDEQ(org), membership.EnabledEQ(true)),
	}
	if options.Keyword != "" {
		predicates = append(predicates, user.Or(
			user.UsernameContainsFold(options.Keyword),
			user.DisplayNameContainsFold(options.Keyword),
			user.SearchKeywordsContainsFold(options.Keyword),
		))
	}
	query := r.data.db.User.Query().Where(predicates...)
	return paginate(ctx, func(ctx context.Context) (int, error) {
		return query.Clone().Count(ctx)
	}, func(ctx context.Context, offset, limit int) ([]*ent.User, error) {
		return query.Order(user.ByDisplayName(), user.ByUsername(), user.ByID()).Offset(offset).Limit(limit).All(ctx)
	}, options.Page, options.PageSize, infalliblePageConverter(func(item *ent.User) *biz.CommissionEmployeeOption {
		return &biz.CommissionEmployeeOption{ID: item.ID, DisplayName: item.DisplayName}
	}))
}

func (r *commissionRepo) ListCandidates(ctx context.Context, org uuid.UUID, f biz.CommissionCandidateFilter) (*biz.CommissionCandidateListResult, error) {
	store := commissionStoreFromClient(r.data.db)
	source, err := loadCommissionCalculationSource(ctx, store, org, f.VerificationID, f.RuleID, false)
	if err != nil {
		return nil, err
	}
	employeePredicates := commissionCandidateEmployeePredicates(org, source, f.Keyword)
	employeeQuery := r.data.db.User.Query().Where(employeePredicates...)
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
	attributions, err := r.data.db.OrderCommissionAttribution.Query().Where(
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

func (r *commissionRepo) ListRules(ctx context.Context, org uuid.UUID, f biz.CommissionRuleFilter) (*biz.CommissionRuleListResult, error) {
	p := []predicate.FinanceCommissionRule{rule.OrganizationIDEQ(org)}
	if f.Keyword != "" {
		p = append(p, rule.NameContainsFold(f.Keyword))
	}
	if f.PersonnelRole != "" {
		p = append(p, rule.PersonnelRoleEQ(rule.PersonnelRole(f.PersonnelRole)))
	}
	if f.Enabled != nil {
		p = append(p, rule.EnabledEQ(*f.Enabled))
	}
	q := r.data.db.FinanceCommissionRule.Query().Where(p...)
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
	return commissionRuleToBiz(x)
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
	return commissionRuleToBiz(updated)
}

func (r *commissionRepo) List(ctx context.Context, org uuid.UUID, f biz.CommissionFilter) (*biz.CommissionListResult, error) {
	p := []predicate.FinanceCommission{commission.OrganizationIDEQ(org)}
	if f.Keyword != "" {
		p = append(p, commission.Or(commission.CommissionNoContainsFold(f.Keyword), commission.EmployeeNameContainsFold(f.Keyword), commission.RuleNameContainsFold(f.Keyword)))
	}
	if f.Status != "" {
		p = append(p, commission.StatusEQ(commission.Status(f.Status)))
	}
	q := r.data.db.FinanceCommission.Query().Where(p...)
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	xs, err := q.WithAdjustments(func(q *ent.FinanceCommissionAdjustmentQuery) {
		q.Order(adjustment.ByCreatedAt())
	}).Order(commission.ByCreatedAt(entsql.OrderDesc())).Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).All(ctx)
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

func (r *commissionRepo) GetByKey(ctx context.Context, org uuid.UUID, key string) (*biz.FinanceCommission, error) {
	x, err := r.data.db.FinanceCommission.Query().Where(commission.OrganizationIDEQ(org), commission.IdempotencyKeyEQ(key)).WithLines(func(q *ent.FinanceCommissionLineQuery) {
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
	rules         *ent.FinanceCommissionRuleClient
	users         *ent.UserClient
	bills         *ent.FinanceBillClient
	attributions  *ent.OrderCommissionAttributionClient
	fees          *ent.OrderFeeClient
	orders        *ent.OrderClient
}

func commissionStoreFromClient(client *ent.Client) commissionCalculationStore {
	return commissionCalculationStore{verifications: client.FinanceVerification, rules: client.FinanceCommissionRule, users: client.User, bills: client.FinanceBill, attributions: client.OrderCommissionAttribution, fees: client.OrderFee, orders: client.Order}
}

func commissionStoreFromTx(tx *ent.Tx) commissionCalculationStore {
	return commissionCalculationStore{verifications: tx.FinanceVerification, rules: tx.FinanceCommissionRule, users: tx.User, bills: tx.FinanceBill, attributions: tx.OrderCommissionAttribution, fees: tx.OrderFee, orders: tx.Order}
}

func (r *commissionRepo) Preview(ctx context.Context, org, verificationID, employeeID, ruleID uuid.UUID) (*biz.CommissionCalculation, error) {
	return calculateCommission(ctx, commissionStoreFromClient(r.data.db), org, verificationID, employeeID, ruleID, false)
}

type commissionCalculationSource struct {
	organizationID  uuid.UUID
	verification    *ent.FinanceVerification
	rule            *ent.FinanceCommissionRule
	rate            decimal.Decimal
	baseCurrency    string
	orderIDs        []uuid.UUID
	orderRealized   map[uuid.UUID]decimal.Decimal
	orderByID       map[uuid.UUID]*ent.Order
	feesByOrder     map[uuid.UUID][]*ent.OrderFee
	fingerprintBase []string
}

func calculateCommission(ctx context.Context, store commissionCalculationStore, org, verificationID, employeeID, ruleID uuid.UUID, lock bool) (*biz.CommissionCalculation, error) {
	source, err := loadCommissionCalculationSource(ctx, store, org, verificationID, ruleID, lock)
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

func loadCommissionCalculationSource(ctx context.Context, store commissionCalculationStore, org, verificationID, ruleID uuid.UUID, lock bool) (*commissionCalculationSource, error) {
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
	rq := store.rules.Query().Where(rule.IDEQ(ruleID), rule.OrganizationIDEQ(org))
	if lock {
		rq.ForUpdate()
	}
	ruleItem, err := rq.Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrCommissionRuleNotFound, nil)
	}
	if !ruleItem.Enabled || (ruleItem.EffectiveFrom != nil && v.VerificationDate < *ruleItem.EffectiveFrom) || (ruleItem.EffectiveTo != nil && v.VerificationDate > *ruleItem.EffectiveTo) {
		return nil, biz.ErrCommissionRuleInvalid
	}
	rate, err := decimal.NewFromString(ruleItem.RatePercent)
	if err != nil {
		return nil, err
	}
	fingerprintParts := []string{
		fmt.Sprintf("calculation|%s", biz.CommissionCalculationVersion),
		fmt.Sprintf("verification|%s|%s|%s|%s|%s|%d", v.ID, v.VerificationNo, v.Status, v.Direction, v.VerificationDate, v.Version),
		fmt.Sprintf("rule|%s|%s|%s|%s|%s|%d|%t|%s|%s", ruleItem.ID, ruleItem.Name, ruleItem.PersonnelRole, ruleItem.CalculationBasis, ruleItem.RatePercent, ruleItem.Version, ruleItem.Enabled, optionalStringValue(ruleItem.EffectiveFrom), optionalStringValue(ruleItem.EffectiveTo)),
	}
	if len(v.Edges.Allocations) == 0 {
		return nil, biz.ErrCommissionSource
	}
	billIDs := make([]uuid.UUID, 0, len(v.Edges.Allocations))
	allocationByBill := make(map[uuid.UUID]decimal.Decimal)
	seenBills := make(map[uuid.UUID]struct{})
	for _, item := range v.Edges.Allocations {
		amount, parseErr := decimal.NewFromString(item.Amount)
		if parseErr != nil {
			return nil, parseErr
		}
		if _, exists := seenBills[item.BillID]; !exists {
			seenBills[item.BillID] = struct{}{}
			billIDs = append(billIDs, item.BillID)
		}
		allocationByBill[item.BillID] = allocationByBill[item.BillID].Add(amount)
		fingerprintParts = append(fingerprintParts, fmt.Sprintf("allocation|%s|%s|%s|%s|%t", item.ID, item.BillID, item.CashflowID, item.Amount, item.Active))
	}
	bq := store.bills.Query().Where(bill.IDIn(billIDs...), bill.OrganizationIDEQ(org), bill.StatusEQ(bill.StatusCONFIRMED), bill.DirectionEQ(bill.DirectionRECEIVABLE)).WithLines()
	if lock {
		bq.ForUpdate()
	}
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
		total, parseErr := decimal.NewFromString(billItem.TotalAmount)
		if parseErr != nil || !total.IsPositive() {
			return nil, biz.ErrCommissionSource
		}
		ratio := allocationByBill[billItem.ID].Div(total)
		fingerprintParts = append(fingerprintParts, fmt.Sprintf("bill|%s|%s|%s|%s|%s|%d", billItem.ID, billItem.BillNo, billItem.Status, billItem.TotalAmount, billItem.BaseCurrencyAmount, billItem.Version))
		for _, billLine := range billItem.Edges.Lines {
			if !billLine.Active {
				continue
			}
			base, parseErr := decimal.NewFromString(billLine.BaseCurrencyAmount)
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
	return &commissionCalculationSource{
		organizationID: org, verification: v, rule: ruleItem, rate: rate, baseCurrency: baseCurrency,
		orderIDs: orderIDs, orderRealized: orderRealized, orderByID: orderByID, feesByOrder: feesByOrder,
		fingerprintBase: fingerprintParts,
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
		VerificationID: source.verification.ID, VerificationNo: source.verification.VerificationNo,
		EmployeeID: employee.ID, EmployeeName: attributions[0].EmployeeName,
		RuleID: source.rule.ID, RuleName: source.rule.Name, PersonnelRole: biz.CommissionPersonnelRole(source.rule.PersonnelRole),
		CalculationBasis: biz.CommissionCalculationBasis(source.rule.CalculationBasis), RuleVersion: source.rule.Version,
		CalculationVersion: biz.CommissionCalculationVersion, BaseCurrency: source.baseCurrency, RatePercent: source.rate,
		Lines: make([]*biz.FinanceCommissionLine, 0, len(eligibleOrderIDs)),
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
			CustomerID: customer.ID, CustomerCode: customer.Code, CustomerName: customer.LegalName,
			CustomerAssignmentID: attributionItem.SourceAssignmentID, CustomerAssignmentOrganizationID: attributionItem.OrganizationID, CustomerAssignedAt: attributionItem.AttributedAt,
			EmployeeID: employee.ID, EmployeeName: attributionItem.EmployeeName, PersonnelRole: result.PersonnelRole,
			CalculationBasis: result.CalculationBasis, RatePercent: source.rate, Fees: make([]*biz.CommissionFeeDetail, 0, len(orderFees)),
		}
		for _, feeItem := range orderFees {
			party, partyErr := feeItem.Edges.SettlementPartyOrErr()
			if partyErr != nil {
				return nil, partyErr
			}
			totalAmount, parseErr := decimal.NewFromString(feeItem.TotalAmount)
			if parseErr != nil {
				return nil, parseErr
			}
			exchangeRate, parseErr := decimal.NewFromString(feeItem.ExchangeRate)
			if parseErr != nil {
				return nil, parseErr
			}
			baseAmount, parseErr := decimal.NewFromString(feeItem.BaseCurrencyAmount)
			if parseErr != nil {
				return nil, parseErr
			}
			if result.BaseCurrency != feeItem.BaseCurrency {
				return nil, biz.ErrCommissionSource
			}
			line.BaseCurrency = result.BaseCurrency
			if feeItem.Direction == fee.DirectionRECEIVABLE {
				line.RealizedRevenue = line.RealizedRevenue.Add(baseAmount)
			} else {
				line.AllocatedCost = line.AllocatedCost.Add(baseAmount)
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

func (r *commissionRepo) Create(ctx context.Context, org, actor uuid.UUID, c *biz.FinanceCommission, audit *biz.AuditEvent) (*biz.FinanceCommission, error) {
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		calculation, err := calculateCommission(ctx, commissionStoreFromTx(tx), org, c.VerificationID, c.EmployeeID, c.RuleID, true)
		if err != nil {
			return err
		}
		hasActive, err := tx.FinanceCommission.Query().Where(commission.VerificationIDEQ(c.VerificationID), commission.EmployeeIDEQ(c.EmployeeID), commission.RuleIDEQ(c.RuleID), commission.StatusNEQ(commission.StatusCANCELLED)).Exist(ctx)
		if err != nil {
			return err
		}
		if hasActive {
			return biz.ErrCommissionDuplicate
		}
		c.VerificationNo, c.EmployeeName, c.RuleName = calculation.VerificationNo, calculation.EmployeeName, calculation.RuleName
		c.PersonnelRole, c.CalculationBasis = calculation.PersonnelRole, calculation.CalculationBasis
		c.RuleVersion, c.CalculationVersion, c.SourceFingerprint = calculation.RuleVersion, calculation.CalculationVersion, calculation.SourceFingerprint
		c.BaseCurrency, c.RatePercent = calculation.BaseCurrency, calculation.RatePercent
		c.CustomerCount, c.OrderCount, c.FeeCount = calculation.CustomerCount, calculation.OrderCount, calculation.FeeCount
		c.RealizedRevenue, c.AllocatedCost, c.RealizedProfit = calculation.RealizedRevenue, calculation.AllocatedCost, calculation.RealizedProfit
		c.CommissionBaseAmount, c.CommissionAmount = calculation.CommissionBaseAmount, calculation.CommissionAmount
		_, err = tx.FinanceCommission.Create().SetID(c.ID).SetOrganizationID(org).SetCommissionNo(c.CommissionNo).SetIdempotencyKey(c.IdempotencyKey).SetVerificationID(c.VerificationID).SetVerificationNo(c.VerificationNo).SetEmployeeID(c.EmployeeID).SetEmployeeName(c.EmployeeName).SetCustomerCount(c.CustomerCount).SetOrderCount(c.OrderCount).SetFeeCount(c.FeeCount).SetRuleID(c.RuleID).SetRuleName(c.RuleName).SetPersonnelRole(string(c.PersonnelRole)).SetCalculationBasis(string(c.CalculationBasis)).SetRuleVersion(c.RuleVersion).SetCalculationVersion(c.CalculationVersion).SetSourceFingerprint(c.SourceFingerprint).SetStatus(commission.StatusDRAFT).SetBaseCurrency(c.BaseCurrency).SetRealizedRevenue(c.RealizedRevenue.StringFixed(8)).SetAllocatedCost(c.AllocatedCost.StringFixed(8)).SetRealizedProfit(c.RealizedProfit.StringFixed(8)).SetCommissionBaseAmount(c.CommissionBaseAmount.StringFixed(8)).SetRatePercent(c.RatePercent.StringFixed(4)).SetCommissionAmount(c.CommissionAmount.StringFixed(8)).SetNillableNote(c.Note).SetVersion(1).Save(ctx)
		if err != nil {
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
			lineBuilders = append(lineBuilders, tx.FinanceCommissionLine.Create().SetID(line.ID).SetOrganizationID(org).SetCommissionID(c.ID).SetOrderID(line.OrderID).SetOrderNo(line.OrderNo).SetOrderDate(line.OrderDate).SetCustomerID(line.CustomerID).SetCustomerCode(line.CustomerCode).SetCustomerName(line.CustomerName).SetPersonnelAssignmentID(line.CustomerAssignmentID).SetPersonnelOrganizationID(line.CustomerAssignmentOrganizationID).SetPersonnelAssignedAt(line.CustomerAssignedAt).SetFeeCount(line.FeeCount).SetFeeSnapshot(string(feeSnapshot)).SetEmployeeID(line.EmployeeID).SetEmployeeName(line.EmployeeName).SetPersonnelRole(string(line.PersonnelRole)).SetCalculationBasis(string(line.CalculationBasis)).SetBaseCurrency(line.BaseCurrency).SetRealizedRevenue(line.RealizedRevenue.StringFixed(8)).SetAllocatedCost(line.AllocatedCost.StringFixed(8)).SetRealizedProfit(line.RealizedProfit.StringFixed(8)).SetCommissionBaseAmount(line.CommissionBaseAmount.StringFixed(8)).SetRatePercent(line.RatePercent.StringFixed(4)).SetCommissionAmount(line.CommissionAmount.StringFixed(8)))
		}
		if _, err = tx.FinanceCommissionLine.CreateBulk(lineBuilders...).Save(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return r.Get(ctx, org, c.ID)
}

func (r *commissionRepo) Transition(ctx context.Context, org, id, actor uuid.UUID, version uint64, target biz.CommissionStatus, reason string, audit *biz.AuditEvent) (*biz.FinanceCommission, error) {
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if target == biz.CommissionConfirmed {
			snapshot, lookupErr := tx.FinanceCommission.Query().Where(commission.IDEQ(id), commission.OrganizationIDEQ(org)).Only(ctx)
			if lookupErr != nil {
				return mapEntError(lookupErr, biz.ErrCommissionNotFound, nil)
			}
			current, calculateErr := calculateCommission(ctx, commissionStoreFromTx(tx), org, snapshot.VerificationID, snapshot.EmployeeID, valueOrNilUUID(snapshot.RuleID), true)
			if calculateErr != nil {
				return biz.ErrCommissionSourceChanged
			}
			if snapshot.SourceFingerprint == "" || snapshot.SourceFingerprint != current.SourceFingerprint {
				return biz.ErrCommissionSourceChanged
			}
			orderIDs := make([]uuid.UUID, 0, len(current.Lines))
			for _, line := range current.Lines {
				orderIDs = append(orderIDs, line.OrderID)
			}
			hasDraftFees, draftErr := tx.OrderFee.Query().Where(fee.OrderIDIn(orderIDs...), fee.StatusEQ(fee.StatusDRAFT)).Exist(ctx)
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

func (r *commissionRepo) Get(ctx context.Context, org, id uuid.UUID) (*biz.FinanceCommission, error) {
	x, err := r.data.db.FinanceCommission.Query().Where(commission.IDEQ(id), commission.OrganizationIDEQ(org)).WithLines(func(q *ent.FinanceCommissionLineQuery) {
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
	return result, nil
}

func commissionToBiz(x *ent.FinanceCommission) (*biz.FinanceCommission, error) {
	revenue, err := decimal.NewFromString(x.RealizedRevenue)
	if err != nil {
		return nil, err
	}
	cost, err := decimal.NewFromString(x.AllocatedCost)
	if err != nil {
		return nil, err
	}
	profit, err := decimal.NewFromString(x.RealizedProfit)
	if err != nil {
		return nil, err
	}
	rate, err := decimal.NewFromString(x.RatePercent)
	if err != nil {
		return nil, err
	}
	amount, err := decimal.NewFromString(x.CommissionAmount)
	if err != nil {
		return nil, err
	}
	commissionBase, err := decimal.NewFromString(x.CommissionBaseAmount)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceCommission{ID: x.ID, OrganizationID: x.OrganizationID, CommissionNo: x.CommissionNo, IdempotencyKey: x.IdempotencyKey, VerificationID: x.VerificationID, VerificationNo: x.VerificationNo, EmployeeID: x.EmployeeID, EmployeeName: x.EmployeeName, CustomerCount: x.CustomerCount, OrderCount: x.OrderCount, FeeCount: x.FeeCount, Status: biz.CommissionStatus(x.Status), BaseCurrency: x.BaseCurrency, RealizedRevenue: revenue, AllocatedCost: cost, RealizedProfit: profit, CommissionBaseAmount: commissionBase, RatePercent: rate, CommissionAmount: amount, EffectiveCommissionAmount: amount, Note: x.Note, Version: x.Version, RuleVersion: x.RuleVersion, CalculationVersion: x.CalculationVersion, SourceFingerprint: x.SourceFingerprint, ConfirmedAt: x.ConfirmedAt, PaidAt: x.PaidAt, CancelledAt: x.CancelledAt, CancellationReason: x.CancellationReason, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt}
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
	revenue, err := decimal.NewFromString(x.RealizedRevenue)
	if err != nil {
		return nil, err
	}
	cost, err := decimal.NewFromString(x.AllocatedCost)
	if err != nil {
		return nil, err
	}
	profit, err := decimal.NewFromString(x.RealizedProfit)
	if err != nil {
		return nil, err
	}
	rate, err := decimal.NewFromString(x.RatePercent)
	if err != nil {
		return nil, err
	}
	amount, err := decimal.NewFromString(x.CommissionAmount)
	if err != nil {
		return nil, err
	}
	commissionBase, err := decimal.NewFromString(x.CommissionBaseAmount)
	if err != nil {
		return nil, err
	}
	fees := make([]*biz.CommissionFeeDetail, 0, x.FeeCount)
	if err = json.Unmarshal([]byte(x.FeeSnapshot), &fees); err != nil {
		return nil, err
	}
	return &biz.FinanceCommissionLine{ID: x.ID, OrganizationID: x.OrganizationID, CommissionID: x.CommissionID, OrderID: x.OrderID, OrderNo: x.OrderNo, OrderDate: x.OrderDate, CustomerID: x.CustomerID, CustomerCode: x.CustomerCode, CustomerName: x.CustomerName, CustomerAssignmentID: x.PersonnelAssignmentID, CustomerAssignmentOrganizationID: x.PersonnelOrganizationID, CustomerAssignedAt: x.PersonnelAssignedAt, EmployeeID: x.EmployeeID, EmployeeName: x.EmployeeName, PersonnelRole: biz.CommissionPersonnelRole(x.PersonnelRole), CalculationBasis: biz.CommissionCalculationBasis(x.CalculationBasis), BaseCurrency: x.BaseCurrency, RealizedRevenue: revenue, AllocatedCost: cost, RealizedProfit: profit, CommissionBaseAmount: commissionBase, RatePercent: rate, CommissionAmount: amount, FeeCount: x.FeeCount, Fees: fees, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt}, nil
}

func (r *commissionRepo) GetAdjustmentByKey(ctx context.Context, org uuid.UUID, key string) (*biz.FinanceCommissionAdjustment, error) {
	x, err := r.data.db.FinanceCommissionAdjustment.Query().Where(adjustment.OrganizationIDEQ(org), adjustment.IdempotencyKeyEQ(key)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
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
		x, err := tx.FinanceCommissionAdjustment.Query().Where(adjustment.IDEQ(id), adjustment.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionAdjustmentNotFound, nil)
		}
		if x.Version != version {
			return biz.ErrCommissionAdjustmentTransition
		}
		parent, err := tx.FinanceCommission.Query().Where(commission.IDEQ(x.CommissionID), commission.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionNotFound, nil)
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
				effective, parseErr := decimal.NewFromString(parent.CommissionAmount)
				if parseErr != nil {
					return parseErr
				}
				for _, old := range active {
					amount, amountErr := decimal.NewFromString(old.Amount)
					if amountErr != nil {
						return amountErr
					}
					if old.Direction == adjustment.DirectionDECREASE {
						effective = effective.Sub(amount)
					} else {
						effective = effective.Add(amount)
					}
				}
				currentAmount, parseErr := decimal.NewFromString(x.Amount)
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
	amount, err := decimal.NewFromString(x.Amount)
	if err != nil {
		return nil, err
	}
	return &biz.FinanceCommissionAdjustment{
		ID: x.ID, OrganizationID: x.OrganizationID, CommissionID: x.CommissionID, OrderID: x.OrderID,
		AdjustmentNo: x.AdjustmentNo, IdempotencyKey: x.IdempotencyKey, CommissionNo: x.CommissionNo,
		OrderNo: x.OrderNo, EmployeeID: x.EmployeeID, EmployeeName: x.EmployeeName,
		Direction: biz.CommissionAdjustmentDirection(x.Direction), SourceType: biz.CommissionAdjustmentSourceType(x.SourceType), SourceVerificationID: x.SourceVerificationID, Status: biz.CommissionStatus(x.Status),
		BaseCurrency: x.BaseCurrency, Amount: amount, Reason: x.Reason, Note: x.Note, Version: x.Version,
		ConfirmedAt: x.ConfirmedAt, PaidAt: x.PaidAt, CancelledAt: x.CancelledAt,
		CancellationReason: x.CancellationReason, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt,
	}, nil
}

func commissionRuleToBiz(x *ent.FinanceCommissionRule) (*biz.FinanceCommissionRule, error) {
	rate, err := decimal.NewFromString(x.RatePercent)
	if err != nil {
		return nil, err
	}
	return &biz.FinanceCommissionRule{ID: x.ID, OrganizationID: x.OrganizationID, Name: x.Name, PersonnelRole: biz.CommissionPersonnelRole(x.PersonnelRole), CalculationBasis: biz.CommissionCalculationBasis(x.CalculationBasis), RatePercent: rate, EffectiveFrom: x.EffectiveFrom, EffectiveTo: x.EffectiveTo, Enabled: x.Enabled, Note: x.Note, Version: x.Version, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt}, nil
}

var _ biz.CommissionRepo = (*commissionRepo)(nil)
