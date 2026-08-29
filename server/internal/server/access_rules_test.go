package server

import (
	"testing"

	accessv1 "github.com/roncin/roncin-go-admin/server/api/access/v1"
	_ "github.com/roncin/roncin-go-admin/server/api/enterprise_resource/v1"
	_ "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestGeneratedAccessRulesMatchProto(t *testing.T) {
	packages := map[protoreflect.FullName]struct{}{
		"admin.v1": {}, "auth.v1": {}, "enterprise_resource.v1": {}, "finance.v1": {}, "masterdata.v1": {}, "order.v1": {}, "partner.v1": {}, "task.v1": {},
	}
	seen := make(map[string]struct{})

	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if _, ok := packages[file.Package()]; !ok {
			return true
		}
		services := file.Services()
		for serviceIndex := 0; serviceIndex < services.Len(); serviceIndex++ {
			service := services.Get(serviceIndex)
			methods := service.Methods()
			for methodIndex := 0; methodIndex < methods.Len(); methodIndex++ {
				method := methods.Get(methodIndex)
				operation := "/" + string(service.FullName()) + "/" + string(method.Name())
				seen[operation] = struct{}{}

				actual, ok := operationAccessRules[operation]
				if !ok {
					t.Errorf("RPC %s 缺少生成的访问规则", operation)
					continue
				}
				expected, ok := accessRuleFromMethod(method)
				if !ok {
					t.Errorf("RPC %s 缺少有效的 Proto 访问规则", operation)
					continue
				}
				if actual != expected {
					t.Errorf("RPC %s 的生成规则为 %+v，Proto 声明为 %+v", operation, actual, expected)
				}
			}
		}
		return true
	})

	for operation := range operationAccessRules {
		if _, ok := seen[operation]; !ok {
			t.Errorf("生成文件包含已不存在的 RPC %s", operation)
		}
	}
}

func TestShippingDocumentAccessInheritsOrderPermissions(t *testing.T) {
	tests := map[string]access.OrderOperation{
		"/order.v1.OrderShippingDocumentService/ListShippingDocuments":            access.OrderRead,
		"/order.v1.OrderShippingDocumentService/AddShippingDocument":              access.OrderUpdate,
		"/order.v1.OrderShippingDocumentService/UpdateShippingDocument":           access.OrderUpdate,
		"/order.v1.OrderShippingDocumentService/TransitionShippingDocumentStatus": access.OrderUpdate,
		"/order.v1.OrderShippingDocumentService/RemoveShippingDocument":           access.OrderUpdate,
	}
	for operation, expected := range tests {
		rule, ok := operationAccessRules[operation]
		if !ok {
			t.Errorf("缺少单证接口访问规则 %s", operation)
			continue
		}
		if rule.mode != accessModeOrderPermission || rule.orderOperation != expected {
			t.Errorf("单证接口 %s 的权限规则为 %+v，期望继承订单权限 %s", operation, rule, expected)
		}
	}
}

func accessRuleFromMethod(method protoreflect.MethodDescriptor) (accessRule, bool) {
	options, ok := method.Options().(*descriptorpb.MethodOptions)
	if !ok || !proto.HasExtension(options, accessv1.E_Rule) {
		return accessRule{}, false
	}
	rule, ok := proto.GetExtension(options, accessv1.E_Rule).(*accessv1.Rule)
	if !ok || rule == nil {
		return accessRule{}, false
	}

	result := accessRule{permission: rule.GetPermission(), orderOperation: access.OrderOperation(rule.GetOrderOperation())}
	switch rule.GetMode() {
	case accessv1.AccessMode_ACCESS_MODE_PUBLIC:
		result.mode = accessModePublic
	case accessv1.AccessMode_ACCESS_MODE_AUTHENTICATED:
		result.mode = accessModeAuthenticated
	case accessv1.AccessMode_ACCESS_MODE_PERMISSION:
		result.mode = accessModePermission
	case accessv1.AccessMode_ACCESS_MODE_ORDER_PERMISSION:
		result.mode = accessModeOrderPermission
	default:
		return accessRule{}, false
	}
	switch rule.GetScope() {
	case accessv1.DataScope_DATA_SCOPE_UNSPECIFIED:
	case accessv1.DataScope_DATA_SCOPE_ALL:
		result.scope = biz.DataScopeAll
	case accessv1.DataScope_DATA_SCOPE_ORGANIZATION:
		result.scope = biz.DataScopeOrganization
	case accessv1.DataScope_DATA_SCOPE_ORGANIZATION_TREE:
		result.scope = biz.DataScopeOrganizationTree
	case accessv1.DataScope_DATA_SCOPE_SELF:
		result.scope = biz.DataScopeSelf
	default:
		return accessRule{}, false
	}
	return result, true
}
