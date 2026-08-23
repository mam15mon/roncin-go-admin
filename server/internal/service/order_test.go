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
	shippingAgentID := uuid.New()
	referenceNo := "CUST-001"
	contractNo := "CONTRACT-001"
	cargoValue := "100000.25"
	cargoCurrency := "USD"
	foreignAgentIDString := foreignAgentID.String()
	shippingAgentIDString := shippingAgentID.String()
	internalReferenceNo := "INTERNAL-001"
	insurancePremium := "100.50"
	insuranceCurrency := "CNY"
	unNumber := "1234"
	hazardClass := "3"
	factoryName := "测试工厂"
	cargoReadyAt := "2026-08-23T12:00:00+08:00"
	loadingTerms := "CY-CY"
	receivedAt := "2026-08-23T10:00:00+08:00"

	order, err := orderFromCreateRequest(&v1.CreateOrderRequest{
		CustomerId: customerID.String(), StatusTemplateId: templateID.String(),
		BusinessType: v1.BusinessType_BUSINESS_TYPE_SE, TradeDirection: v1.TradeDirection_TRADE_DIRECTION_EXPORT,
		TradeTerm: v1.TradeTerm_TRADE_TERM_FOB, PaymentTerm: v1.PaymentTerm_PAYMENT_TERM_PREPAID,
		CustomerReferenceNo: &referenceNo, ForeignAgentId: &foreignAgentIDString,
		ContractNo: &contractNo, CargoValue: &cargoValue, CargoCurrency: &cargoCurrency,
		InternalReferenceNo: &internalReferenceNo, ShippingAgentId: &shippingAgentIDString,
		InsurancePremium: &insurancePremium, InsuranceCurrency: &insuranceCurrency,
		UnNumber: &unNumber, HazardClass: &hazardClass, FactoryName: &factoryName,
		CargoReadyAt: &cargoReadyAt, LoadingTerms: &loadingTerms,
		ReceivedAt: &receivedAt,
	})
	if err != nil {
		t.Fatalf("orderFromCreateRequest() error = %v", err)
	}
	if order.CustomerReferenceNo != referenceNo || order.InternalReferenceNo != internalReferenceNo || order.ForeignAgentID == nil || *order.ForeignAgentID != foreignAgentID || order.ShippingAgentID == nil || *order.ShippingAgentID != shippingAgentID || order.ContractNo != contractNo || order.CargoValue != cargoValue || order.CargoCurrency != cargoCurrency || order.InsurancePremium != insurancePremium || order.InsuranceCurrency != insuranceCurrency || order.UNNumber != unNumber || order.HazardClass != hazardClass || order.FactoryName != factoryName || order.CargoReadyAt != cargoReadyAt || order.LoadingTerms != loadingTerms || order.ReceivedAt != receivedAt {
		t.Fatalf("converted order business fields = %#v", order)
	}

	apiOrder := orderToAPI(order)
	if apiOrder.GetCustomerReferenceNo() != referenceNo || apiOrder.GetInternalReferenceNo() != internalReferenceNo || apiOrder.GetForeignAgentId() != foreignAgentIDString || apiOrder.GetShippingAgentId() != shippingAgentIDString || apiOrder.GetContractNo() != contractNo || apiOrder.GetCargoValue() != cargoValue || apiOrder.GetCargoCurrency() != cargoCurrency || apiOrder.GetInsurancePremium() != insurancePremium || apiOrder.GetInsuranceCurrency() != insuranceCurrency || apiOrder.GetUnNumber() != unNumber || apiOrder.GetHazardClass() != hazardClass || apiOrder.GetFactoryName() != factoryName || apiOrder.GetCargoReadyAt() != cargoReadyAt || apiOrder.GetLoadingTerms() != loadingTerms || apiOrder.GetReceivedAt() != receivedAt {
		t.Fatalf("orderToAPI() business fields = %#v", apiOrder)
	}
}
