package data

import (
	"context"
	"sort"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordercommissionattributionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	seahousebill "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillorderlink "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

// seaMasterBillUpdateLockContext 记录 UpdateDraft 在取得共享 MBL 写锁前预锁定的成员订单。
// 后续锁定 MBL 后必须重验成员集合未变化，避免在既有锁序之后补锁新成员而形成死锁。
type seaMasterBillUpdateLockContext struct {
	activeLinkID   uuid.UUID
	masterBillID   uuid.UUID
	memberOrderIDs []uuid.UUID
	memberOrders   []*ent.Order
}

func orderUpdateNeedsSharedMasterBillMemberLocks(input *biz.Order) bool {
	if input == nil || input.BusinessType != biz.OrderBusinessSE {
		return false
	}
	return input.SeaMasterBillInput != nil ||
		input.SeaDocumentInput != nil && input.SeaDocumentInput.MasterBillContent != nil
}

// lockOrderDraftUpdateTarget 为共享 MBL 更新预先按主键升序锁定全部活动成员 Order。
// 非海运出口或未修改共享 MBL 身份、航程、内容时只锁定目标订单。
func lockOrderDraftUpdateTarget(
	ctx context.Context,
	tx *ent.Tx,
	organizationID, orderID uuid.UUID,
	input *biz.Order,
) (*ent.Order, *seaMasterBillUpdateLockContext, error) {
	lockTargetOnly := func() (*ent.Order, *seaMasterBillUpdateLockContext, error) {
		lockedOrder, err := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return nil, nil, mapEntError(err, biz.ErrOrderNotFound, nil)
		}
		return lockedOrder, nil, nil
	}

	if !orderUpdateNeedsSharedMasterBillMemberLocks(input) {
		return lockTargetOnly()
	}

	activeLink, err := tx.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.OrganizationIDEQ(organizationID),
			seamasterbillorderlink.OrderIDEQ(orderID),
			seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return lockTargetOnly()
		}
		return nil, nil, err
	}

	activeLinks, err := tx.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.OrganizationIDEQ(organizationID),
			seamasterbillorderlink.MasterBillIDEQ(activeLink.MasterBillID),
			seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
		).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}
	memberOrderIDs := make([]uuid.UUID, 0, len(activeLinks))
	for _, link := range activeLinks {
		memberOrderIDs = append(memberOrderIDs, link.OrderID)
	}
	memberOrderIDs = sortAndDeduplicateUUIDs(memberOrderIDs)
	if len(memberOrderIDs) == 0 {
		return nil, nil, biz.ErrSeaDocumentStructureConflict
	}

	memberOrders, err := tx.Order.Query().
		Where(
			orderent.OrganizationIDEQ(organizationID),
			orderent.IDIn(memberOrderIDs...),
		).
		Order(orderent.ByID()).
		ForUpdate().
		All(ctx)
	if err != nil {
		return nil, nil, err
	}
	if len(memberOrders) != len(memberOrderIDs) {
		return nil, nil, biz.ErrSeaDocumentStructureConflict
	}

	var targetOrder *ent.Order
	for _, memberOrder := range memberOrders {
		if memberOrder.ID == orderID {
			targetOrder = memberOrder
			break
		}
	}
	if targetOrder == nil {
		return nil, nil, biz.ErrSeaDocumentStructureConflict
	}

	return targetOrder, &seaMasterBillUpdateLockContext{
		activeLinkID:   activeLink.ID,
		masterBillID:   activeLink.MasterBillID,
		memberOrderIDs: memberOrderIDs,
		memberOrders:   memberOrders,
	}, nil
}

