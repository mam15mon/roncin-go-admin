package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type SeaSharedContainerService struct {
	v1.UnimplementedSeaSharedContainerServiceServer
	usecase *biz.SeaSharedContainerUsecase
}

func NewSeaSharedContainerService(usecase *biz.SeaSharedContainerUsecase) *SeaSharedContainerService {
	return &SeaSharedContainerService{usecase: usecase}
}

var _ v1.SeaSharedContainerServiceServer = (*SeaSharedContainerService)(nil)

func (s *SeaSharedContainerService) ListSeaSharedContainers(ctx context.Context, request *v1.ListSeaSharedContainersRequest) (*v1.ListSeaSharedContainersResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	executionID, err := uuid.Parse(request.GetTransportExecutionId())
	if err != nil {
		return nil, biz.ErrSeaSharedContainerInvalidArgument
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrSeaSharedContainerInvalidArgument)
	if err != nil {
		return nil, err
	}
	items, total, err := s.usecase.List(ctx, principal.Organization.ID, executionID, request.GetKeyword(), page, pageSize)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.SeaSharedContainer, 0, len(items))
	for _, item := range items {
		data = append(data, seaSharedContainerToAPI(item))
	}
	return okList(ctx, &v1.ListSeaSharedContainersResponse{Data: data, Total: int32(total), Page: int32(page), PageSize: int32(pageSize)}), nil
}

func (s *SeaSharedContainerService) GetSeaSharedContainer(ctx context.Context, request *v1.GetSeaSharedContainerRequest) (*v1.GetSeaSharedContainerResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrSeaSharedContainerInvalidArgument
	}
	item, err := s.usecase.Get(ctx, principal.Organization.ID, id)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.GetSeaSharedContainerResponse{Data: seaSharedContainerToAPI(item)}), nil
}

func (s *SeaSharedContainerService) ListSeaSharedContainerCandidates(ctx context.Context, request *v1.ListSeaSharedContainerCandidatesRequest) (*v1.ListSeaSharedContainerCandidatesResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	executionID, err := uuid.Parse(request.GetTransportExecutionId())
	if err != nil {
		return nil, biz.ErrSeaSharedContainerInvalidArgument
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrSeaSharedContainerInvalidArgument)
	if err != nil {
		return nil, err
	}
	items, total, err := s.usecase.ListCandidates(ctx, principal.Organization.ID, executionID, request.GetKeyword(), page, pageSize)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.SeaSharedContainerCandidateOrder, 0, len(items))
	for _, item := range items {
		data = append(data, seaSharedContainerCandidateToAPI(item))
	}
	return okList(ctx, &v1.ListSeaSharedContainerCandidatesResponse{Data: data, Total: int32(total), Page: int32(page), PageSize: int32(pageSize)}), nil
}

func (s *SeaSharedContainerService) CreateSeaSharedContainer(ctx context.Context, request *v1.CreateSeaSharedContainerRequest) (*v1.CreateSeaSharedContainerResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	input, err := seaSharedContainerInputFromAPI(request.GetInput())
	if err != nil {
		return nil, err
	}
	item, err := s.usecase.Create(ctx, principal.Organization.ID, principal.UserID, input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateSeaSharedContainerResponse{Data: seaSharedContainerToAPI(item)}), nil
}

func (s *SeaSharedContainerService) UpdateSeaSharedContainer(ctx context.Context, request *v1.UpdateSeaSharedContainerRequest) (*v1.UpdateSeaSharedContainerResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrSeaSharedContainerInvalidArgument
	}
	input, err := seaSharedContainerInputFromAPI(request.GetInput())
	if err != nil {
		return nil, err
	}
	item, err := s.usecase.Update(ctx, principal.Organization.ID, principal.UserID, id, request.GetExpectedVersion(), input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.UpdateSeaSharedContainerResponse{Data: seaSharedContainerToAPI(item)}), nil
}

func (s *SeaSharedContainerService) DeleteSeaSharedContainer(ctx context.Context, request *v1.DeleteSeaSharedContainerRequest) (*v1.DeleteSeaSharedContainerResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrSeaSharedContainerInvalidArgument
	}
	if err := s.usecase.Delete(ctx, principal.Organization.ID, principal.UserID, id, request.GetExpectedVersion()); err != nil {
		return nil, err
	}
	return ok(ctx, &v1.DeleteSeaSharedContainerResponse{}), nil
}

