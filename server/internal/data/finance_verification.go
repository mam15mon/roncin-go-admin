package data

import (
	"context"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	bill "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	cash "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	ver "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	alloc "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/shopspring/decimal"
	"time"
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
func (r *verificationRepo) Create(ctx context.Context, org, actor uuid.UUID, v *biz.FinanceVerification, audit *biz.AuditEvent) (*biz.FinanceVerification, error) {
	tx, e := r.data.db.Tx(ctx)
	if e != nil {
		return nil, e
	}
	rollback := func(e error) (*biz.FinanceVerification, error) { _ = tx.Rollback(); return nil, e }
	cashIDs, billIDs := []uuid.UUID{}, []uuid.UUID{}
	for _, a := range v.Allocations {
		cashIDs = append(cashIDs, a.CashflowID)
		billIDs = append(billIDs, a.BillID)
	}
	cashs, e := tx.FinanceCashflow.Query().Where(cash.IDIn(cashIDs...), cash.OrganizationIDEQ(org)).ForUpdate().All(ctx)
	if e != nil {
		return rollback(e)
	}
	bills, e := tx.FinanceBill.Query().Where(bill.IDIn(billIDs...), bill.OrganizationIDEQ(org)).ForUpdate().All(ctx)
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
		if c.Direction != cash.Direction(b.Direction) || c.SettlementPartyID != b.SettlementPartyID || c.Currency != b.Currency {
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
		usedCash[c.ID] = usedCash[c.ID].Add(x.Amount)
		usedBill[b.ID] = usedBill[b.ID].Add(x.Amount)
		if usedCash[c.ID].GreaterThan(ca) || usedBill[b.ID].GreaterThan(ba) {
			return rollback(biz.ErrVerificationBalance)
		}
		x.CashflowNo = c.FlowNo
		x.BillNo = b.BillNo
	}
	_, e = tx.FinanceVerification.Create().SetID(v.ID).SetOrganizationID(org).SetVerificationNo(v.VerificationNo).SetIdempotencyKey(v.IdempotencyKey).SetStatus(ver.StatusACTIVE).SetDirection(ver.Direction(v.Direction)).SetSettlementPartyID(v.SettlementPartyID).SetSettlementPartyName(v.SettlementPartyName).SetCurrency(v.Currency).SetAmount(v.Amount.StringFixed(8)).SetVerificationDate(v.VerificationDate).SetNillableNote(v.Note).SetVersion(1).Save(ctx)
	if e != nil {
		return rollback(e)
	}
	builders := make([]*ent.FinanceVerificationAllocationCreate, 0, len(v.Allocations))
	for _, x := range v.Allocations {
		builders = append(builders, tx.FinanceVerificationAllocation.Create().SetID(x.ID).SetVerificationID(v.ID).SetCashflowID(x.CashflowID).SetBillID(x.BillID).SetCashflowNo(x.CashflowNo).SetBillNo(x.BillNo).SetAmount(x.Amount.StringFixed(8)).SetActive(true))
	}
	if _, e = tx.FinanceVerificationAllocation.CreateBulk(builders...).Save(ctx); e != nil {
		return rollback(e)
	}
	if e = writeAudit(ctx, tx.AuditLog, audit); e != nil {
		return rollback(e)
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	return r.Get(ctx, org, v.ID)
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
	v := &biz.FinanceVerification{ID: x.ID, OrganizationID: x.OrganizationID, VerificationNo: x.VerificationNo, IdempotencyKey: x.IdempotencyKey, Status: biz.VerificationStatus(x.Status), Direction: biz.OrderFeeDirection(x.Direction), SettlementPartyID: x.SettlementPartyID, SettlementPartyName: x.SettlementPartyName, Currency: x.Currency, Amount: amount, VerificationDate: x.VerificationDate, Note: x.Note, Version: x.Version, ReversedAt: x.ReversedAt, ReversedBy: x.ReversedBy, ReversalReason: x.ReversalReason, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt, Allocations: make([]*biz.VerificationAllocation, 0, len(x.Edges.Allocations))}
	for _, a := range x.Edges.Allocations {
		z, e := decimal.NewFromString(a.Amount)
		if e != nil {
			return nil, e
		}
		v.Allocations = append(v.Allocations, &biz.VerificationAllocation{ID: a.ID, VerificationID: a.VerificationID, CashflowID: a.CashflowID, BillID: a.BillID, CashflowNo: a.CashflowNo, BillNo: a.BillNo, Amount: z, Active: a.Active})
	}
	return v, nil
}

var _ biz.VerificationRepo = (*verificationRepo)(nil)
