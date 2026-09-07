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

type anchorAwareOrderRepoStub struct {
	biz.OrderRepo
	order *biz.Order
}

func (s *anchorAwareOrderRepoStub) Find(_ context.Context, id uuid.UUID) (*biz.Order, error) {
	// 仅锚点订单 ID 能命中；共享箱 ID 等其他 ID 一律查不到
	if s.order != nil && id == s.order.ID {
		return s.order, nil
	}
	return nil, biz.ErrOrderNotFound
}

func (s *anchorAwareOrderRepoStub) Get(context.Context, uuid.UUID, uuid.UUID) (*biz.Order, error) {
	return nil, biz.ErrOrderNotFound
}

// sharedContainerAuthRequests 返回全部共享箱请求及其所需权限操作
func sharedContainerAuthRequests(anchorOrderID uuid.UUID) []struct {
	name      string
	request   any
	operation access.OrderOperation
} {
	containerID := uuid.New()
	return []struct {
		name      string
		request   any
		operation access.OrderOperation
	}{
		{"ListSeaSharedContainers", &orderv1.ListSeaSharedContainersRequest{OrderId: anchorOrderID.String(), TransportExecutionId: uuid.New().String()}, access.OrderContainerRead},
		{"GetSeaSharedContainer", &orderv1.GetSeaSharedContainerRequest{OrderId: anchorOrderID.String(), Id: containerID.String()}, access.OrderContainerRead},
		{"ListSeaSharedContainerCandidates", &orderv1.ListSeaSharedContainerCandidatesRequest{OrderId: anchorOrderID.String(), TransportExecutionId: uuid.New().String()}, access.OrderContainerRead},
		{"CreateSeaSharedContainer", &orderv1.CreateSeaSharedContainerRequest{OrderId: anchorOrderID.String()}, access.OrderContainerCreate},
		{"UpdateSeaSharedContainer", &orderv1.UpdateSeaSharedContainerRequest{OrderId: anchorOrderID.String(), Id: containerID.String()}, access.OrderContainerUpdate},
		{"DeleteSeaSharedContainer", &orderv1.DeleteSeaSharedContainerRequest{OrderId: anchorOrderID.String(), Id: containerID.String()}, access.OrderContainerDelete},
		{"SaveSeaSharedContainerAllocationsDraft", &orderv1.SaveSeaSharedContainerAllocationsDraftRequest{OrderId: anchorOrderID.String(), Id: containerID.String()}, access.OrderContainerUpdate},
		{"ConfirmSeaSharedContainer", &orderv1.ConfirmSeaSharedContainerRequest{OrderId: anchorOrderID.String(), Id: containerID.String()}, access.OrderContainerUpdate},
		{"WithdrawSeaSharedContainer", &orderv1.WithdrawSeaSharedContainerRequest{OrderId: anchorOrderID.String(), Id: containerID.String()}, access.OrderContainerUpdate},
	}
}

func TestSharedContainerRequestsAuthorizeThroughAnchorOrder(t *testing.T) {
	organizationID := uuid.New()
	anchorOrder := &biz.Order{ID: uuid.New(), OrganizationID: organizationID, BusinessType: biz.OrderBusinessSE}
	repo := &anchorAwareOrderRepoStub{order: anchorOrder}
	orderUsecase := biz.NewOrderUsecase(repo, nil, nil, nil)

	for _, tc := range sharedContainerAuthRequests(anchorOrder.ID) {
		principal := principalWithOrderPermission(access.OrderBusinessSE, tc.operation)
		principal.Organization = biz.Organization{ID: organizationID}
		rule := accessRule{orderOperation: tc.operation, scope: biz.DataScopeOrganization}

		order, direct := requestOrder(t.Context(), tc.request, orderUsecase)
		if !direct || order == nil || order.ID != anchorOrder.ID {
			t.Fatalf("%s 应通过 order_id 锚点定位 SE 订单，实际 order=%v direct=%v", tc.name, order, direct)
		}
		if !principal.CanAccessOrderOrganization(order.OrganizationID, orderOperationWrites(tc.operation)) {
			t.Fatalf("%s 锚点订单组织应可访问", tc.name)
		}
		if !hasPermission(t.Context(), tc.request, principal, rule, orderUsecase) {
			t.Fatalf("持有 SE %s 权限时 %s 应放行", tc.operation, tc.name)
		}
	}
}

