package server

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
)

// Recovery 捕获传输层 panic，但不会把请求 DTO 写入日志，避免泄漏密码、令牌和文件内容。
func Recovery(logger *slog.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, request any) (reply any, err error) {
			defer func() {
				if recovered := recover(); recovered != nil {
					stack := make([]byte, 64<<10)
					stack = stack[:runtime.Stack(stack, false)]
					operation := ""
					if currentTransport, ok := transport.FromServerContext(ctx); ok {
						operation = currentTransport.Operation()
					}
					logger.ErrorContext(ctx, "panic recovered",
						slog.String("event", "transport.panic.recovered"),
						slog.String("request.id", requestmeta.FromContext(ctx)),
						slog.String("trace.id", requestmeta.TraceID(ctx)),
						slog.String("rpc.operation", operation),
						slog.String("panic.type", fmt.Sprintf("%T", recovered)),
						slog.String("stack", string(stack)),
					)
					err = errors.InternalServer(errors.UnknownReason, internalErrorMessage)
				}
			}()
			return handler(ctx, request)
		}
	}
}
