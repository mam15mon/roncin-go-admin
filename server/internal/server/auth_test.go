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
	scopes    []biz.OrderOrganizationScope
	getCalls  int
}

func (s *authorizationOrderRepoStub) FindAuthorized(_ context.Context, _ uuid.UUID, scopes []biz.OrderOrganizationScope) (*biz.Order, error) {
	s.findCalls++
	s.scopes = scopes
	return s.order, nil
}

func (s *authorizationOrderRepoStub) Get(context.Context, uuid.UUID, uuid.UUID) (*biz.Order, error) {
	s.getCalls++
	return nil, biz.ErrOrderNotFound
}

func TestHasPermissionAllowsUnfilteredOrderListWithAnyReadableBusinessType(t *testing.T) {
	principal := principalWithOrderPermission(access.OrderBusinessSE, access.OrderRead)
	rule := accessRule{orderOperation: access.OrderRead, scope: biz.DataScopeOrganization}

	if !hasPermission(&orderv1.ListOrdersRequest{}, principal, rule) {
		t.Fatal("拥有海运出口订单读取权限时，未指定业务类型的订单列表应允许访问")
	}
}

func TestHasPermissionChecksFilteredOrderListBusinessType(t *testing.T) {
	principal := principalWithOrderPermission(access.OrderBusinessSE, access.OrderRead)
	rule := accessRule{orderOperation: access.OrderRead, scope: biz.DataScopeOrganization}
	se := orderv1.BusinessType_BUSINESS_TYPE_SE
	si := orderv1.BusinessType_BUSINESS_TYPE_SI

	if !hasPermission(&orderv1.ListOrdersRequest{BusinessType: &se}, principal, rule) {
		t.Fatal("拥有海运出口订单读取权限时，应允许筛选海运出口订单")
	}
	if hasPermission(&orderv1.ListOrdersRequest{BusinessType: &si}, principal, rule) {
		t.Fatal("仅拥有海运出口订单读取权限时，不应允许筛选海运进口订单")
	}
}

func TestHasPermissionRejectsUnfilteredOrderListWithoutReadPermission(t *testing.T) {
	principal := principalWithOrderPermission(access.OrderBusinessSE, access.OrderCreate)
	rule := accessRule{orderOperation: access.OrderRead, scope: biz.DataScopeOrganization}

	if hasPermission(&orderv1.ListOrdersRequest{}, principal, rule) {
		t.Fatal("没有任何订单读取权限时，未指定业务类型的订单列表应拒绝访问")
	}
}

func TestHasAnyOrderPermissionIncludesLandAndRail(t *testing.T) {
	for _, businessType := range []access.OrderBusinessType{access.OrderBusinessLand, access.OrderBusinessRail} {
		principal := principalWithOrderPermission(businessType, access.OrderRead)
		if !hasAnyOrderPermission(principal, access.OrderRead, false) {
			t.Fatalf("任一订单权限检查未覆盖业务类型 %s", businessType)
		}
	}
}

func TestOrderOperationWritesClassifiesEveryDeclaredOperation(t *testing.T) {
	tests := map[access.OrderOperation]bool{
		access.OrderRead:                 false,
		access.OrderCreate:               true,
		access.OrderUpdate:               true,
		access.OrderTransition:           true,
		access.OrderMilestoneRead:        false,
		access.OrderMilestoneSet:         true,
		access.OrderAttachmentRead:       false,
		access.OrderAttachmentRegister:   true,
		access.OrderPersonnelRead:        false,
		access.OrderPersonnelAssign:      true,
		access.OrderPersonnelRemove:      true,
		access.OrderContainerRead:        false,
		access.OrderContainerCreate:      true,
		access.OrderContainerUpdate:      true,
		access.OrderContainerDelete:      true,
		access.OrderCargoItemRead:        false,
		access.OrderCargoItemCreate:      true,
		access.OrderCargoItemUpdate:      true,
		access.OrderCargoItemDelete:      true,
		access.OrderAbnormalCaseRead:     false,
		access.OrderAbnormalCaseCreate:   true,
		access.OrderAbnormalCaseResolve:  true,
		access.OrderAbnormalCaseDelete:   true,
		access.OrderReleasePodRead:       false,
		access.OrderReleasePodCreate:     true,
		access.OrderReleasePodUpdate:     true,
		access.OrderReleasePodTransition: true,
		access.OrderReleasePodDelete:     true,
		access.OrderFeeRead:              false,
		access.OrderFeeCreate:            true,
		access.OrderFeeUpdate:            true,
		access.OrderFeeDelete:            true,
		access.OrderSplit:                true,
		access.OrderReassign:             true,
		access.OrderLock:                 true,
		access.OrderAmend:                true,
		access.OrderVoid:                 true,
		access.OrderSwitch:               true,
	}
	for operation, want := range tests {
		if got := orderOperationWrites(operation); got != want {
			t.Errorf("%s writable = %t, want %t", operation, got, want)
		}
	}
}

