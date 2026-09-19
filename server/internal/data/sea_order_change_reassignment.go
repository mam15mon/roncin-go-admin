package data

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderlifecycleeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	portent "github.com/roncin/roncin-go-admin/server/internal/data/ent/port"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seaorderreassignmenteventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaorderreassignmentevent"
	seatransportexecutionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
	seatransportexecutionversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecutionversion"
	shippinglineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/shippingline"
)

func (r *seaOrderChangeRepo) PreviewReassignment(ctx context.Context, organizationID uuid.UUID, input *biz.SeaOrderReassignmentInput) (*biz.SeaOrderReassignmentPreview, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	order, err := client.Order.Query().
		Where(orderent.IDEQ(input.OrderID), orderent.OrganizationIDEQ(organizationID)).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}

	activeLink, err := client.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlinkent.OrderIDEQ(input.OrderID),
			seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).
		WithMasterBill().
		WithTransportExecution().
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}

	curMBL := activeLink.Edges.MasterBill
	if curMBL == nil {
		return nil, biz.ErrSeaMasterBillNotFound
	}
	curSummary, err := mblToSummary(ctx, client, organizationID, curMBL, activeLink.Edges.TransportExecution)
	if err != nil {
		return nil, err
	}

	preview := &biz.SeaOrderReassignmentPreview{
		IsValid:            true,
		Errors:             []string{},
		CurrentMasterBill:  curSummary,
		Differences:        []*biz.VoyageDifference{},
		OrderVersion:       order.Version,
		CurrentLinkVersion: activeLink.Version,
	}
	// 改配 Execute 在事务内执行统一内容门禁；预览同口径提前拦截，避免预览通过、
	// 提交才被拒的体验裂缝。错误分两档与 Execute 对齐：终止/结案走改配域既定
	// SEA_ORDER_REASSIGNMENT_BLOCKED（不设 GateBlockedError，由 biz 包装）；
	// 业务锁经 GateBlockedError 原样透传，保留 409 与锁定元数据。
	if order.TerminationStatus != orderent.TerminationStatusACTIVE || order.ClosureStatus != orderent.ClosureStatusOPEN {
		preview.IsValid = false
		preview.Errors = append(preview.Errors, "订单 "+order.OrderNo+" 已终止或已结案，不允许改配")
	} else if gateErr := ensureOrderBusinessContentEditable(ctx, client.User, order); gateErr != nil {
		preview.IsValid = false
		preview.GateBlockedError = gateErr
		preview.Errors = append(preview.Errors, "订单 "+order.OrderNo+" "+orderBusinessEditBlockReason(gateErr))
	}
	var targetSummary *biz.SeaMasterBillSummary
	targetMemberCount := int32(0)

	switch input.Target.TargetType {
	case biz.SplitTargetTypeCandidate:
		if input.Target.CandidateID == nil || *input.Target.CandidateID == uuid.Nil {
			return nil, biz.ErrSeaOrderReassignmentInvalidArgument
		}
		candMBL, err := client.SeaMasterBill.Query().
			Where(
				seamasterbillent.IDEQ(*input.Target.CandidateID),
				seamasterbillent.OrganizationIDEQ(organizationID),
			).
			WithOrderLinks(func(lq *ent.SeaMasterBillOrderLinkQuery) {
				lq.Where(
					seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
					seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
				)
			}).
			Only(ctx)
		if err != nil {
			if !ent.IsNotFound(err) {
				return nil, err
			}
			preview.IsValid = false
			preview.Errors = append(preview.Errors, "目标母单不存在或不属于当前组织")
			return preview, nil
		}
		if input.Target.CandidateTEID == nil || input.Target.CandidateTEVersion == nil {
			return nil, biz.ErrSeaOrderReassignmentInvalidArgument
		}
		candidateTE, err := client.SeaTransportExecution.Query().Where(
			seatransportexecutionent.IDEQ(*input.Target.CandidateTEID),
			seatransportexecutionent.OrganizationIDEQ(organizationID),
		).Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrSeaTransportExecutionNotFound, nil)
		}
		if candidateTE == nil ||
			candMBL.Status != seamasterbillent.StatusDRAFT ||
			candMBL.Version != *input.Target.CandidateVersion ||
			candidateTE.ID != *input.Target.CandidateTEID ||
			candidateTE.Version != *input.Target.CandidateTEVersion {
			return nil, biz.ErrSeaOrderReassignmentVersionConflict
		}
		if !seaMasterBillShippingLineConsistent(candMBL, candidateTE) {
			return nil, biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
				"reason": "CANDIDATE_MBL_SHIPPING_LINE_INCONSISTENT",
			})
		}
		if candMBL.ShippingLineID != *input.Target.ShippingLineID {
			return nil, biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
				"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
			})
		}
		shippingLineEnabled, queryErr := enabledShippingLineExists(ctx, client, organizationID, candMBL.ShippingLineID, false)
		if queryErr != nil {
			return nil, queryErr
		}
		if !shippingLineEnabled {
			return nil, biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
				"reason": "CANDIDATE_MBL_SHIPPING_LINE_UNAVAILABLE",
			})
		}
		if candidateTE.ID == activeLink.TransportExecutionID || (candMBL.ID != curMBL.ID && candMBL.ShippingLineID == curMBL.ShippingLineID) || (candMBL.ID == curMBL.ID && candidateTE.ShippingLineID != curMBL.ShippingLineID) {
			preview.IsValid = false
			preview.Errors = append(preview.Errors, "改配目标与当前 MBL/实际航次组合不符合船公司变更规则")
		}
		// 批次内排重（含作废行，一号一案）：预览与执行同口径，本单分单号在目标
		// 批次已存在时提前阻断，避免预览通过、提交才被拒的体验裂缝。
		orderHBLs, hblErr := client.SeaHouseBill.Query().
			Where(seahousebillent.OrderIDEQ(order.ID)).
			All(ctx)
		if hblErr != nil {
			return nil, hblErr
		}
		for _, h := range orderHBLs {
			conflictExists, conflictErr := client.SeaHouseBill.Query().
				Where(
					seahousebillent.MasterBillIDEQ(candMBL.ID),
					seahousebillent.NormalizedHouseNoEQ(h.NormalizedHouseNo),
					seahousebillent.IDNEQ(h.ID),
				).
				Exist(ctx)
			if conflictErr != nil {
				return nil, conflictErr
			}
			if conflictExists {
				preview.IsValid = false
				preview.Errors = append(preview.Errors,
					"分单号 "+h.HouseNo+" 在目标主单批次内已存在（含作废），请更换改配目标或先调整分单号")
			}
		}
		targetSummary, err = mblToSummary(ctx, client, organizationID, candMBL, candidateTE)
		if err != nil {
			return nil, err
		}
		targetMemberCount = int32(len(candMBL.Edges.OrderLinks))
	case biz.SplitTargetTypeNew:
		splitTarget := &biz.SeaOrderSplitTargetInput{
			MasterNo:            input.Target.MasterNo,
			ShippingLineID:      input.Target.ShippingLineID,
			VesselName:          input.Target.VesselName,
			VoyageNo:            input.Target.VoyageNo,
			ETD:                 input.Target.ETD,
			ETA:                 input.Target.ETA,
			OriginLocationID:    input.Target.OriginLocationID,
			DischargeLocationID: input.Target.DischargeLocationID,
			TransitLocationID:   input.Target.TransitLocationID,
		}
		normalizedMasterNo, err := biz.ValidateAndNormalizeSeaMasterNo(splitTarget.MasterNo)
		if err != nil {
			return nil, err
		}
		reuseCurrentMBL := input.Target.ShippingLineID != nil && *input.Target.ShippingLineID == curMBL.ShippingLineID && normalizedMasterNo == curMBL.NormalizedMasterNo
		if reuseCurrentMBL {
			if err := validateTransportExecutionTargetInput(ctx, client, organizationID, splitTarget, false); err != nil {
				return nil, err
			}
		} else {
			if input.Target.ShippingLineID != nil && *input.Target.ShippingLineID == curMBL.ShippingLineID {
				return nil, biz.ErrSeaOrderReassignmentTargetConflict
			}
			normalizedMasterNo, err = validateNewMasterBillInput(ctx, client, organizationID, splitTarget, false)
			if err != nil {
				return nil, err
			}
		}
		targetSummary = &biz.SeaMasterBillSummary{
			MasterNo:   normalizedMasterNo,
			VesselName: input.Target.VesselName,
			VoyageNo:   input.Target.VoyageNo,
			ETD:        input.Target.ETD,
			ETA:        input.Target.ETA,
		}
		if reuseCurrentMBL {
			targetSummary.MasterBillID = curMBL.ID
			targetSummary.Version = curMBL.Version
			targetSummary.Status = string(curMBL.Status)
		}
		if input.Target.ShippingLineID != nil {
			targetSummary.ShippingLineID = *input.Target.ShippingLineID
			line, err := client.ShippingLine.Query().Where(
				shippinglineent.IDEQ(*input.Target.ShippingLineID),
			).Only(ctx)
			if err != nil {
				return nil, err
			}
			targetSummary.ShippingLineName = formatShippingLineName(line.NameZh, line.NameEn, line.ScacCode)
		}
		if input.Target.OriginLocationID != nil {
			targetSummary.OriginLocationID = input.Target.OriginLocationID
			p, err := client.Port.Query().Where(
				portent.IDEQ(*input.Target.OriginLocationID),
				portent.OrganizationIDEQ(organizationID),
			).Only(ctx)
			if err != nil {
				return nil, err
			}
			targetSummary.OriginLocationName = p.NameEn
		}
		if input.Target.DischargeLocationID != nil {
			targetSummary.DischargeLocationID = input.Target.DischargeLocationID
			p, err := client.Port.Query().Where(
				portent.IDEQ(*input.Target.DischargeLocationID),
				portent.OrganizationIDEQ(organizationID),
			).Only(ctx)
			if err != nil {
				return nil, err
			}
			targetSummary.DischargeLocationName = p.NameEn
		}
		if input.Target.TransitLocationID != nil {
			targetSummary.TransitLocationID = input.Target.TransitLocationID
			p, err := client.Port.Query().Where(
				portent.IDEQ(*input.Target.TransitLocationID),
				portent.OrganizationIDEQ(organizationID),
			).Only(ctx)
			if err != nil {
				return nil, err
			}
			targetSummary.TransitLocationName = p.NameEn
		}
	default:
		return nil, biz.ErrSeaOrderReassignmentInvalidArgument
	}

	preview.TargetMasterBill = targetSummary
	preview.TargetMemberCount = targetMemberCount

	preview.Differences = append(preview.Differences,
		makeDiff("master_no", "提单号(MBL)", curSummary.MasterNo, targetSummary.MasterNo),
		makeDiff("shipping_line_name", "承运人/船东", curSummary.ShippingLineName, targetSummary.ShippingLineName),
		makeDiff("vessel_name", "船名", curSummary.VesselName, targetSummary.VesselName),
		makeDiff("voyage_no", "航次", curSummary.VoyageNo, targetSummary.VoyageNo),
		makeDiff("origin_location_name", "起运港(POL)", curSummary.OriginLocationName, targetSummary.OriginLocationName),
		makeDiff("discharge_location_name", "卸货港(POD)", curSummary.DischargeLocationName, targetSummary.DischargeLocationName),
		makeDiff("transit_location_name", "中转港", curSummary.TransitLocationName, targetSummary.TransitLocationName),
		makeDiff("etd", "预计开航时间(ETD)", curSummary.ETD, targetSummary.ETD),
		makeDiff("eta", "预计到港时间(ETA)", curSummary.ETA, targetSummary.ETA),
	)

	_ = order
	return preview, nil
}

