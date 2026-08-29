package biz

import (
	"context"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

func (uc *OrderUsecase) Get(ctx context.Context, organizationID, id uuid.UUID) (*Order, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrOrderNotFound
	}
	order, err := uc.repo.Get(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	if err := attachOrderTags(ctx, uc.tagRepo, order); err != nil {
		return nil, err
	}
	return order, nil
}

func attachOrderTags(ctx context.Context, tagRepo BusinessTagRepo, orders ...*Order) error {
	if tagRepo == nil || len(orders) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(orders))
	for _, order := range orders {
		ids = append(ids, order.ID)
	}
	tags, err := tagRepo.LoadOrderTags(ctx, ids)
	if err != nil {
		return err
	}
	for _, order := range orders {
		order.Tags = tags[order.ID]
	}
	return nil
}

func (uc *OrderUsecase) Find(ctx context.Context, id uuid.UUID) (*Order, error) {
	if id == uuid.Nil {
		return nil, ErrOrderNotFound
	}
	return uc.repo.Find(ctx, id)
}

func (uc *OrderUsecase) List(ctx context.Context, organizationIDs []uuid.UUID, options OrderListOptions) (*OrderList, error) {
	if len(organizationIDs) == 0 || !ValidListPagination(options.Page, options.PageSize) || options.BusinessType != "" && !options.BusinessType.Valid() || options.BusinessType == "" && len(options.BusinessTypes) == 0 {
		return nil, ErrOrderInvalidArgument
	}
	for _, organizationID := range organizationIDs {
		if organizationID == uuid.Nil {
			return nil, ErrOrderInvalidArgument
		}
	}
	for _, businessType := range options.BusinessTypes {
		if !businessType.Valid() {
			return nil, ErrOrderInvalidArgument
		}
	}
	options.Keyword = strings.TrimSpace(options.Keyword)
	options.NumberKeyword = strings.TrimSpace(options.NumberKeyword)
	options.ConsigneeShortName = strings.TrimSpace(options.ConsigneeShortName)
	options.ShipperShortName = strings.TrimSpace(options.ShipperShortName)
	if options.NumberKeyword != "" && !options.NumberType.Valid() || options.NumberKeyword == "" && options.NumberType != "" {
		return nil, ErrOrderInvalidArgument
	}
	if options.FlowStatus != "" && !options.FlowStatus.Valid() {
		return nil, ErrOrderInvalidArgument
	}
	if options.TerminationStatus != "" && !options.TerminationStatus.Valid() || options.ClosureStatus != "" && !options.ClosureStatus.Valid() {
		return nil, ErrOrderInvalidArgument
	}
	result, err := uc.repo.List(ctx, organizationIDs, options)
	if err != nil {
		return nil, err
	}
	if err := attachOrderTags(ctx, uc.tagRepo, result.Items...); err != nil {
		return nil, err
	}
	return result, nil
}

func (uc *OrderUsecase) CheckReference(ctx context.Context, organizationID uuid.UUID, check OrderReferenceCheck) (*OrderReferenceMatch, error) {
	check.ReferenceNo = strings.TrimSpace(check.ReferenceNo)
	if organizationID == uuid.Nil || !check.ReferenceType.Valid() || check.ReferenceNo == "" || utf8.RuneCountInString(check.ReferenceNo) > 100 {
		return nil, ErrOrderInvalidArgument
	}
	if check.ReferenceType == OrderReferenceCustomer && (check.CustomerID == nil || *check.CustomerID == uuid.Nil) {
		return nil, ErrOrderInvalidArgument
	}
	if check.ExcludeOrderID != nil && *check.ExcludeOrderID == uuid.Nil {
		return nil, ErrOrderInvalidArgument
	}
	return uc.repo.FindReferenceDuplicate(ctx, organizationID, check)
}

func (uc *OrderUsecase) ListPersonnelOptions(ctx context.Context, organizationID uuid.UUID, options SelectorListOptions) (*PagedList[*OrderPersonnelOption], error) {
	if organizationID == uuid.Nil || !ValidListPagination(options.Page, options.PageSize) {
		return nil, ErrOrderInvalidArgument
	}
	options.Keyword = strings.TrimSpace(options.Keyword)
	return uc.repo.ListPersonnelOptions(ctx, organizationID, options)
}

func (uc *OrderUsecase) ListConsolidationSummaries(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderConsolidationSummary, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderInvalidArgument
	}
	return uc.repo.ListConsolidationSummaries(ctx, organizationID, orderID)
}

