package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// OrderCargoItemService 订单货物明细服务，只做 DTO 转换、边界校验和用例调用。
type OrderCargoItemService struct {
	v1.UnimplementedOrderCargoItemServiceServer
	usecase *biz.OrderCargoItemUsecase
}

func NewOrderCargoItemService(usecase *biz.OrderCargoItemUsecase) *OrderCargoItemService {
	return &OrderCargoItemService{usecase: usecase}
}

func (s *OrderCargoItemService) ListCargoItems(ctx context.Context, request *v1.ListCargoItemsRequest) (*v1.ListCargoItemsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderCargoItemInvalidArgument
	}
	items, err := s.usecase.List(ctx, principal.Organization.ID, orderID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.OrderCargoItem, 0, len(items))
	for _, item := range items {
		data = append(data, orderCargoItemToAPI(item))
	}
	return okList(ctx, &v1.ListCargoItemsResponse{
		Data: data,
	}), nil
}

func (s *OrderCargoItemService) AddCargoItem(ctx context.Context, request *v1.AddCargoItemRequest) (*v1.AddCargoItemResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, input, err := orderCargoItemInputFromAPI(request.GetOrderId(), request.GetCargoName(), request.GetPackageCount(), request.GetGrossWeightKg(), request.GetVolumeCbm(), request.NetWeightKg, request.GetNote())
	if err != nil {
		return nil, err
	}
	created, err := s.usecase.Add(ctx, principal.Organization.ID, principal.UserID, orderID, input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.AddCargoItemResponse{Data: orderCargoItemToAPI(created)}), nil
}

func (s *OrderCargoItemService) UpdateCargoItem(ctx context.Context, request *v1.UpdateCargoItemRequest) (*v1.UpdateCargoItemResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderCargoItemInvalidArgument
	}
	orderID, input, err := orderCargoItemInputFromAPI(request.GetOrderId(), request.GetCargoName(), request.GetPackageCount(), request.GetGrossWeightKg(), request.GetVolumeCbm(), request.NetWeightKg, request.GetNote())
	if err != nil {
		return nil, err
	}
	updated, err := s.usecase.Update(ctx, principal.Organization.ID, principal.UserID, orderID, id, request.GetExpectedVersion(), input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.UpdateCargoItemResponse{Data: orderCargoItemToAPI(updated)}), nil
}

func (s *OrderCargoItemService) RemoveCargoItem(ctx context.Context, request *v1.RemoveCargoItemRequest) (*v1.RemoveCargoItemResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderCargoItemInvalidArgument
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderCargoItemInvalidArgument
	}
	if err := s.usecase.Remove(ctx, principal.Organization.ID, principal.UserID, orderID, id, request.GetExpectedVersion()); err != nil {
		return nil, err
	}
	return ok(ctx, &v1.RemoveCargoItemResponse{}), nil
}

func orderCargoItemToAPI(value *biz.OrderCargoItem) *v1.OrderCargoItem {
	return &v1.OrderCargoItem{
		Id:            value.ID.String(),
		OrderId:       value.OrderID.String(),
		CargoName:     value.CargoName,
		PackageCount:  int32(value.PackageCount),
		GrossWeightKg: value.GrossWeightKg,
		VolumeCbm:     value.VolumeCbm,
		NetWeightKg:   value.NetWeightKg,
		Note:          value.Note,
		Version:       value.Version,
		CreatedAt:     value.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     value.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func orderCargoItemInputFromAPI(orderIDText, cargoName string, packageCount int32, grossWeightKg, volumeCbm float64, netWeightKg *float64, note string) (uuid.UUID, *biz.OrderCargoItem, error) {
	orderID, err := uuid.Parse(orderIDText)
	if err != nil {
		return uuid.Nil, nil, biz.ErrOrderCargoItemInvalidArgument
	}
	item := &biz.OrderCargoItem{
		CargoName:     cargoName,
		PackageCount:  int(packageCount),
		GrossWeightKg: grossWeightKg,
		VolumeCbm:     volumeCbm,
	}
	if netWeightKg != nil {
		v := *netWeightKg
		item.NetWeightKg = &v
	}
	if note != "" {
		v := note
		item.Note = &v
	}
	return orderID, item, nil
}
