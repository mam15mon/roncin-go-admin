package service

import (
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestOrderDateRangeFromAPIUsesInclusiveEndDate(t *testing.T) {
	result, err := orderDateRangeFromAPI("2026-08-01", "2026-08-31")
	if err != nil {
		t.Fatalf("orderDateRangeFromAPI() error = %v", err)
	}
	if result.From == nil || result.From.Format("2006-01-02") != "2026-08-01" {
		t.Fatalf("from = %v", result.From)
	}
	if result.ToExclusive == nil || result.ToExclusive.Format("2006-01-02") != "2026-09-01" {
		t.Fatalf("toExclusive = %v", result.ToExclusive)
	}
	if _, err := orderDateRangeFromAPI("2026-09-02", "2026-09-01"); err == nil {
		t.Fatal("倒置日期范围应返回参数错误")
	}
	if _, err := orderDateRangeFromAPI("2026/09/01", ""); err == nil {
		t.Fatal("非法日期格式应返回参数错误")
	}
}

func TestOrderPersonnelFilterFromAPIRequiresEmployee(t *testing.T) {
	userID := uuid.New()
	organizationID := uuid.New()
	result, err := orderPersonnelFilterFromAPI(userID.String(), organizationID.String())
	if err != nil {
		t.Fatalf("orderPersonnelFilterFromAPI() error = %v", err)
	}
	if result.UserID == nil || *result.UserID != userID || result.OrganizationID == nil || *result.OrganizationID != organizationID {
		t.Fatalf("personnel filter = %#v", result)
	}
	if _, err := orderPersonnelFilterFromAPI("", organizationID.String()); err == nil {
		t.Fatal("仅传部门时应返回参数错误")
	}
}


func TestReadableOrderBusinessTypesUsesScopedReadPermissions(t *testing.T) {
	seRead := access.OrderPermission(access.OrderBusinessSE, access.OrderRead)
	aiRead := access.OrderPermission(access.OrderBusinessAI, access.OrderRead)
	siCreate := access.OrderPermission(access.OrderBusinessSI, access.OrderCreate)
	principal := &biz.Principal{
		Permissions: []string{seRead, aiRead, siCreate},
		RoleScopes: []biz.RoleScope{
			{RoleCode: "operator", DataScope: biz.DataScopeOrganization},
			{RoleCode: "self", DataScope: biz.DataScopeSelf},
		},
		RolePermissions: map[string]map[string]struct{}{
			"operator": {seRead: {}, siCreate: {}},
			"self":     {aiRead: {}},
		},
	}

	got := readableOrderBusinessTypes(principal)
	want := []biz.OrderBusinessType{biz.OrderBusinessSE}
	if !slices.Equal(got, want) {
		t.Fatalf("readableOrderBusinessTypes() = %v, want %v", got, want)
	}
}

func TestOrderBusinessFieldsRoundTrip(t *testing.T) {
	customerID := uuid.New()
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
	declarationCutoffAt := "2026-08-22T16:00:00+08:00"
	shipperShortName := "华东发货人"
	consigneeShortName := "美西收货人"

	order, err := orderFromCreateRequest(&v1.CreateOrderRequest{
		CustomerId:   customerID.String(),
		BusinessType: v1.BusinessType_BUSINESS_TYPE_SE, TradeDirection: v1.TradeDirection_TRADE_DIRECTION_EXPORT,
		TradeTerm: v1.TradeTerm_TRADE_TERM_FOB, PaymentTerm: v1.PaymentTerm_PAYMENT_TERM_PREPAID,
		CustomerReferenceNo: &referenceNo, ForeignAgentId: &foreignAgentIDString,
		ContractNo: &contractNo, CargoValue: &cargoValue, CargoCurrency: &cargoCurrency,
		InternalReferenceNo: &internalReferenceNo, ShippingAgentId: &shippingAgentIDString,
		InsurancePremium: &insurancePremium, InsuranceCurrency: &insuranceCurrency,
		UnNumber: &unNumber, HazardClass: &hazardClass, FactoryName: &factoryName,
		CargoReadyAt: &cargoReadyAt, LoadingTerms: &loadingTerms,
		DeclarationCutoffAt: &declarationCutoffAt, ReceivedAt: &receivedAt,
		ShipperShortName: &shipperShortName, ConsigneeShortName: &consigneeShortName,
	})
	if err != nil {
		t.Fatalf("orderFromCreateRequest() error = %v", err)
	}
	if order.CustomerReferenceNo != referenceNo || order.InternalReferenceNo != internalReferenceNo || order.ForeignAgentID == nil || *order.ForeignAgentID != foreignAgentID || order.ShippingAgentID == nil || *order.ShippingAgentID != shippingAgentID || order.ContractNo != contractNo || order.CargoValue != cargoValue || order.CargoCurrency != cargoCurrency || order.InsurancePremium != insurancePremium || order.InsuranceCurrency != insuranceCurrency || order.UNNumber != unNumber || order.HazardClass != hazardClass || order.FactoryName != factoryName || order.CargoReadyAt != cargoReadyAt || order.LoadingTerms != loadingTerms || order.DeclarationCutoffAt != declarationCutoffAt || order.ReceivedAt != receivedAt || order.ShipperShortName != shipperShortName || order.ConsigneeShortName != consigneeShortName {
		t.Fatalf("converted order business fields = %#v", order)
	}

	apiOrder := orderToAPI(order)
	if apiOrder.GetCustomerReferenceNo() != referenceNo || apiOrder.GetInternalReferenceNo() != internalReferenceNo || apiOrder.GetForeignAgentId() != foreignAgentIDString || apiOrder.GetShippingAgentId() != shippingAgentIDString || apiOrder.GetContractNo() != contractNo || apiOrder.GetCargoValue() != cargoValue || apiOrder.GetCargoCurrency() != cargoCurrency || apiOrder.GetInsurancePremium() != insurancePremium || apiOrder.GetInsuranceCurrency() != insuranceCurrency || apiOrder.GetUnNumber() != unNumber || apiOrder.GetHazardClass() != hazardClass || apiOrder.GetFactoryName() != factoryName || apiOrder.GetCargoReadyAt() != cargoReadyAt || apiOrder.GetLoadingTerms() != loadingTerms || apiOrder.GetDeclarationCutoffAt() != declarationCutoffAt || apiOrder.GetReceivedAt() != receivedAt || apiOrder.GetShipperShortName() != shipperShortName || apiOrder.GetConsigneeShortName() != consigneeShortName {
		t.Fatalf("orderToAPI() business fields = %#v", apiOrder)
	}
}

func TestOrderPlanFieldsRoundTrip(t *testing.T) {
	customerID := uuid.New()
	containerSpecID := uuid.New()
	order, err := orderFromCreateRequest(&v1.CreateOrderRequest{
		CustomerId:   customerID.String(),
		BusinessType: v1.BusinessType_BUSINESS_TYPE_SE, TradeDirection: v1.TradeDirection_TRADE_DIRECTION_EXPORT,
		TradeTerm: v1.TradeTerm_TRADE_TERM_FOB, PaymentTerm: v1.PaymentTerm_PAYMENT_TERM_PREPAID,
		ShippingDocuments: []*v1.OrderShippingDocumentInput{
			{MasterNo: "MBL-001", HouseNo: "HBL-001"},
			{MasterNo: "MBL-001", HouseNo: "HBL-002"},
		},
		ContainerRequests: []*v1.OrderContainerRequestInput{{ContainerSpecId: containerSpecID.String(), Quantity: 2}},
	})
	if err != nil {
		t.Fatalf("orderFromCreateRequest() error = %v", err)
	}
	if len(order.ShippingDocuments) != 2 || order.ShippingDocuments[1].HouseNo != "HBL-002" || len(order.ContainerRequests) != 1 || order.ContainerRequests[0].Quantity != 2 {
		t.Fatalf("converted order plan fields = %#v, %#v", order.ShippingDocuments, order.ContainerRequests)
	}

	now := time.Now().UTC()
	order.ID = uuid.New()
	order.OrganizationID = uuid.New()
	order.CreatedAt = now
	order.UpdatedAt = now
	order.ShippingDocuments[0].ID = uuid.New()
	order.ShippingDocuments[0].OrderID = order.ID
	order.ShippingDocuments[0].Status = biz.OrderShippingDocumentStatusDraft
	order.ShippingDocuments[0].CreatedAt = now
	order.ShippingDocuments[0].UpdatedAt = now
	order.ContainerRequests[0].ID = uuid.New()
	order.ContainerRequests[0].OrderID = order.ID
	order.ContainerRequests[0].CreatedAt = now
	order.ContainerRequests[0].UpdatedAt = now
	apiOrder := orderToAPI(order)
	if len(apiOrder.GetShippingDocuments()) != 2 || apiOrder.GetShippingDocuments()[0].GetMasterNo() != "MBL-001" || len(apiOrder.GetContainerRequests()) != 1 || apiOrder.GetContainerRequests()[0].GetContainerSpecId() != containerSpecID.String() {
		t.Fatalf("orderToAPI() plan fields = %#v, %#v", apiOrder.GetShippingDocuments(), apiOrder.GetContainerRequests())
	}
}
