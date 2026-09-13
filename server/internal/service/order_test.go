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

func TestOrderOrganizationScopesKeepBusinessTypeAndOrganizationPaired(t *testing.T) {
	currentOrganizationID := uuid.New()
	beijingOrganizationID := uuid.New()
	beijingParentID := currentOrganizationID
	seRead := access.OrderPermission(access.OrderBusinessSE, access.OrderRead)
	aiRead := access.OrderPermission(access.OrderBusinessAI, access.OrderRead)
	principal := &biz.Principal{
		Organization: biz.Organization{ID: currentOrganizationID},
		OrganizationNodes: []biz.OrganizationScopeNode{
			{ID: currentOrganizationID},
			{ID: beijingOrganizationID, ParentID: &beijingParentID},
		},
		RoleGrants: []biz.RoleGrant{
			{RoleID: uuid.New(), RoleCode: "se-operator", DataScope: biz.DataScopeOrganizationTree, Permissions: map[string]struct{}{seRead: {}}},
			{RoleID: uuid.New(), RoleCode: "ai-operator", DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{aiRead: {}}},
		},
	}

	scopes, err := orderOrganizationScopesForOperation(principal, access.OrderRead, false, "")
	if err != nil {
		t.Fatalf("orderOrganizationScopesForOperation() error = %v", err)
	}
	if len(scopes) != 2 || scopes[0].BusinessType != biz.OrderBusinessSE || len(scopes[0].OrganizationIDs) != 2 || !slices.Contains(scopes[0].OrganizationIDs, currentOrganizationID) || !slices.Contains(scopes[0].OrganizationIDs, beijingOrganizationID) || scopes[1].BusinessType != biz.OrderBusinessAI || !slices.Equal(scopes[1].OrganizationIDs, []uuid.UUID{currentOrganizationID}) {
		t.Fatalf("order scopes = %#v", scopes)
	}
}

