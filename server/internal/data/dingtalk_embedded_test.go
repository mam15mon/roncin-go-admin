package data

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestDingTalkResolveEmbeddedIdentityUsesEnterpriseProtocol(t *testing.T) {
	tokenCalls, identityCalls := 0, 0
	provider := &dingTalkIdentityProvider{enabled: true, clientID: "app", clientSecret: "secret", corpID: "fixed-corp"}
	provider.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost || request.Header.Get("Content-Type") != "application/json" {
			t.Fatal("免登协议必须使用 JSON POST")
		}
		var payload map[string]string
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		switch request.URL.Path {
		case "/v1.0/oauth2/accessToken":
			tokenCalls++
			if len(payload) != 2 || payload["appKey"] != "app" || payload["appSecret"] != "secret" {
				t.Fatal("企业应用凭据载荷错误")
			}
			return jsonResponse(200, `{"accessToken":"enterprise-token","expireIn":7200}`), nil
		case "/topapi/v2/user/getuserinfo":
			identityCalls++
			if request.URL.Host != "oapi.dingtalk.com" || request.URL.Query().Get("access_token") != "enterprise-token" || len(payload) != 1 || payload["code"] != "temporary-code" || request.Header.Get("x-acs-dingtalk-access-token") != "" {
				t.Fatal("免登换码端点、企业令牌或 code 载荷错误")
			}
			return jsonResponse(200, `{"errcode":0,"result":{"unionid":" union ","userid":" user ","name":" 员工 "}}`), nil
		default:
			t.Fatal("客户端免登不得调用 OAuth 或详情补读接口")
			return nil, nil
		}
	})}
	for range 2 {
		identity, err := provider.ResolveEmbeddedIdentity(context.Background(), "temporary-code")
		if err != nil || identity.UnionID != "union" || identity.UserID != "user" || identity.Name != "员工" || identity.CorpID != "fixed-corp" {
			t.Fatalf("身份解析错误: %+v %v", identity, err)
		}
	}
	if tokenCalls != 1 || identityCalls != 2 {
		t.Fatalf("未复用企业 token: %d/%d", tokenCalls, identityCalls)
	}
}

func TestDingTalkResolveEmbeddedIdentityRejectsUpstreamFailures(t *testing.T) {
	for _, item := range []struct {
		name, body string
		status     int
		network    bool
		want       error
	}{
		{"失效码", `{"errcode":40029}`, 200, false, biz.ErrDingTalkCodeInvalid},
		{"业务错误", `{"errcode":99999,"errmsg":"secret"}`, 200, false, biz.ErrDingTalkLoginFailed},
		{"权限错误", "secret", 403, false, biz.ErrDingTalkPermissionDenied},
		{"HTTP失效码", "secret", 400, false, biz.ErrDingTalkCodeInvalid},
		{"服务器错误", "secret", 500, false, biz.ErrDingTalkLoginFailed},
		{"畸形JSON", "{secret", 200, false, biz.ErrDingTalkLoginFailed},
		{"空结果", `{"errcode":0}`, 200, false, biz.ErrDingTalkLoginFailed},
		{"缺错误码", `{"result":{"unionid":"u","userid":"i","name":"n"}}`, 200, false, biz.ErrDingTalkLoginFailed},
		{"缺unionid", `{"errcode":0,"result":{"userid":"i","name":"n"}}`, 200, false, biz.ErrDingTalkLoginFailed},
		{"缺userid", `{"errcode":0,"result":{"unionid":"u","name":"n"}}`, 200, false, biz.ErrDingTalkLoginFailed},
		{"空姓名", `{"errcode":0,"result":{"unionid":"u","userid":"i","name":" "}}`, 200, false, biz.ErrDingTalkLoginFailed},
		{"网络错误", "", 0, true, biz.ErrDingTalkLoginFailed},
	} {
		t.Run(item.name, func(t *testing.T) {
			provider := &dingTalkIdentityProvider{enabled: true, appToken: "secret", appTokenTill: time.Now().Add(time.Hour), client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				if item.network {
					return nil, errors.New("secret")
				}
				return jsonResponse(item.status, item.body), nil
			})}}
			identity, err := provider.ResolveEmbeddedIdentity(context.Background(), "code")
			if identity != nil || !errors.Is(err, item.want) || strings.Contains(err.Error(), "secret") {
				t.Fatalf("上游错误未安全映射: %+v %v", identity, err)
			}
		})
	}
}

func TestDingTalkResolveEmbeddedIdentityDisabledAndTokenFailure(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		calls := 0
		provider := &dingTalkIdentityProvider{enabled: enabled, client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("secret") })}}
		_, err := provider.ResolveEmbeddedIdentity(context.Background(), "code")
		want := biz.ErrDingTalkDisabled
		if enabled {
			want = biz.ErrDingTalkPermissionDenied
		}
		if err != want || (!enabled && calls != 0) || (enabled && calls != 1) {
			t.Fatalf("禁用/token 错误未拒绝: %v calls=%d", err, calls)
		}
	}
}
