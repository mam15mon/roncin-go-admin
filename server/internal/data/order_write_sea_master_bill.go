package data

import (
	"context"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	financecommissionadjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	financecommissionlineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordercontainerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainer"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	ordermilestoneent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordermilestone"
	orderreleasepodent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderreleasepod"
	ordershippingdocumentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordershippingdocument"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	seamasterbill "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlink "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seatransportexecution "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
	shippinglineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/shippingline"
)

func syncOrderSeaMasterBillOnCreate(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, order *ent.Order, input *biz.Order) error {
	if input.BusinessType != biz.OrderBusinessSE {
		return nil
	}
	if input.SeaMasterBillInput == nil {
		return biz.ErrSeaMasterBillInvalidArgument
	}
	if input.SeaDocumentInput == nil || input.SeaDocumentInput.DocumentStructure == nil {
		return biz.ErrSeaDocumentStructureInvalid
	}
	documentStructure := seamasterbillorderlink.DocumentStructure(*input.SeaDocumentInput.DocumentStructure)
	mblInput := input.SeaMasterBillInput
	if input.ShippingLineID == nil || *input.ShippingLineID == uuid.Nil {
		return biz.ErrSeaMasterBillInvalidArgument
	}
	shippingLineID := *input.ShippingLineID
	if err := validateSeaMasterBillShippingLine(ctx, tx, organizationID, shippingLineID, true); err != nil {
		return err
	}

	vesselName, voyageNo := biz.SplitVesselVoyage(input.VesselVoyage)
	orderVoyage := &biz.SeaTransportExecution{
		VesselName: vesselName,
		VoyageNo:   voyageNo,
		ETD:        parseOptionalTime(input.ETD),
		ETA:        parseOptionalTime(input.ETA),
	}
	if input.ShippingLineID != nil {
		orderVoyage.ShippingLineID = *input.ShippingLineID
	}
	if input.OriginLocationID != nil {
		orderVoyage.OriginLocationID = *input.OriginLocationID
	}
	if input.DischargeLocationID != nil {
		orderVoyage.DischargeLocationID = *input.DischargeLocationID
	}
	if input.TransitLocationID != nil {
		orderVoyage.TransitLocationID = input.TransitLocationID
	}

	if mblInput.CandidateID != nil && *mblInput.CandidateID != uuid.Nil {
		candidateID := *mblInput.CandidateID
		if mblInput.CandidateTEID == nil || *mblInput.CandidateTEID == uuid.Nil ||
			mblInput.ExpectedCandidateTEVersion == nil || *mblInput.ExpectedCandidateTEVersion == 0 {
			return biz.ErrSeaMasterBillInvalidArgument
		}
		candidateTEID := *mblInput.CandidateTEID
		targetMBL, err := tx.SeaMasterBill.Query().
			Where(seamasterbill.IDEQ(candidateID), seamasterbill.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
		}
		if mblInput.ExpectedCandidateVersion == nil || targetMBL.Version != *mblInput.ExpectedCandidateVersion {
			return biz.ErrSeaMasterBillStatusConflict
		}
		if targetMBL.ShippingLineID != shippingLineID || targetMBL.NormalizedMasterNo != mblInput.MasterNo {
			return biz.ErrSeaMasterBillStatusConflict
		}

		activeLinks, err := querySeaMasterBillActiveMemberLinks(ctx, tx.Client(), organizationID, targetMBL.ID, true)
		if err != nil {
			return err
		}
		teLinkCount := 0
		for _, link := range activeLinks {
			if link.TransportExecutionID == candidateTEID {
				teLinkCount++
			}
		}
		if teLinkCount == 0 {
			return biz.ErrSeaMasterBillStatusConflict
		}
		// 共享主单批次规则（全员分单制）：批次存在其他活动成员票时，
		// 任一直单成员即禁止加拼（直单票独占主单）；全 HOUSE 批次要求本单
		// 必须为 HOUSE 且分单号非空。本单为新建订单，尚无活动 Link，不会
		// 出现在成员列表中。
		if len(activeLinks) > 0 {
			for _, member := range activeLinks {
				if member.DocumentStructure == seamasterbillorderlink.DocumentStructureDIRECT {
					return biz.ErrSeaMasterBillBatchDirectBlocked
				}
			}
			seaDocument := input.SeaDocumentInput
			if documentStructure != seamasterbillorderlink.DocumentStructureHOUSE ||
				seaDocument == nil || seaDocument.HouseBill == nil ||
				strings.TrimSpace(seaDocument.HouseBill.HouseNo) == "" {
				return biz.ErrSeaOrderBatchRequiresHouse
			}
		}
		targetTE, err := tx.SeaTransportExecution.Query().
			Where(seatransportexecution.IDEQ(candidateTEID), seatransportexecution.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
		}
		if targetTE.Version != *mblInput.ExpectedCandidateTEVersion || targetTE.ShippingLineID != targetMBL.ShippingLineID || targetTE.ShippingLineID != shippingLineID {
			return biz.ErrSeaMasterBillStatusConflict
		}

		candidateVoyage := &biz.SeaTransportExecution{
			ID:                targetTE.ID,
			TransitLocationID: targetTE.TransitLocationID,
			VesselName:        targetTE.VesselName,
			VoyageNo:          targetTE.VoyageNo,
			ETD:               targetTE.Etd,
			ETA:               targetTE.Eta,
		}
		candidateVoyage.ShippingLineID = targetTE.ShippingLineID
		if targetTE.OriginLocationID != nil {
			candidateVoyage.OriginLocationID = *targetTE.OriginLocationID
		}
		if targetTE.DischargeLocationID != nil {
			candidateVoyage.DischargeLocationID = *targetTE.DischargeLocationID
		}
		conflicts := biz.CheckSeaVoyageConflicts(candidateVoyage, orderVoyage)
		if len(conflicts) > 0 {
			return biz.ErrSeaMasterBillVoyageConflict
		}

		_, err = tx.SeaMasterBillOrderLink.Create().
			SetID(uuid.Must(uuid.NewV7())).
			SetOrganizationID(organizationID).
			SetMasterBillID(targetMBL.ID).
			SetTransportExecutionID(targetTE.ID).
			SetOrderID(order.ID).
			SetStatus(seamasterbillorderlink.StatusACTIVE).
			SetDocumentStructure(documentStructure).
			SetStartedAt(time.Now().UTC()).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			return mapEntConstraint(err, "idx_sea_mbl_order_links_active_order", biz.ErrOrderStatusConflict)
		}
		return nil
	}
	if mblInput.CandidateTEID != nil || mblInput.ExpectedCandidateTEVersion != nil {
		return biz.ErrSeaMasterBillInvalidArgument
	}

	exists, err := tx.SeaMasterBill.Query().
		Where(
			seamasterbill.OrganizationIDEQ(organizationID),
			seamasterbill.ShippingLineIDEQ(shippingLineID),
			seamasterbill.NormalizedMasterNoEQ(mblInput.MasterNo),
		).
		Exist(ctx)
	if err != nil {
		return err
	}
	if exists {
		return biz.ErrSeaMasterBillConfirmationRequired
	}

	teBuilder := tx.SeaTransportExecution.Create().
		SetID(uuid.Must(uuid.NewV7())).
		SetOrganizationID(organizationID).
		SetShippingLineID(shippingLineID).
		SetVesselName(vesselName).
		SetVoyageNo(voyageNo).
		SetVersion(1)
	if input.OriginLocationID != nil {
		teBuilder.SetOriginLocationID(*input.OriginLocationID)
	}
	if input.DischargeLocationID != nil {
		teBuilder.SetDischargeLocationID(*input.DischargeLocationID)
	}
	if input.TransitLocationID != nil {
		teBuilder.SetTransitLocationID(*input.TransitLocationID)
	}
	if etd := parseOptionalTime(input.ETD); etd != nil {
		teBuilder.SetEtd(*etd)
	}
	if eta := parseOptionalTime(input.ETA); eta != nil {
		teBuilder.SetEta(*eta)
	}

	te, err := teBuilder.Save(ctx)
	if err != nil {
		return err
	}

	mblBuilder := tx.SeaMasterBill.Create().
		SetID(uuid.Must(uuid.NewV7())).
		SetOrganizationID(organizationID).
		SetShippingLineID(shippingLineID).
		SetMasterNo(mblInput.MasterNo).
		SetNormalizedMasterNo(mblInput.MasterNo).
		SetStatus(seamasterbill.StatusDRAFT).
		SetVersion(1)

	mbl, err := mblBuilder.Save(ctx)
	if err != nil {
		return mapEntConstraint(err, "seamasterbill_organization_id_shipping_line_id_normalized_master_no", biz.ErrSeaMasterBillConfirmationRequired)
	}

	_, err = tx.SeaMasterBillOrderLink.Create().
		SetID(uuid.Must(uuid.NewV7())).
		SetOrganizationID(organizationID).
		SetMasterBillID(mbl.ID).
		SetTransportExecutionID(te.ID).
		SetOrderID(order.ID).
		SetStatus(seamasterbillorderlink.StatusACTIVE).
		SetDocumentStructure(documentStructure).
		SetStartedAt(time.Now().UTC()).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		return mapEntConstraint(err, "idx_sea_mbl_order_links_active_order", biz.ErrOrderStatusConflict)
	}

	return nil
}

