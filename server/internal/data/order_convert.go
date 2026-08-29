package data

import (
	"strings"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderabnormalcaseent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderabnormalcase"
	ordercargoent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargocategory"
	ordercontainerrequestent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainerrequest"
	orderserviceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderservicetype"
	ordershippingdocumentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordershippingdocument"
)

func withOrderEdges(query *ent.OrderQuery) *ent.OrderQuery {
	return query.
		WithOrganization().
		WithServiceTypes(func(q *ent.OrderServiceTypeQuery) { q.Order(orderserviceent.ByCreatedAt()) }).
		WithCargoCategories(func(q *ent.OrderCargoCategoryQuery) { q.Order(ordercargoent.ByCreatedAt()) }).
		WithShippingDocuments(func(q *ent.OrderShippingDocumentQuery) {
			q.WithConsolidation().Order(ordershippingdocumentent.ByCreatedAt())
		}).
		WithContainerRequests(func(q *ent.OrderContainerRequestQuery) { q.Order(ordercontainerrequestent.ByCreatedAt()) }).
		WithAbnormalCases(func(q *ent.OrderAbnormalCaseQuery) {
			q.Where(orderabnormalcaseent.StatusEQ(orderabnormalcaseent.StatusACTIVE))
		})
}

func orderToBiz(item *ent.Order) *biz.Order {
	result := &biz.Order{
		ID: item.ID, OrganizationID: item.OrganizationID, OrganizationName: item.Edges.Organization.Name, OrderNo: item.OrderNo, CustomerID: item.CustomerID,
		CarrierID: item.CarrierID, BookingAgentID: item.BookingAgentID, ForeignAgentID: item.ForeignAgentID, ShippingAgentID: item.ShippingAgentID, BusinessType: biz.OrderBusinessType(item.BusinessType),
		CustomerReferenceNo: item.CustomerReferenceNo, InternalReferenceNo: item.InternalReferenceNo, ContractNo: item.ContractNo, CargoValue: item.CargoValue, CargoCurrency: item.CargoCurrency,
		ShipperShortName: item.ShipperShortName, ConsigneeShortName: item.ConsigneeShortName, LockedAt: item.LockedAt, IsShared: item.IsShared, Tags: append([]string(nil), item.Tags...),
		InsurancePremium: item.InsurancePremium, InsuranceCurrency: item.InsuranceCurrency, UNNumber: item.UnNumber, HazardClass: item.HazardClass, FactoryName: item.FactoryName, CargoReadyAt: item.CargoReadyAt, LoadingTerms: item.LoadingTerms,
		DeclarationCutoffAt: item.DeclarationCutoffAt, ReceivedAt: item.ReceivedAt,
		TradeDirection: biz.OrderTradeDirection(item.TradeDirection), TradeTerm: biz.OrderTradeTerm(item.TradeTerm), PaymentTerm: biz.OrderPaymentTerm(item.PaymentTerm),
		FlowStatus: biz.OrderFlowStatus(item.FlowStatus), TerminationStatus: biz.OrderTerminationStatus(item.TerminationStatus), TerminationReason: orderOptionalStringValue(item.TerminationReason),
		TerminatedAt: item.TerminatedAt, TerminatedBy: item.TerminatedBy, ClosureStatus: biz.OrderClosureStatus(item.ClosureStatus), ClosureReason: orderOptionalStringValue(item.ClosureReason),
		ClosedAt: item.ClosedAt, ClosedBy: item.ClosedBy, Version: item.Version, HasActiveException: len(item.Edges.AbnormalCases) > 0, ActiveExceptionCount: len(item.Edges.AbnormalCases),
		OriginLocationID: item.OriginLocationID, DestinationLocationID: item.DestinationLocationID,
		DischargeLocationID: item.DischargeLocationID, TransitLocationID: item.TransitLocationID, VesselVoyage: item.VesselVoyage, ETD: item.Etd, ETA: item.Eta,
		SICutoff: item.SiCutoff, DocCutoff: item.DocCutoff, CustomsCutoff: item.CustomsCutoff, VGMCutoff: item.VgmCutoff,
		GoodsDescription: item.GoodsDescription, TotalPackages: item.TotalPackages, TotalGrossWeightKg: item.TotalGrossWeightKg, TotalVolumeCbm: item.TotalVolumeCbm, TotalPackageUnit: item.TotalPackageUnit,
		SpecialRequirements: item.SpecialRequirements, OrderDate: item.OrderDate, Notes: item.Notes,
		BookingNotes: item.BookingNotes, AllocationNotes: item.AllocationNotes, OperationNotes: item.OperationNotes,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
	if item.ShipmentType != nil {
		value := biz.OrderShipmentType(*item.ShipmentType)
		result.ShipmentType = &value
	}
	if item.TerminationType != nil {
		value := biz.OrderTerminationType(*item.TerminationType)
		result.TerminationType = &value
	}
	if item.ContainerOwnership != nil {
		value := biz.OrderContainerOwnership(*item.ContainerOwnership)
		result.ContainerOwnership = &value
	}
	if item.ShipmentMode != nil {
		value := biz.OrderShipmentMode(*item.ShipmentMode)
		result.ShipmentMode = &value
	}
	result.ServiceTypeIDs = make([]uuid.UUID, 0, len(item.Edges.ServiceTypes))
	for _, link := range item.Edges.ServiceTypes {
		result.ServiceTypeIDs = append(result.ServiceTypeIDs, link.MasterDataItemID)
	}
	result.CargoCategoryIDs = make([]uuid.UUID, 0, len(item.Edges.CargoCategories))
	for _, link := range item.Edges.CargoCategories {
		result.CargoCategoryIDs = append(result.CargoCategoryIDs, link.MasterDataItemID)
	}
	result.ShippingDocuments = make([]*biz.OrderShippingDocument, 0, len(item.Edges.ShippingDocuments))
	for _, document := range item.Edges.ShippingDocuments {
		result.ShippingDocuments = append(result.ShippingDocuments, orderShippingDocumentToBiz(document))
	}
	result.ContainerRequests = make([]*biz.OrderContainerRequest, 0, len(item.Edges.ContainerRequests))
	for _, request := range item.Edges.ContainerRequests {
		result.ContainerRequests = append(result.ContainerRequests, &biz.OrderContainerRequest{
			ID: request.ID, OrderID: request.OrderID, ContainerSpecID: request.ContainerSpecID,
			Quantity: request.Quantity, CreatedAt: request.CreatedAt, UpdatedAt: request.UpdatedAt,
		})
	}
	result.AllowedActions = orderAllowedActions(result)
	return result
}

func orderAllowedActions(order *biz.Order) []biz.OrderAllowedAction {
	if order.ClosureStatus == biz.OrderClosureClosed {
		return []biz.OrderAllowedAction{biz.OrderActionReopen}
	}
	result := make([]biz.OrderAllowedAction, 0, 4)
	if order.TerminationStatus == biz.OrderTerminationActive {
		if order.FlowStatus == biz.OrderFlowDraft {
			result = append(result, biz.OrderActionEdit)
		}
		if order.FlowStatus != biz.OrderFlowDocumentReleased {
			result = append(result, biz.OrderActionTransitionFlow)
		}
		result = append(result, biz.OrderActionStartTermination)
	} else if order.TerminationStatus == biz.OrderTerminationTerminating {
		result = append(result, biz.OrderActionCompleteTermination, biz.OrderActionCancelTermination)
	} else {
		result = append(result, biz.OrderActionCancelTermination)
	}
	if !order.HasActiveException && (order.FlowStatus == biz.OrderFlowDocumentReleased || order.TerminationStatus == biz.OrderTerminationTerminated) {
		result = append(result, biz.OrderActionClose)
	}
	return result
}

func setOrderOptionalReferences(update *ent.OrderUpdateOne, input *biz.Order) {
	if input.CarrierID == nil {
		update.ClearCarrierID()
	} else {
		update.SetCarrierID(*input.CarrierID)
	}
	if input.BookingAgentID == nil {
		update.ClearBookingAgentID()
	} else {
		update.SetBookingAgentID(*input.BookingAgentID)
	}
	if input.ForeignAgentID == nil {
		update.ClearForeignAgentID()
	} else {
		update.SetForeignAgentID(*input.ForeignAgentID)
	}
	if input.ShippingAgentID == nil {
		update.ClearShippingAgentID()
	} else {
		update.SetShippingAgentID(*input.ShippingAgentID)
	}
	if input.ShipmentType == nil {
		update.ClearShipmentType()
	} else {
		update.SetShipmentType(orderent.ShipmentType(*input.ShipmentType))
	}
	if input.ContainerOwnership == nil {
		update.ClearContainerOwnership()
	} else {
		update.SetContainerOwnership(orderent.ContainerOwnership(*input.ContainerOwnership))
	}
	if input.ShipmentMode == nil {
		update.ClearShipmentMode()
	} else {
		update.SetShipmentMode(orderent.ShipmentMode(*input.ShipmentMode))
	}
	if input.OriginLocationID == nil {
		update.ClearOriginLocationID()
	} else {
		update.SetOriginLocationID(*input.OriginLocationID)
	}
	if input.DestinationLocationID == nil {
		update.ClearDestinationLocationID()
	} else {
		update.SetDestinationLocationID(*input.DestinationLocationID)
	}
	if input.DischargeLocationID == nil {
		update.ClearDischargeLocationID()
	} else {
		update.SetDischargeLocationID(*input.DischargeLocationID)
	}
	if input.TransitLocationID == nil {
		update.ClearTransitLocationID()
	} else {
		update.SetTransitLocationID(*input.TransitLocationID)
	}
}

func setOrderOptionalAmounts(update *ent.OrderUpdateOne, input *biz.Order) {
	if input.CargoCurrency == "" {
		update.ClearCargoCurrency()
	} else {
		update.SetCargoCurrency(input.CargoCurrency)
	}
	if input.InsuranceCurrency == "" {
		update.ClearInsuranceCurrency()
	} else {
		update.SetInsuranceCurrency(input.InsuranceCurrency)
	}
}

func nonEmptyStringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func orderOptionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func orderShipmentTypeToEnt(value *biz.OrderShipmentType) *orderent.ShipmentType {
	if value == nil {
		return nil
	}
	result := orderent.ShipmentType(*value)
	return &result
}

func orderContainerOwnershipToEnt(value *biz.OrderContainerOwnership) *orderent.ContainerOwnership {
	if value == nil {
		return nil
	}
	result := orderent.ContainerOwnership(*value)
	return &result
}

func orderShipmentModeToEnt(value *biz.OrderShipmentMode) *orderent.ShipmentMode {
	if value == nil {
		return nil
	}
	result := orderent.ShipmentMode(*value)
	return &result
}

func nonNilUUIDs(values ...*uuid.UUID) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if value != nil {
			result = append(result, *value)
		}
	}
	return result
}

func mapOrderConstraint(err error) error {
	if ent.IsConstraintError(err) && strings.Contains(err.Error(), "order_organization_id_order_no") {
		return biz.ErrOrderNumberExists
	}
	return err
}