// revalidateSeaMasterBillUpdateMemberSet 必须在 MBL 与目标 Link 均已加锁后调用。
// 成员集合变化时直接回滚，禁止在 MBL 锁后补取新的 Order 锁。
func revalidateSeaMasterBillUpdateMemberSet(
	ctx context.Context,
	tx *ent.Tx,
	organizationID, activeLinkID, masterBillID uuid.UUID,
	lockContext *seaMasterBillUpdateLockContext,
) ([]uuid.UUID, error) {
	if lockContext == nil ||
		lockContext.activeLinkID != activeLinkID ||
		lockContext.masterBillID != masterBillID {
		return nil, biz.ErrSeaDocumentStructureConflict
	}

	lockedActiveLinks, err := tx.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.OrganizationIDEQ(organizationID),
			seamasterbillorderlink.MasterBillIDEQ(masterBillID),
			seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	lockedMemberOrderIDs := make([]uuid.UUID, 0, len(lockedActiveLinks))
	for _, activeMemberLink := range lockedActiveLinks {
		lockedMemberOrderIDs = append(lockedMemberOrderIDs, activeMemberLink.OrderID)
	}
	lockedMemberOrderIDs = sortAndDeduplicateUUIDs(lockedMemberOrderIDs)
	if len(lockedMemberOrderIDs) != len(lockContext.memberOrderIDs) {
		return nil, biz.ErrSeaDocumentStructureConflict
	}
	for index, memberOrderID := range lockedMemberOrderIDs {
		if memberOrderID != lockContext.memberOrderIDs[index] {
			return nil, biz.ErrSeaDocumentStructureConflict
		}
	}
	return lockedMemberOrderIDs, nil
}

func ensureLockedMembersAllowSharedMasterBillUpdate(lockContext *seaMasterBillUpdateLockContext) error {
	if lockContext == nil {
		return biz.ErrSeaDocumentStructureConflict
	}
	lockedOrderNos := make([]string, 0, len(lockContext.memberOrders))
	for _, memberOrder := range lockContext.memberOrders {
		if memberOrder.LockedAt != nil {
			lockedOrderNos = append(lockedOrderNos, memberOrder.OrderNo)
		}
	}
	if len(lockedOrderNos) == 0 {
		return nil
	}
	sort.Strings(lockedOrderNos)
	return biz.NewErrSeaMasterBillMemberOrderLocked(len(lockedOrderNos), lockedOrderNos)
}

