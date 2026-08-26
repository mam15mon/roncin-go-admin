package data

import (
	"context"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financeinvoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	financeinvoicebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/shopspring/decimal"
)

type financeInvoiceRepo struct{ data *Data }

func NewFinanceInvoiceRepo(data *Data) biz.FinanceInvoiceRepo { return &financeInvoiceRepo{data: data} }

func (r *financeInvoiceRepo) List(ctx context.Context, org uuid.UUID, filter biz.FinanceInvoiceFilter) (*biz.FinanceInvoiceListResult, error) {
	p := []predicate.FinanceInvoice{financeinvoiceent.OrganizationIDEQ(org)}
	if filter.Keyword != "" {
		p = append(p, financeinvoiceent.Or(financeinvoiceent.RecordNoContainsFold(filter.Keyword), financeinvoiceent.SettlementPartyNameContainsFold(filter.Keyword), financeinvoiceent.TaxInvoiceNoContainsFold(filter.Keyword), financeinvoiceent.HasBillLinksWith(financeinvoicebillent.BillNoContainsFold(filter.Keyword))))
	}
	if filter.Direction != "" {
		p = append(p, financeinvoiceent.DirectionEQ(financeinvoiceent.Direction(filter.Direction)))
	}
	if filter.Status != "" {
		p = append(p, financeinvoiceent.StatusEQ(financeinvoiceent.Status(filter.Status)))
	}
	q := r.data.db.FinanceInvoice.Query().Where(p...)
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := q.Order(financeinvoiceent.ByCreatedAt(entsql.OrderDesc()), financeinvoiceent.ByID(entsql.OrderDesc())).Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceInvoiceListResult{Items: make([]*biz.FinanceInvoice, 0, len(items)), Total: int64(total)}
	for _, item := range items {
		converted, e := financeInvoiceToBiz(item)
		if e != nil {
			return nil, e
		}
		result.Items = append(result.Items, converted)
	}
	return result, nil
}

