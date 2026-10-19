package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/uuid"
	authv1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/service"
	"google.golang.org/protobuf/encoding/protojson"
)

type embeddedProvider struct {
	enabled bool
	calls   int
}

func (p *embeddedProvider) Enabled() bool      { return p.enabled }
func (*embeddedProvider) EnterpriseID() string { return "fixed-corp" }
func (*embeddedProvider) AuthorizeURL(string) (string, error) {
	panic("免登不应申请扫码配置")
}
func (*embeddedProvider) ResolveIdentity(context.Context, string) (*biz.DingTalkIdentity, error) {
	panic("免登不应解析 OAuth code")
}
func (p *embeddedProvider) ResolveEmbeddedIdentity(_ context.Context, code string) (*biz.DingTalkIdentity, error) {
	p.calls++
	if code != "temporary-code" {
		return nil, biz.ErrDingTalkCodeInvalid
	}
	return &biz.DingTalkIdentity{UnionID: "union", UserID: "user", CorpID: "fixed-corp", Name: "成员"}, nil
}

type embeddedAuthRepo struct {
	biz.AuthRepo
	credential *biz.Credential
	choices    []biz.OrganizationChoice
	sessions   []*biz.Session
	conflict   bool
}

func (r *embeddedAuthRepo) FindDingTalkCredential(_ context.Context, identity *biz.DingTalkIdentity) (*biz.Credential, error) {
	if identity.UnionID != "union" || identity.UserID != "user" || identity.CorpID != "fixed-corp" {
		return nil, biz.ErrDingTalkLoginFailed
	}
	if r.conflict {
		return nil, biz.ErrDingTalkIdentityConflict
	}
	if r.credential == nil {
		return nil, biz.ErrDingTalkNotRegistered
	}
	return r.credential, nil
}
func (r *embeddedAuthRepo) ListEnabledMembershipOrganizations(context.Context, uuid.UUID) ([]biz.OrganizationChoice, error) {
	return r.choices, nil
}
func (r *embeddedAuthRepo) ResolvePrincipal(_ context.Context, userID, orgID uuid.UUID) (*biz.Principal, error) {
	p := &biz.Principal{UserID: userID, Organization: biz.Organization{ID: orgID, Kind: biz.OrganizationKindCompany}}
	for _, choice := range r.choices {
		p.Organizations = append(p.Organizations, biz.Organization{ID: choice.OrganizationID, Name: choice.OrganizationName, Kind: biz.OrganizationKindCompany})
	}
	return p, nil
}
func (r *embeddedAuthRepo) CreateSession(_ context.Context, s *biz.Session, _ string, audit *biz.AuditEvent) error {
	if audit.Action != "auth.dingtalk.login" {
		panic("审计口径改变")
	}
	r.sessions = append(r.sessions, s)
	return nil
}

