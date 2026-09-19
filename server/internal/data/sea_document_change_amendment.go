package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seahousebillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebillversion"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillversion"
	seatransportexecutionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
)

// amendmentImpactHouseBillID 返回改单影响收集的 HBL 上下文：只有 HBL 改单需要
// 查询箱货分配，MBL 改单传入零值跳过。
func amendmentImpactHouseBillID(input *biz.SeaDocumentAmendmentCommand) uuid.UUID {
	if input.DocumentType == biz.SeaDocumentTypeHouseBill {
		return input.DocumentID
	}
	return uuid.Nil
}

func (r *seaDocumentChangeRepo) ExecuteAmendment(ctx context.Context, orgID, actorID uuid.UUID, input *biz.SeaDocumentAmendmentCommand, audit *biz.AuditEvent) (*biz.SeaDocumentVersion, error) {
	fingerprint := changeFingerprint(input)
	var resultID uuid.UUID
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if input.DocumentType == biz.SeaDocumentTypeMasterBill {
			existing, err := tx.SeaMasterBillVersion.Query().Where(seamasterbillversionent.OrganizationIDEQ(orgID), seamasterbillversionent.IdempotencyKeyEQ(input.IdempotencyKey)).Only(ctx)
			if err == nil {
				if existing.RequestFingerprint != nil && *existing.RequestFingerprint == fingerprint {
					resultID = existing.ID
					return nil
				}
				return biz.ErrSeaDocumentVersionConflict
			}
			if !ent.IsNotFound(err) {
				return err
			}
			return r.executeMasterAmendment(ctx, tx, orgID, actorID, input, fingerprint, audit, &resultID)
		}
		existing, err := tx.SeaHouseBillVersion.Query().Where(seahousebillversionent.OrganizationIDEQ(orgID), seahousebillversionent.IdempotencyKeyEQ(input.IdempotencyKey)).Only(ctx)
		if err == nil {
			if existing.RequestFingerprint != nil && *existing.RequestFingerprint == fingerprint {
				resultID = existing.ID
				return nil
			}
			return biz.ErrSeaDocumentVersionConflict
		}
		if !ent.IsNotFound(err) {
			return err
		}
		return r.executeHouseAmendment(ctx, tx, orgID, actorID, input, fingerprint, audit, &resultID)
	})
	if err != nil {
		replayID, found, replayErr := r.findAmendmentReplay(ctx, orgID, input.DocumentType, input.IdempotencyKey, fingerprint)
		if replayErr != nil {
			return nil, replayErr
		}
		if !found {
			return nil, err
		}
		resultID = replayID
	}
	return r.GetDocumentVersion(ctx, orgID, input.OrderID, resultID, input.DocumentType)
}