func TestSharedContainerRequestsDeniedWithoutMatchingPermission(t *testing.T) {
	organizationID := uuid.New()
	anchorOrder := &biz.Order{ID: uuid.New(), OrganizationID: organizationID, BusinessType: biz.OrderBusinessSE}
	orderUsecase := biz.NewOrderUsecase(&anchorAwareOrderRepoStub{order: anchorOrder}, nil, nil, nil)

	for _, tc := range sharedContainerAuthRequests(anchorOrder.ID) {
		// 持有其他业务线的同名细粒度权限不得授权 SE 共享箱
		otherLine := principalWithOrderPermission(access.OrderBusinessSI, tc.operation)
		rule := accessRule{orderOperation: tc.operation, scope: biz.DataScopeOrganization}
		if hasPermission(t.Context(), tc.request, otherLine, rule, orderUsecase) {
			t.Fatalf("SI %s 权限不应授权 SE 共享箱 %s", tc.operation, tc.name)
		}
		// 完全没有对应权限时拒绝
		none := principalWithOrderPermission(access.OrderBusinessSE, access.OrderMilestoneRead)
		if hasPermission(t.Context(), tc.request, none, rule, orderUsecase) {
			t.Fatalf("无 SE %s 权限时 %s 应拒绝", tc.operation, tc.name)
		}
	}
}

func TestSharedContainerIdIsNotTreatedAsOrderID(t *testing.T) {
	anchorOrder := &biz.Order{ID: uuid.New(), OrganizationID: uuid.New(), BusinessType: biz.OrderBusinessSE}
	orderUsecase := biz.NewOrderUsecase(&anchorAwareOrderRepoStub{order: anchorOrder}, nil, nil, nil)

	// 缺少 order_id 时，GetId()（共享箱 ID）会被当成订单 ID 查询并失败，授权必须拒绝
	legacyRequest := &orderv1.GetSeaSharedContainerRequest{Id: uuid.New().String()}
	order, direct := requestOrder(t.Context(), legacyRequest, orderUsecase)
	if !direct || order != nil {
		t.Fatalf("缺少 order_id 的共享箱请求应无法定位订单并拒绝，实际 order=%v direct=%v", order, direct)
	}
	if _, ok := requestOrderBusinessType(t.Context(), legacyRequest, anchorOrder.OrganizationID, orderUsecase); ok {
		t.Fatal("缺少 order_id 的共享箱请求不应解析出业务类型")
	}

	// 携带 order_id 时，中间件优先 GetOrderId()，共享箱 id 不参与订单定位
	anchoredRequest := &orderv1.DeleteSeaSharedContainerRequest{OrderId: anchorOrder.ID.String(), Id: uuid.New().String()}
	businessType, ok := requestOrderBusinessType(t.Context(), anchoredRequest, anchorOrder.OrganizationID, orderUsecase)
	if !ok || businessType != access.OrderBusinessSE {
		t.Fatalf("携带 order_id 的共享箱请求应按锚点解析 SE 业务类型，实际 %q %v", businessType, ok)
	}
}

func TestSharedContainerAnchorOrderResolvesOrganizationContext(t *testing.T) {
	// 多组织场景：锚点订单所在组织成为有效组织上下文
	anchorOrg := uuid.New()
	principalOrg := uuid.New()
	anchorOrder := &biz.Order{ID: uuid.New(), OrganizationID: anchorOrg, BusinessType: biz.OrderBusinessSE}
	repo := &anchorAwareOrderRepoStub{order: anchorOrder}
	orderUsecase := biz.NewOrderUsecase(repo, nil, nil, nil)

	principal := &biz.Principal{
		Organization: biz.Organization{ID: principalOrg},
		Permissions:  []string{access.OrderPermission(access.OrderBusinessSE, access.OrderContainerRead)},
		RoleScopes:   []biz.RoleScope{{RoleCode: "operator", DataScope: biz.DataScopeOrganization}},
		RolePermissions: map[string]map[string]struct{}{
			"operator": {access.OrderPermission(access.OrderBusinessSE, access.OrderContainerRead): {}},
		},
		OrderOrganizationAccesses: []biz.OrderOrganizationAccess{{OrganizationID: anchorOrg, Writable: true}},
	}

	request := &orderv1.ListSeaSharedContainersRequest{OrderId: anchorOrder.ID.String(), TransportExecutionId: uuid.New().String()}
	order, direct := requestOrder(t.Context(), request, orderUsecase)
	if !direct || order == nil {
		t.Fatal("锚点订单应可定位")
	}
	if !principal.CanAccessOrderOrganization(order.OrganizationID, orderOperationWrites(access.OrderContainerRead)) {
		t.Fatal("多组织用户应可访问锚点订单组织")
	}
	// 锚点组织上下文下权限检查放行
	rule := accessRule{orderOperation: access.OrderContainerRead, scope: biz.DataScopeOrganization}
	effective := *principal
	effective.Organization = biz.Organization{ID: anchorOrg}
	if !hasPermission(t.Context(), request, &effective, rule, orderUsecase) {
		t.Fatal("以锚点订单组织为上下文时应放行 container.read")
	}

	// 无锚点组织访问权限时拒绝
	noAccess := &biz.Principal{Organization: biz.Organization{ID: principalOrg}}
	if noAccess.CanAccessOrderOrganization(anchorOrg, false) {
		t.Fatal("无锚点组织访问权限时应拒绝")
	}
}
