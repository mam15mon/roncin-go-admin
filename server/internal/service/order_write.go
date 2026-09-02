package service

import (
	"context"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"

	"github.com/google/uuid"
)

func (s *OrderService) CreateOrder(ctx context.Context, request *v1.CreateOrderRequest) (*v1.CreateOrderResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	input, err := orderFromCreateRequest(request)
	if err != nil {
		return nil, err
	}
	created, err := s.usecase.Create(ctx, principal.Organization.ID, principal.UserID, input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateOrderResponse{Data: orderToAPI(created)}), nil
}

func (s *OrderService) UpdateOrder(ctx context.Context, request *v1.UpdateOrderRequest) (*v1.UpdateOrderResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
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
	updated, err := s.usecase.UpdateDraft(ctx, principal.Organization.ID, principal.UserID, id, request.GetExpectedVersion(), input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.UpdateOrderResponse{Data: orderToAPI(updated)}), nil
}

func (s *OrderService) TransitionOrderStatus(ctx context.Context, request *v1.TransitionOrderStatusRequest) (*v1.TransitionOrderStatusResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderNotFound
	}
	updated, err := s.usecase.TransitionStatus(ctx, principal.Organization.ID, principal.UserID, id, request.GetExpectedVersion(), orderFlowStatusFromAPI(request.GetTargetFlowStatus()), request.GetReason())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.TransitionOrderStatusResponse{Data: orderToAPI(updated)}), nil
}

func (s *OrderService) TransitionOrderTermination(ctx context.Context, request *v1.TransitionOrderTerminationRequest) (*v1.TransitionOrderTerminationResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderNotFound
	}
	terminationType := orderTerminationTypeFromAPI(request.TerminationType)
	updated, err := s.usecase.TransitionTermination(ctx, principal.Organization.ID, principal.UserID, id, request.GetExpectedVersion(), orderTerminationStatusFromAPI(request.GetTargetStatus()), terminationType, request.GetReason())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.TransitionOrderTerminationResponse{Data: orderToAPI(updated)}), nil
}

func (s *OrderService) TransitionOrderClosure(ctx context.Context, request *v1.TransitionOrderClosureRequest) (*v1.TransitionOrderClosureResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderNotFound
	}
	updated, err := s.usecase.TransitionClosure(ctx, principal.Organization.ID, principal.UserID, id, request.GetExpectedVersion(), orderClosureStatusFromAPI(request.GetTargetStatus()), request.GetReason())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.TransitionOrderClosureResponse{Data: orderToAPI(updated)}), nil
}

func orderFromCreateRequest(request *v1.CreateOrderRequest) (*biz.Order, error) {
	customerID, err := uuid.Parse(request.GetCustomerId())
	if err != nil {
		return nil, biz.ErrOrderCustomerInvalid
	}
	serviceTypeIDs, err := parseUUIDValues(request.GetServiceTypeIds(), biz.ErrOrderInvalidArgument)
	if err != nil {
		return nil, err
	}
	cargoCategoryIDs, err := parseUUIDValues(request.GetCargoCategoryIds(), biz.ErrOrderInvalidArgument)
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
	seaMasterBillInput, err := seaMasterBillInputFromAPI(request.GetSeaMasterBill())
	if err != nil {
		return nil, err
	}
	return &biz.Order{
		CustomerID: customerID,
		CarrierID:  carrierID, BookingAgentID: bookingAgentID, ForeignAgentID: foreignAgentID, ShippingAgentID: shippingAgentID,
		CustomerReferenceNo: request.GetCustomerReferenceNo(), InternalReferenceNo: request.GetInternalReferenceNo(), ContractNo: request.GetContractNo(),
		ShipperShortName: request.GetShipperShortName(), ConsigneeShortName: request.GetConsigneeShortName(),
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
		SeaMasterBillInput: seaMasterBillInput,
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
	output.ServiceTypeIDs, err = parseUUIDValues(request.GetServiceTypeIds(), biz.ErrOrderInvalidArgument)
	if err != nil {
		return nil, err
	}
	output.CargoCategoryIDs, err = parseUUIDValues(request.GetCargoCategoryIds(), biz.ErrOrderInvalidArgument)
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
	if request.ShipperShortName != nil {
		output.ShipperShortName = request.GetShipperShortName()
	}
	if request.ConsigneeShortName != nil {
		output.ConsigneeShortName = request.GetConsigneeShortName()
	}
	output.ShippingDocuments, err = shippingDocumentsFromAPI(request.GetShippingDocuments())
	if err != nil {
		return nil, err
	}
	output.ContainerRequests, err = containerRequestsFromAPI(request.GetContainerRequests())
	if err != nil {
		return nil, err
	}
	if request.SeaMasterBill != nil {
		output.SeaMasterBillInput, err = seaMasterBillInputFromAPI(request.GetSeaMasterBill())
		if err != nil {
			return nil, err
		}
	}
	return &output, nil
}
