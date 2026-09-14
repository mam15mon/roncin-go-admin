package server

import (
	"context"
	"testing"

	adminv1 "github.com/roncin/roncin-go-admin/server/api/admin/v1"
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

func TestRequiredFieldsValidatorAllowsEmptyOptionalRoleCode(t *testing.T) {
	// 角色编码已改为可选：留空由业务层自动生成，中间件不得再拦截。
	handler := RequiredFieldsValidator()(func(_ context.Context, _ any) (any, error) {
		return nil, nil
	})

	request := &adminv1.CreateRoleRequest{Name: "财务", DataScope: adminv1.DataScope_DATA_SCOPE_ORGANIZATION}
	if _, err := handler(t.Context(), request); err != nil {
		t.Fatalf("空编码的创建角色请求被必填校验拦截: %v", err)
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
