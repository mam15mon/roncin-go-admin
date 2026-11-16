package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	khttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/uuid"
	orderv1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/service"
)

// 保留HTTP绑定、真实鉴权、Service与Usecase，仅替换仓储边界。
type seaTransportRouteRepo struct {
	biz.SeaOrderChangeRepo
	called bool
}

func (r *seaTransportRouteRepo) PreviewTransportExecutionUpdate(_ context.Context, _ uuid.UUID, input *biz.SeaTransportExecutionUpdateCommand) (*biz.SeaTransportExecutionUpdatePreview, error) {
	r.called = true
	return &biz.SeaTransportExecutionUpdatePreview{TransportExecutionID: uuid.New(), MemberOrderIDs: []uuid.UUID{input.OrderID}, Executable: true}, nil
}

func (r *seaTransportRouteRepo) ExecuteTransportExecutionUpdate(_ context.Context, _, _ uuid.UUID, _ *biz.SeaTransportExecutionUpdateCommand, _ *biz.AuditEvent) (*biz.SeaTransportExecutionUpdateResult, error) {
	r.called = true
	return &biz.SeaTransportExecutionUpdateResult{TransportExecution: &biz.SeaTransportExecution{ID: uuid.New(), Version: 2}, VersionID: uuid.New()}, nil
}

func TestSeaTransportHTTPAuthorizationUsesActualBusinessPermissions(t *testing.T) {
	companyID, foreignCompanyID := uuid.New(), uuid.New()
	for _, endpoint := range []string{"preview", "execute", "reassignment"} {
		for _, test := range []struct {
			name       string
			kind       access.OrderBusinessType
			permission string
			companyID  uuid.UUID
			allowed    bool
		}{
			{"SI编辑可更新航程", access.OrderBusinessSI, access.OrderPermission(access.OrderBusinessSI, access.OrderUpdate), companyID, true},
			{"SE保留改配权限", access.OrderBusinessSE, access.OrderPermission(access.OrderBusinessSE, access.OrderReassign), companyID, true},
			{"SE仅编辑不能改航程", access.OrderBusinessSE, access.OrderPermission(access.OrderBusinessSE, access.OrderUpdate), companyID, false},
			{"SI不能借SE改配", access.OrderBusinessSI, access.OrderPermission(access.OrderBusinessSE, access.OrderReassign), companyID, false},
			{"SI仅读不能改航程", access.OrderBusinessSI, access.OrderPermission(access.OrderBusinessSI, access.OrderRead), companyID, false},
			{"SI不能跨公司", access.OrderBusinessSI, access.OrderPermission(access.OrderBusinessSI, access.OrderUpdate), foreignCompanyID, false},
		} {
			if endpoint == "reassignment" && test.kind == access.OrderBusinessSE {
				continue // 本组只验证SI编辑权限未扩大成整票改配。
			}
			t.Run(endpoint+"/"+test.name, func(t *testing.T) {
				anchor := &biz.Order{ID: uuid.New(), OrganizationID: companyID, BusinessType: biz.OrderBusinessType(test.kind)}
				principal := &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: test.companyID, Kind: biz.OrganizationKindCompany}, OrganizationNodes: serverOrganizationNodes(test.companyID), RoleGrants: []biz.RoleGrant{serverRoleGrant("operator", biz.DataScopeOrganization, []string{test.permission})}}
				policy := &biz.SessionPolicy{CookieName: "sid", TTL: time.Hour, SameSite: "lax"}
				authUC := biz.NewAuthUsecase(&middlewareAuthRepoStub{session: &biz.Session{UserID: principal.UserID, OrganizationID: test.companyID, ExpiresAt: time.Now().Add(time.Hour)}, principal: principal}, policy, nil, nil)
				orderUC := biz.NewOrderUsecase(&anchorAwareOrderRepoStub{order: anchor}, nil, nil, nil, nil)
				srv := khttp.NewServer(khttp.ErrorEncoder(encodeError), khttp.Middleware(Authorization(authUC, policy, orderUC, nil), RequiredFieldsValidator()))
				repo := &seaTransportRouteRepo{}
				orderv1.RegisterSeaOrderChangeServiceHTTPServer(srv, service.NewSeaOrderChangeService(biz.NewSeaOrderChangeUsecase(repo, nil)))
				path := "/api/v1/orders/" + anchor.ID.String() + "/sea-order-change/transport-execution"
				body := `{"expectedTransportExecutionVersion":"1","input":{"eta":"2026-11-20T08:00:00Z"},"reason":"修订实际航次","idempotencyKey":"route-test"}`
				if endpoint == "preview" {
					path += "/preview"
				} else if endpoint == "reassignment" {
					path = "/api/v1/orders/" + anchor.ID.String() + "/sea-order-change/reassignment/execute"
					body = `{}`
				}
				request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
				request.Header.Set("Cookie", "sid=test-session")
				request.Header.Set("Content-Type", "application/json")
				response := httptest.NewRecorder()
				srv.ServeHTTP(response, request)
				expected := http.StatusForbidden
				allowed := test.allowed && endpoint != "reassignment"
				if allowed {
					expected = http.StatusOK
				}
				if response.Code != expected || repo.called != allowed {
					t.Fatalf("HTTP=%d 期望=%d 仓储进入=%v 响应=%s", response.Code, expected, repo.called, response.Body)
				}
			})
		}
	}
}

