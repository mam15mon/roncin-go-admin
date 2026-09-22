package data

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordercommissionattributionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	orderlifecycleeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
	orderpersonnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
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

// snapshotOrderCommissionAttributions 以同事务内已写入的订单人员（销售/操作/
// 客服三岗）为唯一真相生成提成归属快照：source_assignment_id 记录订单人员行 ID，
// customer_id 写入订单客户冗余列。三岗齐全由 biz 层创建门禁保证，本函数不再
// 承担缺配校验；同岗同人由 normalizeOrder 去重保证，此处仅保留 userID:role
// 防御性去重。
func snapshotOrderCommissionAttributions(ctx context.Context, tx *ent.Tx, organizationID, orderID, customerID uuid.UUID, attributedAt time.Time) error {
	personnel, err := tx.OrderPersonnel.Query().Where(
		orderpersonnelent.OrderIDEQ(orderID),
		orderpersonnelent.OrganizationIDEQ(organizationID),
		orderpersonnelent.RoleIn(orderpersonnelent.RoleSALES, orderpersonnelent.RoleOPERATOR, orderpersonnelent.RoleCUSTOMER_SERVICE),
	).WithUser().All(ctx)
	if err != nil {
		return err
	}
	builders := make([]*ent.OrderCommissionAttributionCreate, 0, len(personnel))
	seenAttributions := make(map[string]struct{}, len(personnel))
	for _, item := range personnel {
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
