package server

import (
	"testing"

	orderv1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestHasPermissionAllowsUnfilteredOrderListWithAnyReadableBusinessType(t *testing.T) {
	principal := principalWithOrderPermission(access.OrderBusinessSE, access.OrderRead)
	rule := accessRule{orderOperation: access.OrderRead, scope: biz.DataScopeOrganization}

	if !hasPermission(t.Context(), &orderv1.ListOrdersRequest{}, principal, rule, nil) {
		t.Fatal("拥有海运出口订单读取权限时，未指定业务类型的订单列表应允许访问")
	}
}

func TestHasPermissionChecksFilteredOrderListBusinessType(t *testing.T) {
	principal := principalWithOrderPermission(access.OrderBusinessSE, access.OrderRead)
	rule := accessRule{orderOperation: access.OrderRead, scope: biz.DataScopeOrganization}
	se := orderv1.BusinessType_BUSINESS_TYPE_SE
	si := orderv1.BusinessType_BUSINESS_TYPE_SI

	if !hasPermission(t.Context(), &orderv1.ListOrdersRequest{BusinessType: &se}, principal, rule, nil) {
		t.Fatal("拥有海运出口订单读取权限时，应允许筛选海运出口订单")
	}
	if hasPermission(t.Context(), &orderv1.ListOrdersRequest{BusinessType: &si}, principal, rule, nil) {
		t.Fatal("仅拥有海运出口订单读取权限时，不应允许筛选海运进口订单")
	}
}

func TestHasPermissionRejectsUnfilteredOrderListWithoutReadPermission(t *testing.T) {
	principal := principalWithOrderPermission(access.OrderBusinessSE, access.OrderCreate)
	rule := accessRule{orderOperation: access.OrderRead, scope: biz.DataScopeOrganization}

	if hasPermission(t.Context(), &orderv1.ListOrdersRequest{}, principal, rule, nil) {
		t.Fatal("没有任何订单读取权限时，未指定业务类型的订单列表应拒绝访问")
	}
}

func principalWithOrderPermission(businessType access.OrderBusinessType, operation access.OrderOperation) *biz.Principal {
	permission := access.OrderPermission(businessType, operation)
	return &biz.Principal{
		Permissions:     []string{permission},
		RoleScopes:      []biz.RoleScope{{RoleCode: "operator", DataScope: biz.DataScopeOrganization}},
		RolePermissions: map[string]map[string]struct{}{"operator": {permission: {}}},
	}
}