func (r *financeInvoiceRepo) queryWithLinks(q *ent.FinanceInvoiceQuery) *ent.FinanceInvoiceQuery {
	return q.WithBillLinks(func(lq *ent.FinanceInvoiceBillQuery) {
		lq.Order(financeinvoicebillent.ByCreatedAt(), financeinvoicebillent.ByID())
	})
}
func (r *financeInvoiceRepo) Get(ctx context.Context, org, id uuid.UUID) (*biz.FinanceInvoice, error) {
	item, err := r.queryWithLinks(r.data.db.FinanceInvoice.Query()).Where(financeinvoiceent.IDEQ(id), financeinvoiceent.OrganizationIDEQ(org)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, biz.ErrFinanceInvoiceNotFound
	}
	if err != nil {
		return nil, err
	}
	return financeInvoiceToBiz(item)
}
func (r *financeInvoiceRepo) GetByIdempotencyKey(ctx context.Context, org uuid.UUID, key string) (*biz.FinanceInvoice, error) {
	item, err := r.queryWithLinks(r.data.db.FinanceInvoice.Query()).Where(financeinvoiceent.OrganizationIDEQ(org), financeinvoiceent.IdempotencyKeyEQ(key)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return financeInvoiceToBiz(item)
}
func (r *financeInvoiceRepo) LoadBills(ctx context.Context, org uuid.UUID, ids []uuid.UUID) ([]*biz.FinanceBill, error) {
	items, err := r.data.db.FinanceBill.Query().Where(financebillent.IDIn(ids...), financebillent.OrganizationIDEQ(org)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.FinanceBill, 0, len(items))
	for _, x := range items {
		b, e := financeBillToBiz(x)
		if e != nil {
			return nil, e
		}
		out = append(out, b)
	}
	return out, nil
}

func (r *financeInvoiceRepo) Create(ctx context.Context, invoice *biz.FinanceInvoice, audit *biz.AuditEvent) (*biz.FinanceInvoice, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	rollback := func(e error) (*biz.FinanceInvoice, error) { _ = tx.Rollback(); return nil, e }
	ids := make([]uuid.UUID, 0, len(invoice.Links))
	for _, l := range invoice.Links {
		ids = append(ids, l.BillID)
	}
	bills, err := tx.FinanceBill.Query().Where(financebillent.IDIn(ids...), financebillent.OrganizationIDEQ(invoice.OrganizationID)).ForUpdate().All(ctx)
	if err != nil {
		return rollback(err)
	}
	if len(bills) != len(ids) {
		return rollback(biz.ErrFinanceInvoiceBillInvalid)
	}
	for _, b := range bills {
		if b.Status != financebillent.StatusCONFIRMED || string(b.Direction) != string(invoice.Direction) || b.SettlementPartyID != invoice.SettlementPartyID || b.Currency != invoice.Currency {
			return rollback(biz.ErrFinanceInvoiceBillInvalid)
		}
	}
	active, err := tx.FinanceInvoiceBill.Query().Where(financeinvoicebillent.BillIDIn(ids...), financeinvoicebillent.ActiveEQ(true)).Exist(ctx)
	if err != nil {
		return rollback(err)
	}
	if active {
		return rollback(biz.ErrFinanceInvoiceBillInvalid)
	}
	_, err = tx.FinanceInvoice.Create().SetID(invoice.ID).SetOrganizationID(invoice.OrganizationID).SetRecordNo(invoice.RecordNo).SetIdempotencyKey(invoice.IdempotencyKey).SetDirection(financeinvoiceent.Direction(invoice.Direction)).SetStatus(financeinvoiceent.StatusDRAFT).SetInvoiceType(financeinvoiceent.InvoiceType(invoice.InvoiceType)).SetSettlementPartyID(invoice.SettlementPartyID).SetSettlementPartyName(invoice.SettlementPartyName).SetCurrency(invoice.Currency).SetTotalAmount(invoice.TotalAmount.StringFixed(8)).SetTaxAmount(invoice.TaxAmount.StringFixed(8)).SetBillCount(invoice.BillCount).SetNillableNote(invoice.Note).SetVersion(1).Save(ctx)
	if err != nil {
		return rollback(err)
	}
	builders := make([]*ent.FinanceInvoiceBillCreate, 0, len(invoice.Links))
	for _, l := range invoice.Links {
		builders = append(builders, tx.FinanceInvoiceBill.Create().SetID(l.ID).SetInvoiceID(invoice.ID).SetBillID(l.BillID).SetBillNo(l.BillNo).SetAmount(l.Amount.StringFixed(8)).SetTaxAmount(l.TaxAmount.StringFixed(8)).SetActive(true))
	}
	if _, err = tx.FinanceInvoiceBill.CreateBulk(builders...).Save(ctx); err != nil {
		return rollback(biz.ErrFinanceInvoiceBillInvalid)
	}
	if err = writeAudit(ctx, tx.AuditLog, audit); err != nil {
		return rollback(err)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, invoice.OrganizationID, invoice.ID)
}

func (r *financeInvoiceRepo) Issue(ctx context.Context, org, id, actor uuid.UUID, version uint64, taxNo, date string, audit *biz.AuditEvent) (*biz.FinanceInvoice, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	rollback := func(e error) (*biz.FinanceInvoice, error) { _ = tx.Rollback(); return nil, e }
	item, err := tx.FinanceInvoice.Query().Where(financeinvoiceent.IDEQ(id), financeinvoiceent.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return rollback(biz.ErrFinanceInvoiceNotFound)
	}
	if err != nil {
		return rollback(err)
	}
	if item.Version != version {
		return rollback(biz.ErrFinanceInvoiceVersionConflict)
	}
	if item.Status != financeinvoiceent.StatusDRAFT {
		return rollback(biz.ErrFinanceInvoiceInvalidTransition)
	}
	now := time.Now()
	if _, err = tx.FinanceInvoice.UpdateOneID(id).SetStatus(financeinvoiceent.StatusISSUED).SetTaxInvoiceNo(taxNo).SetInvoiceDate(date).SetIssuedAt(now).SetIssuedBy(actor).SetVersion(item.Version + 1).Save(ctx); err != nil {
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

func (r *financeInvoiceRepo) Cancel(ctx context.Context, org, id, actor uuid.UUID, version uint64, reason string, audit *biz.AuditEvent) (*biz.FinanceInvoice, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	rollback := func(e error) (*biz.FinanceInvoice, error) { _ = tx.Rollback(); return nil, e }
	item, err := tx.FinanceInvoice.Query().Where(financeinvoiceent.IDEQ(id), financeinvoiceent.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return rollback(biz.ErrFinanceInvoiceNotFound)
	}
	if err != nil {
		return rollback(err)
	}
	if item.Version != version {
		return rollback(biz.ErrFinanceInvoiceVersionConflict)
	}
	if item.Status == financeinvoiceent.StatusCANCELLED {
		return rollback(biz.ErrFinanceInvoiceInvalidTransition)
	}
	links, err := tx.FinanceInvoiceBill.Query().Where(financeinvoicebillent.InvoiceIDEQ(id), financeinvoicebillent.ActiveEQ(true)).ForUpdate().All(ctx)
	if err != nil {
		return rollback(err)
	}
	if len(links) == 0 {
		return rollback(biz.ErrFinanceInvoiceInvalidTransition)
	}
	if _, err = tx.FinanceInvoiceBill.Update().Where(financeinvoicebillent.InvoiceIDEQ(id), financeinvoicebillent.ActiveEQ(true)).SetActive(false).Save(ctx); err != nil {
		return rollback(err)
	}
	now := time.Now()
	if _, err = tx.FinanceInvoice.UpdateOneID(id).SetStatus(financeinvoiceent.StatusCANCELLED).SetCancelledAt(now).SetCancelledBy(actor).SetCancellationReason(reason).SetVersion(item.Version + 1).Save(ctx); err != nil {
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

func financeInvoiceToBiz(x *ent.FinanceInvoice) (*biz.FinanceInvoice, error) {
	total, e := decimal.NewFromString(x.TotalAmount)
	if e != nil {
		return nil, e
	}
	tax, e := decimal.NewFromString(x.TaxAmount)
	if e != nil {
		return nil, e
	}
	out := &biz.FinanceInvoice{ID: x.ID, OrganizationID: x.OrganizationID, RecordNo: x.RecordNo, IdempotencyKey: x.IdempotencyKey, Direction: biz.OrderFeeDirection(x.Direction), Status: biz.FinanceInvoiceStatus(x.Status), InvoiceType: biz.FinanceInvoiceType(x.InvoiceType), SettlementPartyID: x.SettlementPartyID, SettlementPartyName: x.SettlementPartyName, Currency: x.Currency, TotalAmount: total, TaxAmount: tax, BillCount: x.BillCount, TaxInvoiceNo: x.TaxInvoiceNo, InvoiceDate: x.InvoiceDate, Note: x.Note, Version: x.Version, IssuedAt: x.IssuedAt, IssuedBy: x.IssuedBy, CancelledAt: x.CancelledAt, CancelledBy: x.CancelledBy, CancellationReason: x.CancellationReason, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt, Links: make([]*biz.FinanceInvoiceBill, 0, len(x.Edges.BillLinks))}
	for _, l := range x.Edges.BillLinks {
		amount, e := decimal.NewFromString(l.Amount)
		if e != nil {
			return nil, e
		}
		tax, e := decimal.NewFromString(l.TaxAmount)
		if e != nil {
			return nil, e
		}
		out.Links = append(out.Links, &biz.FinanceInvoiceBill{ID: l.ID, InvoiceID: l.InvoiceID, BillID: l.BillID, BillNo: l.BillNo, Amount: amount, TaxAmount: tax, Active: l.Active})
	}
	return out, nil
}

var _ biz.FinanceInvoiceRepo = (*financeInvoiceRepo)(nil)
