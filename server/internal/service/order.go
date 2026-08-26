package service

import (
	"context"
	"strings"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"

	"github.com/google/uuid"
)

type OrderService struct {
	v1.UnimplementedOrderServiceServer
	usecase *biz.OrderUsecase
}

func NewOrderService(usecase *biz.OrderUsecase) *OrderService { return &OrderService{usecase: usecase} }

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
	options := biz.OrderListOptions{Page: page, PageSize: pageSize, Keyword: request.GetKeyword(), Status: request.GetStatus()}
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

func readableOrderBusinessTypes(principal *biz.Principal) []biz.OrderBusinessType {
	types := []struct {
		access access.OrderBusinessType
		biz    biz.OrderBusinessType
	}{
		{access: access.OrderBusinessSE, biz: biz.OrderBusinessSE},
		{access: access.OrderBusinessSI, biz: biz.OrderBusinessSI},
		{access: access.OrderBusinessAE, biz: biz.OrderBusinessAE},
		{access: access.OrderBusinessAI, biz: biz.OrderBusinessAI},
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

func (s *OrderService) ListPersonnelOptions(ctx context.Context, _ *v1.ListPersonnelOptionsRequest) (*v1.ListPersonnelOptionsResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	items, err := s.usecase.ListPersonnelOptions(ctx, principal.Organization.ID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.OrderPersonnelOption, 0, len(items))
	for _, item := range items {
		data = append(data, &v1.OrderPersonnelOption{
			UserId: item.UserID.String(), DisplayName: item.DisplayName,
			OrganizationId: item.OrganizationID.String(), OrganizationName: item.OrganizationName,
		})
	}
	return &v1.ListPersonnelOptionsResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
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

func (s *OrderService) CreateOrder(ctx context.Context, request *v1.CreateOrderRequest) (*v1.CreateOrderResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	input, err := orderFromCreateRequest(request)
	if err != nil {
		return nil, err
	}
	created, err := s.usecase.Create(ctx, principal.Organization.ID, principal.UserID, input)
	if err != nil {
		return nil, err
	}
	return &v1.CreateOrderResponse{Success: true, Code: 0, Message: "OK", Data: orderToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderService) UpdateOrder(ctx context.Context, request *v1.UpdateOrderRequest) (*v1.UpdateOrderResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderNotFound
	}
	existing, err := s.usecase.Get(ctx, principal.Organization.ID, id)
	if err != nil {
		return nil, err
	}
	input, err := mergeOrderUpdateRequest(existing, request)
	if err != nil {
		return nil, err
	}
	updated, err := s.usecase.UpdateDraft(ctx, principal.Organization.ID, principal.UserID, id, request.GetExpectedStatus(), input)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateOrderResponse{Success: true, Code: 0, Message: "OK", Data: orderToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderService) TransitionOrderStatus(ctx context.Context, request *v1.TransitionOrderStatusRequest) (*v1.TransitionOrderStatusResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderNotFound
	}
	updated, err := s.usecase.TransitionStatus(ctx, principal.Organization.ID, principal.UserID, id, request.GetExpectedStatus(), request.GetTargetStatus(), request.GetReason())
	if err != nil {
		return nil, err
	}
	return &v1.TransitionOrderStatusResponse{Success: true, Code: 0, Message: "OK", Data: orderToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func orderFromCreateRequest(request *v1.CreateOrderRequest) (*biz.Order, error) {
	customerID, err := uuid.Parse(request.GetCustomerId())
	if err != nil {
		return nil, biz.ErrOrderCustomerInvalid
	}
	statusTemplateID, err := uuid.Parse(request.GetStatusTemplateId())
	if err != nil {
		return nil, biz.ErrOrderStatusTemplate
	}
	serviceTypeIDs, err := parseUUIDStrings(request.GetServiceTypeIds())
	if err != nil {
		return nil, err
	}
	cargoCategoryIDs, err := parseUUIDStrings(request.GetCargoCategoryIds())
	if err != nil {
		return nil, err
	}
	carrierID, err := parseOptionalUUIDPointer(request.CarrierId)
	if err != nil {
		return nil, err
	}
	bookingAgentID, err := parseOptionalUUIDPointer(request.BookingAgentId)
	if err != nil {
		return nil, err
	}
	foreignAgentID, err := parseOptionalUUIDPointer(request.ForeignAgentId)
	if err != nil {
		return nil, err
	}
	shippingAgentID, err := parseOptionalUUIDPointer(request.ShippingAgentId)
	if err != nil {
		return nil, err
	}
	originLocationID, err := parseOptionalUUIDPointer(request.OriginLocationId)
	if err != nil {
		return nil, err
	}
	destinationLocationID, err := parseOptionalUUIDPointer(request.DestinationLocationId)
	if err != nil {
		return nil, err
	}
	dischargeLocationID, err := parseOptionalUUIDPointer(request.DischargeLocationId)
	if err != nil {
		return nil, err
	}
	transitLocationID, err := parseOptionalUUIDPointer(request.TransitLocationId)
	if err != nil {
		return nil, err
	}
	personnelAssignments, err := personnelAssignmentsFromAPI(request.GetPersonnelAssignments())
	if err != nil {
		return nil, err
	}
	shippingDocuments, err := shippingDocumentsFromAPI(request.GetShippingDocuments())
	if err != nil {
		return nil, err
	}
	containerRequests, err := containerRequestsFromAPI(request.GetContainerRequests())
	if err != nil {
		return nil, err
	}
	return &biz.Order{
		CustomerID: customerID, StatusTemplateID: statusTemplateID,
		CarrierID: carrierID, BookingAgentID: bookingAgentID, ForeignAgentID: foreignAgentID, ShippingAgentID: shippingAgentID,
		CustomerReferenceNo: request.GetCustomerReferenceNo(), InternalReferenceNo: request.GetInternalReferenceNo(), ContractNo: request.GetContractNo(),
		CargoValue: request.GetCargoValue(), CargoCurrency: request.GetCargoCurrency(), InsurancePremium: request.GetInsurancePremium(), InsuranceCurrency: request.GetInsuranceCurrency(),
		UNNumber: request.GetUnNumber(), HazardClass: request.GetHazardClass(), FactoryName: request.GetFactoryName(), CargoReadyAt: request.GetCargoReadyAt(), LoadingTerms: request.GetLoadingTerms(),
		DeclarationCutoffAt: request.GetDeclarationCutoffAt(), ReceivedAt: request.GetReceivedAt(),
		BusinessType: orderBusinessTypeFromAPI(request.GetBusinessType()), TradeDirection: orderTradeDirectionFromAPI(request.GetTradeDirection()),
		TradeTerm: orderTradeTermFromAPI(request.GetTradeTerm()), PaymentTerm: orderPaymentTermFromAPI(request.GetPaymentTerm()),
		ShipmentType: orderShipmentTypeFromAPI(request.ShipmentType), ContainerOwnership: orderContainerOwnershipFromAPI(request.ContainerOwnership), ShipmentMode: orderShipmentModeFromAPI(request.ShipmentMode),
		ServiceTypeIDs: serviceTypeIDs, CargoCategoryIDs: cargoCategoryIDs,
		OriginLocationID: originLocationID, DestinationLocationID: destinationLocationID,
		DischargeLocationID: dischargeLocationID, TransitLocationID: transitLocationID,
		VesselVoyage: request.GetVesselVoyage(), ETD: request.GetEtd(), ETA: request.GetEta(), SICutoff: request.GetSiCutoff(), DocCutoff: request.GetDocCutoff(),
		CustomsCutoff: request.GetCustomsCutoff(), VGMCutoff: request.GetVgmCutoff(), GoodsDescription: request.GetGoodsDescription(),
		TotalPackages: optionalInt32ToInt(request.TotalPackages), TotalGrossWeightKg: request.TotalGrossWeightKg, TotalVolumeCbm: request.TotalVolumeCbm, TotalPackageUnit: request.GetTotalPackageUnit(), SpecialRequirements: request.GetSpecialRequirements(),
		OrderDate: request.GetOrderDate(), Notes: request.GetNotes(),
		BookingNotes: request.GetBookingNotes(), AllocationNotes: request.GetAllocationNotes(), OperationNotes: request.GetOperationNotes(),
		PersonnelAssignments: personnelAssignments, ShippingDocuments: shippingDocuments, ContainerRequests: containerRequests,
	}, nil
}

func mergeOrderUpdateRequest(existing *biz.Order, request *v1.UpdateOrderRequest) (*biz.Order, error) {
	output := *existing
	var err error
	if request.CustomerId != nil {
		output.CustomerID, err = uuid.Parse(request.GetCustomerId())
		if err != nil {
			return nil, biz.ErrOrderCustomerInvalid
		}
	}
	if request.BusinessType != nil {
		output.BusinessType = orderBusinessTypeFromAPI(request.GetBusinessType())
	}
	if request.TradeDirection != nil {
		output.TradeDirection = orderTradeDirectionFromAPI(request.GetTradeDirection())
	}
	if request.TradeTerm != nil {
		output.TradeTerm = orderTradeTermFromAPI(request.GetTradeTerm())
	}
	if request.PaymentTerm != nil {
		output.PaymentTerm = orderPaymentTermFromAPI(request.GetPaymentTerm())
	}
	if request.CarrierId != nil {
		output.CarrierID, err = parseOptionalUUID(request.GetCarrierId())
		if err != nil {
			return nil, err
		}
	}
	if request.BookingAgentId != nil {
		output.BookingAgentID, err = parseOptionalUUID(request.GetBookingAgentId())
		if err != nil {
			return nil, err
		}
	}
	if request.ForeignAgentId != nil {
		output.ForeignAgentID, err = parseOptionalUUID(request.GetForeignAgentId())
		if err != nil {
			return nil, err
		}
	}
	if request.ShippingAgentId != nil {
		output.ShippingAgentID, err = parseOptionalUUID(request.GetShippingAgentId())
		if err != nil {
			return nil, err
		}
	}
	if request.CustomerReferenceNo != nil {
		output.CustomerReferenceNo = request.GetCustomerReferenceNo()
	}
	if request.InternalReferenceNo != nil {
		output.InternalReferenceNo = request.GetInternalReferenceNo()
	}
	if request.ContractNo != nil {
		output.ContractNo = request.GetContractNo()
	}
	if request.CargoValue != nil {
		output.CargoValue = request.GetCargoValue()
	}
	if request.CargoCurrency != nil {
		output.CargoCurrency = request.GetCargoCurrency()
	}
	if request.InsurancePremium != nil {
		output.InsurancePremium = request.GetInsurancePremium()
	}
	if request.InsuranceCurrency != nil {
		output.InsuranceCurrency = request.GetInsuranceCurrency()
	}
	if request.UnNumber != nil {
		output.UNNumber = request.GetUnNumber()
	}
	if request.HazardClass != nil {
		output.HazardClass = request.GetHazardClass()
	}
	if request.FactoryName != nil {
		output.FactoryName = request.GetFactoryName()
	}
	if request.CargoReadyAt != nil {
		output.CargoReadyAt = request.GetCargoReadyAt()
	}
	if request.LoadingTerms != nil {
		output.LoadingTerms = request.GetLoadingTerms()
	}
	if request.DeclarationCutoffAt != nil {
		output.DeclarationCutoffAt = request.GetDeclarationCutoffAt()
	}
	if request.ReceivedAt != nil {
		output.ReceivedAt = request.GetReceivedAt()
	}
	if request.ShipmentType != nil {
		output.ShipmentType = orderShipmentTypeFromAPI(request.ShipmentType)
	}
	if request.ContainerOwnership != nil {
		output.ContainerOwnership = orderContainerOwnershipFromAPI(request.ContainerOwnership)
	}
	if request.ShipmentMode != nil {
		output.ShipmentMode = orderShipmentModeFromAPI(request.ShipmentMode)
	}
	output.ServiceTypeIDs, err = parseUUIDStrings(request.GetServiceTypeIds())
	if err != nil {
		return nil, err
	}
	output.CargoCategoryIDs, err = parseUUIDStrings(request.GetCargoCategoryIds())
	if err != nil {
		return nil, err
	}
	if request.OriginLocationId != nil {
		output.OriginLocationID, err = parseOptionalUUID(request.GetOriginLocationId())
		if err != nil {
			return nil, err
		}
	}
	if request.DestinationLocationId != nil {
		output.DestinationLocationID, err = parseOptionalUUID(request.GetDestinationLocationId())
		if err != nil {
			return nil, err
		}
	}
	if request.DischargeLocationId != nil {
		output.DischargeLocationID, err = parseOptionalUUID(request.GetDischargeLocationId())
		if err != nil {
			return nil, err
		}
	}
	if request.TransitLocationId != nil {
		output.TransitLocationID, err = parseOptionalUUID(request.GetTransitLocationId())
		if err != nil {
			return nil, err
		}
	}
	if request.VesselVoyage != nil {
		output.VesselVoyage = request.GetVesselVoyage()
	}
	if request.Etd != nil {
		output.ETD = request.GetEtd()
	}
	if request.Eta != nil {
		output.ETA = request.GetEta()
	}
	if request.SiCutoff != nil {
		output.SICutoff = request.GetSiCutoff()
	}
	if request.DocCutoff != nil {
		output.DocCutoff = request.GetDocCutoff()
	}
	if request.CustomsCutoff != nil {
		output.CustomsCutoff = request.GetCustomsCutoff()
	}
	if request.VgmCutoff != nil {
		output.VGMCutoff = request.GetVgmCutoff()
	}
	if request.GoodsDescription != nil {
		output.GoodsDescription = request.GetGoodsDescription()
	}
	if request.TotalPackages != nil {
		output.TotalPackages = optionalInt32ToInt(request.TotalPackages)
	}
	if request.TotalGrossWeightKg != nil {
		output.TotalGrossWeightKg = request.TotalGrossWeightKg
	}
	if request.TotalVolumeCbm != nil {
		output.TotalVolumeCbm = request.TotalVolumeCbm
	}
	if request.TotalPackageUnit != nil {
		output.TotalPackageUnit = request.GetTotalPackageUnit()
	}
	if request.SpecialRequirements != nil {
		output.SpecialRequirements = request.GetSpecialRequirements()
	}
	if request.OrderDate != nil {
		output.OrderDate = request.GetOrderDate()
	}
	if request.Notes != nil {
		output.Notes = request.GetNotes()
	}
	if request.BookingNotes != nil {
		output.BookingNotes = request.GetBookingNotes()
	}
	if request.AllocationNotes != nil {
		output.AllocationNotes = request.GetAllocationNotes()
	}
	if request.OperationNotes != nil {
		output.OperationNotes = request.GetOperationNotes()
	}
	output.ShippingDocuments, err = shippingDocumentsFromAPI(request.GetShippingDocuments())
	if err != nil {
		return nil, err
	}
	output.ContainerRequests, err = containerRequestsFromAPI(request.GetContainerRequests())
	if err != nil {
		return nil, err
	}
	return &output, nil
}

func orderToAPI(item *biz.Order) *v1.Order {
	result := &v1.Order{
		Id: item.ID.String(), OrganizationId: item.OrganizationID.String(), OrganizationName: item.OrganizationName, OrderNo: item.OrderNo, CustomerId: item.CustomerID.String(),
		BusinessType: orderBusinessTypeToAPI(item.BusinessType), TradeDirection: orderTradeDirectionToAPI(item.TradeDirection), TradeTerm: orderTradeTermToAPI(item.TradeTerm), PaymentTerm: orderPaymentTermToAPI(item.PaymentTerm),
		Status: item.Status, StatusTemplateId: item.StatusTemplateID.String(), ServiceTypeIds: uuidStrings(item.ServiceTypeIDs), CargoCategoryIds: uuidStrings(item.CargoCategoryIDs),
		CarrierId: uuidStringPtr(item.CarrierID), BookingAgentId: uuidStringPtr(item.BookingAgentID), ForeignAgentId: uuidStringPtr(item.ForeignAgentID), ShippingAgentId: uuidStringPtr(item.ShippingAgentID), ShipmentType: orderShipmentTypeToAPI(item.ShipmentType), ContainerOwnership: orderContainerOwnershipToAPI(item.ContainerOwnership), ShipmentMode: orderShipmentModeToAPI(item.ShipmentMode),
		CustomerReferenceNo: stringPtrIfNotEmpty(item.CustomerReferenceNo), InternalReferenceNo: stringPtrIfNotEmpty(item.InternalReferenceNo), ContractNo: stringPtrIfNotEmpty(item.ContractNo), CargoValue: stringPtrIfNotEmpty(item.CargoValue), CargoCurrency: stringPtrIfNotEmpty(item.CargoCurrency),
		InsurancePremium: stringPtrIfNotEmpty(item.InsurancePremium), InsuranceCurrency: stringPtrIfNotEmpty(item.InsuranceCurrency), UnNumber: stringPtrIfNotEmpty(item.UNNumber), HazardClass: stringPtrIfNotEmpty(item.HazardClass), FactoryName: stringPtrIfNotEmpty(item.FactoryName), CargoReadyAt: stringPtrIfNotEmpty(item.CargoReadyAt), LoadingTerms: stringPtrIfNotEmpty(item.LoadingTerms),
		DeclarationCutoffAt: stringPtrIfNotEmpty(item.DeclarationCutoffAt), ReceivedAt: stringPtrIfNotEmpty(item.ReceivedAt),
		OriginLocationId: uuidStringPtr(item.OriginLocationID), DestinationLocationId: uuidStringPtr(item.DestinationLocationID), DischargeLocationId: uuidStringPtr(item.DischargeLocationID), TransitLocationId: uuidStringPtr(item.TransitLocationID),
		VesselVoyage: stringPtrIfNotEmpty(item.VesselVoyage), Etd: stringPtrIfNotEmpty(item.ETD), Eta: stringPtrIfNotEmpty(item.ETA), SiCutoff: stringPtrIfNotEmpty(item.SICutoff), DocCutoff: stringPtrIfNotEmpty(item.DocCutoff), CustomsCutoff: stringPtrIfNotEmpty(item.CustomsCutoff), VgmCutoff: stringPtrIfNotEmpty(item.VGMCutoff),
		GoodsDescription: stringPtrIfNotEmpty(item.GoodsDescription), TotalPackages: intToInt32Ptr(item.TotalPackages), TotalGrossWeightKg: item.TotalGrossWeightKg, TotalVolumeCbm: item.TotalVolumeCbm, TotalPackageUnit: stringPtrIfNotEmpty(item.TotalPackageUnit), SpecialRequirements: stringPtrIfNotEmpty(item.SpecialRequirements), OrderDate: stringPtrIfNotEmpty(item.OrderDate), Notes: stringPtrIfNotEmpty(item.Notes),
		BookingNotes: stringPtrIfNotEmpty(item.BookingNotes), AllocationNotes: stringPtrIfNotEmpty(item.AllocationNotes), OperationNotes: stringPtrIfNotEmpty(item.OperationNotes),
		CreatedAt: item.CreatedAt.UTC().Format(timeFormatRFC3339), UpdatedAt: item.UpdatedAt.UTC().Format(timeFormatRFC3339),
	}
	result.ShippingDocuments = make([]*v1.OrderShippingDocument, 0, len(item.ShippingDocuments))
	for _, document := range item.ShippingDocuments {
		result.ShippingDocuments = append(result.ShippingDocuments, orderShippingDocumentToAPI(document))
	}
	result.ContainerRequests = make([]*v1.OrderContainerRequest, 0, len(item.ContainerRequests))
	for _, request := range item.ContainerRequests {
		result.ContainerRequests = append(result.ContainerRequests, &v1.OrderContainerRequest{
			Id: request.ID.String(), OrderId: request.OrderID.String(), ContainerSpecId: request.ContainerSpecID.String(), Quantity: int32(request.Quantity),
			CreatedAt: request.CreatedAt.UTC().Format(timeFormatRFC3339), UpdatedAt: request.UpdatedAt.UTC().Format(timeFormatRFC3339),
		})
	}
	return result
}

func shippingDocumentsFromAPI(values []*v1.OrderShippingDocumentInput) ([]*biz.OrderShippingDocument, error) {
	result := make([]*biz.OrderShippingDocument, 0, len(values))
	for _, value := range values {
		if value == nil {
			return nil, biz.ErrOrderInvalidArgument
		}
		id, err := parseOptionalInputID(value.Id)
		if err != nil {
			return nil, err
		}
		result = append(result, &biz.OrderShippingDocument{
			ID: id, MasterNo: value.GetMasterNo(), HouseNo: value.GetHouseNo(),
			MasterDocumentType: optionalStringPointer(value.MasterDocumentType), MasterReleaseMethod: optionalStringPointer(value.MasterReleaseMethod),
			ReleaseType: optionalStringPointer(value.ReleaseType), Note: optionalStringPointer(value.Note),
		})
	}
	return result, nil
}

func containerRequestsFromAPI(values []*v1.OrderContainerRequestInput) ([]*biz.OrderContainerRequest, error) {
	result := make([]*biz.OrderContainerRequest, 0, len(values))
	for _, value := range values {
		if value == nil {
			return nil, biz.ErrOrderInvalidArgument
		}
		id, err := parseOptionalInputID(value.Id)
		if err != nil {
			return nil, err
		}
		containerSpecID, err := uuid.Parse(value.GetContainerSpecId())
		if err != nil {
			return nil, biz.ErrOrderInvalidArgument
		}
		result = append(result, &biz.OrderContainerRequest{ID: id, ContainerSpecID: containerSpecID, Quantity: int(value.GetQuantity())})
	}
	return result, nil
}

func parseOptionalInputID(value *string) (uuid.UUID, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return uuid.Nil, nil
	}
	id, err := uuid.Parse(strings.TrimSpace(*value))
	if err != nil {
		return uuid.Nil, biz.ErrOrderInvalidArgument
	}
	return id, nil
}

func optionalStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func personnelAssignmentsFromAPI(values []*v1.OrderPersonnelAssignmentInput) ([]*biz.OrderPersonnel, error) {
	result := make([]*biz.OrderPersonnel, 0, len(values))
	for _, value := range values {
		if value == nil {
			return nil, biz.ErrOrderInvalidArgument
		}
		userID, err := uuid.Parse(value.GetUserId())
		if err != nil {
			return nil, biz.ErrOrderInvalidArgument
		}
		organizationID, err := uuid.Parse(value.GetOrganizationId())
		if err != nil {
			return nil, biz.ErrOrderInvalidArgument
		}
		role, err := protoRoleToBiz(value.GetRole())
		if err != nil {
			return nil, biz.ErrOrderInvalidArgument
		}
		result = append(result, &biz.OrderPersonnel{UserID: userID, OrganizationID: organizationID, Role: role})
	}
	return result, nil
}

const timeFormatRFC3339 = "2006-01-02T15:04:05Z07:00"

func parseUUIDStrings(values []string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, biz.ErrOrderInvalidArgument
		}
		result = append(result, id)
	}
	return result, nil
}

func parseOptionalUUID(value string) (*uuid.UUID, error) {
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, biz.ErrOrderInvalidArgument
	}
	return &id, nil
}

func parseOptionalUUIDPointer(value *string) (*uuid.UUID, error) {
	if value == nil {
		return nil, nil
	}
	return parseOptionalUUID(*value)
}

func uuidStringPtr(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}
	result := value.String()
	return &result
}
func stringPtrIfNotEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
func optionalInt32ToInt(value *int32) *int {
	if value == nil {
		return nil
	}
	result := int(*value)
	return &result
}
func intToInt32Ptr(value *int) *int32 {
	if value == nil {
		return nil
	}
	result := int32(*value)
	return &result
}

