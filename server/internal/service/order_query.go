package service

import (
	"context"
	"strings"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"

	"github.com/google/uuid"
)

func (s *OrderService) MatchSeaMasterBillCandidate(ctx context.Context, request *v1.MatchSeaMasterBillCandidateRequest) (*v1.MatchSeaMasterBillCandidateResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	shippingLineID, err := uuid.Parse(request.GetShippingLineId())
	if err != nil {
		return nil, biz.ErrSeaMasterBillInvalidArgument
	}

	etd, err := parseSeaCandidateTime(request.Etd)
	if err != nil {
		return nil, err
	}
	eta, err := parseSeaCandidateTime(request.Eta)
	if err != nil {
		return nil, err
	}
	orderVoyage := &biz.SeaTransportExecution{
		ShippingLineID: shippingLineID,
		VesselName:     strings.TrimSpace(request.GetVesselName()),
		VoyageNo:       strings.TrimSpace(request.GetVoyageNo()),
		ETD:            etd,
		ETA:            eta,
	}
	originLocationID, err := parseOptionalUUIDPointer(request.OriginLocationId)
	if err != nil {
		return nil, biz.ErrSeaMasterBillInvalidArgument
	}
	if originLocationID != nil {
		orderVoyage.OriginLocationID = *originLocationID
	}
	dischargeLocationID, err := parseOptionalUUIDPointer(request.DischargeLocationId)
	if err != nil {
		return nil, biz.ErrSeaMasterBillInvalidArgument
	}
	if dischargeLocationID != nil {
		orderVoyage.DischargeLocationID = *dischargeLocationID
	}
	transitLocationID, err := parseOptionalUUIDPointer(request.TransitLocationId)
	if err != nil {
		return nil, biz.ErrSeaMasterBillInvalidArgument
	}
	if transitLocationID != nil {
		orderVoyage.TransitLocationID = transitLocationID
	}

	result, err := s.usecase.MatchSeaMasterBillCandidate(ctx, principal.Organization.ID, shippingLineID, request.GetMasterNo(), orderVoyage)
	if err != nil {
		return nil, err
	}

	resp := &v1.MatchSeaMasterBillCandidateResponse{
		Matched: result.Matched,
	}
	if result.Matched && result.Candidate != nil {
		resp.Candidate = seaMasterBillCandidateToAPI(result.Candidate)
		for _, c := range result.Conflicts {
			resp.Conflicts = append(resp.Conflicts, &v1.SeaVoyageConflict{
				Field:       c.Field,
				MasterValue: c.MasterValue,
				OrderValue:  c.OrderValue,
				Message:     c.Message,
			})
		}
	}

	return ok(ctx, resp), nil
}

func parseSeaCandidateTime(value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	parsed := parseOptionalTime(value)
	if parsed == nil {
		return nil, biz.ErrSeaMasterBillInvalidArgument
	}
	return parsed, nil
}

func (s *OrderService) GetOrder(ctx context.Context, request *v1.GetOrderRequest) (*v1.GetOrderResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderNotFound
	}
	item, err := s.usecase.Get(ctx, principal.Organization.ID, id)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.GetOrderResponse{Data: orderToAPI(item)}), nil
}

