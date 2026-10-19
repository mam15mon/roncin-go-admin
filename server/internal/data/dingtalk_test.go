package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

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

func TestDingTalkSendMessageUsesEnterpriseRobot(t *testing.T) {
	const orderPath = "/orders/sea-export/019924d6-75a0-7000-8000-000000000001/fees"
	for _, withLink := range []bool{false, true} {
		t.Run(fmt.Sprint(withLink), func(t *testing.T) {
			message := biz.DingTalkMessage{Title: "测试通知", Text: "### 测试通知\n\n**订单：** 中文 \"引号\" \\斜杠 & <普通文本>"}
			wantParameter := map[string]string{"title": message.Title, "text": message.Text}
			wantKey := "sampleMarkdown"
			if withLink {
				message.ActionTitle, message.ActionPath = "查看订单费用", orderPath
				wantKey = "sampleActionCard"
				wantParameter["singleTitle"], wantParameter["singleURL"] = "查看订单费用", "https://admin.example.com"+orderPath
			}
			calls := 0
			provider := &dingTalkIdentityProvider{
				enabled: true, clientID: "client-id", clientSecret: "client-secret",
				redirectURI: "https://admin.example.com/user/login/dingtalk/callback?state=ignored",
				client: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
					switch request.URL.String() {
					case dingTalkAppTokenURL:
						return jsonResponse(http.StatusOK, `{"accessToken":"app-access-token","expireIn":7200}`), nil
					case dingTalkRobotURL:
						calls++
						if request.Method != http.MethodPost || request.Header.Get("x-acs-dingtalk-access-token") != "app-access-token" || request.Header.Get("Content-Type") != "application/json" {
							t.Fatal("机器人接口请求方式或请求头错误")
						}
						var payload struct {
							RobotCode string   `json:"robotCode"`
							UserIDs   []string `json:"userIds"`
							MsgKey    string   `json:"msgKey"`
							MsgParam  string   `json:"msgParam"`
						}
						if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
							t.Fatalf("机器人请求或 msgParam 字符串类型错误: %v", err)
						}
						if payload.RobotCode != "client-id" || !reflect.DeepEqual(payload.UserIDs, []string{"ding-user-id"}) || payload.MsgKey != wantKey {
							t.Fatalf("机器人请求错误: %+v", payload)
						}
						var parameter map[string]string
						if err := json.Unmarshal([]byte(payload.MsgParam), &parameter); err != nil {
							t.Fatal(err)
						}
						if !reflect.DeepEqual(parameter, wantParameter) {
							t.Fatalf("标题、正文或入口错误: %+v", parameter)
						}
						return jsonResponse(http.StatusOK, `{"processQueryKey":"query-key"}`), nil
					default:
						t.Fatalf("非预期钉钉请求: %s", request.URL)
						return nil, nil
					}
				})},
			}
			if err := provider.SendMessage(context.Background(), " ding-user-id ", message); err != nil {
				t.Fatalf("SendMessage() error = %v", err)
			}
			if calls != 1 {
				t.Fatalf("发送次数 = %d", calls)
			}
		})
	}
}

func TestDingTalkSendMessageRejectsInvalidInput(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		enabled bool
		userID  string
		message biz.DingTalkMessage
	}{
		{name: "禁用", userID: "user", message: biz.DingTalkMessage{Title: "标题", Text: "正文"}},
		{name: "无收件人", enabled: true, userID: " ", message: biz.DingTalkMessage{Title: "标题", Text: "正文"}},
		{name: "无标题", enabled: true, userID: "user", message: biz.DingTalkMessage{Title: " ", Text: "正文"}},
		{name: "无正文", enabled: true, userID: "user", message: biz.DingTalkMessage{Title: "标题", Text: " "}},
		{name: "入口无站点", enabled: true, userID: "user", message: biz.DingTalkMessage{Title: "标题", Text: "正文", ActionTitle: "查看订单费用", ActionPath: "/orders/sea-export/00000000-0000-4000-8000-000000000001/fees"}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			provider := &dingTalkIdentityProvider{enabled: scenario.enabled, client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				t.Fatal("无效输入不应请求钉钉")
				return nil, nil
			})}}
			if err := provider.SendMessage(context.Background(), scenario.userID, scenario.message); err == nil {
				t.Fatal("无效输入应拒绝")
			} else if !scenario.enabled && !errors.Is(err, biz.ErrDingTalkDisabled) {
				t.Fatalf("禁用错误 = %v", err)
			}
		})
	}
}

func TestDingTalkSendMessagePropagatesFailures(t *testing.T) {
	for _, scenario := range []struct {
		name         string
		status       int
		body         string
		transportErr error
	}{
		{name: "拒绝", status: http.StatusForbidden, body: `{"code":"Forbidden","message":"敏感响应不应外传"}`},
		{name: "缺受理键", status: http.StatusOK, body: `{}`},
		{name: "响应不合法", status: http.StatusOK, body: `invalid`},
		{name: "超时", transportErr: context.DeadlineExceeded},
		{name: "有受理键但收件人无效", status: http.StatusOK, body: `{"processQueryKey":"accepted","invalidStaffIdList":["user"]}`},
		{name: "有受理键但收件人限流", status: http.StatusOK, body: `{"processQueryKey":"accepted","flowControlledStaffIdList":["user"]}`},
	} {
		for _, withAction := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/action=%t", scenario.name, withAction), func(t *testing.T) {
				calls := 0
				provider := &dingTalkIdentityProvider{enabled: true, redirectURI: "https://admin.example.com/callback", appToken: "cached-token", appTokenTill: time.Now().Add(time.Minute), client: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
					calls++
					if request.URL.String() != dingTalkRobotURL {
						t.Fatal("缓存有效期内不应重新取令牌")
					}
					if scenario.transportErr != nil {
						return nil, scenario.transportErr
					}
					return jsonResponse(scenario.status, scenario.body), nil
				})}}
				message := biz.DingTalkMessage{Title: "验证码", Text: "**012345**"}
				if withAction {
					message.ActionTitle, message.ActionPath = "打开系统首页", "/welcome"
				}
				err := provider.SendMessage(context.Background(), "user", message)
				if err == nil || calls != 1 || strings.Contains(err.Error(), "敏感响应") || strings.Contains(err.Error(), "012345") {
					t.Fatalf("失败处理错误: calls=%d error=%v", calls, err)
				}
			})
		}
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
