package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type SeaOrderChangeService struct {
	v1.UnimplementedSeaOrderChangeServiceServer
	usecase *biz.SeaOrderChangeUsecase
}

func NewSeaOrderChangeService(usecase *biz.SeaOrderChangeUsecase) *SeaOrderChangeService {
	return &SeaOrderChangeService{usecase: usecase}
}

func (s *SeaOrderChangeService) GetSeaOrderChangeActions(ctx context.Context, request *v1.GetSeaOrderChangeActionsRequest) (*v1.GetSeaOrderChangeActionsResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrSeaOrderSplitInvalidArgument
	}

	actions, err := s.usecase.GetChangeActions(ctx, principal.Organization.ID, orderID)
	if err != nil {
		return nil, err
	}

	return ok(ctx, &v1.GetSeaOrderChangeActionsResponse{
		Data: &v1.SeaOrderChangeActionsData{
			CanSplit:               actions.CanSplit,
			CanReassign:            actions.CanReassign,
			SplitBlockedReasons:    actions.SplitBlockedReasons,
			ReassignBlockedReasons: actions.ReassignBlockedReasons,
		},
	}), nil
}

func (s *SeaOrderChangeService) GetSeaOrderSplitContext(ctx context.Context, request *v1.GetSeaOrderSplitContextRequest) (*v1.GetSeaOrderSplitContextResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrSeaOrderSplitInvalidArgument
	}

	splitCtx, err := s.usecase.GetSplitContext(ctx, principal.Organization.ID, orderID)
	if err != nil {
		return nil, err
	}

	hbls := make([]*v1.SeaOrderSplitHouseBillItem, 0, len(splitCtx.HouseBills))
	for _, h := range splitCtx.HouseBills {
		hbls = append(hbls, &v1.SeaOrderSplitHouseBillItem{
			Id:      h.ID.String(),
			HouseNo: h.HouseNo,
			Status:  h.Status,
			Version: h.Version,
		})
	}

	cargoItems := make([]*v1.SeaOrderSplitCargoItem, 0, len(splitCtx.CargoItems))
	for _, ci := range splitCtx.CargoItems {
		cargoItems = append(cargoItems, &v1.SeaOrderSplitCargoItem{
			Id:            ci.ID.String(),
			CargoName:     ci.CargoName,
			PackageCount:  ci.PackageCount,
			GrossWeightKg: biz.FormatDecimal3(ci.GrossWeightKg),
			VolumeCbm:     biz.FormatDecimal6(ci.VolumeCbm),
			Version:       ci.Version,
		})
	}

	containers := make([]*v1.SeaOrderSplitContainerItem, 0, len(splitCtx.Containers))
	for _, c := range splitCtx.Containers {
		containers = append(containers, &v1.SeaOrderSplitContainerItem{
			Id:                c.ID.String(),
			ContainerNo:       c.ContainerNo,
			ContainerSpecId:   c.ContainerSpecID.String(),
			ContainerSpecName: c.ContainerSpecName,
			PackageCount:      c.PackageCount,
			GrossWeightKg:     biz.FormatDecimal3(c.GrossWeightKg),
			VolumeCbm:         biz.FormatDecimal6(c.VolumeCbm),
			Version:           c.Version,
		})
	}

	allocations := make([]*v1.SeaOrderSplitAllocationItem, 0, len(splitCtx.Allocations))
	for _, a := range splitCtx.Allocations {
		cID := ""
		if a.ContainerID != nil {
			cID = a.ContainerID.String()
		}
		allocations = append(allocations, &v1.SeaOrderSplitAllocationItem{
			Id:            a.ID.String(),
			CargoItemId:   a.CargoItemID.String(),
			HouseBillId:   a.HouseBillID.String(),
			ContainerId:   cID,
			PackageCount:  a.PackageCount,
			GrossWeightKg: biz.FormatDecimal3(a.GrossWeightKg),
			VolumeCbm:     biz.FormatDecimal6(a.VolumeCbm),
		})
	}

	fees := make([]*v1.SeaOrderSplitDraftFeeItem, 0, len(splitCtx.DraftFees))
	for _, f := range splitCtx.DraftFees {
		fees = append(fees, &v1.SeaOrderSplitDraftFeeItem{
			Id:                  f.ID.String(),
			FeeCode:             f.FeeCode,
			FeeName:             f.FeeName,
			Direction:           f.Direction,
			SettlementPartyId:   f.SettlementPartyID.String(),
			SettlementPartyName: f.SettlementPartyName,
			Currency:            f.Currency,
			TotalAmount:         biz.FormatDecimal8(f.TotalAmount),
			BaseCurrency:        f.BaseCurrency,
			BaseCurrencyAmount:  biz.FormatDecimal8(f.BaseCurrencyAmount),
			Version:             f.Version,
		})
	}

	attachments := make([]*v1.SeaOrderSplitAttachmentItem, 0, len(splitCtx.Attachments))
	for _, att := range splitCtx.Attachments {
		attachments = append(attachments, &v1.SeaOrderSplitAttachmentItem{
			Id:       att.ID.String(),
			AssetId:  att.AssetID.String(),
			FileName: att.FileName,
			MimeType: att.MIMEType,
			FileSize: att.FileSize,
			DocType:  att.DocType,
		})
	}

	plans := make([]*v1.SeaOrderSplitContainerPlanItem, 0, len(splitCtx.ContainerPlans))
	for _, p := range splitCtx.ContainerPlans {
		plans = append(plans, &v1.SeaOrderSplitContainerPlanItem{
			ContainerSpecId:   p.ContainerSpecID.String(),
			ContainerSpecName: p.ContainerSpecName,
			Quantity:          p.Quantity,
		})
	}

	var mblSummary *v1.SeaOrderSplitMasterBillSummary
	if mb := splitCtx.CurrentMasterBill; mb != nil {
		cID := ""
		if mb.CarrierID != nil {
			cID = mb.CarrierID.String()
		}
		origLoc := ""
		if mb.OriginLocationID != nil {
			origLoc = mb.OriginLocationID.String()
		}
		disLoc := ""
		if mb.DischargeLocationID != nil {
			disLoc = mb.DischargeLocationID.String()
		}
		tranLoc := ""
		if mb.TransitLocationID != nil {
			tranLoc = mb.TransitLocationID.String()
		}
		mblSummary = &v1.SeaOrderSplitMasterBillSummary{
			Id:                        mb.MasterBillID.String(),
			MasterNo:                  mb.MasterNo,
			IssuerPartnerId:           mb.IssuerPartnerID.String(),
			IssuerPartnerName:         mb.IssuerPartnerName,
			CarrierId:                 cID,
			CarrierName:               mb.CarrierName,
			VesselName:                mb.VesselName,
			VoyageNo:                  mb.VoyageNo,
			Etd:                       mb.ETD,
			Eta:                       mb.ETA,
			Version:                   mb.Version,
			OriginLocationId:          origLoc,
			OriginLocationName:        mb.OriginLocationName,
			DischargeLocationId:       disLoc,
			DischargeLocationName:     mb.DischargeLocationName,
			TransitLocationId:         tranLoc,
			TransitLocationName:       mb.TransitLocationName,
			TransportExecutionId:      mb.TransportExecutionID.String(),
			TransportExecutionVersion: mb.TransportExecutionVersion,
		}
	}

	fp := splitCtx.AttachmentReferenceFingerprint
	data := &v1.SeaOrderSplitContextData{
		OrderId:                        splitCtx.OrderID.String(),
		OrderNo:                        splitCtx.OrderNo,
		BusinessType:                   splitCtx.BusinessType,
		ShipmentType:                   splitCtx.ShipmentType,
		FlowStatus:                     splitCtx.FlowStatus,
		OrderVersion:                   splitCtx.OrderVersion,
		CustomerReferenceNo:            splitCtx.CustomerReferenceNo,
		InternalReferenceNo:            splitCtx.InternalReferenceNo,
		BookingNotes:                   splitCtx.BookingNotes,
		AllocationNotes:                splitCtx.AllocationNotes,
		OperationNotes:                 splitCtx.OperationNotes,
		CurrentMasterBill:              mblSummary,
		CurrentLinkId:                  splitCtx.CurrentLinkID.String(),
		CurrentLinkVersion:             splitCtx.CurrentLinkVersion,
		DocumentStructure:              splitCtx.DocumentStructure,
		CargoAllocationStatus:          splitCtx.CargoAllocationStatus,
		CargoAllocationVersion:         splitCtx.CargoAllocationVersion,
		HouseBills:                     hbls,
		CargoItems:                     cargoItems,
		Containers:                     containers,
		Allocations:                    allocations,
		DraftFees:                      fees,
		Attachments:                    attachments,
		ContainerPlans:                 plans,
		AttachmentReferenceFingerprint: &fp,
	}

	return ok(ctx, &v1.GetSeaOrderSplitContextResponse{
		Data: data,
	}), nil
}

