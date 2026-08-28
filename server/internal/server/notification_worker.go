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
	notificationPollInterval  = 2 * time.Second
	notificationLeaseDuration = 30 * time.Second
	notificationSendTimeout   = 15 * time.Second
)

// NotificationWorker 通过 Kratos Server 生命周期消费通知发件箱。
type NotificationWorker struct {
	usecase *biz.NotificationUsecase
	logger  *slog.Logger
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{}
	once    sync.Once
}

func NewNotificationWorker(usecase *biz.NotificationUsecase, logger *slog.Logger) *NotificationWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &NotificationWorker{usecase: usecase, logger: logger, ctx: ctx, cancel: cancel, done: make(chan struct{})}
}

func (w *NotificationWorker) Start(context.Context) error {
	defer close(w.done)
	if !w.usecase.Enabled() {
		w.logger.Info("notification worker disabled", slog.String("channel", "DINGTALK"))
		<-w.ctx.Done()
		return nil
	}
	w.logger.Info("notification worker started", slog.String("channel", "DINGTALK"))
	for {
		select {
		case <-w.ctx.Done():
			return nil
		default:
		}
		processCtx, cancel := context.WithTimeout(w.ctx, notificationSendTimeout)
		err := w.usecase.ProcessNext(processCtx, notificationLeaseDuration)
		cancel()
		if err != nil && !stderrors.Is(err, biz.ErrBackgroundTaskNoTask) {
			w.logger.Error("process notification", slog.String("channel", "DINGTALK"), slog.Any("error", err))
		}
		if err == nil {
			continue
		}
		timer := time.NewTimer(notificationPollInterval)
		select {
		case <-w.ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil
		case <-timer.C:
		}
	}
}

func (w *NotificationWorker) Stop(ctx context.Context) error {
	w.once.Do(w.cancel)
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