func TestCheckOrderReferenceRequiresCreateScopeInCurrentOrganization(t *testing.T) {
	currentOrganizationID := uuid.New()
	otherOrganizationID := uuid.New()
	principal := &biz.Principal{
		Organization:      biz.Organization{ID: currentOrganizationID},
		OrganizationNodes: serverOrganizationNodes(currentOrganizationID, otherOrganizationID),
		RoleGrants: []biz.RoleGrant{serverRoleGrant("reader", biz.DataScopeOrganization,
			[]string{access.OrderPermission(access.OrderBusinessSE, access.OrderRead)},
			[]biz.OrganizationAccess{{OrganizationID: otherOrganizationID, Writable: true}})},
	}
	rule := accessRule{orderOperation: access.OrderCreate, scope: biz.DataScopeOrganization}
	if hasPermission(&orderv1.CheckOrderReferenceRequest{}, principal, rule) {
		t.Fatal("仅在其他组织拥有读权限不得检查当前组织的订单编号")
	}

	principal.RoleGrants[0].Permissions[access.OrderPermission(access.OrderBusinessSE, access.OrderCreate)] = struct{}{}
	if !hasPermission(&orderv1.CheckOrderReferenceRequest{}, principal, rule) {
		t.Fatal("当前组织的订单创建权限应允许检查订单编号")
	}
}

func TestMatchSeaMasterBillCandidateUsesCurrentOrganizationSEReadScope(t *testing.T) {
	rule := accessRule{orderOperation: access.OrderRead, scope: biz.DataScopeOrganization}
	request := &orderv1.MatchSeaMasterBillCandidateRequest{}
	if !hasPermission(request, principalWithOrderPermission(access.OrderBusinessSE, access.OrderRead), rule) {
		t.Fatal("海运出口订单 read 权限应允许共享主单候选匹配")
	}
	if hasPermission(request, principalWithOrderPermission(access.OrderBusinessSI, access.OrderRead), rule) {
		t.Fatal("海运进口订单 read 权限不得授权海运出口共享主单候选匹配")
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
		if hasPermission(&orderv1.CreateOrderRequest{BusinessType: businessType}, principal, rule) {
			t.Fatalf("仅持有 SE 锁权限时不应通过 %s 锁权限检查", businessType)
		}
	}
}

func TestRequestOrderUsesPermissionScopedRepositoryQuery(t *testing.T) {
	organizationID := uuid.New()
	orderID := uuid.New()
	repo := &authorizationOrderRepoStub{order: &biz.Order{
		ID:             orderID,
		OrganizationID: organizationID,
		BusinessType:   biz.OrderBusinessSE,
	}}
	usecase := biz.NewOrderUsecase(repo, nil, nil, nil)

	principal := principalWithOrderPermission(access.OrderBusinessSE, access.OrderRead)
	principal.Organization = biz.Organization{ID: organizationID}
	principal.OrganizationNodes = serverOrganizationNodes(organizationID)
	order, direct := requestOrder(t.Context(), &orderv1.GetOrderRequest{Id: orderID.String()}, usecase, orderOrganizationScopes(principal, access.OrderRead, false))
	if !direct || order == nil || order.ID != orderID {
		t.Fatalf("授权查询应定位订单，actual order=%#v direct=%v", order, direct)
	}
	if repo.findCalls != 1 || repo.getCalls != 0 {
		t.Fatalf("鉴权应仅查询带权限范围的订单基础数据，实际 FindAuthorized=%d Get=%d", repo.findCalls, repo.getCalls)
	}
	if len(repo.scopes) != 1 || repo.scopes[0].BusinessType != biz.OrderBusinessSE || len(repo.scopes[0].OrganizationIDs) != 1 || repo.scopes[0].OrganizationIDs[0] != organizationID {
		t.Fatalf("鉴权查询范围 = %#v", repo.scopes)
	}
}

