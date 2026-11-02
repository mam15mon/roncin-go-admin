package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/middleware"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func assertSessionCookiePair(t *testing.T, cookies []*http.Cookie, token string, remember, cleared bool) {
	t.Helper()
	if len(cookies) != 2 {
		t.Fatalf("会话及偏好Cookie数量=%d", len(cookies))
	}
	byName := make(map[string]*http.Cookie)
	for _, c := range cookies {
		if _, duplicate := byName[c.Name]; duplicate {
			t.Fatal("响应重复写入同名Cookie")
		}
		byName[c.Name] = c
	}
	preference := "false"
	if remember {
		preference = "true"
	}
	if cleared {
		preference = ""
	}
	for name, want := range map[string]string{"roncin_session": token, "roncin_session_remember": preference} {
		c := byName[name]
		if c == nil || c.Value != want || !c.HttpOnly || !c.Secure || c.Path != "/" || c.SameSite != http.SameSiteLaxMode {
			t.Fatalf("Cookie %s属性错误: %+v", name, c)
		}
		switch {
		case cleared:
			if c.MaxAge != -1 || !c.Expires.Before(time.Now()) || c.Expires.IsZero() {
				t.Fatal("未明确清除Cookie")
			}
		case remember:
			if c.MaxAge != int(sessionCookieRetention/time.Second) || time.Until(c.Expires) < sessionCookieRetention-time.Minute || time.Until(c.Expires) > sessionCookieRetention {
				t.Fatal("持久Cookie保留期错误")
			}
		default:
			if c.MaxAge != 0 || !c.Expires.IsZero() || strings.Contains(c.String(), "Expires=") || strings.Contains(c.String(), "Max-Age=") {
				t.Fatal("未勾选自动登录仍写入持久Cookie")
			}
		}
	}
}

func TestSessionCookieRetentionAndLogout(t *testing.T) {
	s := NewAuthService(nil, &biz.SessionPolicy{CookieName: "roncin_session", Secure: true, SameSite: "lax"}, nil)
	for _, tc := range []struct {
		name            string
		remember, clear bool
	}{{"自动登录", true, false}, {"当前会话", false, false}, {"退出", false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			server := kratoshttp.NewServer()
			server.Route("/").GET("cookie", func(ctx kratoshttp.Context) error {
				if tc.clear {
					s.clearSessionCookies(ctx)
				} else {
					s.setSessionCookies(ctx, "session-token", tc.remember)
				}
				return ctx.String(200, "ok")
			})
			response := httptest.NewRecorder()
			server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/cookie", nil))
			token := "session-token"
			if tc.clear {
				token = ""
			}
			assertSessionCookiePair(t, response.Result().Cookies(), token, tc.remember, tc.clear)
		})
	}
}

