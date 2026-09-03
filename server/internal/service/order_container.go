package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// OrderContainerService 订单集装箱服务，只做 DTO 转换、边界校验和用例调用。
type OrderContainerService struct {
	v1.UnimplementedOrderContainerServiceServer
	usecase *biz.OrderContainerUsecase
}

func NewOrderContainerService(usecase *biz.OrderContainerUsecase) *OrderContainerService {
	return &OrderContainerService{usecase: usecase}
}

func (s *OrderContainerService) ListContainers(ctx context.Context, request *v1.ListContainersRequest) (*v1.ListContainersResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderContainerInvalidArgument
	}
	items, err := s.usecase.List(ctx, principal.Organization.ID, orderID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.OrderContainer, 0, len(items))
	for _, item := range items {
		data = append(data, orderContainerToAPI(item))
	}
	return okList(ctx, &v1.ListContainersResponse{
		Data: data,
	}), nil
}

func (s *OrderContainerService) AddContainer(ctx context.Context, request *v1.AddContainerRequest) (*v1.AddContainerResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, input, err := orderContainerInputFromAPI(request.GetOrderId(), request.GetContainerSpecId(), request.GetContainerNo(), request.GetPackageCount(), request.GetSealNo(), request.GetGrossWeightKg(), request.GetVolumeCbm(), request.GetNote())
	if err != nil {
		return nil, err
	}
	created, err := s.usecase.Add(ctx, principal.Organization.ID, principal.UserID, orderID, input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.AddContainerResponse{Data: orderContainerToAPI(created)}), nil
}

func (s *OrderContainerService) UpdateContainer(ctx context.Context, request *v1.UpdateContainerRequest) (*v1.UpdateContainerResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderContainerInvalidArgument
	}
	orderID, input, err := orderContainerInputFromAPI(request.GetOrderId(), request.GetContainerSpecId(), request.GetContainerNo(), request.GetPackageCount(), request.GetSealNo(), request.GetGrossWeightKg(), request.GetVolumeCbm(), request.GetNote())
	if err != nil {
		return nil, err
	}
	updated, err := s.usecase.Update(ctx, principal.Organization.ID, principal.UserID, orderID, id, request.GetExpectedVersion(), input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.UpdateContainerResponse{Data: orderContainerToAPI(updated)}), nil
}

func (s *OrderContainerService) RemoveContainer(ctx context.Context, request *v1.RemoveContainerRequest) (*v1.RemoveContainerResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderContainerInvalidArgument
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderContainerInvalidArgument
	}
	if err := s.usecase.Remove(ctx, principal.Organization.ID, principal.UserID, orderID, id, request.GetExpectedVersion()); err != nil {
		return nil, err
	}
	return ok(ctx, &v1.RemoveContainerResponse{}), nil
}

func orderContainerToAPI(value *biz.OrderContainer) *v1.OrderContainer {
	return &v1.OrderContainer{
		Id:              value.ID.String(),
		OrderId:         value.OrderID.String(),
		ContainerNo:     value.ContainerNo,
		ContainerSpecId: value.ContainerSpecID.String(),
		PackageCount:    value.PackageCount,
		SealNo:          value.SealNo,
		GrossWeightKg:   value.GrossWeightKg,
		VolumeCbm:       value.VolumeCbm,
		Note:            value.Note,
		Version:         value.Version,
		CreatedAt:       value.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       value.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func orderContainerInputFromAPI(orderIDText, specIDText, containerNo string, packageCount int32, sealNo string, grossWeightKg, volumeCbm float64, note string) (uuid.UUID, *biz.OrderContainer, error) {
	orderID, err := uuid.Parse(orderIDText)
	if err != nil {
		return uuid.Nil, nil, biz.ErrOrderContainerInvalidArgument
	}
	specID, err := uuid.Parse(specIDText)
	if err != nil {
		return uuid.Nil, nil, biz.ErrOrderContainerInvalidArgument
	}
	item := &biz.OrderContainer{
		ContainerNo:     containerNo,
		ContainerSpecID: specID,
		PackageCount:    packageCount,
		GrossWeightKg:   grossWeightKg,
		VolumeCbm:       volumeCbm,
	}
	if sealNo != "" {
		v := sealNo
		item.SealNo = &v
	}
	if note != "" {
		v := note
		item.Note = &v
	}
	return orderID, item, nil
}
