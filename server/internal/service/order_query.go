package service

import (
	"context"
	"strings"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"

	"github.com/google/uuid"
)

func (s *OrderService) GetOrder(ctx context.Context, request *v1.GetOrderRequest) (*v1.GetOrderResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderNotFound
	}
	item, err := s.usecase.Get(ctx, principal.Organization.ID, id)
	if err != nil {
		return nil, err
	}
	return &v1.GetOrderResponse{Success: true, Code: 0, Message: "OK", Data: orderToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderService) ListOrders(ctx context.Context, request *v1.ListOrdersRequest) (*v1.ListOrdersResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	page, pageSize, err := pageValues(request.GetPage(), request.GetPageSize())
	if err != nil {
		return nil, err
	}
	options := biz.OrderListOptions{Page: page, PageSize: pageSize, Keyword: request.GetKeyword()}
	if request.FlowStatus != nil {
		options.FlowStatus = orderFlowStatusFromAPI(request.GetFlowStatus())
	}
	if request.TerminationStatus != nil {
		options.TerminationStatus = orderTerminationStatusFromAPI(request.GetTerminationStatus())
	}
	if request.ClosureStatus != nil {
		options.ClosureStatus = orderClosureStatusFromAPI(request.GetClosureStatus())
	}
	if request.HasActiveException != nil {
		value := request.GetHasActiveException()
		options.HasActiveException = &value
	}
	if request.BusinessType != nil {
		options.BusinessType = orderBusinessTypeFromAPI(request.GetBusinessType())
	} else {
		options.BusinessTypes = readableOrderBusinessTypes(principal)
	}
	if request.GetCustomerId() != "" {
		value, parseErr := uuid.Parse(request.GetCustomerId())
		if parseErr != nil {
			return nil, biz.ErrOrderInvalidArgument
		}
		options.CustomerID = &value
	}
	if request.NumberType != nil {
		switch request.GetNumberType() {
		case v1.OrderNumberFilterType_ORDER_NUMBER_FILTER_TYPE_ORDER:
			options.NumberType = biz.OrderNumberFilterOrder
		case v1.OrderNumberFilterType_ORDER_NUMBER_FILTER_TYPE_MASTER:
			options.NumberType = biz.OrderNumberFilterMaster
		case v1.OrderNumberFilterType_ORDER_NUMBER_FILTER_TYPE_CONSOLIDATED_MASTER:
			options.NumberType = biz.OrderNumberFilterConsolidatedMaster
		default:
			return nil, biz.ErrOrderInvalidArgument
		}
		options.NumberKeyword = request.GetNumberKeyword()
	} else if strings.TrimSpace(request.GetNumberKeyword()) != "" {
		return nil, biz.ErrOrderInvalidArgument
	}
	if options.CreatedAtRange, err = orderDateRangeFromAPI(request.GetCreatedAtFrom(), request.GetCreatedAtTo()); err != nil {
		return nil, err
	}
	if options.ETDRange, err = orderDateRangeFromAPI(request.GetEtdFrom(), request.GetEtdTo()); err != nil {
		return nil, err
	}
	if options.ETARange, err = orderDateRangeFromAPI(request.GetEtaFrom(), request.GetEtaTo()); err != nil {
		return nil, err
	}
	if options.StatusTimeRange, err = orderDateRangeFromAPI(request.GetStatusTimeFrom(), request.GetStatusTimeTo()); err != nil {
		return nil, err
	}
	if options.LockedAtRange, err = orderDateRangeFromAPI(request.GetLockedAtFrom(), request.GetLockedAtTo()); err != nil {
		return nil, err
	}
	if options.OriginLocationID, err = listOptionalUUID(request.GetOriginLocationId()); err != nil {
		return nil, err
	}
	if options.DestinationLocationID, err = listOptionalUUID(request.GetDestinationLocationId()); err != nil {
		return nil, err
	}
	if options.CarrierID, err = listOptionalUUID(request.GetCarrierId()); err != nil {
		return nil, err
	}
	options.ConsigneeShortName = request.GetConsigneeShortName()
	options.ShipperShortName = request.GetShipperShortName()
	if options.Operator, err = orderPersonnelFilterFromAPI(request.GetOperatorId(), request.GetOperatorOrganizationId()); err != nil {
		return nil, err
	}
	if options.Sales, err = orderPersonnelFilterFromAPI(request.GetSalesId(), request.GetSalesOrganizationId()); err != nil {
		return nil, err
	}
	if options.CustomerService, err = orderPersonnelFilterFromAPI(request.GetCustomerServiceId(), request.GetCustomerServiceOrganizationId()); err != nil {
		return nil, err
	}
	if options.Creator, err = orderPersonnelFilterFromAPI(request.GetCreatorId(), request.GetCreatorOrganizationId()); err != nil {
		return nil, err
	}
	options.Tags = request.GetTags()
	if request.TagMatchMode != nil {
		switch request.GetTagMatchMode() {
		case v1.OrderTagMatchMode_ORDER_TAG_MATCH_MODE_FUZZY_OR:
			options.TagMatchMode = biz.OrderTagMatchFuzzyOr
		case v1.OrderTagMatchMode_ORDER_TAG_MATCH_MODE_EXACT_AND:
			options.TagMatchMode = biz.OrderTagMatchExactAnd
		default:
			return nil, biz.ErrOrderInvalidArgument
		}
	}
	if request.IsLocked != nil {
		value := request.GetIsLocked()
		options.IsLocked = &value
	}
	if request.IsShared != nil {
		value := request.GetIsShared()
		options.IsShared = &value
	}
	result, err := s.usecase.List(ctx, principal.OrderOrganizationIDs(), options)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.Order, 0, len(result.Items))
	for _, item := range result.Items {
		output := orderToAPI(item)
		output.CanModify = principal.CanAccessOrderOrganization(item.OrganizationID, true)
		data = append(data, output)
	}
	return &v1.ListOrdersResponse{Success: true, Code: 0, Message: "OK", Data: data, Total: int32(result.Total), Page: int32(result.Page), PageSize: int32(result.PageSize), TraceId: requestmeta.TraceID(ctx)}, nil
}