func (s *OrderService) ListOrders(ctx context.Context, request *v1.ListOrdersRequest) (*v1.ListOrdersResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrOrderInvalidArgument)
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
	}
	orderScopes, err := orderOrganizationScopesForOperation(principal, access.OrderRead, false, options.BusinessType)
	if err != nil {
		return nil, err
	}
	if options.BusinessType == "" {
		options.BusinessTypes = orderBusinessTypesFromScopes(orderScopes)
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
		case v1.OrderNumberFilterType_ORDER_NUMBER_FILTER_TYPE_CUSTOMER_REFERENCE:
			options.NumberType = biz.OrderNumberFilterCustomerReference
		case v1.OrderNumberFilterType_ORDER_NUMBER_FILTER_TYPE_BOOKING:
			options.NumberType = biz.OrderNumberFilterBooking
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
	if options.ShippingLineID, err = listOptionalUUID(request.GetShippingLineId()); err != nil {
		return nil, err
	}
	options.ConsigneeShortName = request.GetConsigneeShortName()
	options.ShipperShortName = request.GetShipperShortName()
	if options.Operator, err = orderPersonnelFilterFromAPI(request.GetOperatorId()); err != nil {
		return nil, err
	}
	if options.Sales, err = orderPersonnelFilterFromAPI(request.GetSalesId()); err != nil {
		return nil, err
	}
	if options.CustomerService, err = orderPersonnelFilterFromAPI(request.GetCustomerServiceId()); err != nil {
		return nil, err
	}
	if options.Creator, err = orderPersonnelFilterFromAPI(request.GetCreatorId()); err != nil {
		return nil, err
	}
	tagIDs, err := parseUUIDValues(request.GetTagIds(), biz.ErrOrderInvalidArgument)
	if err != nil {
		return nil, err
	}
	options.TagIDs = tagIDs
	if request.IsLocked != nil {
		value := request.GetIsLocked()
		options.IsLocked = &value
	}
	if request.IsShared != nil {
		value := request.GetIsShared()
		options.IsShared = &value
	}
	result, err := s.usecase.List(ctx, orderScopes, options)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.Order, 0, len(result.Items))
	for _, item := range result.Items {
		output := orderToAPI(item)
		output.CanModify = canModifyOrder(principal, item)
		data = append(data, output)
	}
	// 海运出口订单页批量附加提成摘要：一次性把本页订单 ID 交给提成用例，
	// 不逐行发起查询；可见模式与隐私裁剪由服务端完成，详情接口不附加。
	if err := s.attachListCommissionSummaries(ctx, principal, result.Items, data); err != nil {
		return nil, err
	}
	return okList(ctx, &v1.ListOrdersResponse{Data: data, Total: int32(result.Total), Page: int32(result.Page), PageSize: int32(result.PageSize)}), nil
}

// attachListCommissionSummaries 为已授权海运出口订单页批量附加提成摘要：
// 仅收集本页海运出口订单，调用提成用例按组织逐个判定可见模式后批量聚合，
// 并把领域投影转换为 API 结构；非海运出口订单不携带摘要。
func (s *OrderService) attachListCommissionSummaries(ctx context.Context, principal *biz.Principal, items []*biz.Order, outputs []*v1.Order) error {
	if s.commission == nil || len(items) != len(outputs) {
		return nil
	}
	targets := make([]biz.OrderCommissionSummaryTarget, 0, len(items))
	outputByOrderID := make(map[uuid.UUID]*v1.Order, len(items))
	for index, item := range items {
		if item.BusinessType != biz.OrderBusinessSE {
			continue
		}
		targets = append(targets, biz.OrderCommissionSummaryTarget{OrderID: item.ID, OrganizationID: item.OrganizationID})
		outputByOrderID[item.ID] = outputs[index]
	}
	if len(targets) == 0 {
		return nil
	}
	summaries, err := s.commission.BuildOrderListSummaries(ctx, principal, targets)
	if err != nil {
		return err
	}
	for orderID, summary := range summaries {
		output, exists := outputByOrderID[orderID]
		if !exists || summary == nil {
			continue
		}
		output.CommissionSummary = orderCommissionSummaryToAPI(summary)
	}
	return nil
}