func (s *SeaOrderChangeService) PreviewSeaOrderSplit(ctx context.Context, request *v1.PreviewSeaOrderSplitRequest) (*v1.PreviewSeaOrderSplitResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	input, err := mapSplitInput(request.GetOrderId(), request.Note, request.GetTargets(), request.GetResults(), request.GetExpectedVersions())
	if err != nil {
		return nil, err
	}

	preview, err := s.usecase.PreviewSplit(ctx, principal.Organization.ID, input)
	if err != nil {
		return nil, err
	}

	return ok(ctx, &v1.PreviewSeaOrderSplitResponse{
		Data: mapSplitPreviewToAPI(preview),
	}), nil
}

func (s *SeaOrderChangeService) ExecuteSeaOrderSplit(ctx context.Context, request *v1.ExecuteSeaOrderSplitRequest) (*v1.ExecuteSeaOrderSplitResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	input, err := mapSplitInput(request.GetOrderId(), request.Note, request.GetTargets(), request.GetResults(), request.GetExpectedVersions())
	if err != nil {
		return nil, err
	}
	if splitUsesDifferentMasterBill(input) && !principal.HasPermissionInScope(
		access.OrderPermission(access.OrderBusinessSE, access.OrderReassign),
		biz.DataScopeOrganization,
	) {
		return nil, biz.ErrPermissionDenied
	}
	input.IdempotencyKey = request.GetIdempotencyKey()
	input.RequestFingerprint = request.GetRequestFingerprint()

	event, err := s.usecase.ExecuteSplit(ctx, principal.Organization.ID, principal.UserID, input)
	if err != nil {
		return nil, err
	}

	var origRef *v1.SeaOrderSplitOrderReference
	createdOrders := make([]*v1.SeaOrderSplitCreatedOrder, 0)
	for _, res := range event.Results {
		if res.ResultRole == biz.ResultRoleOriginal {
			origRef = &v1.SeaOrderSplitOrderReference{
				OrderId: res.OrderID.String(),
				OrderNo: res.OrderNo,
			}
		} else {
			createdOrders = append(createdOrders, &v1.SeaOrderSplitCreatedOrder{
				OrderId:         res.OrderID.String(),
				OrderNo:         res.OrderNo,
				ClientResultKey: res.ClientResultKey,
			})
		}
	}

	reassignIDs := make([]string, 0, len(event.ReassignmentEventIDs))
	for _, rid := range event.ReassignmentEventIDs {
		reassignIDs = append(reassignIDs, rid.String())
	}

	return ok(ctx, &v1.ExecuteSeaOrderSplitResponse{
		Data: &v1.ExecuteSeaOrderSplitData{
			SplitEventId:         event.ID.String(),
			CreatedAt:            event.CreatedAt.UTC().Format(time.RFC3339),
			OriginalOrder:        origRef,
			CreatedOrders:        createdOrders,
			ReassignmentEventIds: reassignIDs,
		},
	}), nil
}

