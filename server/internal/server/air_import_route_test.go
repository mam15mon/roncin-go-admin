package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	khttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/service"
)

// 真实HTTP绑定、Authorization及Service，仅替换仓储；目标类型不由请求猜测。
type airRouteRepo struct {
	biz.AirExportRepo
	org, masterID, orderID uuid.UUID
	kind                   biz.OrderBusinessType
	called                 bool
}

func (r *airRouteRepo) BusinessType(_ context.Context, org uuid.UUID, scope string, id uuid.UUID) (biz.OrderBusinessType, error) {
	if org != r.org || (scope != "MASTER" || id != r.masterID) && (scope != "ORDER" || id != r.orderID) {
		return "", biz.ErrAirWaybillNotFound
	}
	return r.kind, nil
}
func (r *airRouteRepo) detail() *biz.AirMasterDetail {
	return &biz.AirMasterDetail{Master: &biz.AirMasterWaybill{ID: r.masterID, OrganizationID: r.org, BusinessType: r.kind, Version: 1}}
}
func (r *airRouteRepo) List(_ context.Context, _ uuid.UUID, kind biz.OrderBusinessType, _ biz.SelectorListOptions) ([]*biz.AirMasterWaybill, int, error) {
	r.called = true
	if kind != r.kind {
		return nil, 0, biz.ErrAirWaybillInvalid
	}
	return []*biz.AirMasterWaybill{r.detail().Master}, 1, nil
}
func (r *airRouteRepo) Get(context.Context, uuid.UUID, uuid.UUID, biz.SelectorListOptions) (*biz.AirMasterDetail, error) {
	r.called = true
	return r.detail(), nil
}
func (r *airRouteRepo) Preview(context.Context, uuid.UUID, uuid.UUID, uint64, *biz.AirMasterWaybillInput) (*biz.AirMasterPreview, error) {
	r.called = true
	return &biz.AirMasterPreview{CanExecute: true}, nil
}
func (r *airRouteRepo) Update(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, *biz.AirMasterWaybillInput, string, string) error {
	r.called = true
	return nil
}
func (r *airRouteRepo) Link(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uint64, uint64, string, string) error {
	r.called = true
	return nil
}
func (r *airRouteRepo) LinkCandidates(context.Context, uuid.UUID, uuid.UUID, biz.SelectorListOptions) ([]*biz.AirMasterMember, int, error) {
	r.called = true
	return nil, 0, nil
}
func (r *airRouteRepo) HouseVersions(context.Context, uuid.UUID, uuid.UUID, biz.SelectorListOptions) ([]*biz.AirHouseVersion, int, error) {
	r.called = true
	return nil, 0, nil
}

type airWeightRouteRepo struct {
	biz.AirWeightRepo
	called bool
}

func (r *airWeightRouteRepo) Preview(context.Context, uuid.UUID, *biz.AirWeightInput) (*biz.AirWeightPreview, error) {
	r.called = true
	return &biz.AirWeightPreview{}, nil
}
func (r *airWeightRouteRepo) Confirm(context.Context, uuid.UUID, uuid.UUID, *biz.AirWeightInput) (*biz.AirWeightConfirmation, error) {
	r.called = true
	return &biz.AirWeightConfirmation{}, nil
}
func (r *airWeightRouteRepo) List(context.Context, uuid.UUID, string, uuid.UUID, int, int) ([]*biz.AirWeightConfirmation, int, error) {
	r.called = true
	return nil, 0, nil
}

