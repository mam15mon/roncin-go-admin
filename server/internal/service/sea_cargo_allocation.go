package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type SeaCargoAllocationService struct {
	v1.UnimplementedSeaCargoAllocationServiceServer
	usecase *biz.SeaCargoAllocationUsecase
}

func NewSeaCargoAllocationService(usecase *biz.SeaCargoAllocationUsecase) *SeaCargoAllocationService {
	return &SeaCargoAllocationService{usecase: usecase}
}

var _ v1.SeaCargoAllocationServiceServer = (*SeaCargoAllocationService)(nil)

func (s *SeaCargoAllocationService) GetSeaCargoAllocation(ctx context.Context, req *v1.GetSeaCargoAllocationRequest) (*v1.GetSeaCargoAllocationResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, biz.ErrSeaCargoAllocationInvalidArgument
	}

	agg, err := s.usecase.GetSeaCargoAllocation(ctx, principal.Organization.ID, orderID)
	if err != nil {
		return nil, err
	}

	return ok(ctx, &v1.GetSeaCargoAllocationResponse{
		Data: seaCargoAllocationAggregateToAPI(agg),
	}), nil
}

func (s *SeaCargoAllocationService) SaveSeaCargoAllocationDraft(ctx context.Context, req *v1.SaveSeaCargoAllocationDraftRequest) (*v1.SaveSeaCargoAllocationDraftResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, biz.ErrSeaCargoAllocationInvalidArgument
	}

	inputs := make([]*biz.SeaCargoAllocationInput, 0, len(req.GetAllocations()))
	for _, a := range req.GetAllocations() {
		cargoItemID, err := uuid.Parse(a.GetCargoItemId())
		if err != nil {
			return nil, biz.ErrSeaCargoAllocationInvalidArgument
		}
		houseBillID, err := uuid.Parse(a.GetHouseBillId())
		if err != nil {
			return nil, biz.ErrSeaCargoAllocationInvalidArgument
		}
		var containerID *uuid.UUID
		if a.GetContainerId() != "" {
			cid, err := uuid.Parse(a.GetContainerId())
			if err != nil {
				return nil, biz.ErrSeaCargoAllocationInvalidArgument
			}
			containerID = &cid
		}

		weight, err := biz.ParseAndValidateWeight(a.GetGrossWeightKg())
		if err != nil {
			return nil, err
		}
		volume, err := biz.ParseAndValidateVolume(a.GetVolumeCbm())
		if err != nil {
			return nil, err
		}

		var id *uuid.UUID
		if a.GetId() != "" {
			parsedID, err := uuid.Parse(a.GetId())
			if err != nil {
				return nil, biz.ErrSeaCargoAllocationInvalidArgument
			}
			id = &parsedID
		}

		inputs = append(inputs, &biz.SeaCargoAllocationInput{
			ID:            id,
			CargoItemID:   cargoItemID,
			HouseBillID:   houseBillID,
			ContainerID:   containerID,
			PackageCount:  a.GetPackageCount(),
			GrossWeightKg: weight,
			VolumeCbm:     volume,
		})
	}

	audit := &biz.AuditEvent{
		OrganizationID: &principal.Organization.ID,
		UserID:         &principal.UserID,
		Result:         "success",
	}

	agg, err := s.usecase.SaveDraft(ctx, principal.Organization.ID, principal.UserID, orderID, req.GetExpectedAllocationVersion(), inputs, audit)
	if err != nil {
		return nil, err
	}

	return ok(ctx, &v1.SaveSeaCargoAllocationDraftResponse{
		Data: seaCargoAllocationAggregateToAPI(agg),
	}), nil
}

func (s *SeaCargoAllocationService) ConfirmSeaCargoAllocation(ctx context.Context, req *v1.ConfirmSeaCargoAllocationRequest) (*v1.ConfirmSeaCargoAllocationResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, biz.ErrSeaCargoAllocationInvalidArgument
	}

	audit := &biz.AuditEvent{
		OrganizationID: &principal.Organization.ID,
		UserID:         &principal.UserID,
		Result:         "success",
	}

	agg, err := s.usecase.Confirm(ctx, principal.Organization.ID, principal.UserID, orderID, req.GetExpectedAllocationVersion(), audit)
	if err != nil {
		return nil, err
	}

	return ok(ctx, &v1.ConfirmSeaCargoAllocationResponse{
		Data: seaCargoAllocationAggregateToAPI(agg),
	}), nil
}