func orderDateRangeFromAPI(from, to string) (biz.OrderDateRange, error) {
	var result biz.OrderDateRange
	if strings.TrimSpace(from) != "" {
		value, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(from), orderListDateLocation)
		if err != nil {
			return result, biz.ErrOrderInvalidArgument
		}
		result.From = &value
	}
	if strings.TrimSpace(to) != "" {
		value, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(to), orderListDateLocation)
		if err != nil {
			return result, biz.ErrOrderInvalidArgument
		}
		value = value.AddDate(0, 0, 1)
		result.ToExclusive = &value
	}
	if result.From != nil && result.ToExclusive != nil && !result.From.Before(*result.ToExclusive) {
		return result, biz.ErrOrderInvalidArgument
	}
	return result, nil
}

func listOptionalUUID(value string) (*uuid.UUID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, biz.ErrOrderInvalidArgument
	}
	return &id, nil
}

func orderPersonnelFilterFromAPI(userID, organizationID string) (biz.OrderPersonnelFilter, error) {
	user, err := listOptionalUUID(userID)
	if err != nil {
		return biz.OrderPersonnelFilter{}, err
	}
	organization, err := listOptionalUUID(organizationID)
	if err != nil {
		return biz.OrderPersonnelFilter{}, err
	}
	if user == nil && organization != nil {
		return biz.OrderPersonnelFilter{}, biz.ErrOrderInvalidArgument
	}
	return biz.OrderPersonnelFilter{UserID: user, OrganizationID: organization}, nil
}

func readableOrderBusinessTypes(principal *biz.Principal) []biz.OrderBusinessType {
	types := []struct {
		access access.OrderBusinessType
		biz    biz.OrderBusinessType
	}{
		{access: access.OrderBusinessSE, biz: biz.OrderBusinessSE},
	}
	result := make([]biz.OrderBusinessType, 0, len(types))
	for _, businessType := range types {
		if principal.HasPermissionInScope(access.OrderPermission(businessType.access, access.OrderRead), biz.DataScopeOrganization) {
			result = append(result, businessType.biz)
		}
	}
	return result
}

