package biz

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrOrderNotFound                  = errors.NotFound("ORDER_NOT_FOUND", "订单不存在")
	ErrOrderInvalidArgument           = errors.BadRequest("ORDER_INVALID_ARGUMENT", "订单字段不合法")
	ErrOrderNumberExists              = errors.Conflict("ORDER_NUMBER_EXISTS", "订单编号已存在")
	ErrOrderCustomerInvalid           = errors.BadRequest("ORDER_CUSTOMER_INVALID", "订单客户必须是启用的客户角色")
	ErrOrderStatusInvalid             = errors.BadRequest("ORDER_STATUS_INVALID", "订单状态不合法")
	ErrOrderStatusConflict            = errors.Conflict("ORDER_STATUS_CONFLICT", "订单状态已被其他操作修改")
	ErrOrderBusinessUnsupported       = errors.BadRequest("ORDER_BUSINESS_UNSUPPORTED", "当前仅支持海运出口订单")
	ErrOrderTerminationInvalid        = errors.BadRequest("ORDER_TERMINATION_INVALID", "订单终止状态流转不合法")
	ErrOrderClosureInvalid            = errors.BadRequest("ORDER_CLOSURE_INVALID", "订单结案状态流转不合法")
	ErrOrderClosureBlocked            = errors.Conflict("ORDER_CLOSURE_BLOCKED", "订单尚未满足结案条件")
	ErrOrderConsolidationShipmentType = errors.BadRequest("ORDER_CONSOLIDATION_SHIPMENT_TYPE_INVALID", "仅拼箱订单可查看自拼汇总")
)

type OrderBusinessType string

const (
	OrderBusinessSE   OrderBusinessType = "SE"
	OrderBusinessSI   OrderBusinessType = "SI"
	OrderBusinessAE   OrderBusinessType = "AE"
	OrderBusinessAI   OrderBusinessType = "AI"
	OrderBusinessLand OrderBusinessType = "LAND"
	OrderBusinessRail OrderBusinessType = "RAIL"
)

func (v OrderBusinessType) Valid() bool {
	return v == OrderBusinessSE || v == OrderBusinessSI || v == OrderBusinessAE || v == OrderBusinessAI || v == OrderBusinessLand || v == OrderBusinessRail
}

type OrderFlowStatus string

const (
	OrderFlowDraft                      OrderFlowStatus = "DRAFT"
	OrderFlowBooked                     OrderFlowStatus = "BOOKED"
	OrderFlowSpaceAllocated             OrderFlowStatus = "SPACE_ALLOCATED"
	OrderFlowTruckingArranged           OrderFlowStatus = "TRUCKING_ARRANGED"
	OrderFlowDocumentCutoff             OrderFlowStatus = "DOCUMENT_CUTOFF"
	OrderFlowCustomsDeclarationArranged OrderFlowStatus = "CUSTOMS_DECLARATION_ARRANGED"
	OrderFlowDocumentReleased           OrderFlowStatus = "DOCUMENT_RELEASED"
)

func (v OrderFlowStatus) Valid() bool {
	switch v {
	case OrderFlowDraft, OrderFlowBooked, OrderFlowSpaceAllocated, OrderFlowTruckingArranged, OrderFlowDocumentCutoff, OrderFlowCustomsDeclarationArranged, OrderFlowDocumentReleased:
		return true
	default:
		return false
	}
}

type OrderTerminationStatus string

const (
	OrderTerminationActive      OrderTerminationStatus = "ACTIVE"
	OrderTerminationTerminating OrderTerminationStatus = "TERMINATING"
	OrderTerminationTerminated  OrderTerminationStatus = "TERMINATED"
)

func (v OrderTerminationStatus) Valid() bool {
	return v == OrderTerminationActive || v == OrderTerminationTerminating || v == OrderTerminationTerminated
}

type OrderTerminationType string