func (s *SeaCargoAllocationService) WithdrawSeaCargoAllocation(ctx context.Context, req *v1.WithdrawSeaCargoAllocationRequest) (*v1.WithdrawSeaCargoAllocationResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, biz.ErrSeaCargoAllocationInvalidArgument
	}

	audit := &biz.AuditEvent{
		OrganizationID: &principal.Organization.ID,
		UserID:         &principal.UserID,
		Result:         "success",
	}

	agg, err := s.usecase.Withdraw(ctx, principal.Organization.ID, principal.UserID, orderID, req.GetExpectedAllocationVersion(), audit)
	if err != nil {
		return nil, err
	}

	return ok(ctx, &v1.WithdrawSeaCargoAllocationResponse{
		Data: seaCargoAllocationAggregateToAPI(agg),
	}), nil
}

func (s *SeaCargoAllocationService) ApplySeaHouseBillAllocationSummary(ctx context.Context, req *v1.ApplySeaHouseBillAllocationSummaryRequest) (*v1.ApplySeaHouseBillAllocationSummaryResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, biz.ErrSeaCargoAllocationInvalidArgument
	}
	houseBillID, err := uuid.Parse(req.GetHouseBillId())
	if err != nil {
		return nil, biz.ErrSeaCargoAllocationInvalidArgument
	}

	audit := &biz.AuditEvent{
		OrganizationID: &principal.Organization.ID,
		UserID:         &principal.UserID,
		Result:         "success",
	}

	hb, err := s.usecase.ApplyHouseBillSummary(ctx, principal.Organization.ID, principal.UserID, orderID, houseBillID, req.GetExpectedAllocationVersion(), req.GetExpectedHouseBillVersion(), audit)
	if err != nil {
		return nil, err
	}

	return ok(ctx, &v1.ApplySeaHouseBillAllocationSummaryResponse{
		Data: seaHouseBillToAPI(hb),
	}), nil
}

func (s *SeaCargoAllocationService) ApplySeaOrderCargoSummaryToMasterBill(ctx context.Context, req *v1.ApplySeaOrderCargoSummaryToMasterBillRequest) (*v1.ApplySeaOrderCargoSummaryToMasterBillResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, biz.ErrSeaCargoAllocationInvalidArgument
	}

	audit := &biz.AuditEvent{
		OrganizationID: &principal.Organization.ID,
		UserID:         &principal.UserID,
		Result:         "success",
	}

	mbl, err := s.usecase.ApplyMasterBillSummary(ctx, principal.Organization.ID, principal.UserID, orderID, req.GetExpectedMblVersion(), audit)
	if err != nil {
		return nil, err
	}

	return ok(ctx, &v1.ApplySeaOrderCargoSummaryToMasterBillResponse{
		Data: seaMasterBillDetailToAPI(mbl),
	}), nil
}

