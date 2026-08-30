package requestmeta

import (
	"context"
	"log/slog"
	"net"
	nethttp "net/http"
	"regexp"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

type requestIDKey struct{}
type ipAddressKey struct{}

type requestCarrier interface {
	Request() *nethttp.Request
}

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

func FromContext(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey{}).(string)
	return value
}

func IPAddress(ctx context.Context) string {
	value, _ := ctx.Value(ipAddressKey{}).(string)
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
			ctx = context.WithValue(ctx, requestIDKey{}, requestID)
			if tr, ok := transport.FromServerContext(ctx); ok {
				if requestTransport, ok := tr.(requestCarrier); ok && requestTransport.Request() != nil {
					ctx = context.WithValue(ctx, ipAddressKey{}, remoteIPAddress(requestTransport.Request().RemoteAddr))
				}
			}
			if tr, ok := transport.FromServerContext(ctx); ok {
				tr.ReplyHeader().Set("X-Request-ID", requestID)
			}
			return handler(ctx, req)
		}
	}
}

func remoteIPAddress(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}

// Logging writes one structured completion event per RPC without request bodies.
func Logging(logger *slog.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			startedAt := time.Now()
			operation := ""
			transportKind := ""
			method := ""
			path := ""
			userAgent := ""
			if tr, ok := transport.FromServerContext(ctx); ok {
				operation = tr.Operation()
				transportKind = tr.Kind().String()
				userAgent = tr.RequestHeader().Get("User-Agent")
				if requestTransport, ok := tr.(requestCarrier); ok && requestTransport.Request() != nil {
					method = requestTransport.Request().Method
					path = requestTransport.Request().URL.Path
				}
			}
			reply, err = handler(ctx, req)
			level := slog.LevelInfo
			statusCode := 200
			if err != nil {
				level = slog.LevelError
				statusCode = int(kratoserrors.FromError(err).Code)
			}
			logger.Log(ctx, level, "request completed",
				slog.String("event", "transport.request.completed"),
				slog.String("request.id", FromContext(ctx)),
				slog.String("trace.id", TraceID(ctx)),
				slog.String("transport.kind", transportKind),
				slog.String("rpc.operation", operation),
				slog.String("http.method", method),
				slog.String("http.path", path),
				slog.Int("status.code", statusCode),
				slog.String("client.ip", IPAddress(ctx)),
				slog.String("user_agent", userAgent),
				slog.Duration("duration", time.Since(startedAt)),
				slog.Any("error", err),
			)
			return reply, err
		}
	}
}
