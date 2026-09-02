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
	orderabnormalcaseent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderabnormalcase"
	ordercommissionattributionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	ordercontainerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainer"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderlifecycleeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
	ordermilestoneent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordermilestone"
	orderreleasepodent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderreleasepod"
	ordershippingdocumentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordershippingdocument"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partnerassignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerassignment"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	seamasterbill "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlink "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seatransportexecution "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
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
		if err := syncOrderSeaMasterBillOnCreate(ctx, tx, organizationID, created, input); err != nil {
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
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, queryErr := tx.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
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
		if validateErr := validateOrderReferences(ctx, tx, organizationID, input); validateErr != nil {
			return validateErr
		}
		// 共享 MBL 门禁必须在订单/HBL 等下游数据发生任何写入前完成，避免通过同一请求
		// 删除既有单证后绕过“无下游事实”校验。后续任一步失败仍由同一事务整体回滚。
		if syncErr := syncOrderSeaMasterBillOnUpdate(ctx, tx, organizationID, existing, input, audit); syncErr != nil {
			return syncErr
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
			if _, deleteErr := tx.OrderCommissionAttribution.Delete().Where(ordercommissionattributionent.OrderIDEQ(id)).Exec(ctx); deleteErr != nil {
				return deleteErr
			}
			if snapshotErr := snapshotOrderCommissionAttributions(ctx, tx, organizationID, id, input.CustomerID, existing.CreatedAt); snapshotErr != nil {
				return snapshotErr
			}
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
	if input.BusinessType != biz.OrderBusinessSE || input.SeaMasterBillInput == nil {
		return nil
	}
	mblInput := input.SeaMasterBillInput
	if err := validateSeaMasterBillIssuer(ctx, tx, organizationID, mblInput.IssuerPartnerID); err != nil {
		return err
	}

	vesselName, voyageNo := biz.SplitVesselVoyage(input.VesselVoyage)
	orderVoyage := &biz.SeaTransportExecution{
		VesselName: vesselName,
		VoyageNo:   voyageNo,
		ETD:        parseOptionalTime(input.ETD),
		ETA:        parseOptionalTime(input.ETA),
	}
	if input.CarrierID != nil {
		orderVoyage.CarrierID = *input.CarrierID
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
		if targetMBL.IssuerPartnerID != mblInput.IssuerPartnerID || targetMBL.NormalizedMasterNo != mblInput.MasterNo {
			return biz.ErrSeaMasterBillStatusConflict
		}

		targetTE, err := tx.SeaTransportExecution.Query().
			Where(seatransportexecution.IDEQ(targetMBL.TransportExecutionID), seatransportexecution.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
		}

		candidateVoyage := &biz.SeaTransportExecution{
			ID:                targetTE.ID,
			TransitLocationID: targetTE.TransitLocationID,
			VesselName:        targetTE.VesselName,
			VoyageNo:          targetTE.VoyageNo,
			ETD:               targetTE.Etd,
			ETA:               targetTE.Eta,
		}
		if targetTE.CarrierID != nil {
			candidateVoyage.CarrierID = *targetTE.CarrierID
		}
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
			SetOrderID(order.ID).
			SetStatus(seamasterbillorderlink.StatusACTIVE).
			SetStartedAt(time.Now().UTC()).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			return mapEntConstraint(err, "idx_sea_mbl_order_links_active_order", biz.ErrOrderStatusConflict)
		}
		return nil
	}

	exists, err := tx.SeaMasterBill.Query().
		Where(
			seamasterbill.OrganizationIDEQ(organizationID),
			seamasterbill.IssuerPartnerIDEQ(mblInput.IssuerPartnerID),
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
		SetVesselName(vesselName).
		SetVoyageNo(voyageNo).
		SetVersion(1)
	if input.CarrierID != nil {
		teBuilder.SetCarrierID(*input.CarrierID)
	}
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
		SetIssuerPartnerID(mblInput.IssuerPartnerID).
		SetTransportExecutionID(te.ID).
		SetMasterNo(mblInput.MasterNo).
		SetNormalizedMasterNo(mblInput.MasterNo).
		SetStatus(seamasterbill.StatusDRAFT).
		SetVersion(1)

	mbl, err := mblBuilder.Save(ctx)
	if err != nil {
		return mapEntConstraint(err, "seamasterbill_organization_id_issuer_partner_id_normalized_master_no", biz.ErrSeaMasterBillConfirmationRequired)
	}

	_, err = tx.SeaMasterBillOrderLink.Create().
		SetID(uuid.Must(uuid.NewV7())).
		SetOrganizationID(organizationID).
		SetMasterBillID(mbl.ID).
		SetOrderID(order.ID).
		SetStatus(seamasterbillorderlink.StatusACTIVE).
		SetStartedAt(time.Now().UTC()).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		return mapEntConstraint(err, "idx_sea_mbl_order_links_active_order", biz.ErrOrderStatusConflict)
	}

	return nil
}