func (s *SeaOrderChangeService) PreviewSeaOrderReassignment(ctx context.Context, request *v1.PreviewSeaOrderReassignmentRequest) (*v1.PreviewSeaOrderReassignmentResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrSeaOrderReassignmentInvalidArgument
	}

	target, err := mapReassignTarget(request.GetTarget())
	if err != nil {
		return nil, err
	}
	input := &biz.SeaOrderReassignmentInput{
		OrderID: orderID,
		Target:  target,
	}

	preview, err := s.usecase.PreviewReassignment(ctx, principal.Organization.ID, input)
	if err != nil {
		return nil, err
	}

	diffs := make([]*v1.VoyageDifferenceItem, 0, len(preview.Differences))
	for _, d := range preview.Differences {
		diffs = append(diffs, &v1.VoyageDifferenceItem{
			FieldName:    d.FieldName,
			Label:        d.Label,
			CurrentValue: d.CurrentValue,
			TargetValue:  d.TargetValue,
			IsDifferent:  d.IsDifferent,
		})
	}

	data := &v1.SeaOrderReassignmentPreviewData{
		IsValid:            preview.IsValid,
		Errors:             preview.Errors,
		CurrentMasterBill:  mapMblSummaryToAPI(preview.CurrentMasterBill),
		TargetMasterBill:   mapMblSummaryToAPI(preview.TargetMasterBill),
		TargetMemberCount:  preview.TargetMemberCount,
		Differences:        diffs,
		OrderVersion:       preview.OrderVersion,
		CurrentLinkVersion: preview.CurrentLinkVersion,
	}

	return ok(ctx, &v1.PreviewSeaOrderReassignmentResponse{
		Data: data,
	}), nil
}