func (r *orderRepo) UpdateDraft(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, input *biz.Order, audit *biz.AuditEvent) (*biz.Order, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, mblLockContext, queryErr := lockOrderDraftUpdateTarget(ctx, tx, organizationID, id, input)
		if queryErr != nil {
			return queryErr
		}
		// 幂等重放（可变最新键）：同键 + 同 expectedVersion 的请求若已成功写入
		// 一轮（当前行版本 == expectedVersion + 1 且行上键为本次键），直接返回
		// 当前草稿，不产生任何副作用；键不同或版本不匹配走下方既有乐观锁 409。
		if input.IdempotencyKey != "" && existing.IdempotencyKey == input.IdempotencyKey && existing.Version == expectedVersion+1 {
			return nil
		}
		if err := ensureOrderBusinessEditable(ctx, tx, existing); err != nil {
			return err
		}
		if existing.Version != expectedVersion {
			return biz.ErrOrderStatusConflict
		}
		if existing.FlowStatus != orderent.FlowStatusDRAFT || existing.TerminationStatus != orderent.TerminationStatusACTIVE || existing.ClosureStatus != orderent.ClosureStatusOPEN {
			return biz.ErrOrderStatusConflict
		}
		if existing.BusinessType != orderent.BusinessType(input.BusinessType) {
			return biz.ErrOrderBusinessUnsupported
		}
		if validateErr := validateOrderReferences(ctx, tx, organizationID, input, existing); validateErr != nil {
			return validateErr
		}

		// 共享 MBL 门禁必须在 Link/HBL/Order 等下游数据发生任何写入前完成，避免通过
		// 同一请求改变下游事实后绕过校验。后续任一行失败仍由同一事务整体回滚。
		if syncErr := syncOrderSeaMasterBillOnUpdate(ctx, tx, organizationID, existing, input, audit, mblLockContext); syncErr != nil {
			return syncErr
		}
		if syncErr := syncOrderSeaDocumentOnUpdate(ctx, tx, organizationID, existing, input, audit, mblLockContext); syncErr != nil {
			return syncErr
		}
		update := existing.Update().
			SetVersion(existing.Version + 1).
			SetCustomerID(input.CustomerID).
			SetCustomerReferenceNo(input.CustomerReferenceNo).
			SetBookingNo(input.BookingNo).
			SetInternalReferenceNo(input.InternalReferenceNo).
			SetShipperShortName(input.ShipperShortName).
			SetConsigneeShortName(input.ConsigneeShortName).
			SetContractNo(input.ContractNo).
			SetCargoValue(input.CargoValue).
			SetInsurancePremium(input.InsurancePremium).
			SetUnNumber(input.UNNumber).
			SetHazardClass(input.HazardClass).
			SetFactoryName(input.FactoryName).
			SetCargoReadyAt(input.CargoReadyAt).
			SetDeclarationCutoffAt(input.DeclarationCutoffAt).
			SetReceivedAt(input.ReceivedAt).
			SetTradeDirection(orderent.TradeDirection(input.TradeDirection)).
			SetPaymentTerm(orderent.PaymentTerm(input.PaymentTerm)).
			SetVesselVoyage(input.VesselVoyage).
			SetEtd(input.ETD).
			SetEta(input.ETA).
			SetSiCutoff(input.SICutoff).
			SetDocCutoff(input.DocCutoff).
			SetCustomsCutoff(input.CustomsCutoff).
			SetVgmCutoff(input.VGMCutoff).
			SetGoodsDescription(input.GoodsDescription).
			SetTotalPackageUnit(input.TotalPackageUnit).
			SetSpecialRequirements(input.SpecialRequirements).
			SetOrderDate(input.OrderDate).
			SetNotes(input.Notes).
			SetBookingNotes(input.BookingNotes).
			SetAllocationNotes(input.AllocationNotes).
			SetOperationNotes(input.OperationNotes)
		setOrderOptionalReferences(update, input)
		setOrderOptionalAmounts(update, input)
		// 可变最新键：更新成功后以本次请求的幂等键覆写（可选；未提供时保留原键）。
		if input.IdempotencyKey != "" {
			update.SetIdempotencyKey(input.IdempotencyKey)
		}
		if input.TotalPackages == nil {
			update.ClearTotalPackages()
		} else {
			update.SetTotalPackages(*input.TotalPackages)
		}
		if input.TotalGrossWeightKg == nil {
			update.ClearTotalGrossWeightKg()
		} else {
			update.SetTotalGrossWeightKg(*input.TotalGrossWeightKg)
		}
		if input.TotalVolumeCbm == nil {
			update.ClearTotalVolumeCbm()
		} else {
			update.SetTotalVolumeCbm(*input.TotalVolumeCbm)
		}
		if _, updateErr := update.Save(ctx); updateErr != nil {
			return mapEntConstraint(updateErr, "order_organization_id_order_no", biz.ErrOrderNumberExists)
		}
		if replaceErr := replaceOrderSelections(ctx, tx, id, input.ServiceTypeIDs, input.CargoCategoryIDs); replaceErr != nil {
			return replaceErr
		}
		if syncErr := syncOrderShippingDocuments(ctx, tx, organizationID, input.BusinessType, id, input.ShippingDocuments); syncErr != nil {
			return syncErr
		}
		if syncErr := syncOrderContainerRequests(ctx, tx, organizationID, id, input.ContainerRequests); syncErr != nil {
			return syncErr
		}
		if existing.CustomerID != input.CustomerID {
			if existing.BusinessType == orderent.BusinessTypeSE {
				hasCustomerHBL, checkErr := tx.SeaHouseBill.Query().Where(
					seahousebill.OrderIDEQ(id),
					seahousebill.OrganizationIDEQ(organizationID),
					seahousebill.IssuerSourceEQ(seahousebill.IssuerSourceCUSTOMER_PARTNER),
				).Exist(ctx)
				if checkErr != nil {
					return checkErr
				}
				if hasCustomerHBL {
					return biz.ErrOrderCustomerChangeWithHouseBillBlocked
				}
			}
			// 换客户不重拍归属：订单人员分工未变，归属行仅同步客户冗余列；
			// 零行匹配（存量草稿无归属行）视为无操作。
			if _, updateErr := tx.OrderCommissionAttribution.Update().
				Where(ordercommissionattributionent.OrderIDEQ(id)).
				SetCustomerID(input.CustomerID).Save(ctx); updateErr != nil {
				return updateErr
			}
		}
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		if audit.Action == "" {
			audit.Action = "order.update"
		}
		audit.Details["order.no"] = existing.OrderNo
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}
