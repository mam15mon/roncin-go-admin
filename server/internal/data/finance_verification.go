package data

import (
	"context"
	"sort"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	bill "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	cash "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	ver "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	alloc "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/shopspring/decimal"
)

type verificationRepo struct{ data *Data }

func NewVerificationRepo(d *Data) biz.VerificationRepo { return &verificationRepo{d} }
func (r *verificationRepo) withAll(q *ent.FinanceVerificationQuery) *ent.FinanceVerificationQuery {
	return q.WithAllocations(func(x *ent.FinanceVerificationAllocationQuery) { x.Order(alloc.ByCreatedAt()) })
}
func (r *verificationRepo) List(ctx context.Context, org uuid.UUID, f biz.VerificationFilter) (*biz.VerificationListResult, error) {
	p := []predicate.FinanceVerification{ver.OrganizationIDEQ(org)}
	if f.Keyword != "" {
		p = append(p, ver.Or(ver.VerificationNoContainsFold(f.Keyword), ver.SettlementPartyNameContainsFold(f.Keyword), ver.HasAllocationsWith(alloc.Or(alloc.BillNoContainsFold(f.Keyword), alloc.CashflowNoContainsFold(f.Keyword)))))
	}
	if f.Status != "" {
		p = append(p, ver.StatusEQ(ver.Status(f.Status)))
	}
	q := r.data.db.FinanceVerification.Query().Where(p...)
	n, e := q.Clone().Count(ctx)
	if e != nil {
		return nil, e
	}
	xs, e := r.withAll(q).Order(ver.ByVerificationDate(entsql.OrderDesc()), ver.ByCreatedAt(entsql.OrderDesc())).Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).All(ctx)
	if e != nil {
		return nil, e
	}
	out := &biz.VerificationListResult{Items: make([]*biz.FinanceVerification, 0, len(xs)), Total: int64(n)}
	for _, x := range xs {
		v, e := verificationToBiz(x)
		if e != nil {
			return nil, e
		}
		out.Items = append(out.Items, v)
	}
	return out, nil
}
func (r *verificationRepo) Get(ctx context.Context, org, id uuid.UUID) (*biz.FinanceVerification, error) {
	x, e := r.withAll(r.data.db.FinanceVerification.Query()).Where(ver.IDEQ(id), ver.OrganizationIDEQ(org)).Only(ctx)
	if ent.IsNotFound(e) {
		return nil, biz.ErrVerificationNotFound
	}
	if e != nil {
		return nil, e
	}
	return verificationToBiz(x)
}
func (r *verificationRepo) GetByKey(ctx context.Context, org uuid.UUID, key string) (*biz.FinanceVerification, error) {
	x, e := r.withAll(r.data.db.FinanceVerification.Query()).Where(ver.OrganizationIDEQ(org), ver.IdempotencyKeyEQ(key)).Only(ctx)
	if ent.IsNotFound(e) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	return verificationToBiz(x)
}
func (r *verificationRepo) LoadCashflowContext(ctx context.Context, org, id uuid.UUID) (*biz.FinanceCashflow, error) {
	x, e := r.data.db.FinanceCashflow.Query().Where(cash.IDEQ(id), cash.OrganizationIDEQ(org)).Only(ctx)
	if ent.IsNotFound(e) {
		return nil, biz.ErrFinanceCashflowNotFound
	}
	if e != nil {
		return nil, e
	}
	return cashflowToBiz(x)
}
func (r *verificationRepo) Create(ctx context.Context, org, actor uuid.UUID, v *biz.FinanceVerification, audit *biz.AuditEvent) (*biz.FinanceVerification, error) {
	tx, e := r.data.db.Tx(ctx)
	if e != nil {
		return nil, e
	}
	rollback := func(e error) (*biz.FinanceVerification, error) { _ = tx.Rollback(); return nil, e }
	cashSet, billSet := make(map[uuid.UUID]struct{}), make(map[uuid.UUID]struct{})
	for _, a := range v.Allocations {
		cashSet[a.CashflowID] = struct{}{}
		billSet[a.BillID] = struct{}{}
	}
	cashIDs := sortedFinanceUUIDs(cashSet)
	billIDs := sortedFinanceUUIDs(billSet)
	cashs, e := tx.FinanceCashflow.Query().Where(cash.IDIn(cashIDs...), cash.OrganizationIDEQ(org)).Order(cash.ByID()).ForUpdate().All(ctx)
	if e != nil {
		return rollback(e)
	}
	bills, e := tx.FinanceBill.Query().Where(bill.IDIn(billIDs...), bill.OrganizationIDEQ(org)).Order(bill.ByID()).ForUpdate().All(ctx)
	if e != nil {
		return rollback(e)
	}
	cm := map[uuid.UUID]*ent.FinanceCashflow{}
	for _, x := range cashs {
		cm[x.ID] = x
	}
	bm := map[uuid.UUID]*ent.FinanceBill{}
	for _, x := range bills {
		bm[x.ID] = x
	}
	usedCash, usedBill := map[uuid.UUID]decimal.Decimal{}, map[uuid.UUID]decimal.Decimal{}
	v.BaseAmount, v.BillBaseAmount, v.CashflowBaseAmount, v.ExchangeGainLoss = decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
	existing, e := tx.FinanceVerificationAllocation.Query().Where(alloc.ActiveEQ(true), alloc.Or(alloc.CashflowIDIn(cashIDs...), alloc.BillIDIn(billIDs...))).All(ctx)
	if e != nil {
		return rollback(e)
	}
	for _, x := range existing {
		z, _ := decimal.NewFromString(x.Amount)
		usedCash[x.CashflowID] = usedCash[x.CashflowID].Add(z)
		usedBill[x.BillID] = usedBill[x.BillID].Add(z)
	}
	for i, x := range v.Allocations {
		c, b := cm[x.CashflowID], bm[x.BillID]
		if c == nil || b == nil || c.Status != cash.StatusCONFIRMED || b.Status != bill.StatusCONFIRMED {
			return rollback(biz.ErrVerificationBalance)
		}
		if c.Direction != cash.Direction(b.Direction) || c.SettlementPartyID != b.SettlementPartyID || c.Currency != b.Currency || c.BaseCurrency != b.BaseCurrency || c.BaseCurrency != v.BaseCurrency || biz.OrderFeeDirection(c.Direction) != v.Direction || c.Currency != v.Currency {
			return rollback(biz.ErrVerificationMismatch)
		}
		if i == 0 {
			v.Direction = biz.OrderFeeDirection(c.Direction)
			v.SettlementPartyID = c.SettlementPartyID
			v.SettlementPartyName = c.SettlementPartyName
			v.Currency = c.Currency
		} else if v.Direction != biz.OrderFeeDirection(c.Direction) || v.SettlementPartyID != c.SettlementPartyID || v.Currency != c.Currency {
			return rollback(biz.ErrVerificationMismatch)
		}
		ca, _ := decimal.NewFromString(c.Amount)
		ba, _ := decimal.NewFromString(b.TotalAmount)
		cashBaseTotal, parseErr := decimal.NewFromString(c.BaseAmount)
		if parseErr != nil {
			return rollback(parseErr)
		}
		billBaseTotal, parseErr := decimal.NewFromString(b.BaseCurrencyAmount)
		if parseErr != nil {
			return rollback(parseErr)
		}
		usedCash[c.ID] = usedCash[c.ID].Add(x.Amount)
		usedBill[b.ID] = usedBill[b.ID].Add(x.Amount)
		if usedCash[c.ID].GreaterThan(ca) || usedBill[b.ID].GreaterThan(ba) {
			return rollback(biz.ErrVerificationBalance)
		}
		x.CashflowNo = c.FlowNo
		x.BillNo = b.BillNo
		x.BillBaseAmount, x.CashflowBaseAmount, x.WriteOffBaseAmount, x.ExchangeGainLoss, e = biz.CalculateVerificationAllocationAmounts(v.Direction, x.Amount, ba, billBaseTotal, ca, cashBaseTotal, v.ExchangeRate)
		if e != nil {
			return rollback(e)
		}
		v.BillBaseAmount = v.BillBaseAmount.Add(x.BillBaseAmount)
		v.CashflowBaseAmount = v.CashflowBaseAmount.Add(x.CashflowBaseAmount)
		v.BaseAmount = v.BaseAmount.Add(x.WriteOffBaseAmount)
		v.ExchangeGainLoss = v.ExchangeGainLoss.Add(x.ExchangeGainLoss)
	}
	v.BaseAmount = v.BaseAmount.RoundBank(8)
	v.BillBaseAmount = v.BillBaseAmount.RoundBank(8)
	v.CashflowBaseAmount = v.CashflowBaseAmount.RoundBank(8)
	v.ExchangeGainLoss = v.ExchangeGainLoss.RoundBank(8)
	now := time.Now().UTC()
	rule, sequence, e := allocateNumberInTx(ctx, tx, org, biz.DocumentTypeWriteOff, now)
	if e != nil {
		return rollback(e)
	}
	v.VerificationNo, e = biz.FormatAllocatedNumber(now, rule, sequence, "")
	if e != nil {
		return rollback(e)
	}
	_, e = tx.FinanceVerification.Create().SetID(v.ID).SetOrganizationID(org).SetVerificationNo(v.VerificationNo).SetIdempotencyKey(v.IdempotencyKey).SetStatus(ver.StatusACTIVE).SetDirection(ver.Direction(v.Direction)).SetSettlementPartyID(v.SettlementPartyID).SetSettlementPartyName(v.SettlementPartyName).SetCurrency(v.Currency).SetAmount(v.Amount.StringFixed(8)).SetBaseCurrency(v.BaseCurrency).SetExchangeRate(v.ExchangeRate.StringFixed(8)).SetExchangeRateSource(ver.ExchangeRateSource(v.ExchangeRateSource)).SetExchangeRateDate(v.ExchangeRateDate).SetNillableExchangeRateSettingID(v.ExchangeRateSettingID).SetBaseAmount(v.BaseAmount.StringFixed(8)).SetBillBaseAmount(v.BillBaseAmount.StringFixed(8)).SetCashflowBaseAmount(v.CashflowBaseAmount.StringFixed(8)).SetExchangeGainLoss(v.ExchangeGainLoss.StringFixed(8)).SetVerificationDate(v.VerificationDate).SetNillableNote(v.Note).SetVersion(1).Save(ctx)
	if e != nil {
		return rollback(mapVerificationConstraint(e))
	}
	builders := make([]*ent.FinanceVerificationAllocationCreate, 0, len(v.Allocations))
	for _, x := range v.Allocations {
		builders = append(builders, tx.FinanceVerificationAllocation.Create().SetID(x.ID).SetVerificationID(v.ID).SetCashflowID(x.CashflowID).SetBillID(x.BillID).SetCashflowNo(x.CashflowNo).SetBillNo(x.BillNo).SetAmount(x.Amount.StringFixed(8)).SetBillBaseAmount(x.BillBaseAmount.StringFixed(8)).SetCashflowBaseAmount(x.CashflowBaseAmount.StringFixed(8)).SetWriteOffBaseAmount(x.WriteOffBaseAmount.StringFixed(8)).SetExchangeGainLoss(x.ExchangeGainLoss.StringFixed(8)).SetActive(true))
	}
	if _, e = tx.FinanceVerificationAllocation.CreateBulk(builders...).Save(ctx); e != nil {
		return rollback(mapVerificationConstraint(e))
	}
	if e = writeAudit(ctx, tx.AuditLog, audit); e != nil {
		return rollback(e)
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	return r.Get(ctx, org, v.ID)
}

func sortedFinanceUUIDs(values map[uuid.UUID]struct{}) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	return ids
}

