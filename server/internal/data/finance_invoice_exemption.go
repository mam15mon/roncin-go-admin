package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoiceexemption"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

func (r *financeInvoiceRepo) SetBillInvoiceExemption(ctx context.Context, organizationIDs []uuid.UUID, actorID, billID uuid.UUID, expectedVersion uint64, reason string, revoke bool, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if err := lockExistingBillOrders(ctx, tx.Client(), organizationIDs, []uuid.UUID{billID}); err != nil {
			return err
		}
		bill, err := tx.FinanceBill.Query().Where(financebill.IDEQ(billID), financebill.OrganizationIDIn(organizationIDs...)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrFinanceBillNotFound, nil)
		}
		if bill.Version != expectedVersion {
			return biz.ErrFinanceBillVersionConflict
		}
		if bill.Status != financebill.StatusCONFIRMED {
			return biz.ErrFinanceBillInvalidTransition
		}
		active, err := tx.FinanceInvoiceExemption.Query().Where(financeinvoiceexemption.BillIDEQ(billID), financeinvoiceexemption.RevokedAtIsNil()).ForUpdate().Only(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return err
		}
		if revoke {
			if active == nil {
				return biz.ErrFinanceInvoiceExemptionNotFound
			}
			if _, err := tx.FinanceInvoiceExemption.UpdateOneID(active.ID).SetRevokedAt(time.Now()).SetRevokedBy(actorID).SetRevocationReason(reason).SetVersion(active.Version + 1).Save(ctx); err != nil {
				return err
			}
		} else {
			if active != nil {
				return biz.ErrFinanceInvoiceExemptionConflict
			}
			hasInvoice, err := tx.FinanceInvoiceBill.Query().Where(financeinvoicebill.BillIDEQ(billID), financeinvoicebill.ActiveEQ(true), financeinvoicebill.HasInvoiceWith(financeinvoice.StatusIn(financeinvoice.StatusDRAFT, financeinvoice.StatusISSUED))).Exist(ctx)
			if err != nil {
				return err
			}
			if hasInvoice {
				return biz.ErrFinanceInvoiceExemptionConflict
			}
			if _, err := tx.FinanceInvoiceExemption.Create().SetOrganizationID(bill.OrganizationID).SetBillID(billID).SetMarkedBy(actorID).SetReason(reason).SetVersion(1).Save(ctx); err != nil {
				return mapEntError(err, nil, biz.ErrFinanceInvoiceExemptionConflict)
			}
		}
		if _, err := tx.FinanceBill.UpdateOneID(billID).SetVersion(bill.Version + 1).Save(ctx); err != nil {
			return err
		}
		audit.OrganizationID = &bill.OrganizationID
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

func invoiceExemptionToBiz(item *ent.FinanceInvoiceExemption) *biz.FinanceInvoiceExemption {
	return &biz.FinanceInvoiceExemption{ID: item.ID, OrganizationID: item.OrganizationID, BillID: item.BillID, MarkedBy: item.MarkedBy, Reason: item.Reason, Version: item.Version, CreatedAt: item.CreatedAt, RevokedAt: item.RevokedAt, RevokedBy: item.RevokedBy, RevocationReason: item.RevocationReason}
}

func (r *financeBillRepo) enrichInvoiceExemptionHistory(ctx context.Context, bill *biz.FinanceBill) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	items, err := client.FinanceInvoiceExemption.Query().Where(financeinvoiceexemption.BillIDEQ(bill.ID), financeinvoiceexemption.OrganizationIDEQ(bill.OrganizationID)).Order(financeinvoiceexemption.ByCreatedAt(), financeinvoiceexemption.ByID()).All(ctx)
	if err != nil {
		return err
	}
	actorIDs := make([]uuid.UUID, 0, len(items)*2)
	for _, item := range items {
		bill.InvoiceExemptions = append(bill.InvoiceExemptions, invoiceExemptionToBiz(item))
		actorIDs = append(actorIDs, item.MarkedBy)
		if item.RevokedBy != nil {
			actorIDs = append(actorIDs, *item.RevokedBy)
		}
	}
	if len(actorIDs) == 0 {
		return nil
	}
	actors, err := client.User.Query().Where(user.IDIn(actorIDs...)).Select(user.FieldID, user.FieldDisplayName).All(ctx)
	if err != nil {
		return err
	}
	names := make(map[uuid.UUID]string, len(actors))
	for _, actor := range actors {
		names[actor.ID] = actor.DisplayName
	}
	for _, item := range bill.InvoiceExemptions {
		item.MarkedByName = names[item.MarkedBy]
		if item.RevokedBy != nil {
			item.RevokedByName = names[*item.RevokedBy]
		}
	}
	return nil
}