func (s *SeaOrderChangeService) ExecuteSeaOrderReassignment(ctx context.Context, request *v1.ExecuteSeaOrderReassignmentRequest) (*v1.ExecuteSeaOrderReassignmentResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrSeaOrderReassignmentInvalidArgument
	}
	if request.GetExpectedOrderVersion() == 0 || request.GetExpectedLinkVersion() == 0 {
		return nil, biz.ErrSeaOrderReassignmentInvalidArgument
	}

	target, err := mapReassignTarget(request.GetTarget())
	if err != nil {
		return nil, err
	}
	var respPartnerID *uuid.UUID
	if request.GetResponsiblePartnerId() != "" {
		pID, err := uuid.Parse(request.GetResponsiblePartnerId())
		if err != nil || pID == uuid.Nil {
			return nil, biz.ErrSeaOrderReassignmentInvalidArgument
		}
		respPartnerID = &pID
	}

	input := &biz.SeaOrderReassignmentInput{
		OrderID:                     orderID,
		IdempotencyKey:              request.GetIdempotencyKey(),
		RequestFingerprint:          request.GetRequestFingerprint(),
		Target:                      target,
		Reason:                      request.GetReason(),
		ResponsibilityType:          request.GetResponsibilityType(),
		ResponsiblePartnerID:        respPartnerID,
		ExpectedOrderVersion:        request.GetExpectedOrderVersion(),
		ExpectedLinkVersion:         request.GetExpectedLinkVersion(),
		ExpectedCandidateMBLVersion: request.ExpectedCandidateMblVersion,
		ExpectedCandidateTEVersion:  request.ExpectedCandidateTeVersion,
	}

	event, err := s.usecase.ExecuteReassignment(ctx, principal.Organization.ID, principal.UserID, input)
	if err != nil {
		return nil, err
	}

	return ok(ctx, &v1.ExecuteSeaOrderReassignmentResponse{
		Data: &v1.ExecuteSeaOrderReassignmentData{
			ReassignmentEventId: event.ID.String(),
			CreatedAt:           event.CreatedAt.UTC().Format(time.RFC3339),
			OrderId:             event.OrderID.String(),
			OrderNo:             event.OrderNo,
			TargetMasterBillId:  event.TargetMasterBillID.String(),
			TargetMasterNo:      request.GetTarget().GetMasterNo(),
		},
	}), nil
}

func (s *SeaOrderChangeService) ListSeaOrderChangeEvents(ctx context.Context, request *v1.ListSeaOrderChangeEventsRequest) (*v1.ListSeaOrderChangeEventsResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrSeaOrderSplitInvalidArgument
	}

	page, pageSize := biz.ListPagination(int(request.GetPage()), int(request.GetPageSize()), 20)
	if !biz.ValidListPagination(page, pageSize) {
		return nil, biz.ErrSeaOrderSplitInvalidArgument
	}

	items, total, err := s.usecase.ListChangeEvents(ctx, principal.Organization.ID, orderID, int32(page), int32(pageSize))
	if err != nil {
		return nil, err
	}

	data := make([]*v1.SeaOrderChangeEventSummary, 0, len(items))
	for _, it := range items {
		opID := ""
		if it.OperatorID != nil {
			opID = it.OperatorID.String()
		}
		item := &v1.SeaOrderChangeEventSummary{
			Id:           it.ID.String(),
			EventType:    it.EventType,
			CreatedAt:    it.CreatedAt.UTC().Format(time.RFC3339),
			OperatorId:   opID,
			OperatorName: it.OperatorName,
			NoteOrReason: it.NoteOrReason,
		}
		if ss := it.SplitSummary; ss != nil {
			res := make([]*v1.SeaOrderSplitResultSummaryItem, 0, len(ss.Results))
			for _, r := range ss.Results {
				res = append(res, &v1.SeaOrderSplitResultSummaryItem{
					ResultRole:    r.ResultRole,
					OrderId:       r.OrderID.String(),
					OrderNo:       r.OrderNo,
					FinalMasterNo: r.FinalMasterNo,
					PackageCount:  r.PackageCount,
					GrossWeightKg: biz.FormatDecimal3(r.GrossWeightKg),
					VolumeCbm:     biz.FormatDecimal6(r.VolumeCbm),
				})
			}
			item.SplitSummary = &v1.SeaOrderSplitEventSummary{
				SourceOrderId: ss.SourceOrderID.String(),
				SourceOrderNo: ss.SourceOrderNo,
				ResultCount:   ss.ResultCount,
				Results:       res,
			}
		}
		if rs := it.ReassignmentSummary; rs != nil {
			item.ReassignmentSummary = &v1.SeaOrderReassignmentEventSummary{
				OrderId:                rs.OrderID.String(),
				OrderNo:                rs.OrderNo,
				PreviousMasterNo:       rs.PreviousMasterNo,
				TargetMasterNo:         rs.TargetMasterNo,
				ResponsibilityType:     rs.ResponsibilityType,
				ResponsiblePartnerName: rs.ResponsiblePartnerName,
				Reason:                 rs.Reason,
			}
		}
		data = append(data, item)
	}

	return okList(ctx, &v1.ListSeaOrderChangeEventsResponse{
		Data:  data,
		Total: total,
	}), nil
}