func TestLoginRememberMeAndOrganizationSwitchKeepCookieMode(t *testing.T) {
	hash := authServicePasswordHash(t)
	for _, tc := range []struct {
		name     string
		remember *bool
		want     bool
	}{{"缺省自动登录", nil, true}, {"显式自动登录", boolPointer(true), true}, {"取消自动登录", boolPointer(false), false}} {
		t.Run(tc.name, func(t *testing.T) {
			userID, sourceID, targetID := uuid.New(), uuid.New(), uuid.New()
			principal := &biz.Principal{UserID: userID, Organization: biz.Organization{ID: sourceID}, SessionTokenHash: "previous"}
			repo := &authServiceRepoStub{credential: &biz.Credential{UserID: userID, Enabled: true, PasswordHash: hash}, choices: []biz.OrganizationChoice{{OrganizationID: sourceID}, {OrganizationID: targetID}}}
			policy := &biz.SessionPolicy{CookieName: "roncin_session", TTL: 30 * 24 * time.Hour, Secure: true, SameSite: "lax"}
			s := NewAuthService(biz.NewAuthUsecase(repo, policy, nil, nil), policy, nil)
			server := kratoshttp.NewServer(kratoshttp.RequestDecoder(authCookieProtoDecoder), kratoshttp.Middleware(func(next middleware.Handler) middleware.Handler {
				return func(ctx context.Context, request any) (any, error) {
					return next(biz.WithPrincipal(ctx, principal), request)
				}
			}))
			v1.RegisterAuthServiceHTTPServer(server, s)
			payload, _ := json.Marshal(&v1.LoginRequest{Username: "admin", Password: "correct-password", RememberMe: tc.remember})
			login := postAuthCookieRequest(server, "/api/v1/auth/login", string(payload), "")
			if login.Code != 200 || len(repo.createdSessions) != 1 {
				t.Fatalf("登录失败: %d %s", login.Code, login.Body.String())
			}
			token := ""
			cookieHeader := []string{}
			for _, c := range login.Result().Cookies() {
				cookieHeader = append(cookieHeader, c.Name+"="+c.Value)
				if c.Name == policy.CookieName {
					token = c.Value
				}
			}
			if token == "" {
				t.Fatal("登录未发会话凭证")
			}
			assertSessionCookiePair(t, login.Result().Cookies(), token, tc.want, false)
			if remaining := time.Until(repo.createdSessions[0].ExpiresAt); remaining < policy.TTL-time.Minute || remaining > policy.TTL {
				t.Fatal("浏览器偏好修改了服务端30天期限")
			}
			switched := postAuthCookieRequest(server, "/api/v1/auth/switch-organization", fmt.Sprintf(`{"organizationId":%q}`, targetID.String()), strings.Join(cookieHeader, "; "))
			if switched.Code != 200 || len(repo.rotatedSessions) != 1 {
				t.Fatalf("切换失败: %d %s", switched.Code, switched.Body.String())
			}
			nextToken := ""
			for _, c := range switched.Result().Cookies() {
				if c.Name == policy.CookieName {
					nextToken = c.Value
				}
			}
			if nextToken == "" || nextToken == token {
				t.Fatal("切换没有轮转令牌")
			}
			assertSessionCookiePair(t, switched.Result().Cookies(), nextToken, tc.want, false)
			loggedOut := postAuthCookieRequest(server, "/api/v1/auth/logout", `{}`, strings.Join(cookieHeader, "; "))
			if loggedOut.Code != 200 {
				t.Fatal("退出失败")
			}
			assertSessionCookiePair(t, loggedOut.Result().Cookies(), "", false, true)
		})
	}
}

func TestRememberCookieInvalidOrMissingStaysSessionOnly(t *testing.T) {
	s := NewAuthService(nil, &biz.SessionPolicy{CookieName: "roncin_session", Secure: true, SameSite: "lax"}, nil)
	for _, cookie := range []string{"", "roncin_session_remember=false", "roncin_session_remember=garbage", "roncin_session_remember=TRUE", "roncin_session_remember=true; roncin_session_remember=false"} {
		server := kratoshttp.NewServer()
		server.Route("/").GET("switch", func(ctx kratoshttp.Context) error {
			s.setSessionCookies(ctx, "next-token", s.rememberSessionFromRequest(ctx))
			return ctx.String(200, "ok")
		})
		request := httptest.NewRequest(http.MethodGet, "/switch", nil)
		request.Header.Set("Cookie", cookie)
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, request)
		assertSessionCookiePair(t, recorder.Result().Cookies(), "next-token", false, false)
	}
}

func postAuthCookieRequest(server *kratoshttp.Server, path, body, cookie string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Cookie", cookie)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

// service 包测试不运行 server 的全局 JSON 编解码器注册，显式沿用真实入口的 ProtoJSON 请求契约。
func authCookieProtoDecoder(request *http.Request, value any) error {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return err
	}
	message, ok := value.(proto.Message)
	if !ok {
		return fmt.Errorf("测试请求不是Protobuf消息")
	}
	return (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(body, message)
}
