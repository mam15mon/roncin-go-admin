package webassets

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"html"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

//go:embed all:dist
var embedded embed.FS

type handler struct {
	files       fs.FS
	fileServer  http.Handler
	loginConfig func(context.Context) (json.RawMessage, *http.Cookie, error)
}

// Handler 提供编译后的前端资源；无扩展名应用路径返回 SPA，API 和健康检查保留给服务端。
func Handler(loginConfig func(context.Context) (json.RawMessage, *http.Cookie, error)) http.Handler {
	dist, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err)
	}
	return &handler{files: dist, fileServer: http.FileServer(http.FS(dist)), loginConfig: loginConfig}
}

func (h *handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		response.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	assetPath := strings.TrimPrefix(path.Clean(request.URL.Path), "/")
	if assetPath == "." || assetPath == "" {
		h.serveEntry(response, request)
		return
	}
	if assetPath == "api" || strings.HasPrefix(assetPath, "api/") || assetPath == "health" || strings.HasPrefix(assetPath, "health/") {
		http.NotFound(response, request)
		return
	}
	if info, err := fs.Stat(h.files, assetPath); err == nil && !info.IsDir() {
		h.fileServer.ServeHTTP(response, request)
		return
	}
	if path.Ext(assetPath) != "" {
		http.NotFound(response, request)
		return
	}
	h.serveEntry(response, request)
}

func (h *handler) serveEntry(response http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodGet && request.URL.Path == "/user/login" {
		h.serveLoginEntry(response, request)
		return
	}
	entryRequest := request.Clone(request.Context())
	entryRequest.URL.Path = "/"
	h.fileServer.ServeHTTP(response, entryRequest)
}

func (h *handler) serveLoginEntry(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("Pragma", "no-cache")
	entry, err := fs.ReadFile(h.files, "index.html")
	if err != nil || !bytes.Contains(entry, []byte("</head>")) || !bytes.Contains(entry, []byte("</body>")) {
		http.Error(response, "登录页面加载失败", http.StatusInternalServerError)
		return
	}
	// UA 仅控制入口体验；客户端身份仍由免登换码验证。
	if strings.Contains(strings.ToLower(request.UserAgent()), "dingtalk") {
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = response.Write(entry)
		return
	}
	payload, cookie, err := h.loginConfig(request.Context())
	if err != nil {
		http.Error(response, "登录配置加载失败", http.StatusInternalServerError)
		return
	}
	// RawMessage 再经过标准 JSON 编码，转义 <、>、&，防止字符串闭合 script 标签。
	safePayload, err := json.Marshal(payload)
	if err != nil {
		http.Error(response, "登录配置编码失败", http.StatusInternalServerError)
		return
	}
	script := append([]byte(`<script type="application/json" id="roncin-dingtalk-login-config">`), safePayload...)
	script = append(script, []byte("</script></head>")...)
	entry = bytes.Replace(entry, []byte("</head>"), script, 1)
	if frame := preloadedLoginFrame(payload, request); frame != "" {
		entry = bytes.Replace(entry, []byte("</body>"), []byte(frame+"</body>"), 1)
	}
	if cookie != nil {
		http.SetCookie(response, cookie)
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	// 逐次生成的配置不能使用静态文件的条件请求/304 逻辑。
	_, _ = response.Write(entry)
}

// 只从本次公开响应输出合法、未到期的扫码实例，不参与认证领域转换。
func preloadedLoginFrame(payload json.RawMessage, request *http.Request) string {
	var response struct {
		Success *bool `json:"success"`
		Data    struct {
			Enabled      bool   `json:"enabled"`
			AuthorizeURL string `json:"authorizeUrl"`
			ExpiresAt    string `json:"expiresAt"`
		} `json:"data"`
	}
	if json.Unmarshal(payload, &response) != nil || (response.Success != nil && !*response.Success) || !response.Data.Enabled {
		return ""
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, response.Data.ExpiresAt)
	if err != nil || !expiresAt.After(time.Now()) {
		return ""
	}
	authorize, err := url.Parse(response.Data.AuthorizeURL)
	if err != nil || authorize.Scheme != "https" || authorize.Host != "login.dingtalk.com" || authorize.Path != "/oauth2/auth" || authorize.User != nil {
		return ""
	}
	query, err := url.ParseQuery(authorize.RawQuery)
	if err != nil || query.Get("state") == "" || query.Get("client_id") == "" || query.Get("response_type") != "code" || query.Get("scope") == "" {
		return ""
	}
	callback, err := url.Parse(query.Get("redirect_uri"))
	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	// 反向代理透传的协议与浏览器入口一致；只接受实际 HTTP 协议值。
	if forwarded := request.Header.Get("X-Forwarded-Proto"); forwarded == "http" || forwarded == "https" {
		scheme = forwarded
	}
	if err != nil || callback.Scheme != scheme || callback.Host != request.Host || callback.Path != "/user/login/dingtalk/callback" || callback.User != nil {
		return ""
	}
	query.Set("iframe", "true")
	authorize.RawQuery = query.Encode()
	return `<iframe id="roncin-dingtalk-login-frame" title="钉钉扫码登录二维码" src="` + html.EscapeString(authorize.String()) + `" data-expires-at="` + strconv.FormatInt(expiresAt.UnixMilli(), 10) + `" style="position:absolute;visibility:hidden;width:280px;height:280px;border:0" referrerpolicy="no-referrer" onload="this.dataset.navigation='loaded'" onerror="this.dataset.navigation='failed'"></iframe>`
}