func principalWithOrderPermission(businessType access.OrderBusinessType, operation access.OrderOperation) *biz.Principal {
	permission := access.OrderPermission(businessType, operation)
	organizationID := uuid.New()
	return &biz.Principal{
		Organization:      biz.Organization{ID: organizationID},
		OrganizationNodes: serverOrganizationNodes(organizationID),
		RoleGrants:        []biz.RoleGrant{serverRoleGrant("operator", biz.DataScopeOrganization, []string{permission}, nil)},
	}
}

func serverRoleGrant(code string, scope biz.DataScope, permissions []string, accesses []biz.OrganizationAccess) biz.RoleGrant {
	permissionSet := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		permissionSet[permission] = struct{}{}
	}
	return biz.RoleGrant{RoleID: uuid.New(), RoleCode: code, DataScope: scope, Permissions: permissionSet, OrganizationAccesses: accesses}
}

func serverOrganizationNodes(ids ...uuid.UUID) []biz.OrganizationScopeNode {
	result := make([]biz.OrganizationScopeNode, 0, len(ids))
	for _, id := range ids {
		result = append(result, biz.OrganizationScopeNode{ID: id})
	}
	return result
}

type anchorAwareOrderRepoStub struct {
	biz.OrderRepo
	order *biz.Order
}

