package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"google.golang.org/protobuf/encoding/protojson"
)

type bootstrapDingTalkProvider struct {
	enabled bool
	failure error
}

func (p *bootstrapDingTalkProvider) Enabled() bool { return p.enabled }
func (p *bootstrapDingTalkProvider) AuthorizeURL(state string) (string, error) {
	if p.failure != nil {
		return "", p.failure
	}
	return "https://login.dingtalk.com/oauth2/auth?client_id=public-app&state=" + state, nil
}
func (*bootstrapDingTalkProvider) ResolveIdentity(context.Context, string) (*biz.DingTalkIdentity, error) {
	return nil, errors.New("本测试不调用身份解析")
}

func newBootstrapAuthService(provider *bootstrapDingTalkProvider, sameSite string) *AuthService {
	policy := &biz.SessionPolicy{CookieName: "roncin_session", TTL: time.Hour, Secure: true, SameSite: sameSite}
	return NewAuthService(biz.NewAuthUsecase(nil, policy, provider, nil), policy, nil)
}

func assertBootstrapStateCookie(t *testing.T, payload []byte, cookie *http.Cookie, sameSite http.SameSite) string {
	t.Helper()
	var reply v1.GetDingTalkLoginConfigResponse
	if err := protojson.Unmarshal(payload, &reply); err != nil {
		t.Fatal(err)
	}
	if !reply.Success || !reply.GetData().GetEnabled() {
		t.Fatalf("错误的配置信封: %s", payload)
	}
	authorizeURL, err := url.Parse(reply.Data.GetAuthorizeUrl())
	if err != nil {
		t.Fatal(err)
	}
	if cookie == nil || cookie.Value == "" || authorizeURL.Query().Get("state") != cookie.Value {
		t.Fatalf("授权 state 与 Cookie 不一致: %s %+v", payload, cookie)
	}
	if cookie.Name != "roncin_session_dingtalk_state" || cookie.Path != "/" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != sameSite || cookie.MaxAge != 300 {
		t.Fatalf("Cookie 安全属性错误: %+v", cookie)
	}
	expiresAt, err := time.Parse(time.RFC3339, reply.Data.GetExpiresAt())
	if err != nil || !expiresAt.Equal(cookie.Expires.Truncate(time.Second)) || time.Until(cookie.Expires) < 4*time.Minute || time.Until(cookie.Expires) > 5*time.Minute {
		t.Fatalf("失效时间不对应: %s %+v %v", payload, cookie, err)
	}
	return cookie.Value
}

func TestDingTalkLoginBootstrapAndAPIUseSameConfigAndCookieContract(t *testing.T) {
	for _, policy := range []struct {
		name string
		mode http.SameSite
	}{{"lax", http.SameSiteLaxMode}, {"strict", http.SameSiteStrictMode}} {
		t.Run(policy.name, func(t *testing.T) {
			s := newBootstrapAuthService(&bootstrapDingTalkProvider{enabled: true}, policy.name)
			var previousState string
			for range 2 {
				payload, cookie, err := s.DingTalkLoginBootstrap(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(payload), `"authorizeUrl"`) || !strings.Contains(string(payload), `"expiresAt"`) || strings.Contains(string(payload), `"authorize_url"`) {
					t.Fatalf("HTML 配置字段未沿用 API 的 camelCase JSON: %s", payload)
				}
				state := assertBootstrapStateCookie(t, payload, cookie, policy.mode)
				if state == previousState {
					t.Fatal("两次 HTML 共享登录 state")
				}
				previousState = state
			}
			server := kratoshttp.NewServer()
			v1.RegisterAuthServiceHTTPServer(server, s)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/auth/dingtalk/login-config", nil))
			cookies := response.Result().Cookies()
			if response.Code != 200 || len(cookies) != 1 || response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("Pragma") != "no-cache" {
				t.Fatalf("原配置接口行为改变: %d %v %s", response.Code, response.Header(), response.Body.String())
			}
			assertBootstrapStateCookie(t, response.Body.Bytes(), cookies[0], policy.mode)
		})
	}
}

func TestDingTalkLoginBootstrapDisabledAndFailureAreExplicit(t *testing.T) {
	for _, item := range []struct {
		name     string
		provider *bootstrapDingTalkProvider
		success  bool
		code     int
	}{
		{"禁用", &bootstrapDingTalkProvider{}, true, 0},
		{"配置失败", &bootstrapDingTalkProvider{enabled: true, failure: errors.New("secret=敏感错误")}, false, 500},
	} {
		t.Run(item.name, func(t *testing.T) {
			s := newBootstrapAuthService(item.provider, "lax")
			payload, cookie, err := s.DingTalkLoginBootstrap(context.Background())
			if err != nil || cookie != nil {
				t.Fatalf("无效 Cookie 或错误: %+v %v", cookie, err)
			}
			var reply v1.GetDingTalkLoginConfigResponse
			if err := protojson.Unmarshal(payload, &reply); err != nil {
				t.Fatal(err)
			}
			if reply.Success != item.success || int(reply.Code) != item.code || reply.GetData().GetEnabled() || reply.GetData().GetAuthorizeUrl() != "" || strings.Contains(string(payload), "敏感错误") {
				t.Fatalf("错误的禁用/失败信封: %s", payload)
			}
			if !item.success {
				var fields map[string]json.RawMessage
				if err := json.Unmarshal(payload, &fields); err != nil {
					t.Fatal(err)
				}
				if string(fields["success"]) != "false" || reply.Message != "钉钉登录配置加载失败，请稍后重试" {
					t.Fatalf("失败未显式呈现: %s", payload)
				}
			}
		})
	}
}