func seaCargoAllocationAggregateToAPI(agg *biz.SeaCargoAllocationAggregate) *v1.SeaCargoAllocationAggregate {
	if agg == nil {
		return nil
	}

	var confirmedAt *string
	if agg.ConfirmedAt != nil {
		s := agg.ConfirmedAt.UTC().Format(time.RFC3339)
		confirmedAt = &s
	}
	var confirmedBy *string
	if agg.ConfirmedBy != nil {
		s := agg.ConfirmedBy.String()
		confirmedBy = &s
	}
	var confirmedByName *string
	if agg.ConfirmedByName != "" {
		s := agg.ConfirmedByName
		confirmedByName = &s
	}

	cargoItems := make([]*v1.OrderCargoItem, 0, len(agg.CargoItems))
	for _, ci := range agg.CargoItems {
		cargoItems = append(cargoItems, orderCargoItemToAPI(ci))
	}

	containers := make([]*v1.OrderContainer, 0, len(agg.Containers))
	for _, c := range agg.Containers {
		containers = append(containers, orderContainerToAPI(c))
	}

	houseBills := make([]*v1.SeaHouseBill, 0, len(agg.HouseBills))
	for _, hb := range agg.HouseBills {
		houseBills = append(houseBills, seaHouseBillToAPI(hb))
	}

	allocations := make([]*v1.SeaCargoAllocationItem, 0, len(agg.Allocations))
	for _, a := range agg.Allocations {
		var cid *string
		if a.ContainerID != nil {
			s := a.ContainerID.String()
			cid = &s
		}
		allocations = append(allocations, &v1.SeaCargoAllocationItem{
			Id:            a.ID.String(),
			CargoItemId:   a.CargoItemID.String(),
			HouseBillId:   a.HouseBillID.String(),
			ContainerId:   cid,
			PackageCount:  a.PackageCount,
			GrossWeightKg: a.GrossWeightKg.StringFixed(3),
			VolumeCbm:     a.VolumeCbm.StringFixed(6),
		})
	}

	allowedActions := make([]v1.SeaCargoAllocationAction, 0, len(agg.AllowedActions))
	for _, act := range agg.AllowedActions {
		allowedActions = append(allowedActions, seaCargoAllocationActionToAPI(act))
	}

	return &v1.SeaCargoAllocationAggregate{
		OrderId:           agg.OrderID.String(),
		DocumentStructure: seaDocumentStructureToAPI(agg.DocumentStructure),
		ShipmentType:      agg.ShipmentType,
		AllocationStatus:  seaCargoAllocationStatusToAPI(agg.AllocationStatus),
		AllocationVersion: agg.AllocationVersion,
		ConfirmedAt:       confirmedAt,
		ConfirmedBy:       confirmedBy,
		ConfirmedByName:   confirmedByName,
		CargoItems:        cargoItems,
		Containers:        containers,
		HouseBills:        houseBills,
		Allocations:       allocations,
		Progress:          seaCargoAllocationProgressToAPI(agg.Progress),
		AllowedActions:    allowedActions,
	}
}

func seaCargoAllocationProgressToAPI(p *biz.SeaCargoAllocationProgress) *v1.SeaCargoAllocationProgress {
	if p == nil {
		return nil
	}

	cargoSummaries := make([]*v1.SeaCargoAllocationCargoItemSummary, 0, len(p.CargoSummaries))
	for _, cs := range p.CargoSummaries {
		cargoSummaries = append(cargoSummaries, &v1.SeaCargoAllocationCargoItemSummary{
			CargoItemId:            cs.CargoItemID.String(),
			CargoName:              cs.CargoName,
			BaselinePackageCount:   cs.BaselinePackageCount,
			AllocatedPackageCount:  cs.AllocatedPackageCount,
			RemainingPackageCount:  cs.RemainingPackageCount,
			BaselineGrossWeightKg:  cs.BaselineGrossWeightKg.StringFixed(3),
			AllocatedGrossWeightKg: cs.AllocatedGrossWeightKg.StringFixed(3),
			RemainingGrossWeightKg: cs.RemainingGrossWeightKg.StringFixed(3),
			BaselineVolumeCbm:      cs.BaselineVolumeCbm.StringFixed(6),
			AllocatedVolumeCbm:     cs.AllocatedVolumeCbm.StringFixed(6),
			RemainingVolumeCbm:     cs.RemainingVolumeCbm.StringFixed(6),
			Status:                 string(cs.Status),
		})
	}

	containerSummaries := make([]*v1.SeaCargoAllocationContainerSummary, 0, len(p.ContainerSummaries))
	for _, cns := range p.ContainerSummaries {
		containerSummaries = append(containerSummaries, &v1.SeaCargoAllocationContainerSummary{
			ContainerId:            cns.ContainerID.String(),
			ContainerNo:            cns.ContainerNo,
			BaselinePackageCount:   cns.BaselinePackageCount,
			AllocatedPackageCount:  cns.AllocatedPackageCount,
			RemainingPackageCount:  cns.RemainingPackageCount,
			BaselineGrossWeightKg:  cns.BaselineGrossWeightKg.StringFixed(3),
			AllocatedGrossWeightKg: cns.AllocatedGrossWeightKg.StringFixed(3),
			RemainingGrossWeightKg: cns.RemainingGrossWeightKg.StringFixed(3),
			BaselineVolumeCbm:      cns.BaselineVolumeCbm.StringFixed(6),
			AllocatedVolumeCbm:     cns.AllocatedVolumeCbm.StringFixed(6),
			RemainingVolumeCbm:     cns.RemainingVolumeCbm.StringFixed(6),
			Status:                 string(cns.Status),
		})
	}

	houseBillSummaries := make([]*v1.SeaCargoAllocationHouseBillSummary, 0, len(p.HouseBillSummaries))
	for _, hbs := range p.HouseBillSummaries {
		var dispPkg *int32
		if hbs.DisplayPackageCount != nil {
			v := *hbs.DisplayPackageCount
			dispPkg = &v
		}
		var dispWeight *string
		if hbs.DisplayGrossWeightKg != nil {
			s := hbs.DisplayGrossWeightKg.StringFixed(3)
			dispWeight = &s
		}
		var dispVol *string
		if hbs.DisplayVolumeCbm != nil {
			s := hbs.DisplayVolumeCbm.StringFixed(6)
			dispVol = &s
		}
		houseBillSummaries = append(houseBillSummaries, &v1.SeaCargoAllocationHouseBillSummary{
			HouseBillId:                 hbs.HouseBillID.String(),
			HouseNo:                     hbs.HouseNo,
			AllocatedPackageCount:       hbs.AllocatedPackageCount,
			AllocatedGrossWeightKg:      hbs.AllocatedGrossWeightKg.StringFixed(3),
			AllocatedVolumeCbm:          hbs.AllocatedVolumeCbm.StringFixed(6),
			OrderRemainingPackageCount:  hbs.OrderRemainingPackageCount,
			OrderRemainingGrossWeightKg: hbs.OrderRemainingGrossWeightKg.StringFixed(3),
			OrderRemainingVolumeCbm:     hbs.OrderRemainingVolumeCbm.StringFixed(6),
			DisplayPackageCount:         dispPkg,
			DisplayGrossWeightKg:        dispWeight,
			DisplayVolumeCbm:            dispVol,
			DiffPackageCount:            hbs.DiffPackageCount,
			DiffGrossWeightKg:           hbs.DiffGrossWeightKg.StringFixed(3),
			DiffVolumeCbm:               hbs.DiffVolumeCbm.StringFixed(6),
			DisplayMatches:              hbs.DisplayMatches,
		})
	}

	return &v1.SeaCargoAllocationProgress{
		CargoSummaries:              cargoSummaries,
		ContainerSummaries:          containerSummaries,
		HouseBillSummaries:          houseBillSummaries,
		OrderRemainingPackageCount:  p.OrderRemainingPackageCount,
		OrderRemainingGrossWeightKg: p.OrderRemainingGrossWeightKg.StringFixed(3),
		OrderRemainingVolumeCbm:     p.OrderRemainingVolumeCbm.StringFixed(6),
	}
}