func TestCanModifyOrderUsesBusinessTypeSpecificUpdateScope(t *testing.T) {
	tianjinID := uuid.New()
	beijingID := uuid.New()
	orderRead := access.OrderPermission(access.OrderBusinessSE, access.OrderRead)
	orderUpdate := access.OrderPermission(access.OrderBusinessSE, access.OrderUpdate)
	principal := &biz.Principal{
		Organization:      biz.Organization{ID: tianjinID},
		OrganizationNodes: []biz.OrganizationScopeNode{{ID: tianjinID}, {ID: beijingID}},
		RoleGrants: []biz.RoleGrant{
			{RoleID: uuid.New(), RoleCode: "order-reader", DataScope: biz.DataScopeAll, Permissions: map[string]struct{}{orderRead: {}}},
			{RoleID: uuid.New(), RoleCode: "order-editor", DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{orderUpdate: {}}},
			{RoleID: uuid.New(), RoleCode: "finance-reader", DataScope: biz.DataScopeAll, Permissions: map[string]struct{}{"finance.bill.read": {}}},
		},
	}

	if !canModifyOrder(principal, &biz.Order{BusinessType: biz.OrderBusinessSE, OrganizationID: tianjinID}) {
		t.Fatal("当前组织的订单 update 权限应允许修改")
	}
	if canModifyOrder(principal, &biz.Order{BusinessType: biz.OrderBusinessSE, OrganizationID: beijingID}) {
		t.Fatal("订单 read 的北京范围或财务角色范围不得授予订单 update")
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
	receivedAt := "2026-08-23T10:00:00+08:00"
	declarationCutoffAt := "2026-08-22T16:00:00+08:00"
	shipperShortName := "华东发货人"
	consigneeShortName := "美西收货人"
	tradeTerm := v1.TradeTerm_TRADE_TERM_FOB

	order, err := orderFromCreateRequest(&v1.CreateOrderRequest{
		CustomerId:   customerID.String(),
		BusinessType: v1.BusinessType_BUSINESS_TYPE_SE, TradeDirection: v1.TradeDirection_TRADE_DIRECTION_EXPORT,
		TradeTerm: &tradeTerm, PaymentTerm: v1.PaymentTerm_PAYMENT_TERM_PREPAID,
		CustomerReferenceNo: &referenceNo, ForeignAgentId: &foreignAgentIDString,
		ContractNo: &contractNo, CargoValue: &cargoValue, CargoCurrency: &cargoCurrency,
		InternalReferenceNo: &internalReferenceNo, ShippingAgentId: &shippingAgentIDString,
		InsurancePremium: &insurancePremium, InsuranceCurrency: &insuranceCurrency,
		UnNumber: &unNumber, HazardClass: &hazardClass, FactoryName: &factoryName,
		CargoReadyAt:        &cargoReadyAt,
		DeclarationCutoffAt: &declarationCutoffAt, ReceivedAt: &receivedAt,
		ShipperShortName: &shipperShortName, ConsigneeShortName: &consigneeShortName,
	})
	if err != nil {
		t.Fatalf("orderFromCreateRequest() error = %v", err)
	}
	if order.CustomerReferenceNo != referenceNo || order.InternalReferenceNo != internalReferenceNo || order.ForeignAgentID == nil || *order.ForeignAgentID != foreignAgentID || order.ShippingAgentID == nil || *order.ShippingAgentID != shippingAgentID || order.ContractNo != contractNo || order.CargoValue != cargoValue || order.CargoCurrency != cargoCurrency || order.InsurancePremium != insurancePremium || order.InsuranceCurrency != insuranceCurrency || order.UNNumber != unNumber || order.HazardClass != hazardClass || order.FactoryName != factoryName || order.CargoReadyAt != cargoReadyAt || order.DeclarationCutoffAt != declarationCutoffAt || order.ReceivedAt != receivedAt || order.ShipperShortName != shipperShortName || order.ConsigneeShortName != consigneeShortName {
		t.Fatalf("converted order business fields = %#v", order)
	}

	apiOrder := orderToAPI(order)
	if apiOrder.GetCustomerReferenceNo() != referenceNo || apiOrder.GetInternalReferenceNo() != internalReferenceNo || apiOrder.GetForeignAgentId() != foreignAgentIDString || apiOrder.GetShippingAgentId() != shippingAgentIDString || apiOrder.GetContractNo() != contractNo || apiOrder.GetCargoValue() != cargoValue || apiOrder.GetCargoCurrency() != cargoCurrency || apiOrder.GetInsurancePremium() != insurancePremium || apiOrder.GetInsuranceCurrency() != insuranceCurrency || apiOrder.GetUnNumber() != unNumber || apiOrder.GetHazardClass() != hazardClass || apiOrder.GetFactoryName() != factoryName || apiOrder.GetCargoReadyAt() != cargoReadyAt || apiOrder.GetDeclarationCutoffAt() != declarationCutoffAt || apiOrder.GetReceivedAt() != receivedAt || apiOrder.GetShipperShortName() != shipperShortName || apiOrder.GetConsigneeShortName() != consigneeShortName {
		t.Fatalf("orderToAPI() business fields = %#v", apiOrder)
	}
}

func TestOrderPlanFieldsRoundTrip(t *testing.T) {
	customerID := uuid.New()
	containerSpecID := uuid.New()
	order, err := orderFromCreateRequest(&v1.CreateOrderRequest{
		CustomerId:   customerID.String(),
		BusinessType: v1.BusinessType_BUSINESS_TYPE_SE, TradeDirection: v1.TradeDirection_TRADE_DIRECTION_EXPORT,
		PaymentTerm: v1.PaymentTerm_PAYMENT_TERM_PREPAID,
		ShippingDocuments: []*v1.OrderShippingDocumentInput{
			{HouseNo: "HBL-001"},
			{HouseNo: "HBL-002"},
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
	if len(apiOrder.GetShippingDocuments()) != 2 || apiOrder.GetShippingDocuments()[0].GetHouseNo() != "HBL-001" || len(apiOrder.GetContainerRequests()) != 1 || apiOrder.GetContainerRequests()[0].GetContainerSpecId() != containerSpecID.String() {
		t.Fatalf("orderToAPI() plan fields = %#v, %#v", apiOrder.GetShippingDocuments(), apiOrder.GetContainerRequests())
	}
}

func TestSeaTransportExecutionToAPIOmitsUnsetUUIDs(t *testing.T) {
	shippingLineID := uuid.New()
	item := &biz.SeaTransportExecution{ID: uuid.New(), ShippingLineID: shippingLineID}
	output := seaTransportExecutionToAPI(item)
	if output.GetShippingLineId() != shippingLineID.String() || output.OriginLocationId != nil || output.DischargeLocationId != nil {
		t.Fatalf("未设置的运输执行 UUID 不应输出全零值: %#v", output)
	}

	originID, dischargeID := uuid.New(), uuid.New()
	item.OriginLocationID = originID
	item.DischargeLocationID = dischargeID
	output = seaTransportExecutionToAPI(item)
	if output.GetShippingLineId() != shippingLineID.String() || output.GetOriginLocationId() != originID.String() || output.GetDischargeLocationId() != dischargeID.String() {
		t.Fatalf("已设置的运输执行 UUID 转换错误: %#v", output)
	}
}
