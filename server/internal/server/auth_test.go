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

func TestRequestOrderBusinessTypeOnlyLoadsOrderBaseData(t *testing.T) {
	organizationID := uuid.New()
	orderID := uuid.New()
	repo := &authorizationOrderRepoStub{order: &biz.Order{
		ID:             orderID,
		OrganizationID: organizationID,
		BusinessType:   biz.OrderBusinessSE,
	}}
	usecase := biz.NewOrderUsecase(repo, nil)

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
