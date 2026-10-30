package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	khttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/uuid"
	orderv1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/service"
)

// 保留真实 HTTP 绑定、鉴权中间件、Service 与 Usecase，只替换仓储边界。
type sharedRouteRepo struct {
	biz.SeaSharedContainerRepo
	called                               string
	organizationID, orderID, executionID uuid.UUID
	keyword                              string
	page, pageSize                       int
}

func (r *sharedRouteRepo) List(_ context.Context, org, order, execution uuid.UUID, keyword string, page, size int) ([]*biz.SeaSharedContainer, int, error) {
	r.called, r.organizationID, r.orderID, r.executionID, r.keyword, r.page, r.pageSize = "List", org, order, execution, keyword, page, size
	return []*biz.SeaSharedContainer{}, 0, nil
}
func (r *sharedRouteRepo) ListCandidates(_ context.Context, org, order, execution uuid.UUID, keyword string, page, size int) ([]*biz.SeaSharedContainerCandidateOrder, int, error) {
	r.called, r.organizationID, r.orderID, r.executionID, r.keyword, r.page, r.pageSize = "Candidates", org, order, execution, keyword, page, size
	return []*biz.SeaSharedContainerCandidateOrder{}, 0, nil
}

func TestSharedContainerHTTPAuthorizationAndBinding(t *testing.T) {
	org, otherOrg, execution := uuid.New(), uuid.New(), uuid.New()
	anchor := &biz.Order{ID: uuid.New(), OrganizationID: org, BusinessType: biz.OrderBusinessSE}
	read := access.OrderPermission(access.OrderBusinessSE, access.OrderContainerRead)
	for _, resource := range []struct{ path, method, operation string }{
		{"sea-shared-containers", "List", orderv1.OperationSeaSharedContainerServiceListSeaSharedContainers},
		{"sea-shared-container-candidates", "Candidates", orderv1.OperationSeaSharedContainerServiceListSeaSharedContainerCandidates},
	} {
		for _, tc := range []struct {
			name            string
			permissions     []string
			company         uuid.UUID
			order           string
			snake, oldOrder bool
			expected        int
		}{
			{name: "合法同公司容器读权限及camelCase绑定", permissions: []string{read}, company: org, order: anchor.ID.String(), expected: 200},
			{name: "snake_case绑定", permissions: []string{read}, company: org, order: anchor.ID.String(), snake: true, expected: 200},
			{name: "仅普通订单读权限仍拒绝", permissions: []string{access.OrderPermission(access.OrderBusinessSE, access.OrderRead)}, company: org, order: anchor.ID.String(), expected: 403},
			{name: "仅创建权限无读取权限仍拒绝", permissions: []string{access.OrderPermission(access.OrderBusinessSE, access.OrderContainerCreate)}, company: org, order: anchor.ID.String(), expected: 403},
			{name: "其他品类权限仍拒绝", permissions: []string{access.OrderPermission(access.OrderBusinessAE, access.OrderContainerRead)}, company: org, order: anchor.ID.String(), expected: 403},
			{name: "其他公司权限仍拒绝", permissions: []string{read}, company: otherOrg, order: anchor.ID.String(), expected: 403},
			{name: "错误锚点拒绝", permissions: []string{read}, company: org, order: uuid.NewString(), expected: 403},
			{name: "缺少锚点拒绝", permissions: []string{read}, company: org, expected: 403},
			{name: "旧注册顺序复现误鉴权403", permissions: []string{read}, company: org, order: anchor.ID.String(), oldOrder: true, expected: 403},
		} {
			t.Run(resource.method+"/"+tc.name, func(t *testing.T) {
				principal := &biz.Principal{Organization: biz.Organization{ID: tc.company, Kind: biz.OrganizationKindCompany}, OrganizationNodes: serverOrganizationNodes(tc.company), RoleGrants: []biz.RoleGrant{serverRoleGrant("operator", biz.DataScopeOrganization, tc.permissions)}}
				policy := &biz.SessionPolicy{CookieName: "sid", TTL: time.Hour, SameSite: "lax"}
				authUC := biz.NewAuthUsecase(&middlewareAuthRepoStub{session: &biz.Session{UserID: uuid.New(), OrganizationID: tc.company, ExpiresAt: time.Now().Add(time.Hour)}, principal: principal}, policy, nil, nil)
				orderUC := biz.NewOrderUsecase(&anchorAwareOrderRepoStub{order: anchor}, nil, nil, nil, newReminderModeCreditControl(), nil)
				var operation string
				srv := khttp.NewServer(khttp.ErrorEncoder(encodeError), khttp.Middleware(func(next middleware.Handler) middleware.Handler {
					return func(ctx context.Context, req any) (any, error) {
						tr, _ := transport.FromServerContext(ctx)
						operation = tr.Operation()
						return next(ctx, req)
					}
				}, Authorization(authUC, policy, orderUC, nil), RequiredFieldsValidator()))
				repo := &sharedRouteRepo{}
				shared := service.NewSeaSharedContainerService(biz.NewSeaSharedContainerUsecase(repo))
				orders := &recordingOrderServiceHTTPServer{}
				if tc.oldOrder {
					orderv1.RegisterOrderServiceHTTPServer(srv, orders)
					orderv1.RegisterSeaSharedContainerServiceHTTPServer(srv, shared)
				} else {
					registerOrderAndSharedContainerHTTPServices(srv, orders, shared)
				}
				query := url.Values{"orderId": {tc.order}, "transportExecutionId": {execution.String()}, "page": {"2"}, "pageSize": {"17"}, "keyword": {"测试箱"}}
				if tc.snake {
					query.Set("order_id", query.Get("orderId"))
					query.Del("orderId")
					query.Set("transport_execution_id", query.Get("transportExecutionId"))
					query.Del("transportExecutionId")
					query.Set("page_size", query.Get("pageSize"))
					query.Del("pageSize")
				}
				request := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+resource.path+"?"+query.Encode(), nil)
				request.Header.Set("Cookie", "sid=test-session")
				response := httptest.NewRecorder()
				srv.ServeHTTP(response, request)
				if response.Code != tc.expected {
					t.Fatalf("status=%d expected=%d body=%s operation=%s", response.Code, tc.expected, response.Body, operation)
				}
				expectedOperation := resource.operation
				if tc.oldOrder {
					expectedOperation = orderv1.OperationOrderServiceGetOrder
				}
				if operation != expectedOperation {
					t.Fatalf("operation=%s expected=%s", operation, expectedOperation)
				}
				if tc.expected != 200 {
					if repo.called != "" {
						t.Fatal("无权请求进入业务仓储")
					}
					return
				}
				if orders.calledMethod != "" || repo.called != resource.method || repo.orderID != anchor.ID || repo.executionID != execution || repo.organizationID != org || repo.page != 2 || repo.pageSize != 17 || repo.keyword != "测试箱" {
					t.Fatalf("绑定或组织上下文错误：repo=%+v order=%+v", repo, orders)
				}
			})
		}
	}
}

func TestSharedContainerRegistrationPreservesOrderDetail(t *testing.T) {
	orders := &recordingOrderServiceHTTPServer{}
	srv := khttp.NewServer()
	registerOrderAndSharedContainerHTTPServices(srv, orders, service.NewSeaSharedContainerService(nil))
	id := uuid.NewString()
	response := httptest.NewRecorder()
	srv.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+id, nil))
	if response.Code != 200 || orders.calledMethod != "GetOrder" || orders.getOrderID != id {
		t.Fatalf("订单详情分发错误: %d %+v", response.Code, orders)
	}
}
