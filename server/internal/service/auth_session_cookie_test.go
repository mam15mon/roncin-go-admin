package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestSessionCookieRetentionAndLogout(t *testing.T) {
	s := NewAuthService(nil, &biz.SessionPolicy{CookieName: "roncin_session", Secure: true, SameSite: "lax"}, nil)
	for _, tc := range []struct {
		name, value string
		maxAge      int
	}{{"登录", "session-token", 0}, {"退出", "", -1}} {
		t.Run(tc.name, func(t *testing.T) {
			server := kratoshttp.NewServer()
			server.Route("/").GET("cookie", func(ctx kratoshttp.Context) error {
				s.setCookie(ctx, tc.value, time.Unix(1, 0), tc.maxAge)
				return ctx.String(200, "ok")
			})
			response := httptest.NewRecorder()
			server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/cookie", nil))
			cookies := response.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("Cookie数量=%d", len(cookies))
			}
			c := cookies[0]
			if c.Value != tc.value || !c.HttpOnly || !c.Secure || c.Path != "/" || c.SameSite != http.SameSiteLaxMode {
				t.Fatal("会话Cookie属性错误")
			}
			if tc.value != "" {
				if c.MaxAge != int(sessionCookieRetention/time.Second) || time.Until(c.Expires) < sessionCookieRetention-time.Minute {
					t.Fatal("浏览器未持久保留会话载体")
				}
			} else if c.MaxAge != -1 || !c.Expires.Before(time.Now()) {
				t.Fatal("退出未清除Cookie")
			}
		})
	}
}
