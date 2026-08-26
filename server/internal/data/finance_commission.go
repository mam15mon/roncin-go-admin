package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	commissionline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
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
	items, err := r.data.db.OrderPersonnel.Query().Where(personnel.OrderIDIn(orderIDs...), personnel.RoleEQ(personnel.Role(ruleItem.PersonnelRole)), personnel.HasUserWith(user.EnabledEQ(true))).WithUser().All(ctx)
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
	x, err := r.data.db.FinanceCommission.Query().Where(commission.OrganizationIDEQ(org), commission.IdempotencyKeyEQ(key)).WithLines(func(q *ent.FinanceCommissionLineQuery) {
		q.Order(commissionline.ByOrderNo(), commissionline.ByOrderID())
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
	personnel     *ent.OrderPersonnelClient
	fees          *ent.OrderFeeClient
}

func commissionStoreFromClient(client *ent.Client) commissionCalculationStore {
	return commissionCalculationStore{verifications: client.FinanceVerification, rules: client.FinanceCommissionRule, users: client.User, bills: client.FinanceBill, personnel: client.OrderPersonnel, fees: client.OrderFee}
}

func commissionStoreFromTx(tx *ent.Tx) commissionCalculationStore {
	return commissionCalculationStore{verifications: tx.FinanceVerification, rules: tx.FinanceCommissionRule, users: tx.User, bills: tx.FinanceBill, personnel: tx.OrderPersonnel, fees: tx.OrderFee}
}

func (r *commissionRepo) Preview(ctx context.Context, org, verificationID, employeeID, ruleID uuid.UUID) (*biz.CommissionCalculation, error) {
	return calculateCommission(ctx, commissionStoreFromClient(r.data.db), org, verificationID, employeeID, ruleID, false)
}

func calculateCommission(ctx context.Context, store commissionCalculationStore, org, verificationID, employeeID, ruleID uuid.UUID, lock bool) (*biz.CommissionCalculation, error) {
	vq := store.verifications.Query().Where(verification.IDEQ(verificationID), verification.OrganizationIDEQ(org)).WithAllocations(func(q *ent.FinanceVerificationAllocationQuery) {
		q.Where(allocation.ActiveEQ(true))
	})
	if lock {
		vq.ForUpdate()
	}
	v, err := vq.Only(ctx)
	if ent.IsNotFound(err) || (err == nil && (v.Status != verification.StatusACTIVE || v.Direction != verification.DirectionRECEIVABLE)) {
		return nil, biz.ErrCommissionSource
	}
	if err != nil {
		return nil, err
	}
	rq := store.rules.Query().Where(rule.IDEQ(ruleID), rule.OrganizationIDEQ(org))
	if lock {
		rq.ForUpdate()
	}
	ruleItem, err := rq.Only(ctx)
	if ent.IsNotFound(err) {
		return nil, biz.ErrCommissionRuleNotFound
	}
	if err != nil {
		return nil, err
	}
	if !ruleItem.Enabled || (ruleItem.EffectiveFrom != nil && v.VerificationDate < *ruleItem.EffectiveFrom) || (ruleItem.EffectiveTo != nil && v.VerificationDate > *ruleItem.EffectiveTo) {
		return nil, biz.ErrCommissionRuleInvalid
	}
	rate, err := decimal.NewFromString(ruleItem.RatePercent)
	if err != nil {
		return nil, err
	}
	uq := store.users.Query().Where(user.IDEQ(employeeID), user.EnabledEQ(true))
	if lock {
		uq.ForUpdate()
	}
	employee, err := uq.Only(ctx)
	if ent.IsNotFound(err) {
		return nil, biz.ErrCommissionInvalid
	}
	if err != nil {
		return nil, err
	}
	if len(v.Edges.Allocations) == 0 {
		return nil, biz.ErrCommissionSource
	}

	fingerprintParts := []string{
		fmt.Sprintf("calculation|%s", biz.CommissionCalculationVersion),
		fmt.Sprintf("verification|%s|%s|%s|%s|%s|%d", v.ID, v.VerificationNo, v.Status, v.Direction, v.VerificationDate, v.Version),
		fmt.Sprintf("rule|%s|%s|%s|%s|%s|%d|%t|%s|%s", ruleItem.ID, ruleItem.Name, ruleItem.PersonnelRole, ruleItem.CalculationBasis, ruleItem.RatePercent, ruleItem.Version, ruleItem.Enabled, optionalStringValue(ruleItem.EffectiveFrom), optionalStringValue(ruleItem.EffectiveTo)),
		fmt.Sprintf("employee|%s|%s|%t", employee.ID, employee.DisplayName, employee.Enabled),
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
		fingerprintParts = append(fingerprintParts, fmt.Sprintf("allocation|%s|%s|%s|%s|%s|%t", item.ID, item.BillID, item.CashflowID, item.Amount, item.BillNo, item.Active))
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
	orderNos := make(map[uuid.UUID]string)
	baseCurrency := ""
	for _, billItem := range bills {
		total, parseErr := decimal.NewFromString(billItem.TotalAmount)
		if parseErr != nil || !total.IsPositive() {
			return nil, biz.ErrCommissionSource
		}
		ratio := allocationByBill[billItem.ID].Div(total)
		fingerprintParts = append(fingerprintParts, fmt.Sprintf("bill|%s|%s|%s|%s|%s|%d", billItem.ID, billItem.BillNo, billItem.Status, billItem.TotalAmount, billItem.BaseCurrencyAmount, billItem.Version))
		for _, line := range billItem.Edges.Lines {
			if !line.Active {
				continue
			}
			base, parseErr := decimal.NewFromString(line.BaseCurrencyAmount)
			if parseErr != nil {
				return nil, parseErr
			}
			if baseCurrency == "" {
				baseCurrency = line.BaseCurrency
			} else if baseCurrency != line.BaseCurrency {
				return nil, biz.ErrCommissionSource
			}
			orderRealized[line.OrderID] = orderRealized[line.OrderID].Add(base.Mul(ratio))
			orderNos[line.OrderID] = line.OrderNo
			fingerprintParts = append(fingerprintParts, fmt.Sprintf("bill_line|%s|%s|%s|%s|%s|%s|%t", line.ID, line.OrderFeeID, line.OrderID, line.TotalAmount, line.BaseCurrencyAmount, line.BaseCurrency, line.Active))
		}
	}
	orderIDs := make([]uuid.UUID, 0, len(orderRealized))
	for id := range orderRealized {
		orderIDs = append(orderIDs, id)
	}
	if len(orderIDs) == 0 || baseCurrency == "" {
		return nil, biz.ErrCommissionSource
	}
	pq := store.personnel.Query().Where(personnel.OrderIDIn(orderIDs...), personnel.UserIDEQ(employeeID), personnel.RoleEQ(personnel.Role(ruleItem.PersonnelRole)))
	if lock {
		pq.ForUpdate()
	}
	assignments, err := pq.All(ctx)
	if err != nil {
		return nil, err
	}
	assignmentByOrder := make(map[uuid.UUID]*ent.OrderPersonnel, len(assignments))
	eligibleOrderIDs := make([]uuid.UUID, 0, len(assignments))
	for _, assignment := range assignments {
		assignmentByOrder[assignment.OrderID] = assignment
		eligibleOrderIDs = append(eligibleOrderIDs, assignment.OrderID)
		fingerprintParts = append(fingerprintParts, fmt.Sprintf("personnel|%s|%s|%s|%s|%s|%s", assignment.ID, assignment.OrderID, assignment.UserID, assignment.OrganizationID, assignment.Role, assignment.AssignedAt.UTC().Format(time.RFC3339Nano)))
	}
	if len(eligibleOrderIDs) == 0 {
		return nil, biz.ErrCommissionEmployeeRole
	}
	fq := store.fees.Query().Where(fee.OrderIDIn(eligibleOrderIDs...), fee.StatusIn(fee.StatusCONFIRMED, fee.StatusBILLED))
	if lock {
		fq.ForUpdate()
	}
	fees, err := fq.All(ctx)
	if err != nil {
		return nil, err
	}
	receivable, payable := make(map[uuid.UUID]decimal.Decimal), make(map[uuid.UUID]decimal.Decimal)
	for _, feeItem := range fees {
		amount, parseErr := decimal.NewFromString(feeItem.BaseCurrencyAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		if feeItem.BaseCurrency != baseCurrency {
			return nil, biz.ErrCommissionSource
		}
		if feeItem.Direction == fee.DirectionRECEIVABLE {
			receivable[feeItem.OrderID] = receivable[feeItem.OrderID].Add(amount)
		} else {
			payable[feeItem.OrderID] = payable[feeItem.OrderID].Add(amount)
		}
		fingerprintParts = append(fingerprintParts, fmt.Sprintf("fee|%s|%s|%s|%s|%s|%s|%d", feeItem.ID, feeItem.OrderID, feeItem.Direction, feeItem.Status, feeItem.BaseCurrencyAmount, feeItem.BaseCurrency, feeItem.Version))
	}

	result := &biz.CommissionCalculation{
		VerificationID: verificationID, VerificationNo: v.VerificationNo,
		EmployeeID: employeeID, EmployeeName: employee.DisplayName,
		RuleID: ruleID, RuleName: ruleItem.Name, PersonnelRole: biz.CommissionPersonnelRole(ruleItem.PersonnelRole),
		CalculationBasis: biz.CommissionCalculationBasis(ruleItem.CalculationBasis), RuleVersion: ruleItem.Version,
		CalculationVersion: biz.CommissionCalculationVersion, BaseCurrency: baseCurrency, RatePercent: rate,
		Lines: make([]*biz.FinanceCommissionLine, 0, len(eligibleOrderIDs)),
	}
	for orderID, realizedValue := range orderRealized {
		assignment, eligible := assignmentByOrder[orderID]
		if !eligible {
			continue
		}
		if !receivable[orderID].IsPositive() {
			return nil, biz.ErrCommissionSource
		}
		realized := realizedValue.Round(8)
		cost, profit, amount, calculateErr := biz.CalculateCommissionLine(realized, receivable[orderID], payable[orderID], rate, result.CalculationBasis)
		if calculateErr != nil {
			return nil, calculateErr
		}
		line := &biz.FinanceCommissionLine{
			OrderID: orderID, OrderNo: orderNos[orderID], PersonnelAssignmentID: assignment.ID,
			PersonnelOrganizationID: assignment.OrganizationID, PersonnelAssignedAt: assignment.AssignedAt,
			EmployeeID: employeeID, EmployeeName: employee.DisplayName,
			PersonnelRole: result.PersonnelRole, CalculationBasis: result.CalculationBasis, BaseCurrency: baseCurrency,
			RealizedRevenue: realized, AllocatedCost: cost, RealizedProfit: profit, RatePercent: rate, CommissionAmount: amount,
		}
		result.Lines = append(result.Lines, line)
		result.RealizedRevenue = result.RealizedRevenue.Add(realized)
		result.AllocatedCost = result.AllocatedCost.Add(cost)
		result.RealizedProfit = result.RealizedProfit.Add(profit)
		result.CommissionAmount = result.CommissionAmount.Add(amount)
		fingerprintParts = append(fingerprintParts, fmt.Sprintf("result_line|%s|%s|%s|%s|%s", orderID, realized.StringFixed(8), cost.StringFixed(8), profit.StringFixed(8), amount.StringFixed(8)))
	}
	if len(result.Lines) == 0 || !result.RealizedRevenue.IsPositive() {
		return nil, biz.ErrCommissionSource
	}
	sort.Slice(result.Lines, func(i, j int) bool {
		if result.Lines[i].OrderNo == result.Lines[j].OrderNo {
			return result.Lines[i].OrderID.String() < result.Lines[j].OrderID.String()
		}
		return result.Lines[i].OrderNo < result.Lines[j].OrderNo
	})
	result.RealizedRevenue = result.RealizedRevenue.Round(8)
	result.AllocatedCost = result.AllocatedCost.Round(8)
	result.RealizedProfit = result.RealizedProfit.Round(8)
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
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	rollback := func(err error) (*biz.FinanceCommission, error) { _ = tx.Rollback(); return nil, err }
	calculation, err := calculateCommission(ctx, commissionStoreFromTx(tx), org, c.VerificationID, c.EmployeeID, c.RuleID, true)
	if err != nil {
		return rollback(err)
	}
	hasActive, err := tx.FinanceCommission.Query().Where(commission.VerificationIDEQ(c.VerificationID), commission.EmployeeIDEQ(c.EmployeeID), commission.RuleIDEQ(c.RuleID), commission.StatusNEQ(commission.StatusCANCELLED)).Exist(ctx)
	if err != nil {
		return rollback(err)
	}
	if hasActive {
		return rollback(biz.ErrCommissionDuplicate)
	}
	c.VerificationNo, c.EmployeeName, c.RuleName = calculation.VerificationNo, calculation.EmployeeName, calculation.RuleName
	c.PersonnelRole, c.CalculationBasis = calculation.PersonnelRole, calculation.CalculationBasis
	c.RuleVersion, c.CalculationVersion, c.SourceFingerprint = calculation.RuleVersion, calculation.CalculationVersion, calculation.SourceFingerprint
	c.BaseCurrency, c.RatePercent = calculation.BaseCurrency, calculation.RatePercent
	c.RealizedRevenue, c.AllocatedCost, c.RealizedProfit, c.CommissionAmount = calculation.RealizedRevenue, calculation.AllocatedCost, calculation.RealizedProfit, calculation.CommissionAmount
	_, err = tx.FinanceCommission.Create().SetID(c.ID).SetOrganizationID(org).SetCommissionNo(c.CommissionNo).SetIdempotencyKey(c.IdempotencyKey).SetVerificationID(c.VerificationID).SetVerificationNo(c.VerificationNo).SetEmployeeID(c.EmployeeID).SetEmployeeName(c.EmployeeName).SetRuleID(c.RuleID).SetRuleName(c.RuleName).SetPersonnelRole(string(c.PersonnelRole)).SetCalculationBasis(string(c.CalculationBasis)).SetRuleVersion(c.RuleVersion).SetCalculationVersion(c.CalculationVersion).SetSourceFingerprint(c.SourceFingerprint).SetStatus(commission.StatusDRAFT).SetBaseCurrency(c.BaseCurrency).SetRealizedRevenue(c.RealizedRevenue.StringFixed(8)).SetAllocatedCost(c.AllocatedCost.StringFixed(8)).SetRealizedProfit(c.RealizedProfit.StringFixed(8)).SetRatePercent(c.RatePercent.StringFixed(4)).SetCommissionAmount(c.CommissionAmount.StringFixed(8)).SetNillableNote(c.Note).SetVersion(1).Save(ctx)
	if ent.IsConstraintError(err) {
		return rollback(biz.ErrCommissionDuplicate)
	}
	if err != nil {
		return rollback(err)
	}
	lineBuilders := make([]*ent.FinanceCommissionLineCreate, 0, len(calculation.Lines))
	for _, line := range calculation.Lines {
		line.ID = uuid.Must(uuid.NewV7())
		line.OrganizationID = org
		line.CommissionID = c.ID
		lineBuilders = append(lineBuilders, tx.FinanceCommissionLine.Create().SetID(line.ID).SetOrganizationID(org).SetCommissionID(c.ID).SetOrderID(line.OrderID).SetOrderNo(line.OrderNo).SetPersonnelAssignmentID(line.PersonnelAssignmentID).SetPersonnelOrganizationID(line.PersonnelOrganizationID).SetPersonnelAssignedAt(line.PersonnelAssignedAt).SetEmployeeID(line.EmployeeID).SetEmployeeName(line.EmployeeName).SetPersonnelRole(string(line.PersonnelRole)).SetCalculationBasis(string(line.CalculationBasis)).SetBaseCurrency(line.BaseCurrency).SetRealizedRevenue(line.RealizedRevenue.StringFixed(8)).SetAllocatedCost(line.AllocatedCost.StringFixed(8)).SetRealizedProfit(line.RealizedProfit.StringFixed(8)).SetRatePercent(line.RatePercent.StringFixed(4)).SetCommissionAmount(line.CommissionAmount.StringFixed(8)))
	}
	if _, err = tx.FinanceCommissionLine.CreateBulk(lineBuilders...).Save(ctx); err != nil {
		return rollback(err)
	}
	if err = writeAudit(ctx, tx.AuditLog, audit); err != nil {
		return rollback(err)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, org, c.ID)
}

func (r *commissionRepo) Transition(ctx context.Context, org, id, actor uuid.UUID, version uint64, target biz.CommissionStatus, reason string, audit *biz.AuditEvent) (*biz.FinanceCommission, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	rollback := func(err error) (*biz.FinanceCommission, error) { _ = tx.Rollback(); return nil, err }
	if target == biz.CommissionConfirmed {
		snapshot, lookupErr := tx.FinanceCommission.Query().Where(commission.IDEQ(id), commission.OrganizationIDEQ(org)).Only(ctx)
		if ent.IsNotFound(lookupErr) {
			return rollback(biz.ErrCommissionNotFound)
		}
		if lookupErr != nil {
			return rollback(lookupErr)
		}
		current, calculateErr := calculateCommission(ctx, commissionStoreFromTx(tx), org, snapshot.VerificationID, snapshot.EmployeeID, valueOrNilUUID(snapshot.RuleID), true)
		if calculateErr != nil {
			return rollback(biz.ErrCommissionSourceChanged)
		}
		if snapshot.SourceFingerprint == "" || snapshot.SourceFingerprint != current.SourceFingerprint {
			return rollback(biz.ErrCommissionSourceChanged)
		}
	}
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
	}).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, biz.ErrCommissionNotFound
	}
	if err != nil {
		return nil, err
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
	result := &biz.FinanceCommission{ID: x.ID, OrganizationID: x.OrganizationID, CommissionNo: x.CommissionNo, IdempotencyKey: x.IdempotencyKey, VerificationID: x.VerificationID, VerificationNo: x.VerificationNo, EmployeeID: x.EmployeeID, EmployeeName: x.EmployeeName, Status: biz.CommissionStatus(x.Status), BaseCurrency: x.BaseCurrency, RealizedRevenue: revenue, AllocatedCost: cost, RealizedProfit: profit, RatePercent: rate, CommissionAmount: amount, Note: x.Note, Version: x.Version, RuleVersion: x.RuleVersion, CalculationVersion: x.CalculationVersion, SourceFingerprint: x.SourceFingerprint, ConfirmedAt: x.ConfirmedAt, PaidAt: x.PaidAt, CancelledAt: x.CancelledAt, CancellationReason: x.CancellationReason, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt}
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
	return &biz.FinanceCommissionLine{ID: x.ID, OrganizationID: x.OrganizationID, CommissionID: x.CommissionID, OrderID: x.OrderID, OrderNo: x.OrderNo, PersonnelAssignmentID: x.PersonnelAssignmentID, PersonnelOrganizationID: x.PersonnelOrganizationID, PersonnelAssignedAt: x.PersonnelAssignedAt, EmployeeID: x.EmployeeID, EmployeeName: x.EmployeeName, PersonnelRole: biz.CommissionPersonnelRole(x.PersonnelRole), CalculationBasis: biz.CommissionCalculationBasis(x.CalculationBasis), BaseCurrency: x.BaseCurrency, RealizedRevenue: revenue, AllocatedCost: cost, RealizedProfit: profit, RatePercent: rate, CommissionAmount: amount, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt}, nil
}

func commissionRuleToBiz(x *ent.FinanceCommissionRule) (*biz.FinanceCommissionRule, error) {
	rate, err := decimal.NewFromString(x.RatePercent)
	if err != nil {
		return nil, err
	}
	return &biz.FinanceCommissionRule{ID: x.ID, OrganizationID: x.OrganizationID, Name: x.Name, PersonnelRole: biz.CommissionPersonnelRole(x.PersonnelRole), CalculationBasis: biz.CommissionCalculationBasis(x.CalculationBasis), RatePercent: rate, EffectiveFrom: x.EffectiveFrom, EffectiveTo: x.EffectiveTo, Enabled: x.Enabled, Note: x.Note, Version: x.Version, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt}, nil
}

var _ biz.CommissionRepo = (*commissionRepo)(nil)
