package server

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	khttp "github.com/go-kratos/kratos/v3/transport/http"
	resourcev1 "github.com/roncin/roncin-go-admin/server/api/enterprise_resource/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/service"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// 使用生产构造函数检查实际注册，避免只检查生成源码却漏掉接线残留。
func TestBusinessTagRetirementRegistrations(t *testing.T) {
	configuration := &conf.Server{Http: &conf.Server_HTTP{}, Grpc: &conf.Server_GRPC{}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	httpServer := NewHTTPServer(
		configuration,                           // c
		&service.EnterpriseResourceService{},    // enterpriseResource
		&service.AuthService{},                  // auth
		&service.PasskeyService{},               // passkey
		&service.PartnerService{},               // partner
		&service.AdminService{},                 // admin
		&service.MasterDataService{},            // masterData
		&service.AirExportService{},             // airExport
		&service.AirWeightService{},             // airWeight
		&service.OrderService{},                 // order
		&service.OrderLockService{},             // orderLock
		&service.OrderAttachmentService{},       // orderAttachment
		&service.OrderPersonnelService{},        // orderPersonnel
		&service.BackgroundTaskService{},        // backgroundTask
		&service.OrderContainerService{},        // orderContainer
		&service.OrderCargoItemService{},        // orderCargoItem
		&service.OrderShippingDocumentService{}, // shippingDocument
		&service.SeaDocumentService{},           // seaDocument
		&service.SeaSharedContainerService{},    // seaSharedContainer
		&service.SeaOrderChangeService{},        // seaOrderChange
		&service.OrderAbnormalCaseService{},     // abnormalCase
		&service.OrderReleasePodService{},       // releasePod
		&service.ExchangeRateService{},          // exchangeRate
		&service.FeeCatalogService{},            // feeCatalog
		&service.OrderFeeService{},              // orderFee
		&service.SettlementService{},            // settlement
		&service.WorkbenchService{},             // workbench
		nil,                                     // authUsecase
		nil,                                     // orderUsecase
		nil,                                     // partnerUsecase
		nil,                                     // approvalUsecase
		nil,                                     // policy
		nil,                                     // readiness
		logger,                                  // logger
	)
	routes := map[string]bool{}
	if err := httpServer.WalkRoute(func(route khttp.RouteInfo) error {
		routes[route.Method+" "+route.Path] = true
		if route.Path == "/api/v1/finance/organization-options" {
			t.Errorf("财务公司候选 HTTP 入口仍注册: %s", route.Path)
		}
		if strings.Contains(route.Path, "tag-") || strings.Contains(route.Path, "tags/") || strings.Contains(route.Path, "enterprise-tag-groups") {
			t.Errorf("标签 HTTP 入口仍注册: %s %s", route.Method, route.Path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{"GET /api/v1/orders", "GET /api/v1/orders/{order_id}/fees", "GET /api/v1/finance/bills", "GET /api/v1/finance/fees", "GET /api/v1/finance/settlement-party-options", "GET /api/v1/enterprise-resources"} {
		if !routes[route] {
			t.Errorf("相邻业务 HTTP 入口缺失: %s", route)
		}
	}
	grpcServer := NewGRPCServer(
		configuration,                           // c
		&service.EnterpriseResourceService{},    // enterpriseResource
		&service.AuthService{},                  // auth
		&service.PasskeyService{},               // passkey
		&service.PartnerService{},               // partner
		&service.AdminService{},                 // admin
		&service.MasterDataService{},            // masterData
		&service.AirExportService{},             // airExport
		&service.AirWeightService{},             // airWeight
		&service.OrderService{},                 // order
		&service.OrderLockService{},             // orderLock
		&service.OrderAttachmentService{},       // orderAttachment
		&service.OrderPersonnelService{},        // orderPersonnel
		&service.BackgroundTaskService{},        // backgroundTask
		&service.OrderContainerService{},        // orderContainer
		&service.OrderCargoItemService{},        // orderCargoItem
		&service.OrderShippingDocumentService{}, // shippingDocument
		&service.SeaDocumentService{},           // seaDocument
		&service.SeaSharedContainerService{},    // seaSharedContainer
		&service.SeaOrderChangeService{},        // seaOrderChange
		&service.OrderAbnormalCaseService{},     // abnormalCase
		&service.OrderReleasePodService{},       // releasePod
		&service.ExchangeRateService{},          // exchangeRate
		&service.FeeCatalogService{},            // feeCatalog
		&service.OrderFeeService{},              // orderFee
		&service.SettlementService{},            // settlement
		&service.WorkbenchService{},             // workbench
		nil,                                     // authUsecase
		nil,                                     // orderUsecase
		nil,                                     // partnerUsecase
		nil,                                     // policy
		logger,                                  // logger
	)
	defer grpcServer.Server.Stop()
	infos := grpcServer.GetServiceInfo()
	if _, exists := infos["order.v1.OrderTagService"]; exists {
		t.Error("订单标签服务仍注册")
	}
	for _, name := range []string{"order.v1.OrderService", "order.v1.OrderFeeService", "finance.v1.SettlementService", "enterprise_resource.v1.EnterpriseResourceService"} {
		info, exists := infos[name]
		if !exists {
			t.Errorf("相邻业务 gRPC 服务缺失: %s", name)
		}
		for _, method := range info.Methods {
			if method.Name == "ListFinanceOrganizationOptions" {
				t.Errorf("财务公司候选 RPC 仍注册: %s/%s", name, method.Name)
			}
			if strings.Contains(method.Name, "Tag") {
				t.Errorf("标签 RPC 仍注册: %s/%s", name, method.Name)
			}
		}
	}
	for name, required := range map[string]string{
		"order.v1.OrderService":                            "ListOrders",
		"order.v1.OrderFeeService":                         "ListFees",
		"finance.v1.SettlementService":                     "ListBills",
		"enterprise_resource.v1.EnterpriseResourceService": "ListEnterpriseResources",
	} {
		found := false
		for _, method := range infos[name].Methods {
			if method.Name == required {
				found = true
			}
		}
		if !found {
			t.Errorf("相邻业务 RPC 缺失: %s/%s", name, required)
		}
	}
	partyOptionsFound := false
	for _, method := range infos["finance.v1.SettlementService"].Methods {
		if method.Name == "ListFinanceSettlementPartyOptions" {
			partyOptionsFound = true
		}
	}
	if !partyOptionsFound {
		t.Error("结算单位候选 RPC 缺失")
	}
	for _, permission := range access.Manifest() {
		if permission.Key == "system.finance.fee.tag" {
			t.Error("标签权限仍在清单内")
		}
	}
}

func TestBusinessTagRetirementReservedContract(t *testing.T) {
	for _, field := range []struct {
		message protoreflect.FullName
		name    protoreflect.Name
		number  protoreflect.FieldNumber
	}{
		{"order.v1.Order", "tags", 79},
		{"order.v1.ListOrdersRequest", "tag_ids", 35},
		{"order.v1.OrderFee", "tags", 36},
		{"finance.v1.ListFeeLedgerRequest", "tag_ids", 15},
		{"finance.v1.FeeLedgerItem", "tags", 34},
		{"finance.v1.ListBillsRequest", "tag_ids", 10},
		{"finance.v1.FinanceBill", "tags", 36},
		{"enterprise_resource.v1.EnterpriseResource", "tag", 13},
		{"enterprise_resource.v1.EnterpriseResourceInput", "tag", 12},
	} {
		descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(field.message)
		if err != nil {
			t.Fatal(err)
		}
		message := descriptor.(protoreflect.MessageDescriptor)
		if message.Fields().ByName(field.name) != nil || message.Fields().ByNumber(field.number) != nil {
			t.Errorf("标签字段仍存在: %s.%s", field.message, field.name)
		}
		if !message.ReservedNames().Has(field.name) || !message.ReservedRanges().Has(field.number) {
			t.Errorf("退役字段未保留编号与名称: %s.%s", field.message, field.name)
		}
	}
	enum := resourcev1.EnterpriseResourceType(0).Descriptor()
	if enum.Values().ByNumber(4) != nil || !enum.ReservedRanges().Has(4) || !enum.ReservedNames().Has("ENTERPRISE_RESOURCE_TYPE_TAG") {
		t.Error("标签枚举4未正确退役并保留")
	}
	for name, number := range map[protoreflect.Name]protoreflect.EnumNumber{"ENTERPRISE_RESOURCE_TYPE_SHIPPER": 5, "ENTERPRISE_RESOURCE_TYPE_CONSIGNEE": 6, "ENTERPRISE_RESOURCE_TYPE_NOTIFY_PARTY": 7, "ENTERPRISE_RESOURCE_TYPE_ENGLISH_CARGO_NAME": 8, "ENTERPRISE_RESOURCE_TYPE_HS_CODE": 9, "ENTERPRISE_RESOURCE_TYPE_MARKS": 10} {
		value := enum.Values().ByName(name)
		if value == nil || value.Number() != number {
			t.Errorf("保留资源类型编号变化: %s", name)
		}
	}
	for _, name := range []protoreflect.FullName{"order.v1.OrderTagService", "order.v1.BusinessTagSummary", "finance.v1.BusinessTagSummary", "enterprise_resource.v1.EnterpriseTagGroup", "enterprise_resource.v1.EnterpriseResourceTag"} {
		if _, err := protoregistry.GlobalFiles.FindDescriptorByName(name); err == nil {
			t.Errorf("标签独占契约仍存在: %s", name)
		}
	}
}