func mapVerificationConstraint(err error) error {
	if !ent.IsConstraintError(err) {
		return err
	}
	message := err.Error()
	switch {
	case strings.Contains(message, "financeverification_org_idempotency"):
		return biz.ErrVerificationIdempotency
	case strings.Contains(message, "verification_allocation_pair_unique"):
		return biz.ErrVerificationInvalid
	default:
		return err
	}
}
func (r *verificationRepo) Reverse(ctx context.Context, org, id, actor uuid.UUID, version uint64, reason string, audit *biz.AuditEvent) (*biz.FinanceVerification, error) {
	tx, e := r.data.db.Tx(ctx)
	if e != nil {
		return nil, e
	}
	rollback := func(e error) (*biz.FinanceVerification, error) { _ = tx.Rollback(); return nil, e }
	x, e := tx.FinanceVerification.Query().Where(ver.IDEQ(id), ver.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
	if ent.IsNotFound(e) {
		return rollback(biz.ErrVerificationNotFound)
	}
	if e != nil {
		return rollback(e)
	}
	if x.Version != version || x.Status != ver.StatusACTIVE {
		return rollback(biz.ErrVerificationTransition)
	}
	hasCommission, e := tx.FinanceCommission.Query().Where(commission.VerificationIDEQ(id), commission.StatusNEQ(commission.StatusCANCELLED)).Exist(ctx)
	if e != nil {
		return rollback(e)
	}
	if hasCommission {
		return rollback(biz.ErrVerificationHasCommission)
	}
	if _, e = tx.FinanceVerificationAllocation.Update().Where(alloc.VerificationIDEQ(id), alloc.ActiveEQ(true)).SetActive(false).Save(ctx); e != nil {
		return rollback(e)
	}
	now := time.Now()
	if _, e = tx.FinanceVerification.UpdateOneID(id).SetStatus(ver.StatusREVERSED).SetReversedAt(now).SetReversedBy(actor).SetReversalReason(reason).SetVersion(version + 1).Save(ctx); e != nil {
		return rollback(e)
	}
	if e = writeAudit(ctx, tx.AuditLog, audit); e != nil {
		return rollback(e)
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	return r.Get(ctx, org, id)
}
func verificationToBiz(x *ent.FinanceVerification) (*biz.FinanceVerification, error) {
	amount, e := decimal.NewFromString(x.Amount)
	if e != nil {
		return nil, e
	}
	exchangeRate, e := decimal.NewFromString(x.ExchangeRate)
	if e != nil {
		return nil, e
	}
	baseAmount, e := decimal.NewFromString(x.BaseAmount)
	if e != nil {
		return nil, e
	}
	billBaseAmount, e := decimal.NewFromString(x.BillBaseAmount)
	if e != nil {
		return nil, e
	}
	cashflowBaseAmount, e := decimal.NewFromString(x.CashflowBaseAmount)
	if e != nil {
		return nil, e
	}
	exchangeGainLoss, e := decimal.NewFromString(x.ExchangeGainLoss)
	if e != nil {
		return nil, e
	}
	v := &biz.FinanceVerification{ID: x.ID, OrganizationID: x.OrganizationID, VerificationNo: x.VerificationNo, IdempotencyKey: x.IdempotencyKey, Status: biz.VerificationStatus(x.Status), Direction: biz.OrderFeeDirection(x.Direction), SettlementPartyID: x.SettlementPartyID, SettlementPartyName: x.SettlementPartyName, Currency: x.Currency, Amount: amount, BaseCurrency: x.BaseCurrency, ExchangeRate: exchangeRate, ExchangeRateSource: string(x.ExchangeRateSource), ExchangeRateDate: x.ExchangeRateDate, ExchangeRateSettingID: x.ExchangeRateSettingID, BaseAmount: baseAmount, BillBaseAmount: billBaseAmount, CashflowBaseAmount: cashflowBaseAmount, ExchangeGainLoss: exchangeGainLoss, VerificationDate: x.VerificationDate, Note: x.Note, Version: x.Version, ReversedAt: x.ReversedAt, ReversedBy: x.ReversedBy, ReversalReason: x.ReversalReason, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt, Allocations: make([]*biz.VerificationAllocation, 0, len(x.Edges.Allocations))}
	for _, a := range x.Edges.Allocations {
		z, e := decimal.NewFromString(a.Amount)
		if e != nil {
			return nil, e
		}
		billBase, e := decimal.NewFromString(a.BillBaseAmount)
		if e != nil {
			return nil, e
		}
		cashBase, e := decimal.NewFromString(a.CashflowBaseAmount)
		if e != nil {
			return nil, e
		}
		writeOffBase, e := decimal.NewFromString(a.WriteOffBaseAmount)
		if e != nil {
			return nil, e
		}
		gainLoss, e := decimal.NewFromString(a.ExchangeGainLoss)
		if e != nil {
			return nil, e
		}
		v.Allocations = append(v.Allocations, &biz.VerificationAllocation{ID: a.ID, VerificationID: a.VerificationID, CashflowID: a.CashflowID, BillID: a.BillID, CashflowNo: a.CashflowNo, BillNo: a.BillNo, Amount: z, BillBaseAmount: billBase, CashflowBaseAmount: cashBase, WriteOffBaseAmount: writeOffBase, ExchangeGainLoss: gainLoss, Active: a.Active})
	}
	return v, nil
}

var _ biz.VerificationRepo = (*verificationRepo)(nil)