func (s *SeaSharedContainerService) SaveSeaSharedContainerAllocationsDraft(ctx context.Context, request *v1.SaveSeaSharedContainerAllocationsDraftRequest) (*v1.SaveSeaSharedContainerAllocationsDraftResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrSeaSharedContainerInvalidArgument
	}
	allocations, err := seaSharedAllocationInputsFromAPI(request.GetAllocations())
	if err != nil {
		return nil, err
	}
	item, err := s.usecase.SaveDraft(ctx, principal.Organization.ID, principal.UserID, id, request.GetExpectedVersion(), allocations)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.SaveSeaSharedContainerAllocationsDraftResponse{Data: seaSharedContainerToAPI(item)}), nil
}

func (s *SeaSharedContainerService) ConfirmSeaSharedContainer(ctx context.Context, request *v1.ConfirmSeaSharedContainerRequest) (*v1.ConfirmSeaSharedContainerResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrSeaSharedContainerInvalidArgument
	}
	item, err := s.usecase.Confirm(ctx, principal.Organization.ID, principal.UserID, id, request.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ConfirmSeaSharedContainerResponse{Data: seaSharedContainerToAPI(item)}), nil
}

func (s *SeaSharedContainerService) WithdrawSeaSharedContainer(ctx context.Context, request *v1.WithdrawSeaSharedContainerRequest) (*v1.WithdrawSeaSharedContainerResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrSeaSharedContainerInvalidArgument
	}
	item, err := s.usecase.Withdraw(ctx, principal.Organization.ID, principal.UserID, id, request.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.WithdrawSeaSharedContainerResponse{Data: seaSharedContainerToAPI(item)}), nil
}

func seaSharedContainerInputFromAPI(input *v1.SeaSharedContainerInput) (*biz.SeaSharedContainer, error) {
	if input == nil {
		return nil, biz.ErrSeaSharedContainerInvalidArgument
	}
	executionID, err := uuid.Parse(input.GetTransportExecutionId())
	if err != nil {
		return nil, biz.ErrSeaSharedContainerInvalidArgument
	}
	specID, err := uuid.Parse(input.GetContainerSpecId())
	if err != nil {
		return nil, biz.ErrSeaSharedContainerInvalidArgument
	}
	weight, err := biz.ParseSharedWeight(input.GetGrossWeightKg())
	if err != nil {
		return nil, err
	}
	volume, err := biz.ParseSharedVolume(input.GetVolumeCbm())
	if err != nil {
		return nil, err
	}
	return &biz.SeaSharedContainer{TransportExecutionID: executionID, ContainerNo: input.GetContainerNo(), ContainerSpecID: specID, SealNo: input.SealNo, PackageCount: input.GetPackageCount(), GrossWeightKg: weight, VolumeCbm: volume, Note: input.Note}, nil
}

func seaSharedAllocationInputsFromAPI(inputs []*v1.SeaSharedContainerAllocationInput) ([]*biz.SeaSharedContainerAllocationInput, error) {
	result := make([]*biz.SeaSharedContainerAllocationInput, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			return nil, biz.ErrSeaSharedContainerInvalidArgument
		}
		orderID, err := uuid.Parse(input.GetOrderId())
		if err != nil {
			return nil, biz.ErrSeaSharedContainerInvalidArgument
		}
		houseBillID, err := uuid.Parse(input.GetHouseBillId())
		if err != nil {
			return nil, biz.ErrSeaSharedContainerInvalidArgument
		}
		cargoID, err := uuid.Parse(input.GetCargoItemId())
		if err != nil {
			return nil, biz.ErrSeaSharedContainerInvalidArgument
		}
		weight, err := biz.ParseSharedWeight(input.GetGrossWeightKg())
		if err != nil {
			return nil, err
		}
		volume, err := biz.ParseSharedVolume(input.GetVolumeCbm())
		if err != nil {
			return nil, err
		}
		result = append(result, &biz.SeaSharedContainerAllocationInput{OrderID: orderID, HouseBillID: houseBillID, CargoItemID: cargoID, PackageCount: input.GetPackageCount(), GrossWeightKg: weight, VolumeCbm: volume})
		result[len(result)-1].ExpectedOrderVersion = input.GetExpectedOrderVersion()
		result[len(result)-1].ExpectedLinkVersion = input.GetExpectedLinkVersion()
		result[len(result)-1].ExpectedHouseBillVersion = input.GetExpectedHouseBillVersion()
		result[len(result)-1].ExpectedCargoItemVersion = input.GetExpectedCargoItemVersion()
	}
	return result, nil
}

