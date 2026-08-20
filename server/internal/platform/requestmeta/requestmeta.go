package requestmeta

import (
	"context"
	"log/slog"
	"regexp"
	"time"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

type contextKey struct{}

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

func FromContext(ctx context.Context) string {
	value, _ := ctx.Value(contextKey{}).(string)
	return value
}

func TraceID(ctx context.Context) string {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return ""
	}
	return spanContext.TraceID().String()
}

// Middleware establishes one trusted request id and echoes it to the client.
func Middleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			requestID := ""
			if tr, ok := transport.FromServerContext(ctx); ok {
				candidate := tr.RequestHeader().Get("X-Request-ID")
				if validRequestID.MatchString(candidate) {
					requestID = candidate
				}
			}
			if requestID == "" {
				requestID = uuid.NewString()
			}
			ctx = context.WithValue(ctx, contextKey{}, requestID)
			if tr, ok := transport.FromServerContext(ctx); ok {
				tr.ReplyHeader().Set("X-Request-ID", requestID)
			}
			return handler(ctx, req)
		}
	}
}

// Logging writes one structured completion event per RPC without request bodies.
func Logging(logger *slog.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			startedAt := time.Now()
			operation := ""
			if tr, ok := transport.FromServerContext(ctx); ok {
				operation = tr.Operation()
			}
			reply, err = handler(ctx, req)
			level := slog.LevelInfo
			if err != nil {
				level = slog.LevelError
			}
			logger.Log(ctx, level, "request completed",
				slog.String("event", "transport.request.completed"),
				slog.String("request.id", FromContext(ctx)),
				slog.String("trace.id", TraceID(ctx)),
				slog.String("rpc.operation", operation),
				slog.Duration("duration", time.Since(startedAt)),
				slog.Any("error", err),
			)
			return reply, err
		}
	}
}
