package server

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/middleware"
)

func TestRecoveryDoesNotLogRequestContents(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	handler := Recovery(logger)(middleware.Handler(func(context.Context, any) (any, error) {
		panic("模拟异常")
	}))
	request := struct {
		Password string
		Token    string
	}{Password: "super-secret-password", Token: "preview-secret-token"}

	response, err := handler(context.Background(), request)
	if response != nil {
		t.Fatalf("panic 后响应 = %#v，期望 nil", response)
	}
	serviceError := kratoserrors.FromError(err)
	if serviceError.Code != 500 || serviceError.Reason != kratoserrors.UnknownReason {
		t.Fatalf("panic 错误 = %#v，期望通用 500 错误", serviceError)
	}
	logged := output.String()
	for _, secret := range []string{request.Password, request.Token} {
		if strings.Contains(logged, secret) {
			t.Fatalf("panic 日志泄漏敏感请求内容 %q: %s", secret, logged)
		}
	}
	if !strings.Contains(logged, `"event":"transport.panic.recovered"`) || !strings.Contains(logged, `"panic.type":"string"`) {
		t.Fatalf("panic 日志缺少结构化诊断字段: %s", logged)
	}
}