func seaSharedContainerToAPI(item *biz.SeaSharedContainer) *v1.SeaSharedContainer {
	if item == nil {
		return nil
	}
	result := &v1.SeaSharedContainer{Id: item.ID.String(), OrganizationId: item.OrganizationID.String(), TransportExecutionId: item.TransportExecutionID.String(), ContainerNo: item.ContainerNo, ContainerSpecId: item.ContainerSpecID.String(), ContainerSpecName: item.ContainerSpecName, SealNo: item.SealNo, PackageCount: item.PackageCount, GrossWeightKg: item.GrossWeightKg.StringFixed(3), VolumeCbm: item.VolumeCbm.StringFixed(6), Status: seaSharedContainerStatusToAPI(item.Status), Note: item.Note, Version: item.Version, CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339)}
	if item.ConfirmedAt != nil {
		value := item.ConfirmedAt.UTC().Format(time.RFC3339)
		result.ConfirmedAt = &value
	}
	if item.ConfirmedBy != nil {
		value := item.ConfirmedBy.String()
		result.ConfirmedBy = &value
	}
	if item.ConfirmedByName != "" {
		value := item.ConfirmedByName
		result.ConfirmedByName = &value
	}
	for _, allocation := range item.Allocations {
		result.Allocations = append(result.Allocations, &v1.SeaSharedContainerAllocation{Id: allocation.ID.String(), SharedContainerId: allocation.SharedContainerID.String(), OrderId: allocation.OrderID.String(), OrderNo: allocation.OrderNo, HouseBillId: allocation.HouseBillID.String(), HouseNo: allocation.HouseNo, CargoItemId: allocation.CargoItemID.String(), CargoName: allocation.CargoName, PackageCount: allocation.PackageCount, GrossWeightKg: allocation.GrossWeightKg.StringFixed(3), VolumeCbm: allocation.VolumeCbm.StringFixed(6), Version: allocation.Version, CreatedAt: allocation.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: allocation.UpdatedAt.UTC().Format(time.RFC3339), OrderVersion: allocation.OrderVersion, LinkVersion: allocation.LinkVersion, HouseBillVersion: allocation.HouseBillVersion, CargoItemVersion: allocation.CargoItemVersion})
	}
	if item.Progress != nil {
		result.Progress = &v1.SeaSharedContainerProgress{AllocatedPackageCount: item.Progress.AllocatedPackageCount, AllocatedGrossWeightKg: item.Progress.AllocatedGrossWeightKg.StringFixed(3), AllocatedVolumeCbm: item.Progress.AllocatedVolumeCbm.StringFixed(6), RemainingPackageCount: item.Progress.RemainingPackageCount, RemainingGrossWeightKg: item.Progress.RemainingGrossWeightKg.StringFixed(3), RemainingVolumeCbm: item.Progress.RemainingVolumeCbm.StringFixed(6), ContainerBalanced: item.Progress.ContainerBalanced, CargoBalanced: item.Progress.CargoBalanced}
	}
	return result
}

func seaSharedContainerCandidateToAPI(item *biz.SeaSharedContainerCandidateOrder) *v1.SeaSharedContainerCandidateOrder {
	result := &v1.SeaSharedContainerCandidateOrder{OrderId: item.OrderID.String(), OrderNo: item.OrderNo, HouseBillId: item.HouseBillID.String(), HouseNo: item.HouseNo, OrderVersion: item.OrderVersion, LinkVersion: item.LinkVersion, HouseBillVersion: item.HouseBillVersion}
	for _, cargo := range item.CargoItems {
		result.CargoItems = append(result.CargoItems, &v1.SeaSharedContainerCandidateCargoItem{Id: cargo.ID.String(), CargoName: cargo.CargoName, PackageCount: cargo.PackageCount, GrossWeightKg: cargo.GrossWeightKg.StringFixed(3), VolumeCbm: cargo.VolumeCbm.StringFixed(6), Version: cargo.Version})
	}
	return result
}

func seaSharedContainerStatusToAPI(status biz.SeaSharedContainerStatus) v1.SeaSharedContainerStatus {
	switch status {
	case biz.SeaSharedContainerStatusDraft:
		return v1.SeaSharedContainerStatus_SEA_SHARED_CONTAINER_STATUS_DRAFT
	case biz.SeaSharedContainerStatusConfirmed:
		return v1.SeaSharedContainerStatus_SEA_SHARED_CONTAINER_STATUS_CONFIRMED
	default:
		return v1.SeaSharedContainerStatus_SEA_SHARED_CONTAINER_STATUS_UNSPECIFIED
	}
}
