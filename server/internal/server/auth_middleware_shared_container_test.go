package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	orderv1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"

	"github.com/go-kratos/kratos/v3/transport"
)

// middlewareHeader 实现 transport.Header
type middlewareHeader struct {
	cookie string
}

func (h middlewareHeader) Get(key string) string {
	if key == "Cookie" {
		return h.cookie
	}
	return ""
}
func (h middlewareHeader) Set(string, string) {}
func (h middlewareHeader) Add(string, string) {}
func (h middlewareHeader) Keys() []string     { return []string{"Cookie"} }
func (h middlewareHeader) Values(key string) []string {
	if key == "Cookie" {
		return []string{h.cookie}
	}
	return nil
}

// middlewareTransport 模拟 kratos 服务端传输上下文
type middlewareTransport struct {
	operation string
	cookie    string
}

func (t *middlewareTransport) Kind() transport.Kind { return transport.KindHTTP }
func (t *middlewareTransport) Endpoint() string     { return "" }
func (t *middlewareTransport) Operation() string    { return t.operation }
func (t *middlewareTransport) RequestHeader() transport.Header {
	return middlewareHeader{cookie: t.cookie}
}
func (t *middlewareTransport) ReplyHeader() transport.Header { return nil }

// middlewareAuthRepoStub 提供固定的会话与主体解析结果
type middlewareAuthRepoStub struct {
	biz.AuthRepo
	session   *biz.Session
	principal *biz.Principal
}

func (s *middlewareAuthRepoStub) FindSession(_ context.Context, tokenHash string, _ time.Time) (*biz.Session, error) {
	if s.session != nil && tokenHash != "" {
		return s.session, nil
	}
	return nil, biz.ErrSessionRequired
}

func (s *middlewareAuthRepoStub) ResolvePrincipal(context.Context, uuid.UUID, uuid.UUID) (*biz.Principal, error) {
	return s.principal, nil
}

func newAuthorizationTestMiddleware(t *testing.T, principal *biz.Principal, anchorOrder *biz.Order) (middlewareFunc func(ctx context.Context, request any) (any, error), handlerState *middlewareHandlerState) {
	t.Helper()
	policy := &biz.SessionPolicy{CookieName: "sid", TTL: time.Hour, SameSite: "lax", Secure: false}
	authUC := biz.NewAuthUsecase(&middlewareAuthRepoStub{
		session:   &biz.Session{TokenHash: "any", UserID: uuid.New(), OrganizationID: principal.Organization.ID, ExpiresAt: time.Now().Add(time.Hour)},
		principal: principal,
	}, policy, nil, nil, nil)
	orderUC := biz.NewOrderUsecase(&anchorAwareOrderRepoStub{order: anchorOrder}, nil, nil, nil)
	state := &middlewareHandlerState{}
	mw := Authorization(authUC, policy, orderUC)
	return func(ctx context.Context, request any) (any, error) {
		return mw(state.handler)(ctx, request)
	}, state
}

type middlewareHandlerState struct {
	called bool
	orgID  uuid.UUID
}

func (s *middlewareHandlerState) handler(ctx context.Context, _ any) (any, error) {
	s.called = true
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	s.orgID = principal.Organization.ID
	return nil, nil
}

func runSharedContainerMiddleware(
	t *testing.T,
	operation string,
	cookie string,
	principal *biz.Principal,
	anchorOrder *biz.Order,
	request any,
) *middlewareHandlerState {
	t.Helper()
	invoke, state := newAuthorizationTestMiddleware(t, principal, anchorOrder)
	ctx := transport.NewServerContext(t.Context(), &middlewareTransport{operation: operation, cookie: cookie})
	if _, err := invoke(ctx, request); err != nil {
		state.called = false
	}
	return state
}

