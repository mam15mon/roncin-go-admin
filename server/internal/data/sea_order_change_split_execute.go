package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	financecommissionlineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderattachmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderattachment"
	ordercargoitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargoitem"
	ordercommissionattributionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	ordercontainerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainer"
	ordercontainerrequestent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainerrequest"
	orderenterprisetagent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderenterprisetag"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderfeeenterprisetagent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeeenterprisetag"
	orderlifecycleeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
	orderpersonnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seaorderreassignmenteventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaorderreassignmentevent"
	seaorderspliteventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaordersplitevent"
	seaordersplitresultent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaordersplitresult"
	seasharedcontainerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seasharedcontainer"
	seasharedcontainerallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seasharedcontainerallocation"
	seatransportexecutionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
)

func (r *seaOrderChangeRepo) ExecuteSplit(ctx context.Context, organizationID, actorID uuid.UUID, input *biz.SeaOrderSplitInput, audit *biz.AuditEvent) (*biz.SeaOrderSplitEvent, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	existingEvent, err := client.SeaOrderSplitEvent.Query().
		Where(
			seaorderspliteventent.OrganizationIDEQ(organizationID),
			seaorderspliteventent.IdempotencyKeyEQ(input.IdempotencyKey),
		).
		WithResults().
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}
	if err == nil && existingEvent != nil {
		if existingEvent.RequestFingerprint != input.RequestFingerprint {
			return nil, biz.ErrSeaOrderSplitIdempotencyConflict
		}
		res := &biz.SeaOrderSplitEvent{
			ID:                 existingEvent.ID,
			CreatedAt:          existingEvent.CreatedAt,
			OrganizationID:     existingEvent.OrganizationID,
			SourceOrderID:      existingEvent.SourceOrderID,
			SourceOrderNo:      existingEvent.SourceOrderNo,
			IdempotencyKey:     existingEvent.IdempotencyKey,
			RequestFingerprint: existingEvent.RequestFingerprint,
			Note:               existingEvent.Note,
			BeforeSnapshot:     existingEvent.BeforeSnapshot,
			CreatedBy:          existingEvent.CreatedBy,
		}
		for _, r := range existingEvent.Edges.Results {
			res.Results = append(res.Results, &biz.SeaOrderSplitResult{
				ID:                  r.ID,
				CreatedAt:           r.CreatedAt,
				SplitEventID:        r.SplitEventID,
				OrganizationID:      r.OrganizationID,
				OrderID:             r.OrderID,
				OrderNo:             r.OrderNo,
				ResultRole:          string(r.ResultRole),
				Sequence:            r.Sequence,
				ClientResultKey:     r.ClientResultKey,
				InitialMasterBillID: r.InitialMasterBillID,
				FinalMasterBillID:   r.FinalMasterBillID,
				ResultSnapshot:      r.ResultSnapshot,
			})
		}
		return res, nil
	}

	var splitEventResult *biz.SeaOrderSplitEvent
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 锁序 1: 锁定源订单 Order
		sourceOrder, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(input.OrderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		if err := ensureOrderBusinessEditable(ctx, tx, sourceOrder); err != nil {
			return err
		}

		if input.ExpectedVersions == nil || input.ExpectedVersions.OrderVersion == 0 || sourceOrder.Version != input.ExpectedVersions.OrderVersion {
			return biz.ErrSeaOrderSplitVersionConflict
		}

		// 内嵌改配目标必须携带外部确认（领域层已校验，锁内兜底防 nil 解引用）
		hasNonCurrentTarget := false
		for _, t := range input.Targets {
			if t != nil && t.TargetType != biz.SplitTargetTypeCurrent {
				hasNonCurrentTarget = true
				break
			}
		}
		if hasNonCurrentTarget && input.Confirmation == nil {
			return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
				"reason": "CONFIRMATION_REQUIRED",
			})
		}

		if sourceOrder.BusinessType != orderent.BusinessTypeSE {
			return biz.ErrSeaOrderSplitBlocked
		}
		if sourceOrder.TerminationStatus != orderent.TerminationStatusACTIVE || sourceOrder.ClosureStatus != orderent.ClosureStatusOPEN {
			return biz.ErrSeaOrderSplitBlocked
		}

		// 锁序 2: 组织订单号码序列：预先分配订单号
		allocatedAt := time.Now().UTC()
		createdOrdersCount := 0
		for _, r := range input.Results {
			if r.ResultRole == biz.ResultRoleCreated {
				createdOrdersCount++
			}
		}

		createdOrderNumbers := make(map[string]string, createdOrdersCount)
		for _, r := range input.Results {
			if r.ResultRole == biz.ResultRoleCreated {
				rule, sequence, allocErr := allocateNumberInTx(ctx, tx, organizationID, biz.DocumentTypeOrder, allocatedAt)
				if allocErr != nil {
					return allocErr
				}
				num, fmtErr := biz.FormatAllocatedNumber(allocatedAt, rule, sequence, string(biz.OrderBusinessSE))
				if fmtErr != nil {
					return fmtErr
				}
				createdOrderNumbers[r.ClientResultKey] = num
			}
		}

		// 锁序 3-5: 涉及的 MBL, Links, TEs (UUID 升序锁定)
		// 1) 先无锁定位当前 Link
		activeLink, linkErr := tx.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrderIDEQ(sourceOrder.ID),
				seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Select(seamasterbillorderlinkent.FieldID, seamasterbillorderlinkent.FieldMasterBillID, seamasterbillorderlinkent.FieldTransportExecutionID).
			Only(ctx)
		if linkErr != nil {
			if ent.IsNotFound(linkErr) {
				return biz.ErrSeaOrderSplitBlocked
			}
			return linkErr
		}

		mblIDsToLock := []uuid.UUID{activeLink.MasterBillID}
		for _, t := range input.Targets {
			if t.CandidateID != nil && *t.CandidateID != uuid.Nil {
				mblIDsToLock = append(mblIDsToLock, *t.CandidateID)
			}
		}
		mblIDsToLock = sortAndDeduplicateUUIDs(mblIDsToLock)

		lockedMBLs := make(map[uuid.UUID]*ent.SeaMasterBill, len(mblIDsToLock))
		teIDsToLock := make([]uuid.UUID, 0, len(mblIDsToLock)+len(input.Targets))
		teIDsToLock = append(teIDsToLock, activeLink.TransportExecutionID)
		for _, t := range input.Targets {
			if t.CandidateTEID != nil && *t.CandidateTEID != uuid.Nil {
				teIDsToLock = append(teIDsToLock, *t.CandidateTEID)
			}
		}
		for _, mblID := range mblIDsToLock {
			mbl, err := tx.SeaMasterBill.Query().
				Where(seamasterbillent.IDEQ(mblID), seamasterbillent.OrganizationIDEQ(organizationID)).
				ForUpdate().
				Only(ctx)
			if err != nil {
				return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
			}
			lockedMBLs[mblID] = mbl
		}
		if lockedMBLs[activeLink.MasterBillID].Status != seamasterbillent.StatusDRAFT {
			return biz.ErrSeaOrderSplitBlocked
		}

		// 锁序 4: 锁定 Link 并重验它仍为该 Order 唯一 ACTIVE 且字段未变
		lockedActiveLink, err := tx.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlinkent.IDEQ(activeLink.ID)).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return err
		}
		if lockedActiveLink.OrderID != sourceOrder.ID || lockedActiveLink.OrganizationID != organizationID ||
			lockedActiveLink.Status != seamasterbillorderlinkent.StatusACTIVE || lockedActiveLink.MasterBillID != activeLink.MasterBillID {
			return biz.ErrSeaOrderSplitBlocked
		}
		activeLinkCount, err := tx.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrderIDEQ(sourceOrder.ID),
				seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Count(ctx)
		if err != nil {
			return err
		}
		if activeLinkCount != 1 {
			return biz.ErrSeaOrderSplitBlocked
		}
		if input.ExpectedVersions.LinkVersion == 0 || lockedActiveLink.Version != input.ExpectedVersions.LinkVersion {
			return biz.ErrSeaOrderSplitVersionConflict
		}
		if lockedActiveLink.DocumentStructure != seamasterbillorderlinkent.DocumentStructureHOUSE {
			return biz.ErrSeaOrderSplitBlocked
		}

		// 锁序 5: 锁定 TE
		teIDsToLock = sortAndDeduplicateUUIDs(teIDsToLock)
		lockedTEs := make(map[uuid.UUID]*ent.SeaTransportExecution, len(teIDsToLock))
		for _, teID := range teIDsToLock {
			te, err := tx.SeaTransportExecution.Query().
				Where(seatransportexecutionent.IDEQ(teID), seatransportexecutionent.OrganizationIDEQ(organizationID)).
				ForUpdate().
				Only(ctx)
			if err != nil {
				return err
			}
			lockedTEs[teID] = te
		}

		// 锁后精确核对所有候选 MBL/TE 版本与目标输入兼容性
		for _, t := range input.Targets {
			if t.CandidateID != nil && *t.CandidateID != uuid.Nil {
				candID := *t.CandidateID
				candMBL := lockedMBLs[candID]
				if candMBL == nil || t.CandidateVersion == nil || candMBL.Version != *t.CandidateVersion ||
					t.CandidateTEID == nil || *t.CandidateTEID == uuid.Nil {
					return biz.ErrSeaOrderSplitVersionConflict
				}
				if candMBL.Status != seamasterbillent.StatusDRAFT {
					return biz.ErrSeaOrderSplitBlocked
				}
				if input.ExpectedVersions.CandidateMBLVersions == nil {
					return biz.ErrSeaOrderSplitVersionConflict
				}
				expCandVer, hasMblVer := input.ExpectedVersions.CandidateMBLVersions[candID]
				if !hasMblVer || expCandVer == 0 || candMBL.Version != expCandVer {
					return biz.ErrSeaOrderSplitVersionConflict
				}

				candTE := lockedTEs[*t.CandidateTEID]
				if candTE == nil {
					return biz.ErrSeaTransportExecutionNotFound
				}
				if !seaMasterBillShippingLineConsistent(candMBL, candTE) {
					return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_SHIPPING_LINE_INCONSISTENT",
					})
				}
				shippingLineEnabled, err := enabledShippingLineExists(ctx, tx.Client(), organizationID, candMBL.ShippingLineID, true)
				if err != nil {
					return err
				}
				if !shippingLineEnabled {
					return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_SHIPPING_LINE_UNAVAILABLE",
					})
				}
				if t.CandidateTEVersion == nil || candTE.Version != *t.CandidateTEVersion {
					return biz.ErrSeaOrderSplitVersionConflict
				}
				if input.ExpectedVersions.CandidateTEVersions == nil {
					return biz.ErrSeaOrderSplitVersionConflict
				}
				expTeVer, hasTeVer := input.ExpectedVersions.CandidateTEVersions[candTE.ID]
				if !hasTeVer || expTeVer == 0 || candTE.Version != expTeVer {
					return biz.ErrSeaOrderSplitVersionConflict
				}

				// 校验目标输入与候选权威 TE/MBL 是否一致 (允许不同于原票航程，但目标输入必须与候选一致)
				if t.MasterNo != "" {
					normInputMasterNo, err := biz.ValidateAndNormalizeSeaMasterNo(t.MasterNo)
					if err != nil {
						return err
					}
					if candMBL.NormalizedMasterNo != normInputMasterNo {
						return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
							"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
						})
					}
				}
				if t.ShippingLineID != nil && *t.ShippingLineID != uuid.Nil && candMBL.ShippingLineID != *t.ShippingLineID {
					return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
				if t.ShippingLineID != nil && *t.ShippingLineID != uuid.Nil && candTE.ShippingLineID != *t.ShippingLineID {
					return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
				if t.VesselName != "" && candTE.VesselName != t.VesselName {
					return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
				if t.VoyageNo != "" && candTE.VoyageNo != t.VoyageNo {
					return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
				if t.OriginLocationID != nil && *t.OriginLocationID != uuid.Nil && (candTE.OriginLocationID == nil || *candTE.OriginLocationID != *t.OriginLocationID) {
					return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
				if t.DischargeLocationID != nil && *t.DischargeLocationID != uuid.Nil && (candTE.DischargeLocationID == nil || *candTE.DischargeLocationID != *t.DischargeLocationID) {
					return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
				if t.TransitLocationID != nil && *t.TransitLocationID != uuid.Nil && (candTE.TransitLocationID == nil || *candTE.TransitLocationID != *t.TransitLocationID) {
					return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
				if t.ETD != "" {
					etdTime := parseOptionalTime(t.ETD)
					if etdTime == nil || candTE.Etd == nil || !etdTime.Equal(*candTE.Etd) {
						return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
							"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
						})
					}
				}
				if t.ETA != "" {
					etaTime := parseOptionalTime(t.ETA)
					if etaTime == nil || candTE.Eta == nil || !etaTime.Equal(*candTE.Eta) {
						return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
							"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
						})
					}
				}
			}
		}

		// 锁序 6-10: 货物、HBL、集装箱、分配、费用 (UUID 升序锁定)
		if input.Confirmation != nil {
			if err := validateConfirmationAttachment(ctx, tx.Client(), organizationID, input.OrderID, input.Confirmation); err != nil {
				return err
			}
		}
		cargoItems, err := tx.OrderCargoItem.Query().
			Where(ordercargoitement.OrderIDEQ(sourceOrder.ID)).
			Order(ordercargoitement.ByID()).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}
		if input.ExpectedVersions.CargoItemVersions == nil || len(input.ExpectedVersions.CargoItemVersions) != len(cargoItems) {
			return biz.ErrSeaOrderSplitVersionConflict
		}
		lockedCargoItemMap := make(map[uuid.UUID]*ent.OrderCargoItem, len(cargoItems))
		for _, ci := range cargoItems {
			expV, ok := input.ExpectedVersions.CargoItemVersions[ci.ID]
			if !ok || expV == 0 || ci.Version != expV {
				return biz.ErrSeaOrderSplitVersionConflict
			}
			lockedCargoItemMap[ci.ID] = ci
		}

		hbls, err := tx.SeaHouseBill.Query().
			Where(seahousebillent.OrderIDEQ(sourceOrder.ID)).
			Order(seahousebillent.ByID()).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}
		lockedHBLMap := make(map[uuid.UUID]*ent.SeaHouseBill, len(hbls))
		for _, h := range hbls {
			lockedHBLMap[h.ID] = h
		}

		var curHBL *ent.SeaHouseBill
		for _, h := range hbls {
			if h.Status != seahousebillent.StatusVOIDED {
				if h.Status != seahousebillent.StatusDRAFT {
					return biz.ErrSeaOrderSplitBlocked
				}
				curHBL = h
				break
			}
		}
		if lockedActiveLink.DocumentStructure == seamasterbillorderlinkent.DocumentStructureHOUSE && curHBL == nil {
			return biz.ErrSeaHouseBillNotFound
		}
		// HOUSE 拆票必须携带唯一当前 HBL 的非零期望版本；缺失或为 0 属于参数错误
		if input.ExpectedVersions.CurrentHBLVersion == nil || *input.ExpectedVersions.CurrentHBLVersion == 0 {
			return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
				"reason": "CURRENT_HBL_VERSION_REQUIRED",
			})
		}
		if curHBL == nil || curHBL.Version != *input.ExpectedVersions.CurrentHBLVersion {
			return biz.ErrSeaOrderSplitVersionConflict
		}

		containers, err := tx.OrderContainer.Query().
			Where(ordercontainerent.OrderIDEQ(sourceOrder.ID)).
			Order(ordercontainerent.ByID()).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}
		lockedContainerMap := make(map[uuid.UUID]*ent.OrderContainer, len(containers))
		for _, c := range containers {
			lockedContainerMap[c.ID] = c
		}
		// 独占箱将被移动并递增版本，期望版本 Map 必须完整覆盖且非零；
		// 缺 Map、缺 key 或版本为 0 属于参数错误，版本不一致返回 409。
		if len(containers) > 0 {
			if input.ExpectedVersions.ContainerVersions == nil {
				return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
					"reason": "CONTAINER_VERSION_REQUIRED",
				})
			}
			if len(input.ExpectedVersions.ContainerVersions) != len(containers) {
				return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
					"reason": "CONTAINER_VERSION_REQUIRED",
				})
			}
			for _, c := range containers {
				expV, ok := input.ExpectedVersions.ContainerVersions[c.ID]
				if !ok || expV == 0 {
					return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
						"reason":       "CONTAINER_VERSION_REQUIRED",
						"container_id": c.ID.String(),
					})
				}
				if c.Version != expV {
					return biz.ErrSeaOrderSplitVersionConflict
				}
			}
		}

		// 共享箱固定锁序：先无锁定位本订单分配所属共享箱 ID，按序锁定 SharedContainer，
		// 再锁定 Allocation，并在锁内重验分配集合未漂移；与共享箱工作台/删除路径的
		// SharedContainer → Allocation 顺序保持一致，避免跨操作死锁。
		preReadAllocIDs, err := tx.SeaSharedContainerAllocation.Query().
			Where(
				seasharedcontainerallocationent.OrganizationIDEQ(organizationID),
				seasharedcontainerallocationent.OrderIDEQ(sourceOrder.ID),
			).
			Order(seasharedcontainerallocationent.ByID()).
			IDs(ctx)
		if err != nil {
			return err
		}
		preReadAllocIDSet := make(map[uuid.UUID]struct{}, len(preReadAllocIDs))
		sharedContainerIDs := make([]uuid.UUID, 0, len(preReadAllocIDs))
		for _, allocID := range preReadAllocIDs {
			preReadAllocIDSet[allocID] = struct{}{}
		}
		if len(preReadAllocIDs) > 0 {
			scIDRows, err := tx.SeaSharedContainerAllocation.Query().
				Where(
					seasharedcontainerallocationent.OrganizationIDEQ(organizationID),
					seasharedcontainerallocationent.IDIn(preReadAllocIDs...),
				).
				Select(seasharedcontainerallocationent.FieldSharedContainerID).
				All(ctx)
			if err != nil {
				return err
			}
			for _, row := range scIDRows {
				sharedContainerIDs = append(sharedContainerIDs, row.SharedContainerID)
			}
		}
		sharedContainerIDs = sortAndDeduplicateUUIDs(sharedContainerIDs)
		lockedSharedContainers := make(map[uuid.UUID]*ent.SeaSharedContainer, len(sharedContainerIDs))
		for _, scID := range sharedContainerIDs {
			sc, err := tx.SeaSharedContainer.Query().
				Where(
					seasharedcontainerent.IDEQ(scID),
					seasharedcontainerent.OrganizationIDEQ(organizationID),
				).
				ForUpdate().
				Only(ctx)
			if err != nil {
				return err
			}
			lockedSharedContainers[scID] = sc
		}
		sharedAllocs, err := tx.SeaSharedContainerAllocation.Query().
			Where(
				seasharedcontainerallocationent.OrganizationIDEQ(organizationID),
				seasharedcontainerallocationent.OrderIDEQ(sourceOrder.ID),
			).
			Order(seasharedcontainerallocationent.ByID()).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}
		if len(sharedAllocs) != len(preReadAllocIDs) {
			return biz.ErrSeaOrderSplitVersionConflict
		}
		lockedSharedAllocMap := make(map[uuid.UUID]*ent.SeaSharedContainerAllocation, len(sharedAllocs))
		for _, sa := range sharedAllocs {
			if _, seen := preReadAllocIDSet[sa.ID]; !seen {
				return biz.ErrSeaOrderSplitVersionConflict
			}
			lockedSharedAllocMap[sa.ID] = sa
		}
		// 拆票修改共享箱 Allocation 前，所有受影响共享箱必须携带完整且非零的期望版本；
		// 缺 Map、缺 key 或版本为 0 属于参数错误，版本不一致返回 409。
		if len(sharedContainerIDs) > 0 {
			if input.ExpectedVersions.SharedContainerVersions == nil {
				return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
					"reason": "SHARED_CONTAINER_VERSION_REQUIRED",
				})
			}
			for _, scID := range sharedContainerIDs {
				expV, ok := input.ExpectedVersions.SharedContainerVersions[scID]
				if !ok || expV == 0 {
					return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
						"reason":              "SHARED_CONTAINER_VERSION_REQUIRED",
						"shared_container_id": scID.String(),
					})
				}
				if sc := lockedSharedContainers[scID]; sc.Version != expV {
					return biz.ErrSeaOrderSplitVersionConflict
				}
			}
		}

		fees, err := tx.OrderFee.Query().
			Where(orderfeeent.OrderIDEQ(sourceOrder.ID)).
			Order(orderfeeent.ByID()).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}
		lockedFeesMap := make(map[uuid.UUID]*ent.OrderFee, len(fees))
		unbilledFeeCount := 0
		for _, f := range fees {
			if f.Status != orderfeeent.StatusUNBILLED && f.Status != orderfeeent.StatusCANCELLED {
				return biz.ErrSeaOrderSplitBlocked
			}
			if f.Status == orderfeeent.StatusUNBILLED {
				unbilledFeeCount++
				lockedFeesMap[f.ID] = f
			}
		}
		if input.ExpectedVersions.FeeVersions == nil || len(input.ExpectedVersions.FeeVersions) != unbilledFeeCount {
			return biz.ErrSeaOrderSplitVersionConflict
		}
		for _, f := range fees {
			if f.Status == orderfeeent.StatusUNBILLED {
				expV, ok := input.ExpectedVersions.FeeVersions[f.ID]
				if !ok || expV == 0 || f.Version != expV {
					return biz.ErrSeaOrderSplitVersionConflict
				}
			}
		}

		// 锁序 11: 附件资产与引用 (UUID 升序锁定) 并精确核对指纹
		orderAttachments, err := tx.OrderAttachment.Query().
			Where(orderattachmentent.OrderIDEQ(sourceOrder.ID)).
			Order(orderattachmentent.ByID()).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}
		attItems := make([]*biz.SeaOrderSplitAttachmentItem, 0, len(orderAttachments))
		for _, oa := range orderAttachments {
			attItems = append(attItems, &biz.SeaOrderSplitAttachmentItem{
				ID:      oa.ID,
				AssetID: oa.AssetID,
				DocType: oa.DocType,
			})
		}
		currentAttFp := biz.ComputeAttachmentFingerprint(attItems)
		if input.ExpectedVersions.AttachmentReferenceFingerprint == "" || input.ExpectedVersions.AttachmentReferenceFingerprint != currentAttFp {
			return biz.ErrSeaOrderSplitVersionConflict
		}

		// 锁序 12: 下游门禁重验 (账单、核销、提成)
		hasActiveBillLine, err := tx.FinanceBillLine.Query().
			Where(financebilllineent.OrderIDEQ(sourceOrder.ID), financebilllineent.ActiveEQ(true)).
			Exist(ctx)
		if err != nil {
			return err
		}
		if hasActiveBillLine {
			return biz.ErrSeaOrderSplitBlocked
		}

		hasCommission, err := tx.FinanceCommissionLine.Query().
			Where(financecommissionlineent.OrderIDEQ(sourceOrder.ID)).
			Exist(ctx)
		if err != nil {
			return err
		}
		if hasCommission {
			return biz.ErrSeaOrderSplitBlocked
		}

		// -------------------------------------------------------------------
		// 锁后重算全部分配、守恒、结构、不跨箱与费用完整性
		// -------------------------------------------------------------------

		// 1. HBL 新票分单号基础校验（HOUSE 结构必填、号合法、签发来源必填）。
		// 批次内排重（含作废行、请求内同目标批次去重）在「处理 HBL」阶段执行：
		// 此时 resultFinalMblMap 已完备，才能拿到各结果票的目标主单；此处的
		// 组织级排重已随全局唯一索引撤销一并移除（组织+签发主体全局唯一不再
		// 是契约，跨批次重号合法）。
		for _, res := range input.Results {
			if res.ResultRole == biz.ResultRoleCreated {
				if lockedActiveLink.DocumentStructure == seamasterbillorderlinkent.DocumentStructureHOUSE {
					if res.HouseBill == nil || strings.TrimSpace(res.HouseBill.HouseNo) == "" {
						return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
							"reason":            "HOUSE_BILL_REQUIRED",
							"client_result_key": res.ClientResultKey,
						})
					}
					if _, err := biz.NormalizeSeaHouseNo(res.HouseBill.HouseNo); err != nil {
						return err
					}
					if res.HouseBill.IssuerSource == "" {
						return biz.ErrSeaOrderSplitInvalidArgument
					}
				}
			}
		}

		// 2. 真实独占箱分配校验：每只箱恰好分配到 1 个结果
		containerToResultKey := make(map[uuid.UUID]string)
		for _, res := range input.Results {
			for _, cID := range res.ContainerIDs {
				if _, exists := lockedContainerMap[cID]; !exists {
					return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
						"reason":            "CONTAINER_NOT_FOUND",
						"container_id":      cID.String(),
						"client_result_key": res.ClientResultKey,
					})
				}
				if prevKey, seen := containerToResultKey[cID]; seen {
					return biz.MetadataError(biz.ErrSeaOrderSplitEntityCrossesResults, map[string]string{
						"reason":        "CONTAINER_CROSSES_RESULTS",
						"container_id":  cID.String(),
						"first_result":  prevKey,
						"second_result": res.ClientResultKey,
					})
				}
				containerToResultKey[cID] = res.ClientResultKey
			}
		}
		if len(containerToResultKey) != len(containers) {
			return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
				"reason":  "CONTAINER_ALLOCATION_INCOMPLETE",
				"message": "所有集装箱必须分配且只能属于一个结果",
			})
		}

		// 3. 货物数量与重量体积严格守恒重算
		type resultCargoStat struct {
			pkg    int
			weight decimal.Decimal
			vol    decimal.Decimal
		}
		resCargoStats := make(map[string]*resultCargoStat)
		for _, res := range input.Results {
			resCargoStats[res.ClientResultKey] = &resultCargoStat{
				weight: decimal.Zero,
				vol:    decimal.Zero,
			}
		}

		allocPkgByItem := make(map[uuid.UUID]int)
		allocWtByItem := make(map[uuid.UUID]decimal.Decimal)
		allocVolByItem := make(map[uuid.UUID]decimal.Decimal)
		for _, ci := range cargoItems {
			allocWtByItem[ci.ID] = decimal.Zero
			allocVolByItem[ci.ID] = decimal.Zero
		}
		resCargoPkgByItem := make(map[string]map[uuid.UUID]int)
		resCargoWtByItem := make(map[string]map[uuid.UUID]decimal.Decimal)
		resCargoVolByItem := make(map[string]map[uuid.UUID]decimal.Decimal)

		for _, res := range input.Results {
			rs := resCargoStats[res.ClientResultKey]
			resCargoPkgByItem[res.ClientResultKey] = make(map[uuid.UUID]int)
			resCargoWtByItem[res.ClientResultKey] = make(map[uuid.UUID]decimal.Decimal)
			resCargoVolByItem[res.ClientResultKey] = make(map[uuid.UUID]decimal.Decimal)
			for _, ca := range res.CargoAllocations {
				if _, ok := allocWtByItem[ca.CargoItemID]; !ok {
					return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
						"reason":        "CARGO_ITEM_NOT_FOUND",
						"cargo_item_id": ca.CargoItemID.String(),
					})
				}
				if ca.PackageCount < 0 || ca.GrossWeightKg.IsNegative() || ca.VolumeCbm.IsNegative() {
					return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
						"reason":        "CARGO_ITEM_INVALID_QUANTITY",
						"cargo_item_id": ca.CargoItemID.String(),
					})
				}
				allocPkgByItem[ca.CargoItemID] += int(ca.PackageCount)
				allocWtByItem[ca.CargoItemID] = allocWtByItem[ca.CargoItemID].Add(ca.GrossWeightKg)
				allocVolByItem[ca.CargoItemID] = allocVolByItem[ca.CargoItemID].Add(ca.VolumeCbm)

				rs.pkg += int(ca.PackageCount)
				rs.weight = rs.weight.Add(ca.GrossWeightKg)
				rs.vol = rs.vol.Add(ca.VolumeCbm)

				resCargoPkgByItem[res.ClientResultKey][ca.CargoItemID] += int(ca.PackageCount)
				resCargoWtByItem[res.ClientResultKey][ca.CargoItemID] = resCargoWtByItem[res.ClientResultKey][ca.CargoItemID].Add(ca.GrossWeightKg)
				resCargoVolByItem[res.ClientResultKey][ca.CargoItemID] = resCargoVolByItem[res.ClientResultKey][ca.CargoItemID].Add(ca.VolumeCbm)
			}
		}

		for _, ci := range cargoItems {
			ciAllocPkg := allocPkgByItem[ci.ID]
			ciAllocWeight := allocWtByItem[ci.ID]
			ciAllocVol := allocVolByItem[ci.ID]

			ciWeightDec := decimal.NewFromFloat(ci.GrossWeightKg)
			ciVolDec := decimal.NewFromFloat(ci.VolumeCbm)

			if ciAllocPkg != ci.PackageCount || !ciAllocWeight.Equal(ciWeightDec) || !ciAllocVol.Equal(ciVolDec) {
				return biz.MetadataError(biz.ErrSeaOrderSplitConservationFailed, map[string]string{
					"reason":        "QUANTITY_CONSERVATION_FAILED",
					"cargo_item_id": ci.ID.String(),
				})
			}
		}

		for resKey, rs := range resCargoStats {
			if rs.pkg <= 0 && rs.weight.IsZero() && rs.vol.IsZero() {
				return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
					"reason":            "EMPTY_RESULT_NOT_ALLOWED",
					"client_result_key": resKey,
				})
			}
		}

		// 3.1 共享箱分配守恒校验 (如果存在)
		if len(sharedAllocs) > 0 {
			allocPkgByShared := make(map[uuid.UUID]int)
			allocWtByShared := make(map[uuid.UUID]decimal.Decimal)
			allocVolByShared := make(map[uuid.UUID]decimal.Decimal)
			for _, sa := range sharedAllocs {
				allocWtByShared[sa.ID] = decimal.Zero
				allocVolByShared[sa.ID] = decimal.Zero
			}
			resSharedPkgByItem := make(map[string]map[uuid.UUID]int)
			resSharedWtByItem := make(map[string]map[uuid.UUID]decimal.Decimal)
			resSharedVolByItem := make(map[string]map[uuid.UUID]decimal.Decimal)
			for _, res := range input.Results {
				resSharedPkgByItem[res.ClientResultKey] = make(map[uuid.UUID]int)
				resSharedWtByItem[res.ClientResultKey] = make(map[uuid.UUID]decimal.Decimal)
				resSharedVolByItem[res.ClientResultKey] = make(map[uuid.UUID]decimal.Decimal)
				for _, sca := range res.SharedContainerAllocations {
					origAlloc, exists := lockedSharedAllocMap[sca.AllocationID]
					if !exists {
						return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
							"reason":        "SHARED_ALLOCATION_NOT_FOUND",
							"allocation_id": sca.AllocationID.String(),
						})
					}
					if sca.PackageCount < 0 || sca.GrossWeightKg.IsNegative() || sca.VolumeCbm.IsNegative() {
						return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
							"reason":        "SHARED_ALLOCATION_INVALID_QUANTITY",
							"allocation_id": sca.AllocationID.String(),
						})
					}
					allocPkgByShared[sca.AllocationID] += int(sca.PackageCount)
					allocWtByShared[sca.AllocationID] = allocWtByShared[sca.AllocationID].Add(sca.GrossWeightKg)
					allocVolByShared[sca.AllocationID] = allocVolByShared[sca.AllocationID].Add(sca.VolumeCbm)

					resSharedPkgByItem[res.ClientResultKey][origAlloc.CargoItemID] += int(sca.PackageCount)
					resSharedWtByItem[res.ClientResultKey][origAlloc.CargoItemID] = resSharedWtByItem[res.ClientResultKey][origAlloc.CargoItemID].Add(sca.GrossWeightKg)
					resSharedVolByItem[res.ClientResultKey][origAlloc.CargoItemID] = resSharedVolByItem[res.ClientResultKey][origAlloc.CargoItemID].Add(sca.VolumeCbm)
				}
			}
			for _, sa := range sharedAllocs {
				sPkg := allocPkgByShared[sa.ID]
				sWt := allocWtByShared[sa.ID]
				sVol := allocVolByShared[sa.ID]
				saWt, _ := decimal.NewFromString(sa.GrossWeightKg)
				saVol, _ := decimal.NewFromString(sa.VolumeCbm)
				if sPkg != sa.PackageCount || !sWt.Equal(saWt) || !sVol.Equal(saVol) {
					return biz.MetadataError(biz.ErrSeaOrderSplitConservationFailed, map[string]string{
						"reason":        "SHARED_ALLOCATION_CONSERVATION_FAILED",
						"allocation_id": sa.ID.String(),
					})
				}
			}
			// 3.2 交叉守恒：每个结果票内，共享箱分配合计不得超过该结果分到的同一来源货物件重尺。
			for resKey, itemPkg := range resSharedPkgByItem {
				for ciID, sharedPkg := range itemPkg {
					sharedWt := resSharedWtByItem[resKey][ciID]
					sharedVol := resSharedVolByItem[resKey][ciID]
					cargoPkg := resCargoPkgByItem[resKey][ciID]
					cargoWt := resCargoWtByItem[resKey][ciID]
					cargoVol := resCargoVolByItem[resKey][ciID]
					if sharedPkg > cargoPkg || sharedWt.GreaterThan(cargoWt) || sharedVol.GreaterThan(cargoVol) {
						return biz.MetadataError(biz.ErrSeaOrderSplitConservationFailed, map[string]string{
							"reason":            "SHARED_ALLOCATION_EXCEEDS_RESULT_CARGO",
							"client_result_key": resKey,
							"cargo_item_id":     ciID.String(),
						})
					}
				}
			}
		}

		// 4. 未建账费用分配完整性与唯一性
		assignedFees := make(map[uuid.UUID]string)
		for _, res := range input.Results {
			for _, fID := range res.DraftFeeIDs {
				if _, exists := lockedFeesMap[fID]; !exists {
					return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
						"reason": "DRAFT_FEE_NOT_FOUND",
						"fee_id": fID.String(),
					})
				}
				if prevKey, seen := assignedFees[fID]; seen {
					return biz.MetadataError(biz.ErrSeaOrderSplitEntityCrossesResults, map[string]string{
						"reason":        "FEE_CROSSES_RESULTS",
						"fee_id":        fID.String(),
						"first_result":  prevKey,
						"second_result": res.ClientResultKey,
					})
				}
				assignedFees[fID] = res.ClientResultKey
			}
		}
		if len(assignedFees) != unbilledFeeCount {
			return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
				"reason":  "UNBILLED_FEE_ALLOCATION_INCOMPLETE",
				"message": "所有未取消费用必须分配且只能属于一个结果",
			})
		}

		// -------------------------------------------------------------------
		// 锁序 13: 业务写入与快照
		// -------------------------------------------------------------------
		splitEventID := uuid.Must(uuid.NewV7())

		// 查询现有箱计划，用于完整 before_snapshot
		existingBeforePlans, err := tx.OrderContainerRequest.Query().
			Where(ordercontainerrequestent.OrderIDEQ(sourceOrder.ID)).
			All(ctx)
		if err != nil {
			return err
		}
		beforePlanData := make([]map[string]interface{}, 0, len(existingBeforePlans))
		for _, bp := range existingBeforePlans {
			beforePlanData = append(beforePlanData, map[string]interface{}{
				"container_spec_id": bp.ContainerSpecID,
				"quantity":          bp.Quantity,
			})
		}

		beforeHblData := make([]map[string]interface{}, 0, len(hbls))
		for _, h := range hbls {
			beforeHblData = append(beforeHblData, map[string]interface{}{
				"id":       h.ID,
				"house_no": h.HouseNo,
				"status":   h.Status,
				"version":  h.Version,
			})
		}

		beforeCargoData := make([]map[string]interface{}, 0, len(cargoItems))
		for _, ci := range cargoItems {
			beforeCargoData = append(beforeCargoData, map[string]interface{}{
				"id":              ci.ID,
				"cargo_name":      ci.CargoName,
				"package_count":   ci.PackageCount,
				"gross_weight_kg": ci.GrossWeightKg,
				"volume_cbm":      ci.VolumeCbm,
				"version":         ci.Version,
			})
		}

		beforeContainerData := make([]map[string]interface{}, 0, len(containers))
		for _, c := range containers {
			beforeContainerData = append(beforeContainerData, map[string]interface{}{
				"id":                c.ID,
				"container_no":      c.ContainerNo,
				"container_spec_id": c.ContainerSpecID,
				"version":           c.Version,
			})
		}

		beforeSharedAllocData := make([]map[string]interface{}, 0, len(sharedAllocs))
		for _, sa := range sharedAllocs {
			beforeSharedAllocData = append(beforeSharedAllocData, map[string]interface{}{
				"id":                  sa.ID,
				"shared_container_id": sa.SharedContainerID,
				"cargo_item_id":       sa.CargoItemID,
				"package_count":       sa.PackageCount,
				"gross_weight_kg":     sa.GrossWeightKg,
				"volume_cbm":          sa.VolumeCbm,
			})
		}

		beforeFeeData := make([]map[string]interface{}, 0, len(fees))
		for _, f := range fees {
			beforeFeeData = append(beforeFeeData, map[string]interface{}{
				"id":           f.ID,
				"fee_code":     f.FeeCode,
				"total_amount": f.TotalAmount,
				"currency":     f.Currency,
				"status":       f.Status,
				"version":      f.Version,
			})
		}

		beforeAttData := make([]map[string]interface{}, 0, len(orderAttachments))
		for _, oa := range orderAttachments {
			beforeAttData = append(beforeAttData, map[string]interface{}{
				"id":       oa.ID,
				"asset_id": oa.AssetID,
				"doc_type": oa.DocType,
			})
		}

		beforeSnapshotMap := map[string]interface{}{
			"schema_version": 1,
			"order": map[string]interface{}{
				"id":                    sourceOrder.ID,
				"order_no":              sourceOrder.OrderNo,
				"version":               sourceOrder.Version,
				"flow_status":           sourceOrder.FlowStatus,
				"customer_id":           sourceOrder.CustomerID,
				"customer_reference_no": sourceOrder.CustomerReferenceNo,
				"internal_reference_no": sourceOrder.InternalReferenceNo,
			},
			"active_link": map[string]interface{}{
				"id":                       lockedActiveLink.ID,
				"version":                  lockedActiveLink.Version,
				"cargo_allocation_version": 0,
				"status":                   lockedActiveLink.Status,
				"master_bill_id":           lockedActiveLink.MasterBillID,
			},
			"master_bill": map[string]interface{}{
				"id":        lockedMBLs[lockedActiveLink.MasterBillID].ID,
				"master_no": lockedMBLs[lockedActiveLink.MasterBillID].MasterNo,
				"version":   lockedMBLs[lockedActiveLink.MasterBillID].Version,
				"status":    lockedMBLs[lockedActiveLink.MasterBillID].Status,
			},
			"house_bills":                  beforeHblData,
			"cargo_items":                  beforeCargoData,
			"containers":                   beforeContainerData,
			"shared_container_allocations": beforeSharedAllocData,
			"draft_fees":                   beforeFeeData,
			"attachments":                  beforeAttData,
			"container_plans":              beforePlanData,
		}
		beforeSnapshotBytes, err := json.Marshal(beforeSnapshotMap)
		if err != nil {
			return err
		}

		conservationSnapshotMap := map[string]interface{}{
			"schema_version":                     1,
			"conservation_passed":                true,
			"cargo_items_count":                  len(cargoItems),
			"house_bills_count":                  len(hbls),
			"containers_count":                   len(containers),
			"shared_container_allocations_count": len(sharedAllocs),
			"unbilled_fees_count":                unbilledFeeCount,
		}
		conservationSnapshotBytes, err := json.Marshal(conservationSnapshotMap)
		if err != nil {
			return err
		}

		splitEventBuilder := tx.SeaOrderSplitEvent.Create().
			SetID(splitEventID).
			SetOrganizationID(organizationID).
			SetSourceOrderID(sourceOrder.ID).
			SetSourceOrderNo(sourceOrder.OrderNo).
			SetIdempotencyKey(input.IdempotencyKey).
			SetRequestFingerprint(input.RequestFingerprint).
			SetSourceOrderVersion(sourceOrder.Version).
			SetSourceLinkID(lockedActiveLink.ID).
			SetSourceLinkVersion(lockedActiveLink.Version).
			SetSourceAllocationVersion(0).
			SetBeforeSnapshot(beforeSnapshotBytes).
			SetConservationSnapshot(conservationSnapshotBytes).
			SetCreatedBy(actorID)
		if input.Note != nil {
			splitEventBuilder.SetNote(*input.Note)
		}
		savedSplitEvent, err := splitEventBuilder.Save(ctx)
		if err != nil {
			return mapEntConstraint(err, "sea_order_split_event_idempotency_key", biz.ErrSeaOrderSplitIdempotencyConflict)
		}

		targetInputMap := make(map[string]*biz.SeaOrderSplitTargetInput)
		for _, t := range input.Targets {
			targetInputMap[t.ClientTargetKey] = t
		}

		type targetEntityInfo struct {
			MBL *ent.SeaMasterBill
			TE  *ent.SeaTransportExecution
		}
		targetEntities := make(map[string]*targetEntityInfo)

		for _, target := range input.Targets {
			if target.TargetType == biz.SplitTargetTypeCandidate {
				if target.CandidateID != nil && *target.CandidateID != uuid.Nil {
					candMBL := lockedMBLs[*target.CandidateID]
					candTE := lockedTEs[*target.CandidateTEID]
					targetEntities[target.ClientTargetKey] = &targetEntityInfo{
						MBL: candMBL,
						TE:  candTE,
					}
				}
			} else if target.TargetType == biz.SplitTargetTypeNew {
				if _, exists := targetEntities[target.ClientTargetKey]; !exists {
					newMbl, newTE, createErr := createNewMasterBillInTx(ctx, tx, organizationID, target)
					if createErr != nil {
						return createErr
					}
					lockedMBLs[newMbl.ID] = newMbl
					lockedTEs[newTE.ID] = newTE
					targetEntities[target.ClientTargetKey] = &targetEntityInfo{
						MBL: newMbl,
						TE:  newTE,
					}
				}
			}
		}

		resultOrderMap := make(map[string]uuid.UUID)
		resultOrderNoMap := make(map[string]string)
		resultFinalMblMap := make(map[string]uuid.UUID)
		resultLinkMap := make(map[string]uuid.UUID)
		reassignmentEventIDs := make([]uuid.UUID, 0)
		createdOrdersMap := make(map[string]*ent.Order)
		originalResultKey := ""

		for _, res := range input.Results {
			target := targetInputMap[res.ClientTargetKey]
			if target == nil {
				return biz.ErrSeaOrderSplitInvalidArgument
			}
			now := time.Now().UTC()

			if res.ResultRole == biz.ResultRoleOriginal {
				originalResultKey = res.ClientResultKey
				resultOrderMap[res.ClientResultKey] = sourceOrder.ID
				resultOrderNoMap[res.ClientResultKey] = sourceOrder.OrderNo

				finalMblID := lockedActiveLink.MasterBillID
				if target.TargetType != biz.SplitTargetTypeCurrent {
					tEntity := targetEntities[target.ClientTargetKey]
					if tEntity == nil {
						return biz.ErrSeaOrderSplitInvalidArgument
					}
					reassignMblID := tEntity.MBL.ID
					reassignTEID := tEntity.TE.ID

					// 结束原票当前活动 Link
					if _, err := tx.SeaMasterBillOrderLink.UpdateOneID(lockedActiveLink.ID).
						SetStatus(seamasterbillorderlinkent.StatusENDED).
						SetEndedAt(now).
						SetVersion(lockedActiveLink.Version + 1).
						Save(ctx); err != nil {
						return err
					}

					// 为原票创建目标 MBL 的最终 Link
					finalLinkBuilder := tx.SeaMasterBillOrderLink.Create().
						SetID(uuid.Must(uuid.NewV7())).
						SetOrganizationID(organizationID).
						SetMasterBillID(reassignMblID).
						SetTransportExecutionID(reassignTEID).
						SetOrderID(sourceOrder.ID).
						SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
						SetStatus(seamasterbillorderlinkent.StatusACTIVE).
						SetStartedAt(now).
						SetVersion(1)
					finalLink, err := finalLinkBuilder.Save(ctx)
					if err != nil {
						return err
					}
					resultLinkMap[res.ClientResultKey] = finalLink.ID

					origBeforeReassignMap := map[string]interface{}{
						"schema_version": 1,
						"link_id":        lockedActiveLink.ID,
						"link_version":   lockedActiveLink.Version,
						"master_bill_id": lockedActiveLink.MasterBillID,
						"master_no":      lockedMBLs[lockedActiveLink.MasterBillID].MasterNo,
					}
					origAfterReassignMap := map[string]interface{}{
						"schema_version": 1,
						"link_id":        finalLink.ID,
						"link_version":   finalLink.Version,
						"master_bill_id": reassignMblID,
						"master_no":      lockedMBLs[reassignMblID].MasterNo,
					}
					origBeforeReassignBytes, err := json.Marshal(origBeforeReassignMap)
					if err != nil {
						return err
					}
					origAfterReassignBytes, err := json.Marshal(origAfterReassignMap)
					if err != nil {
						return err
					}

					reassignEvtID := uuid.Must(uuid.NewV7())
					reassignEvt, err := tx.SeaOrderReassignmentEvent.Create().
						SetID(reassignEvtID).
						SetOrganizationID(organizationID).
						SetOrderID(sourceOrder.ID).
						SetOrderNo(sourceOrder.OrderNo).
						SetSplitEventID(splitEventID).
						SetIdempotencyKey(input.IdempotencyKey + ":reassign:" + res.ClientResultKey).
						SetRequestFingerprint(input.RequestFingerprint).
						SetPreviousMasterBillID(lockedActiveLink.MasterBillID).
						SetTargetMasterBillID(reassignMblID).
						SetPreviousTransportExecutionID(lockedActiveLink.TransportExecutionID).
						SetTargetTransportExecutionID(reassignTEID).
						SetPreviousLinkID(lockedActiveLink.ID).
						SetTargetLinkID(finalLink.ID).
						SetPreviousLinkVersion(lockedActiveLink.Version).
						SetTargetLinkVersion(1).
						SetReason("部分拆票原票改配").
						SetResponsibilityType(seaorderreassignmenteventent.ResponsibilityTypeOWN_COMPANY).
						SetBeforeSnapshot(origBeforeReassignBytes).
						SetAfterSnapshot(origAfterReassignBytes).
						SetCreatedBy(actorID).
						SetConfirmedByParty(input.Confirmation.ConfirmedByParty).
						SetConfirmedAt(input.Confirmation.ConfirmedAt).
						SetConfirmationNote(input.Confirmation.ConfirmationNote).
						SetNillableConfirmationAttachmentID(input.Confirmation.ConfirmationAttachmentID).
						Save(ctx)
					if err != nil {
						return err
					}
					reassignmentEventIDs = append(reassignmentEventIDs, reassignEvt.ID)
					finalMblID = reassignMblID

					// 更新原订单权威航程投影
					targetTE := lockedTEs[reassignTEID]
					orderUpdate := tx.Order.UpdateOneID(sourceOrder.ID).
						SetVesselVoyage(biz.CombineVesselVoyage(targetTE.VesselName, targetTE.VoyageNo))
					orderUpdate.SetShippingLineID(targetTE.ShippingLineID)
					if targetTE.OriginLocationID != nil {
						orderUpdate.SetOriginLocationID(*targetTE.OriginLocationID)
					} else {
						orderUpdate.ClearOriginLocationID()
					}
					if targetTE.DischargeLocationID != nil {
						orderUpdate.SetDischargeLocationID(*targetTE.DischargeLocationID)
					} else {
						orderUpdate.ClearDischargeLocationID()
					}
					if targetTE.TransitLocationID != nil {
						orderUpdate.SetTransitLocationID(*targetTE.TransitLocationID)
					} else {
						orderUpdate.ClearTransitLocationID()
					}
					if targetTE.Etd != nil {
						orderUpdate.SetEtd(targetTE.Etd.Format(time.RFC3339))
					} else {
						orderUpdate.ClearEtd()
					}
					if targetTE.Eta != nil {
						orderUpdate.SetEta(targetTE.Eta.Format(time.RFC3339))
					} else {
						orderUpdate.ClearEta()
					}
					if _, err := orderUpdate.Save(ctx); err != nil {
						return err
					}
				} else {
					// 留在当前母单
					resultLinkMap[res.ClientResultKey] = lockedActiveLink.ID
				}
				resultFinalMblMap[res.ClientResultKey] = finalMblID

			} else {
				// 新建子操作票
				orderNo := createdOrderNumbers[res.ClientResultKey]
				// 拆票幂等由拆票事件键承担；子单幂等键列为 NOT NULL，
				// 由服务端生成随机键填充（每个子单独立生成，避免同批
				// 子单撞 (organization_id, idempotency_key) 唯一索引）。
				createOrder := tx.Order.Create().
					SetOrganizationID(organizationID).
					SetOrderNo(orderNo).
					SetIdempotencyKey(uuid.NewString()).
					SetBusinessType(sourceOrder.BusinessType).
					SetTradeDirection(sourceOrder.TradeDirection).
					SetNillableTradeTerm(sourceOrder.TradeTerm).
					SetPaymentTerm(sourceOrder.PaymentTerm).
					SetNillableShipmentType(sourceOrder.ShipmentType).
					SetNillableContainerOwnership(sourceOrder.ContainerOwnership).
					SetNillableShipmentMode(sourceOrder.ShipmentMode).
					SetCustomerID(sourceOrder.CustomerID).
					SetCustomerReferenceNo(sourceOrder.CustomerReferenceNo).
					SetShipperShortName(sourceOrder.ShipperShortName).
					SetConsigneeShortName(sourceOrder.ConsigneeShortName).
					SetNillableBookingAgentID(sourceOrder.BookingAgentID).
					SetNillableForeignAgentID(sourceOrder.ForeignAgentID).
					SetNillableShippingAgentID(sourceOrder.ShippingAgentID).
					SetNillableDestinationLocationID(sourceOrder.DestinationLocationID).
					SetOrderDate(sourceOrder.OrderDate).
					SetNotes(sourceOrder.Notes).
					SetBookingNotes(sourceOrder.BookingNotes).
					SetOperationNotes(sourceOrder.OperationNotes).
					SetFlowStatus(sourceOrder.FlowStatus).
					SetTerminationStatus(orderent.TerminationStatusACTIVE).
					SetClosureStatus(orderent.ClosureStatusOPEN).
					SetVersion(1)

				if target.TargetType != biz.SplitTargetTypeCurrent {
					tEntity := targetEntities[target.ClientTargetKey]
					if tEntity == nil {
						return biz.ErrSeaOrderSplitInvalidArgument
					}
					targetTE := tEntity.TE
					createOrder.SetVesselVoyage(biz.CombineVesselVoyage(targetTE.VesselName, targetTE.VoyageNo))
					createOrder.SetShippingLineID(targetTE.ShippingLineID)
					createOrder.SetNillableOriginLocationID(targetTE.OriginLocationID)
					createOrder.SetNillableDischargeLocationID(targetTE.DischargeLocationID)
					createOrder.SetNillableTransitLocationID(targetTE.TransitLocationID)
					if targetTE.Etd != nil {
						createOrder.SetEtd(targetTE.Etd.Format(time.RFC3339))
					}
					if targetTE.Eta != nil {
						createOrder.SetEta(targetTE.Eta.Format(time.RFC3339))
					}
				} else {
					createOrder.SetVesselVoyage(sourceOrder.VesselVoyage)
					createOrder.SetNillableShippingLineID(sourceOrder.ShippingLineID)
					createOrder.SetNillableOriginLocationID(sourceOrder.OriginLocationID)
					createOrder.SetNillableDischargeLocationID(sourceOrder.DischargeLocationID)
					createOrder.SetNillableTransitLocationID(sourceOrder.TransitLocationID)
					createOrder.SetEtd(sourceOrder.Etd)
					createOrder.SetEta(sourceOrder.Eta)
				}

				if res.InternalReferenceNo != nil && strings.TrimSpace(*res.InternalReferenceNo) != "" {
					createOrder.SetInternalReferenceNo(strings.TrimSpace(*res.InternalReferenceNo))
				} else {
					createOrder.SetInternalReferenceNo("")
				}
				if res.BookingNotes != nil {
					createOrder.SetBookingNotes(*res.BookingNotes)
				}
				if res.AllocationNotes != nil {
					createOrder.SetAllocationNotes(*res.AllocationNotes)
				} else if target == nil || target.TargetType == biz.SplitTargetTypeCurrent {
					createOrder.SetAllocationNotes(sourceOrder.AllocationNotes)
				} else {
					createOrder.SetAllocationNotes("")
				}
				if res.OperationNotes != nil {
					createOrder.SetOperationNotes(*res.OperationNotes)
				}

				newOrder, err := createOrder.Save(ctx)
				if err != nil {
					return mapEntConstraint(err, "order_organization_id_order_no", biz.ErrOrderNumberExists)
				}

				createdOrdersMap[res.ClientResultKey] = newOrder
				resultOrderMap[res.ClientResultKey] = newOrder.ID
				resultOrderNoMap[res.ClientResultKey] = newOrder.OrderNo

				// 复制人员
				sourcePersonnel, err := tx.OrderPersonnel.Query().
					Where(orderpersonnelent.OrderIDEQ(sourceOrder.ID)).
					All(ctx)
				if err != nil {
					return err
				}
				pList := make([]*biz.OrderPersonnel, 0, len(sourcePersonnel))
				for _, sp := range sourcePersonnel {
					pList = append(pList, &biz.OrderPersonnel{
						UserID:         sp.UserID,
						OrganizationID: sp.OrganizationID,
						Role:           biz.OrderPersonnelRole(sp.Role),
					})
				}
				if err := createOrderPersonnel(ctx, tx, organizationID, newOrder.ID, newOrder.OrderNo, pList); err != nil {
					return err
				}

				// 复制提成归属快照
				sourceAttributions, err := tx.OrderCommissionAttribution.Query().
					Where(ordercommissionattributionent.OrderIDEQ(sourceOrder.ID)).
					All(ctx)
				if err != nil {
					return err
				}
				attrBuilders := make([]*ent.OrderCommissionAttributionCreate, 0, len(sourceAttributions))
				for _, sa := range sourceAttributions {
					attrBuilders = append(attrBuilders, tx.OrderCommissionAttribution.Create().
						SetID(uuid.Must(uuid.NewV7())).
						SetOrganizationID(organizationID).
						SetOrderID(newOrder.ID).
						SetCustomerID(sa.CustomerID).
						SetSourceAssignmentID(sa.SourceAssignmentID).
						SetEmployeeID(sa.EmployeeID).
						SetEmployeeName(sa.EmployeeName).
						SetPersonnelRole(sa.PersonnelRole).
						SetAttributedAt(sa.AttributedAt),
					)
				}
				if len(attrBuilders) > 0 {
					if _, err := tx.OrderCommissionAttribution.CreateBulk(attrBuilders...).Save(ctx); err != nil {
						return err
					}
				}

				// 复制订单组织标签
				sourceTags, err := tx.OrderEnterpriseTag.Query().
					Where(orderenterprisetagent.OrderIDEQ(sourceOrder.ID)).
					All(ctx)
				if err != nil {
					return err
				}
				tagBuilders := make([]*ent.OrderEnterpriseTagCreate, 0, len(sourceTags))
				for _, st := range sourceTags {
					tagBuilders = append(tagBuilders, tx.OrderEnterpriseTag.Create().
						SetOrganizationID(organizationID).
						SetOrderID(newOrder.ID).
						SetTagResourceID(st.TagResourceID),
					)
				}
				if len(tagBuilders) > 0 {
					if _, err := tx.OrderEnterpriseTag.CreateBulk(tagBuilders...).Save(ctx); err != nil {
						return err
					}
				}

				// 复制服务类型与品类
				serviceTypes, err := sourceOrder.QueryServiceTypes().All(ctx)
				if err != nil {
					return err
				}
				cargoCategories, err := sourceOrder.QueryCargoCategories().All(ctx)
				if err != nil {
					return err
				}
				sIDs := make([]uuid.UUID, 0, len(serviceTypes))
				for _, s := range serviceTypes {
					sIDs = append(sIDs, s.MasterDataItemID)
				}
				cIDs := make([]uuid.UUID, 0, len(cargoCategories))
				for _, c := range cargoCategories {
					cIDs = append(cIDs, c.MasterDataItemID)
				}
				if err := replaceOrderSelections(ctx, tx, newOrder.ID, sIDs, cIDs); err != nil {
					return err
				}

				// 写入来源生命周期事件
				if _, err := tx.OrderLifecycleEvent.Create().
					SetOrderID(newOrder.ID).
					SetDimension(orderlifecycleeventent.DimensionORIGIN).
					SetToStatus(string(sourceOrder.FlowStatus)).
					SetAction("CREATED_BY_SPLIT").
					SetReferenceType("SEA_ORDER_SPLIT_EVENT").
					SetReferenceID(splitEventID).
					SetOperatorID(actorID).
					Save(ctx); err != nil {
					return err
				}

				// A.1: 新票必须先建立指向来源当前 MBL 的初始 ACTIVE Link!
				initialChildLinkBuilder := tx.SeaMasterBillOrderLink.Create().
					SetID(uuid.Must(uuid.NewV7())).
					SetOrganizationID(organizationID).
					SetMasterBillID(lockedActiveLink.MasterBillID).
					SetTransportExecutionID(lockedActiveLink.TransportExecutionID).
					SetOrderID(newOrder.ID).
					SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
					SetStatus(seamasterbillorderlinkent.StatusACTIVE).
					SetStartedAt(now).
					SetVersion(1)
				initialChildLink, err := initialChildLinkBuilder.Save(ctx)
				if err != nil {
					return err
				}

				var finalMblID uuid.UUID
				var finalTEID uuid.UUID

				if target.TargetType == biz.SplitTargetTypeCurrent {
					finalMblID = lockedActiveLink.MasterBillID
					resultLinkMap[res.ClientResultKey] = initialChildLink.ID
				} else {
					tEntity := targetEntities[target.ClientTargetKey]
					if tEntity == nil {
						return biz.ErrSeaOrderSplitInvalidArgument
					}
					finalMblID = tEntity.MBL.ID
					finalTEID = tEntity.TE.ID

					// 结束该新票自己的初始 Link
					if _, err := tx.SeaMasterBillOrderLink.UpdateOneID(initialChildLink.ID).
						SetStatus(seamasterbillorderlinkent.StatusENDED).
						SetEndedAt(now).
						SetVersion(initialChildLink.Version + 1).
						Save(ctx); err != nil {
						return err
					}

					// 建立 final Link
					finalChildLinkBuilder := tx.SeaMasterBillOrderLink.Create().
						SetID(uuid.Must(uuid.NewV7())).
						SetOrganizationID(organizationID).
						SetMasterBillID(finalMblID).
						SetTransportExecutionID(finalTEID).
						SetOrderID(newOrder.ID).
						SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
						SetStatus(seamasterbillorderlinkent.StatusACTIVE).
						SetStartedAt(now).
						SetVersion(1)
					finalChildLink, err := finalChildLinkBuilder.Save(ctx)
					if err != nil {
						return err
					}
					resultLinkMap[res.ClientResultKey] = finalChildLink.ID

					childBeforeReassignMap := map[string]interface{}{
						"schema_version": 1,
						"link_id":        initialChildLink.ID,
						"link_version":   initialChildLink.Version,
						"master_bill_id": lockedActiveLink.MasterBillID,
						"master_no":      lockedMBLs[lockedActiveLink.MasterBillID].MasterNo,
					}
					childAfterReassignMap := map[string]interface{}{
						"schema_version": 1,
						"link_id":        finalChildLink.ID,
						"link_version":   finalChildLink.Version,
						"master_bill_id": finalMblID,
						"master_no":      lockedMBLs[finalMblID].MasterNo,
					}
					childBeforeBytes, err := json.Marshal(childBeforeReassignMap)
					if err != nil {
						return err
					}
					childAfterBytes, err := json.Marshal(childAfterReassignMap)
					if err != nil {
						return err
					}

					reassignEvtID := uuid.Must(uuid.NewV7())
					reassignEvt, err := tx.SeaOrderReassignmentEvent.Create().
						SetID(reassignEvtID).
						SetOrganizationID(organizationID).
						SetOrderID(newOrder.ID).
						SetOrderNo(newOrder.OrderNo).
						SetSplitEventID(splitEventID).
						SetIdempotencyKey(input.IdempotencyKey + ":reassign:" + res.ClientResultKey).
						SetRequestFingerprint(input.RequestFingerprint).
						SetPreviousMasterBillID(lockedActiveLink.MasterBillID).
						SetTargetMasterBillID(finalMblID).
						SetPreviousTransportExecutionID(lockedActiveLink.TransportExecutionID).
						SetTargetTransportExecutionID(finalTEID).
						SetPreviousLinkID(initialChildLink.ID). // 必须属于该结果订单!
						SetTargetLinkID(finalChildLink.ID).
						SetPreviousLinkVersion(initialChildLink.Version).
						SetTargetLinkVersion(1).
						SetReason("部分拆票新票改配").
						SetResponsibilityType(seaorderreassignmenteventent.ResponsibilityTypeOWN_COMPANY).
						SetBeforeSnapshot(childBeforeBytes).
						SetAfterSnapshot(childAfterBytes).
						SetCreatedBy(actorID).
						SetConfirmedByParty(input.Confirmation.ConfirmedByParty).
						SetConfirmedAt(input.Confirmation.ConfirmedAt).
						SetConfirmationNote(input.Confirmation.ConfirmationNote).
						SetNillableConfirmationAttachmentID(input.Confirmation.ConfirmationAttachmentID).
						Save(ctx)
					if err != nil {
						return err
					}
					reassignmentEventIDs = append(reassignmentEventIDs, reassignEvt.ID)

					// 更新新订单权威航程投影
					targetTE := lockedTEs[finalTEID]
					orderUpdate := tx.Order.UpdateOneID(newOrder.ID).
						SetVesselVoyage(biz.CombineVesselVoyage(targetTE.VesselName, targetTE.VoyageNo))
					orderUpdate.SetShippingLineID(targetTE.ShippingLineID)
					if targetTE.OriginLocationID != nil {
						orderUpdate.SetOriginLocationID(*targetTE.OriginLocationID)
					} else {
						orderUpdate.ClearOriginLocationID()
					}
					if targetTE.DischargeLocationID != nil {
						orderUpdate.SetDischargeLocationID(*targetTE.DischargeLocationID)
					} else {
						orderUpdate.ClearDischargeLocationID()
					}
					if targetTE.TransitLocationID != nil {
						orderUpdate.SetTransitLocationID(*targetTE.TransitLocationID)
					} else {
						orderUpdate.ClearTransitLocationID()
					}
					if targetTE.Etd != nil {
						orderUpdate.SetEtd(targetTE.Etd.Format(time.RFC3339))
					} else {
						orderUpdate.ClearEtd()
					}
					if targetTE.Eta != nil {
						orderUpdate.SetEta(targetTE.Eta.Format(time.RFC3339))
					} else {
						orderUpdate.ClearEta()
					}
					if _, err := orderUpdate.Save(ctx); err != nil {
						return err
					}
				}
				resultFinalMblMap[res.ClientResultKey] = finalMblID
			}
		}

		// -------------------------------------------------------------------
		// 处理 HBL（批次内一号一案：此处 resultFinalMblMap 已完备，按目标批次
		// 排重，含作废行；请求内同目标批次同号拒绝、跨目标批次放行。并发竞态
		// 由 (master_bill_id, normalized_house_no) 唯一索引兜底。）
		// -------------------------------------------------------------------
		resultHouseBillMap := make(map[string]*ent.SeaHouseBill)
		resultBatchHouseNos := make(map[string]string)
		for _, res := range input.Results {
			targetOrderID := resultOrderMap[res.ClientResultKey]
			finalMblID := resultFinalMblMap[res.ClientResultKey]

			if res.ResultRole == biz.ResultRoleOriginal {
				if curHBL != nil {
					if curHBL.MasterBillID != finalMblID {
						// 原票改挂新批次前按（目标主单, 现有分单号）查重，
						// 排除自身、不过滤状态：新批次已有同号行（含作废）
						// 即阻断改挂。
						duplicateExists, err := tx.SeaHouseBill.Query().
							Where(
								seahousebillent.OrganizationIDEQ(organizationID),
								seahousebillent.MasterBillIDEQ(finalMblID),
								seahousebillent.NormalizedHouseNoEQ(curHBL.NormalizedHouseNo),
								seahousebillent.IDNEQ(curHBL.ID),
							).
							Exist(ctx)
						if err != nil {
							return err
						}
						if duplicateExists {
							return biz.MetadataError(biz.ErrSeaHouseBillExists, map[string]string{
								"reason":         "HOUSE_BILL_DUPLICATE",
								"house_no":       curHBL.HouseNo,
								"master_bill_id": finalMblID.String(),
							})
						}
						updatedHBL, err := tx.SeaHouseBill.UpdateOneID(curHBL.ID).
							SetMasterBillID(finalMblID).
							SetVersion(curHBL.Version + 1).
							Save(ctx)
						if err != nil {
							return err
						}
						resultHouseBillMap[res.ClientResultKey] = updatedHBL
					} else {
						resultHouseBillMap[res.ClientResultKey] = curHBL
					}
				}
			} else {
				// CREATED 新票
				if lockedActiveLink.DocumentStructure == seamasterbillorderlinkent.DocumentStructureHOUSE {
					if res.HouseBill == nil {
						return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
							"reason":            "HOUSE_BILL_REQUIRED",
							"client_result_key": res.ClientResultKey,
						})
					}
					normNo, err := biz.NormalizeSeaHouseNo(res.HouseBill.HouseNo)
					if err != nil {
						return err
					}
					// 请求内批次排重：同请求两个结果票同号且目标同一批次拒绝；
					// 目标批次不同放行（跨批次重号合法）。
					batchHouseKey := finalMblID.String() + "|" + normNo
					if prevKey, seen := resultBatchHouseNos[batchHouseKey]; seen {
						return biz.MetadataError(biz.ErrSeaHouseBillExists, map[string]string{
							"reason":            "HOUSE_BILL_DUPLICATE",
							"house_no":          res.HouseBill.HouseNo,
							"master_bill_id":    finalMblID.String(),
							"conflict_result":   prevKey,
							"client_result_key": res.ClientResultKey,
						})
					}
					// 目标批次既有行查重（含作废，一号一案）。
					duplicateExists, err := tx.SeaHouseBill.Query().
						Where(
							seahousebillent.OrganizationIDEQ(organizationID),
							seahousebillent.MasterBillIDEQ(finalMblID),
							seahousebillent.NormalizedHouseNoEQ(normNo),
						).
						Exist(ctx)
					if err != nil {
						return err
					}
					if duplicateExists {
						return biz.MetadataError(biz.ErrSeaHouseBillExists, map[string]string{
							"reason":            "HOUSE_BILL_DUPLICATE",
							"house_no":          res.HouseBill.HouseNo,
							"master_bill_id":    finalMblID.String(),
							"client_result_key": res.ClientResultKey,
						})
					}
					resultBatchHouseNos[batchHouseKey] = res.ClientResultKey
					hbInput := &biz.SeaHouseBillInput{
						HouseNo:         res.HouseBill.HouseNo,
						IssuerSource:    biz.SeaHouseBillIssuerSource(res.HouseBill.IssuerSource),
						IssuerPartnerID: res.HouseBill.IssuerPartnerID,
						Note:            res.HouseBill.Note,
					}
					issuerOrgID, issuerPartnerID, err := validateSeaHouseBillIssuer(ctx, tx.Client(), organizationID, sourceOrder.OrganizationID, sourceOrder.CustomerID, hbInput)
					if err != nil {
						return err
					}
					b := tx.SeaHouseBill.Create().
						SetID(uuid.Must(uuid.NewV7())).
						SetOrganizationID(organizationID).
						SetOrderID(targetOrderID).
						SetMasterBillID(finalMblID).
						SetHouseNo(res.HouseBill.HouseNo).
						SetNormalizedHouseNo(normNo).
						SetIssuerSource(seahousebillent.IssuerSource(res.HouseBill.IssuerSource)).
						SetStatus(seahousebillent.StatusDRAFT).
						SetVersion(1)
					if issuerOrgID != nil {
						b.SetIssuerOrganizationID(*issuerOrgID)
					}
					if issuerPartnerID != nil {
						b.SetIssuerPartnerID(*issuerPartnerID)
					}
					if res.HouseBill.Note != nil {
						b.SetNote(*res.HouseBill.Note)
					}
					createdHBL, err := b.Save(ctx)
					if err != nil {
						if ent.IsConstraintError(err) {
							return biz.ErrSeaHouseBillExists
						}
						return err
					}
					resultHouseBillMap[res.ClientResultKey] = createdHBL
				}
			}
		}

		// -------------------------------------------------------------------
		// 迁移集装箱 OrderContainer
		// -------------------------------------------------------------------
		for cID, rKey := range containerToResultKey {
			targetOrderID := resultOrderMap[rKey]
			lockedContainer := lockedContainerMap[cID]
			if lockedContainer != nil && lockedContainer.OrderID != targetOrderID {
				if _, err := tx.OrderContainer.UpdateOneID(cID).
					SetOrderID(targetOrderID).
					SetVersion(lockedContainer.Version + 1).
					Save(ctx); err != nil {
					return err
				}
			}
		}

		// -------------------------------------------------------------------
		// 货物重构与分配迁移
		// -------------------------------------------------------------------
		resultCargoOldNewMap := make(map[string]map[string]string)
		resultCargoRetainedIDs := make(map[string][]string)
		resultAllocOldNewMap := make(map[string]map[string]string)
		for _, res := range input.Results {
			resultCargoOldNewMap[res.ClientResultKey] = make(map[string]string)
			resultCargoRetainedIDs[res.ClientResultKey] = make([]string, 0)
			resultAllocOldNewMap[res.ClientResultKey] = make(map[string]string)
		}

		// 1. 处理货物 OrderCargoItem
		var zeroRemainingCargoIDs []uuid.UUID
		for _, res := range input.Results {
			targetOrderID := resultOrderMap[res.ClientResultKey]

			if res.ResultRole == biz.ResultRoleCreated {
				for _, ca := range res.CargoAllocations {
					if ca.PackageCount <= 0 && ca.GrossWeightKg.IsZero() && ca.VolumeCbm.IsZero() {
						continue
					}
					origCargoItem := lockedCargoItemMap[ca.CargoItemID]
					if origCargoItem == nil {
						continue
					}
					fWeight, _ := ca.GrossWeightKg.Float64()
					fVol, _ := ca.VolumeCbm.Float64()

					createdItem, err := tx.OrderCargoItem.Create().
						SetID(uuid.Must(uuid.NewV7())).
						SetOrganizationID(organizationID).
						SetOrderID(targetOrderID).
						SetCargoName(origCargoItem.CargoName).
						SetPackageCount(int(ca.PackageCount)).
						SetGrossWeightKg(fWeight).
						SetVolumeCbm(fVol).
						SetVersion(1).
						Save(ctx)
					if err != nil {
						return err
					}
					resultCargoOldNewMap[res.ClientResultKey][origCargoItem.ID.String()] = createdItem.ID.String()
				}
			} else {
				// 原票：逐个原始货物比对分配剩余
				caMap := make(map[uuid.UUID]*biz.SeaOrderSplitCargoAllocationInput)
				for _, ca := range res.CargoAllocations {
					caMap[ca.CargoItemID] = ca
				}
				for _, ci := range cargoItems {
					ca := caMap[ci.ID]
					if ca != nil && (ca.PackageCount > 0 || !ca.GrossWeightKg.IsZero() || !ca.VolumeCbm.IsZero()) {
						fWeight, _ := ca.GrossWeightKg.Float64()
						fVol, _ := ca.VolumeCbm.Float64()

						if _, err := tx.OrderCargoItem.UpdateOneID(ci.ID).
							SetPackageCount(int(ca.PackageCount)).
							SetGrossWeightKg(fWeight).
							SetVolumeCbm(fVol).
							SetVersion(ci.Version + 1).
							Save(ctx); err != nil {
							return err
						}
						resultCargoRetainedIDs[res.ClientResultKey] = append(resultCargoRetainedIDs[res.ClientResultKey], ci.ID.String())
						resultCargoOldNewMap[res.ClientResultKey][ci.ID.String()] = ci.ID.String()
					} else {
						zeroRemainingCargoIDs = append(zeroRemainingCargoIDs, ci.ID)
					}
				}
			}
		}

		// 2. 处理共享箱分配 SeaSharedContainerAllocation
		affectedSharedContainers := make(map[uuid.UUID]struct{})
		for _, res := range input.Results {
			targetOrderID := resultOrderMap[res.ClientResultKey]
			targetHBL := resultHouseBillMap[res.ClientResultKey]
			var targetHBLID uuid.UUID
			if targetHBL != nil {
				targetHBLID = targetHBL.ID
			}

			if res.ResultRole == biz.ResultRoleCreated {
				for _, sca := range res.SharedContainerAllocations {
					if sca.PackageCount <= 0 && sca.GrossWeightKg.IsZero() && sca.VolumeCbm.IsZero() {
						continue
					}
					origAlloc := lockedSharedAllocMap[sca.AllocationID]
					if origAlloc == nil {
						continue
					}
					newCargoItemIDStr, ok := resultCargoOldNewMap[res.ClientResultKey][origAlloc.CargoItemID.String()]
					if !ok {
						return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
							"reason":        "CARGO_ITEM_MAPPING_NOT_FOUND",
							"cargo_item_id": origAlloc.CargoItemID.String(),
						})
					}
					newCargoItemID := uuid.MustParse(newCargoItemIDStr)

					createdAlloc, err := tx.SeaSharedContainerAllocation.Create().
						SetID(uuid.Must(uuid.NewV7())).
						SetOrganizationID(organizationID).
						SetSharedContainerID(origAlloc.SharedContainerID).
						SetOrderID(targetOrderID).
						SetHouseBillID(targetHBLID).
						SetCargoItemID(newCargoItemID).
						SetPackageCount(int(sca.PackageCount)).
						SetGrossWeightKg(sca.GrossWeightKg.StringFixed(3)).
						SetVolumeCbm(sca.VolumeCbm.StringFixed(6)).
						SetVersion(1).
						Save(ctx)
					if err != nil {
						return err
					}
					resultAllocOldNewMap[res.ClientResultKey][origAlloc.ID.String()] = createdAlloc.ID.String()
					affectedSharedContainers[origAlloc.SharedContainerID] = struct{}{}
				}
			} else {
				// 原票：更新剩余或删除零剩余分配；数值未变化时跳过写入避免虚假版本递增
				scaMap := make(map[uuid.UUID]*biz.SeaOrderSplitSharedContainerAllocationInput)
				for _, sca := range res.SharedContainerAllocations {
					scaMap[sca.AllocationID] = sca
				}
				for _, origAlloc := range sharedAllocs {
					sca := scaMap[origAlloc.ID]
					origWt, wtErr := decimal.NewFromString(origAlloc.GrossWeightKg)
					if wtErr != nil {
						return wtErr
					}
					origVol, volErr := decimal.NewFromString(origAlloc.VolumeCbm)
					if volErr != nil {
						return volErr
					}
					if sca != nil && (sca.PackageCount > 0 || !sca.GrossWeightKg.IsZero() || !sca.VolumeCbm.IsZero()) {
						if int(sca.PackageCount) != origAlloc.PackageCount || !sca.GrossWeightKg.Equal(origWt) || !sca.VolumeCbm.Equal(origVol) {
							if _, err := tx.SeaSharedContainerAllocation.UpdateOneID(origAlloc.ID).
								SetPackageCount(int(sca.PackageCount)).
								SetGrossWeightKg(sca.GrossWeightKg.StringFixed(3)).
								SetVolumeCbm(sca.VolumeCbm.StringFixed(6)).
								SetVersion(origAlloc.Version + 1).
								Save(ctx); err != nil {
								return err
							}
							affectedSharedContainers[origAlloc.SharedContainerID] = struct{}{}
						}
						resultAllocOldNewMap[res.ClientResultKey][origAlloc.ID.String()] = origAlloc.ID.String()
					} else {
						// 零剩余共享箱分配先删除
						if err := tx.SeaSharedContainerAllocation.DeleteOneID(origAlloc.ID).Exec(ctx); err != nil {
							return err
						}
						affectedSharedContainers[origAlloc.SharedContainerID] = struct{}{}
					}
				}
			}
		}

		// 2.1 共享箱聚合版本闭环：Allocation 发生创建、变更或删除后，
		// 每个受影响 SharedContainer 在同一事务内恰好递增一次版本，按已排序 ID 固定顺序写入。
		for _, scID := range sharedContainerIDs {
			if _, affected := affectedSharedContainers[scID]; !affected {
				continue
			}
			lockedSC := lockedSharedContainers[scID]
			if _, err := tx.SeaSharedContainer.UpdateOneID(scID).
				SetVersion(lockedSC.Version + 1).
				Save(ctx); err != nil {
				return err
			}
		}

		// 3. 彻底删除零剩余货物项 (此时其共享箱分配已删除，外键安全)
		for _, zID := range zeroRemainingCargoIDs {
			if err := tx.OrderCargoItem.DeleteOneID(zID).Exec(ctx); err != nil {
				return err
			}
		}

		// A.8: 迁移未建账费用：整行克隆到新订单，记录每个结果自己的 fee old->new ID 映射
		resultFeeOldNewMap := make(map[string]map[string]string)
		for _, res := range input.Results {
			resultFeeOldNewMap[res.ClientResultKey] = make(map[string]string)
			targetOrderID := resultOrderMap[res.ClientResultKey]
			if res.ResultRole == biz.ResultRoleCreated {
				for _, feeID := range res.DraftFeeIDs {
					oldFee := lockedFeesMap[feeID]
					newFeeID := uuid.Must(uuid.NewV7())
					newFeeIdempotencyKey := fmt.Sprintf("%s:split:%s", oldFee.IdempotencyKey, res.ClientResultKey)

					createBuilder := tx.OrderFee.Create().
						SetID(newFeeID).
						SetOrderID(targetOrderID).
						SetIdempotencyKey(newFeeIdempotencyKey).
						SetDirection(oldFee.Direction).
						SetStatus(oldFee.Status).
						SetNillableFeeSettingID(oldFee.FeeSettingID).
						SetFeeCode(oldFee.FeeCode).
						SetFeeName(oldFee.FeeName).
						SetNillableFeeNameEn(oldFee.FeeNameEn).
						SetSettlementPartyID(oldFee.SettlementPartyID).
						SetNillableBillingUnitID(oldFee.BillingUnitID).
						SetBillingUnit(oldFee.BillingUnit).
						SetNillableTaxRate(oldFee.TaxRate).
						SetNillableTaxableServiceName(oldFee.TaxableServiceName).
						SetQuantity(oldFee.Quantity).
						SetUnitPrice(oldFee.UnitPrice).
						SetTotalAmount(oldFee.TotalAmount).
						SetTaxInclusive(oldFee.TaxInclusive).
						SetNetAmount(oldFee.NetAmount).
						SetTaxAmount(oldFee.TaxAmount).
						SetCurrency(oldFee.Currency).
						SetExchangeRate(oldFee.ExchangeRate).
						SetExchangeRateSource(oldFee.ExchangeRateSource).
						SetExchangeRateDate(oldFee.ExchangeRateDate).
						SetNillableExchangeRateSettingID(oldFee.ExchangeRateSettingID).
						SetBaseCurrency(oldFee.BaseCurrency).
						SetBaseCurrencyAmount(oldFee.BaseCurrencyAmount).
						SetExpenseDate(oldFee.ExpenseDate).
						SetNote(oldFee.Note).
						SetVersion(1)

					if _, err := createBuilder.Save(ctx); err != nil {
						return err
					}

					// 复制企业标签关联
					existingTags, err := tx.OrderFeeEnterpriseTag.Query().
						Where(orderfeeenterprisetagent.OrderFeeIDEQ(oldFee.ID)).
						All(ctx)
					if err != nil {
						return err
					}
					for _, et := range existingTags {
						if _, err := tx.OrderFeeEnterpriseTag.Create().
							SetOrganizationID(et.OrganizationID).
							SetOrderFeeID(newFeeID).
							SetTagResourceID(et.TagResourceID).
							Save(ctx); err != nil {
							return err
						}
					}

					// 删除原费用的标签关联及原费用
					if _, err := tx.OrderFeeEnterpriseTag.Delete().
						Where(orderfeeenterprisetagent.OrderFeeIDEQ(oldFee.ID)).
						Exec(ctx); err != nil {
						return err
					}
					if err := tx.OrderFee.DeleteOneID(oldFee.ID).Exec(ctx); err != nil {
						return err
					}

					resultFeeOldNewMap[res.ClientResultKey][oldFee.ID.String()] = newFeeID.String()
				}
			}
		}

		// A.9: 处理附件引用 OrderAttachment (显式引用，不复制对象)
		for _, res := range input.Results {
			if res.ResultRole == biz.ResultRoleCreated {
				targetOrderID := resultOrderMap[res.ClientResultKey]
				for _, attID := range res.AttachmentReferenceIDs {
					var origAtt *ent.OrderAttachment
					for _, oa := range orderAttachments {
						if oa.ID == attID {
							origAtt = oa
							break
						}
					}
					if origAtt == nil {
						return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
							"reason":                  "ATTACHMENT_REFERENCE_NOT_FOUND",
							"attachment_reference_id": attID.String(),
						})
					}
					newKey := fmt.Sprintf("split:%s:%s", targetOrderID.String(), origAtt.AssetID.String())
					if _, err := tx.OrderAttachment.Create().
						SetOrderID(targetOrderID).
						SetAssetID(origAtt.AssetID).
						SetDocType(origAtt.DocType).
						SetIdempotencyKey(newKey).
						SetCreatedBy(actorID).
						Save(ctx); err != nil {
						return err
					}
				}
			}
		}

		// A.5: FCL 箱计划：新票=各规格实际箱数；原票=原计划减全部拆出实际箱后的未落实余量 + 原票实际箱；LCL/散杂不得生成计划
		resultContainerPlans := make(map[string][]map[string]interface{})
		for _, res := range input.Results {
			resultContainerPlans[res.ClientResultKey] = make([]map[string]interface{}, 0)
		}

		if sourceOrder.ShipmentType != nil && *sourceOrder.ShipmentType == orderent.ShipmentTypeFCL {
			origPlanBySpec := make(map[uuid.UUID]int)
			for _, bp := range existingBeforePlans {
				origPlanBySpec[bp.ContainerSpecID] = bp.Quantity
			}

			// 删除原票现有箱计划
			if _, err := tx.OrderContainerRequest.Delete().
				Where(ordercontainerrequestent.OrderIDEQ(sourceOrder.ID)).
				Exec(ctx); err != nil {
				return err
			}

			actualBySpecByResult := make(map[string]map[uuid.UUID]int)
			totalSplitOutBySpec := make(map[uuid.UUID]int)
			retainedActualBySpec := make(map[uuid.UUID]int)
			for _, res := range input.Results {
				actualBySpecByResult[res.ClientResultKey] = make(map[uuid.UUID]int)
			}
			for cID, rKey := range containerToResultKey {
				c := lockedContainerMap[cID]
				if c != nil && c.ContainerSpecID != uuid.Nil {
					actualBySpecByResult[rKey][c.ContainerSpecID]++
					if rKey != originalResultKey {
						totalSplitOutBySpec[c.ContainerSpecID]++
					} else {
						retainedActualBySpec[c.ContainerSpecID]++
					}
				}
			}

			allSpecs := make(map[uuid.UUID]bool)
			for specID := range origPlanBySpec {
				allSpecs[specID] = true
			}
			for _, specMap := range actualBySpecByResult {
				for specID := range specMap {
					allSpecs[specID] = true
				}
			}

			// 原票箱计划: 原计划减全部拆出实际箱后的未落实余量 + 原票实际箱
			for specID := range allSpecs {
				origPlan := origPlanBySpec[specID]
				splitOut := totalSplitOutBySpec[specID]
				retained := retainedActualBySpec[specID]
				planQty := (origPlan - splitOut)
				if planQty < retained {
					planQty = retained
				}
				if planQty > 0 {
					if _, err := tx.OrderContainerRequest.Create().
						SetOrderID(sourceOrder.ID).
						SetContainerSpecID(specID).
						SetQuantity(planQty).
						Save(ctx); err != nil {
						return err
					}
					resultContainerPlans[originalResultKey] = append(resultContainerPlans[originalResultKey], map[string]interface{}{
						"container_spec_id": specID,
						"quantity":          planQty,
					})
				}
			}

			// 新票箱计划: 各规格实际箱数
			for _, res := range input.Results {
				if res.ResultRole == biz.ResultRoleCreated {
					childOrderID := resultOrderMap[res.ClientResultKey]
					for specID, actualCount := range actualBySpecByResult[res.ClientResultKey] {
						if actualCount > 0 {
							if _, err := tx.OrderContainerRequest.Create().
								SetOrderID(childOrderID).
								SetContainerSpecID(specID).
								SetQuantity(actualCount).
								Save(ctx); err != nil {
								return err
							}
							resultContainerPlans[res.ClientResultKey] = append(resultContainerPlans[res.ClientResultKey], map[string]interface{}{
								"container_spec_id": specID,
								"quantity":          actualCount,
							})
						}
					}
				}
			}
		} else {
			// LCL / 散杂等不得生成计划，清理原票已存在的箱计划
			if _, err := tx.OrderContainerRequest.Delete().
				Where(ordercontainerrequestent.OrderIDEQ(sourceOrder.ID)).
				Exec(ctx); err != nil {
				return err
			}
		}

		// -------------------------------------------------------------------
		// 更新原订单版本与生命周期事件
		// -------------------------------------------------------------------
		if _, err := tx.Order.UpdateOneID(sourceOrder.ID).
			SetVersion(sourceOrder.Version + 1).
			Save(ctx); err != nil {
			return err
		}

		if _, err := tx.OrderLifecycleEvent.Create().
			SetOrderID(sourceOrder.ID).
			SetDimension(orderlifecycleeventent.DimensionFLOW).
			SetToStatus(string(sourceOrder.FlowStatus)).
			SetAction("SPLIT_SOURCE").
			SetReferenceType("SEA_ORDER_SPLIT_EVENT").
			SetReferenceID(splitEventID).
			SetOperatorID(actorID).
			Save(ctx); err != nil {
			return err
		}

		// -------------------------------------------------------------------
		// 写入 SeaOrderSplitResult 记录
		// -------------------------------------------------------------------
		createdResults := make([]*biz.SeaOrderSplitResult, 0, len(input.Results))
		for seqIdx, res := range input.Results {
			targetOrderID := resultOrderMap[res.ClientResultKey]
			targetOrderNo := resultOrderNoMap[res.ClientResultKey]
			finalMblID := resultFinalMblMap[res.ClientResultKey]
			cStat := resCargoStats[res.ClientResultKey]

			var hbSnapshot map[string]interface{}
			if hb := resultHouseBillMap[res.ClientResultKey]; hb != nil {
				hbSnapshot = map[string]interface{}{
					"id":       hb.ID,
					"house_no": hb.HouseNo,
					"version":  hb.Version,
				}
			}

			resultSnapshotData := map[string]interface{}{
				"schema_version":            1,
				"order_id":                  targetOrderID,
				"order_no":                  targetOrderNo,
				"client_result_key":         res.ClientResultKey,
				"result_role":               res.ResultRole,
				"client_target_key":         res.ClientTargetKey,
				"package_count":             cStat.pkg,
				"gross_weight_kg":           biz.FormatDecimal3(cStat.weight),
				"volume_cbm":                biz.FormatDecimal6(cStat.vol),
				"house_bill":                hbSnapshot,
				"draft_fee_ids":             res.DraftFeeIDs,
				"attachment_reference_ids":  res.AttachmentReferenceIDs,
				"container_plans":           resultContainerPlans[res.ClientResultKey],
				"cargo_old_new_id_map":      resultCargoOldNewMap[res.ClientResultKey],
				"cargo_retained_ids":        resultCargoRetainedIDs[res.ClientResultKey],
				"allocation_old_new_id_map": resultAllocOldNewMap[res.ClientResultKey],
				"fee_old_new_id_map":        resultFeeOldNewMap[res.ClientResultKey], // 严格属于该结果
			}
			resultSnapshotBytes, err := json.Marshal(resultSnapshotData)
			if err != nil {
				return err
			}

			splitRes, err := tx.SeaOrderSplitResult.Create().
				SetID(uuid.Must(uuid.NewV7())).
				SetSplitEventID(splitEventID).
				SetOrganizationID(organizationID).
				SetOrderID(targetOrderID).
				SetOrderNo(targetOrderNo).
				SetResultRole(seaordersplitresultent.ResultRole(res.ResultRole)).
				SetSequence(seqIdx + 1).
				SetClientResultKey(res.ClientResultKey).
				SetInitialMasterBillID(lockedActiveLink.MasterBillID).
				SetFinalMasterBillID(finalMblID).
				SetResultSnapshot(resultSnapshotBytes).
				Save(ctx)
			if err != nil {
				return err
			}

			createdResults = append(createdResults, &biz.SeaOrderSplitResult{
				ID:                  splitRes.ID,
				CreatedAt:           splitRes.CreatedAt,
				SplitEventID:        splitRes.SplitEventID,
				OrganizationID:      splitRes.OrganizationID,
				OrderID:             splitRes.OrderID,
				OrderNo:             splitRes.OrderNo,
				ResultRole:          string(splitRes.ResultRole),
				Sequence:            splitRes.Sequence,
				ClientResultKey:     splitRes.ClientResultKey,
				InitialMasterBillID: splitRes.InitialMasterBillID,
				FinalMasterBillID:   splitRes.FinalMasterBillID,
				ResultSnapshot:      splitRes.ResultSnapshot,
			})
		}

		splitEventResult = &biz.SeaOrderSplitEvent{
			ID:                   savedSplitEvent.ID,
			CreatedAt:            savedSplitEvent.CreatedAt,
			OrganizationID:       savedSplitEvent.OrganizationID,
			SourceOrderID:        savedSplitEvent.SourceOrderID,
			SourceOrderNo:        savedSplitEvent.SourceOrderNo,
			IdempotencyKey:       savedSplitEvent.IdempotencyKey,
			RequestFingerprint:   savedSplitEvent.RequestFingerprint,
			Note:                 savedSplitEvent.Note,
			SourceOrderVersion:   savedSplitEvent.SourceOrderVersion,
			SourceLinkID:         savedSplitEvent.SourceLinkID,
			SourceLinkVersion:    savedSplitEvent.SourceLinkVersion,
			SourceAllocationVer:  savedSplitEvent.SourceAllocationVersion,
			BeforeSnapshot:       savedSplitEvent.BeforeSnapshot,
			ConservationSnapshot: savedSplitEvent.ConservationSnapshot,
			CreatedBy:            savedSplitEvent.CreatedBy,
			Results:              createdResults,
			ReassignmentEventIDs: reassignmentEventIDs,
		}

		audit.Details["split_event_id"] = splitEventID.String()
		audit.Details["source_order_no"] = sourceOrder.OrderNo
		audit.Details["result_count"] = fmt.Sprintf("%d", len(createdResults))
		return writeAudit(ctx, tx.AuditLog, audit)
	})

	if err != nil {
		return nil, err
	}
	return splitEventResult, nil
}

// ---------------------------------------------------------------------------
// 5. 改配预览与原子执行
// ---------------------------------------------------------------------------
