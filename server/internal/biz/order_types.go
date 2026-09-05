package biz

import (
	"time"

	"github.com/google/uuid"
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

// DisplayName 返回审批表单和审计中使用的稳定业务类型名称。
func (v OrderBusinessType) DisplayName() string {
	switch v {
	case OrderBusinessSE:
		return "海运出口（SE）"
	case OrderBusinessSI:
		return "海运进口（SI）"
	case OrderBusinessAE:
		return "空运出口（AE）"
	case OrderBusinessAI:
		return "空运进口（AI）"
	case OrderBusinessLand:
		return "陆运（LAND）"
	case OrderBusinessRail:
		return "铁路（RAIL）"
	default:
		return ""
	}
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

type OrderDateRange struct {
	From        *time.Time
	ToExclusive *time.Time
}

type OrderPersonnelFilter struct {
	UserID         *uuid.UUID
	OrganizationID *uuid.UUID
}

type Order struct {
	ID                     uuid.UUID
	OrganizationID         uuid.UUID
	OrganizationName       string
	OrderNo                string
	CustomerID             uuid.UUID
	CustomerReferenceNo    string
	InternalReferenceNo    string
	ShipperShortName       string
	ConsigneeShortName     string
	CarrierID              *uuid.UUID
	BookingAgentID         *uuid.UUID
	ForeignAgentID         *uuid.UUID
	ShippingAgentID        *uuid.UUID
	ContractNo             string
	CargoValue             string
	CargoCurrency          string
	InsurancePremium       string
	InsuranceCurrency      string
	UNNumber               string
	HazardClass            string
	FactoryName            string
	CargoReadyAt           string
	LoadingTerms           string
	DeclarationCutoffAt    string
	ReceivedAt             string
	BusinessType           OrderBusinessType
	TradeDirection         OrderTradeDirection
	TradeTerm              OrderTradeTerm
	PaymentTerm            OrderPaymentTerm
	ShipmentType           *OrderShipmentType
	ContainerOwnership     *OrderContainerOwnership
	ShipmentMode           *OrderShipmentMode
	FlowStatus             OrderFlowStatus
	TerminationStatus      OrderTerminationStatus
	TerminationType        *OrderTerminationType
	TerminationReason      string
	TerminatedAt           *time.Time
	TerminatedBy           *uuid.UUID
	ClosureStatus          OrderClosureStatus
	ClosureReason          string
	ClosedAt               *time.Time
	ClosedBy               *uuid.UUID
	LockedAt               *time.Time
	IsShared               bool
	Version                uint64
	Tags                   []*BusinessTagSummary
	HasActiveException     bool
	ActiveExceptionCount   int
	AllowedActions         []OrderAllowedAction
	ServiceTypeIDs         []uuid.UUID
	CargoCategoryIDs       []uuid.UUID
	OriginLocationID       *uuid.UUID
	DestinationLocationID  *uuid.UUID
	DischargeLocationID    *uuid.UUID
	TransitLocationID      *uuid.UUID
	VesselVoyage           string
	ETD                    string
	ETA                    string
	SICutoff               string
	DocCutoff              string
	CustomsCutoff          string
	VGMCutoff              string
	GoodsDescription       string
	TotalPackages          *int
	TotalGrossWeightKg     *float64
	TotalVolumeCbm         *float64
	TotalPackageUnit       string
	SpecialRequirements    string
	OrderDate              string
	Notes                  string
	BookingNotes           string
	AllocationNotes        string
	OperationNotes         string
	PersonnelAssignments   []*OrderPersonnel
	ShippingDocuments      []*OrderShippingDocument
	ContainerRequests      []*OrderContainerRequest
	SeaMasterBill          *SeaMasterBillSummary
	SeaMasterBillInput     *SeaMasterBillInput
	SeaDocumentStructure   *SeaDocumentStructure
	SeaDocumentLinkVersion *uint64
	SeaDocumentSummary     *SeaOrderDocumentSummary
	SeaDocumentInput       *SeaOrderDocumentInput
	CreatedAt              time.Time
	UpdatedAt              time.Time
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
	TagIDs                []uuid.UUID
	IsLocked              *bool
	IsShared              *bool
}

type OrderList = PagedList[*Order]

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
