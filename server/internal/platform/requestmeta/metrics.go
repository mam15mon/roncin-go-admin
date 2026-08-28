package requestmeta

import (
	"context"
	"strconv"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	transportRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "roncin",
			Subsystem: "server",
			Name:      "requests_total",
			Help:      "服务端处理的传输请求总数。",
		},
		[]string{"transport", "operation", "code"},
	)
	transportDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "roncin",
			Subsystem: "server",
			Name:      "request_duration_seconds",
			Help:      "服务端传输请求处理耗时（秒）。",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"transport", "operation"},
	)
)

func init() {
	prometheus.MustRegister(transportRequests, transportDuration)
}

// Metrics 记录低基数的服务端请求数量与耗时，不采集请求参数或响应内容。
func Metrics() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			startedAt := time.Now()
			transportKind, operation := transportLabels(ctx)
			defer func() {
				code := "200"
				if recovered := recover(); recovered != nil {
					transportRequests.WithLabelValues(transportKind, operation, "500").Inc()
					transportDuration.WithLabelValues(transportKind, operation).Observe(time.Since(startedAt).Seconds())
					panic(recovered)
				}
				if err != nil {
					code = strconv.Itoa(int(kratoserrors.FromError(err).Code))
				}
				transportRequests.WithLabelValues(transportKind, operation, code).Inc()
				transportDuration.WithLabelValues(transportKind, operation).Observe(time.Since(startedAt).Seconds())
			}()
			return handler(ctx, req)
		}
	}
}

func transportLabels(ctx context.Context) (string, string) {
	transportKind := "unknown"
	operation := "unknown"
	if tr, ok := transport.FromServerContext(ctx); ok {
		if tr.Kind() != "" {
			transportKind = tr.Kind().String()
		}
		if tr.Operation() != "" {
			operation = tr.Operation()
		}
	}
	return transportKind, operation
}