func syncOrderSeaMasterBillOnUpdate(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, order *ent.Order, input *biz.Order, audit *biz.AuditEvent) error {
	if input.BusinessType != biz.OrderBusinessSE || input.SeaMasterBillInput == nil {
		return nil
	}
	orderID := order.ID
	mblInput := input.SeaMasterBillInput
	if err := validateSeaMasterBillIssuer(ctx, tx, organizationID, mblInput.IssuerPartnerID); err != nil {
		return err
	}

	link, err := tx.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.OrganizationIDEQ(organizationID),
			seamasterbillorderlink.OrderIDEQ(orderID),
			seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
		).
		ForUpdate().
		WithMasterBill(func(q *ent.SeaMasterBillQuery) {
			q.Where(seamasterbill.OrganizationIDEQ(organizationID)).
				ForUpdate().
				WithTransportExecution(func(tq *ent.SeaTransportExecutionQuery) {
					tq.Where(seatransportexecution.OrganizationIDEQ(organizationID)).ForUpdate()
				})
		}).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return syncOrderSeaMasterBillOnCreate(ctx, tx, organizationID, order, input)
		}
		return err
	}

	currentMBL := link.Edges.MasterBill
	if currentMBL == nil {
		return biz.ErrSeaMasterBillNotFound
	}
	currentTE := currentMBL.Edges.TransportExecution

	activeCount, err := tx.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.OrganizationIDEQ(organizationID),
			seamasterbillorderlink.MasterBillIDEQ(currentMBL.ID),
			seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
		).
		Count(ctx)
	if err != nil {
		return err
	}

	vesselName, voyageNo := biz.SplitVesselVoyage(input.VesselVoyage)

	identityChanged := (currentMBL.IssuerPartnerID != mblInput.IssuerPartnerID || currentMBL.NormalizedMasterNo != mblInput.MasterNo)
	voyageChanged := seaTransportExecutionDiffersFromOrder(currentTE, input)

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
				seamasterbill.IssuerPartnerIDEQ(mblInput.IssuerPartnerID),
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
		audit.Details["sea_master_bill.old_issuer_partner_id"] = currentMBL.IssuerPartnerID.String()
		audit.Details["sea_master_bill.new_issuer_partner_id"] = mblInput.IssuerPartnerID.String()
		audit.Details["sea_master_bill.correction_reason"] = strings.TrimSpace(mblInput.CorrectionReason)

		if _, err := currentMBL.Update().
			SetIssuerPartnerID(mblInput.IssuerPartnerID).
			SetMasterNo(mblInput.MasterNo).
			SetNormalizedMasterNo(mblInput.MasterNo).
			SetVersion(currentMBL.Version + 1).
			Save(ctx); err != nil {
			return mapEntConstraint(err, "seamasterbill_organization_id_issuer_partner_id_normalized_master_no", biz.ErrSeaMasterBillConfirmationRequired)
		}

		if currentTE != nil && voyageChanged {
			teUpdate := currentTE.Update().
				SetVesselName(vesselName).
				SetVoyageNo(voyageNo).
				SetVersion(currentTE.Version + 1)
			if input.CarrierID != nil {
				teUpdate.SetCarrierID(*input.CarrierID)
			} else {
				teUpdate.ClearCarrierID()
			}
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
				if input.CarrierID != nil {
					orderVoyage.CarrierID = *input.CarrierID
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
					if input.CarrierID != nil {
						orderVoyage.CarrierID = *input.CarrierID
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
					if input.CarrierID != nil {
						teUpdate.SetCarrierID(*input.CarrierID)
					} else {
						teUpdate.ClearCarrierID()
					}
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

func validateSeaMasterBillIssuer(ctx context.Context, tx *ent.Tx, organizationID, issuerPartnerID uuid.UUID) error {
	if issuerPartnerID == uuid.Nil {
		return biz.ErrSeaMasterBillInvalidArgument
	}
	exists, err := tx.PartnerRole.Query().Where(
		partnerroleent.PartnerIDEQ(issuerPartnerID),
		partnerroleent.RoleTypeIn(partnerroleent.RoleTypeSupplier, partnerroleent.RoleTypeCarrier),
		partnerroleent.EnabledEQ(true),
		partnerroleent.HasPartnerWith(partnerent.OrganizationIDEQ(organizationID), partnerent.EnabledEQ(true)),
	).Exist(ctx)
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
	return !optionalUUIDEquals(current.CarrierID, input.CarrierID) ||
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