func (s *anchorAwareOrderRepoStub) FindAuthorized(_ context.Context, id uuid.UUID, scopes []biz.OrderOrganizationScope) (*biz.Order, error) {
	// 仅锚点订单 ID 能命中；共享箱 ID 等其他 ID 一律查不到
	if s.order != nil && id == s.order.ID {
		for _, scope := range scopes {
			if scope.BusinessType == s.order.BusinessType {
				for _, organizationID := range scope.OrganizationIDs {
					if organizationID == s.order.OrganizationID {
						return s.order, nil
					}
				}
			}
		}
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
		principal.OrganizationNodes = serverOrganizationNodes(organizationID)
		rule := accessRule{orderOperation: tc.operation, scope: biz.DataScopeOrganization}

		order, direct := requestOrder(t.Context(), tc.request, orderUsecase, orderOrganizationScopes(principal, tc.operation, orderOperationWrites(tc.operation)))
		if !direct || order == nil || order.ID != anchorOrder.ID {
			t.Fatalf("%s 应通过 order_id 锚点定位 SE 订单，实际 order=%v direct=%v", tc.name, order, direct)
		}
		if hasPermission(tc.request, principal, rule) {
			t.Fatalf("直接订单请求不得回退到未限定 ID 的通用鉴权路径: %s", tc.name)
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
		order, direct := requestOrder(t.Context(), tc.request, orderUsecase, orderOrganizationScopes(otherLine, tc.operation, orderOperationWrites(tc.operation)))
		if !direct || order != nil {
			t.Fatalf("SI %s 权限不应授权 SE 共享箱 %s", tc.operation, tc.name)
		}
		// 完全没有对应权限时拒绝
		none := principalWithOrderPermission(access.OrderBusinessSE, access.OrderMilestoneRead)
		order, direct = requestOrder(t.Context(), tc.request, orderUsecase, orderOrganizationScopes(none, tc.operation, orderOperationWrites(tc.operation)))
		if !direct || order != nil {
			t.Fatalf("无 SE %s 权限时 %s 应拒绝", tc.operation, tc.name)
		}
	}
}

func TestSharedContainerIdIsNotTreatedAsOrderID(t *testing.T) {
	anchorOrder := &biz.Order{ID: uuid.New(), OrganizationID: uuid.New(), BusinessType: biz.OrderBusinessSE}
	orderUsecase := biz.NewOrderUsecase(&anchorAwareOrderRepoStub{order: anchorOrder}, nil, nil, nil)

	// 缺少 order_id 时，GetId()（共享箱 ID）会被当成订单 ID 查询并失败，授权必须拒绝
	legacyRequest := &orderv1.GetSeaSharedContainerRequest{Id: uuid.New().String()}
	principal := principalWithOrderPermission(access.OrderBusinessSE, access.OrderContainerRead)
	order, direct := requestOrder(t.Context(), legacyRequest, orderUsecase, orderOrganizationScopes(principal, access.OrderContainerRead, false))
	if !direct || order != nil {
		t.Fatalf("缺少 order_id 的共享箱请求应无法定位订单并拒绝，实际 order=%v direct=%v", order, direct)
	}
	if _, ok := requestOrderBusinessType(legacyRequest); ok {
		t.Fatal("缺少 order_id 的共享箱请求不应解析出业务类型")
	}

	// 携带 order_id 时，中间件优先 GetOrderId()，共享箱 id 不参与订单定位
	anchoredRequest := &orderv1.DeleteSeaSharedContainerRequest{OrderId: anchorOrder.ID.String(), Id: uuid.New().String()}
	if _, ok := requestOrderBusinessType(anchoredRequest); ok {
		t.Fatal("携带 order_id 的请求不得通过未受限的业务类型预查询")
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
		Organization:      biz.Organization{ID: principalOrg},
		OrganizationNodes: serverOrganizationNodes(principalOrg, anchorOrg),
		RoleGrants: []biz.RoleGrant{serverRoleGrant("operator", biz.DataScopeOrganization,
			[]string{access.OrderPermission(access.OrderBusinessSE, access.OrderContainerRead)},
			[]biz.OrganizationAccess{{OrganizationID: anchorOrg, Writable: true}})},
	}

	request := &orderv1.ListSeaSharedContainersRequest{OrderId: anchorOrder.ID.String(), TransportExecutionId: uuid.New().String()}
	order, direct := requestOrder(t.Context(), request, orderUsecase, orderOrganizationScopes(principal, access.OrderContainerRead, false))
	if !direct || order == nil {
		t.Fatal("锚点订单应可定位")
	}
	if order.OrganizationID != anchorOrg {
		t.Fatal("授权查询应返回跨组织锚点订单")
	}

	// 无锚点组织访问权限时拒绝
	noAccess := &biz.Principal{Organization: biz.Organization{ID: principalOrg}, OrganizationNodes: serverOrganizationNodes(principalOrg, anchorOrg)}
	denied, direct := requestOrder(t.Context(), request, orderUsecase, orderOrganizationScopes(noAccess, access.OrderContainerRead, false))
	if !direct || denied != nil {
		t.Fatal("无锚点组织访问权限时应拒绝")
	}
}

func TestRequestOrderRejectsReadOnlyCrossOrganizationUpdate(t *testing.T) {
	tianjinID := uuid.New()
	beijingID := uuid.New()
	order := &biz.Order{ID: uuid.New(), OrganizationID: beijingID, BusinessType: biz.OrderBusinessSE}
	usecase := biz.NewOrderUsecase(&anchorAwareOrderRepoStub{order: order}, nil, nil, nil)
	principal := &biz.Principal{
		Organization:      biz.Organization{ID: tianjinID},
		OrganizationNodes: serverOrganizationNodes(tianjinID, beijingID),
		RoleGrants: []biz.RoleGrant{serverRoleGrant("order-editor", biz.DataScopeOrganization,
			[]string{access.OrderPermission(access.OrderBusinessSE, access.OrderUpdate)},
			[]biz.OrganizationAccess{{OrganizationID: beijingID}})},
	}

	resolved, direct := requestOrder(t.Context(), &orderv1.UpdateOrderRequest{Id: order.ID.String()}, usecase, orderOrganizationScopes(principal, access.OrderUpdate, true))
	if !direct || resolved != nil {
		t.Fatal("北京只读组织范围不得通过订单更新的授权查询")
	}
}

func TestRequestOrderDoesNotBorrowFinanceRoleOrganizationAccess(t *testing.T) {
	tianjinID := uuid.New()
	beijingID := uuid.New()
	order := &biz.Order{ID: uuid.New(), OrganizationID: beijingID, BusinessType: biz.OrderBusinessSE}
	usecase := biz.NewOrderUsecase(&anchorAwareOrderRepoStub{order: order}, nil, nil, nil)
	principal := &biz.Principal{
		Organization:      biz.Organization{ID: tianjinID},
		OrganizationNodes: serverOrganizationNodes(tianjinID, beijingID),
		RoleGrants: []biz.RoleGrant{
			serverRoleGrant("order-reader", biz.DataScopeOrganization, []string{access.OrderPermission(access.OrderBusinessSE, access.OrderRead)}, nil),
			serverRoleGrant("finance-reader", biz.DataScopeOrganization, []string{"finance.bill.read"}, []biz.OrganizationAccess{{OrganizationID: beijingID}}),
		},
	}

	resolved, direct := requestOrder(t.Context(), &orderv1.GetOrderRequest{Id: order.ID.String()}, usecase, orderOrganizationScopes(principal, access.OrderRead, false))
	if !direct || resolved != nil {
		t.Fatal("财务角色的北京访问项不得被订单 read 权限借用")
	}
}