func orderBusinessTypeFromAPI(value v1.BusinessType) biz.OrderBusinessType {
	switch value {
	case v1.BusinessType_BUSINESS_TYPE_SE:
		return biz.OrderBusinessSE
	case v1.BusinessType_BUSINESS_TYPE_SI:
		return biz.OrderBusinessSI
	case v1.BusinessType_BUSINESS_TYPE_AE:
		return biz.OrderBusinessAE
	case v1.BusinessType_BUSINESS_TYPE_AI:
		return biz.OrderBusinessAI
	case v1.BusinessType_BUSINESS_TYPE_LAND:
		return biz.OrderBusinessLand
	case v1.BusinessType_BUSINESS_TYPE_RAIL:
		return biz.OrderBusinessRail
	default:
		return ""
	}
}
func orderBusinessTypeToAPI(value biz.OrderBusinessType) v1.BusinessType {
	switch value {
	case biz.OrderBusinessSE:
		return v1.BusinessType_BUSINESS_TYPE_SE
	case biz.OrderBusinessSI:
		return v1.BusinessType_BUSINESS_TYPE_SI
	case biz.OrderBusinessAE:
		return v1.BusinessType_BUSINESS_TYPE_AE
	case biz.OrderBusinessAI:
		return v1.BusinessType_BUSINESS_TYPE_AI
	case biz.OrderBusinessLand:
		return v1.BusinessType_BUSINESS_TYPE_LAND
	case biz.OrderBusinessRail:
		return v1.BusinessType_BUSINESS_TYPE_RAIL
	default:
		return v1.BusinessType_BUSINESS_TYPE_UNSPECIFIED
	}
}
func orderTradeDirectionFromAPI(value v1.TradeDirection) biz.OrderTradeDirection {
	if value == v1.TradeDirection_TRADE_DIRECTION_EXPORT {
		return biz.OrderTradeExport
	}
	if value == v1.TradeDirection_TRADE_DIRECTION_IMPORT {
		return biz.OrderTradeImport
	}
	return ""
}
func orderTradeDirectionToAPI(value biz.OrderTradeDirection) v1.TradeDirection {
	if value == biz.OrderTradeExport {
		return v1.TradeDirection_TRADE_DIRECTION_EXPORT
	}
	if value == biz.OrderTradeImport {
		return v1.TradeDirection_TRADE_DIRECTION_IMPORT
	}
	return v1.TradeDirection_TRADE_DIRECTION_UNSPECIFIED
}
func orderTradeTermFromAPI(value v1.TradeTerm) biz.OrderTradeTerm {
	switch value {
	case v1.TradeTerm_TRADE_TERM_EXW:
		return biz.OrderTradeEXW
	case v1.TradeTerm_TRADE_TERM_FCA:
		return biz.OrderTradeFCA
	case v1.TradeTerm_TRADE_TERM_FOB:
		return biz.OrderTradeFOB
	case v1.TradeTerm_TRADE_TERM_CFR:
		return biz.OrderTradeCFR
	case v1.TradeTerm_TRADE_TERM_CIF:
		return biz.OrderTradeCIF
	case v1.TradeTerm_TRADE_TERM_CPT:
		return biz.OrderTradeCPT
	case v1.TradeTerm_TRADE_TERM_CIP:
		return biz.OrderTradeCIP
	case v1.TradeTerm_TRADE_TERM_DAP:
		return biz.OrderTradeDAP
	case v1.TradeTerm_TRADE_TERM_DPU:
		return biz.OrderTradeDPU
	case v1.TradeTerm_TRADE_TERM_DDU:
		return biz.OrderTradeDDU
	case v1.TradeTerm_TRADE_TERM_DDP:
		return biz.OrderTradeDDP
	case v1.TradeTerm_TRADE_TERM_LDP:
		return biz.OrderTradeLDP
	default:
		return ""
	}
}
func orderTradeTermToAPI(value biz.OrderTradeTerm) v1.TradeTerm {
	switch value {
	case biz.OrderTradeEXW:
		return v1.TradeTerm_TRADE_TERM_EXW
	case biz.OrderTradeFCA:
		return v1.TradeTerm_TRADE_TERM_FCA
	case biz.OrderTradeFOB:
		return v1.TradeTerm_TRADE_TERM_FOB
	case biz.OrderTradeCFR:
		return v1.TradeTerm_TRADE_TERM_CFR
	case biz.OrderTradeCIF:
		return v1.TradeTerm_TRADE_TERM_CIF
	case biz.OrderTradeCPT:
		return v1.TradeTerm_TRADE_TERM_CPT
	case biz.OrderTradeCIP:
		return v1.TradeTerm_TRADE_TERM_CIP
	case biz.OrderTradeDAP:
		return v1.TradeTerm_TRADE_TERM_DAP
	case biz.OrderTradeDPU:
		return v1.TradeTerm_TRADE_TERM_DPU
	case biz.OrderTradeDDU:
		return v1.TradeTerm_TRADE_TERM_DDU
	case biz.OrderTradeDDP:
		return v1.TradeTerm_TRADE_TERM_DDP
	case biz.OrderTradeLDP:
		return v1.TradeTerm_TRADE_TERM_LDP
	default:
		return v1.TradeTerm_TRADE_TERM_UNSPECIFIED
	}
}
func orderPaymentTermFromAPI(value v1.PaymentTerm) biz.OrderPaymentTerm {
	if value == v1.PaymentTerm_PAYMENT_TERM_PREPAID {
		return biz.OrderPaymentPrepaid
	}
	if value == v1.PaymentTerm_PAYMENT_TERM_COLLECT {
		return biz.OrderPaymentCollect
	}
	return ""
}
func orderPaymentTermToAPI(value biz.OrderPaymentTerm) v1.PaymentTerm {
	if value == biz.OrderPaymentPrepaid {
		return v1.PaymentTerm_PAYMENT_TERM_PREPAID
	}
	if value == biz.OrderPaymentCollect {
		return v1.PaymentTerm_PAYMENT_TERM_COLLECT
	}
	return v1.PaymentTerm_PAYMENT_TERM_UNSPECIFIED
}
func orderShipmentTypeFromAPI(value *v1.ShipmentType) *biz.OrderShipmentType {
	if value == nil {
		return nil
	}
	var result biz.OrderShipmentType
	switch *value {
	case v1.ShipmentType_SHIPMENT_TYPE_FCL:
		result = biz.OrderShipmentFCL
	case v1.ShipmentType_SHIPMENT_TYPE_LCL:
		result = biz.OrderShipmentLCL
	case v1.ShipmentType_SHIPMENT_TYPE_BREAK_BULK:
		result = biz.OrderShipmentBreakBulk
	default:
		return nil
	}
	return &result
}
func orderShipmentTypeToAPI(value *biz.OrderShipmentType) *v1.ShipmentType {
	if value == nil {
		return nil
	}
	var result v1.ShipmentType
	switch *value {
	case biz.OrderShipmentFCL:
		result = v1.ShipmentType_SHIPMENT_TYPE_FCL
	case biz.OrderShipmentLCL:
		result = v1.ShipmentType_SHIPMENT_TYPE_LCL
	case biz.OrderShipmentBreakBulk:
		result = v1.ShipmentType_SHIPMENT_TYPE_BREAK_BULK
	default:
		return nil
	}
	return &result
}
func orderContainerOwnershipFromAPI(value *v1.ContainerOwnership) *biz.OrderContainerOwnership {
	if value == nil {
		return nil
	}
	var result biz.OrderContainerOwnership
	switch *value {
	case v1.ContainerOwnership_CONTAINER_OWNERSHIP_COC:
		result = biz.OrderContainerCOC
	case v1.ContainerOwnership_CONTAINER_OWNERSHIP_SOC:
		result = biz.OrderContainerSOC
	default:
		return nil
	}
	return &result
}
func orderContainerOwnershipToAPI(value *biz.OrderContainerOwnership) *v1.ContainerOwnership {
	if value == nil {
		return nil
	}
	var result v1.ContainerOwnership
	switch *value {
	case biz.OrderContainerCOC:
		result = v1.ContainerOwnership_CONTAINER_OWNERSHIP_COC
	case biz.OrderContainerSOC:
		result = v1.ContainerOwnership_CONTAINER_OWNERSHIP_SOC
	default:
		return nil
	}
	return &result
}
func orderShipmentModeFromAPI(value *v1.ShipmentMode) *biz.OrderShipmentMode {
	if value == nil {
		return nil
	}
	var result biz.OrderShipmentMode
	switch *value {
	case v1.ShipmentMode_SHIPMENT_MODE_TRADITIONAL_FORWARDING:
		result = biz.OrderShipmentTraditionalForwarding
	case v1.ShipmentMode_SHIPMENT_MODE_CROSS_BORDER:
		result = biz.OrderShipmentCrossBorder
	default:
		return nil
	}
	return &result
}
func orderShipmentModeToAPI(value *biz.OrderShipmentMode) *v1.ShipmentMode {
	if value == nil {
		return nil
	}
	var result v1.ShipmentMode
	switch *value {
	case biz.OrderShipmentTraditionalForwarding:
		result = v1.ShipmentMode_SHIPMENT_MODE_TRADITIONAL_FORWARDING
	case biz.OrderShipmentCrossBorder:
		result = v1.ShipmentMode_SHIPMENT_MODE_CROSS_BORDER
	default:
		return nil
	}
	return &result
}

var _ v1.OrderServiceServer = (*OrderService)(nil)
