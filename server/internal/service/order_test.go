package service

import (
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
)

func TestOrderBusinessFieldsRoundTrip(t *testing.T) {
	customerID := uuid.New()
	templateID := uuid.New()
	foreignAgentID := uuid.New()
	referenceNo := "CUST-001"
	contractNo := "CONTRACT-001"
	cargoValue := "100000.25"
	cargoCurrency := "USD"
	foreignAgentIDString := foreignAgentID.String()

	order, err := orderFromCreateRequest(&v1.CreateOrderRequest{
		CustomerId: customerID.String(), StatusTemplateId: templateID.String(),
		BusinessType: v1.BusinessType_BUSINESS_TYPE_SE, TradeDirection: v1.TradeDirection_TRADE_DIRECTION_EXPORT,
		TradeTerm: v1.TradeTerm_TRADE_TERM_FOB, PaymentTerm: v1.PaymentTerm_PAYMENT_TERM_PREPAID,
		CustomerReferenceNo: &referenceNo, ForeignAgentId: &foreignAgentIDString,
		ContractNo: &contractNo, CargoValue: &cargoValue, CargoCurrency: &cargoCurrency,
	})
	if err != nil {
		t.Fatalf("orderFromCreateRequest() error = %v", err)
	}
	if order.CustomerReferenceNo != referenceNo || order.ForeignAgentID == nil || *order.ForeignAgentID != foreignAgentID || order.ContractNo != contractNo || order.CargoValue != cargoValue || order.CargoCurrency != cargoCurrency {
		t.Fatalf("converted order business fields = %#v", order)
	}

	apiOrder := orderToAPI(order)
	if apiOrder.GetCustomerReferenceNo() != referenceNo || apiOrder.GetForeignAgentId() != foreignAgentIDString || apiOrder.GetContractNo() != contractNo || apiOrder.GetCargoValue() != cargoValue || apiOrder.GetCargoCurrency() != cargoCurrency {
		t.Fatalf("orderToAPI() business fields = %#v", apiOrder)
	}
}
