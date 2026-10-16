package server

import (
	"context"
	stderrors "errors"
	"log/slog"
	"sync"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

const (
	objectDeletionLeaseDuration = 60 * time.Second
	objectDeletionSendTimeout   = 30 * time.Second
)

// ObjectDeletionWorker 通过 Kratos Server 生命周期消费对象存储删除任务。
type ObjectDeletionWorker struct {
	usecase   *biz.ObjectDeletionUsecase
	logger    *slog.Logger
	scheduler biz.BackgroundTaskScheduler
	ctx       context.Context
	cancel    context.CancelFunc
	done      chan struct{}
	once      sync.Once
}

func NewObjectDeletionWorker(usecase *biz.ObjectDeletionUsecase, logger *slog.Logger, scheduler biz.BackgroundTaskScheduler) *ObjectDeletionWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &ObjectDeletionWorker{usecase: usecase, logger: logger, scheduler: scheduler, ctx: ctx, cancel: cancel, done: make(chan struct{})}
}

func (w *ObjectDeletionWorker) Start(context.Context) error {
	defer close(w.done)
	if !w.usecase.Enabled() {
		w.logger.Info("object deletion worker disabled", slog.String("storage", "OBJECT_STORAGE"))
		<-w.ctx.Done()
		return nil
	}
	w.logger.Info("object deletion worker started", slog.String("storage", "OBJECT_STORAGE"))
	waiter := newWorkerScheduleWait(w.scheduler, biz.TaskConsumerObjectDeletion, false, w.logger)
	defer waiter.close()
	for {
		select {
		case <-w.ctx.Done():
			return nil
		default:
		}
		waiter.drain()
		processCtx, cancel := context.WithTimeout(w.ctx, objectDeletionSendTimeout)
		err := w.usecase.ProcessNext(processCtx, objectDeletionLeaseDuration)
		cancel()
		if err != nil && !stderrors.Is(err, biz.ErrBackgroundTaskNoTask) {
			w.logger.Error("process object deletion", slog.String("storage", "OBJECT_STORAGE"), slog.Any("error", err))
		}
		if err == nil {
			continue
		}
		if !waiter.wait(w.ctx, !stderrors.Is(err, biz.ErrBackgroundTaskNoTask)) {
			return nil
		}
	}
}

func (w *ObjectDeletionWorker) Stop(ctx context.Context) error {
	w.once.Do(w.cancel)
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
