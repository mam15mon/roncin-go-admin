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
	dingTalkApprovalInitialPollInterval = 2 * time.Second
	dingTalkApprovalLeaseDuration       = 45 * time.Second
	dingTalkApprovalCallTimeout         = 20 * time.Second
)

// DingTalkApprovalWorker 通过 Kratos Server 生命周期可靠派发原生 OA 审批。
type DingTalkApprovalWorker struct {
	usecase *biz.DingTalkApprovalUsecase
	logger  *slog.Logger
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{}
	once    sync.Once
}

func NewDingTalkApprovalWorker(usecase *biz.DingTalkApprovalUsecase, logger *slog.Logger) *DingTalkApprovalWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &DingTalkApprovalWorker{usecase: usecase, logger: logger, ctx: ctx, cancel: cancel, done: make(chan struct{})}
}

func (w *DingTalkApprovalWorker) Start(context.Context) error {
	defer close(w.done)
	if !w.usecase.Enabled() {
		w.logger.Info("dingtalk approval worker disabled")
		<-w.ctx.Done()
		return nil
	}
	callbackEnabled := w.usecase.CallbackEnabled()
	w.logger.Info("dingtalk approval worker started", slog.Bool("callback_enabled", callbackEnabled))
	backoff := newWorkerPollBackoff(dingTalkApprovalInitialPollInterval, backgroundWorkerMaxPollInterval)
	for {
		select {
		case <-w.ctx.Done():
			return nil
		default:
		}

		dispatchErr := w.processDispatch()
		var inboxErr error = biz.ErrBackgroundTaskNoTask
		if callbackEnabled {
			inboxErr = w.processInbox()
		}
		if dispatchErr == nil || inboxErr == nil {
			backoff.Reset()
			continue
		}
		if !waitForWorkerPoll(w.ctx, backoff.Next()) {
			return nil
		}
	}
}

func (w *DingTalkApprovalWorker) processDispatch() error {
	processCtx, cancel := context.WithTimeout(w.ctx, dingTalkApprovalCallTimeout)
	defer cancel()
	err := w.usecase.ProcessNextDispatch(processCtx, dingTalkApprovalLeaseDuration)
	if err != nil && !stderrors.Is(err, biz.ErrBackgroundTaskNoTask) {
		w.logger.Error("process dingtalk approval dispatch", slog.Any("error", err))
	}
	return err
}

func (w *DingTalkApprovalWorker) processInbox() error {
	processCtx, cancel := context.WithTimeout(w.ctx, dingTalkApprovalCallTimeout)
	defer cancel()
	err := w.usecase.ProcessNextInbox(processCtx, dingTalkApprovalLeaseDuration)
	if err != nil && !stderrors.Is(err, biz.ErrBackgroundTaskNoTask) {
		w.logger.Error("process dingtalk approval inbox", slog.Any("error", err))
	}
	return err
}

func (w *DingTalkApprovalWorker) Stop(ctx context.Context) error {
	w.once.Do(w.cancel)
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
