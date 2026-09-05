package server

import (
	"context"
	"testing"

	"github.com/google/uuid"

	orderv1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type authorizationOrderRepoStub struct {
	biz.OrderRepo
	order     *biz.Order
	findCalls int
	getCalls  int
}

func (s *authorizationOrderRepoStub) Find(context.Context, uuid.UUID) (*biz.Order, error) {
	s.findCalls++
	return s.order, nil
}

func (s *authorizationOrderRepoStub) Get(context.Context, uuid.UUID, uuid.UUID) (*biz.Order, error) {
	s.getCalls++
	return nil, biz.ErrOrderNotFound
}

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

func TestHasAnyOrderPermissionIncludesLandAndRail(t *testing.T) {
	for _, businessType := range []access.OrderBusinessType{access.OrderBusinessLand, access.OrderBusinessRail} {
		principal := principalWithOrderPermission(businessType, access.OrderRead)
		if !hasAnyOrderPermission(principal, access.OrderRead, biz.DataScopeOrganization) {
			t.Fatalf("任一订单权限检查未覆盖业务类型 %s", businessType)
		}
	}
}

func TestOrderBusinessTypeMappingsCoverAllTypes(t *testing.T) {
	cases := []struct {
		api    orderv1.BusinessType
		biz    biz.OrderBusinessType
		access access.OrderBusinessType
	}{
		{api: orderv1.BusinessType_BUSINESS_TYPE_SE, biz: biz.OrderBusinessSE, access: access.OrderBusinessSE},
		{api: orderv1.BusinessType_BUSINESS_TYPE_SI, biz: biz.OrderBusinessSI, access: access.OrderBusinessSI},
		{api: orderv1.BusinessType_BUSINESS_TYPE_AE, biz: biz.OrderBusinessAE, access: access.OrderBusinessAE},
		{api: orderv1.BusinessType_BUSINESS_TYPE_AI, biz: biz.OrderBusinessAI, access: access.OrderBusinessAI},
		{api: orderv1.BusinessType_BUSINESS_TYPE_LAND, biz: biz.OrderBusinessLand, access: access.OrderBusinessLand},
		{api: orderv1.BusinessType_BUSINESS_TYPE_RAIL, biz: biz.OrderBusinessRail, access: access.OrderBusinessRail},
	}
	for _, tc := range cases {
		gotAPI, ok := orderBusinessTypeFromAPI(tc.api)
		if !ok || gotAPI != tc.access {
			t.Errorf("API 业务类型 %s 映射 = %q, %v，期望 %q, true", tc.api, gotAPI, ok, tc.access)
		}
		gotBiz, ok := orderBusinessTypeFromBiz(tc.biz)
		if !ok || gotBiz != tc.access {
			t.Errorf("Biz 业务类型 %s 映射 = %q, %v，期望 %q, true", tc.biz, gotBiz, ok, tc.access)
		}
	}
	if _, ok := orderBusinessTypeFromAPI(orderv1.BusinessType_BUSINESS_TYPE_UNSPECIFIED); ok {
		t.Fatal("未指定 API 业务类型不得通过鉴权映射")
	}
	if _, ok := orderBusinessTypeFromBiz(""); ok {
		t.Fatal("空 Biz 业务类型不得通过鉴权映射")
	}
}

func TestHasPermissionKeepsOrderBusinessTypesIsolated(t *testing.T) {
	principal := principalWithOrderPermission(access.OrderBusinessSE, access.OrderLock)
	rule := accessRule{orderOperation: access.OrderLock, scope: biz.DataScopeOrganization}
	for _, businessType := range []orderv1.BusinessType{
		orderv1.BusinessType_BUSINESS_TYPE_SI,
		orderv1.BusinessType_BUSINESS_TYPE_AE,
		orderv1.BusinessType_BUSINESS_TYPE_AI,
		orderv1.BusinessType_BUSINESS_TYPE_LAND,
		orderv1.BusinessType_BUSINESS_TYPE_RAIL,
	} {
		if hasPermission(t.Context(), &orderv1.CreateOrderRequest{BusinessType: businessType}, principal, rule, nil) {
			t.Fatalf("仅持有 SE 锁权限时不应通过 %s 锁权限检查", businessType)
		}
	}
}

func TestRequestOrderBusinessTypeOnlyLoadsOrderBaseData(t *testing.T) {
	organizationID := uuid.New()
	orderID := uuid.New()
	repo := &authorizationOrderRepoStub{order: &biz.Order{
		ID:             orderID,
		OrganizationID: organizationID,
		BusinessType:   biz.OrderBusinessSE,
	}}
	usecase := biz.NewOrderUsecase(repo, nil, nil, nil)

	businessType, ok := requestOrderBusinessType(
		t.Context(),
		&orderv1.GetOrderRequest{Id: orderID.String()},
		organizationID,
		usecase,
	)

	if !ok || businessType != access.OrderBusinessSE {
		t.Fatalf("应识别海运出口订单，实际 businessType=%q ok=%v", businessType, ok)
	}
	if repo.findCalls != 1 || repo.getCalls != 0 {
		t.Fatalf("鉴权应仅查询订单基础数据，实际 Find=%d Get=%d", repo.findCalls, repo.getCalls)
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
