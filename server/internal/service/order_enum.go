package service

import (
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func orderFlowStatusFromAPI(value v1.OrderFlowStatus) biz.OrderFlowStatus {
	switch value {
	case v1.OrderFlowStatus_ORDER_FLOW_STATUS_DRAFT:
		return biz.OrderFlowDraft
	case v1.OrderFlowStatus_ORDER_FLOW_STATUS_BOOKED:
		return biz.OrderFlowBooked
	case v1.OrderFlowStatus_ORDER_FLOW_STATUS_SPACE_ALLOCATED:
		return biz.OrderFlowSpaceAllocated
	case v1.OrderFlowStatus_ORDER_FLOW_STATUS_TRUCKING_ARRANGED:
		return biz.OrderFlowTruckingArranged
	case v1.OrderFlowStatus_ORDER_FLOW_STATUS_DOCUMENT_CUTOFF:
		return biz.OrderFlowDocumentCutoff
	case v1.OrderFlowStatus_ORDER_FLOW_STATUS_CUSTOMS_DECLARATION_ARRANGED:
		return biz.OrderFlowCustomsDeclarationArranged
	case v1.OrderFlowStatus_ORDER_FLOW_STATUS_DOCUMENT_RELEASED:
		return biz.OrderFlowDocumentReleased
	default:
		return ""
	}
}

func orderFlowStatusToAPI(value biz.OrderFlowStatus) v1.OrderFlowStatus {
	return map[biz.OrderFlowStatus]v1.OrderFlowStatus{
		biz.OrderFlowDraft: v1.OrderFlowStatus_ORDER_FLOW_STATUS_DRAFT, biz.OrderFlowBooked: v1.OrderFlowStatus_ORDER_FLOW_STATUS_BOOKED,
		biz.OrderFlowSpaceAllocated: v1.OrderFlowStatus_ORDER_FLOW_STATUS_SPACE_ALLOCATED, biz.OrderFlowTruckingArranged: v1.OrderFlowStatus_ORDER_FLOW_STATUS_TRUCKING_ARRANGED,
		biz.OrderFlowDocumentCutoff: v1.OrderFlowStatus_ORDER_FLOW_STATUS_DOCUMENT_CUTOFF, biz.OrderFlowCustomsDeclarationArranged: v1.OrderFlowStatus_ORDER_FLOW_STATUS_CUSTOMS_DECLARATION_ARRANGED,
		biz.OrderFlowDocumentReleased: v1.OrderFlowStatus_ORDER_FLOW_STATUS_DOCUMENT_RELEASED,
	}[value]
}

func orderTerminationStatusFromAPI(value v1.OrderTerminationStatus) biz.OrderTerminationStatus {
	return map[v1.OrderTerminationStatus]biz.OrderTerminationStatus{
		v1.OrderTerminationStatus_ORDER_TERMINATION_STATUS_ACTIVE: biz.OrderTerminationActive, v1.OrderTerminationStatus_ORDER_TERMINATION_STATUS_TERMINATING: biz.OrderTerminationTerminating,
		v1.OrderTerminationStatus_ORDER_TERMINATION_STATUS_TERMINATED: biz.OrderTerminationTerminated,
	}[value]
}

func orderTerminationStatusToAPI(value biz.OrderTerminationStatus) v1.OrderTerminationStatus {
	return map[biz.OrderTerminationStatus]v1.OrderTerminationStatus{
		biz.OrderTerminationActive: v1.OrderTerminationStatus_ORDER_TERMINATION_STATUS_ACTIVE, biz.OrderTerminationTerminating: v1.OrderTerminationStatus_ORDER_TERMINATION_STATUS_TERMINATING,
		biz.OrderTerminationTerminated: v1.OrderTerminationStatus_ORDER_TERMINATION_STATUS_TERMINATED,
	}[value]
}

func orderTerminationTypeFromAPI(value *v1.OrderTerminationType) *biz.OrderTerminationType {
	if value == nil {
		return nil
	}
	mapped := map[v1.OrderTerminationType]biz.OrderTerminationType{
		v1.OrderTerminationType_ORDER_TERMINATION_TYPE_CUSTOMER_CANCEL: biz.OrderTerminationCustomerCancel, v1.OrderTerminationType_ORDER_TERMINATION_TYPE_CARRIER_CANCEL: biz.OrderTerminationCarrierCancel,
		v1.OrderTerminationType_ORDER_TERMINATION_TYPE_CUSTOMS_RETURN: biz.OrderTerminationCustomsReturn, v1.OrderTerminationType_ORDER_TERMINATION_TYPE_OPERATION_CANCEL: biz.OrderTerminationOperationCancel,
		v1.OrderTerminationType_ORDER_TERMINATION_TYPE_OTHER: biz.OrderTerminationOther,
	}[*value]
	if mapped == "" {
		return nil
	}
	return &mapped
}

func orderTerminationTypeToAPI(value *biz.OrderTerminationType) *v1.OrderTerminationType {
	if value == nil {
		return nil
	}
	mapped := map[biz.OrderTerminationType]v1.OrderTerminationType{
		biz.OrderTerminationCustomerCancel: v1.OrderTerminationType_ORDER_TERMINATION_TYPE_CUSTOMER_CANCEL, biz.OrderTerminationCarrierCancel: v1.OrderTerminationType_ORDER_TERMINATION_TYPE_CARRIER_CANCEL,
		biz.OrderTerminationCustomsReturn: v1.OrderTerminationType_ORDER_TERMINATION_TYPE_CUSTOMS_RETURN, biz.OrderTerminationOperationCancel: v1.OrderTerminationType_ORDER_TERMINATION_TYPE_OPERATION_CANCEL,
		biz.OrderTerminationOther: v1.OrderTerminationType_ORDER_TERMINATION_TYPE_OTHER,
	}[*value]
	return &mapped
}

func orderClosureStatusFromAPI(value v1.OrderClosureStatus) biz.OrderClosureStatus {
	return map[v1.OrderClosureStatus]biz.OrderClosureStatus{v1.OrderClosureStatus_ORDER_CLOSURE_STATUS_OPEN: biz.OrderClosureOpen, v1.OrderClosureStatus_ORDER_CLOSURE_STATUS_CLOSED: biz.OrderClosureClosed}[value]
}

func orderClosureStatusToAPI(value biz.OrderClosureStatus) v1.OrderClosureStatus {
	return map[biz.OrderClosureStatus]v1.OrderClosureStatus{biz.OrderClosureOpen: v1.OrderClosureStatus_ORDER_CLOSURE_STATUS_OPEN, biz.OrderClosureClosed: v1.OrderClosureStatus_ORDER_CLOSURE_STATUS_CLOSED}[value]
}

func orderAllowedActionsToAPI(values []biz.OrderAllowedAction) []v1.OrderAllowedAction {
	mapping := map[biz.OrderAllowedAction]v1.OrderAllowedAction{
		biz.OrderActionEdit: v1.OrderAllowedAction_ORDER_ALLOWED_ACTION_EDIT, biz.OrderActionTransitionFlow: v1.OrderAllowedAction_ORDER_ALLOWED_ACTION_TRANSITION_FLOW,
		biz.OrderActionStartTermination: v1.OrderAllowedAction_ORDER_ALLOWED_ACTION_START_TERMINATION, biz.OrderActionCompleteTermination: v1.OrderAllowedAction_ORDER_ALLOWED_ACTION_COMPLETE_TERMINATION,
		biz.OrderActionCancelTermination: v1.OrderAllowedAction_ORDER_ALLOWED_ACTION_CANCEL_TERMINATION, biz.OrderActionClose: v1.OrderAllowedAction_ORDER_ALLOWED_ACTION_CLOSE,
		biz.OrderActionReopen: v1.OrderAllowedAction_ORDER_ALLOWED_ACTION_REOPEN,
	}
	result := make([]v1.OrderAllowedAction, 0, len(values))
	for _, value := range values {
		result = append(result, mapping[value])
	}
	return result
}

func orderFlowStatusesToAPI(values []biz.OrderFlowStatus) []v1.OrderFlowStatus {
	result := make([]v1.OrderFlowStatus, 0, len(values))
	for _, value := range values {
		result = append(result, orderFlowStatusToAPI(value))
	}
	return result
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
