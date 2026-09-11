package server

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type disabledApprovalGateway struct{}

func (disabledApprovalGateway) Enabled() bool { return false }

func (disabledApprovalGateway) Create(context.Context, *biz.DingTalkApprovalCreateCommand) (*biz.DingTalkApprovalCreateResult, error) {
	panic("钉钉审批关闭时不应创建审批")
}

func (disabledApprovalGateway) Query(context.Context, string) (*biz.DingTalkApprovalQueryResult, error) {
	panic("钉钉审批关闭时不应查询审批")
}

func TestWorkerPollBackoffIncreasesAndResets(t *testing.T) {
	backoff := newWorkerPollBackoff(2*time.Second, 15*time.Second)
	want := []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second, 15 * time.Second, 15 * time.Second}
	for index, expected := range want {
		if got := backoff.Next(); got != expected {
			t.Fatalf("第 %d 次退避 = %s，期望 %s", index+1, got, expected)
		}
	}

	backoff.Reset()
	if got := backoff.Next(); got != 2*time.Second {
		t.Fatalf("重置后的退避 = %s，期望 2s", got)
	}
}

func TestWaitForWorkerPollStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if waitForWorkerPoll(ctx, time.Hour) {
		t.Fatal("上下文取消后不应继续轮询")
	}
}

func TestDingTalkApprovalWorkerDoesNotPollWhenDisabled(t *testing.T) {
	usecase := biz.NewDingTalkApprovalUsecase(nil, nil, disabledApprovalGateway{}, nil)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	worker := NewDingTalkApprovalWorker(usecase, logger)
	started := make(chan error, 1)
	go func() {
		started <- worker.Start(context.Background())
	}()

	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := worker.Stop(stopCtx); err != nil {
		t.Fatalf("停止已禁用的钉钉审批 worker: %v", err)
	}
	if err := <-started; err != nil {
		t.Fatalf("已禁用的钉钉审批 worker 退出失败: %v", err)
	}
}

var _ biz.DingTalkApprovalGateway = disabledApprovalGateway{}