func TestAuthorizationMiddlewareSharedContainerRequests(t *testing.T) {
	anchorOrg := uuid.New()
	anchorOrder := &biz.Order{ID: uuid.New(), OrganizationID: anchorOrg, BusinessType: biz.OrderBusinessSE}

	principalWith := func(op access.OrderOperation, writable bool) *biz.Principal {
		permission := access.OrderPermission(access.OrderBusinessSE, op)
		return &biz.Principal{
			Organization:    biz.Organization{ID: anchorOrg},
			Permissions:     []string{permission},
			RoleScopes:      []biz.RoleScope{{RoleCode: "operator", DataScope: biz.DataScopeOrganization}},
			RolePermissions: map[string]map[string]struct{}{"operator": {permission: {}}},
			OrganizationAccesses: []biz.OrganizationAccess{{
				OrganizationID: anchorOrg,
				Writable:       writable,
			}},
		}
	}

	t.Run("持有对应 SE 权限时全部共享箱请求进入 handler 且组织上下文为锚点订单组织", func(t *testing.T) {
		for _, tc := range sharedContainerAuthRequests(anchorOrder.ID) {
			principal := principalWith(tc.operation, true)
			operation := "/order.v1.SeaSharedContainerService/" + operationNameFromRequest(tc.request)
			state := runSharedContainerMiddleware(t, operation, "sid=valid-token", principal, anchorOrder, tc.request)
			if !state.called {
				t.Fatalf("%s 持有 SE %s 权限时应进入 handler", operation, tc.operation)
			}
			if state.orgID != anchorOrg {
				t.Fatalf("%s handler 内组织上下文应为锚点组织 %s, 实际 %s", operation, anchorOrg, state.orgID)
			}
		}
	})

	t.Run("无权限或仅其他业务线权限时请求被拒绝且不进入 handler", func(t *testing.T) {
		for _, tc := range sharedContainerAuthRequests(anchorOrder.ID) {
			operation := "/order.v1.SeaSharedContainerService/" + operationNameFromRequest(tc.request)

			noPermission := principalWith(access.OrderMilestoneRead, true)
			if state := runSharedContainerMiddleware(t, operation, "sid=valid-token", noPermission, anchorOrder, tc.request); state.called {
				t.Fatalf("%s 无权限时不应进入 handler", operation)
			}

			otherLine := principalWith(tc.operation, true)
			otherLine.Permissions = []string{access.OrderPermission(access.OrderBusinessSI, tc.operation)}
			otherLine.RolePermissions = map[string]map[string]struct{}{"operator": {access.OrderPermission(access.OrderBusinessSI, tc.operation): {}}}
			if state := runSharedContainerMiddleware(t, operation, "sid=valid-token", otherLine, anchorOrder, tc.request); state.called {
				t.Fatalf("%s 仅 SI 权限不应进入 handler", operation)
			}
		}
	})

	t.Run("缺少 order_id 锚点时拒绝且不进入 handler", func(t *testing.T) {
		containerID := uuid.New()
		requests := []struct {
			operation string
			request   any
		}{
			{"ListSeaSharedContainers", &orderv1.ListSeaSharedContainersRequest{TransportExecutionId: uuid.New().String()}},
			{"ListSeaSharedContainerCandidates", &orderv1.ListSeaSharedContainerCandidatesRequest{TransportExecutionId: uuid.New().String()}},
			{"CreateSeaSharedContainer", &orderv1.CreateSeaSharedContainerRequest{}},
			{"GetSeaSharedContainer", &orderv1.GetSeaSharedContainerRequest{Id: containerID.String()}},
			{"UpdateSeaSharedContainer", &orderv1.UpdateSeaSharedContainerRequest{Id: containerID.String(), ExpectedVersion: 1}},
			{"DeleteSeaSharedContainer", &orderv1.DeleteSeaSharedContainerRequest{Id: containerID.String(), ExpectedVersion: 1}},
			{"SaveSeaSharedContainerAllocationsDraft", &orderv1.SaveSeaSharedContainerAllocationsDraftRequest{Id: containerID.String(), ExpectedVersion: 1}},
			{"ConfirmSeaSharedContainer", &orderv1.ConfirmSeaSharedContainerRequest{Id: containerID.String(), ExpectedVersion: 1}},
			{"WithdrawSeaSharedContainer", &orderv1.WithdrawSeaSharedContainerRequest{Id: containerID.String(), ExpectedVersion: 1}},
		}
		for _, tc := range requests {
			principal := principalWith(access.OrderContainerRead, true)
			state := runSharedContainerMiddleware(t, "/order.v1.SeaSharedContainerService/"+tc.operation, "sid=valid-token", principal, anchorOrder, tc.request)
			if state.called {
				t.Fatalf("%s 缺少 order_id 时不应进入 handler", tc.operation)
			}
		}
	})

	t.Run("跨组织主体经真实中间件切换到锚点订单组织", func(t *testing.T) {
		orgA := uuid.New()
		orgB := uuid.New()
		anchorOrderB := &biz.Order{ID: uuid.New(), OrganizationID: orgB, BusinessType: biz.OrderBusinessSE}
		permission := access.OrderPermission(access.OrderBusinessSE, access.OrderContainerRead)
		principal := &biz.Principal{
			// 当前主体组织是 A，通过组织访问授权操作锚点订单所在组织 B
			Organization: biz.Organization{ID: orgA},
			Permissions:  []string{permission},
			RoleScopes: []biz.RoleScope{
				{RoleCode: "operator", DataScope: biz.DataScopeOrganization},
			},
			RolePermissions: map[string]map[string]struct{}{
				"operator": {permission: {}},
			},
			OrganizationAccesses: []biz.OrganizationAccess{
				{OrganizationID: orgB, Writable: true},
			},
		}

		request := &orderv1.ListSeaSharedContainersRequest{
			OrderId:              anchorOrderB.ID.String(),
			TransportExecutionId: uuid.New().String(),
		}
		state := runSharedContainerMiddleware(
			t,
			"/order.v1.SeaSharedContainerService/ListSeaSharedContainers",
			"sid=valid-token",
			principal,
			anchorOrderB,
			request,
		)
		if !state.called {
			t.Fatal("跨组织主体持有锚点组织访问与对应权限时应进入 handler")
		}
		if state.orgID != orgB {
			t.Fatalf("handler 内有效组织应为锚点订单组织 %s, 实际 %s", orgB, state.orgID)
		}

		// 无锚点组织访问权限时拒绝且不进入 handler
		principal.OrganizationAccesses = nil
		denied := runSharedContainerMiddleware(
			t,
			"/order.v1.SeaSharedContainerService/ListSeaSharedContainers",
			"sid=valid-token",
			principal,
			anchorOrderB,
			request,
		)
		if denied.called {
			t.Fatal("对锚点组织无访问权限时不应进入 handler")
		}
	})

	t.Run("跨组织可写访问的写操作正向放行并切换到锚点组织", func(t *testing.T) {
		orgA := uuid.New()
		orgB := uuid.New()
		anchorOrderB := &biz.Order{ID: uuid.New(), OrganizationID: orgB, BusinessType: biz.OrderBusinessSE}
		permission := access.OrderPermission(access.OrderBusinessSE, access.OrderContainerUpdate)
		principal := &biz.Principal{
			Organization: biz.Organization{ID: orgA},
			Permissions:  []string{permission},
			RoleScopes: []biz.RoleScope{
				{RoleCode: "operator", DataScope: biz.DataScopeOrganization},
			},
			RolePermissions: map[string]map[string]struct{}{
				"operator": {permission: {}},
			},
			OrganizationAccesses: []biz.OrganizationAccess{
				{OrganizationID: orgB, Writable: true},
			},
		}

		request := &orderv1.ConfirmSeaSharedContainerRequest{
			OrderId:         anchorOrderB.ID.String(),
			Id:              uuid.New().String(),
			ExpectedVersion: 1,
		}
		state := runSharedContainerMiddleware(
			t,
			"/order.v1.SeaSharedContainerService/ConfirmSeaSharedContainer",
			"sid=valid-token",
			principal,
			anchorOrderB,
			request,
		)
		if !state.called {
			t.Fatal("跨组织 Writable=true 时写操作应进入 handler")
		}
		if state.orgID != orgB {
			t.Fatalf("写操作 handler 内有效组织应为锚点组织 %s, 实际 %s", orgB, state.orgID)
		}
	})

	t.Run("写操作在无锚点组织写权限时拒绝", func(t *testing.T) {
		principal := principalWith(access.OrderContainerUpdate, false)
		// 主体属于其他组织且对锚点组织只有只读访问
		principal.Organization = biz.Organization{ID: uuid.New()}
		request := &orderv1.ConfirmSeaSharedContainerRequest{
			OrderId: anchorOrder.ID.String(),
			Id:      uuid.New().String(),
		}
		state := runSharedContainerMiddleware(t, "/order.v1.SeaSharedContainerService/ConfirmSeaSharedContainer", "sid=valid-token", principal, anchorOrder, request)
		if state.called {
			t.Fatal("对锚点组织无写权限的写操作不应进入 handler")
		}
	})
}

func operationNameFromRequest(request any) string {
	switch request.(type) {
	case *orderv1.ListSeaSharedContainersRequest:
		return "ListSeaSharedContainers"
	case *orderv1.GetSeaSharedContainerRequest:
		return "GetSeaSharedContainer"
	case *orderv1.ListSeaSharedContainerCandidatesRequest:
		return "ListSeaSharedContainerCandidates"
	case *orderv1.CreateSeaSharedContainerRequest:
		return "CreateSeaSharedContainer"
	case *orderv1.UpdateSeaSharedContainerRequest:
		return "UpdateSeaSharedContainer"
	case *orderv1.DeleteSeaSharedContainerRequest:
		return "DeleteSeaSharedContainer"
	case *orderv1.SaveSeaSharedContainerAllocationsDraftRequest:
		return "SaveSeaSharedContainerAllocationsDraft"
	case *orderv1.ConfirmSeaSharedContainerRequest:
		return "ConfirmSeaSharedContainer"
	case *orderv1.WithdrawSeaSharedContainerRequest:
		return "WithdrawSeaSharedContainer"
	default:
		return ""
	}
}
