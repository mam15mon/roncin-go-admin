package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func qrTestProvider(transport http.RoundTripper) *dingTalkIdentityProvider {
	return &dingTalkIdentityProvider{enabled: true, clientID: "qr-app", corpID: "qr-corp", redirectURI: "https://admin.example.com/user/login/dingtalk/callback", client: &http.Client{Transport: transport}}
}

func qrTestResponse(value any) *http.Response {
	body, _ := json.Marshal(value)
	return jsonResponse(http.StatusOK, string(body))
}

func TestDingTalkQRCodeProtocolAndAttemptIsolation(t *testing.T) {
	var pollCount atomic.Int32
	provider := qrTestProvider(roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Scheme != "https" || request.URL.Host != "login.dingtalk.com" {
			t.Error("访问了非固定上游")
		}
		if request.Method == http.MethodGet {
			if request.URL.Path != "/oauth2/challenge.htm" {
				t.Error("未走官方初始化页面")
			}
			response := jsonResponse(http.StatusOK, "<html></html>")
			response.Header.Add("Set-Cookie", "qr_session="+request.URL.Query().Get("state")+"; Secure; HttpOnly; Path=/")
			return response, nil
		}
		if err := request.ParseForm(); err != nil {
			t.Fatal(err)
		}
		form := request.PostForm
		for key, want := range map[string]string{"iframe": "true", "client_id": "qr-app", "corpId": "qr-corp", "scope": "openid corpid", "response_type": "code", "redirect_uri": "https://admin.example.com/user/login/dingtalk/callback", "prompt": "consent", "pdmTitle": "Linux Web", "pdmModel": "Unknown", "pdmToken": "FAKE_UMID"} {
			if form.Get(key) != want {
				t.Errorf("上游表单 %s=%q，期望%q", key, form.Get(key), want)
			}
		}
		cookie, err := request.Cookie("qr_session")
		if err != nil || cookie.Value != form.Get("state") {
			t.Error("不同扫码尝试串用Cookie")
		}
		if request.Header.Get("sec_umidToken") != "FAKE_UMID" || request.Header.Get("Content-Type") != "application/x-www-form-urlencoded" || request.Header.Get("Origin") != dingTalkLoginOrigin {
			t.Error("上游请求头错误")
		}
		switch request.URL.Path {
		case "/oauth2/generate_qrcode":
			return qrTestResponse(map[string]any{"success": true, "result": dingTalkLoginOrigin + "/oauth2/qr_confirm.htm?code=qr-" + form.Get("state")}), nil
		case "/oauth2/login_with_qr":
			pollCount.Add(1)
			if form.Get("code") != "qr-"+form.Get("state") || form.Get("stayLogin") != "false" || !form.Has("exclusiveCorpId") {
				t.Error("轮询标识未绑定原尝试")
			}
			return qrTestResponse(map[string]any{"success": false, "errorCode": "11021"}), nil
		default:
			t.Errorf("未知上游路径%s", request.URL.Path)
			return nil, errors.New("未知路径")
		}
	}))
	for _, state := range []string{"browser-a", "browser-b"} {
		qr, err := provider.CreateQRCode(t.Context(), state, time.Now().Add(5*time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		parsed, _ := url.Parse(qr.URL)
		if parsed.Query().Get("dd_mini_app_id") != "5000000004998565" || parsed.Query().Get("showmenu") != "false" {
			t.Fatal("未沿用官方真实二维码参数")
		}
	}
	for _, state := range []string{"browser-a", "browser-b", "browser-a"} {
		result, err := provider.PollQRCode(t.Context(), state)
		if err != nil || result.Status != biz.DingTalkQRCodeWaiting {
			t.Fatalf("等待结果错误: %+v %v", result, err)
		}
	}
	if pollCount.Load() != 3 {
		t.Fatal("轮询请求丢失")
	}
}

func TestDingTalkQRCodeStatusesAndOneTimeConfirmation(t *testing.T) {
	for _, tc := range []struct {
		name, code string
		want       biz.DingTalkQRCodeStatus
		failure    bool
	}{
		{"等待", "11021", biz.DingTalkQRCodeWaiting, false},
		{"已扫", "11041", biz.DingTalkQRCodeScanned, false},
		{"到期", "11019", biz.DingTalkQRCodeExpired, false},
		{"风控停止", "283960126", "", true},
		{"未知失败", "other", "", true},
		{"确认", "", biz.DingTalkQRCodeConfirmed, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := qrTestProvider(roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.Method == http.MethodGet {
					return jsonResponse(http.StatusOK, "ok"), nil
				}
				if request.URL.Path == "/oauth2/generate_qrcode" {
					return qrTestResponse(map[string]any{"success": true, "result": dingTalkLoginOrigin + "/oauth2/qr_confirm.htm?code=generated"}), nil
				}
				if tc.code != "" {
					return qrTestResponse(map[string]any{"success": false, "errorCode": tc.code}), nil
				}
				return qrTestResponse(map[string]any{"success": true, "result": map[string]string{"url": "https://admin.example.com/user/login/dingtalk/callback?state=current&authCode=oauth-code"}}), nil
			}))
			if _, err := provider.CreateQRCode(t.Context(), "current", time.Now().Add(time.Minute)); err != nil {
				t.Fatal(err)
			}
			result, err := provider.PollQRCode(t.Context(), "current")
			if tc.failure {
				if err != biz.ErrDingTalkLoginFailed {
					t.Fatalf("错误状态未停止: %v", err)
				}
				return
			}
			if err != nil || result.Status != tc.want {
				t.Fatalf("状态错误: %+v %v", result, err)
			}
			if tc.want == biz.DingTalkQRCodeConfirmed && result.AuthCode != "oauth-code" {
				t.Fatal("未提取真正OAuth授权码")
			}
			if tc.want == biz.DingTalkQRCodeConfirmed || tc.want == biz.DingTalkQRCodeExpired {
				next, err := provider.PollQRCode(t.Context(), "current")
				if err != nil || next.Status != biz.DingTalkQRCodeExpired || next.AuthCode != "" {
					t.Fatal("终态尝试被再次消费")
				}
			}
		})
	}
}

