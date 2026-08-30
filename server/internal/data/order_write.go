package data

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderabnormalcaseent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderabnormalcase"
	ordercommissionattributionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderlifecycleeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
	partnerassignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerassignment"
)

func (r *orderRepo) Create(ctx context.Context, organizationID, actorID uuid.UUID, input *biz.Order, audit *biz.AuditEvent) (*biz.Order, error) {
	var createdID uuid.UUID
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		allocatedAt := time.Now().UTC()
		rule, sequence, err := allocateNumberInTx(ctx, tx, organizationID, biz.DocumentTypeOrder, allocatedAt)
		if err != nil {
			return err
		}
		number, err := biz.FormatAllocatedNumber(allocatedAt, rule, sequence, string(input.BusinessType))
		if err != nil {
			return err
		}
		if err := validateOrderReferences(ctx, tx, organizationID, input); err != nil {
			return err
		}
		create := tx.Order.Create().
			SetOrganizationID(organizationID).
			SetOrderNo(number).
			SetCustomerID(input.CustomerID).
			SetCustomerReferenceNo(input.CustomerReferenceNo).
			SetInternalReferenceNo(input.InternalReferenceNo).
			SetShipperShortName(input.ShipperShortName).
			SetConsigneeShortName(input.ConsigneeShortName).
			SetNillableCarrierID(input.CarrierID).
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
			SetLoadingTerms(input.LoadingTerms).
			SetDeclarationCutoffAt(input.DeclarationCutoffAt).
			SetReceivedAt(input.ReceivedAt).
			SetBusinessType(orderent.BusinessType(input.BusinessType)).
			SetTradeDirection(orderent.TradeDirection(input.TradeDirection)).
			SetTradeTerm(orderent.TradeTerm(input.TradeTerm)).
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
			return mapEntConstraint(err, "order_organization_id_order_no", biz.ErrOrderNumberExists)
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
	if len(assignments) == 0 {
		return nil
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

func (r *orderRepo) UpdateDraft(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, input *biz.Order, audit *biz.AuditEvent) (*biz.Order, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	existing, err := tx.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderNotFound
		}
		return nil, err
	}
	if existing.Version != expectedVersion {
		_ = tx.Rollback()
		return nil, biz.ErrOrderStatusConflict
	}
	if existing.FlowStatus != orderent.FlowStatusDRAFT || existing.TerminationStatus != orderent.TerminationStatusACTIVE || existing.ClosureStatus != orderent.ClosureStatusOPEN {
		_ = tx.Rollback()
		return nil, biz.ErrOrderStatusConflict
	}
	if existing.BusinessType != orderent.BusinessType(input.BusinessType) {
		_ = tx.Rollback()
		return nil, biz.ErrOrderBusinessUnsupported
	}
	if err := validateOrderReferences(ctx, tx, organizationID, input); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	update := existing.Update().
		SetVersion(existing.Version + 1).
		SetCustomerID(input.CustomerID).
		SetCustomerReferenceNo(input.CustomerReferenceNo).
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
		SetLoadingTerms(input.LoadingTerms).
		SetDeclarationCutoffAt(input.DeclarationCutoffAt).
		SetReceivedAt(input.ReceivedAt).
		SetTradeDirection(orderent.TradeDirection(input.TradeDirection)).
		SetTradeTerm(orderent.TradeTerm(input.TradeTerm)).
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
	if _, err := update.Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, mapEntConstraint(err, "order_organization_id_order_no", biz.ErrOrderNumberExists)
	}
	if err := replaceOrderSelections(ctx, tx, id, input.ServiceTypeIDs, input.CargoCategoryIDs); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := syncOrderShippingDocuments(ctx, tx, organizationID, input.BusinessType, id, input.ShippingDocuments); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := syncOrderContainerRequests(ctx, tx, organizationID, id, input.ContainerRequests); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if existing.CustomerID != input.CustomerID {
		if _, err := tx.OrderCommissionAttribution.Delete().Where(ordercommissionattributionent.OrderIDEQ(id)).Exec(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if err := snapshotOrderCommissionAttributions(ctx, tx, organizationID, id, input.CustomerID, existing.CreatedAt); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	audit.Details["order.no"] = existing.OrderNo
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}

func (r *orderRepo) TransitionStatus(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, targetStatus biz.OrderFlowStatus, reason string, actorID uuid.UUID, event *biz.OrderStatusChangedEvent) (*biz.Order, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	existing, err := tx.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderNotFound
		}
		return nil, err
	}
	if existing.Version != expectedVersion || biz.OrderFlowStatus(existing.FlowStatus) != event.FromStatus {
		_ = tx.Rollback()
		return nil, biz.ErrOrderStatusConflict
	}
	if _, err := existing.Update().SetFlowStatus(orderent.FlowStatus(targetStatus)).SetVersion(existing.Version + 1).Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.OrderLifecycleEvent.Create().SetOrderID(id).SetDimension(orderlifecycleeventent.DimensionFLOW).SetFromStatus(string(event.FromStatus)).SetToStatus(string(targetStatus)).SetAction("transition").SetReason(reason).SetOperatorID(actorID).SetChangedAt(event.OccurredAt).Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := writeAudit(ctx, tx.AuditLog, event.AuditEvent()); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}

