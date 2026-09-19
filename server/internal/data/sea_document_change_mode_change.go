package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	seadocumentmodechangeeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seadocumentmodechangeevent"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

func (r *seaDocumentChangeRepo) ExecuteModeChange(ctx context.Context, orgID, actorID uuid.UUID, input *biz.SeaDocumentModeChangeCommand, audit *biz.AuditEvent) error {
	fingerprint := changeFingerprint(input)
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, err := tx.SeaDocumentModeChangeEvent.Query().Where(
			seadocumentmodechangeeventent.OrganizationIDEQ(orgID),
			seadocumentmodechangeeventent.IdempotencyKeyEQ(input.IdempotencyKey),
		).Only(ctx)
		if err == nil {
			if existing.RequestFingerprint == fingerprint {
				return nil
			}
			return biz.ErrSeaDocumentModeChangeConflict
		}
		if !ent.IsNotFound(err) {
			return err
		}
		order, err := tx.Order.Query().Where(orderent.IDEQ(input.OrderID), orderent.OrganizationIDEQ(orgID)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrOrderNotFound, nil)
		}
		if order.Version != input.ExpectedOrderVersion {
			return biz.ErrOrderStatusConflict
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}
		located, err := tx.SeaMasterBillOrderLink.Query().Where(
			seamasterbillorderlinkent.OrganizationIDEQ(orgID),
			seamasterbillorderlinkent.OrderIDEQ(input.OrderID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaDocumentNoActiveLink, nil)
		}
		mbl, err := tx.SeaMasterBill.Query().Where(seamasterbillent.IDEQ(located.MasterBillID), seamasterbillent.OrganizationIDEQ(orgID)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
		}
		link, err := tx.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlinkent.IDEQ(located.ID)).ForUpdate().Only(ctx)
		if err != nil {
			return err
		}
		if !seaDocumentLinkMatches(link, orgID, input.OrderID, mbl.ID) || link.Version != input.ExpectedLinkVersion {
			return biz.ErrSeaDocumentModeChangeConflict
		}
		previousMode := biz.SeaDocumentStructure(link.DocumentStructure)
		if previousMode == input.TargetMode {
			return biz.ErrSeaDocumentModeChangeConflict
		}
		// 共享主单批次规则：批次内还有其他活动成员票时禁止转为直单，防止经模式
		// 切换绕过全员分单制；批次仅剩本票时允许退出。MBL 已按锁序锁定，
		// 成员 Link 按主键排序加锁。
		if input.TargetMode == biz.SeaDocumentStructureDirect {
			batchMembers, err := querySeaMasterBillActiveMemberLinks(ctx, tx.Client(), orgID, mbl.ID, true)
			if err != nil {
				return err
			}
			for _, member := range batchMembers {
				if member.OrderID != order.ID {
					return biz.ErrSeaDocumentBatchMemberExitBlocked
				}
			}
		}
		if err := validateConfirmationAttachment(ctx, tx.Client(), orgID, order.ID, input.Confirmation); err != nil {
			return err
		}
		// HOUSE→DIRECT 会把当前唯一活动 HBL 置为 VOIDED：先定位该 HBL 并把其 ID
		// 传入影响收集，使箱货分配进入阻断事实；口径与 PreviewModeChange 一致，
		// 不允许 Preview 可执行而 Execute 被阻断（或反之）的漂移。
		impactHouseBillID := uuid.Nil
		if previousMode == biz.SeaDocumentStructureHouse {
			activeHBL, err := tx.SeaHouseBill.Query().Where(
				seahousebillent.OrganizationIDEQ(orgID),
				seahousebillent.OrderIDEQ(order.ID),
				seahousebillent.MasterBillIDEQ(mbl.ID),
				seahousebillent.StatusNotIn(seahousebillent.StatusVOIDED),
			).Only(ctx)
			if err != nil {
				return mapEntError(err, biz.ErrSeaDocumentStructureConflict, nil)
			}
			impactHouseBillID = activeHBL.ID
		}
		impacts, err := collectDocumentImpacts(ctx, tx.Client(), orgID, []uuid.UUID{order.ID}, impactHouseBillID)
		if err != nil {
			return err
		}
		if hasBlockingImpact(impacts) {
			return impactError(biz.ErrSeaDocumentChangeBlocked, impacts)
		}
		builder := tx.SeaDocumentModeChangeEvent.Create().
			SetOrganizationID(orgID).
			SetOrderID(order.ID).
			SetPreviousMode(seadocumentmodechangeeventent.PreviousMode(previousMode)).
			SetTargetMode(seadocumentmodechangeeventent.TargetMode(input.TargetMode)).
			SetReason(input.Reason).
			SetImpactSummary(impactSummary(impacts)).
			SetConfirmedByParty(input.Confirmation.ConfirmedByParty).
			SetConfirmedAt(input.Confirmation.ConfirmedAt).
			SetConfirmationNote(input.Confirmation.ConfirmationNote).
			SetNillableConfirmationAttachmentID(input.Confirmation.ConfirmationAttachmentID).
			SetCreatedBy(actorID).
			SetIdempotencyKey(input.IdempotencyKey).
			SetRequestFingerprint(fingerprint)

		switch {
		case previousMode == biz.SeaDocumentStructureHouse && input.TargetMode == biz.SeaDocumentStructureDirect:
			if input.ExpectedHouseBillVersion == nil || input.ExpectedCurrentVersionID == nil {
				return biz.ErrSeaDocumentInvalidArgument
			}
			hbl, err := tx.SeaHouseBill.Query().Where(
				seahousebillent.OrganizationIDEQ(orgID),
				seahousebillent.OrderIDEQ(order.ID),
				seahousebillent.MasterBillIDEQ(mbl.ID),
				seahousebillent.StatusNotIn(seahousebillent.StatusVOIDED),
			).ForUpdate().Only(ctx)
			if err != nil {
				return mapEntError(err, biz.ErrSeaDocumentStructureConflict, nil)
			}
			if hbl.Version != *input.ExpectedHouseBillVersion || hbl.CurrentVersionID == nil || *hbl.CurrentVersionID != *input.ExpectedCurrentVersionID {
				return biz.ErrSeaDocumentModeChangeConflict
			}
			updated, err := hbl.Update().SetStatus(seahousebillent.StatusVOIDED).SetVersion(hbl.Version + 1).Save(ctx)
			if err != nil {
				return err
			}
			version, err := createHouseVersion(ctx, tx, updated, actorID, biz.VersionSourceModeChange, &input.Reason, nil, nil, input.Confirmation)
			if err != nil {
				return err
			}
			if _, err = updated.Update().SetCurrentVersionID(version.ID).Save(ctx); err != nil {
				return err
			}
			builder.SetPreviousHouseBillID(hbl.ID).SetPreviousHouseBillVersionID(*input.ExpectedCurrentVersionID)
		case previousMode == biz.SeaDocumentStructureDirect && input.TargetMode == biz.SeaDocumentStructureHouse:
			if input.NewHouseBill == nil || input.ExpectedHouseBillVersion != nil || input.ExpectedCurrentVersionID != nil {
				return biz.ErrSeaDocumentInvalidArgument
			}
			activeCount, err := tx.SeaHouseBill.Query().Where(seahousebillent.OrderIDEQ(order.ID), seahousebillent.StatusNotIn(seahousebillent.StatusVOIDED)).Count(ctx)
			if err != nil {
				return err
			}
			if activeCount != 0 {
				return biz.ErrSeaDocumentStructureConflict
			}
			normalized, err := biz.NormalizeSeaHouseNo(input.NewHouseBill.HouseNo)
			if err != nil {
				return err
			}
			// 批次内排重预查（含作废行，一号一案）：先给出含冲突分单号的友好报错，
			// 唯一索引仅作并发兜底。跨批次重号不受限制。
			duplicateExists, err := tx.SeaHouseBill.Query().Where(
				seahousebillent.MasterBillIDEQ(mbl.ID),
				seahousebillent.NormalizedHouseNoEQ(normalized),
			).Exist(ctx)
			if err != nil {
				return err
			}
			if duplicateExists {
				return biz.SeaHouseBillBatchNoDuplicateError([]string{normalized})
			}
			issuerOrgID, issuerPartnerID, err := validateSeaHouseBillIssuer(ctx, tx.Client(), orgID, order.OrganizationID, order.CustomerID, input.NewHouseBill)
			if err != nil {
				return err
			}
			hblBuilder := tx.SeaHouseBill.Create().SetID(uuid.Must(uuid.NewV7())).SetOrganizationID(orgID).SetOrderID(order.ID).SetMasterBillID(mbl.ID).SetHouseNo(input.NewHouseBill.HouseNo).SetNormalizedHouseNo(normalized).SetIssuerSource(seahousebillent.IssuerSource(input.NewHouseBill.IssuerSource)).SetStatus(seahousebillent.StatusDRAFT).SetVersion(1)
			if issuerOrgID != nil {
				hblBuilder.SetIssuerOrganizationID(*issuerOrgID)
			}
			if issuerPartnerID != nil {
				hblBuilder.SetIssuerPartnerID(*issuerPartnerID)
			}
			if input.NewHouseBill.Note != nil {
				hblBuilder.SetNote(*input.NewHouseBill.Note)
			}
			setSeaHouseBillContentCreate(hblBuilder, input.NewHouseBill.Content)
			hbl, err := hblBuilder.Save(ctx)
			if err != nil {
				if ent.IsConstraintError(err) {
					return biz.ErrSeaHouseBillBatchNoDuplicate
				}
				return err
			}
			version, err := createHouseVersion(ctx, tx, hbl, actorID, biz.VersionSourceModeChange, &input.Reason, nil, nil, input.Confirmation)
			if err != nil {
				return err
			}
			if _, err = hbl.Update().SetCurrentVersionID(version.ID).Save(ctx); err != nil {
				return err
			}
			builder.SetTargetHouseBillID(hbl.ID).SetTargetHouseBillVersionID(version.ID)
		default:
			return biz.ErrSeaDocumentModeChangeConflict
		}
		if _, err = link.Update().SetDocumentStructure(seamasterbillorderlinkent.DocumentStructure(input.TargetMode)).SetVersion(link.Version + 1).Save(ctx); err != nil {
			return err
		}
		if _, err = order.Update().SetVersion(order.Version + 1).Save(ctx); err != nil {
			return err
		}
		if _, err = builder.Save(ctx); err != nil {
			return err
		}
		audit.Action = "sea_document.mode_change"
		audit.Details = map[string]string{"order.id": order.ID.String(), "previous_mode": string(previousMode), "target_mode": string(input.TargetMode), "reason": input.Reason}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}