func seaCargoAllocationStatusToAPI(s biz.SeaCargoAllocationStatus) v1.SeaCargoAllocationStatus {
	switch s {
	case biz.SeaCargoAllocationStatusDraft:
		return v1.SeaCargoAllocationStatus_SEA_CARGO_ALLOCATION_STATUS_DRAFT
	case biz.SeaCargoAllocationStatusConfirmed:
		return v1.SeaCargoAllocationStatus_SEA_CARGO_ALLOCATION_STATUS_CONFIRMED
	default:
		return v1.SeaCargoAllocationStatus_SEA_CARGO_ALLOCATION_STATUS_UNSPECIFIED
	}
}

func seaCargoAllocationActionToAPI(a biz.SeaCargoAllocationAction) v1.SeaCargoAllocationAction {
	switch a {
	case biz.SeaCargoAllocationActionSaveDraft:
		return v1.SeaCargoAllocationAction_SEA_CARGO_ALLOCATION_ACTION_SAVE_DRAFT
	case biz.SeaCargoAllocationActionConfirm:
		return v1.SeaCargoAllocationAction_SEA_CARGO_ALLOCATION_ACTION_CONFIRM
	case biz.SeaCargoAllocationActionWithdraw:
		return v1.SeaCargoAllocationAction_SEA_CARGO_ALLOCATION_ACTION_WITHDRAW
	case biz.SeaCargoAllocationActionApplyHouseBillSummary:
		return v1.SeaCargoAllocationAction_SEA_CARGO_ALLOCATION_ACTION_APPLY_HOUSE_BILL_SUMMARY
	case biz.SeaCargoAllocationActionApplyMasterBillSummary:
		return v1.SeaCargoAllocationAction_SEA_CARGO_ALLOCATION_ACTION_APPLY_MASTER_BILL_SUMMARY
	default:
		return v1.SeaCargoAllocationAction_SEA_CARGO_ALLOCATION_ACTION_UNSPECIFIED
	}
}
