package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	seadocumentvoideventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seadocumentvoidevent"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seatransportexecutionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
)

func (r *seaDocumentChangeRepo) ExecuteVoid(ctx context.Context, orgID, actorID uuid.UUID, input *biz.SeaDocumentVoidCommand, audit *biz.AuditEvent) (*biz.SeaDocumentEvent, error) {
	fingerprint := changeFingerprint(input)
	var eventID uuid.UUID
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, err := tx.SeaDocumentVoidEvent.Query().Where(seadocumentvoideventent.OrganizationIDEQ(orgID), seadocumentvoideventent.IdempotencyKeyEQ(input.IdempotencyKey)).Only(ctx)
		if err == nil {
			if existing.RequestFingerprint == fingerprint {
				eventID = existing.ID
				return nil
			}
			return biz.ErrSeaDocumentVersionConflict
		}
		if !ent.IsNotFound(err) {
			return err
		}
		if input.DocumentType == biz.SeaDocumentTypeMasterBill {
			return r.executeMasterVoid(ctx, tx, orgID, actorID, input, fingerprint, audit, &eventID)
		}
		return r.executeHouseVoid(ctx, tx, orgID, actorID, input, fingerprint, audit, &eventID)
	})
	if err != nil {
		replayID, found, replayErr := r.findVoidReplay(ctx, orgID, input.IdempotencyKey, fingerprint)
		if replayErr != nil {
			return nil, replayErr
		}
		if !found {
			return nil, err
		}
		eventID = replayID
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	row, err := client.SeaDocumentVoidEvent.Query().Where(seadocumentvoideventent.IDEQ(eventID), seadocumentvoideventent.OrganizationIDEQ(orgID)).WithMasterBillVersion().WithHouseBillVersion().Only(ctx)
	if err != nil {
		return nil, err
	}
	return voidEventToBiz(row), nil
}

func (r *seaDocumentChangeRepo) executeMasterVoid(ctx context.Context, tx *ent.Tx, orgID, actorID uuid.UUID, input *biz.SeaDocumentVoidCommand, fingerprint string, audit *biz.AuditEvent, eventID *uuid.UUID) error {
	memberIDs, activeLinkID, err := locateMasterMemberOrderIDs(ctx, tx.Client(), orgID, input.OrderID, input.DocumentID)
	if err != nil {
		return err
	}
	orders, err := tx.Order.Query().Where(orderent.OrganizationIDEQ(orgID), orderent.IDIn(memberIDs...)).Order(orderent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	var requestOrder *ent.Order
	for _, order := range orders {
		if order.ID == input.OrderID {
			requestOrder = order
			if order.Version != input.ExpectedOrderVersion {
				return biz.ErrOrderStatusConflict
			}
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}
	}
	mbl, err := tx.SeaMasterBill.Query().Where(seamasterbillent.IDEQ(input.DocumentID), seamasterbillent.OrganizationIDEQ(orgID)).ForUpdate().Only(ctx)
	if err != nil {
		return err
	}
	activeLink, err := lockAndValidateMasterMemberLinks(ctx, tx, orgID, input.OrderID, mbl.ID, activeLinkID, memberIDs)
	if err != nil {
		return err
	}
	if mbl.Status == seamasterbillent.StatusVOIDED {
		return biz.ErrSeaDocumentVoided
	}
	if mbl.Version != input.ExpectedDocumentVersion || mbl.CurrentVersionID == nil || *mbl.CurrentVersionID != input.ExpectedCurrentVersionID {
		return biz.ErrSeaDocumentVersionConflict
	}
	if err := validateConfirmationAttachment(ctx, tx.Client(), orgID, input.OrderID, input.Confirmation); err != nil {
		return err
	}
	exec, err := tx.SeaTransportExecution.Query().Where(seatransportexecutionent.IDEQ(activeLink.TransportExecutionID)).ForUpdate().Only(ctx)
	if err != nil {
		return err
	}
	impacts, err := collectDocumentImpacts(ctx, tx.Client(), orgID, memberIDs, uuid.Nil)
	if err != nil {
		return err
	}
	if hasBlockingImpact(impacts) {
		return impactError(biz.ErrSeaDocumentChangeBlocked, impacts)
	}
	updated, err := mbl.Update().SetStatus(seamasterbillent.StatusVOIDED).SetVersion(mbl.Version + 1).Save(ctx)
	if err != nil {
		return err
	}
	version, err := createMasterVersion(ctx, tx, updated, exec, actorID, biz.VersionSourceVoid, &input.Reason, nil, nil, input.Confirmation)
	if err != nil {
		return err
	}
	if _, err = updated.Update().SetCurrentVersionID(version.ID).Save(ctx); err != nil {
		return err
	}
	row, err := tx.SeaDocumentVoidEvent.Create().SetOrganizationID(orgID).SetOrderID(input.OrderID).SetDocumentType(seadocumentvoideventent.DocumentTypeMASTER).SetMasterBillID(mbl.ID).SetMasterBillVersionID(version.ID).SetPreviousMasterBillVersionID(input.ExpectedCurrentVersionID).SetPreviousStatus(string(mbl.Status)).SetVoidedStatus("VOIDED").SetReason(input.Reason).SetImpactSummary(impactSummary(impacts)).SetCreatedBy(actorID).SetIdempotencyKey(input.IdempotencyKey).SetRequestFingerprint(fingerprint).SetConfirmedByParty(input.Confirmation.ConfirmedByParty).SetConfirmedAt(input.Confirmation.ConfirmedAt).SetConfirmationNote(input.Confirmation.ConfirmationNote).SetNillableConfirmationAttachmentID(input.Confirmation.ConfirmationAttachmentID).Save(ctx)
	if err != nil {
		return err
	}
	*eventID = row.ID
	if requestOrder != nil {
		if _, err = requestOrder.Update().SetVersion(requestOrder.Version + 1).Save(ctx); err != nil {
			return err
		}
	}
	audit.Action = "sea_master_bill.void"
	audit.Details = map[string]string{"order.id": input.OrderID.String(), "master_bill.id": mbl.ID.String(), "previous_version.id": input.ExpectedCurrentVersionID.String(), "result_version.id": version.ID.String(), "reason": input.Reason}
	return writeAudit(ctx, tx.AuditLog, audit)
}

func (r *seaDocumentChangeRepo) executeHouseVoid(ctx context.Context, tx *ent.Tx, orgID, actorID uuid.UUID, input *biz.SeaDocumentVoidCommand, fingerprint string, audit *biz.AuditEvent, eventID *uuid.UUID) error {
	return biz.ErrSeaDocumentStructureConflict
}

func (r *seaDocumentChangeRepo) findVoidReplay(ctx context.Context, orgID uuid.UUID, idempotencyKey, fingerprint string) (uuid.UUID, bool, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return uuid.Nil, false, err
	}
	row, err := client.SeaDocumentVoidEvent.Query().Where(seadocumentvoideventent.OrganizationIDEQ(orgID), seadocumentvoideventent.IdempotencyKeyEQ(idempotencyKey)).Only(ctx)
	if ent.IsNotFound(err) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	if row.RequestFingerprint != fingerprint {
		return uuid.Nil, false, biz.ErrSeaDocumentVersionConflict
	}
	return row.ID, true, nil
}