func (s *SeaOrderChangeService) GetSeaOrderChangeEvent(ctx context.Context, request *v1.GetSeaOrderChangeEventRequest) (*v1.GetSeaOrderChangeEventResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrSeaOrderSplitInvalidArgument
	}
	eventID, err := uuid.Parse(request.GetEventId())
	if err != nil {
		return nil, biz.ErrSeaOrderSplitInvalidArgument
	}

	detail, err := s.usecase.GetChangeEvent(ctx, principal.Organization.ID, orderID, eventID, request.GetEventType())
	if err != nil {
		return nil, err
	}

	opID := ""
	if detail.OperatorID != nil {
		opID = detail.OperatorID.String()
	}

	data := &v1.SeaOrderChangeEventDetailData{
		Id:                       detail.ID.String(),
		EventType:                detail.EventType,
		CreatedAt:                detail.CreatedAt.UTC().Format(time.RFC3339),
		OperatorId:               opID,
		OperatorName:             detail.OperatorName,
		NoteOrReason:             detail.NoteOrReason,
		BeforeSnapshotJson:       detail.BeforeSnapshotJSON,
		AfterSnapshotJson:        detail.AfterSnapshotJSON,
		ConservationSnapshotJson: detail.ConservationSnapshotJSON,
	}
	if ss := detail.SplitSummary; ss != nil {
		res := make([]*v1.SeaOrderSplitResultSummaryItem, 0, len(ss.Results))
		for _, r := range ss.Results {
			res = append(res, &v1.SeaOrderSplitResultSummaryItem{
				ResultRole:    r.ResultRole,
				OrderId:       r.OrderID.String(),
				OrderNo:       r.OrderNo,
				FinalMasterNo: r.FinalMasterNo,
				PackageCount:  r.PackageCount,
				GrossWeightKg: biz.FormatDecimal3(r.GrossWeightKg),
				VolumeCbm:     biz.FormatDecimal6(r.VolumeCbm),
			})
		}
		data.SplitSummary = &v1.SeaOrderSplitEventSummary{
			SourceOrderId: ss.SourceOrderID.String(),
			SourceOrderNo: ss.SourceOrderNo,
			ResultCount:   ss.ResultCount,
			Results:       res,
		}
	}
	if rs := detail.ReassignmentSummary; rs != nil {
		data.ReassignmentSummary = &v1.SeaOrderReassignmentEventSummary{
			OrderId:                rs.OrderID.String(),
			OrderNo:                rs.OrderNo,
			PreviousMasterNo:       rs.PreviousMasterNo,
			TargetMasterNo:         rs.TargetMasterNo,
			ResponsibilityType:     rs.ResponsibilityType,
			ResponsiblePartnerName: rs.ResponsiblePartnerName,
			Reason:                 rs.Reason,
		}
	}

	return ok(ctx, &v1.GetSeaOrderChangeEventResponse{
		Data: data,
	}), nil
}

// ---------------------------------------------------------------------------
// 转换映射辅助函数
// ---------------------------------------------------------------------------