func (r *seaOrderChangeRepo) ExecuteReassignment(ctx context.Context, organizationID, actorID uuid.UUID, input *biz.SeaOrderReassignmentInput, audit *biz.AuditEvent) (*biz.SeaOrderReassignmentEvent, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	existingEvent, err := client.SeaOrderReassignmentEvent.Query().
		Where(
			seaorderreassignmenteventent.OrganizationIDEQ(organizationID),
			seaorderreassignmenteventent.IdempotencyKeyEQ(input.IdempotencyKey),
		).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}
	if err == nil && existingEvent != nil {
		if existingEvent.RequestFingerprint != input.RequestFingerprint {
			return nil, biz.ErrSeaOrderReassignmentIdempotencyConflict
		}
		return &biz.SeaOrderReassignmentEvent{
			ID:                   existingEvent.ID,
			CreatedAt:            existingEvent.CreatedAt,
			OrganizationID:       existingEvent.OrganizationID,
			OrderID:              existingEvent.OrderID,
			OrderNo:              existingEvent.OrderNo,
			IdempotencyKey:       existingEvent.IdempotencyKey,
			RequestFingerprint:   existingEvent.RequestFingerprint,
			PreviousMasterBillID: existingEvent.PreviousMasterBillID,
			TargetMasterBillID:   existingEvent.TargetMasterBillID,
			TargetLinkID:         existingEvent.TargetLinkID,
			Reason:               existingEvent.Reason,
			ResponsibilityType:   string(existingEvent.ResponsibilityType),
			Confirmation:         externalConfirmationFromReassignment(existingEvent),
		}, nil
	}

	var reassignmentResult *biz.SeaOrderReassignmentEvent
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 锁序 1: Order ForUpdate
		order, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(input.OrderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		if input.ExpectedOrderVersion == 0 || order.Version != input.ExpectedOrderVersion {
			return biz.ErrSeaOrderReassignmentVersionConflict
		}

		if order.BusinessType != orderent.BusinessTypeSE {
			return biz.ErrSeaOrderReassignmentBlocked
		}
		if order.TerminationStatus != orderent.TerminationStatusACTIVE || order.ClosureStatus != orderent.ClosureStatusOPEN {
			return biz.ErrSeaOrderReassignmentBlocked
		}
		// 改配属订单业务内容写入：在既有终止/结案检查之上执行统一内容门禁，
		// 业务锁定的订单不允许改配（ORDER_BUSINESS_LOCKED，409）。
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}

		// C.1: 锁序改造：先无锁定位 Link，仅作 ID 定位
		unlockedLink, linkErr := tx.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrderIDEQ(order.ID),
				seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Select(seamasterbillorderlinkent.FieldID, seamasterbillorderlinkent.FieldMasterBillID).
			Only(ctx)
		if linkErr != nil {
			if ent.IsNotFound(linkErr) {
				return biz.ErrSeaOrderReassignmentBlocked
			}
			return linkErr
		}

		// 按 UUID 升序收集并锁定 MBL
		mblIDs := []uuid.UUID{unlockedLink.MasterBillID}
		if input.Target.TargetType == biz.SplitTargetTypeCandidate && input.Target.CandidateID != nil && *input.Target.CandidateID != uuid.Nil {
			mblIDs = append(mblIDs, *input.Target.CandidateID)
		}
		mblIDs = sortAndDeduplicateUUIDs(mblIDs)

		mbls := make(map[uuid.UUID]*ent.SeaMasterBill, len(mblIDs))
		for _, mID := range mblIDs {
			m, err := tx.SeaMasterBill.Query().
				Where(seamasterbillent.IDEQ(mID), seamasterbillent.OrganizationIDEQ(organizationID)).
				ForUpdate().
				Only(ctx)
			if err != nil {
				return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
			}
			mbls[mID] = m
		}

		// 锁定 Link 并重验它仍为该 Order 唯一 ACTIVE 且 master_bill_id/status/org/version 未变
		oldLink, linkErr := tx.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlinkent.IDEQ(unlockedLink.ID)).
			ForUpdate().
			Only(ctx)
		if linkErr != nil {
			if ent.IsNotFound(linkErr) {
				return biz.ErrSeaOrderReassignmentBlocked
			}
			return linkErr
		}
		if oldLink.OrderID != order.ID || oldLink.OrganizationID != organizationID ||
			oldLink.Status != seamasterbillorderlinkent.StatusACTIVE || oldLink.MasterBillID != unlockedLink.MasterBillID {
			return biz.ErrSeaOrderReassignmentBlocked
		}
		activeLinkCount, err := tx.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrderIDEQ(order.ID),
				seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Count(ctx)
		if err != nil {
			return err
		}
		if activeLinkCount != 1 {
			return biz.ErrSeaOrderReassignmentBlocked
		}
		if input.ExpectedLinkVersion == 0 || oldLink.Version != input.ExpectedLinkVersion {
			return biz.ErrSeaOrderReassignmentVersionConflict
		}

		oldMBL := mbls[oldLink.MasterBillID]
		if oldMBL == nil || oldMBL.Status != seamasterbillent.StatusDRAFT {
			return biz.ErrSeaOrderReassignmentBlocked
		}

		// 收集并锁定 TE
		teIDs := []uuid.UUID{oldLink.TransportExecutionID}
		if input.Target.TargetType == biz.SplitTargetTypeCandidate && input.Target.CandidateTEID != nil && *input.Target.CandidateTEID != uuid.Nil {
			teIDs = append(teIDs, *input.Target.CandidateTEID)
		}
		teIDs = sortAndDeduplicateUUIDs(teIDs)
		lockedTEs := make(map[uuid.UUID]*ent.SeaTransportExecution, len(teIDs))
		for _, tid := range teIDs {
			te, err := tx.SeaTransportExecution.Query().
				Where(seatransportexecutionent.IDEQ(tid), seatransportexecutionent.OrganizationIDEQ(organizationID)).
				ForUpdate().
				Only(ctx)
			if err != nil {
				return err
			}
			lockedTEs[tid] = te
		}
		oldTE := lockedTEs[oldLink.TransportExecutionID]
		if err := validateConfirmationAttachment(ctx, tx.Client(), organizationID, order.ID, input.Confirmation); err != nil {
			return err
		}

		var targetMBLID uuid.UUID
		var targetTEID uuid.UUID
		var targetMBLNo string
		var targetTE *ent.SeaTransportExecution

		switch input.Target.TargetType {
		case biz.SplitTargetTypeCandidate:
			if input.Target.CandidateID == nil || *input.Target.CandidateID == uuid.Nil {
				return biz.ErrSeaOrderReassignmentInvalidArgument
			}
			targetMBLID = *input.Target.CandidateID
			targetMBL := mbls[targetMBLID]
			if targetMBL == nil || input.Target.CandidateVersion == nil || targetMBL.Version != *input.Target.CandidateVersion ||
				input.Target.CandidateTEID == nil || *input.Target.CandidateTEID == uuid.Nil {
				return biz.ErrSeaOrderReassignmentVersionConflict
			}
			if targetMBL.Status != seamasterbillent.StatusDRAFT {
				return biz.ErrSeaOrderReassignmentBlocked
			}
			if input.ExpectedCandidateMBLVersion == nil || *input.ExpectedCandidateMBLVersion == 0 || targetMBL.Version != *input.ExpectedCandidateMBLVersion {
				return biz.ErrSeaOrderReassignmentVersionConflict
			}
			targetMBLNo = targetMBL.MasterNo
			targetTEID = *input.Target.CandidateTEID
			targetTE = lockedTEs[targetTEID]
			if targetTE == nil {
				return biz.ErrSeaTransportExecutionNotFound
			}
			if !seaMasterBillShippingLineConsistent(targetMBL, targetTE) {
				return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_SHIPPING_LINE_INCONSISTENT",
				})
			}
			shippingLineEnabled, err := enabledShippingLineExists(ctx, tx.Client(), organizationID, targetMBL.ShippingLineID, true)
			if err != nil {
				return err
			}
			if !shippingLineEnabled {
				return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_SHIPPING_LINE_UNAVAILABLE",
				})
			}
			if input.Target.CandidateTEVersion == nil || targetTE.Version != *input.Target.CandidateTEVersion {
				return biz.ErrSeaOrderReassignmentVersionConflict
			}

			if input.ExpectedCandidateTEVersion == nil || *input.ExpectedCandidateTEVersion == 0 || targetTE.Version != *input.ExpectedCandidateTEVersion {
				return biz.ErrSeaOrderReassignmentVersionConflict
			}

			// C.2: 校验用户目标输入与 candidate authoritative TE/MBL 是否一致 (禁止比对源订单 origin/discharge)
			if input.Target.MasterNo != "" {
				normInputMasterNo, err := biz.ValidateAndNormalizeSeaMasterNo(input.Target.MasterNo)
				if err != nil {
					return err
				}
				if targetMBL.NormalizedMasterNo != normInputMasterNo {
					return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
			}
			if input.Target.ShippingLineID != nil && *input.Target.ShippingLineID != uuid.Nil && targetMBL.ShippingLineID != *input.Target.ShippingLineID {
				return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
				})
			}
			if input.Target.ShippingLineID != nil && *input.Target.ShippingLineID != uuid.Nil && targetTE.ShippingLineID != *input.Target.ShippingLineID {
				return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
				})
			}
			if input.Target.VesselName != "" && targetTE.VesselName != input.Target.VesselName {
				return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
				})
			}
			if input.Target.VoyageNo != "" && targetTE.VoyageNo != input.Target.VoyageNo {
				return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
				})
			}
			if input.Target.OriginLocationID != nil && *input.Target.OriginLocationID != uuid.Nil && (targetTE.OriginLocationID == nil || *targetTE.OriginLocationID != *input.Target.OriginLocationID) {
				return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
				})
			}
			if input.Target.DischargeLocationID != nil && *input.Target.DischargeLocationID != uuid.Nil && (targetTE.DischargeLocationID == nil || *targetTE.DischargeLocationID != *input.Target.DischargeLocationID) {
				return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
				})
			}
			if input.Target.TransitLocationID != nil && *input.Target.TransitLocationID != uuid.Nil && (targetTE.TransitLocationID == nil || *targetTE.TransitLocationID != *input.Target.TransitLocationID) {
				return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
				})
			}
			if input.Target.ETD != "" {
				etdTime := parseOptionalTime(input.Target.ETD)
				if etdTime == nil || targetTE.Etd == nil || !etdTime.Equal(*targetTE.Etd) {
					return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
			}
			if input.Target.ETA != "" {
				etaTime := parseOptionalTime(input.Target.ETA)
				if etaTime == nil || targetTE.Eta == nil || !etaTime.Equal(*targetTE.Eta) {
					return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
			}
		case biz.SplitTargetTypeNew:
			splitTargetInput := &biz.SeaOrderSplitTargetInput{
				MasterNo:            input.Target.MasterNo,
				ShippingLineID:      input.Target.ShippingLineID,
				VesselName:          input.Target.VesselName,
				VoyageNo:            input.Target.VoyageNo,
				ETD:                 input.Target.ETD,
				ETA:                 input.Target.ETA,
				OriginLocationID:    input.Target.OriginLocationID,
				DischargeLocationID: input.Target.DischargeLocationID,
				TransitLocationID:   input.Target.TransitLocationID,
			}
			newMBL, newTE, err := createNewMasterBillInTx(ctx, tx, organizationID, splitTargetInput)
			if err != nil {
				return err
			}
			targetMBLID = newMBL.ID
			targetTEID = newTE.ID
			targetMBLNo = newMBL.MasterNo
			targetTE = newTE
		default:
			return biz.ErrSeaOrderReassignmentInvalidArgument
		}
		if targetTEID == oldTE.ID {
			return biz.ErrSeaOrderReassignmentTargetConflict
		}
		if targetMBLID == oldMBL.ID && targetTE.ShippingLineID != oldMBL.ShippingLineID {
			return biz.ErrSeaOrderReassignmentTargetConflict
		}
		if targetMBLID != oldMBL.ID && targetTE.ShippingLineID == oldMBL.ShippingLineID {
			return biz.ErrSeaOrderReassignmentTargetConflict
		}
		if _, err := ensureSeaTransportExecutionVersion(
			ctx, tx, organizationID, targetTE, &actorID,
			seatransportexecutionversionent.SourceREASSIGNMENT,
			&input.Reason, nil, nil, input.Confirmation,
		); err != nil {
			return err
		}

		now := time.Now().UTC()
		if _, err := tx.SeaMasterBillOrderLink.UpdateOneID(oldLink.ID).
			SetStatus(seamasterbillorderlinkent.StatusENDED).
			SetEndedAt(now).
			SetVersion(oldLink.Version + 1).
			Save(ctx); err != nil {
			return err
		}

		newLinkBuilder := tx.SeaMasterBillOrderLink.Create().
			SetID(uuid.Must(uuid.NewV7())).
			SetOrganizationID(organizationID).
			SetMasterBillID(targetMBLID).
			SetTransportExecutionID(targetTEID).
			SetOrderID(order.ID).
			SetDocumentStructure(oldLink.DocumentStructure).
			SetStatus(seamasterbillorderlinkent.StatusACTIVE).
			SetStartedAt(now).
			SetVersion(1)
		newLink, err := newLinkBuilder.Save(ctx)
		if err != nil {
			return err
		}

		// 锁定并迁移该订单现有的分单 HBL 至目标 MBL (UUID 升序锁定)
		hbls, err := tx.SeaHouseBill.Query().
			Where(seahousebillent.OrderIDEQ(order.ID)).
			Order(seahousebillent.ByID()).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}
		// 批次内排重（含作废行，一号一案）：改挂前与目标批次既有分单比对并排除
		// 自身，冲突给出明确业务错误；唯一索引作并发兜底。
		for _, h := range hbls {
			conflictExists, err := tx.SeaHouseBill.Query().
				Where(
					seahousebillent.MasterBillIDEQ(targetMBLID),
					seahousebillent.NormalizedHouseNoEQ(h.NormalizedHouseNo),
					seahousebillent.IDNEQ(h.ID),
				).
				Exist(ctx)
			if err != nil {
				return err
			}
			if conflictExists {
				return biz.MetadataError(biz.ErrSeaHouseBillExists, map[string]string{
					"reason":         "HOUSE_BILL_DUPLICATE",
					"house_no":       h.HouseNo,
					"master_bill_id": targetMBLID.String(),
				})
			}
		}
		for _, h := range hbls {
			if _, err := tx.SeaHouseBill.UpdateOneID(h.ID).
				SetMasterBillID(targetMBLID).
				SetVersion(h.Version + 1).
				Save(ctx); err != nil {
				if ent.IsConstraintError(err) {
					return biz.ErrSeaHouseBillExists
				}
				return err
			}
		}

		beforeSnapshotMap := map[string]interface{}{
			"schema_version": 1,
			"link": map[string]interface{}{
				"id":                     oldLink.ID,
				"version":                oldLink.Version,
				"status":                 oldLink.Status,
				"transport_execution_id": oldLink.TransportExecutionID,
			},
			"master_bill": map[string]interface{}{
				"id":               oldMBL.ID,
				"master_no":        oldMBL.MasterNo,
				"shipping_line_id": oldMBL.ShippingLineID,
				"version":          oldMBL.Version,
				"status":           oldMBL.Status,
			},
			"transport_execution": map[string]interface{}{
				"id":                    oldTE.ID,
				"version":               oldTE.Version,
				"vessel_name":           oldTE.VesselName,
				"voyage_no":             oldTE.VoyageNo,
				"shipping_line_id":      oldTE.ShippingLineID,
				"origin_location_id":    oldTE.OriginLocationID,
				"discharge_location_id": oldTE.DischargeLocationID,
				"transit_location_id":   oldTE.TransitLocationID,
				"etd":                   oldTE.Etd,
				"eta":                   oldTE.Eta,
			},
		}
		afterMBLVersion := uint64(1)
		afterMBLIssuer := uuid.Nil
		afterMBLStatus := "DRAFT"
		if cand, ok := mbls[targetMBLID]; ok && cand != nil {
			afterMBLVersion = cand.Version
			afterMBLIssuer = cand.ShippingLineID
			afterMBLStatus = string(cand.Status)
		} else if input.Target.ShippingLineID != nil {
			afterMBLIssuer = *input.Target.ShippingLineID
		}

		afterSnapshotMap := map[string]interface{}{
			"schema_version": 1,
			"link": map[string]interface{}{
				"id":                     newLink.ID,
				"version":                newLink.Version,
				"status":                 newLink.Status,
				"transport_execution_id": newLink.TransportExecutionID,
			},
			"master_bill": map[string]interface{}{
				"id":               targetMBLID,
				"master_no":        targetMBLNo,
				"shipping_line_id": afterMBLIssuer,
				"version":          afterMBLVersion,
				"status":           afterMBLStatus,
			},
			"transport_execution": map[string]interface{}{
				"id":                    targetTE.ID,
				"version":               targetTE.Version,
				"vessel_name":           targetTE.VesselName,
				"voyage_no":             targetTE.VoyageNo,
				"shipping_line_id":      targetTE.ShippingLineID,
				"origin_location_id":    targetTE.OriginLocationID,
				"discharge_location_id": targetTE.DischargeLocationID,
				"transit_location_id":   targetTE.TransitLocationID,
				"etd":                   targetTE.Etd,
				"eta":                   targetTE.Eta,
			},
		}
		beforeSnapshotBytes, err := json.Marshal(beforeSnapshotMap)
		if err != nil {
			return err
		}
		afterSnapshotBytes, err := json.Marshal(afterSnapshotMap)
		if err != nil {
			return err
		}

		if _, err := tx.Order.UpdateOneID(order.ID).SetVersion(order.Version + 1).Save(ctx); err != nil {
			return err
		}

		reassignEvtID := uuid.Must(uuid.NewV7())
		reassignBuilder := tx.SeaOrderReassignmentEvent.Create().
			SetID(reassignEvtID).
			SetOrganizationID(organizationID).
			SetOrderID(order.ID).
			SetOrderNo(order.OrderNo).
			SetIdempotencyKey(input.IdempotencyKey).
			SetRequestFingerprint(input.RequestFingerprint).
			SetPreviousMasterBillID(oldMBL.ID).
			SetTargetMasterBillID(targetMBLID).
			SetPreviousTransportExecutionID(oldTE.ID).
			SetTargetTransportExecutionID(targetTE.ID).
			SetPreviousLinkID(oldLink.ID).
			SetTargetLinkID(newLink.ID).
			SetPreviousLinkVersion(oldLink.Version).
			SetTargetLinkVersion(1).
			SetReason(input.Reason).
			SetResponsibilityType(seaorderreassignmenteventent.ResponsibilityType(input.ResponsibilityType)).
			SetBeforeSnapshot(beforeSnapshotBytes).
			SetAfterSnapshot(afterSnapshotBytes).
			SetCreatedBy(actorID).
			SetConfirmedByParty(input.Confirmation.ConfirmedByParty).
			SetConfirmedAt(input.Confirmation.ConfirmedAt).
			SetConfirmationNote(input.Confirmation.ConfirmationNote).
			SetNillableConfirmationAttachmentID(input.Confirmation.ConfirmationAttachmentID)

		if input.ResponsiblePartnerID != nil && *input.ResponsiblePartnerID != uuid.Nil {
			p, err := tx.Partner.Query().
				Where(partnerent.IDEQ(*input.ResponsiblePartnerID), partnerent.OrganizationIDEQ(organizationID)).
				Only(ctx)
			if err != nil {
				return err
			}
			reassignBuilder.SetResponsiblePartnerID(p.ID)
			reassignBuilder.SetResponsiblePartnerName(p.LegalName)
		}

		savedEvent, err := reassignBuilder.Save(ctx)
		if err != nil {
			return mapEntConstraint(err, "sea_order_reassignment_event_idempotency_key", biz.ErrSeaOrderReassignmentIdempotencyConflict)
		}

		if _, err := tx.OrderLifecycleEvent.Create().
			SetOrderID(order.ID).
			SetDimension(orderlifecycleeventent.DimensionFLOW).
			SetToStatus(string(order.FlowStatus)).
			SetAction("REASSIGNED_MBL").
			SetReason(input.Reason).
			SetReferenceType("SEA_ORDER_REASSIGNMENT_EVENT").
			SetReferenceID(reassignEvtID).
			SetOperatorID(actorID).
			Save(ctx); err != nil {
			return err
		}

		reassignmentResult = &biz.SeaOrderReassignmentEvent{
			ID:                   savedEvent.ID,
			CreatedAt:            savedEvent.CreatedAt,
			OrganizationID:       savedEvent.OrganizationID,
			OrderID:              savedEvent.OrderID,
			OrderNo:              savedEvent.OrderNo,
			IdempotencyKey:       savedEvent.IdempotencyKey,
			RequestFingerprint:   savedEvent.RequestFingerprint,
			PreviousMasterBillID: savedEvent.PreviousMasterBillID,
			TargetMasterBillID:   savedEvent.TargetMasterBillID,
			TargetLinkID:         savedEvent.TargetLinkID,
			Reason:               savedEvent.Reason,
			ResponsibilityType:   string(savedEvent.ResponsibilityType),
			Confirmation:         externalConfirmationFromReassignment(savedEvent),
		}

		audit.Details["reassignment_event_id"] = reassignEvtID.String()
		audit.Details["target_master_no"] = targetMBLNo
		audit.Details["reason"] = input.Reason
		return writeAudit(ctx, tx.AuditLog, audit)
	})

	if err != nil {
		return nil, err
	}
	return reassignmentResult, nil
}

