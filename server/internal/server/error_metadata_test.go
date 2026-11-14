package server

import (
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
)

func TestEncodeErrorExposesOnlyAllowedMetadata(t *testing.T) {
	err := errors.New(400, "FEE_EXCHANGE_RATE_MISSING", "汇率日期未命中生效汇率").WithMetadata(map[string]string{
		"currency": "USD", "week_start": "2026-11-09", "direction": "RECEIVABLE", "internal_secret": "x",
	})
	recorder := httptest.NewRecorder()
	encodeError(recorder, httptest.NewRequest(nethttp.MethodPut, "/api/v1/orders/1/fees", nil), err)
	body := recorder.Body.String()
	for _, want := range []string{`"currency":"USD"`, `"week_start":"2026-11-09"`, `"direction":"RECEIVABLE"`} {
		if !contains(body, want) {
			t.Fatalf("响应缺少 %s：%s", want, body)
		}
	}
	if contains(body, "internal_secret") {
		t.Fatalf("不应暴露非白名单 metadata：%s", body)
	}
}

func TestEncodeErrorOmitsMetadataWhenNoneAllowed(t *testing.T) {
	err := errors.New(409, "ORDER_FEE_VERSION_CONFLICT", "已被更新").WithMetadata(map[string]string{"x": "y"})
	recorder := httptest.NewRecorder()
	encodeError(recorder, httptest.NewRequest(nethttp.MethodPut, "/api/v1/orders/1/fees", nil), err)
	if contains(recorder.Body.String(), "metadata") {
		t.Fatalf("无白名单键时不应输出 metadata：%s", recorder.Body.String())
	}
}

func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}
