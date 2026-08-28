package requestmeta

import (
	"context"
	"testing"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsRecordsSuccessAndError(t *testing.T) {
	successCounter := transportRequests.WithLabelValues("unknown", "unknown", "200")
	successBefore := testutil.ToFloat64(successCounter)
	if _, err := Metrics()(func(context.Context, any) (any, error) {
		return "ok", nil
	})(context.Background(), nil); err != nil {
		t.Fatalf("成功处理请求失败: %v", err)
	}
	if got := testutil.ToFloat64(successCounter); got != successBefore+1 {
		t.Fatalf("成功请求计数未增加：处理前 %.0f，处理后 %.0f", successBefore, got)
	}

	errorCounter := transportRequests.WithLabelValues("unknown", "unknown", "400")
	errorBefore := testutil.ToFloat64(errorCounter)
	if _, err := Metrics()(func(context.Context, any) (any, error) {
		return nil, kratoserrors.BadRequest("INVALID_ARGUMENT", "参数错误")
	})(context.Background(), nil); err == nil {
		t.Fatal("错误请求应返回错误")
	}
	if got := testutil.ToFloat64(errorCounter); got != errorBefore+1 {
		t.Fatalf("错误请求计数未增加：处理前 %.0f，处理后 %.0f", errorBefore, got)
	}
}

func TestMetricsRecordsPanicAndKeepsRecoveryResponsibility(t *testing.T) {
	panicCounter := transportRequests.WithLabelValues("unknown", "unknown", "500")
	panicBefore := testutil.ToFloat64(panicCounter)
	recovered := false
	func() {
		defer func() {
			if recover() != nil {
				recovered = true
			}
		}()
		_, _ = Metrics()(func(context.Context, any) (any, error) {
			panic("测试异常")
		})(context.Background(), nil)
	}()
	if !recovered {
		t.Fatal("指标中间件不得吞掉异常恢复流程")
	}
	if got := testutil.ToFloat64(panicCounter); got != panicBefore+1 {
		t.Fatalf("异常请求计数未增加：处理前 %.0f，处理后 %.0f", panicBefore, got)
	}
}