func TestDingTalkQRCodeCallbackRejectsUnboundTargets(t *testing.T) {
	provider := qrTestProvider(nil)
	for _, raw := range []string{
		"https://other.example.com/user/login/dingtalk/callback?state=current&authCode=code",
		"https://admin.example.com/other?state=current&authCode=code",
		"http://admin.example.com/user/login/dingtalk/callback?state=current&authCode=code",
		"https://user@admin.example.com/user/login/dingtalk/callback?state=current&authCode=code",
		"https://admin.example.com/user/login/dingtalk/callback?state=old&authCode=code",
		"https://admin.example.com/user/login/dingtalk/callback?state=current&state=current&authCode=code",
		"https://admin.example.com/user/login/dingtalk/callback?state=current&state=%ZZ&authCode=code",
		"https://admin.example.com/user/login/dingtalk/callback?state=current&authCode=code&authCode=%ZZ",
		"https://admin.example.com/user/login/dingtalk/callback?state=current&authCode=code&authCode=other",
		"https://admin.example.com/user/login/dingtalk/callback?state=current&authCode=code#fragment",
		"https://admin.example.com/user/login/dingtalk/callback?state=current&authCode=",
	} {
		if _, err := provider.validatedQRAuthCode(raw, "current"); err != biz.ErrDingTalkLoginFailed {
			t.Errorf("错误回调被接受: %q", raw)
		}
	}
}

func TestDingTalkQRCodeRejectsInvalidGenerationAndReleasesCapacity(t *testing.T) {
	for _, link := range []string{
		"https://other.example.com/oauth2/qr_confirm.htm?code=x",
		"https://login.dingtalk.com/oauth2/auth?code=x",
		"https://login.dingtalk.com/oauth2/qr_confirm.htm?code=",
		"https://login.dingtalk.com/oauth2/qr_confirm.htm?code=x&code=%ZZ",
		"https://login.dingtalk.com/oauth2/qr_confirm.htm?code=x&redirect_uri=https://other.example.com",
	} {
		provider := qrTestProvider(roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.Method == http.MethodGet {
				return jsonResponse(http.StatusOK, "ok"), nil
			}
			return qrTestResponse(map[string]any{"success": true, "result": link}), nil
		}))
		if _, err := provider.CreateQRCode(t.Context(), "state", time.Now().Add(time.Minute)); err != biz.ErrDingTalkLoginFailed {
			t.Fatal("接受了非法扫码链接")
		}
		if len(provider.qrAttempts) != 0 {
			t.Fatal("创建失败占用缓存未释放")
		}
	}
}