// orderCommissionSummaryToAPI 转换提成摘要领域投影：金额为组织本位币口径的
// StringFixed(8)，仅在对应事实存在时输出；空态不携带金额字段。
func orderCommissionSummaryToAPI(summary *biz.OrderCommissionSummary) *v1.OrderCommissionSummary {
	result := &v1.OrderCommissionSummary{BaseCurrency: summary.BaseCurrency}
	switch summary.Visibility {
	case biz.OrderCommissionVisibilityOrganization:
		result.VisibilityMode = v1.OrderCommissionVisibilityMode_ORDER_COMMISSION_VISIBILITY_MODE_ORGANIZATION
	case biz.OrderCommissionVisibilityEmployee:
		result.VisibilityMode = v1.OrderCommissionVisibilityMode_ORDER_COMMISSION_VISIBILITY_MODE_EMPLOYEE
	default:
		result.VisibilityMode = v1.OrderCommissionVisibilityMode_ORDER_COMMISSION_VISIBILITY_MODE_UNSPECIFIED
	}
	result.HasExpectedOpportunity = summary.HasExpectedOpportunity
	result.ExpectedOpportunityCount = int32(summary.ExpectedOpportunityCount)
	result.HasDraftCommission = summary.HasDraftCommission
	result.DraftCommissionCount = int32(summary.DraftCommissionCount)
	if summary.HasDraftCommission {
		amount := summary.DraftCommissionAmount.StringFixed(8)
		result.DraftCommissionAmount = &amount
	}
	result.HasConfirmedCommission = summary.HasConfirmedCommission
	result.ConfirmedCommissionCount = int32(summary.ConfirmedCommissionCount)
	if summary.HasConfirmedCommission {
		amount := summary.ConfirmedCommissionAmount.StringFixed(8)
		result.ConfirmedCommissionAmount = &amount
	}
	result.HasPaidCommission = summary.HasPaidCommission
	result.PaidCommissionCount = int32(summary.PaidCommissionCount)
	if summary.HasPaidCommission {
		amount := summary.PaidCommissionAmount.StringFixed(8)
		result.PaidCommissionAmount = &amount
	}
	result.HasPendingDecrease = summary.HasPendingDecrease
	result.PendingDecreaseCount = int32(summary.PendingDecreaseCount)
	if summary.HasPendingDecrease {
		amount := summary.PendingDecreaseAmount.StringFixed(8)
		result.PendingDecreaseAmount = &amount
	}
	return result
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

func orderPersonnelFilterFromAPI(userID string) (biz.OrderPersonnelFilter, error) {
	user, err := listOptionalUUID(userID)
	if err != nil {
		return biz.OrderPersonnelFilter{}, err
	}
	return biz.OrderPersonnelFilter{UserID: user}, nil
}

func orderOrganizationScopesForOperation(principal *biz.Principal, operation access.OrderOperation, writable bool, onlyBusinessType biz.OrderBusinessType) ([]biz.OrderOrganizationScope, error) {
	if principal == nil {
		return nil, biz.ErrPermissionDenied
	}
	scopes := make([]biz.OrderOrganizationScope, 0, len(access.OrderBusinessTypes()))
	for _, accessBusinessType := range access.OrderBusinessTypes() {
		businessType, ok := orderBusinessTypeFromAccess(accessBusinessType)
		if !ok || onlyBusinessType != "" && businessType != onlyBusinessType {
			continue
		}
		permission := access.OrderPermission(accessBusinessType, operation)
		if permission == "" {
			continue
		}
		permissionScope, err := principal.ResolvePermissionOrganizationScope(permission)
		if err != nil {
			continue
		}
		organizationIDs := permissionScope.ReadableOrganizationIDs
		if writable {
			organizationIDs = permissionScope.WritableOrganizationIDs
		}
		if len(organizationIDs) == 0 {
			continue
		}
		scopes = append(scopes, biz.OrderOrganizationScope{BusinessType: businessType, OrganizationIDs: organizationIDs})
	}
	if len(scopes) == 0 {
		return nil, biz.ErrPermissionDenied
	}
	return scopes, nil
}

func orderBusinessTypesFromScopes(scopes []biz.OrderOrganizationScope) []biz.OrderBusinessType {
	result := make([]biz.OrderBusinessType, 0, len(scopes))
	for _, scope := range scopes {
		result = append(result, scope.BusinessType)
	}
	return result
}

func canModifyOrder(principal *biz.Principal, order *biz.Order) bool {
	if order == nil {
		return false
	}
	accessBusinessType, ok := orderBusinessTypeToAccess(order.BusinessType)
	if !ok {
		return false
	}
	permission := access.OrderPermission(accessBusinessType, access.OrderUpdate)
	return permission != "" && principal.CanAccessOrganizationForPermission(permission, order.OrganizationID, true)
}

func canOperateOrderInCurrentOrganization(principal *biz.Principal, businessType biz.OrderBusinessType, operation access.OrderOperation, writable bool) bool {
	accessBusinessType, ok := orderBusinessTypeToAccess(businessType)
	if !ok || principal == nil {
		return false
	}
	permission := access.OrderPermission(accessBusinessType, operation)
	return permission != "" && principal.CanAccessOrganizationForPermission(permission, principal.Organization.ID, writable)
}

func orderBusinessTypeFromAccess(value access.OrderBusinessType) (biz.OrderBusinessType, bool) {
	switch value {
	case access.OrderBusinessSE:
		return biz.OrderBusinessSE, true
	case access.OrderBusinessSI:
		return biz.OrderBusinessSI, true
	case access.OrderBusinessAE:
		return biz.OrderBusinessAE, true
	case access.OrderBusinessAI:
		return biz.OrderBusinessAI, true
	case access.OrderBusinessLand:
		return biz.OrderBusinessLand, true
	case access.OrderBusinessRail:
		return biz.OrderBusinessRail, true
	default:
		return "", false
	}
}

func orderBusinessTypeToAccess(value biz.OrderBusinessType) (access.OrderBusinessType, bool) {
	switch value {
	case biz.OrderBusinessSE:
		return access.OrderBusinessSE, true
	case biz.OrderBusinessSI:
		return access.OrderBusinessSI, true
	case biz.OrderBusinessAE:
		return access.OrderBusinessAE, true
	case biz.OrderBusinessAI:
		return access.OrderBusinessAI, true
	case biz.OrderBusinessLand:
		return access.OrderBusinessLand, true
	case biz.OrderBusinessRail:
		return access.OrderBusinessRail, true
	default:
		return "", false
	}
}

func (s *OrderService) CheckOrderReference(ctx context.Context, request *v1.CheckOrderReferenceRequest) (*v1.CheckOrderReferenceResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
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
	case v1.OrderReferenceType_ORDER_REFERENCE_TYPE_BOOKING:
		check.ReferenceType = biz.OrderReferenceBooking
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
	return ok(ctx, &v1.CheckOrderReferenceResponse{Data: data}), nil
}

func (s *OrderService) ListPersonnelOptions(ctx context.Context, request *v1.ListPersonnelOptionsRequest) (*v1.ListPersonnelOptionsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrOrderInvalidArgument)
	if err != nil {
		return nil, err
	}
	result, err := s.usecase.ListPersonnelOptions(ctx, principal.Organization.ID, biz.SelectorListOptions{Keyword: request.GetKeyword(), Page: page, PageSize: pageSize})
	if err != nil {
		return nil, err
	}
	data := make([]*v1.OrderPersonnelOption, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, &v1.OrderPersonnelOption{
			UserId: item.UserID.String(), DisplayName: item.DisplayName,
		})
	}
	return okList(ctx, &v1.ListPersonnelOptionsResponse{Data: data, Total: int32(result.Total), Page: int32(result.Page), PageSize: int32(result.PageSize)}), nil
}

