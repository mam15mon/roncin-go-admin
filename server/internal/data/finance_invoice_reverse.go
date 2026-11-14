package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	invoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	linkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
)

// 所有发票反向操作与登记/修订共用订单→账单→发票锁序；先锁发票再锁账单会与并发登记死锁。
func lockInvoiceForReverse(ctx context.Context, client *ent.Client, org, id uuid.UUID, version uint64) (*ent.FinanceInvoice, []uuid.UUID, error) {
	links, err := client.FinanceInvoiceBill.Query().Where(linkent.InvoiceIDEQ(id), linkent.ActiveEQ(true), linkent.HasInvoiceWith(invoiceent.OrganizationIDEQ(org))).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	set := map[uuid.UUID]bool{}
	for _, l := range links {
		set[l.BillID] = true
	}
	ids := mapUUIDs(set)
	if err := lockExistingBillOrders(ctx, client, []uuid.UUID{org}, ids); err != nil {
		return nil, nil, err
	}
	bills, err := client.FinanceBill.Query().Where(billent.IDIn(ids...), billent.OrganizationIDEQ(org)).Order(billent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return nil, nil, err
	}
	if len(bills) != len(ids) {
		return nil, nil, biz.ErrFinanceInvoiceBillInvalid
	}
	item, err := client.FinanceInvoice.Query().Where(invoiceent.IDEQ(id), invoiceent.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, nil, mapEntError(err, biz.ErrFinanceInvoiceNotFound, nil)
	}
	if item.Version != version {
		return nil, nil, biz.ErrFinanceInvoiceVersionConflict
	}
	if item.Status != invoiceent.StatusISSUED {
		return nil, nil, biz.ErrFinanceInvoiceInvalidTransition
	}
	latest, err := client.FinanceInvoiceBill.Query().Where(linkent.InvoiceIDEQ(id), linkent.ActiveEQ(true)).Order(linkent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return nil, nil, err
	}
	if len(latest) == 0 || len(latest) != len(links) {
		return nil, nil, biz.ErrFinanceInvoiceInvalidTransition
	}
	seen := map[uuid.UUID]bool{}
	for _, l := range links {
		seen[l.ID] = true
	}
	for _, l := range latest {
		if !seen[l.ID] {
			return nil, nil, biz.ErrFinanceInvoiceVersionConflict
		}
	}
	if err := assertInvoiceBillsHaveNoSettlement(ctx, client, ids); err != nil {
		return nil, nil, err
	}
	return item, ids, nil
}
func releaseInvoiceLinks(ctx context.Context, client *ent.Client, id uuid.UUID, billIDs []uuid.UUID) error {
	if _, err := client.FinanceInvoiceBill.Update().Where(linkent.InvoiceIDEQ(id), linkent.ActiveEQ(true)).SetActive(false).Save(ctx); err != nil {
		return err
	}
	_, err := client.FinanceBill.Update().Where(billent.IDIn(billIDs...)).AddVersion(1).Save(ctx)
	return err
}
func (r *financeInvoiceRepo) Cancel(ctx context.Context, org, id, actor uuid.UUID, version uint64, reason string, audit *biz.AuditEvent) (*biz.FinanceInvoice, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, bills, err := lockInvoiceForReverse(ctx, tx.Client(), org, id, version)
		if err != nil {
			return err
		}
		if err := releaseInvoiceLinks(ctx, tx.Client(), id, bills); err != nil {
			return err
		}
		if err := tx.FinanceInvoice.UpdateOneID(id).SetStatus(invoiceent.StatusCANCELLED).SetCancelledAt(time.Now()).SetCancelledBy(actor).SetCancellationReason(reason).SetVersion(item.Version + 1).Exec(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, org, id)
}
func (r *financeInvoiceRepo) RedFlush(ctx context.Context, org, id, actor uuid.UUID, version uint64, number, date, reason string, audit *biz.AuditEvent) (*biz.FinanceInvoice, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, bills, err := lockInvoiceForReverse(ctx, tx.Client(), org, id, version)
		if err != nil {
			return err
		}
		if err := releaseInvoiceLinks(ctx, tx.Client(), id, bills); err != nil {
			return err
		}
		if err := tx.FinanceInvoice.UpdateOneID(id).SetStatus(invoiceent.StatusRED_FLUSHED).SetRedInvoiceNo(number).SetRedInvoiceDate(date).SetRedFlushedAt(time.Now()).SetRedFlushedBy(actor).SetRedFlushReason(reason).SetVersion(item.Version + 1).Exec(ctx); err != nil {
			return mapEntConstraint(err, "financeinvoice_org_red_invoice_no", biz.ErrFinanceInvoiceRedNoExists)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, org, id)
}
