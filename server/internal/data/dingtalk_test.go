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
	provider := &dingTalkIdentityProvider{enabled: true, clientID: "client-id", redirectURI: "https://admin.example.com/user/login/dingtalk/callback", corpID: "ding-corp"}
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
	if parsed.Query().Get("scope") != "openid corpid" {
		t.Fatalf("授权地址未申请企业身份范围: %s", authorizeURL)
	}
	if parsed.Query().Get("corpId") != "ding-corp" {
		t.Fatalf("授权地址未限定目标企业: %s", authorizeURL)
	}
}

func TestDingTalkResolveIdentity(t *testing.T) {
	provider := &dingTalkIdentityProvider{
		enabled:      true,
		clientID:     "client-id",
		clientSecret: "client-secret",
		corpID:       "ding-corp",
		client: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			switch {
			case request.URL.String() == dingTalkTokenURL:
				return jsonResponse(http.StatusOK, `{"accessToken":"access-token","corpId":"ding-corp"}`), nil
			case request.URL.String() == dingTalkProfileURL:
				if request.Header.Get("x-acs-dingtalk-access-token") != "access-token" {
					t.Fatal("成员接口缺少钉钉访问令牌")
				}
				return jsonResponse(http.StatusOK, `{"unionId":"union-id","nick":"张三","email":"zhangsan@example.com","avatarUrl":"https://example.com/avatar.png"}`), nil
			case request.URL.String() == dingTalkAppTokenURL:
				return jsonResponse(http.StatusOK, `{"accessToken":"app-access-token","expireIn":7200}`), nil
			case request.URL.Path == "/topapi/user/getbyunionid":
				if request.URL.Query().Get("access_token") != "app-access-token" {
					t.Fatal("企业成员转换接口缺少应用访问令牌")
				}
				return jsonResponse(http.StatusOK, `{"errcode":0,"errmsg":"ok","result":{"userid":"ding-user-id"}}`), nil
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
	if identity.UnionID != "union-id" || identity.UserID != "ding-user-id" || identity.CorpID != "ding-corp" || identity.Name != "张三" || identity.Email == nil || *identity.Email != "zhangsan@example.com" || identity.AvatarURL == nil || *identity.AvatarURL != "https://example.com/avatar.png" {
		t.Fatalf("钉钉身份解析错误: %#v", identity)
	}
}

func TestDingTalkSendTextUsesEnterpriseRobot(t *testing.T) {
	provider := &dingTalkIdentityProvider{
		enabled:      true,
		clientID:     "client-id",
		clientSecret: "client-secret",
		client: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			switch request.URL.String() {
			case dingTalkAppTokenURL:
				return jsonResponse(http.StatusOK, `{"accessToken":"app-access-token","expireIn":7200}`), nil
			case dingTalkRobotURL:
				if request.Header.Get("x-acs-dingtalk-access-token") != "app-access-token" {
					t.Fatal("机器人接口缺少企业访问令牌")
				}
				payload, err := io.ReadAll(request.Body)
				if err != nil {
					t.Fatalf("读取机器人请求失败: %v", err)
				}
				body := string(payload)
				for _, expected := range []string{`"robotCode":"client-id"`, `"userIds":["ding-user-id"]`, `"msgKey":"sampleText"`, `测试通知`} {
					if !strings.Contains(body, expected) {
						t.Fatalf("机器人请求缺少 %s: %s", expected, body)
					}
				}
				return jsonResponse(http.StatusOK, `{"processQueryKey":"query-key"}`), nil
			default:
				t.Fatalf("非预期钉钉请求: %s", request.URL)
				return nil, nil
			}
		})},
	}
	if err := provider.SendText(context.Background(), "ding-user-id", "测试通知"); err != nil {
		t.Fatalf("SendText() error = %v", err)
	}
}

func TestDingTalkResolveIdentityRejectsOtherOrganization(t *testing.T) {
	provider := &dingTalkIdentityProvider{
		enabled:      true,
		clientID:     "client-id",
		clientSecret: "client-secret",
		corpID:       "ding-corp",
		client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"accessToken":"access-token","corpId":"other-corp"}`), nil
		})},
	}
	if _, err := provider.ResolveIdentity(context.Background(), "auth-code"); err != biz.ErrDingTalkOrganizationMismatch {
		t.Fatalf("ResolveIdentity() error = %v, want ErrDingTalkOrganizationMismatch", err)
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