func TestAirHTTPAuthorizationUsesActualTargetBusinessType(t *testing.T) {
	org, masterID, orderID := uuid.New(), uuid.New(), uuid.New()
	input := fmt.Sprintf(`{"masterNo":"78112345675","airlineId":"%s","originAirportId":"%s","destinationAirportId":"%s","flightNo":"MU587","etd":"2026-11-16T01:00:00Z","eta":"2026-11-16T09:00:00Z"}`, uuid.New(), uuid.New(), uuid.New())
	for _, kind := range []biz.OrderBusinessType{biz.OrderBusinessAE, biz.OrderBusinessAI} {
		for _, endpoint := range []struct {
			name, method, path, body string
			op                       access.OrderOperation
			transition, weight       bool
		}{
			{name: "主单详情", method: http.MethodGet, path: "/api/v1/air-export/master-waybills/" + masterID.String() + "?businessType=AE", op: access.OrderRead},
			{name: "共享预览", method: http.MethodPost, path: "/api/v1/air-export/master-waybills/" + masterID.String() + "/preview", body: `{"expectedVersion":"1","input":` + input + `}`, op: access.OrderUpdate},
			{name: "共享写入", method: http.MethodPut, path: "/api/v1/air-export/master-waybills/" + masterID.String(), body: `{"expectedVersion":"1","inputFingerprint":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","reason":"修订","input":` + input + `}`, op: access.OrderUpdate},
			{name: "恢复候选", method: http.MethodGet, path: "/api/v1/air-export/master-waybills/" + masterID.String() + "/link-candidates", op: access.OrderUpdate},
			{name: "原票恢复", method: http.MethodPost, path: "/api/v1/air-export/master-waybills/" + masterID.String() + "/members", body: fmt.Sprintf(`{"orderId":"%s","expectedMasterVersion":"1","expectedOrderVersion":"1","houseNo":"H001","reason":"恢复原票"}`, orderID), op: access.OrderUpdate, transition: true},
			{name: "分单历史", method: http.MethodGet, path: "/api/v1/air-export/orders/" + orderID.String() + "/house-versions", op: access.OrderRead},
			{name: "本票重量预览", method: http.MethodPost, path: "/api/v1/air-weights/preview", body: fmt.Sprintf(`{"scope":"ORDER","targetId":"%s","source":"PLANNED"}`, orderID), op: access.OrderRead, weight: true},
			{name: "主单重量预览", method: http.MethodPost, path: "/api/v1/air-weights/preview", body: fmt.Sprintf(`{"scope":"MASTER","targetId":"%s","source":"ACTUAL"}`, masterID), op: access.OrderRead, weight: true},
			{name: "本票重量确认", method: http.MethodPost, path: "/api/v1/air-weights/confirm", body: fmt.Sprintf(`{"scope":"ORDER","targetId":"%s","source":"PLANNED","inputFingerprint":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","confirmedWeightKg":"100"}`, orderID), op: access.OrderUpdate, weight: true},
			{name: "主单重量确认", method: http.MethodPost, path: "/api/v1/air-weights/confirm", body: fmt.Sprintf(`{"scope":"MASTER","targetId":"%s","source":"ACTUAL","inputFingerprint":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","confirmedWeightKg":"100"}`, masterID), op: access.OrderUpdate, weight: true},
			{name: "重量历史", method: http.MethodGet, path: "/api/v1/air-weights/history?scope=MASTER&targetId=" + masterID.String(), op: access.OrderRead, weight: true},
		} {
			for _, test := range []struct {
				name                                              string
				otherKind, crossCompany, omitTransition, readOnly bool
				allowed                                           bool
			}{
				{name: "真实类型权限", allowed: true}, {name: "只读权限", readOnly: true, allowed: endpoint.op == access.OrderRead}, {name: "另一方向权限", otherKind: true}, {name: "跨公司", crossCompany: true}, {name: "恢复缺生命周期权限", omitTransition: true, allowed: !endpoint.transition},
			} {
				t.Run(string(kind)+"/"+endpoint.name+"/"+test.name, func(t *testing.T) {
					grant := access.OrderBusinessType(kind)
					if test.otherKind {
						if kind == biz.OrderBusinessAE {
							grant = access.OrderBusinessAI
						} else {
							grant = access.OrderBusinessAE
						}
					}
					operation := endpoint.op
					if test.readOnly {
						operation = access.OrderRead
					}
					p := principalWithOrderPermission(grant, operation)
					p.UserID = uuid.New()
					p.Organization.ID = org
					p.OrganizationNodes = serverOrganizationNodes(org)
					if test.crossCompany {
						p.Organization.ID = uuid.New()
						p.OrganizationNodes = serverOrganizationNodes(p.Organization.ID)
					}
					if endpoint.transition && !test.omitTransition {
						p.RoleGrants[0].Permissions[access.OrderPermission(grant, access.OrderTransition)] = struct{}{}
					}
					policy := &biz.SessionPolicy{CookieName: "sid", TTL: time.Hour, SameSite: "lax"}
					auth := biz.NewAuthUsecase(&middlewareAuthRepoStub{session: &biz.Session{UserID: p.UserID, OrganizationID: p.Organization.ID, ExpiresAt: time.Now().Add(time.Hour)}, principal: p}, policy, nil, nil)
					repo := &airRouteRepo{org: org, masterID: masterID, orderID: orderID, kind: kind}
					air := biz.NewAirExportUsecase(repo)
					weights := &airWeightRouteRepo{}
					srv := khttp.NewServer(khttp.ErrorEncoder(encodeError), khttp.Middleware(Authorization(auth, policy, nil, nil, air), RequiredFieldsValidator()))
					v1.RegisterAirExportServiceHTTPServer(srv, service.NewAirExportService(air))
					v1.RegisterAirWeightServiceHTTPServer(srv, service.NewAirWeightService(biz.NewAirWeightUsecase(weights)))
					req := httptest.NewRequest(endpoint.method, endpoint.path, strings.NewReader(endpoint.body))
					req.Header.Set("Cookie", "sid=test")
					req.Header.Set("Content-Type", "application/json")
					response := httptest.NewRecorder()
					srv.ServeHTTP(response, req)
					want := http.StatusForbidden
					if test.allowed {
						want = http.StatusOK
					}
					called := repo.called
					if endpoint.weight {
						called = weights.called
					}
					if response.Code != want || called != test.allowed {
						t.Fatalf("HTTP=%d 期望=%d 仓储=%v 响应=%s", response.Code, want, called, response.Body)
					}
				})
			}
		}
	}
}