func syncOrderSeaMasterBillOnUpdate(
	ctx context.Context,
	tx *ent.Tx,
	organizationID uuid.UUID,
	order *ent.Order,
	input *biz.Order,
	audit *biz.AuditEvent,
	lockContext *seaMasterBillUpdateLockContext,
) error {
	if input.BusinessType != biz.OrderBusinessSE {
		return nil
	}
	contentOnlyUpdate := input.SeaMasterBillInput == nil &&
		input.SeaDocumentInput != nil && input.SeaDocumentInput.MasterBillContent != nil
	if input.SeaMasterBillInput == nil && !contentOnlyUpdate {
		return biz.ErrSeaMasterBillInvalidArgument
	}
	orderID := order.ID
	mblInput := input.SeaMasterBillInput
	if input.ShippingLineID == nil || *input.ShippingLineID == uuid.Nil {
		return biz.ErrSeaMasterBillInvalidArgument
	}
	shippingLineID := *input.ShippingLineID

	// Order 已由 UpdateDraft 首先加锁。这里先无锁读取活动关联用于定位 MBL，随后
	// 严格按 MBL → Link → TransportExecution 加写锁，并在锁后重验关联未变化。
	activeLink, err := tx.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.OrganizationIDEQ(organizationID),
			seamasterbillorderlink.OrderIDEQ(orderID),
			seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			if lockContext != nil {
				return biz.ErrSeaDocumentStructureConflict
			}
			return syncOrderSeaMasterBillOnCreate(ctx, tx, organizationID, order, input)
		}
		return err
	}
	if lockContext == nil ||
		lockContext.activeLinkID != activeLink.ID ||
		lockContext.masterBillID != activeLink.MasterBillID {
		return biz.ErrSeaDocumentStructureConflict
	}

	currentMBL, err := tx.SeaMasterBill.Query().
		Where(
			seamasterbill.IDEQ(activeLink.MasterBillID),
			seamasterbill.OrganizationIDEQ(organizationID),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}

	link, err := tx.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.IDEQ(activeLink.ID),
			seamasterbillorderlink.OrganizationIDEQ(organizationID),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}
	if link.Status != seamasterbillorderlink.StatusACTIVE || link.OrderID != orderID || link.MasterBillID != currentMBL.ID {
		return biz.ErrSeaDocumentStructureConflict
	}

	// MBL 已锁定后重验成员集合。如果定位阶段与锁定阶段之间成员发生变化，则直接
	// 回滚并由调用方重试，禁止在 MBL 锁后追加 Order 锁而破坏全局锁序。
	lockedMemberOrderIDs, err := revalidateSeaMasterBillUpdateMemberSet(
		ctx, tx, organizationID, activeLink.ID, currentMBL.ID, lockContext,
	)
	if err != nil {
		return err
	}

	currentTE, err := tx.SeaTransportExecution.Query().
		Where(
			seatransportexecution.IDEQ(activeLink.TransportExecutionID),
			seatransportexecution.OrganizationIDEQ(organizationID),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}
	if currentTE.ShippingLineID != currentMBL.ShippingLineID {
		return biz.ErrSeaMasterBillStatusConflict
	}

	if err := validateSeaMasterBillShippingLine(ctx, tx, organizationID, shippingLineID, shippingLineID != currentMBL.ShippingLineID); err != nil {
		return err
	}
	if contentOnlyUpdate {
		if shippingLineID != currentMBL.ShippingLineID || seaTransportExecutionDiffersFromOrder(currentTE, input) {
			return biz.ErrSeaMasterBillInvalidArgument
		}
		return nil
	}

	activeCount := len(lockedMemberOrderIDs)

	vesselName, voyageNo := biz.SplitVesselVoyage(input.VesselVoyage)

	identityChanged := currentMBL.ShippingLineID != shippingLineID || currentMBL.NormalizedMasterNo != mblInput.MasterNo
	voyageChanged := seaTransportExecutionDiffersFromOrder(currentTE, input)
	if activeCount > 1 && (identityChanged || voyageChanged) {
		if err := ensureLockedMembersAllowSharedMasterBillUpdate(lockContext); err != nil {
			return err
		}
	}

	if identityChanged {
		if activeCount > 1 {
			return biz.ErrSeaMasterBillCorrectionBlocked.WithMetadata(map[string]string{
				"master_bill_id":        currentMBL.ID.String(),
				"affected_member_count": stringInt(activeCount),
			})
		}
		if mblInput.ExpectedCandidateVersion == nil || currentMBL.Version != *mblInput.ExpectedCandidateVersion {
			return biz.ErrSeaMasterBillStatusConflict
		}
		if currentMBL.Status != seamasterbill.StatusDRAFT {
			return biz.ErrSeaMasterBillStatusConflict
		}
		if strings.TrimSpace(mblInput.CorrectionReason) == "" {
			return errors.BadRequest("SEA_MASTER_BILL_INVALID_ARGUMENT", "单票主单信息更正必须填写更正原因")
		}
		if err := ensureSeaMasterBillHasNoDownstreamFacts(ctx, tx, order); err != nil {
			return err
		}
		otherExists, err := tx.SeaMasterBill.Query().
			Where(
				seamasterbill.OrganizationIDEQ(organizationID),
				seamasterbill.ShippingLineIDEQ(shippingLineID),
				seamasterbill.NormalizedMasterNoEQ(mblInput.MasterNo),
				seamasterbill.IDNEQ(currentMBL.ID),
			).
			Exist(ctx)
		if err != nil {
			return err
		}
		if otherExists {
			return biz.ErrSeaMasterBillConfirmationRequired
		}
		audit.Details["sea_master_bill.id"] = currentMBL.ID.String()
		audit.Details["sea_master_bill.old_master_no"] = currentMBL.MasterNo
		audit.Details["sea_master_bill.new_master_no"] = mblInput.MasterNo
		audit.Details["sea_master_bill.old_shipping_line_id"] = currentMBL.ShippingLineID.String()
		audit.Details["sea_master_bill.new_shipping_line_id"] = shippingLineID.String()
		audit.Details["sea_master_bill.correction_reason"] = strings.TrimSpace(mblInput.CorrectionReason)

		if _, err := currentMBL.Update().
			SetShippingLineID(shippingLineID).
			SetMasterNo(mblInput.MasterNo).
			SetNormalizedMasterNo(mblInput.MasterNo).
			SetVersion(currentMBL.Version + 1).
			Save(ctx); err != nil {
			return mapEntConstraint(err, "seamasterbill_organization_id_shipping_line_id_normalized_master_no", biz.ErrSeaMasterBillConfirmationRequired)
		}

		if currentTE != nil && voyageChanged {
			teUpdate := currentTE.Update().
				SetVesselName(vesselName).
				SetVoyageNo(voyageNo).
				SetVersion(currentTE.Version + 1)
			teUpdate.SetShippingLineID(shippingLineID)
			if input.OriginLocationID != nil {
				teUpdate.SetOriginLocationID(*input.OriginLocationID)
			} else {
				teUpdate.ClearOriginLocationID()
			}
			if input.DischargeLocationID != nil {
				teUpdate.SetDischargeLocationID(*input.DischargeLocationID)
			} else {
				teUpdate.ClearDischargeLocationID()
			}
			if input.TransitLocationID != nil {
				teUpdate.SetTransitLocationID(*input.TransitLocationID)
			} else {
				teUpdate.ClearTransitLocationID()
			}
			if etd := parseOptionalTime(input.ETD); etd != nil {
				teUpdate.SetEtd(*etd)
			} else {
				teUpdate.ClearEtd()
			}
			if eta := parseOptionalTime(input.ETA); eta != nil {
				teUpdate.SetEta(*eta)
			} else {
				teUpdate.ClearEta()
			}
			if _, err := teUpdate.Save(ctx); err != nil {
				return err
			}
		}
	} else {
		if activeCount > 1 {
			if currentTE != nil {
				orderVoyage := &biz.SeaTransportExecution{
					VesselName: vesselName,
					VoyageNo:   voyageNo,
					ETD:        parseOptionalTime(input.ETD),
					ETA:        parseOptionalTime(input.ETA),
				}
				if input.ShippingLineID != nil {
					orderVoyage.ShippingLineID = *input.ShippingLineID
				}
				if input.OriginLocationID != nil {
					orderVoyage.OriginLocationID = *input.OriginLocationID
				}
				if input.DischargeLocationID != nil {
					orderVoyage.DischargeLocationID = *input.DischargeLocationID
				}
				if input.TransitLocationID != nil {
					orderVoyage.TransitLocationID = input.TransitLocationID
				}
				masterVoyageBiz := seaTransportExecutionToBiz(currentTE)
				conflicts := biz.CheckSeaVoyageConflicts(masterVoyageBiz, orderVoyage)
				if len(conflicts) > 0 {
					return biz.ErrSeaMasterBillVoyageConflict.WithMetadata(map[string]string{
						"master_bill_id":        currentMBL.ID.String(),
						"affected_member_count": stringInt(activeCount),
						"conflict_field":        conflicts[0].Field,
						"conflict_message":      conflicts[0].Message,
					})
				}
			}
		} else if activeCount == 1 {
			if currentTE != nil {
				if currentMBL.Status != seamasterbill.StatusDRAFT {
					orderVoyage := &biz.SeaTransportExecution{
						VesselName: vesselName,
						VoyageNo:   voyageNo,
						ETD:        parseOptionalTime(input.ETD),
						ETA:        parseOptionalTime(input.ETA),
					}
					if input.ShippingLineID != nil {
						orderVoyage.ShippingLineID = *input.ShippingLineID
					}
					if input.OriginLocationID != nil {
						orderVoyage.OriginLocationID = *input.OriginLocationID
					}
					if input.DischargeLocationID != nil {
						orderVoyage.DischargeLocationID = *input.DischargeLocationID
					}
					if input.TransitLocationID != nil {
						orderVoyage.TransitLocationID = input.TransitLocationID
					}
					masterVoyageBiz := seaTransportExecutionToBiz(currentTE)
					conflicts := biz.CheckSeaVoyageConflicts(masterVoyageBiz, orderVoyage)
					if len(conflicts) > 0 {
						return biz.ErrSeaMasterBillVoyageConflict.WithMetadata(map[string]string{
							"master_bill_id":   currentMBL.ID.String(),
							"conflict_field":   conflicts[0].Field,
							"conflict_message": conflicts[0].Message,
						})
					}
				} else if voyageChanged {
					if mblInput.ExpectedCandidateVersion == nil || currentMBL.Version != *mblInput.ExpectedCandidateVersion {
						return biz.ErrSeaMasterBillStatusConflict
					}
					if err := ensureSeaMasterBillHasNoDownstreamFacts(ctx, tx, order); err != nil {
						return err
					}
					teUpdate := currentTE.Update().
						SetVesselName(vesselName).
						SetVoyageNo(voyageNo).
						SetVersion(currentTE.Version + 1)
					teUpdate.SetShippingLineID(shippingLineID)
					if input.OriginLocationID != nil {
						teUpdate.SetOriginLocationID(*input.OriginLocationID)
					} else {
						teUpdate.ClearOriginLocationID()
					}
					if input.DischargeLocationID != nil {
						teUpdate.SetDischargeLocationID(*input.DischargeLocationID)
					} else {
						teUpdate.ClearDischargeLocationID()
					}
					if input.TransitLocationID != nil {
						teUpdate.SetTransitLocationID(*input.TransitLocationID)
					} else {
						teUpdate.ClearTransitLocationID()
					}
					if etd := parseOptionalTime(input.ETD); etd != nil {
						teUpdate.SetEtd(*etd)
					} else {
						teUpdate.ClearEtd()
					}
					if eta := parseOptionalTime(input.ETA); eta != nil {
						teUpdate.SetEta(*eta)
					} else {
						teUpdate.ClearEta()
					}
					if _, err := teUpdate.Save(ctx); err != nil {
						return err
					}
					if _, err := currentMBL.Update().SetVersion(currentMBL.Version + 1).Save(ctx); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func validateSeaMasterBillShippingLine(ctx context.Context, tx *ent.Tx, organizationID, shippingLineID uuid.UUID, requireEnabled bool) error {
	if shippingLineID == uuid.Nil {
		return biz.ErrSeaMasterBillInvalidArgument
	}
	predicates := []predicate.ShippingLine{
		shippinglineent.IDEQ(shippingLineID),
	}
	if requireEnabled {
		predicates = append(predicates, shippinglineent.EnabledEQ(true))
	}
	exists, err := tx.ShippingLine.Query().Where(predicates...).ForShare().Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return biz.ErrSeaMasterBillInvalidArgument
	}
	return nil
}

func seaTransportExecutionDiffersFromOrder(current *ent.SeaTransportExecution, input *biz.Order) bool {
	if current == nil {
		return false
	}
	vesselName, voyageNo := biz.SplitVesselVoyage(input.VesselVoyage)
	return input.ShippingLineID == nil || current.ShippingLineID != *input.ShippingLineID ||
		!optionalUUIDEquals(current.OriginLocationID, input.OriginLocationID) ||
		!optionalUUIDEquals(current.DischargeLocationID, input.DischargeLocationID) ||
		!optionalUUIDEquals(current.TransitLocationID, input.TransitLocationID) ||
		current.VesselName != vesselName || current.VoyageNo != voyageNo ||
		!optionalTimeEquals(current.Etd, parseOptionalTime(input.ETD)) ||
		!optionalTimeEquals(current.Eta, parseOptionalTime(input.ETA))
}

func optionalUUIDEquals(left, right *uuid.UUID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func optionalTimeEquals(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func ensureSeaMasterBillHasNoDownstreamFacts(ctx context.Context, tx *ent.Tx, order *ent.Order) error {
	if order.LockedAt != nil || order.FlowStatus != orderent.FlowStatusDRAFT {
		return biz.ErrSeaMasterBillCorrectionBlocked
	}
	hasDownstreamFacts, err := tx.Order.Query().Where(
		orderent.IDEQ(order.ID),
		orderent.Or(
			orderent.HasShippingDocumentsWith(ordershippingdocumentent.StatusNEQ(ordershippingdocumentent.StatusDRAFT)),
			orderent.HasReleasePodsWith(orderreleasepodent.OrderIDEQ(order.ID)),
			orderent.HasContainersWith(ordercontainerent.OrderIDEQ(order.ID)),
			orderent.HasMilestonesWith(ordermilestoneent.OccurredAtNotNil()),
			orderent.HasFeesWith(orderfeeent.StatusIn(orderfeeent.StatusCONFIRMED, orderfeeent.StatusBILLED)),
			orderent.HasFinanceBillLinesWith(financebilllineent.OrderIDEQ(order.ID)),
			orderent.HasFinanceCommissionLinesWith(financecommissionlineent.OrderIDEQ(order.ID)),
			orderent.HasFinanceCommissionAdjustmentsWith(financecommissionadjustmentent.OrderIDEQ(order.ID)),
		),
	).Exist(ctx)
	if err != nil {
		return err
	}
	if hasDownstreamFacts {
		return biz.ErrSeaMasterBillCorrectionBlocked
	}
	return nil
}