func (r *seaOrderChangeRepo) GetReassignmentEventByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*biz.SeaOrderReassignmentEvent, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	existingEvent, err := client.SeaOrderReassignmentEvent.Query().
		Where(
			seaorderreassignmenteventent.OrganizationIDEQ(organizationID),
			seaorderreassignmenteventent.IdempotencyKeyEQ(idempotencyKey),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &biz.SeaOrderReassignmentEvent{
		ID:                           existingEvent.ID,
		CreatedAt:                    existingEvent.CreatedAt,
		OrganizationID:               existingEvent.OrganizationID,
		OrderID:                      existingEvent.OrderID,
		OrderNo:                      existingEvent.OrderNo,
		SplitEventID:                 existingEvent.SplitEventID,
		SplitResultID:                existingEvent.SplitResultID,
		IdempotencyKey:               existingEvent.IdempotencyKey,
		RequestFingerprint:           existingEvent.RequestFingerprint,
		PreviousMasterBillID:         existingEvent.PreviousMasterBillID,
		TargetMasterBillID:           existingEvent.TargetMasterBillID,
		PreviousTransportExecutionID: existingEvent.PreviousTransportExecutionID,
		TargetTransportExecutionID:   existingEvent.TargetTransportExecutionID,
		PreviousLinkID:               existingEvent.PreviousLinkID,
		TargetLinkID:                 existingEvent.TargetLinkID,
		PreviousLinkVersion:          existingEvent.PreviousLinkVersion,
		TargetLinkVersion:            existingEvent.TargetLinkVersion,
		Reason:                       existingEvent.Reason,
		ResponsibilityType:           string(existingEvent.ResponsibilityType),
		ResponsiblePartnerID:         existingEvent.ResponsiblePartnerID,
		BeforeSnapshot:               existingEvent.BeforeSnapshot,
		AfterSnapshot:                existingEvent.AfterSnapshot,
		CreatedBy:                    existingEvent.CreatedBy,
		Confirmation:                 externalConfirmationFromReassignment(existingEvent),
	}, nil
}

