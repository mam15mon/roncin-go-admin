package webassets

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var embedded embed.FS

type handler struct {
	files      fs.FS
	fileServer http.Handler
}

// Handler serves compiled web assets and maps extensionless application routes
// to the SPA entry document. API and health namespaces always remain server-owned.
func Handler() http.Handler {
	dist, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err)
	}
	return &handler{files: dist, fileServer: http.FileServer(http.FS(dist))}
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
	entryRequest := request.Clone(request.Context())
	entryRequest.URL.Path = "/"
	h.fileServer.ServeHTTP(response, entryRequest)
}