const (
	OrderTerminationCustomerCancel  OrderTerminationType = "CUSTOMER_CANCEL"
	OrderTerminationCarrierCancel   OrderTerminationType = "CARRIER_CANCEL"
	OrderTerminationCustomsReturn   OrderTerminationType = "CUSTOMS_RETURN"
	OrderTerminationOperationCancel OrderTerminationType = "OPERATION_CANCEL"
	OrderTerminationOther           OrderTerminationType = "OTHER"
)

func (v OrderTerminationType) Valid() bool {
	return v == OrderTerminationCustomerCancel || v == OrderTerminationCarrierCancel || v == OrderTerminationCustomsReturn || v == OrderTerminationOperationCancel || v == OrderTerminationOther
}

type OrderClosureStatus string

const (
	OrderClosureOpen   OrderClosureStatus = "OPEN"
	OrderClosureClosed OrderClosureStatus = "CLOSED"
)

func (v OrderClosureStatus) Valid() bool { return v == OrderClosureOpen || v == OrderClosureClosed }

type OrderAllowedAction string

const (
	OrderActionEdit                OrderAllowedAction = "EDIT"
	OrderActionTransitionFlow      OrderAllowedAction = "TRANSITION_FLOW"
	OrderActionStartTermination    OrderAllowedAction = "START_TERMINATION"
	OrderActionCompleteTermination OrderAllowedAction = "COMPLETE_TERMINATION"
	OrderActionCancelTermination   OrderAllowedAction = "CANCEL_TERMINATION"
	OrderActionClose               OrderAllowedAction = "CLOSE"
	OrderActionReopen              OrderAllowedAction = "REOPEN"
)

type OrderTradeDirection string

const (
	OrderTradeExport OrderTradeDirection = "export"
	OrderTradeImport OrderTradeDirection = "import"
)

func (v OrderTradeDirection) Valid() bool { return v == OrderTradeExport || v == OrderTradeImport }

type OrderTradeTerm string

const (
	OrderTradeEXW OrderTradeTerm = "EXW"
	OrderTradeFCA OrderTradeTerm = "FCA"
	OrderTradeFOB OrderTradeTerm = "FOB"
	OrderTradeCFR OrderTradeTerm = "CFR"
	OrderTradeCIF OrderTradeTerm = "CIF"
	OrderTradeCPT OrderTradeTerm = "CPT"
	OrderTradeCIP OrderTradeTerm = "CIP"
	OrderTradeDAP OrderTradeTerm = "DAP"
	OrderTradeDPU OrderTradeTerm = "DPU"
	OrderTradeDDU OrderTradeTerm = "DDU"
	OrderTradeDDP OrderTradeTerm = "DDP"
	OrderTradeLDP OrderTradeTerm = "LDP"
)

func (v OrderTradeTerm) Valid() bool {
	switch v {
	case OrderTradeEXW, OrderTradeFCA, OrderTradeFOB, OrderTradeCFR, OrderTradeCIF, OrderTradeCPT, OrderTradeCIP, OrderTradeDAP, OrderTradeDPU, OrderTradeDDU, OrderTradeDDP, OrderTradeLDP:
		return true
	default:
		return false
	}
}

type OrderPaymentTerm string

const (
	OrderPaymentPrepaid OrderPaymentTerm = "PREPAID"
	OrderPaymentCollect OrderPaymentTerm = "COLLECT"
)

func (v OrderPaymentTerm) Valid() bool { return v == OrderPaymentPrepaid || v == OrderPaymentCollect }

type OrderShipmentType string

const (
	OrderShipmentFCL       OrderShipmentType = "FCL"
	OrderShipmentLCL       OrderShipmentType = "LCL"
	OrderShipmentBreakBulk OrderShipmentType = "BREAK_BULK"
)

func (v OrderShipmentType) Valid() bool {
	return v == OrderShipmentFCL || v == OrderShipmentLCL || v == OrderShipmentBreakBulk
}

type OrderContainerOwnership string

const (
	OrderContainerCOC OrderContainerOwnership = "COC"
	OrderContainerSOC OrderContainerOwnership = "SOC"
)

