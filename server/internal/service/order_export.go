package service

import (
	"context"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func (s *OrderService) ExportOrders(ctx context.Context, request *v1.ExportOrdersRequest) (*v1.ExportOrdersResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	businessType := orderBusinessTypeFromAPI(request.GetBusinessType())
	if businessType != biz.OrderBusinessSE && businessType != biz.OrderBusinessSI && businessType != biz.OrderBusinessAE && businessType != biz.OrderBusinessAI {
		return nil, biz.ErrOrderInvalidArgument
	}
	// 两个顶层Proto请求独立，筛选的边界转换与权限解析仍走唯一入口。
	filter := &v1.ListOrdersRequest{
		Keyword:               request.Keyword,
		FlowStatus:            request.FlowStatus,
		BusinessType:          &request.BusinessType,
		CustomerId:            request.CustomerId,
		TerminationStatus:     request.TerminationStatus,
		ClosureStatus:         request.ClosureStatus,
		HasActiveException:    request.HasActiveException,
		NumberType:            request.NumberType,
		NumberKeyword:         request.NumberKeyword,
		CreatedAtFrom:         request.CreatedAtFrom,
		CreatedAtTo:           request.CreatedAtTo,
		EtdFrom:               request.EtdFrom,
		EtdTo:                 request.EtdTo,
		EtaFrom:               request.EtaFrom,
		EtaTo:                 request.EtaTo,
		StatusTimeFrom:        request.StatusTimeFrom,
		StatusTimeTo:          request.StatusTimeTo,
		LockedAtFrom:          request.LockedAtFrom,
		LockedAtTo:            request.LockedAtTo,
		OriginLocationId:      request.OriginLocationId,
		DestinationLocationId: request.DestinationLocationId,
		ShippingLineId:        request.ShippingLineId,
		ConsigneeShortName:    request.ConsigneeShortName,
		ShipperShortName:      request.ShipperShortName,
		OperatorId:            request.OperatorId,
		SalesId:               request.SalesId,
		CustomerServiceId:     request.CustomerServiceId,
		CreatorId:             request.CreatorId,
		IsLocked:              request.IsLocked,
		IsShared:              request.IsShared,
		SortBy:                request.SortBy,
		SortOrder:             request.SortOrder,
	}
	options, scopes, err := orderListOptionsFromAPI(principal, filter)
	if err != nil {
		return nil, err
	}
	result, err := s.usecase.Export(ctx, scopes, options)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.Order, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, orderToAPIForPrincipal(principal, item))
	}
	return okList(ctx, &v1.ExportOrdersResponse{Data: data, Total: int32(len(data)), GeneratedAt: result.GeneratedAt.Format(time.RFC3339Nano)}), nil
}
