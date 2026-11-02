package webassets

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func newTestHandler(provider func(context.Context) (json.RawMessage, *http.Cookie, error)) http.Handler {
	files := fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html><html><head><title>登录</title></head><body><div id=\"root\"></div></body></html>")},
		"assets/app.js": {Data: []byte("window.applicationLoaded=true")},
	}
	return &handler{files: files, fileServer: http.FileServer(http.FS(files)), loginConfig: provider}
}

func TestLoginEntryBootstrapIsPrivateAndUnconditional(t *testing.T) {
	calls := 0
	h := newTestHandler(func(context.Context) (json.RawMessage, *http.Cookie, error) {
		calls++
		state := strings.Repeat("s", calls)
		payload, err := json.Marshal(map[string]any{"success": true, "data": map[string]any{"enabled": true, "state": state, "qrCodeUrl": "https://login.dingtalk.com/oauth2/qr_confirm.htm?code=sample"}})
		return payload, &http.Cookie{Name: "login_state", Value: state, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: 300}, err
	})
	lastState := ""
	for range 2 {
		request := httptest.NewRequest(http.MethodGet, "/user/login?redirect=/orders", nil)
		request.Header.Set("If-None-Match", "*")
		request.Header.Set("If-Modified-Since", "Wed, 21 Oct 2099 07:28:00 GMT")
		response := httptest.NewRecorder()
		h.ServeHTTP(response, request)
		if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("Pragma") != "no-cache" {
			t.Fatalf("登录 HTML 缓存/状态错误: %d %v", response.Code, response.Header())
		}
		cookies := response.Result().Cookies()
		if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode || cookies[0].Path != "/" || cookies[0].MaxAge != 300 {
			t.Fatalf("Cookie 属性错误: %+v", cookies)
		}
		state := cookies[0].Value
		if state == lastState || !strings.Contains(response.Body.String(), `"state":"`+state+`"`) {
			t.Fatalf("HTML 与 Cookie 未独立对应: %s %s", state, response.Body.String())
		}
		if !strings.Contains(response.Body.String(), `id="roncin-dingtalk-login-config"`) || !strings.Contains(response.Body.String(), "</script></head>") {
			t.Fatalf("未在 head 结束前注入配置: %s", response.Body.String())
		}
		lastState = state
	}
	if calls != 2 {
		t.Fatalf("每次 HTML 应各生成一次配置: %d", calls)
	}
}

func TestLoginEntryEscapesScriptAndKeepsJSONRoundTrip(t *testing.T) {
	unsafe := `</script><script>alert("test")</script>&<tag>`
	payload, err := json.Marshal(map[string]string{"message": unsafe})
	if err != nil {
		t.Fatal(err)
	}
	// 模拟未转义 HTML 的上游 ProtoJSON。
	payload = []byte(strings.NewReplacer(`\u003c`, "<", `\u003e`, ">", `\u0026`, "&").Replace(string(payload)))
	h := newTestHandler(func(context.Context) (json.RawMessage, *http.Cookie, error) { return payload, nil, nil })
	response := httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/user/login", nil))
	body := response.Body.String()
	if response.Code != http.StatusOK || strings.Contains(body, "</script><script>") || !strings.Contains(body, `\u003c`) || !strings.Contains(body, `\u0026`) {
		t.Fatalf("script 未安全转义: %d %s", response.Code, body)
	}
	start := strings.Index(body, `id="roncin-dingtalk-login-config">`) + len(`id="roncin-dingtalk-login-config">`)
	end := strings.Index(body[start:], "</script>") + start
	var decoded map[string]string
	if err := json.Unmarshal([]byte(body[start:end]), &decoded); err != nil || decoded["message"] != unsafe {
		t.Fatalf("JSON 往返失真: %v %+v", err, decoded)
	}
}

