package service

import (
	"context"
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
)

type passkeyTransportRepo struct {
	biz.PasskeyRepo
	registeredUser uuid.UUID
	discarded      string
}

func (*passkeyTransportRepo) Settings() biz.PasskeySettings {
	return biz.PasskeySettings{Enabled: true, RPID: "roncin.com.cn", RPName: "Roncin", Origins: []string{"https://ops.roncin.com.cn"}}
}
func (*passkeyTransportRepo) BeginLogin(context.Context, string) (*biz.PasskeyCeremony, error) {
	return &biz.PasskeyCeremony{ID: "login-token", OptionsJSON: `{"challenge":"opaque-options","rpId":"roncin.com.cn"}`, ExpiresAt: time.Now().Add(5 * time.Minute)}, nil
}
func (r *passkeyTransportRepo) BeginRegistration(_ context.Context, userID uuid.UUID, _ string) (*biz.PasskeyCeremony, error) {
	r.registeredUser = userID
	return &biz.PasskeyCeremony{ID: "registration-token", OptionsJSON: `{"challenge":"opaque-registration"}`, ExpiresAt: time.Now().Add(5 * time.Minute)}, nil
}
func (*passkeyTransportRepo) FinishLogin(context.Context, string, string) (*biz.Credential, error) {
	return nil, biz.ErrPasskeyInvalid
}
func (*passkeyTransportRepo) FinishRegistration(context.Context, uuid.UUID, string, string, string, *biz.AuditEvent) (*biz.PasskeyCredentialRecord, error) {
	return nil, biz.ErrPasskeyInvalid
}
func (r *passkeyTransportRepo) DiscardCeremony(id string) { r.discarded = id }

type passkeyTransportTx struct{}

func (passkeyTransportTx) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestPasskeyHTTPOriginAndPurposeBoundCookies(t *testing.T) {
	repo := &passkeyTransportRepo{}
	principal := &biz.Principal{UserID: uuid.New()}
	policy := &biz.SessionPolicy{CookieName: "roncin_session", Secure: true, SameSite: "strict"}
	service := NewPasskeyService(biz.NewPasskeyUsecase(repo, nil, passkeyTransportTx{}), NewAuthService(nil, policy, nil))
	server := kratoshttp.NewServer(kratoshttp.RequestDecoder(authCookieProtoDecoder), kratoshttp.Middleware(func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, request any) (any, error) {
			return next(biz.WithPrincipal(ctx, principal), request)
		}
	}))
	v1.RegisterPasskeyServiceHTTPServer(server, service)
	request := func(method, path, body, origin, cookie string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "https://ops.roncin.com.cn"+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", origin)
		req.Header.Set("Cookie", cookie)
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, req)
		return recorder
	}
	for _, purpose := range []string{"login", "registration"} {
		started := request(http.MethodPost, "/api/v1/auth/passkey/"+purpose+"/begin", `{}`, "https://ops.roncin.com.cn", "")
		cookies := started.Result().Cookies()
		if started.Code != 200 || len(cookies) != 1 || started.Header().Get("Cache-Control") != "no-store" || started.Header().Get("Pragma") != "no-cache" {
			t.Fatal("公开仪式配置未禁用缓存或错误写入会话Cookie")
		}
		c := cookies[0]
		if c.Name != "roncin_session_passkey_"+purpose || c.Value != purpose+"-token" || !c.HttpOnly || !c.Secure || c.Path != "/" || c.SameSite != http.SameSiteStrictMode || c.MaxAge < 295 || c.MaxAge > 300 || time.Until(c.Expires) < 4*time.Minute {
			t.Fatalf("仪式Cookie属性错误: %+v", c)
		}
		body := `{"credentialJson":"{}"}`
		if purpose == "registration" {
			body = `{"credentialJson":"{}","name":"设备"}`
		}
		finished := request(http.MethodPost, "/api/v1/auth/passkey/"+purpose+"/finish", body, "https://ops.roncin.com.cn", c.Name+"="+c.Value)
		cleared := finished.Result().Cookies()
		if finished.Code != 401 || len(cleared) != 1 || cleared[0].Name != c.Name || cleared[0].Value != "" || cleared[0].MaxAge != -1 || !cleared[0].HttpOnly || !cleared[0].Secure || cleared[0].SameSite != http.SameSiteStrictMode || !cleared[0].Expires.Before(time.Now()) || repo.discarded != c.Value {
			t.Fatal("失败finish未消费并清理对应Cookie，或错误发放会话")
		}
	}
	if repo.registeredUser != principal.UserID {
		t.Fatal("注册没有绑定实际登录用户")
	}
	denied := request(http.MethodPost, "/api/v1/auth/passkey/login/begin", `{}`, "https://evil.example.com", "")
	if denied.Code != 403 || len(denied.Result().Cookies()) != 0 {
		t.Fatal("不受信任来源可创建仪式")
	}
	req := httptest.NewRequest(http.MethodGet, "http://192.0.2.10/api/v1/auth/passkey/config", nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, req)
	var config v1.GetPasskeyConfigResponse
	if err := protojson.Unmarshal(recorder.Body.Bytes(), &config); err != nil || recorder.Code != 200 || config.GetData().GetEnabled() || config.GetData().GetRpId() != "roncin.com.cn" {
		t.Fatal("HTTP IP未明确返回功能不可用")
	}
}
