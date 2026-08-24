package data

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestDingTalkAuthorizeURL(t *testing.T) {
	provider := &dingTalkIdentityProvider{enabled: true, clientID: "client-id", redirectURI: "https://admin.example.com/user/login/dingtalk/callback"}
	authorizeURL, err := provider.AuthorizeURL("state-value")
	if err != nil {
		t.Fatalf("AuthorizeURL() error = %v", err)
	}
	parsed, err := url.Parse(authorizeURL)
	if err != nil {
		t.Fatalf("解析授权地址失败: %v", err)
	}
	if parsed.Host != "login.dingtalk.com" || parsed.Query().Get("client_id") != "client-id" || parsed.Query().Get("state") != "state-value" {
		t.Fatalf("授权地址参数错误: %s", authorizeURL)
	}
}

func TestDingTalkResolveIdentity(t *testing.T) {
	provider := &dingTalkIdentityProvider{
		enabled:      true,
		clientID:     "client-id",
		clientSecret: "client-secret",
		client: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			switch request.URL.String() {
			case dingTalkTokenURL:
				return jsonResponse(http.StatusOK, `{"accessToken":"access-token"}`), nil
			case dingTalkProfileURL:
				if request.Header.Get("x-acs-dingtalk-access-token") != "access-token" {
					t.Fatal("成员接口缺少钉钉访问令牌")
				}
				return jsonResponse(http.StatusOK, `{"unionId":"union-id","nick":"张三","email":"zhangsan@example.com","avatarUrl":"https://example.com/avatar.png"}`), nil
			default:
				t.Fatalf("非预期钉钉请求: %s", request.URL)
				return nil, nil
			}
		})},
	}

	identity, err := provider.ResolveIdentity(context.Background(), "auth-code")
	if err != nil {
		t.Fatalf("ResolveIdentity() error = %v", err)
	}
	if identity.UnionID != "union-id" || identity.Name != "张三" || identity.Email == nil || *identity.Email != "zhangsan@example.com" || identity.AvatarURL == nil || *identity.AvatarURL != "https://example.com/avatar.png" {
		t.Fatalf("钉钉身份解析错误: %#v", identity)
	}
}

func TestDingTalkResolveIdentityMapsInvalidCode(t *testing.T) {
	provider := &dingTalkIdentityProvider{
		enabled: true,
		client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusBadRequest, `{"code":"invalidCode"}`), nil
		})},
	}
	if _, err := provider.ResolveIdentity(context.Background(), "expired-code"); err != biz.ErrDingTalkCodeInvalid {
		t.Fatalf("ResolveIdentity() error = %v, want ErrDingTalkCodeInvalid", err)
	}
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