func (r *orderRepo) TransitionTermination(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, target biz.OrderTerminationStatus, terminationType *biz.OrderTerminationType, reason string, actorID uuid.UUID, event *biz.OrderLifecycleChangedEvent) (*biz.Order, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	existing, err := tx.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderNotFound
		}
		return nil, err
	}
	if existing.Version != expectedVersion || string(existing.TerminationStatus) != event.FromStatus {
		_ = tx.Rollback()
		return nil, biz.ErrOrderStatusConflict
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
	if _, err := update.Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.OrderLifecycleEvent.Create().SetOrderID(id).SetDimension(orderlifecycleeventent.DimensionTERMINATION).SetFromStatus(event.FromStatus).SetToStatus(event.ToStatus).SetAction("transition").SetReason(reason).SetOperatorID(actorID).SetChangedAt(event.OccurredAt).Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := writeAudit(ctx, tx.AuditLog, event.AuditEvent()); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}

func (r *orderRepo) ClosureReadiness(ctx context.Context, organizationID, id uuid.UUID) (*biz.OrderClosureReadiness, error) {
	item, err := r.data.db.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderNotFound
		}
		return nil, err
	}
	hasActiveException, err := r.data.db.OrderAbnormalCase.Query().Where(orderabnormalcaseent.OrderIDEQ(id), orderabnormalcaseent.StatusEQ(orderabnormalcaseent.StatusACTIVE)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	hasUnbilledFees, err := r.data.db.OrderFee.Query().Where(orderfeeent.OrderIDEQ(id), orderfeeent.StatusNotIn(orderfeeent.StatusBILLED, orderfeeent.StatusCANCELLED)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	return &biz.OrderClosureReadiness{FlowStatus: biz.OrderFlowStatus(item.FlowStatus), TerminationStatus: biz.OrderTerminationStatus(item.TerminationStatus), ClosureStatus: biz.OrderClosureStatus(item.ClosureStatus), HasActiveException: hasActiveException, HasUnbilledOrderFees: hasUnbilledFees}, nil
}

func (r *orderRepo) TransitionClosure(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, target biz.OrderClosureStatus, reason string, actorID uuid.UUID, event *biz.OrderLifecycleChangedEvent) (*biz.Order, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	existing, err := tx.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderNotFound
		}
		return nil, err
	}
	if existing.Version != expectedVersion || string(existing.ClosureStatus) != event.FromStatus {
		_ = tx.Rollback()
		return nil, biz.ErrOrderStatusConflict
	}
	if target == biz.OrderClosureClosed {
		flowFinished := biz.OrderFlowStatus(existing.FlowStatus) == biz.OrderFlowDocumentReleased
		terminated := biz.OrderTerminationStatus(existing.TerminationStatus) == biz.OrderTerminationTerminated
		if !flowFinished && !terminated {
			_ = tx.Rollback()
			return nil, biz.ErrOrderClosureBlocked
		}
		hasActiveException, err := tx.OrderAbnormalCase.Query().Where(orderabnormalcaseent.OrderIDEQ(id), orderabnormalcaseent.StatusEQ(orderabnormalcaseent.StatusACTIVE)).Exist(ctx)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		hasUnbilledFees, err := tx.OrderFee.Query().Where(orderfeeent.OrderIDEQ(id), orderfeeent.StatusNotIn(orderfeeent.StatusBILLED, orderfeeent.StatusCANCELLED)).Exist(ctx)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if hasActiveException || hasUnbilledFees {
			_ = tx.Rollback()
			return nil, biz.ErrOrderClosureBlocked
		}
	}
	update := existing.Update().SetClosureStatus(orderent.ClosureStatus(target)).SetVersion(existing.Version + 1)
	if target == biz.OrderClosureClosed {
		update.SetClosureReason(reason).SetClosedAt(event.OccurredAt).SetClosedBy(actorID)
	} else {
		update.ClearClosureReason().ClearClosedAt().ClearClosedBy()
	}
	if _, err := update.Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.OrderLifecycleEvent.Create().SetOrderID(id).SetDimension(orderlifecycleeventent.DimensionCLOSURE).SetFromStatus(event.FromStatus).SetToStatus(event.ToStatus).SetAction("transition").SetReason(reason).SetOperatorID(actorID).SetChangedAt(event.OccurredAt).Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := writeAudit(ctx, tx.AuditLog, event.AuditEvent()); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}