type seaCandidateRouteRepo struct {
	biz.SeaMasterBillRepo
	called       bool
	businessType biz.OrderBusinessType
}

func (r *seaCandidateRouteRepo) MatchCandidate(_ context.Context, _, _ uuid.UUID, _ string, _ *biz.SeaTransportExecution, businessType biz.OrderBusinessType) (*biz.SeaMasterBillMatchResult, error) {
	r.called, r.businessType = true, businessType
	return &biz.SeaMasterBillMatchResult{}, nil
}

func TestSeaCandidateHTTPAuthorizationUsesRequestedBusinessIdentity(t *testing.T) {
	for _, requested := range []string{"SE", "SI", "", "AE", "AI"} {
		for _, granted := range []access.OrderBusinessType{access.OrderBusinessSE, access.OrderBusinessSI} {
			t.Run(requested+"/"+string(granted), func(t *testing.T) {
				principal := principalWithOrderPermission(granted, access.OrderRead)
				policy := &biz.SessionPolicy{CookieName: "sid", TTL: time.Hour, SameSite: "lax"}
				authUC := biz.NewAuthUsecase(&middlewareAuthRepoStub{session: &biz.Session{UserID: uuid.New(), OrganizationID: principal.Organization.ID, ExpiresAt: time.Now().Add(time.Hour)}, principal: principal}, policy, nil, nil)
				repo := &seaCandidateRouteRepo{}
				orderUC := biz.NewOrderUsecase(nil, repo, nil, nil, nil)
				srv := khttp.NewServer(khttp.ErrorEncoder(encodeError), khttp.Middleware(Authorization(authUC, policy, orderUC, nil), RequiredFieldsValidator()))
				orderv1.RegisterOrderServiceHTTPServer(srv, service.NewOrderService(orderUC))
				query := url.Values{"businessType": {requested}, "shippingLineId": {uuid.NewString()}, "masterNo": {"IMPORT123"}}
				request := httptest.NewRequest(http.MethodGet, "/api/v1/orders/sea-master-bill-candidate?"+query.Encode(), nil)
				request.Header.Set("Cookie", "sid=test-session")
				response := httptest.NewRecorder()
				srv.ServeHTTP(response, request)
				allowed := requested == string(granted)
				expected := http.StatusForbidden
				if allowed {
					expected = http.StatusOK
				}
				if response.Code != expected || repo.called != allowed || allowed && string(repo.businessType) != requested {
					t.Fatalf("候选HTTP=%d 期望=%d 仓储=%+v 响应=%s", response.Code, expected, repo, response.Body)
				}
			})
		}
	}
}
