package data

import (
	"context"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	bill "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	verification "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	allocation "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
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
	hasActive, err := tx.FinanceCommission.Query().Where(commission.VerificationIDEQ(c.VerificationID), commission.EmployeeIDEQ(c.EmployeeID), commission.StatusNEQ(commission.StatusCANCELLED)).Exist(ctx)
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
	commissionBase := c.RealizedProfit
	if commissionBase.IsNegative() {
		commissionBase = decimal.Zero
	}
	c.CommissionAmount = commissionBase.Mul(c.RatePercent).Div(decimal.NewFromInt(100)).Round(8)
	c.VerificationNo = v.VerificationNo
	c.EmployeeName = employee.DisplayName
	_, err = tx.FinanceCommission.Create().SetID(c.ID).SetOrganizationID(org).SetCommissionNo(c.CommissionNo).SetIdempotencyKey(c.IdempotencyKey).SetVerificationID(c.VerificationID).SetVerificationNo(c.VerificationNo).SetEmployeeID(c.EmployeeID).SetEmployeeName(c.EmployeeName).SetStatus(commission.StatusDRAFT).SetBaseCurrency(c.BaseCurrency).SetRealizedRevenue(c.RealizedRevenue.StringFixed(8)).SetAllocatedCost(c.AllocatedCost.StringFixed(8)).SetRealizedProfit(c.RealizedProfit.StringFixed(8)).SetRatePercent(c.RatePercent.StringFixed(4)).SetCommissionAmount(c.CommissionAmount.StringFixed(8)).SetNillableNote(c.Note).SetVersion(1).Save(ctx)
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
	return &biz.FinanceCommission{ID: x.ID, OrganizationID: x.OrganizationID, CommissionNo: x.CommissionNo, IdempotencyKey: x.IdempotencyKey, VerificationID: x.VerificationID, VerificationNo: x.VerificationNo, EmployeeID: x.EmployeeID, EmployeeName: x.EmployeeName, Status: biz.CommissionStatus(x.Status), BaseCurrency: x.BaseCurrency, RealizedRevenue: revenue, AllocatedCost: cost, RealizedProfit: profit, RatePercent: rate, CommissionAmount: amount, Note: x.Note, Version: x.Version, ConfirmedAt: x.ConfirmedAt, PaidAt: x.PaidAt, CancelledAt: x.CancelledAt, CancellationReason: x.CancellationReason, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt}, nil
}

var _ biz.CommissionRepo = (*commissionRepo)(nil)
