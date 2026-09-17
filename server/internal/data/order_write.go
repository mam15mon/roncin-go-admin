package data

import (
	"context"
	"fmt"
	"sort"
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
	orderabnormalcaseent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderabnormalcase"
	ordercommissionattributionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	ordercontainerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainer"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderlifecycleeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
	ordermilestoneent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordermilestone"
	orderreleasepodent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderreleasepod"
	ordershippingdocumentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordershippingdocument"
	partnerassignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerassignment"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	seahousebill "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbill "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlink "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seatransportexecution "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
	shippinglineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/shippingline"
)

func (r *orderRepo) Create(ctx context.Context, organizationID, actorID uuid.UUID, input *biz.Order, audit *biz.AuditEvent) (*biz.Order, error) {
	var createdID uuid.UUID
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if companyErr := ensureOperatingCompany(ctx, tx, organizationID); companyErr != nil {
			return companyErr
		}
		allocatedAt := time.Now().UTC()
		rule, sequence, err := allocateNumberInTx(ctx, tx, organizationID, biz.DocumentTypeOrder, allocatedAt)
		if err != nil {
			return err
		}
		number, err := biz.FormatAllocatedNumber(allocatedAt, rule, sequence, string(input.BusinessType))
		if err != nil {
			return err
		}
		if err := validateOrderReferences(ctx, tx, organizationID, input, nil); err != nil {
			return err
		}
		// 幂等键列为 NOT NULL；请求未提供时由服务端生成随机键填充（此时
		// 上层不会启用幂等查询，行为与无键创建一致）。
		storedIdempotencyKey := input.IdempotencyKey
		if storedIdempotencyKey == "" {
			storedIdempotencyKey = uuid.Must(uuid.NewV7()).String()
		}
		create := tx.Order.Create().
			SetOrganizationID(organizationID).
			SetOrderNo(number).
			SetIdempotencyKey(storedIdempotencyKey).
			SetCustomerID(input.CustomerID).
			SetCustomerReferenceNo(input.CustomerReferenceNo).
			SetBookingNo(input.BookingNo).
			SetInternalReferenceNo(input.InternalReferenceNo).
			SetShipperShortName(input.ShipperShortName).
			SetConsigneeShortName(input.ConsigneeShortName).
			SetNillableShippingLineID(input.ShippingLineID).
			SetNillableBookingAgentID(input.BookingAgentID).
			SetNillableForeignAgentID(input.ForeignAgentID).
			SetNillableShippingAgentID(input.ShippingAgentID).
			SetContractNo(input.ContractNo).
			SetCargoValue(input.CargoValue).
			SetInsurancePremium(input.InsurancePremium).
			SetUnNumber(input.UNNumber).
			SetHazardClass(input.HazardClass).
			SetFactoryName(input.FactoryName).
			SetCargoReadyAt(input.CargoReadyAt).
			SetDeclarationCutoffAt(input.DeclarationCutoffAt).
			SetReceivedAt(input.ReceivedAt).
			SetBusinessType(orderent.BusinessType(input.BusinessType)).
			SetTradeDirection(orderent.TradeDirection(input.TradeDirection)).
			SetNillableTradeTerm(orderTradeTermToEnt(input.TradeTerm)).
			SetPaymentTerm(orderent.PaymentTerm(input.PaymentTerm)).
			SetNillableShipmentType(orderShipmentTypeToEnt(input.ShipmentType)).
			SetNillableContainerOwnership(orderContainerOwnershipToEnt(input.ContainerOwnership)).
			SetNillableShipmentMode(orderShipmentModeToEnt(input.ShipmentMode)).
			SetFlowStatus(orderent.FlowStatusDRAFT).
			SetTerminationStatus(orderent.TerminationStatusACTIVE).
			SetClosureStatus(orderent.ClosureStatusOPEN).
			SetVersion(1).
			SetNillableOriginLocationID(input.OriginLocationID).
			SetNillableDestinationLocationID(input.DestinationLocationID).
			SetNillableDischargeLocationID(input.DischargeLocationID).
			SetNillableTransitLocationID(input.TransitLocationID).
			SetVesselVoyage(input.VesselVoyage).
			SetEtd(input.ETD).
			SetEta(input.ETA).
			SetSiCutoff(input.SICutoff).
			SetDocCutoff(input.DocCutoff).
			SetCustomsCutoff(input.CustomsCutoff).
			SetVgmCutoff(input.VGMCutoff).
			SetGoodsDescription(input.GoodsDescription).
			SetNillableTotalPackages(input.TotalPackages).
			SetNillableTotalGrossWeightKg(input.TotalGrossWeightKg).
			SetNillableTotalVolumeCbm(input.TotalVolumeCbm).
			SetTotalPackageUnit(input.TotalPackageUnit).
			SetSpecialRequirements(input.SpecialRequirements).
			SetOrderDate(input.OrderDate).
			SetNotes(input.Notes).
			SetBookingNotes(input.BookingNotes).
			SetAllocationNotes(input.AllocationNotes).
			SetOperationNotes(input.OperationNotes)
		create.SetNillableCargoCurrency(nonEmptyStringPointer(input.CargoCurrency))
		create.SetNillableInsuranceCurrency(nonEmptyStringPointer(input.InsuranceCurrency))
		created, err := create.Save(ctx)
		if err != nil {
			// 并发同键创建由 (organization_id, idempotency_key) 唯一索引兜底，
			// 映射为幂等冲突后由上层用例重查解析为重放或冲突。
			return mapEntConstraints(err,
				entConstraintMapping{name: "order_organization_id_order_no", domainErr: biz.ErrOrderNumberExists},
				entConstraintMapping{name: "order_organization_id_idempotency_key", domainErr: biz.ErrOrderIdempotencyConflict},
			)
		}
		createdID = created.ID
		if err := replaceOrderSelections(ctx, tx, created.ID, input.ServiceTypeIDs, input.CargoCategoryIDs); err != nil {
			return err
		}
		if err := syncOrderShippingDocuments(ctx, tx, organizationID, input.BusinessType, created.ID, input.ShippingDocuments); err != nil {
			return err
		}
		if err := syncOrderContainerRequests(ctx, tx, organizationID, created.ID, input.ContainerRequests); err != nil {
			return err
		}
		if err := syncOrderSeaMasterBillOnCreate(ctx, tx, organizationID, created, input); err != nil {
			return err
		}
		if err := syncOrderSeaDocumentOnCreate(ctx, tx, organizationID, created, input, audit); err != nil {
			return err
		}
		if _, err := tx.OrderLifecycleEvent.Create().SetOrderID(created.ID).SetDimension(orderlifecycleeventent.DimensionFLOW).SetToStatus("DRAFT").SetAction("create").SetOperatorID(actorID).Save(ctx); err != nil {
			return err
		}
		personnel := make([]*biz.OrderPersonnel, 0, len(input.PersonnelAssignments)+1)
		personnel = append(personnel, &biz.OrderPersonnel{UserID: actorID, OrganizationID: organizationID, Role: biz.OrderPersonnelRoleCreator})
		personnel = append(personnel, input.PersonnelAssignments...)
		if err := createOrderPersonnel(ctx, tx, organizationID, created.ID, created.OrderNo, personnel); err != nil {
			return err
		}
		if err := snapshotOrderCommissionAttributions(ctx, tx, organizationID, created.ID, input.CustomerID, created.CreatedAt); err != nil {
			return err
		}
		audit.Details["order.id"] = created.ID.String()
		audit.Details["order.no"] = number
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, createdID)
}

