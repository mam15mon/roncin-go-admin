package server

import (
	"context"
	"testing"

	authv1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
)

func TestRequiredFieldsValidatorRejectsMissingProtoField(t *testing.T) {
	called := false
	handler := RequiredFieldsValidator()(func(_ context.Context, _ any) (any, error) {
		called = true
		return nil, nil
	})

	if _, err := handler(t.Context(), &authv1.LoginRequest{}); err == nil {
		t.Fatal("缺少必填字段的 Proto 请求未被拒绝")
	}
	if called {
		t.Fatal("校验失败后不应继续调用业务处理器")
	}
}

func TestRequiredFieldsValidatorAllowsCompleteProtoRequest(t *testing.T) {
	called := false
	handler := RequiredFieldsValidator()(func(_ context.Context, _ any) (any, error) {
		called = true
		return nil, nil
	})

	if _, err := handler(t.Context(), &authv1.LoginRequest{Username: "admin", Password: "secret"}); err != nil {
		t.Fatalf("完整 Proto 请求被拒绝: %v", err)
	}
	if !called {
		t.Fatal("校验通过后未调用业务处理器")
	}
}