func mapSplitInput(orderIDStr string, note *string, targets []*v1.SeaOrderSplitTargetInput, results []*v1.SeaOrderSplitResultInput, exp *v1.SeaOrderSplitExpectedVersions) (*biz.SeaOrderSplitInput, error) {
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return nil, biz.ErrSeaOrderSplitInvalidArgument
	}

	targetInputs := make([]*biz.SeaOrderSplitTargetInput, 0, len(targets))
	for _, t := range targets {
		var candID *uuid.UUID
		if t.CandidateId != nil && *t.CandidateId != "" {
			c, err := uuid.Parse(*t.CandidateId)
			if err != nil || c == uuid.Nil {
				return nil, biz.ErrSeaOrderSplitInvalidArgument
			}
			candID = &c
		}
		var issuerID *uuid.UUID
		if t.IssuerPartnerId != nil && *t.IssuerPartnerId != "" {
			c, err := uuid.Parse(*t.IssuerPartnerId)
			if err != nil || c == uuid.Nil {
				return nil, biz.ErrSeaOrderSplitInvalidArgument
			}
			issuerID = &c
		}
		var carrierID *uuid.UUID
		if t.CarrierId != nil && *t.CarrierId != "" {
			c, err := uuid.Parse(*t.CarrierId)
			if err != nil || c == uuid.Nil {
				return nil, biz.ErrSeaOrderSplitInvalidArgument
			}
			carrierID = &c
		}
		var origLocID *uuid.UUID
		if t.OriginLocationId != nil && *t.OriginLocationId != "" {
			c, err := uuid.Parse(*t.OriginLocationId)
			if err != nil || c == uuid.Nil {
				return nil, biz.ErrSeaOrderSplitInvalidArgument
			}
			origLocID = &c
		}
		var disLocID *uuid.UUID
		if t.DischargeLocationId != nil && *t.DischargeLocationId != "" {
			c, err := uuid.Parse(*t.DischargeLocationId)
			if err != nil || c == uuid.Nil {
				return nil, biz.ErrSeaOrderSplitInvalidArgument
			}
			disLocID = &c
		}
		var tranLocID *uuid.UUID
		if t.TransitLocationId != nil && *t.TransitLocationId != "" {
			c, err := uuid.Parse(*t.TransitLocationId)
			if err != nil || c == uuid.Nil {
				return nil, biz.ErrSeaOrderSplitInvalidArgument
			}
			tranLocID = &c
		}
		var candTeID *uuid.UUID
		if t.CandidateTeId != nil && *t.CandidateTeId != "" {
			c, err := uuid.Parse(*t.CandidateTeId)
			if err != nil || c == uuid.Nil {
				return nil, biz.ErrSeaOrderSplitInvalidArgument
			}
			candTeID = &c
		}
		targetInputs = append(targetInputs, &biz.SeaOrderSplitTargetInput{
			ClientTargetKey:     t.ClientTargetKey,
			TargetType:          t.TargetType,
			CandidateID:         candID,
			CandidateVersion:    t.CandidateVersion,
			MasterNo:            t.GetMasterNo(),
			IssuerPartnerID:     issuerID,
			CarrierID:           carrierID,
			VesselName:          t.GetVesselName(),
			VoyageNo:            t.GetVoyageNo(),
			ETD:                 t.GetEtd(),
			ETA:                 t.GetEta(),
			OriginLocationID:    origLocID,
			DischargeLocationID: disLocID,
			TransitLocationID:   tranLocID,
			CandidateTEID:       candTeID,
			CandidateTEVersion:  t.CandidateTeVersion,
		})
	}

	resultInputs := make([]*biz.SeaOrderSplitResultInput, 0, len(results))
	for _, r := range results {
		hblIDs := make([]uuid.UUID, 0, len(r.HouseBillIds))
		for _, hid := range r.HouseBillIds {
			u, err := uuid.Parse(hid)
			if err != nil || u == uuid.Nil {
				return nil, biz.ErrSeaOrderSplitInvalidArgument
			}
			hblIDs = append(hblIDs, u)
		}
		feeIDs := make([]uuid.UUID, 0, len(r.DraftFeeIds))
		for _, fid := range r.DraftFeeIds {
			u, err := uuid.Parse(fid)
			if err != nil || u == uuid.Nil {
				return nil, biz.ErrSeaOrderSplitInvalidArgument
			}
			feeIDs = append(feeIDs, u)
		}
		attIDs := make([]uuid.UUID, 0, len(r.AttachmentReferenceIds))
		for _, aid := range r.AttachmentReferenceIds {
			u, err := uuid.Parse(aid)
			if err != nil || u == uuid.Nil {
				return nil, biz.ErrSeaOrderSplitInvalidArgument
			}
			attIDs = append(attIDs, u)
		}
		resultInputs = append(resultInputs, &biz.SeaOrderSplitResultInput{
			ClientResultKey:        r.ClientResultKey,
			ResultRole:             r.ResultRole,
			ClientTargetKey:        r.ClientTargetKey,
			HouseBillIDs:           hblIDs,
			DraftFeeIDs:            feeIDs,
			AttachmentReferenceIDs: attIDs,
			InternalReferenceNo:    r.InternalReferenceNo,
			BookingNotes:           r.BookingNotes,
			AllocationNotes:        r.AllocationNotes,
			OperationNotes:         r.OperationNotes,
		})
	}

	var expectedVersions *biz.SeaOrderSplitExpectedVersions
	if exp != nil {
		hbVers := make(map[uuid.UUID]uint64, len(exp.HouseBillVersions))
		for k, v := range exp.HouseBillVersions {
			u, err := uuid.Parse(k)
			if err != nil || u == uuid.Nil {
				return nil, biz.ErrSeaOrderSplitInvalidArgument
			}
			hbVers[u] = v
		}
		ciVers := make(map[uuid.UUID]uint64, len(exp.CargoItemVersions))
		for k, v := range exp.CargoItemVersions {
			u, err := uuid.Parse(k)
			if err != nil || u == uuid.Nil {
				return nil, biz.ErrSeaOrderSplitInvalidArgument
			}
			ciVers[u] = v
		}
		cVers := make(map[uuid.UUID]uint64, len(exp.ContainerVersions))
		for k, v := range exp.ContainerVersions {
			u, err := uuid.Parse(k)
			if err != nil || u == uuid.Nil {
				return nil, biz.ErrSeaOrderSplitInvalidArgument
			}
			cVers[u] = v
		}
		fVers := make(map[uuid.UUID]uint64, len(exp.FeeVersions))
		for k, v := range exp.FeeVersions {
			u, err := uuid.Parse(k)
			if err != nil || u == uuid.Nil {
				return nil, biz.ErrSeaOrderSplitInvalidArgument
			}
			fVers[u] = v
		}
		candVers := make(map[uuid.UUID]uint64, len(exp.CandidateMblVersions))
		for k, v := range exp.CandidateMblVersions {
			u, err := uuid.Parse(k)
			if err != nil || u == uuid.Nil {
				return nil, biz.ErrSeaOrderSplitInvalidArgument
			}
			candVers[u] = v
		}

		candTeVers := make(map[uuid.UUID]uint64, len(exp.CandidateTeVersions))
		for k, v := range exp.CandidateTeVersions {
			u, err := uuid.Parse(k)
			if err != nil || u == uuid.Nil {
				return nil, biz.ErrSeaOrderSplitInvalidArgument
			}
			candTeVers[u] = v
		}

		attFp := ""
		if exp.AttachmentReferenceFingerprint != nil {
			attFp = *exp.AttachmentReferenceFingerprint
		}

		expectedVersions = &biz.SeaOrderSplitExpectedVersions{
			OrderVersion:                   exp.OrderVersion,
			LinkVersion:                    exp.LinkVersion,
			AllocationVersion:              exp.AllocationVersion,
			HouseBillVersions:              hbVers,
			CargoItemVersions:              ciVers,
			ContainerVersions:              cVers,
			FeeVersions:                    fVers,
			CandidateMBLVersions:           candVers,
			AttachmentReferenceFingerprint: attFp,
			CandidateTEVersions:            candTeVers,
		}
	}

	return &biz.SeaOrderSplitInput{
		OrderID:          orderID,
		Note:             note,
		Targets:          targetInputs,
		Results:          resultInputs,
		ExpectedVersions: expectedVersions,
	}, nil
}