func TestAirCandidateHTTPRequiresExplicitBusinessType(t *testing.T) {
	for _, requested := range []string{"AE", "AI", "", "SE", "UNKNOWN"} {
		for _, granted := range []access.OrderBusinessType{access.OrderBusinessAE, access.OrderBusinessAI} {
			t.Run(requested+"/"+string(granted), func(t *testing.T) {
				p := principalWithOrderPermission(granted, access.OrderRead)
				policy := &biz.SessionPolicy{CookieName: "sid", TTL: time.Hour, SameSite: "lax"}
				auth := biz.NewAuthUsecase(&middlewareAuthRepoStub{session: &biz.Session{UserID: p.UserID, OrganizationID: p.Organization.ID, ExpiresAt: time.Now().Add(time.Hour)}, principal: p}, policy, nil, nil)
				repo := &airRouteRepo{org: p.Organization.ID, masterID: uuid.New(), kind: biz.OrderBusinessType(granted)}
				air := biz.NewAirExportUsecase(repo)
				srv := khttp.NewServer(khttp.ErrorEncoder(encodeError), khttp.Middleware(Authorization(auth, policy, nil, nil, air), RequiredFieldsValidator()))
				v1.RegisterAirExportServiceHTTPServer(srv, service.NewAirExportService(air))
				req := httptest.NewRequest(http.MethodGet, "/api/v1/air-export/master-waybills?businessType="+requested, nil)
				req.Header.Set("Cookie", "sid=test")
				response := httptest.NewRecorder()
				srv.ServeHTTP(response, req)
				allowed := requested == string(granted)
				want := http.StatusForbidden
				if allowed {
					want = http.StatusOK
				}
				if response.Code != want || repo.called != allowed {
					t.Fatalf("候选HTTP=%d 期望=%d 仓储=%v 响应=%s", response.Code, want, repo.called, response.Body)
				}
			})
		}
	}
}
