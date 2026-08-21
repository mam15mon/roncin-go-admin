package biz

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrOrderNotFound        = errors.NotFound("ORDER_NOT_FOUND", "订单不存在")
	ErrOrderInvalidArgument = errors.BadRequest("ORDER_INVALID_ARGUMENT", "订单字段不合法")
	ErrOrderNumberExists    = errors.Conflict("ORDER_NUMBER_EXISTS", "订单编号已存在")
	ErrOrderCustomerInvalid = errors.BadRequest("ORDER_CUSTOMER_INVALID", "订单客户必须是启用的客户角色")
	ErrOrderStatusInvalid   = errors.BadRequest("ORDER_STATUS_INVALID", "订单状态不合法")
	ErrOrderStatusConflict  = errors.Conflict("ORDER_STATUS_CONFLICT", "订单状态已被其他操作修改")
	ErrOrderStatusTemplate  = errors.BadRequest("ORDER_STATUS_TEMPLATE_REQUIRED", "订单必须使用已发布状态模板")
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
	OrderShipmentConsolidation OrderShipmentMode = "CONSOLIDATION"
	OrderShipmentCrossBorder   OrderShipmentMode = "CROSS_BORDER"
)

func (v OrderShipmentMode) Valid() bool {
	return v == OrderShipmentConsolidation || v == OrderShipmentCrossBorder
}

type Order struct {
	ID                    uuid.UUID
	OrganizationID        uuid.UUID
	OrderNo               string
	CustomerID            uuid.UUID
	CarrierID             *uuid.UUID
	BookingAgentID        *uuid.UUID
	BusinessType          OrderBusinessType
	TradeDirection        OrderTradeDirection
	TradeTerm             OrderTradeTerm
	PaymentTerm           OrderPaymentTerm
	ShipmentType          *OrderShipmentType
	ContainerOwnership    *OrderContainerOwnership
	ShipmentMode          *OrderShipmentMode
	Status                string
	StatusTemplateID      uuid.UUID
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
	TotalPackageUnit      string
	SpecialRequirements   string
	OrderDate             string
	Notes                 string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type OrderListOptions struct {
	Page         int
	PageSize     int
	Keyword      string
	Status       string
	BusinessType OrderBusinessType
	CustomerID   *uuid.UUID
}

type OrderList struct {
	Items    []*Order
	Total    int
	Page     int
	PageSize int
}

type OrderRepo interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (*Order, error)
	List(context.Context, uuid.UUID, OrderListOptions) (*OrderList, error)
	Create(context.Context, uuid.UUID, uuid.UUID, string, *Order) (*Order, error)
	UpdateDraft(context.Context, uuid.UUID, uuid.UUID, string, *Order) (*Order, error)
	TransitionStatus(context.Context, uuid.UUID, uuid.UUID, string, string, string, uuid.UUID) (*Order, error)
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

func (uc *OrderUsecase) List(ctx context.Context, organizationID uuid.UUID, options OrderListOptions) (*OrderList, error) {
	if organizationID == uuid.Nil || options.Page < 1 || options.PageSize < 1 || options.PageSize > 100 || options.BusinessType != "" && !options.BusinessType.Valid() {
		return nil, ErrOrderInvalidArgument
	}
	options.Keyword = strings.TrimSpace(options.Keyword)
	options.Status = strings.ToUpper(strings.TrimSpace(options.Status))
	return uc.repo.List(ctx, organizationID, options)
}

func (uc *OrderUsecase) Create(ctx context.Context, organizationID, actorID uuid.UUID, input *Order) (*Order, error) {
	normalized, err := normalizeOrder(input, true)
	if err != nil {
		return nil, err
	}
	number, err := uc.config.NextNumber(ctx, organizationID, DocumentTypeOrder)
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

func (uc *OrderUsecase) UpdateDraft(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedStatus string, input *Order) (*Order, error) {
	if id == uuid.Nil || strings.TrimSpace(expectedStatus) == "" {
		return nil, ErrOrderInvalidArgument
	}
	normalized, err := normalizeOrder(input, false)
	if err != nil {
		return nil, err
	}
	updated, err := uc.repo.UpdateDraft(ctx, organizationID, id, strings.ToUpper(strings.TrimSpace(expectedStatus)), normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "order.update", Result: "success", Details: map[string]string{"order.id": updated.ID.String(), "order.no": updated.OrderNo}}); err != nil {
		return nil, fmt.Errorf("write order update audit: %w", err)
	}
	return updated, nil
}

func normalizeOrder(input *Order, creating bool) (*Order, error) {
	if input == nil || input.CustomerID == uuid.Nil || !input.BusinessType.Valid() || !input.TradeDirection.Valid() || !input.TradeTerm.Valid() || !input.PaymentTerm.Valid() || input.StatusTemplateID == uuid.Nil {
		return nil, ErrOrderInvalidArgument
	}
	output := *input
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
	if output.OrderDate == "" && creating {
		output.OrderDate = time.Now().UTC().Format(time.RFC3339)
	}
	if utf8.RuneCountInString(output.VesselVoyage) > 100 || utf8.RuneCountInString(output.GoodsDescription) > 1000 || utf8.RuneCountInString(output.SpecialRequirements) > 1000 || utf8.RuneCountInString(output.Notes) > 1000 || output.TotalPackages != nil && *output.TotalPackages < 0 {
		return nil, ErrOrderInvalidArgument
	}
	for _, value := range []string{output.ETD, output.ETA, output.SICutoff, output.DocCutoff, output.CustomsCutoff, output.VGMCutoff, output.OrderDate} {
		if value != "" {
			if _, err := time.Parse(time.RFC3339, value); err != nil {
				return nil, ErrOrderInvalidArgument
			}
		}
	}
	if output.ShipmentType != nil && !output.ShipmentType.Valid() || output.ContainerOwnership != nil && !output.ContainerOwnership.Valid() || output.ShipmentMode != nil && !output.ShipmentMode.Valid() {
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