func TestLoginEntryDisabledAndExplicitFailureKeepAppShell(t *testing.T) {
	for _, payload := range []string{`{"success":true,"data":{}}`, `{"success":false,"code":500,"message":"钉钉登录配置加载失败，请稍后重试"}`} {
		t.Run(payload, func(t *testing.T) {
			h := newTestHandler(func(context.Context) (json.RawMessage, *http.Cookie, error) {
				return json.RawMessage(payload), nil, nil
			})
			response := httptest.NewRecorder()
			h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/user/login", nil))
			if response.Code != http.StatusOK || len(response.Result().Cookies()) != 0 || !strings.Contains(response.Body.String(), payload) || !strings.Contains(response.Body.String(), `<div id="root">`) {
				t.Fatalf("禁用/错误配置未保留应用入口: %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestLoginBootstrapDoesNotChangeAdjacentRoutes(t *testing.T) {
	calls := 0
	h := newTestHandler(func(context.Context) (json.RawMessage, *http.Cookie, error) {
		calls++
		return nil, nil, errors.New("不应调用")
	})
	for _, item := range []struct {
		method, path string
		status       int
	}{
		{http.MethodGet, "/", 200}, {http.MethodGet, "/orders", 200},
		{http.MethodGet, "/user/login/dingtalk/callback?authCode=sample&state=sample", 200}, {http.MethodGet, "/user/login/", 200},
		{http.MethodHead, "/user/login", 200}, {http.MethodGet, "/index.html", 301},
		{http.MethodGet, "/assets/app.js", 200}, {http.MethodGet, "/missing.js", 404},
		{http.MethodGet, "/api/v1/auth/me", 404}, {http.MethodGet, "/health/live", 404},
		{http.MethodPost, "/user/login", 405},
	} {
		t.Run(item.method+item.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			h.ServeHTTP(response, httptest.NewRequest(item.method, item.path, nil))
			if response.Code != item.status || len(response.Result().Cookies()) != 0 || strings.Contains(response.Body.String(), "roncin-dingtalk-login-config") {
				t.Fatalf("相邻路径被改变: %d %v %s", response.Code, response.Header(), response.Body.String())
			}
		})
	}
	if calls != 0 {
		t.Fatalf("非目标路径生成了配置: %d", calls)
	}
}

func TestLoginEntryRejectsProviderAndEncodingErrors(t *testing.T) {
	for _, provider := range []func(context.Context) (json.RawMessage, *http.Cookie, error){
		func(context.Context) (json.RawMessage, *http.Cookie, error) {
			return nil, nil, errors.New("内部敏感错误")
		},
		func(context.Context) (json.RawMessage, *http.Cookie, error) {
			return json.RawMessage(`{"invalid"`), &http.Cookie{Name: "state", Value: "x"}, nil
		},
	} {
		h := newTestHandler(provider)
		response := httptest.NewRecorder()
		h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/user/login", nil))
		if response.Code != http.StatusInternalServerError || len(response.Result().Cookies()) != 0 || strings.Contains(response.Body.String(), "内部敏感错误") {
			t.Fatalf("内部错误处理不正确: %d %s", response.Code, response.Body.String())
		}
	}
}

func TestLoginEntryFileErrorsDoNotGenerateState(t *testing.T) {
	for _, files := range []fstest.MapFS{
		{},
		{"index.html": {Data: []byte("<html><body>不完整页面</body></html>")}},
	} {
		calls := 0
		h := &handler{files: files, loginConfig: func(context.Context) (json.RawMessage, *http.Cookie, error) {
			calls++
			return json.RawMessage(`{}`), nil, nil
		}}
		response := httptest.NewRecorder()
		h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/user/login", nil))
		if response.Code != http.StatusInternalServerError || calls != 0 || len(response.Result().Cookies()) != 0 {
			t.Fatalf("入口文件错误时生成了 state: %d %d %v", response.Code, calls, response.Header())
		}
	}
}

func TestLoginEntryInjectsQRCodeDataWithoutExternalFrame(t *testing.T) {
	payload := json.RawMessage(`{"success":true,"data":{"enabled":true,"qrCodeUrl":"https://login.dingtalk.com/oauth2/qr_confirm.htm?code=sample","state":"same-response","expiresAt":"2037-01-01T00:00:00Z"}}`)
	h := newTestHandler(func(context.Context) (json.RawMessage, *http.Cookie, error) {
		return payload, &http.Cookie{Name: "login_state", Value: "same-response", Path: "/", HttpOnly: true}, nil
	})
	response := httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://example.com/user/login", nil))
	body := response.Body.String()
	if response.Code != http.StatusOK || strings.Contains(body, "<iframe") || !strings.Contains(body, `"qrCodeUrl"`) || !strings.Contains(body, `"state":"same-response"`) || len(response.Result().Cookies()) != 1 {
		t.Fatalf("本地二维码配置或同响应 Cookie 错误: %d %s", response.Code, body)
	}
}

func TestDingTalkLoginEntrySkipsQRBootstrapAndConditionalCache(t *testing.T) {
	for _, ua := range []string{"DingTalk/7.0", "Mozilla dInGtAlK Desktop"} {
		t.Run(ua, func(t *testing.T) {
			h := newTestHandler(func(context.Context) (json.RawMessage, *http.Cookie, error) {
				t.Fatal("钉钉客户端不得创建扫码配置")
				return nil, nil, nil
			})
			request := httptest.NewRequest(http.MethodGet, "/user/login?redirect=/orders", nil)
			request.Header.Set("User-Agent", ua)
			request.Header.Set("If-None-Match", "*")
			response := httptest.NewRecorder()
			h.ServeHTTP(response, request)
			if response.Code != 200 || response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("Pragma") != "no-cache" || len(response.Result().Cookies()) != 0 || strings.Contains(response.Body.String(), "roncin-dingtalk-login") || !strings.Contains(response.Body.String(), `id="root"`) {
				t.Fatalf("钉钉 HTML 分流错误: %d %s", response.Code, response.Body.String())
			}
		})
	}
}