func (s *OrderService) CheckOrderReference(ctx context.Context, request *v1.CheckOrderReferenceRequest) (*v1.CheckOrderReferenceResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	check := biz.OrderReferenceCheck{ReferenceNo: request.GetReferenceNo()}
	switch request.GetReferenceType() {
	case v1.OrderReferenceType_ORDER_REFERENCE_TYPE_CUSTOMER:
		check.ReferenceType = biz.OrderReferenceCustomer
		customerID, err := uuid.Parse(request.GetCustomerId())
		if err != nil {
			return nil, biz.ErrOrderInvalidArgument
		}
		check.CustomerID = &customerID
	case v1.OrderReferenceType_ORDER_REFERENCE_TYPE_INTERNAL:
		check.ReferenceType = biz.OrderReferenceInternal
	default:
		return nil, biz.ErrOrderInvalidArgument
	}
	if request.GetExcludeOrderId() != "" {
		excludeOrderID, err := uuid.Parse(request.GetExcludeOrderId())
		if err != nil {
			return nil, biz.ErrOrderInvalidArgument
		}
		check.ExcludeOrderID = &excludeOrderID
	}
	match, err := s.usecase.CheckReference(ctx, principal.Organization.ID, check)
	if err != nil {
		return nil, err
	}
	data := &v1.OrderReferenceCheck{Duplicate: match != nil}
	if match != nil {
		orderID := match.OrderID.String()
		data.OrderId = &orderID
		data.OrderNo = &match.OrderNo
	}
	return &v1.CheckOrderReferenceResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderService) ListPersonnelOptions(ctx context.Context, request *v1.ListPersonnelOptionsRequest) (*v1.ListPersonnelOptionsResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	page, pageSize := biz.ListPagination(int(request.GetPage()), int(request.GetPageSize()), 20)
	result, err := s.usecase.ListPersonnelOptions(ctx, principal.Organization.ID, biz.SelectorListOptions{Keyword: request.GetKeyword(), Page: page, PageSize: pageSize})
	if err != nil {
		return nil, err
	}
	data := make([]*v1.OrderPersonnelOption, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, &v1.OrderPersonnelOption{
			UserId: item.UserID.String(), DisplayName: item.DisplayName,
			OrganizationId: item.OrganizationID.String(), OrganizationName: item.OrganizationName,
		})
	}
	return &v1.ListPersonnelOptionsResponse{Success: true, Code: 0, Message: "OK", Data: data, Total: int32(result.Total), Page: int32(result.Page), PageSize: int32(result.PageSize), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderService) ListOrderConsolidations(ctx context.Context, request *v1.ListOrderConsolidationsRequest) (*v1.ListOrderConsolidationsResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	orderID, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderInvalidArgument
	}
	items, err := s.usecase.ListConsolidationSummaries(ctx, principal.Organization.ID, orderID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.OrderConsolidationSummary, 0, len(items))
	for _, item := range items {
		members := make([]*v1.OrderConsolidationMember, 0, len(item.Members))
		for _, member := range item.Members {
			members = append(members, &v1.OrderConsolidationMember{
				OrderId: member.OrderID.String(), OrderNo: member.OrderNo, CustomerReferenceNo: stringPtrIfNotEmpty(member.CustomerReferenceNo), HouseNos: member.HouseNos,
				Entrusted: cargoMeasurementToAPI(member.Entrusted), Actual: cargoMeasurementToAPI(member.Actual),
			})
		}
		data = append(data, &v1.OrderConsolidationSummary{
			ConsolidationId: item.ConsolidationID.String(), MasterNo: item.MasterNo, MemberCount: int32(len(item.Members)),
			Entrusted: cargoMeasurementToAPI(item.Entrusted), Actual: cargoMeasurementToAPI(item.Actual), Members: members,
		})
	}
	return &v1.ListOrderConsolidationsResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func cargoMeasurementToAPI(value biz.OrderCargoMeasurement) *v1.OrderCargoMeasurement {
	return &v1.OrderCargoMeasurement{Packages: int32(value.Packages), GrossWeightKg: value.GrossWeightKg, VolumeCbm: value.VolumeCbm}
}
