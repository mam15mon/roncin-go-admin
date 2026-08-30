package requestmeta

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/transport"
)

type testHeader nethttp.Header

func (header testHeader) Get(key string) string      { return nethttp.Header(header).Get(key) }
func (header testHeader) Set(key, value string)      { nethttp.Header(header).Set(key, value) }
func (header testHeader) Add(key, value string)      { nethttp.Header(header).Add(key, value) }
func (header testHeader) Values(key string) []string { return nethttp.Header(header).Values(key) }
func (header testHeader) Keys() []string {
	keys := make([]string, 0, len(header))
	for key := range header {
		keys = append(keys, key)
	}
	return keys
}

type testHTTPTransport struct {
	request     *nethttp.Request
	replyHeader testHeader
}

func (tr *testHTTPTransport) Kind() transport.Kind            { return transport.KindHTTP }
func (tr *testHTTPTransport) Endpoint() string                { return "http://127.0.0.1:8000" }
func (tr *testHTTPTransport) Operation() string               { return "/test.Service/Get" }
func (tr *testHTTPTransport) RequestHeader() transport.Header { return testHeader(tr.request.Header) }
func (tr *testHTTPTransport) ReplyHeader() transport.Header   { return tr.replyHeader }
func (tr *testHTTPTransport) Request() *nethttp.Request       { return tr.request }

func TestLoggingRecordsHTTPAccessFields(t *testing.T) {
	request := httptest.NewRequest(nethttp.MethodGet, "http://example.com/api/v1/orders?keyword=secret", nil)
	request.RemoteAddr = "203.0.113.9:4567"
	request.Header.Set("User-Agent", "roncin-test-client/1.0")
	request.Header.Set("X-Request-ID", "test-request-id")
	tr := &testHTTPTransport{request: request, replyHeader: testHeader{}}
	ctx := transport.NewServerContext(context.Background(), tr)

	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	handler := Middleware()(Logging(logger)(func(context.Context, any) (any, error) {
		return nil, kratoserrors.BadRequest("INVALID_ARGUMENT", "参数错误")
	}))
	if _, err := handler(ctx, nil); err == nil {
		t.Fatal("测试处理器应返回错误")
	}

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("解析结构化日志失败: %v", err)
	}
	expected := map[string]any{
		"request.id":     "test-request-id",
		"transport.kind": "http",
		"rpc.operation":  "/test.Service/Get",
		"http.method":    "GET",
		"http.path":      "/api/v1/orders",
		"status.code":    float64(400),
		"client.ip":      "203.0.113.9",
		"user_agent":     "roncin-test-client/1.0",
	}
	for key, want := range expected {
		if got := entry[key]; got != want {
			t.Fatalf("日志字段 %s = %v，期望 %v", key, got, want)
		}
	}
	if got := tr.replyHeader.Get("X-Request-ID"); got != "test-request-id" {
		t.Fatalf("响应请求 ID = %q，期望 test-request-id", got)
	}
	if got := entry["http.path"]; got == request.URL.RequestURI() {
		t.Fatal("访问日志路径不得包含查询参数")
	}
}
