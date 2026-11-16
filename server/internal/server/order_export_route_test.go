package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	khttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/service"
)

type exportRouteRepo struct {
	biz.OrderRepo
	org     uuid.UUID
	called  bool
	scopes  []biz.OrderOrganizationScope
	options biz.OrderListOptions
}

func (r *exportRouteRepo) Export(_ context.Context, scopes []biz.OrderOrganizationScope, options biz.OrderListOptions) (*biz.OrderExport, error) {
	r.called, r.scopes, r.options = true, scopes, options
	return &biz.OrderExport{Items: []*biz.Order{}, GeneratedAt: time.Now()}, nil
}

type exportRecordingService struct {
	recordingOrderServiceHTTPServer
}

func (s *exportRecordingService) ExportOrders(context.Context, *v1.ExportOrdersRequest) (*v1.ExportOrdersResponse, error) {
	s.calledMethod = "ExportOrders"
	return &v1.ExportOrdersResponse{Success: true}, nil
}
func TestOrderExportStaticRouteAndDetailDispatch(t *testing.T) {
	for _, tc := range []struct{ path, method, operation string }{{"/api/v1/orders/export?businessType=1", "ExportOrders", v1.OperationOrderServiceExportOrders}, {"/api/v1/orders/" + uuid.NewString(), "GetOrder", v1.OperationOrderServiceGetOrder}} {
		stub := &exportRecordingService{}
		var operation string
		srv := khttp.NewServer(khttp.Middleware(func(next middleware.Handler) middleware.Handler {
			return func(ctx context.Context, req any) (any, error) {
				tr, _ := transport.FromServerContext(ctx)
				operation = tr.Operation()
				return next(ctx, req)
			}
		}))
		v1.RegisterOrderServiceHTTPServer(srv, stub)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if rec.Code != 200 || stub.calledMethod != tc.method || operation != tc.operation {
			t.Fatalf("静态/详情实际分发错误: %d %s %s", rec.Code, stub.calledMethod, operation)
		}
		if tc.method == "GetOrder" {
			if _, err := uuid.Parse(stub.getOrderID); err != nil {
				t.Fatal(err)
			}
		}
	}
}
func TestOrderExportHTTPFourTypePermissionMatrix(t *testing.T) {
	for _, kind := range []struct {
		wire   string
		access access.OrderBusinessType
		biz    biz.OrderBusinessType
	}{{"1", access.OrderBusinessSE, biz.OrderBusinessSE}, {"2", access.OrderBusinessSI, biz.OrderBusinessSI}, {"3", access.OrderBusinessAE, biz.OrderBusinessAE}, {"4", access.OrderBusinessAI, biz.OrderBusinessAI}} {
		for _, tc := range []struct {
			name           string
			permissionType access.OrderBusinessType
			system         bool
			status         int
		}{{"同类公司", kind.access, false, 200}, {"他类拒绝", access.OrderBusinessLand, false, 403}, {"系统工作台拒绝", kind.access, true, 403}} {
			t.Run(string(kind.access)+tc.name, func(t *testing.T) {
				org := uuid.New()
				organizationKind := biz.OrganizationKindCompany
				if tc.system {
					organizationKind = biz.OrganizationKindSystem
				}
				p := &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: org, Kind: organizationKind}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: org, Kind: organizationKind}}, RoleGrants: []biz.RoleGrant{serverRoleGrant("read", biz.DataScopeOrganization, []string{access.OrderPermission(tc.permissionType, access.OrderRead)})}}
				policy := &biz.SessionPolicy{CookieName: "sid", TTL: time.Hour, SameSite: "lax"}
				authUC := biz.NewAuthUsecase(&middlewareAuthRepoStub{session: &biz.Session{UserID: p.UserID, OrganizationID: org, ExpiresAt: time.Now().Add(time.Hour)}, principal: p}, policy, nil, nil)
				repo := &exportRouteRepo{org: org}
				uc := biz.NewOrderUsecase(repo, nil, nil, nil, nil)
				var operation string
				srv := khttp.NewServer(khttp.ErrorEncoder(encodeError), khttp.Middleware(func(next middleware.Handler) middleware.Handler {
					return func(ctx context.Context, req any) (any, error) {
						tr, _ := transport.FromServerContext(ctx)
						operation = tr.Operation()
						return next(ctx, req)
					}
				}, Authorization(authUC, policy, uc, nil, nil), RequiredFieldsValidator()))
				v1.RegisterOrderServiceHTTPServer(srv, service.NewOrderService(uc))
				request := httptest.NewRequest(http.MethodGet, "/api/v1/orders/export?businessType="+kind.wire+"&sortBy=eta&sortOrder=ascend&isLocked=false", nil)
				request.Header.Set("Cookie", "sid=test-session")
				rec := httptest.NewRecorder()
				srv.ServeHTTP(rec, request)
				if rec.Code != tc.status || operation != v1.OperationOrderServiceExportOrders {
					t.Fatalf("HTTP矩阵错误 %d want%d op%s body%s", rec.Code, tc.status, operation, rec.Body)
				}
				if tc.status != 200 {
					if repo.called {
						t.Fatal("越权调用了导出仓储")
					}
					return
				}
				if !repo.called || len(repo.scopes) != 1 || repo.scopes[0].BusinessType != kind.biz || len(repo.scopes[0].OrganizationIDs) != 1 || repo.scopes[0].OrganizationIDs[0] != org || repo.options.Page != 0 || repo.options.PageSize != 0 || repo.options.IsLocked == nil || *repo.options.IsLocked {
					t.Fatalf("导出身份/未设false/无分页错误: %+v %+v", repo.options, repo.scopes)
				}
			})
		}
	}
}