func (v OrderContainerOwnership) Valid() bool {
	return v == OrderContainerCOC || v == OrderContainerSOC
}

type OrderShipmentMode string

const (
	OrderShipmentTraditionalForwarding OrderShipmentMode = "TRADITIONAL_FORWARDING"
	OrderShipmentCrossBorder           OrderShipmentMode = "CROSS_BORDER"
)

func (v OrderShipmentMode) Valid() bool {
	return v == OrderShipmentTraditionalForwarding || v == OrderShipmentCrossBorder
}

type OrderReferenceType string

const (
	OrderReferenceCustomer OrderReferenceType = "customer"
	OrderReferenceInternal OrderReferenceType = "internal"
)

func (v OrderReferenceType) Valid() bool {
	return v == OrderReferenceCustomer || v == OrderReferenceInternal
}

type OrderNumberFilterType string

const (
	OrderNumberFilterOrder              OrderNumberFilterType = "order"
	OrderNumberFilterMaster             OrderNumberFilterType = "master"
	OrderNumberFilterConsolidatedMaster OrderNumberFilterType = "consolidated_master"
)

func (v OrderNumberFilterType) Valid() bool {
	return v == OrderNumberFilterOrder || v == OrderNumberFilterMaster || v == OrderNumberFilterConsolidatedMaster
}

type OrderTagMatchMode string

const (
	OrderTagMatchFuzzyOr  OrderTagMatchMode = "fuzzy_or"
	OrderTagMatchExactAnd OrderTagMatchMode = "exact_and"
)

func (v OrderTagMatchMode) Valid() bool {
	return v == OrderTagMatchFuzzyOr || v == OrderTagMatchExactAnd
}

type OrderDateRange struct {
	From        *time.Time
	ToExclusive *time.Time
}

type OrderPersonnelFilter struct {
	UserID         *uuid.UUID
	OrganizationID *uuid.UUID
}