func (r *seaOrderChangeRepo) GetReassignmentEvent(ctx context.Context, organizationID, orderID, eventID uuid.UUID) (*biz.SeaOrderReassignmentEvent, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	existingEvent, err := client.SeaOrderReassignmentEvent.Query().
		Where(
			seaorderreassignmenteventent.IDEQ(eventID),
			seaorderreassignmenteventent.OrganizationIDEQ(organizationID),
			seaorderreassignmenteventent.OrderIDEQ(orderID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrSeaOrderReassignmentInvalidArgument
		}
		return nil, err
	}
	return &biz.SeaOrderReassignmentEvent{
		ID:                           existingEvent.ID,
		CreatedAt:                    existingEvent.CreatedAt,
		OrganizationID:               existingEvent.OrganizationID,
		OrderID:                      existingEvent.OrderID,
		OrderNo:                      existingEvent.OrderNo,
		SplitEventID:                 existingEvent.SplitEventID,
		SplitResultID:                existingEvent.SplitResultID,
		IdempotencyKey:               existingEvent.IdempotencyKey,
		RequestFingerprint:           existingEvent.RequestFingerprint,
		PreviousMasterBillID:         existingEvent.PreviousMasterBillID,
		TargetMasterBillID:           existingEvent.TargetMasterBillID,
		PreviousTransportExecutionID: existingEvent.PreviousTransportExecutionID,
		TargetTransportExecutionID:   existingEvent.TargetTransportExecutionID,
		PreviousLinkID:               existingEvent.PreviousLinkID,
		TargetLinkID:                 existingEvent.TargetLinkID,
		PreviousLinkVersion:          existingEvent.PreviousLinkVersion,
		TargetLinkVersion:            existingEvent.TargetLinkVersion,
		Reason:                       existingEvent.Reason,
		ResponsibilityType:           string(existingEvent.ResponsibilityType),
		ResponsiblePartnerID:         existingEvent.ResponsiblePartnerID,
		ResponsiblePartnerName:       existingEvent.ResponsiblePartnerName,
		Confirmation:                 externalConfirmationFromReassignment(existingEvent),
		BeforeSnapshot:               existingEvent.BeforeSnapshot,
		AfterSnapshot:                existingEvent.AfterSnapshot,
		CreatedBy:                    existingEvent.CreatedBy,
	}, nil
}
