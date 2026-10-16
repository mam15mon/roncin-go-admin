package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

const (
	backgroundWorkerRecoveryInterval = time.Minute
	backgroundWorkerErrorMaxInterval = 30 * time.Second
)

type workerPollBackoff struct{ initial, maximum, current time.Duration }

func newWorkerPollBackoff(initial, maximum time.Duration) workerPollBackoff {
	return workerPollBackoff{initial: initial, maximum: maximum, current: initial}
}
func (b *workerPollBackoff) Next() time.Duration {
	interval := b.current
	b.current = min(b.current*2, b.maximum)
	return interval
}
func (b *workerPollBackoff) Reset() { b.current = b.initial }

// workerScheduleWait 在订阅后开始领取，通知在查询与等待之间仍保留于缓冲中。
type workerScheduleWait struct {
	scheduler    biz.BackgroundTaskScheduler
	consumer     biz.TaskConsumer
	includeInbox bool
	wake         <-chan struct{}
	close        func()
	backoff      workerPollBackoff
	logger       *slog.Logger
}

func newWorkerScheduleWait(scheduler biz.BackgroundTaskScheduler, consumer biz.TaskConsumer, includeInbox bool, logger *slog.Logger) *workerScheduleWait {
	wake, closeSubscription := scheduler.Subscribe(consumer)
	return &workerScheduleWait{scheduler: scheduler, consumer: consumer, includeInbox: includeInbox, wake: wake, close: closeSubscription, backoff: newWorkerPollBackoff(time.Second, backgroundWorkerErrorMaxInterval), logger: logger}
}

// 每轮领取前合并旧信号；本轮读取后产生的信号不能被清掉。
func (w *workerScheduleWait) drain() {
	select {
	case <-w.wake:
	default:
	}
}
func (w *workerScheduleWait) wait(ctx context.Context, processingFailed bool) bool {
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	next, err := w.scheduler.NextRun(queryCtx, w.consumer, w.includeInbox)
	cancel()
	if ctx.Err() != nil {
		return false
	}
	delay := backgroundWorkerRecoveryInterval
	if err != nil {
		w.logger.Error("查询后台任务下次调度时间失败", slog.String("consumer", string(w.consumer)), slog.Any("error", err))
	}
	if next != nil && err == nil {
		delay = min(delay, time.Until(*next))
	}
	failed := processingFailed || err != nil || delay <= 0
	if failed {
		// 领取冲突、过期时间与存储异常都必须等待，不能在过去时间上忙循环。
		delay = w.backoff.Next()
	} else {
		w.backoff.Reset()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	var minimum <-chan time.Time
	if failed {
		// 领取/失败更新自身也会通知；异常重试不能由该信号绕过最短退避。
		floor := time.NewTimer(time.Second)
		defer floor.Stop()
		minimum = floor.C
	}
	pendingWake := false
	for {
		select {
		case <-ctx.Done():
			return false
		case _, open := <-w.wake:
			if !open {
				return false
			}
			if minimum == nil {
				return true
			}
			pendingWake = true
		case <-minimum:
			minimum = nil
			if pendingWake {
				return true
			}
		case <-timer.C:
			return true
		}
	}
}