func (s *OrderService) ListOrderConsolidations(ctx context.Context, request *v1.ListOrderConsolidationsRequest) (*v1.ListOrderConsolidationsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
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
	return okList(ctx, &v1.ListOrderConsolidationsResponse{Data: data}), nil
}

func cargoMeasurementToAPI(value biz.OrderCargoMeasurement) *v1.OrderCargoMeasurement {
	return &v1.OrderCargoMeasurement{Packages: int32(value.Packages), GrossWeightKg: value.GrossWeightKg, VolumeCbm: value.VolumeCbm}
}

func (s *OrderService) ListSameBatchOrders(ctx context.Context, request *v1.ListSameBatchOrdersRequest) (*v1.ListSameBatchOrdersResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderInvalidArgument
	}
	items, err := s.usecase.ListSameBatchOrders(ctx, principal.Organization.ID, orderID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.SameBatchOrderSummary, 0, len(items))
	for _, item := range items {
		summary := &v1.SameBatchOrderSummary{
			OrderId:      item.OrderID.String(),
			OrderNo:      item.OrderNo,
			FlowStatus:   orderFlowStatusToAPI(item.FlowStatus),
			MatchSources: item.MatchSources,
			CreatedAt:    item.CreatedAt.UTC().Format(timeFormatRFC3339),
		}
		if item.CustomerID != nil {
			summary.CustomerId = stringPtrIfNotEmpty(item.CustomerID.String())
		}
		summary.CustomerReferenceNo = stringPtrIfNotEmpty(item.CustomerReferenceNo)
		summary.BookingNo = stringPtrIfNotEmpty(item.BookingNo)
		summary.MasterNo = stringPtrIfNotEmpty(item.MasterNo)
		summary.HouseNo = stringPtrIfNotEmpty(item.HouseNo)
		if item.TotalPackages != nil {
			val := int32(*item.TotalPackages)
			summary.TotalPackages = &val
		}
		summary.TotalGrossWeightKg = item.TotalGrossWeightKg
		summary.TotalVolumeCbm = item.TotalVolumeCbm
		data = append(data, summary)
	}
	return okList(ctx, &v1.ListSameBatchOrdersResponse{Data: data}), nil
}
