package server

import (
	"context"
	"errors"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"log/slog"
	"sync"
	"time"
)

// DingTalkDirectoryWorker 在租约内执行有界通讯录同步，使用已有调度器接管重启任务。
type DingTalkDirectoryWorker struct {
	usecase *biz.DingTalkDirectoryUsecase
	logger  *slog.Logger
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{}
	once    sync.Once
}

func NewDingTalkDirectoryWorker(uc *biz.DingTalkDirectoryUsecase, logger *slog.Logger) *DingTalkDirectoryWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &DingTalkDirectoryWorker{usecase: uc, logger: logger, ctx: ctx, cancel: cancel, done: make(chan struct{})}
}
func (w *DingTalkDirectoryWorker) Start(context.Context) error {
	defer close(w.done)
	if !w.usecase.Enabled() {
		<-w.ctx.Done()
		return nil
	}
	backoff := newWorkerPollBackoff(2*time.Second, backgroundWorkerMaxPollInterval)
	for {
		if w.ctx.Err() != nil {
			return nil
		}
		err := w.usecase.ProcessNext(w.ctx, 3*time.Minute)
		if err == nil {
			backoff.Reset()
			continue
		}
		if !errors.Is(err, biz.ErrBackgroundTaskNoTask) {
			w.logger.Error("钉钉通讯录同步执行失败", slog.Any("error", err))
		}
		if !waitForWorkerPoll(w.ctx, backoff.Next()) {
			return nil
		}
	}
}
func (w *DingTalkDirectoryWorker) Stop(ctx context.Context) error {
	w.once.Do(w.cancel)
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
