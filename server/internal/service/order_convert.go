package service

import (
	"strings"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"

	"github.com/google/uuid"
)

func orderToAPI(item *biz.Order) *v1.Order {
	result := &v1.Order{
		Id: item.ID.String(), OrganizationId: item.OrganizationID.String(), OrganizationName: item.OrganizationName, OrderNo: item.OrderNo, CustomerId: item.CustomerID.String(),
		BusinessType: orderBusinessTypeToAPI(item.BusinessType), TradeDirection: orderTradeDirectionToAPI(item.TradeDirection), TradeTerm: orderTradeTermToAPI(item.TradeTerm), PaymentTerm: orderPaymentTermToAPI(item.PaymentTerm),
		FlowStatus: orderFlowStatusToAPI(item.FlowStatus), TerminationStatus: orderTerminationStatusToAPI(item.TerminationStatus), TerminationType: orderTerminationTypeToAPI(item.TerminationType),
		TerminationReason: stringPtrIfNotEmpty(item.TerminationReason), TerminatedAt: timePtrToString(item.TerminatedAt), TerminatedBy: uuidStringPtr(item.TerminatedBy),
		ClosureStatus: orderClosureStatusToAPI(item.ClosureStatus), ClosureReason: stringPtrIfNotEmpty(item.ClosureReason), ClosedAt: timePtrToString(item.ClosedAt), ClosedBy: uuidStringPtr(item.ClosedBy),
		Version: item.Version, HasActiveException: item.HasActiveException, ActiveExceptionCount: int32(item.ActiveExceptionCount), AllowedActions: orderAllowedActionsToAPI(item.AllowedActions),
		ServiceTypeIds: uuidStrings(item.ServiceTypeIDs), CargoCategoryIds: uuidStrings(item.CargoCategoryIDs),
		CarrierId: uuidStringPtr(item.CarrierID), BookingAgentId: uuidStringPtr(item.BookingAgentID), ForeignAgentId: uuidStringPtr(item.ForeignAgentID), ShippingAgentId: uuidStringPtr(item.ShippingAgentID), ShipmentType: orderShipmentTypeToAPI(item.ShipmentType), ContainerOwnership: orderContainerOwnershipToAPI(item.ContainerOwnership), ShipmentMode: orderShipmentModeToAPI(item.ShipmentMode),
		CustomerReferenceNo: stringPtrIfNotEmpty(item.CustomerReferenceNo), InternalReferenceNo: stringPtrIfNotEmpty(item.InternalReferenceNo), ContractNo: stringPtrIfNotEmpty(item.ContractNo), CargoValue: stringPtrIfNotEmpty(item.CargoValue), CargoCurrency: stringPtrIfNotEmpty(item.CargoCurrency),
		InsurancePremium: stringPtrIfNotEmpty(item.InsurancePremium), InsuranceCurrency: stringPtrIfNotEmpty(item.InsuranceCurrency), UnNumber: stringPtrIfNotEmpty(item.UNNumber), HazardClass: stringPtrIfNotEmpty(item.HazardClass), FactoryName: stringPtrIfNotEmpty(item.FactoryName), CargoReadyAt: stringPtrIfNotEmpty(item.CargoReadyAt), LoadingTerms: stringPtrIfNotEmpty(item.LoadingTerms),
		DeclarationCutoffAt: stringPtrIfNotEmpty(item.DeclarationCutoffAt), ReceivedAt: stringPtrIfNotEmpty(item.ReceivedAt),
		OriginLocationId: uuidStringPtr(item.OriginLocationID), DestinationLocationId: uuidStringPtr(item.DestinationLocationID), DischargeLocationId: uuidStringPtr(item.DischargeLocationID), TransitLocationId: uuidStringPtr(item.TransitLocationID),
		VesselVoyage: stringPtrIfNotEmpty(item.VesselVoyage), Etd: stringPtrIfNotEmpty(item.ETD), Eta: stringPtrIfNotEmpty(item.ETA), SiCutoff: stringPtrIfNotEmpty(item.SICutoff), DocCutoff: stringPtrIfNotEmpty(item.DocCutoff), CustomsCutoff: stringPtrIfNotEmpty(item.CustomsCutoff), VgmCutoff: stringPtrIfNotEmpty(item.VGMCutoff),
		GoodsDescription: stringPtrIfNotEmpty(item.GoodsDescription), TotalPackages: intToInt32Ptr(item.TotalPackages), TotalGrossWeightKg: item.TotalGrossWeightKg, TotalVolumeCbm: item.TotalVolumeCbm, TotalPackageUnit: stringPtrIfNotEmpty(item.TotalPackageUnit), SpecialRequirements: stringPtrIfNotEmpty(item.SpecialRequirements), OrderDate: stringPtrIfNotEmpty(item.OrderDate), Notes: stringPtrIfNotEmpty(item.Notes),
		BookingNotes: stringPtrIfNotEmpty(item.BookingNotes), AllocationNotes: stringPtrIfNotEmpty(item.AllocationNotes), OperationNotes: stringPtrIfNotEmpty(item.OperationNotes),
		ShipperShortName: stringPtrIfNotEmpty(item.ShipperShortName), ConsigneeShortName: stringPtrIfNotEmpty(item.ConsigneeShortName), LockedAt: timePtrToString(item.LockedAt), IsShared: item.IsShared, Tags: item.Tags,
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

func normalizedOrderTags(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
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

func timePtrToString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(timeFormatRFC3339)
	return &formatted
}