func (uc *OrderUsecase) Create(ctx context.Context, organizationID, actorID uuid.UUID, input *Order) (*Order, error) {
	normalized, err := normalizeOrder(input, true)
	if err != nil {
		return nil, err
	}
	number, err := uc.config.NextOrderNumber(ctx, organizationID, normalized.BusinessType)
	if err != nil {
		return nil, err
	}
	audit := &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.create",
		Result:         "success",
		Details: map[string]string{
			"order.no":      number,
			"customer.id":   normalized.CustomerID.String(),
			"business_type": string(normalized.BusinessType),
		},
	}
	created, err := uc.repo.Create(ctx, organizationID, actorID, number, normalized, audit)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (uc *OrderUsecase) UpdateDraft(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedVersion uint64, input *Order) (*Order, error) {
	if id == uuid.Nil || expectedVersion == 0 {
		return nil, ErrOrderInvalidArgument
	}
	normalized, err := normalizeOrder(input, false)
	if err != nil {
		return nil, err
	}
	if normalized.ShipmentType == nil || *normalized.ShipmentType != OrderShipmentFCL {
		hasContainers, err := uc.repo.HasContainers(ctx, organizationID, id)
		if err != nil {
			return nil, err
		}
		if hasContainers {
			return nil, ErrOrderContainerShipmentType
		}
	}
	audit := &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.update",
		Result:         "success",
		Details:        map[string]string{"order.id": id.String()},
	}
	updated, err := uc.repo.UpdateDraft(ctx, organizationID, id, expectedVersion, normalized, audit)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func normalizeOrder(input *Order, creating bool) (*Order, error) {
	if input == nil || input.CustomerID == uuid.Nil || !input.BusinessType.Valid() || !input.TradeDirection.Valid() || !input.TradeTerm.Valid() || !input.PaymentTerm.Valid() {
		return nil, ErrOrderInvalidArgument
	}
	if input.BusinessType != OrderBusinessSE || input.TradeDirection != OrderTradeExport {
		return nil, ErrOrderBusinessUnsupported
	}
	output := *input
	output.CustomerReferenceNo = strings.TrimSpace(output.CustomerReferenceNo)
	output.InternalReferenceNo = strings.TrimSpace(output.InternalReferenceNo)
	output.ShipperShortName = strings.TrimSpace(output.ShipperShortName)
	output.ConsigneeShortName = strings.TrimSpace(output.ConsigneeShortName)
	output.ContractNo = strings.TrimSpace(output.ContractNo)
	output.CargoValue = strings.TrimSpace(output.CargoValue)
	output.CargoCurrency = strings.ToUpper(strings.TrimSpace(output.CargoCurrency))
	output.InsurancePremium = strings.TrimSpace(output.InsurancePremium)
	output.InsuranceCurrency = strings.ToUpper(strings.TrimSpace(output.InsuranceCurrency))
	output.UNNumber = strings.TrimSpace(output.UNNumber)
	output.HazardClass = strings.TrimSpace(output.HazardClass)
	output.FactoryName = strings.TrimSpace(output.FactoryName)
	output.CargoReadyAt = strings.TrimSpace(output.CargoReadyAt)
	output.LoadingTerms = strings.TrimSpace(output.LoadingTerms)
	output.DeclarationCutoffAt = strings.TrimSpace(output.DeclarationCutoffAt)
	output.ReceivedAt = strings.TrimSpace(output.ReceivedAt)
	output.VesselVoyage = strings.TrimSpace(output.VesselVoyage)
	output.ETD = strings.TrimSpace(output.ETD)
	output.ETA = strings.TrimSpace(output.ETA)
	output.SICutoff = strings.TrimSpace(output.SICutoff)
	output.DocCutoff = strings.TrimSpace(output.DocCutoff)
	output.CustomsCutoff = strings.TrimSpace(output.CustomsCutoff)
	output.VGMCutoff = strings.TrimSpace(output.VGMCutoff)
	output.GoodsDescription = strings.TrimSpace(output.GoodsDescription)
	output.TotalPackageUnit = strings.TrimSpace(output.TotalPackageUnit)
	output.SpecialRequirements = strings.TrimSpace(output.SpecialRequirements)
	output.OrderDate = strings.TrimSpace(output.OrderDate)
	output.Notes = strings.TrimSpace(output.Notes)
	output.BookingNotes = strings.TrimSpace(output.BookingNotes)
	output.AllocationNotes = strings.TrimSpace(output.AllocationNotes)
	output.OperationNotes = strings.TrimSpace(output.OperationNotes)
	if output.OrderDate == "" && creating {
		output.OrderDate = time.Now().UTC().Format(time.RFC3339)
	}
	if utf8.RuneCountInString(output.CustomerReferenceNo) > 100 || utf8.RuneCountInString(output.InternalReferenceNo) > 100 || utf8.RuneCountInString(output.ShipperShortName) > 200 || utf8.RuneCountInString(output.ConsigneeShortName) > 200 || utf8.RuneCountInString(output.ContractNo) > 100 || utf8.RuneCountInString(output.HazardClass) > 16 || utf8.RuneCountInString(output.FactoryName) > 200 || utf8.RuneCountInString(output.LoadingTerms) > 100 || utf8.RuneCountInString(output.VesselVoyage) > 100 || utf8.RuneCountInString(output.GoodsDescription) > 1000 || utf8.RuneCountInString(output.SpecialRequirements) > 1000 || utf8.RuneCountInString(output.Notes) > 1000 || utf8.RuneCountInString(output.BookingNotes) > 1000 || utf8.RuneCountInString(output.AllocationNotes) > 1000 || utf8.RuneCountInString(output.OperationNotes) > 1000 || output.TotalPackages != nil && *output.TotalPackages < 0 || output.TotalGrossWeightKg != nil && *output.TotalGrossWeightKg < 0 || output.TotalVolumeCbm != nil && *output.TotalVolumeCbm < 0 {
		return nil, ErrOrderInvalidArgument
	}
	roleCounts := make(map[OrderPersonnelRole]int, len(output.PersonnelAssignments))
	normalizedPersonnel := make([]*OrderPersonnel, 0, len(output.PersonnelAssignments))
	for _, assignment := range output.PersonnelAssignments {
		if assignment == nil || !assignment.Role.Valid() || assignment.Role == OrderPersonnelRoleCreator || assignment.UserID == uuid.Nil || assignment.OrganizationID == uuid.Nil {
			return nil, ErrOrderInvalidArgument
		}
		roleCounts[assignment.Role]++
		if roleCounts[assignment.Role] > 1 {
			return nil, ErrOrderInvalidArgument
		}
		normalizedAssignment := *assignment
		if creating {
			normalizedAssignment.Notification = NewOrderPersonnelNotification(assignment.UserID)
		} else {
			normalizedAssignment.Notification = nil
		}
		normalizedPersonnel = append(normalizedPersonnel, &normalizedAssignment)
	}
	output.PersonnelAssignments = normalizedPersonnel
	houseNumbers := make(map[string]struct{}, len(output.ShippingDocuments))
	masterAttributes := make(map[string][2]string, len(output.ShippingDocuments))
	for index, document := range output.ShippingDocuments {
		normalized, err := normalizeOrderShippingDocument(document)
		if err != nil {
			return nil, err
		}
		if _, exists := houseNumbers[strings.ToLower(normalized.HouseNo)]; exists {
			return nil, ErrOrderShippingDocumentExists
		}
		houseNumbers[strings.ToLower(normalized.HouseNo)] = struct{}{}
		masterKey := strings.ToLower(normalized.MasterNo)
		attributes := [2]string{stringPointerValue(normalized.MasterDocumentType), stringPointerValue(normalized.MasterReleaseMethod)}
		if existing, exists := masterAttributes[masterKey]; exists && existing != attributes {
			return nil, ErrOrderInvalidArgument
		}
		masterAttributes[masterKey] = attributes
		output.ShippingDocuments[index] = normalized
	}
	containerSpecs := make(map[uuid.UUID]struct{}, len(output.ContainerRequests))
	for _, request := range output.ContainerRequests {
		if request == nil || request.ContainerSpecID == uuid.Nil || request.Quantity < 1 || request.Quantity > 999 {
			return nil, ErrOrderInvalidArgument
		}
		if _, exists := containerSpecs[request.ContainerSpecID]; exists {
			return nil, ErrOrderInvalidArgument
		}
		containerSpecs[request.ContainerSpecID] = struct{}{}
	}
	if (output.CargoValue == "") != (output.CargoCurrency == "") || output.CargoValue != "" && (!cargoValuePattern.MatchString(output.CargoValue) || len(output.CargoCurrency) != 3) {
		return nil, ErrOrderInvalidArgument
	}
	if (output.InsurancePremium == "") != (output.InsuranceCurrency == "") || output.InsurancePremium != "" && (!cargoValuePattern.MatchString(output.InsurancePremium) || len(output.InsuranceCurrency) != 3) || output.UNNumber != "" && !unNumberPattern.MatchString(output.UNNumber) {
		return nil, ErrOrderInvalidArgument
	}
	for _, value := range []string{output.ETD, output.ETA, output.SICutoff, output.DocCutoff, output.CustomsCutoff, output.VGMCutoff, output.CargoReadyAt, output.DeclarationCutoffAt, output.ReceivedAt, output.OrderDate} {
		if value != "" {
			if _, err := time.Parse(time.RFC3339, value); err != nil {
				return nil, ErrOrderInvalidArgument
			}
		}
	}
	if output.ShipmentType != nil && !output.ShipmentType.Valid() || output.ContainerOwnership != nil && !output.ContainerOwnership.Valid() || output.ShipmentMode != nil && !output.ShipmentMode.Valid() {
		return nil, ErrOrderInvalidArgument
	}
	if output.ShipmentType != nil && *output.ShipmentType == OrderShipmentBreakBulk && (len(output.ContainerRequests) > 0 || output.VGMCutoff != "") {
		return nil, ErrOrderInvalidArgument
	}
	if err := validateUUIDSet(output.ServiceTypeIDs); err != nil {
		return nil, err
	}
	if err := validateUUIDSet(output.CargoCategoryIDs); err != nil {
		return nil, err
	}
	return &output, nil
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

var cargoValuePattern = regexp.MustCompile(`^(0|[1-9]\d{0,17})(\.\d{1,4})?$`)
var unNumberPattern = regexp.MustCompile(`^\d{4}$`)

func validateUUIDSet(values []uuid.UUID) error {
	seen := make(map[uuid.UUID]struct{}, len(values))
	for _, value := range values {
		if value == uuid.Nil {
			return ErrOrderInvalidArgument
		}
		if _, exists := seen[value]; exists {
			return ErrOrderInvalidArgument
		}
		seen[value] = struct{}{}
	}
	return nil
}