func (r *seaDocumentChangeRepo) executeMasterAmendment(ctx context.Context, tx *ent.Tx, orgID, actorID uuid.UUID, input *biz.SeaDocumentAmendmentCommand, fingerprint string, audit *biz.AuditEvent, resultID *uuid.UUID) error {
	memberIDs, activeLinkID, err := locateMasterMemberOrderIDs(ctx, tx.Client(), orgID, input.OrderID, input.DocumentID)
	if err != nil {
		return err
	}
	orders, err := tx.Order.Query().Where(orderent.OrganizationIDEQ(orgID), orderent.IDIn(memberIDs...)).Order(orderent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	for _, order := range orders {
		if order.ID == input.OrderID && order.Version != input.ExpectedOrderVersion {
			return biz.ErrOrderStatusConflict
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}
	}
	mbl, err := tx.SeaMasterBill.Query().Where(seamasterbillent.IDEQ(input.DocumentID), seamasterbillent.OrganizationIDEQ(orgID)).ForUpdate().Only(ctx)
	if err != nil {
		return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
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
	exec, err := tx.SeaTransportExecution.Query().Where(seatransportexecutionent.IDEQ(activeLink.TransportExecutionID), seatransportexecutionent.OrganizationIDEQ(orgID)).ForUpdate().Only(ctx)
	if err != nil {
		return err
	}
	base, _, _, err := loadAmendmentPreview(ctx, tx.Client(), orgID, input)
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
	updatedBuilder := mbl.Update().SetVersion(mbl.Version + 1)
	setSeaMasterBillContent(updatedBuilder, input.Input.MasterBillContent)
	updated, err := updatedBuilder.Save(ctx)
	if err != nil {
		return err
	}
	version, err := createMasterVersion(ctx, tx, updated, exec, actorID, biz.VersionSourceAmendment, &input.Reason, &input.IdempotencyKey, &fingerprint, input.Confirmation)
	if err != nil {
		return err
	}
	if _, err = updated.Update().SetCurrentVersionID(version.ID).Save(ctx); err != nil {
		return err
	}
	for _, order := range orders {
		if order.ID == input.OrderID {
			if _, err = order.Update().SetVersion(order.Version + 1).Save(ctx); err != nil {
				return err
			}
			break
		}
	}
	*resultID = version.ID
	audit.Action = "sea_master_bill.amend"
	audit.Details = map[string]string{"order.id": input.OrderID.String(), "master_bill.id": mbl.ID.String(), "previous_version.id": base.ID.String(), "result_version.id": version.ID.String(), "reason": input.Reason}
	return writeAudit(ctx, tx.AuditLog, audit)
}

func (r *seaDocumentChangeRepo) executeHouseAmendment(ctx context.Context, tx *ent.Tx, orgID, actorID uuid.UUID, input *biz.SeaDocumentAmendmentCommand, fingerprint string, audit *biz.AuditEvent, resultID *uuid.UUID) error {
	order, link, mbl, hbl, err := lockHouseDocument(ctx, tx, orgID, input.OrderID, input.DocumentID, input.ExpectedOrderVersion, input.ExpectedDocumentVersion, input.ExpectedCurrentVersionID)
	if err != nil {
		return err
	}
	if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
		return err
	}
	if err := validateConfirmationAttachment(ctx, tx.Client(), orgID, order.ID, input.Confirmation); err != nil {
		return err
	}
	base, _, _, err := loadAmendmentPreview(ctx, tx.Client(), orgID, input)
	if err != nil {
		return err
	}
	impacts, err := collectDocumentImpacts(ctx, tx.Client(), orgID, []uuid.UUID{order.ID}, hbl.ID)
	if err != nil {
		return err
	}
	if hasBlockingImpact(impacts) {
		return impactError(biz.ErrSeaDocumentChangeBlocked, impacts)
	}
	issuerOrgID, issuerPartnerID, err := validateSeaHouseBillIssuer(ctx, tx.Client(), orgID, order.OrganizationID, order.CustomerID, input.Input.HouseBill)
	if err != nil {
		return err
	}
	normalized, _ := biz.NormalizeSeaHouseNo(input.Input.HouseBill.HouseNo)
	builder := hbl.Update().SetHouseNo(input.Input.HouseBill.HouseNo).SetNormalizedHouseNo(normalized).SetIssuerSource(seahousebillent.IssuerSource(input.Input.HouseBill.IssuerSource)).SetVersion(hbl.Version + 1)
	if issuerOrgID != nil {
		builder.SetIssuerOrganizationID(*issuerOrgID).ClearIssuerPartnerID()
	} else {
		builder.SetIssuerPartnerID(*issuerPartnerID).ClearIssuerOrganizationID()
	}
	if input.Input.HouseBill.Note != nil {
		builder.SetNote(*input.Input.HouseBill.Note)
	} else {
		builder.ClearNote()
	}
	setSeaHouseBillContentUpdate(builder, input.Input.HouseBill.Content)
	updated, err := builder.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return biz.ErrSeaHouseBillExists
		}
		return err
	}
	version, err := createHouseVersion(ctx, tx, updated, actorID, biz.VersionSourceAmendment, &input.Reason, &input.IdempotencyKey, &fingerprint, input.Confirmation)
	if err != nil {
		return err
	}
	if _, err = updated.Update().SetCurrentVersionID(version.ID).Save(ctx); err != nil {
		return err
	}
	if _, err = link.Update().SetVersion(link.Version + 1).Save(ctx); err != nil {
		return err
	}
	if _, err = order.Update().SetVersion(order.Version + 1).Save(ctx); err != nil {
		return err
	}
	*resultID = version.ID
	audit.Action = "sea_house_bill.amend"
	audit.Details = map[string]string{"order.id": order.ID.String(), "master_bill.id": mbl.ID.String(), "house_bill.id": hbl.ID.String(), "previous_version.id": base.ID.String(), "result_version.id": version.ID.String(), "reason": input.Reason}
	return writeAudit(ctx, tx.AuditLog, audit)
}

func (r *seaDocumentChangeRepo) findAmendmentReplay(ctx context.Context, orgID uuid.UUID, documentType biz.SeaDocumentType, idempotencyKey, fingerprint string) (uuid.UUID, bool, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return uuid.Nil, false, err
	}
	if documentType == biz.SeaDocumentTypeMasterBill {
		row, err := client.SeaMasterBillVersion.Query().Where(seamasterbillversionent.OrganizationIDEQ(orgID), seamasterbillversionent.IdempotencyKeyEQ(idempotencyKey)).Only(ctx)
		if ent.IsNotFound(err) {
			return uuid.Nil, false, nil
		}
		if err != nil {
			return uuid.Nil, false, err
		}
		if row.RequestFingerprint == nil || *row.RequestFingerprint != fingerprint {
			return uuid.Nil, false, biz.ErrSeaDocumentVersionConflict
		}
		return row.ID, true, nil
	}
	row, err := client.SeaHouseBillVersion.Query().Where(seahousebillversionent.OrganizationIDEQ(orgID), seahousebillversionent.IdempotencyKeyEQ(idempotencyKey)).Only(ctx)
	if ent.IsNotFound(err) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	if row.RequestFingerprint == nil || *row.RequestFingerprint != fingerprint {
		return uuid.Nil, false, biz.ErrSeaDocumentVersionConflict
	}
	return row.ID, true, nil
}
