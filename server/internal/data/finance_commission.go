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
	rule "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	verification "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	allocation "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	personnel "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
	"github.com/shopspring/decimal"
)

type commissionRepo struct{ data *Data }

func NewCommissionRepo(data *Data) biz.CommissionRepo { return &commissionRepo{data: data} }

func (r *commissionRepo) ListEmployees(ctx context.Context, org uuid.UUID) ([]*biz.CommissionEmployeeOption, error) {
	xs, err := r.data.db.User.Query().Where(user.EnabledEQ(true), user.HasMembershipsWith(membership.OrganizationIDEQ(org), membership.EnabledEQ(true))).Order(user.ByDisplayName()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.CommissionEmployeeOption, 0, len(xs))
	for _, x := range xs {
		result = append(result, &biz.CommissionEmployeeOption{ID: x.ID, DisplayName: x.DisplayName})
	}
	return result, nil
}

func (r *commissionRepo) ListCandidates(ctx context.Context, org, verificationID, ruleID uuid.UUID) ([]*biz.CommissionEmployeeOption, error) {
	ruleItem, err := r.data.db.FinanceCommissionRule.Query().Where(rule.IDEQ(ruleID), rule.OrganizationIDEQ(org), rule.EnabledEQ(true)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, biz.ErrCommissionRuleNotFound
	}
	if err != nil {
		return nil, err
	}
	verificationItem, err := r.data.db.FinanceVerification.Query().Where(verification.IDEQ(verificationID), verification.OrganizationIDEQ(org), verification.StatusEQ(verification.StatusACTIVE), verification.DirectionEQ(verification.DirectionRECEIVABLE)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, biz.ErrCommissionSource
	}
	if err != nil {
		return nil, err
	}
	if (ruleItem.EffectiveFrom != nil && verificationItem.VerificationDate < *ruleItem.EffectiveFrom) || (ruleItem.EffectiveTo != nil && verificationItem.VerificationDate > *ruleItem.EffectiveTo) {
		return nil, biz.ErrCommissionRuleInvalid
	}
	allocations, err := r.data.db.FinanceVerificationAllocation.Query().Where(allocation.VerificationIDEQ(verificationID), allocation.ActiveEQ(true)).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(allocations) == 0 {
		return nil, biz.ErrCommissionSource
	}
	billIDs := make([]uuid.UUID, 0, len(allocations))
	for _, item := range allocations {
		billIDs = append(billIDs, item.BillID)
	}
	lines, err := r.data.db.FinanceBillLine.Query().Where(billline.BillIDIn(billIDs...), billline.ActiveEQ(true)).All(ctx)
	if err != nil {
		return nil, err
	}
	orderIDs := make([]uuid.UUID, 0, len(lines))
	seenOrders := map[uuid.UUID]struct{}{}
	for _, line := range lines {
		if _, ok := seenOrders[line.OrderID]; !ok {
			seenOrders[line.OrderID] = struct{}{}
			orderIDs = append(orderIDs, line.OrderID)
		}
	}
	if len(orderIDs) == 0 {
		return []*biz.CommissionEmployeeOption{}, nil
	}
	items, err := r.data.db.OrderPersonnel.Query().Where(personnel.OrganizationIDEQ(org), personnel.OrderIDIn(orderIDs...), personnel.RoleEQ(personnel.Role(ruleItem.PersonnelRole)), personnel.HasUserWith(user.EnabledEQ(true), user.HasMembershipsWith(membership.OrganizationIDEQ(org), membership.EnabledEQ(true)))).WithUser().All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.CommissionEmployeeOption, 0, len(items))
	seenUsers := map[uuid.UUID]struct{}{}
	for _, item := range items {
		employee, edgeErr := item.Edges.UserOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		if _, ok := seenUsers[employee.ID]; ok {
			continue
		}
		seenUsers[employee.ID] = struct{}{}
		result = append(result, &biz.CommissionEmployeeOption{ID: employee.ID, DisplayName: employee.DisplayName})
	}
	return result, nil
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
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	rollback := func(err error) (*biz.FinanceCommissionRule, error) { _ = tx.Rollback(); return nil, err }
	x, err := tx.FinanceCommissionRule.Create().SetID(item.ID).SetOrganizationID(org).SetName(item.Name).SetPersonnelRole(rule.PersonnelRole(item.PersonnelRole)).SetCalculationBasis(rule.CalculationBasis(item.CalculationBasis)).SetRatePercent(item.RatePercent.StringFixed(4)).SetNillableEffectiveFrom(item.EffectiveFrom).SetNillableEffectiveTo(item.EffectiveTo).SetEnabled(item.Enabled).SetNillableNote(item.Note).SetVersion(1).Save(ctx)
	if ent.IsConstraintError(err) {
		return rollback(biz.ErrCommissionRuleConflict)
	}
	if err != nil {
		return rollback(err)
	}
	if err = writeAudit(ctx, tx.AuditLog, audit); err != nil {
		return rollback(err)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return commissionRuleToBiz(x)
}
func (r *commissionRepo) UpdateRule(ctx context.Context, org uuid.UUID, in biz.UpdateCommissionRuleInput, audit *biz.AuditEvent) (*biz.FinanceCommissionRule, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	rollback := func(err error) (*biz.FinanceCommissionRule, error) { _ = tx.Rollback(); return nil, err }
	x, err := tx.FinanceCommissionRule.Query().Where(rule.IDEQ(in.ID), rule.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return rollback(biz.ErrCommissionRuleNotFound)
	}
	if err != nil {
		return rollback(err)
	}
	if x.Version != in.ExpectedVersion {
		return rollback(biz.ErrCommissionRuleConflict)
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
	updated, err := u.Save(ctx)
	if ent.IsConstraintError(err) {
		return rollback(biz.ErrCommissionRuleConflict)
	}
	if err != nil {
		return rollback(err)
	}
	if err = writeAudit(ctx, tx.AuditLog, audit); err != nil {
		return rollback(err)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return commissionRuleToBiz(updated)
}

func (r *commissionRepo) List(ctx context.Context, org uuid.UUID, f biz.CommissionFilter) (*biz.CommissionListResult, error) {
	p := []predicate.FinanceCommission{commission.OrganizationIDEQ(org)}
	if f.Keyword != "" {
		p = append(p, commission.Or(commission.CommissionNoContainsFold(f.Keyword), commission.VerificationNoContainsFold(f.Keyword), commission.EmployeeNameContainsFold(f.Keyword)))
	}
	if f.Status != "" {
		p = append(p, commission.StatusEQ(commission.Status(f.Status)))
	}
	q := r.data.db.FinanceCommission.Query().Where(p...)
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	xs, err := q.Order(commission.ByCreatedAt(entsql.OrderDesc())).Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.CommissionListResult{Items: make([]*biz.FinanceCommission, 0, len(xs)), Total: int64(total)}
	for _, x := range xs {
		item, err := commissionToBiz(x)
		if err != nil {
			return nil, err
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}

func (r *commissionRepo) GetByKey(ctx context.Context, org uuid.UUID, key string) (*biz.FinanceCommission, error) {
	x, err := r.data.db.FinanceCommission.Query().Where(commission.OrganizationIDEQ(org), commission.IdempotencyKeyEQ(key)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return commissionToBiz(x)
}

func (r *commissionRepo) Create(ctx context.Context, org, actor uuid.UUID, c *biz.FinanceCommission, audit *biz.AuditEvent) (*biz.FinanceCommission, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	rollback := func(err error) (*biz.FinanceCommission, error) { _ = tx.Rollback(); return nil, err }
	v, err := tx.FinanceVerification.Query().Where(verification.IDEQ(c.VerificationID), verification.OrganizationIDEQ(org)).WithAllocations(func(q *ent.FinanceVerificationAllocationQuery) { q.Where(allocation.ActiveEQ(true)) }).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) || (err == nil && (v.Status != verification.StatusACTIVE || v.Direction != verification.DirectionRECEIVABLE)) {
		return rollback(biz.ErrCommissionSource)
	}
	if err != nil {
		return rollback(err)
	}
	ruleItem, err := tx.FinanceCommissionRule.Query().Where(rule.IDEQ(c.RuleID), rule.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return rollback(biz.ErrCommissionRuleNotFound)
	}
	if err != nil {
		return rollback(err)
	}
	if !ruleItem.Enabled || (ruleItem.EffectiveFrom != nil && v.VerificationDate < *ruleItem.EffectiveFrom) || (ruleItem.EffectiveTo != nil && v.VerificationDate > *ruleItem.EffectiveTo) {
		return rollback(biz.ErrCommissionRuleInvalid)
	}
	rate, err := decimal.NewFromString(ruleItem.RatePercent)
	if err != nil {
		return rollback(err)
	}
	c.RuleName = ruleItem.Name
	c.PersonnelRole = biz.CommissionPersonnelRole(ruleItem.PersonnelRole)
	c.CalculationBasis = biz.CommissionCalculationBasis(ruleItem.CalculationBasis)
	c.RatePercent = rate
	employee, err := tx.User.Query().Where(user.IDEQ(c.EmployeeID), user.EnabledEQ(true), user.HasMembershipsWith(membership.OrganizationIDEQ(org), membership.EnabledEQ(true))).Only(ctx)
	if ent.IsNotFound(err) {
		return rollback(biz.ErrCommissionInvalid)
	}
	if err != nil {
		return rollback(err)
	}
	if len(v.Edges.Allocations) == 0 {
		return rollback(biz.ErrCommissionSource)
	}
	hasActive, err := tx.FinanceCommission.Query().Where(commission.VerificationIDEQ(c.VerificationID), commission.EmployeeIDEQ(c.EmployeeID), commission.RuleIDEQ(c.RuleID), commission.StatusNEQ(commission.StatusCANCELLED)).Exist(ctx)
	if err != nil {
		return rollback(err)
	}
	if hasActive {
		return rollback(biz.ErrCommissionDuplicate)
	}
	billIDs := make([]uuid.UUID, 0, len(v.Edges.Allocations))
	allocationByBill := make(map[uuid.UUID]decimal.Decimal)
	for _, a := range v.Edges.Allocations {
		amount, parseErr := decimal.NewFromString(a.Amount)
		if parseErr != nil {
			return rollback(parseErr)
		}
		billIDs = append(billIDs, a.BillID)
		allocationByBill[a.BillID] = allocationByBill[a.BillID].Add(amount)
	}
	bills, err := tx.FinanceBill.Query().Where(bill.IDIn(billIDs...)).WithLines().All(ctx)
	if err != nil {
		return rollback(err)
	}
	orderRealized := make(map[uuid.UUID]decimal.Decimal)
	baseCurrency := ""
	for _, b := range bills {
		total, parseErr := decimal.NewFromString(b.TotalAmount)
		if parseErr != nil || !total.IsPositive() {
			return rollback(biz.ErrCommissionSource)
		}
		ratio := allocationByBill[b.ID].Div(total)
		for _, line := range b.Edges.Lines {
			if !line.Active {
				continue
			}
			base, parseErr := decimal.NewFromString(line.BaseCurrencyAmount)
			if parseErr != nil {
				return rollback(parseErr)
			}
			if baseCurrency == "" {
				baseCurrency = line.BaseCurrency
			} else if baseCurrency != line.BaseCurrency {
				return rollback(biz.ErrCommissionSource)
			}
			orderRealized[line.OrderID] = orderRealized[line.OrderID].Add(base.Mul(ratio))
		}
	}
	orderIDs := make([]uuid.UUID, 0, len(orderRealized))
	for id := range orderRealized {
		orderIDs = append(orderIDs, id)
	}
	if len(orderIDs) == 0 {
		return rollback(biz.ErrCommissionSource)
	}
	assignments, err := tx.OrderPersonnel.Query().Where(personnel.OrganizationIDEQ(org), personnel.OrderIDIn(orderIDs...), personnel.UserIDEQ(c.EmployeeID), personnel.RoleEQ(personnel.Role(c.PersonnelRole))).All(ctx)
	if err != nil {
		return rollback(err)
	}
	eligibleOrders := make(map[uuid.UUID]struct{}, len(assignments))
	for _, assignment := range assignments {
		eligibleOrders[assignment.OrderID] = struct{}{}
	}
	if len(eligibleOrders) == 0 {
		return rollback(biz.ErrCommissionEmployeeRole)
	}
	fees, err := tx.OrderFee.Query().Where(fee.OrderIDIn(orderIDs...), fee.StatusIn(fee.StatusCONFIRMED, fee.StatusBILLED)).All(ctx)
	if err != nil {
		return rollback(err)
	}
	receivable, payable := make(map[uuid.UUID]decimal.Decimal), make(map[uuid.UUID]decimal.Decimal)
	for _, f := range fees {
		amount, parseErr := decimal.NewFromString(f.BaseCurrencyAmount)
		if parseErr != nil {
			return rollback(parseErr)
		}
		if f.BaseCurrency != baseCurrency {
			return rollback(biz.ErrCommissionSource)
		}
		if f.Direction == fee.DirectionRECEIVABLE {
			receivable[f.OrderID] = receivable[f.OrderID].Add(amount)
		} else {
			payable[f.OrderID] = payable[f.OrderID].Add(amount)
		}
	}
	for orderID, realized := range orderRealized {
		if _, eligible := eligibleOrders[orderID]; !eligible {
			continue
		}
		if !receivable[orderID].IsPositive() {
			return rollback(biz.ErrCommissionSource)
		}
		cost := realized.Mul(payable[orderID]).Div(receivable[orderID])
		c.RealizedRevenue = c.RealizedRevenue.Add(realized)
		c.AllocatedCost = c.AllocatedCost.Add(cost)
	}
	c.BaseCurrency = baseCurrency
	c.RealizedRevenue = c.RealizedRevenue.Round(8)
	c.AllocatedCost = c.AllocatedCost.Round(8)
	c.RealizedProfit = c.RealizedRevenue.Sub(c.AllocatedCost).Round(8)
	c.CommissionAmount, err = biz.CalculateCommissionAmount(c.RealizedRevenue, c.RealizedProfit, c.RatePercent, c.CalculationBasis)
	if err != nil {
		return rollback(err)
	}
	c.VerificationNo = v.VerificationNo
	c.EmployeeName = employee.DisplayName
	_, err = tx.FinanceCommission.Create().SetID(c.ID).SetOrganizationID(org).SetCommissionNo(c.CommissionNo).SetIdempotencyKey(c.IdempotencyKey).SetVerificationID(c.VerificationID).SetVerificationNo(c.VerificationNo).SetEmployeeID(c.EmployeeID).SetEmployeeName(c.EmployeeName).SetRuleID(c.RuleID).SetRuleName(c.RuleName).SetPersonnelRole(string(c.PersonnelRole)).SetCalculationBasis(string(c.CalculationBasis)).SetStatus(commission.StatusDRAFT).SetBaseCurrency(c.BaseCurrency).SetRealizedRevenue(c.RealizedRevenue.StringFixed(8)).SetAllocatedCost(c.AllocatedCost.StringFixed(8)).SetRealizedProfit(c.RealizedProfit.StringFixed(8)).SetRatePercent(c.RatePercent.StringFixed(4)).SetCommissionAmount(c.CommissionAmount.StringFixed(8)).SetNillableNote(c.Note).SetVersion(1).Save(ctx)
	if ent.IsConstraintError(err) {
		return rollback(biz.ErrCommissionDuplicate)
	}
	if err != nil {
		return rollback(err)
	}
	if err = writeAudit(ctx, tx.AuditLog, audit); err != nil {
		return rollback(err)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.get(ctx, org, c.ID)
}

func (r *commissionRepo) Transition(ctx context.Context, org, id, actor uuid.UUID, version uint64, target biz.CommissionStatus, reason string, audit *biz.AuditEvent) (*biz.FinanceCommission, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	rollback := func(err error) (*biz.FinanceCommission, error) { _ = tx.Rollback(); return nil, err }
	x, err := tx.FinanceCommission.Query().Where(commission.IDEQ(id), commission.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return rollback(biz.ErrCommissionNotFound)
	}
	if err != nil {
		return rollback(err)
	}
	if x.Version != version {
		return rollback(biz.ErrCommissionTransition)
	}
	now := time.Now()
	update := tx.FinanceCommission.UpdateOneID(id).SetVersion(version + 1)
	switch target {
	case biz.CommissionConfirmed:
		if x.Status != commission.StatusDRAFT {
			return rollback(biz.ErrCommissionTransition)
		}
		update.SetStatus(commission.StatusCONFIRMED).SetConfirmedAt(now).SetConfirmedBy(actor)
	case biz.CommissionPaid:
		if x.Status != commission.StatusCONFIRMED {
			return rollback(biz.ErrCommissionTransition)
		}
		update.SetStatus(commission.StatusPAID).SetPaidAt(now).SetPaidBy(actor)
	case biz.CommissionCancelled:
		if x.Status != commission.StatusDRAFT && x.Status != commission.StatusCONFIRMED {
			return rollback(biz.ErrCommissionTransition)
		}
		update.SetStatus(commission.StatusCANCELLED).SetCancelledAt(now).SetCancelledBy(actor).SetCancellationReason(reason)
	default:
		return rollback(biz.ErrCommissionInvalid)
	}
	if _, err = update.Save(ctx); err != nil {
		return rollback(err)
	}
	if err = writeAudit(ctx, tx.AuditLog, audit); err != nil {
		return rollback(err)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.get(ctx, org, id)
}

func (r *commissionRepo) get(ctx context.Context, org, id uuid.UUID) (*biz.FinanceCommission, error) {
	x, err := r.data.db.FinanceCommission.Query().Where(commission.IDEQ(id), commission.OrganizationIDEQ(org)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, biz.ErrCommissionNotFound
	}
	if err != nil {
		return nil, err
	}
	return commissionToBiz(x)
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
	result := &biz.FinanceCommission{ID: x.ID, OrganizationID: x.OrganizationID, CommissionNo: x.CommissionNo, IdempotencyKey: x.IdempotencyKey, VerificationID: x.VerificationID, VerificationNo: x.VerificationNo, EmployeeID: x.EmployeeID, EmployeeName: x.EmployeeName, Status: biz.CommissionStatus(x.Status), BaseCurrency: x.BaseCurrency, RealizedRevenue: revenue, AllocatedCost: cost, RealizedProfit: profit, RatePercent: rate, CommissionAmount: amount, Note: x.Note, Version: x.Version, ConfirmedAt: x.ConfirmedAt, PaidAt: x.PaidAt, CancelledAt: x.CancelledAt, CancellationReason: x.CancellationReason, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt}
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

func commissionRuleToBiz(x *ent.FinanceCommissionRule) (*biz.FinanceCommissionRule, error) {
	rate, err := decimal.NewFromString(x.RatePercent)
	if err != nil {
		return nil, err
	}
	return &biz.FinanceCommissionRule{ID: x.ID, OrganizationID: x.OrganizationID, Name: x.Name, PersonnelRole: biz.CommissionPersonnelRole(x.PersonnelRole), CalculationBasis: biz.CommissionCalculationBasis(x.CalculationBasis), RatePercent: rate, EffectiveFrom: x.EffectiveFrom, EffectiveTo: x.EffectiveTo, Enabled: x.Enabled, Note: x.Note, Version: x.Version, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt}, nil
}

var _ biz.CommissionRepo = (*commissionRepo)(nil)