func embeddedHTTPServer(repo *embeddedAuthRepo, p *embeddedProvider) *kratoshttp.Server {
	policy := &biz.SessionPolicy{CookieName: "roncin_session", TTL: time.Hour, Secure: true, SameSite: "lax"}
	uc := biz.NewAuthUsecase(repo, policy, p, nil)
	server := kratoshttp.NewServer(kratoshttp.ResponseEncoder(encodeResponse), kratoshttp.ErrorEncoder(encodeError), kratoshttp.Middleware(Authorization(uc, policy, nil, nil), RequiredFieldsValidator()))
	authv1.RegisterAuthServiceHTTPServer(server, service.NewAuthService(uc, policy, nil))
	return server
}
func embeddedRequest(server *kratoshttp.Server, method, path, body, cookie string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "test-agent")
	if cookie != "" {
		request.Header.Set("Cookie", cookie)
	}
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	return response
}
func embeddedConfig(t *testing.T, server *kratoshttp.Server) (string, *http.Cookie) {
	t.Helper()
	response := embeddedRequest(server, "GET", "/api/v1/auth/dingtalk/embedded-config", "", "")
	var body struct {
		Success bool `json:"success"`
		Data    struct {
			Enabled   bool   `json:"enabled"`
			CorpID    string `json:"corpId"`
			State     string `json:"state"`
			ExpiresAt string `json:"expiresAt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	cookies := response.Result().Cookies()
	if response.Code != 200 || !body.Success || !body.Data.Enabled || body.Data.CorpID != "fixed-corp" || len(cookies) != 1 || response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("Pragma") != "no-cache" {
		t.Fatalf("公开配置路由错误: %d %s", response.Code, response.Body.String())
	}
	cookie := cookies[0]
	expires, err := time.Parse(time.RFC3339, body.Data.ExpiresAt)
	if err != nil || cookie.Name != "roncin_session_dingtalk_embedded_state" || cookie.Value != body.Data.State || cookie.Value == "" || cookie.MaxAge != 300 || cookie.Path != "/" || !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || !expires.Equal(cookie.Expires) {
		t.Fatalf("独立 Cookie 属性错误: %+v %v", cookie, err)
	}
	return body.Data.State, cookie
}
func TestDingTalkEmbeddedHTTPConfigAndLoginBoundary(t *testing.T) {
	for _, item := range []struct {
		name                                                   string
		pending, disabled, noOrg, unsynced, conflict, multiple bool
		status                                                 int
		want                                                   string
		session                                                bool
	}{
		{name: "单工作台", status: 200, want: `"status":1`, session: true},
		{name: "多工作台", multiple: true, status: 200, want: `"status":1`, session: true},
		{name: "未同步", unsynced: true, status: 200, want: `"status":2`},
		{name: "待分配", pending: true, status: 200, want: `"status":3`},
		{name: "停用", disabled: true, status: 200, want: `"status":4`},
		{name: "无资格", noOrg: true, status: 403, want: "ORGANIZATION_FORBIDDEN"},
		{name: "身份冲突", conflict: true, status: 409, want: "DINGTALK_IDENTITY_CONFLICT"},
	} {
		t.Run(item.name, func(t *testing.T) {
			repo := &embeddedAuthRepo{credential: &biz.Credential{UserID: uuid.New(), Enabled: !item.disabled, PendingAssignment: item.pending}, choices: []biz.OrganizationChoice{{OrganizationID: uuid.New(), OrganizationName: "公司"}}, conflict: item.conflict}
			if item.multiple {
				repo.choices = append(repo.choices, biz.OrganizationChoice{OrganizationID: uuid.New(), OrganizationName: "第二公司"})
			}
			if item.unsynced {
				repo.credential = nil
			}
			if item.noOrg {
				repo.choices = nil
			}
			provider := &embeddedProvider{enabled: true}
			server := embeddedHTTPServer(repo, provider)
			state, cookie := embeddedConfig(t, server)
			body, _ := json.Marshal(map[string]string{"authCode": "temporary-code", "state": state})
			response := embeddedRequest(server, "POST", "/api/v1/auth/dingtalk/embedded-login", string(body), cookie.Name+"="+cookie.Value+"; roncin_session_dingtalk_state=scan-state")
			if response.Code != item.status || !strings.Contains(response.Body.String(), item.want) || response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("Pragma") != "no-cache" || provider.calls != 1 {
				t.Fatalf("换码边界错误: %d %s calls=%d", response.Code, response.Body.String(), provider.calls)
			}
			cookies := response.Result().Cookies()
			wantCookies := 1
			if item.session {
				wantCookies++
			}
			if len(cookies) != wantCookies {
				t.Fatalf("state 清理或会话 Cookie 缺失: %+v", cookies)
			}
			session := false
			for _, c := range cookies {
				switch c.Name {
				case "roncin_session_dingtalk_embedded_state":
					if c.MaxAge != -1 || c.Value != "" {
						t.Fatal("未清理客户端 state")
					}
				case "roncin_session":
					session = c.Value != ""
				default:
					t.Fatal("修改了扫码 Cookie")
				}
			}
			if session != item.session || (len(repo.sessions) == 1) != item.session {
				t.Fatalf("会话发放错误: session=%v created=%d", session, len(repo.sessions))
			}
			if item.multiple {
				var decoded authv1.DingTalkEmbeddedLoginResponse
				if err := protojson.Unmarshal(response.Body.Bytes(), &decoded); err != nil || len(decoded.GetData().GetCurrentUser().GetOrganizations()) != 2 {
					t.Fatalf("多工作台候选缺失: %s %v", response.Body.String(), err)
				}
			}
			if item.session && (!strings.Contains(response.Body.String(), `"currentUser"`) || !strings.Contains(response.Body.String(), `"currentOrganization"`)) {
				t.Fatal("响应缺少 camelCase 当前用户")
			}
		})
	}
}
func TestDingTalkEmbeddedHTTPRejectsInvalidStateAndDisabled(t *testing.T) {
	for _, item := range []struct {
		name, body, cookie string
		enabled            bool
		status             int
	}{
		{"缺Cookie", `{"authCode":"temporary-code","state":"s"}`, "", true, 401},
		{"Cookie不匹配", `{"authCode":"temporary-code","state":"s"}`, "roncin_session_dingtalk_embedded_state=x", true, 401},
		{"扫码Cookie不可替代", `{"authCode":"temporary-code","state":"s"}`, "roncin_session_dingtalk_state=s", true, 401},
		{"缺授权码", `{"state":"s"}`, "roncin_session_dingtalk_embedded_state=s", true, 400},
		{"缺state", `{"authCode":"temporary-code"}`, "roncin_session_dingtalk_embedded_state=s", true, 400},
		{"畸形JSON", `{"authCode":`, "roncin_session_dingtalk_embedded_state=s", true, 400},
		{"禁用", `{"authCode":"temporary-code","state":"s"}`, "roncin_session_dingtalk_embedded_state=s", false, 503},
	} {
		t.Run(item.name, func(t *testing.T) {
			provider := &embeddedProvider{enabled: item.enabled}
			server := embeddedHTTPServer(&embeddedAuthRepo{}, provider)
			response := embeddedRequest(server, "POST", "/api/v1/auth/dingtalk/embedded-login", item.body, item.cookie)
			if response.Code != item.status || provider.calls != 0 || response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("Pragma") != "no-cache" {
				t.Fatalf("未在换码前拒绝: %d %s calls=%d", response.Code, response.Body.String(), provider.calls)
			}
		})
	}
	server := embeddedHTTPServer(&embeddedAuthRepo{}, &embeddedProvider{})
	response := embeddedRequest(server, "GET", "/api/v1/auth/dingtalk/embedded-config", "", "")
	if response.Code != 200 || len(response.Result().Cookies()) != 0 || strings.Contains(response.Body.String(), "corpId") || strings.Contains(response.Body.String(), "state") || strings.Contains(response.Body.String(), "expiresAt") || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("禁用配置泄漏: %s", response.Body.String())
	}
}