func snapshotOrderCommissionAttributions(ctx context.Context, tx *ent.Tx, organizationID, orderID, customerID uuid.UUID, attributedAt time.Time) error {
	assignments, err := tx.PartnerAssignment.Query().Where(
		partnerassignmentent.OrganizationIDEQ(organizationID),
		partnerassignmentent.PartnerIDEQ(customerID),
		partnerassignmentent.RoleIn(partnerassignmentent.RoleSALES, partnerassignmentent.RoleOPERATOR, partnerassignmentent.RoleCUSTOMER_SERVICE),
	).WithUser().Order(partnerassignmentent.ByRole(), partnerassignmentent.BySortOrder()).All(ctx)
	if err != nil {
		return err
	}
	// 提成相关岗位（销售/操作/客服）缺人时阻止开单：归属组织已收敛为客户档案
	// 所属组织，快照严格等值匹配必然命中已配置人员，缺配属于必须显式暴露的配置问题。
	coveredRoles := make(map[partnerassignmentent.Role]struct{}, len(assignments))
	for _, item := range assignments {
		coveredRoles[item.Role] = struct{}{}
	}
	missingRoles := make([]biz.PartnerAssignmentRole, 0, 3)
	for _, check := range []struct {
		role    biz.PartnerAssignmentRole
		entRole partnerassignmentent.Role
	}{
		{biz.PartnerAssignmentSales, partnerassignmentent.RoleSALES},
		{biz.PartnerAssignmentOperator, partnerassignmentent.RoleOPERATOR},
		{biz.PartnerAssignmentCustomerService, partnerassignmentent.RoleCUSTOMER_SERVICE},
	} {
		if _, ok := coveredRoles[check.entRole]; !ok {
			missingRoles = append(missingRoles, check.role)
		}
	}
	if len(missingRoles) > 0 {
		return biz.NewPartnerCommissionAssignmentMissing(missingRoles)
	}
	builders := make([]*ent.OrderCommissionAttributionCreate, 0, len(assignments))
	seenAttributions := make(map[string]struct{}, len(assignments))
	for _, item := range assignments {
		role := ordercommissionattributionent.PersonnelRole(item.Role)
		key := item.UserID.String() + ":" + string(role)
		if _, exists := seenAttributions[key]; exists {
			continue
		}
		employee, edgeErr := item.Edges.UserOrErr()
		if edgeErr != nil {
			return edgeErr
		}
		builders = append(builders, tx.OrderCommissionAttribution.Create().
			SetID(uuid.Must(uuid.NewV7())).SetOrganizationID(organizationID).SetOrderID(orderID).SetCustomerID(customerID).
			SetSourceAssignmentID(item.ID).SetEmployeeID(item.UserID).SetEmployeeName(employee.DisplayName).
			SetPersonnelRole(role).SetAttributedAt(attributedAt))
		seenAttributions[key] = struct{}{}
	}
	if len(builders) == 0 {
		return nil
	}
	_, err = tx.OrderCommissionAttribution.CreateBulk(builders...).Save(ctx)
	return err
}

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
		if validateErr := validateOrderReferences(ctx, tx, organizationID, input, existing.ShippingLineID); validateErr != nil {
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
			if _, deleteErr := tx.OrderCommissionAttribution.Delete().Where(ordercommissionattributionent.OrderIDEQ(id)).Exec(ctx); deleteErr != nil {
				return deleteErr
			}
			if snapshotErr := snapshotOrderCommissionAttributions(ctx, tx, organizationID, id, input.CustomerID, existing.CreatedAt); snapshotErr != nil {
				return snapshotErr
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

func (r *orderRepo) TransitionStatus(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, targetStatus biz.OrderFlowStatus, reason string, actorID uuid.UUID, event *biz.OrderStatusChangedEvent) (*biz.Order, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, queryErr := tx.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		if existing.Version != expectedVersion || biz.OrderFlowStatus(existing.FlowStatus) != event.FromStatus {
			return biz.ErrOrderStatusConflict
		}
		// 主流程推进属于业务写入：生命周期专属校验要求终止维度 ACTIVE、结案维度
		// OPEN；业务锁规则保持现状，锁定后不得推进主流程。
		if existing.TerminationStatus != orderent.TerminationStatusACTIVE || existing.ClosureStatus != orderent.ClosureStatusOPEN {
			return biz.ErrOrderStatusConflict
		}
		if existing.LockedAt != nil {
			return ensureOrderNotBusinessLocked(ctx, tx.User, existing)
		}
		if _, updateErr := existing.Update().SetFlowStatus(orderent.FlowStatus(targetStatus)).SetVersion(existing.Version + 1).Save(ctx); updateErr != nil {
			return updateErr
		}
		if _, eventErr := tx.OrderLifecycleEvent.Create().SetOrderID(id).SetDimension(orderlifecycleeventent.DimensionFLOW).SetFromStatus(string(event.FromStatus)).SetToStatus(string(targetStatus)).SetAction("transition").SetReason(reason).SetOperatorID(actorID).SetChangedAt(event.OccurredAt).Save(ctx); eventErr != nil {
			return eventErr
		}
		return writeAudit(ctx, tx.AuditLog, event.AuditEvent())
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}

func (r *orderRepo) TransitionTermination(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, target biz.OrderTerminationStatus, terminationType *biz.OrderTerminationType, reason string, actorID uuid.UUID, event *biz.OrderLifecycleChangedEvent) (*biz.Order, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, queryErr := tx.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		if existing.Version != expectedVersion || string(existing.TerminationStatus) != event.FromStatus {
			return biz.ErrOrderStatusConflict
		}
		// 终止维度是生命周期命令：不复用内容写门禁。专属校验只要求结案维度
		// 保持 OPEN；完成、取消与恢复不得因业务锁或历史终止状态被误封。
		if existing.ClosureStatus != orderent.ClosureStatusOPEN {
			return biz.ErrOrderTerminationInvalid
		}
		// ACTIVE → TERMINATING 属于发起业务变更，保持“锁定资料不可发起退关”
		// 的现行约束；其余流转路径不因业务锁被阻断。
		if existing.TerminationStatus == orderent.TerminationStatusACTIVE &&
			target == biz.OrderTerminationTerminating && existing.LockedAt != nil {
			return ensureOrderNotBusinessLocked(ctx, tx.User, existing)
		}
		update := existing.Update().SetTerminationStatus(orderent.TerminationStatus(target)).SetVersion(existing.Version + 1)
		if target == biz.OrderTerminationActive {
			update.ClearTerminationType().ClearTerminationReason().ClearTerminatedAt().ClearTerminatedBy()
		} else {
			update.SetTerminationType(orderent.TerminationType(*terminationType)).SetTerminationReason(reason)
			if target == biz.OrderTerminationTerminated {
				update.SetTerminatedAt(event.OccurredAt).SetTerminatedBy(actorID)
			} else {
				update.ClearTerminatedAt().ClearTerminatedBy()
			}
		}
		if _, updateErr := update.Save(ctx); updateErr != nil {
			return updateErr
		}
		// 最终进入 TERMINATED 时在同一事务结束该订单的活动 SE Link；任何一步
		// 失败（订单更新、Link 结束、生命周期事件、审计）整体回滚。
		if target == biz.OrderTerminationTerminated {
			if linkErr := endActiveSeaMasterBillLinksOnTermination(ctx, tx, existing, event.OccurredAt); linkErr != nil {
				return linkErr
			}
		}
		if _, eventErr := tx.OrderLifecycleEvent.Create().SetOrderID(id).SetDimension(orderlifecycleeventent.DimensionTERMINATION).SetFromStatus(event.FromStatus).SetToStatus(event.ToStatus).SetAction("transition").SetReason(reason).SetOperatorID(actorID).SetChangedAt(event.OccurredAt).Save(ctx); eventErr != nil {
			return eventErr
		}
		return writeAudit(ctx, tx.AuditLog, event.AuditEvent())
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}

// orderTerminationLinkEndedReason 是退关结束活动 Link 的固定中文原因。
const orderTerminationLinkEndedReason = "订单退关"

// endActiveSeaMasterBillLinksOnTermination 在订单最终流转到 TERMINATED 的同一
// 事务中结束其活动 SE Link。调用前必须已锁定订单行；活动 Link 按 ID 排序后
// FOR UPDATE，数量超过一条时返回结构冲突（fail-closed，不静默修复）。结束时
// 写入 UTC ended_at、固定中文 ended_reason 并递增 Link 版本；不修改 MBL 与
// TransportExecution 的内容版本。非 SE 订单没有 Link 操作。
func endActiveSeaMasterBillLinksOnTermination(ctx context.Context, tx *ent.Tx, order *ent.Order, occurredAt time.Time) error {
	if order.BusinessType != orderent.BusinessTypeSE {
		return nil
	}
	activeLinks, err := tx.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.OrganizationIDEQ(order.OrganizationID),
			seamasterbillorderlink.OrderIDEQ(order.ID),
			seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
		).
		Order(seamasterbillorderlink.ByID()).
		ForUpdate().
		All(ctx)
	if err != nil {
		return err
	}
	if len(activeLinks) == 0 {
		return nil
	}
	if len(activeLinks) > 1 {
		return biz.ErrSeaDocumentStructureConflict
	}
	link := activeLinks[0]
	_, err = link.Update().
		SetStatus(seamasterbillorderlink.StatusENDED).
		SetEndedAt(occurredAt.UTC()).
		SetEndedReason(orderTerminationLinkEndedReason).
		SetVersion(link.Version + 1).
		Save(ctx)
	return err
}

func (r *orderRepo) ClosureReadiness(ctx context.Context, organizationID, id uuid.UUID) (*biz.OrderClosureReadiness, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	hasActiveException, err := client.OrderAbnormalCase.Query().Where(orderabnormalcaseent.OrderIDEQ(id), orderabnormalcaseent.StatusEQ(orderabnormalcaseent.StatusACTIVE)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	hasUnbilledFees, err := client.OrderFee.Query().Where(orderfeeent.OrderIDEQ(id), orderfeeent.StatusNotIn(orderfeeent.StatusBILLED, orderfeeent.StatusCANCELLED)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	return &biz.OrderClosureReadiness{FlowStatus: biz.OrderFlowStatus(item.FlowStatus), TerminationStatus: biz.OrderTerminationStatus(item.TerminationStatus), ClosureStatus: biz.OrderClosureStatus(item.ClosureStatus), HasActiveException: hasActiveException, HasUnbilledOrderFees: hasUnbilledFees}, nil
}

func (r *orderRepo) TransitionClosure(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, target biz.OrderClosureStatus, reason string, actorID uuid.UUID, event *biz.OrderLifecycleChangedEvent) (*biz.Order, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, queryErr := tx.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		// 结案与反结案是生命周期命令：不复用内容写门禁，也不校验业务锁，
		// 允许已业务锁定的 DOCUMENT_RELEASED 订单结案；版本与来源状态仍在
		// 事务内权威校验，结案 readiness 在下方重验。
		if existing.Version != expectedVersion || string(existing.ClosureStatus) != event.FromStatus {
			return biz.ErrOrderStatusConflict
		}
		if target == biz.OrderClosureClosed {
			flowFinished := biz.OrderFlowStatus(existing.FlowStatus) == biz.OrderFlowDocumentReleased
			terminated := biz.OrderTerminationStatus(existing.TerminationStatus) == biz.OrderTerminationTerminated
			if !flowFinished && !terminated {
				return biz.ErrOrderClosureBlocked
			}
			hasActiveException, readinessErr := tx.OrderAbnormalCase.Query().Where(orderabnormalcaseent.OrderIDEQ(id), orderabnormalcaseent.StatusEQ(orderabnormalcaseent.StatusACTIVE)).Exist(ctx)
			if readinessErr != nil {
				return readinessErr
			}
			hasUnbilledFees, readinessErr := tx.OrderFee.Query().Where(orderfeeent.OrderIDEQ(id), orderfeeent.StatusNotIn(orderfeeent.StatusBILLED, orderfeeent.StatusCANCELLED)).Exist(ctx)
			if readinessErr != nil {
				return readinessErr
			}
			if hasActiveException || hasUnbilledFees {
				return biz.ErrOrderClosureBlocked
			}
		}
		update := existing.Update().SetClosureStatus(orderent.ClosureStatus(target)).SetVersion(existing.Version + 1)
		if target == biz.OrderClosureClosed {
			update.SetClosureReason(reason).SetClosedAt(event.OccurredAt).SetClosedBy(actorID)
		} else {
			update.ClearClosureReason().ClearClosedAt().ClearClosedBy()
		}
		if _, updateErr := update.Save(ctx); updateErr != nil {
			return updateErr
		}
		if _, eventErr := tx.OrderLifecycleEvent.Create().SetOrderID(id).SetDimension(orderlifecycleeventent.DimensionCLOSURE).SetFromStatus(event.FromStatus).SetToStatus(event.ToStatus).SetAction("transition").SetReason(reason).SetOperatorID(actorID).SetChangedAt(event.OccurredAt).Save(ctx); eventErr != nil {
			return eventErr
		}
		return writeAudit(ctx, tx.AuditLog, event.AuditEvent())
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}

func parseOptionalTime(s string) *time.Time {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil
	}
	return &t
}

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

func syncOrderSeaDocumentOnCreate(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, order *ent.Order, input *biz.Order, audit *biz.AuditEvent) error {
	if input.BusinessType != biz.OrderBusinessSE || input.SeaDocumentInput == nil {
		return nil
	}
	docInput := input.SeaDocumentInput

	link, err := tx.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.OrganizationIDEQ(organizationID),
			seamasterbillorderlink.OrderIDEQ(order.ID),
			seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
		).
		WithMasterBill().
		Only(ctx)
	if err != nil {
		return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}

	mbl := link.Edges.MasterBill
	if mbl == nil {
		return biz.ErrSeaMasterBillNotFound
	}

	if docInput.MasterBillContent != nil {
		updater := mbl.Update()
		setSeaMasterBillContent(updater, docInput.MasterBillContent)
		if _, err := updater.Save(ctx); err != nil {
			return err
		}
	}

	hasHBL := docInput.HouseBill != nil
	targetStructure := seamasterbillorderlink.DocumentStructureHOUSE

	if docInput.DocumentStructure != nil {
		switch *docInput.DocumentStructure {
		case biz.SeaDocumentStructureDirect:
			if hasHBL {
				return biz.ErrSeaDocumentStructureInvalid
			}
			targetStructure = seamasterbillorderlink.DocumentStructureDIRECT
		case biz.SeaDocumentStructureHouse:
			if !hasHBL {
				return errors.BadRequest("SEA_DOCUMENT_STRUCTURE_INVALID", "HOUSE 单证结构必须包含分单")
			}
			targetStructure = seamasterbillorderlink.DocumentStructureHOUSE
		}
	}

	if targetStructure != link.DocumentStructure {
		if _, err := link.Update().SetDocumentStructure(targetStructure).Save(ctx); err != nil {
			return err
		}
	}

	if hasHBL {
		hbInput := docInput.HouseBill
		normalized, err := biz.NormalizeSeaHouseNo(hbInput.HouseNo)
		if err != nil {
			return err
		}
		// 批次内排重预查（含作废行，一号一案）：先给出含冲突分单号的友好报错，
		// 唯一索引仅作并发兜底。跨批次重号不受限制。
		duplicateExists, err := tx.SeaHouseBill.Query().
			Where(
				seahousebill.MasterBillIDEQ(mbl.ID),
				seahousebill.NormalizedHouseNoEQ(normalized),
			).
			Exist(ctx)
		if err != nil {
			return err
		}
		if duplicateExists {
			return biz.SeaHouseBillBatchNoDuplicateError([]string{normalized})
		}
		if hbInput.Content != nil {
			if _, err := biz.ValidateSeaBillContent(hbInput.Content); err != nil {
				return err
			}
		}
		issuerOrgID, issuerPartnerID, err := validateSeaHouseBillIssuer(ctx, tx.Client(), organizationID, order.OrganizationID, order.CustomerID, hbInput)
		if err != nil {
			return err
		}
		builder := tx.SeaHouseBill.Create().
			SetID(uuid.Must(uuid.NewV7())).
			SetOrganizationID(organizationID).
			SetOrderID(order.ID).
			SetMasterBillID(mbl.ID).
			SetHouseNo(hbInput.HouseNo).
			SetNormalizedHouseNo(normalized).
			SetIssuerSource(seahousebill.IssuerSource(hbInput.IssuerSource)).
			SetStatus(seahousebill.StatusDRAFT).
			SetVersion(1)
		if issuerOrgID != nil {
			builder.SetIssuerOrganizationID(*issuerOrgID)
		}
		if issuerPartnerID != nil {
			builder.SetIssuerPartnerID(*issuerPartnerID)
		}
		if hbInput.Note != nil {
			builder.SetNote(*hbInput.Note)
		}
		setSeaHouseBillContentCreate(builder, hbInput.Content)
		if _, err := builder.Save(ctx); err != nil {
			if ent.IsConstraintError(err) {
				return biz.ErrSeaHouseBillBatchNoDuplicate
			}
			return err
		}
	}

	if audit.Details == nil {
		audit.Details = make(map[string]string)
	}
	audit.Details["sea_document.initial_structure"] = string(targetStructure)
	initialCount := 0
	if hasHBL {
		initialCount = 1
	}
	audit.Details["sea_house_bills.initial_count"] = fmt.Sprintf("%d", initialCount)

	return nil
}

func syncOrderSeaDocumentOnUpdate(
	ctx context.Context,
	tx *ent.Tx,
	organizationID uuid.UUID,
	order *ent.Order,
	input *biz.Order,
	audit *biz.AuditEvent,
	lockContext *seaMasterBillUpdateLockContext,
) error {
	if input.BusinessType != biz.OrderBusinessSE || input.SeaDocumentInput == nil {
		return nil
	}
	docInput := input.SeaDocumentInput
	if docInput.HouseBill != nil {
		return errors.BadRequest("SEA_DOCUMENT_INVALID_ARGUMENT", "订单整单更新禁止直接提交分单变更，请使用专用单证命令")
	}

	// 1. Order 已在 UpdateDraft 中加锁 ForUpdate
	// 2. 无锁查询定位 active link ID 与 master_bill_id
	activeLinkQuery, err := tx.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.OrganizationIDEQ(organizationID),
			seamasterbillorderlink.OrderIDEQ(order.ID),
			seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
		).
		Only(ctx)
	if err != nil {
		return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}

	// 3. 锁 MBL (ForUpdate)
	mbl, err := tx.SeaMasterBill.Query().
		Where(
			seamasterbill.IDEQ(activeLinkQuery.MasterBillID),
			seamasterbill.OrganizationIDEQ(organizationID),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}

	// 4. 按 ID 锁 Active Link (ForUpdate) 并重验
	link, err := tx.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.IDEQ(activeLinkQuery.ID),
			seamasterbillorderlink.OrganizationIDEQ(organizationID),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}
	if link.Status != seamasterbillorderlink.StatusACTIVE || link.MasterBillID != mbl.ID {
		return biz.ErrSeaDocumentStructureConflict
	}

	// 5. 更新 MBL 内容（严格校验 expected_mbl_version）
	if docInput.MasterBillContent != nil {
		if _, err := revalidateSeaMasterBillUpdateMemberSet(
			ctx, tx, organizationID, link.ID, mbl.ID, lockContext,
		); err != nil {
			return err
		}
		if err := ensureLockedMembersAllowSharedMasterBillUpdate(lockContext); err != nil {
			return err
		}
		if docInput.ExpectedMblVersion == nil {
			return errors.BadRequest("SEA_DOCUMENT_INVALID_ARGUMENT", "修改主单内容必须提供 expected_mbl_version")
		}
		if mbl.Version != *docInput.ExpectedMblVersion {
			return biz.ErrSeaMasterBillConflict
		}
		if _, err := biz.ValidateSeaBillContent(docInput.MasterBillContent); err != nil {
			return err
		}
		updater := mbl.Update().SetVersion(mbl.Version + 1)
		setSeaMasterBillContent(updater, docInput.MasterBillContent)
		if _, err := updater.Save(ctx); err != nil {
			return err
		}
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["sea_master_bill.content_updated"] = "true"
		audit.Details["sea_master_bill.old_version"] = fmt.Sprintf("%d", mbl.Version)
		audit.Details["sea_master_bill.new_version"] = fmt.Sprintf("%d", mbl.Version+1)
	}

	// 6. 更新单证结构（严格校验 expected_link_version 并复用状态门禁）
	if docInput.DocumentStructure != nil {
		if docInput.ExpectedLinkVersion == nil {
			return errors.BadRequest("SEA_DOCUMENT_INVALID_ARGUMENT", "修改单证结构必须提供 expected_link_version")
		}
		if link.Version != *docInput.ExpectedLinkVersion {
			return biz.ErrSeaDocumentStructureConflict
		}
		targetStructure := *docInput.DocumentStructure
		if targetStructure == biz.SeaDocumentStructureDirect {
			hbCount, err := tx.SeaHouseBill.Query().
				Where(
					seahousebill.OrganizationIDEQ(organizationID),
					seahousebill.OrderIDEQ(order.ID),
					seahousebill.MasterBillIDEQ(mbl.ID),
				).Count(ctx)
			if err != nil {
				return err
			}
			if hbCount > 0 {
				return biz.ErrSeaDocumentStructureInvalid
			}
			if link.DocumentStructure != seamasterbillorderlink.DocumentStructureDIRECT {
				if _, err := link.Update().SetDocumentStructure(seamasterbillorderlink.DocumentStructureDIRECT).SetVersion(link.Version + 1).Save(ctx); err != nil {
					return err
				}
			}
		} else if targetStructure == biz.SeaDocumentStructureHouse {
			return errors.BadRequest("SEA_DOCUMENT_INVALID_ARGUMENT", "切换单证结构请使用专用模式切换命令")
		}

		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["sea_document.structure_updated"] = "true"
		audit.Details["sea_document.old_structure"] = string(link.DocumentStructure)
		audit.Details["sea_document.new_structure"] = string(targetStructure)
		audit.Details["sea_document.old_link_version"] = fmt.Sprintf("%d", link.Version)
		audit.Details["sea_document.new_link_version"] = fmt.Sprintf("%d", link.Version+1)
	}

	return nil
}
