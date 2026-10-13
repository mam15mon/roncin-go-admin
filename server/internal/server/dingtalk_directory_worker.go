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
	usecase   *biz.DingTalkDirectoryUsecase
	logger    *slog.Logger
	scheduler biz.BackgroundTaskScheduler
	ctx       context.Context
	cancel    context.CancelFunc
	done      chan struct{}
	once      sync.Once
}

func NewDingTalkDirectoryWorker(uc *biz.DingTalkDirectoryUsecase, logger *slog.Logger, scheduler biz.BackgroundTaskScheduler) *DingTalkDirectoryWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &DingTalkDirectoryWorker{usecase: uc, logger: logger, scheduler: scheduler, ctx: ctx, cancel: cancel, done: make(chan struct{})}
}
func (w *DingTalkDirectoryWorker) Start(context.Context) error {
	defer close(w.done)
	if !w.usecase.Enabled() {
		<-w.ctx.Done()
		return nil
	}
	waiter := newWorkerScheduleWait(w.scheduler, biz.TaskConsumerDingTalkDirectory, false, w.logger)
	defer waiter.close()
	for {
		if w.ctx.Err() != nil {
			return nil
		}
		waiter.drain()
		err := w.usecase.ProcessNext(w.ctx, 3*time.Minute)
		if err == nil {
			continue
		}
		if !errors.Is(err, biz.ErrBackgroundTaskNoTask) {
			w.logger.Error("钉钉通讯录同步执行失败", slog.Any("error", err))
		}
		if !waiter.wait(w.ctx, !errors.Is(err, biz.ErrBackgroundTaskNoTask)) {
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