func mapSplitPreviewToAPI(preview *biz.SeaOrderSplitPreview) *v1.SeaOrderSplitPreviewData {
	validationErrors := make([]*v1.SeaOrderSplitValidationError, 0, len(preview.ValidationErrors))
	for _, ve := range preview.ValidationErrors {
		validationErrors = append(validationErrors, &v1.SeaOrderSplitValidationError{
			Reason:          ve.Reason,
			Message:         ve.Message,
			Field:           ve.Field,
			ClientResultKey: ve.ClientResultKey,
			HouseBillId:     ve.HouseBillID,
			ContainerId:     ve.ContainerID,
			CargoItemId:     ve.CargoItemID,
			FeeId:           ve.FeeID,
			BaselineValue:   ve.BaselineValue,
			AllocatedValue:  ve.AllocatedValue,
			DiffValue:       ve.DiffValue,
		})
	}

	results := make([]*v1.SeaOrderSplitPreviewResultItem, 0, len(preview.Results))
	for _, r := range preview.Results {
		plans := make([]*v1.SeaOrderSplitContainerPlanItem, 0, len(r.ContainerPlans))
		for _, cp := range r.ContainerPlans {
			plans = append(plans, &v1.SeaOrderSplitContainerPlanItem{
				ContainerSpecId:   cp.ContainerSpecID.String(),
				ContainerSpecName: cp.ContainerSpecName,
				Quantity:          cp.Quantity,
			})
		}
		results = append(results, &v1.SeaOrderSplitPreviewResultItem{
			ClientResultKey:     r.ClientResultKey,
			ResultRole:          r.ResultRole,
			ClientTargetKey:     r.ClientTargetKey,
			PackageCount:        r.PackageCount,
			GrossWeightKg:       biz.FormatDecimal3(r.GrossWeightKg),
			VolumeCbm:           biz.FormatDecimal6(r.VolumeCbm),
			ContainerCount:      r.ContainerCount,
			HouseBillCount:      r.HouseBillCount,
			FeeCount:            r.FeeCount,
			AttachmentCount:     r.AttachmentCount,
			ContainerPlans:      plans,
			InternalReferenceNo: r.InternalReferenceNo,
			BookingNotes:        r.BookingNotes,
			AllocationNotes:     r.AllocationNotes,
			OperationNotes:      r.OperationNotes,
		})
	}

	return &v1.SeaOrderSplitPreviewData{
		IsValid:            preview.IsValid,
		ConservationPassed: preview.ConservationPassed,
		ValidationErrors:   validationErrors,
		Baseline:           mapSummaryToAPI(&preview.Baseline),
		Allocated:          mapSummaryToAPI(&preview.Allocated),
		Remaining:          mapSummaryToAPI(&preview.Remaining),
		Results:            results,
	}
}

func mapSummaryToAPI(s *biz.SeaOrderSplitQuantitySummary) *v1.SeaOrderSplitQuantitySummary {
	return &v1.SeaOrderSplitQuantitySummary{
		PackageCount:   s.PackageCount,
		GrossWeightKg:  biz.FormatDecimal3(s.GrossWeightKg),
		VolumeCbm:      biz.FormatDecimal6(s.VolumeCbm),
		ContainerCount: s.ContainerCount,
		HouseBillCount: s.HouseBillCount,
		FeeCount:       s.FeeCount,
	}
}

