package service

import (
	"context"
	"testing"

	adminv1 "github.com/roncin/roncin-go-admin/server/api/admin/v1"
	financev1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"go.opentelemetry.io/otel/trace"
)

func TestOKSetsCommonResponseEnvelope(t *testing.T) {
	ctx := responseTestTraceContext(t)
	response := ok(ctx, &financev1.CreateCashflowResponse{})
	if !response.GetSuccess() || response.GetMessage() != "OK" || response.GetTraceId() != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("成功响应封套 = %+v", response)
	}
}

func TestOKListPreservesTypedListFields(t *testing.T) {
	ctx := responseTestTraceContext(t)
	response := okList(ctx, &adminv1.ListUsersResponse{Total: 2, Page: 1, PageSize: 20})
	if !response.GetSuccess() || response.GetCode() != 0 || response.GetMessage() != "OK" || response.GetTraceId() != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("列表响应封套 = %+v", response)
	}
	if response.GetTotal() != 2 || response.GetPage() != 1 || response.GetPageSize() != 20 {
		t.Fatalf("列表分页字段 = total:%d page:%d pageSize:%d", response.GetTotal(), response.GetPage(), response.GetPageSize())
	}
}

func responseTestTraceContext(t *testing.T) context.Context {
	t.Helper()
	traceID, err := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("构造 trace ID: %v", err)
	}
	spanID, err := trace.SpanIDFromHex("0123456789abcdef")
	if err != nil {
		t.Fatalf("构造 span ID: %v", err)
	}
	return trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	}))
}