type Order struct {
	ID                    uuid.UUID
	OrganizationID        uuid.UUID
	OrganizationName      string
	OrderNo               string
	CustomerID            uuid.UUID
	CustomerReferenceNo   string
	InternalReferenceNo   string
	ShipperShortName      string
	ConsigneeShortName    string
	CarrierID             *uuid.UUID
	BookingAgentID        *uuid.UUID
	ForeignAgentID        *uuid.UUID
	ShippingAgentID       *uuid.UUID
	ContractNo            string
	CargoValue            string
	CargoCurrency         string
	InsurancePremium      string
	InsuranceCurrency     string
	UNNumber              string
	HazardClass           string
	FactoryName           string
	CargoReadyAt          string
	LoadingTerms          string
	DeclarationCutoffAt   string
	ReceivedAt            string
	BusinessType          OrderBusinessType
	TradeDirection        OrderTradeDirection
	TradeTerm             OrderTradeTerm
	PaymentTerm           OrderPaymentTerm
	ShipmentType          *OrderShipmentType
	ContainerOwnership    *OrderContainerOwnership
	ShipmentMode          *OrderShipmentMode
	FlowStatus            OrderFlowStatus
	TerminationStatus     OrderTerminationStatus
	TerminationType       *OrderTerminationType
	TerminationReason     string
	TerminatedAt          *time.Time
	TerminatedBy          *uuid.UUID
	ClosureStatus         OrderClosureStatus
	ClosureReason         string
	ClosedAt              *time.Time
	ClosedBy              *uuid.UUID
	LockedAt              *time.Time
	IsShared              bool
	Tags                  []string
	Version               uint64
	HasActiveException    bool
	ActiveExceptionCount  int
	AllowedActions        []OrderAllowedAction
	ServiceTypeIDs        []uuid.UUID
	CargoCategoryIDs      []uuid.UUID
	OriginLocationID      *uuid.UUID
	DestinationLocationID *uuid.UUID
	DischargeLocationID   *uuid.UUID
	TransitLocationID     *uuid.UUID
	VesselVoyage          string
	ETD                   string
	ETA                   string
	SICutoff              string
	DocCutoff             string
	CustomsCutoff         string
	VGMCutoff             string
	GoodsDescription      string
	TotalPackages         *int
	TotalGrossWeightKg    *float64
	TotalVolumeCbm        *float64
	TotalPackageUnit      string
	SpecialRequirements   string
	OrderDate             string
	Notes                 string
	BookingNotes          string
	AllocationNotes       string
	OperationNotes        string
	PersonnelAssignments  []*OrderPersonnel
	ShippingDocuments     []*OrderShippingDocument
	ContainerRequests     []*OrderContainerRequest
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// OrderContainerRequest 表示订舱阶段的箱型箱量计划，不包含实际箱号。
type OrderContainerRequest struct {
	ID              uuid.UUID
	OrderID         uuid.UUID
	ContainerSpecID uuid.UUID
	Quantity        int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type OrderListOptions struct {
	Page                  int
	PageSize              int
	Keyword               string
	FlowStatus            OrderFlowStatus
	TerminationStatus     OrderTerminationStatus
	ClosureStatus         OrderClosureStatus
	HasActiveException    *bool
	BusinessType          OrderBusinessType
	BusinessTypes         []OrderBusinessType
	CustomerID            *uuid.UUID
	NumberType            OrderNumberFilterType
	NumberKeyword         string
	CreatedAtRange        OrderDateRange
	ETDRange              OrderDateRange
	ETARange              OrderDateRange
	StatusTimeRange       OrderDateRange
	LockedAtRange         OrderDateRange
	OriginLocationID      *uuid.UUID
	DestinationLocationID *uuid.UUID
	CarrierID             *uuid.UUID
	ConsigneeShortName    string
	ShipperShortName      string
	Operator              OrderPersonnelFilter
	Sales                 OrderPersonnelFilter
	CustomerService       OrderPersonnelFilter
	Creator               OrderPersonnelFilter
	Tags                  []string
	TagMatchMode          OrderTagMatchMode
	IsLocked              *bool
	IsShared              *bool
}

type OrderList struct {
	Items    []*Order
	Total    int
	Page     int
	PageSize int
}

type OrderCargoMeasurement struct {
	Packages      int
	GrossWeightKg float64
	VolumeCbm     float64
}

type OrderConsolidationMember struct {
	OrderID             uuid.UUID
	OrderNo             string
	CustomerReferenceNo string
	HouseNos            []string
	Entrusted           OrderCargoMeasurement
	Actual              OrderCargoMeasurement
}

type OrderConsolidationSummary struct {
	ConsolidationID uuid.UUID
	MasterNo        string
	Entrusted       OrderCargoMeasurement
	Actual          OrderCargoMeasurement
	Members         []*OrderConsolidationMember
}

type OrderReferenceCheck struct {
	ReferenceType  OrderReferenceType
	ReferenceNo    string
	CustomerID     *uuid.UUID
	ExcludeOrderID *uuid.UUID
}

type OrderReferenceMatch struct {
	OrderID uuid.UUID
	OrderNo string
}

type OrderPersonnelOption struct {
	UserID           uuid.UUID
	DisplayName      string
	OrganizationID   uuid.UUID
	OrganizationName string
}

type OrderRepo interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (*Order, error)
	Find(context.Context, uuid.UUID) (*Order, error)
	List(context.Context, []uuid.UUID, OrderListOptions) (*OrderList, error)
	FindReferenceDuplicate(context.Context, uuid.UUID, OrderReferenceCheck) (*OrderReferenceMatch, error)
	ListPersonnelOptions(context.Context, uuid.UUID, SelectorListOptions) (*PagedList[*OrderPersonnelOption], error)
	HasContainers(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	ListConsolidationSummaries(context.Context, uuid.UUID, uuid.UUID) ([]*OrderConsolidationSummary, error)
	Create(context.Context, uuid.UUID, uuid.UUID, string, *Order) (*Order, error)
	UpdateDraft(context.Context, uuid.UUID, uuid.UUID, uint64, *Order) (*Order, error)
	TransitionStatus(context.Context, uuid.UUID, uuid.UUID, uint64, OrderFlowStatus, string, uuid.UUID, *OrderStatusChangedEvent) (*Order, error)
	TransitionTermination(context.Context, uuid.UUID, uuid.UUID, uint64, OrderTerminationStatus, *OrderTerminationType, string, uuid.UUID, *OrderLifecycleChangedEvent) (*Order, error)
	ClosureReadiness(context.Context, uuid.UUID, uuid.UUID) (*OrderClosureReadiness, error)
	TransitionClosure(context.Context, uuid.UUID, uuid.UUID, uint64, OrderClosureStatus, string, uuid.UUID, *OrderLifecycleChangedEvent) (*Order, error)
}

type OrderUsecase struct {
	repo   OrderRepo
	config *OrderConfigUsecase
	audit  AuditRepo
}

func NewOrderUsecase(repo OrderRepo, config *OrderConfigUsecase, audit AuditRepo) *OrderUsecase {
	return &OrderUsecase{repo: repo, config: config, audit: audit}
}

func (uc *OrderUsecase) Get(ctx context.Context, organizationID, id uuid.UUID) (*Order, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrOrderNotFound
	}
	return uc.repo.Get(ctx, organizationID, id)
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
	if len(options.Tags) > 0 && !options.TagMatchMode.Valid() || len(options.Tags) == 0 && options.TagMatchMode != "" {
		return nil, ErrOrderInvalidArgument
	}
	cleanTags := make([]string, 0, len(options.Tags))
	for _, tag := range options.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || utf8.RuneCountInString(tag) > 64 {
			return nil, ErrOrderInvalidArgument
		}
		cleanTags = append(cleanTags, tag)
	}
	options.Tags = cleanTags
	if options.FlowStatus != "" && !options.FlowStatus.Valid() {
		return nil, ErrOrderInvalidArgument
	}
	if options.TerminationStatus != "" && !options.TerminationStatus.Valid() || options.ClosureStatus != "" && !options.ClosureStatus.Valid() {
		return nil, ErrOrderInvalidArgument
	}
	return uc.repo.List(ctx, organizationIDs, options)
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
	created, err := uc.repo.Create(ctx, organizationID, actorID, number, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "order.create", Result: "success", Details: map[string]string{"order.id": created.ID.String(), "order.no": created.OrderNo, "customer.id": created.CustomerID.String(), "business_type": string(created.BusinessType)}}); err != nil {
		return nil, fmt.Errorf("write order create audit: %w", err)
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
	updated, err := uc.repo.UpdateDraft(ctx, organizationID, id, expectedVersion, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "order.update", Result: "success", Details: map[string]string{"order.id": updated.ID.String(), "order.no": updated.OrderNo}}); err != nil {
		return nil, fmt.Errorf("write order update audit: %w", err)
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
	cleanTags := make([]string, 0, len(output.Tags))
	seenTags := make(map[string]struct{}, len(output.Tags))
	for _, tag := range output.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || utf8.RuneCountInString(tag) > 64 {
			return nil, ErrOrderInvalidArgument
		}
		if _, exists := seenTags[tag]; exists {
			continue
		}
		seenTags[tag] = struct{}{}
		cleanTags = append(cleanTags, tag)
	}
	output.Tags = cleanTags
	roleCounts := make(map[OrderPersonnelRole]int, len(output.PersonnelAssignments))
	for _, assignment := range output.PersonnelAssignments {
		if assignment == nil || !assignment.Role.Valid() || assignment.Role == OrderPersonnelRoleCreator || assignment.UserID == uuid.Nil || assignment.OrganizationID == uuid.Nil {
			return nil, ErrOrderInvalidArgument
		}
		roleCounts[assignment.Role]++
		if roleCounts[assignment.Role] > 1 {
			return nil, ErrOrderInvalidArgument
		}
	}
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