func mapReassignTarget(t *v1.SeaOrderReassignmentTargetInput) (*biz.SeaOrderReassignmentTargetInput, error) {
	if t == nil {
		return nil, biz.ErrSeaOrderReassignmentInvalidArgument
	}
	var candID *uuid.UUID
	if t.CandidateId != nil && *t.CandidateId != "" {
		c, err := uuid.Parse(*t.CandidateId)
		if err != nil || c == uuid.Nil {
			return nil, biz.ErrSeaOrderReassignmentInvalidArgument
		}
		candID = &c
	}
	var issuerID *uuid.UUID
	if t.IssuerPartnerId != nil && *t.IssuerPartnerId != "" {
		c, err := uuid.Parse(*t.IssuerPartnerId)
		if err != nil || c == uuid.Nil {
			return nil, biz.ErrSeaOrderReassignmentInvalidArgument
		}
		issuerID = &c
	}
	var carrierID *uuid.UUID
	if t.CarrierId != nil && *t.CarrierId != "" {
		c, err := uuid.Parse(*t.CarrierId)
		if err != nil || c == uuid.Nil {
			return nil, biz.ErrSeaOrderReassignmentInvalidArgument
		}
		carrierID = &c
	}
	var origLocID *uuid.UUID
	if t.OriginLocationId != nil && *t.OriginLocationId != "" {
		c, err := uuid.Parse(*t.OriginLocationId)
		if err != nil || c == uuid.Nil {
			return nil, biz.ErrSeaOrderReassignmentInvalidArgument
		}
		origLocID = &c
	}
	var disLocID *uuid.UUID
	if t.DischargeLocationId != nil && *t.DischargeLocationId != "" {
		c, err := uuid.Parse(*t.DischargeLocationId)
		if err != nil || c == uuid.Nil {
			return nil, biz.ErrSeaOrderReassignmentInvalidArgument
		}
		disLocID = &c
	}
	var tranLocID *uuid.UUID
	if t.TransitLocationId != nil && *t.TransitLocationId != "" {
		c, err := uuid.Parse(*t.TransitLocationId)
		if err != nil || c == uuid.Nil {
			return nil, biz.ErrSeaOrderReassignmentInvalidArgument
		}
		tranLocID = &c
	}
	var candTeID *uuid.UUID
	if t.CandidateTeId != nil && *t.CandidateTeId != "" {
		c, err := uuid.Parse(*t.CandidateTeId)
		if err != nil || c == uuid.Nil {
			return nil, biz.ErrSeaOrderReassignmentInvalidArgument
		}
		candTeID = &c
	}
	return &biz.SeaOrderReassignmentTargetInput{
		TargetType:          t.GetTargetType(),
		CandidateID:         candID,
		CandidateVersion:    t.CandidateVersion,
		CandidateTEID:       candTeID,
		CandidateTEVersion:  t.CandidateTeVersion,
		MasterNo:            t.GetMasterNo(),
		IssuerPartnerID:     issuerID,
		CarrierID:           carrierID,
		VesselName:          t.GetVesselName(),
		VoyageNo:            t.GetVoyageNo(),
		ETD:                 t.GetEtd(),
		ETA:                 t.GetEta(),
		OriginLocationID:    origLocID,
		DischargeLocationID: disLocID,
		TransitLocationID:   tranLocID,
	}, nil
}

func mapMblSummaryToAPI(mb *biz.SeaMasterBillSummary) *v1.SeaOrderSplitMasterBillSummary {
	if mb == nil {
		return nil
	}
	cID := ""
	if mb.CarrierID != nil {
		cID = mb.CarrierID.String()
	}
	origLoc := ""
	if mb.OriginLocationID != nil {
		origLoc = mb.OriginLocationID.String()
	}
	disLoc := ""
	if mb.DischargeLocationID != nil {
		disLoc = mb.DischargeLocationID.String()
	}
	tranLoc := ""
	if mb.TransitLocationID != nil {
		tranLoc = mb.TransitLocationID.String()
	}
	return &v1.SeaOrderSplitMasterBillSummary{
		Id:                        mb.MasterBillID.String(),
		MasterNo:                  mb.MasterNo,
		IssuerPartnerId:           mb.IssuerPartnerID.String(),
		IssuerPartnerName:         mb.IssuerPartnerName,
		CarrierId:                 cID,
		CarrierName:               mb.CarrierName,
		VesselName:                mb.VesselName,
		VoyageNo:                  mb.VoyageNo,
		Etd:                       mb.ETD,
		Eta:                       mb.ETA,
		Version:                   mb.Version,
		OriginLocationId:          origLoc,
		OriginLocationName:        mb.OriginLocationName,
		DischargeLocationId:       disLoc,
		DischargeLocationName:     mb.DischargeLocationName,
		TransitLocationId:         tranLoc,
		TransitLocationName:       mb.TransitLocationName,
		TransportExecutionId:      mb.TransportExecutionID.String(),
		TransportExecutionVersion: mb.TransportExecutionVersion,
	}
}

func splitUsesDifferentMasterBill(input *biz.SeaOrderSplitInput) bool {
	if input == nil {
		return false
	}
	targetTypes := make(map[string]string, len(input.Targets))
	for _, target := range input.Targets {
		if target != nil {
			targetTypes[target.ClientTargetKey] = target.TargetType
		}
	}
	for _, result := range input.Results {
		if result == nil {
			continue
		}
		targetType := targetTypes[result.ClientTargetKey]
		if targetType == biz.SplitTargetTypeNew || targetType == biz.SplitTargetTypeCandidate {
			return true
		}
	}
	return false
}

var _ v1.SeaOrderChangeServiceServer = (*SeaOrderChangeService)(nil)
