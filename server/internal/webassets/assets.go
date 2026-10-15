package webassets

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"path"
	"strings"
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
	if err != nil || !bytes.Contains(entry, []byte("</head>")) {
		http.Error(response, "登录页面加载失败", http.StatusInternalServerError)
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
	if cookie != nil {
		http.SetCookie(response, cookie)
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	// 逐次生成的配置不能使用静态文件的条件请求/304 逻辑。
	_, _ = response.Write(entry)
}