func TestDingTalkQRCodeCancellationSerializesPolling(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var active, maxActive, calls atomic.Int32
	provider := qrTestProvider(roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, "ok"), nil
		}
		if request.URL.Path == "/oauth2/generate_qrcode" {
			return qrTestResponse(map[string]any{"success": true, "result": dingTalkLoginOrigin + "/oauth2/qr_confirm.htm?code=x"}), nil
		}
		current := active.Add(1)
		defer active.Add(-1)
		if current > maxActive.Load() {
			maxActive.Store(current)
		}
		if calls.Add(1) == 1 {
			close(entered)
			<-request.Context().Done()
			<-release
			return nil, request.Context().Err()
		}
		return qrTestResponse(map[string]any{"success": false, "errorCode": "11021"}), nil
	}))
	if _, err := provider.CreateQRCode(t.Context(), "state", time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	firstCtx, cancel := context.WithCancel(t.Context())
	first := make(chan error, 1)
	go func() { _, err := provider.PollQRCode(firstCtx, "state"); first <- err }()
	<-entered
	cancel()
	queuedCtx, cancelQueued := context.WithCancel(t.Context())
	cancelQueued()
	if _, err := provider.PollQRCode(queuedCtx, "state"); !errors.Is(err, context.Canceled) {
		t.Fatalf("排队中的已取消请求没有退出: %v", err)
	}
	second := make(chan error, 1)
	go func() {
		result, err := provider.PollQRCode(t.Context(), "state")
		if err == nil && result.Status != biz.DingTalkQRCodeWaiting {
			err = fmt.Errorf("等待状态错误")
		}
		second <- err
	}()
	close(release)
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Fatalf("取消没有传递: %v", err)
	}
	if err := <-second; err != nil {
		t.Fatalf("取消后重挂载误报失败: %v", err)
	}
	if maxActive.Load() != 1 {
		t.Fatal("同attempt并发访问上游")
	}
}

func TestDingTalkQRCodeExpiryAndCapacity(t *testing.T) {
	provider := qrTestProvider(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Error("过期或满容量不应请求上游")
		return nil, errors.New("不应访问")
	}))
	provider.qrAttempts = make(map[string]*dingTalkQRAttempt)
	for i := range dingTalkQRMaxAttempts {
		provider.qrAttempts[fmt.Sprint(i)] = &dingTalkQRAttempt{expiresAt: time.Now().Add(time.Minute)}
	}
	if _, err := provider.CreateQRCode(t.Context(), "extra", time.Now().Add(time.Minute)); err != biz.ErrDingTalkLoginFailed {
		t.Fatal("缓存超出固定容量")
	}
	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	provider.qrAttempts["expired"] = &dingTalkQRAttempt{gate: gate, expiresAt: time.Now().Add(-time.Second)}
	result, err := provider.PollQRCode(t.Context(), "expired")
	if err != nil || result.Status != biz.DingTalkQRCodeExpired {
		t.Fatal("过期尝试仍访问上游")
	}
}

func TestDingTalkQRCodeHTTPFailureDoesNotExposeBody(t *testing.T) {
	provider := qrTestProvider(roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, "ok"), nil
		}
		return jsonResponse(http.StatusOK, strings.Repeat("secret", dingTalkQRBodyLimit)), nil
	}))
	if _, err := provider.CreateQRCode(t.Context(), "state", time.Now().Add(time.Minute)); err != biz.ErrDingTalkLoginFailed {
		t.Fatal("超大上游报文没有被拒绝")
	}
}
