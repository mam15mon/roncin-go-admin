package server

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

const (
	// exchangeRateReminderPollInterval 督办轮询间隔；检查本身按周触发，
	// 无需高频轮询，30 分钟粒度足够贴合周一 10:00 的督办时点。
	exchangeRateReminderPollInterval = 30 * time.Minute
	// exchangeRateReminderCheckStartHour 周一 10:00（Asia/Shanghai）督办起点。
	exchangeRateReminderCheckStartHour = 10
	// exchangeRateReminderCheckWindowHours 检查窗口 24 小时：服务在周一停机时
	// 周二上午仍可补发一次；确定性任务 ID 保证同周期不重发。
	exchangeRateReminderCheckWindowHours = 24
)

// ExchangeRateReminderWorker 周一 10:00（Asia/Shanghai）后检测分公司当周汇率
// 未同步并经既有通知发件箱（notification_delivery）推送财务待办。
type ExchangeRateReminderWorker struct {
	usecase *biz.ExchangeRateReminderUsecase
	logger  *slog.Logger
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{}
	once    sync.Once
}

func NewExchangeRateReminderWorker(usecase *biz.ExchangeRateReminderUsecase, logger *slog.Logger) *ExchangeRateReminderWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &ExchangeRateReminderWorker{usecase: usecase, logger: logger, ctx: ctx, cancel: cancel, done: make(chan struct{})}
}

func (w *ExchangeRateReminderWorker) Start(context.Context) error {
	defer close(w.done)
	w.logger.Info("exchange rate reminder worker started")
	ticker := time.NewTicker(exchangeRateReminderPollInterval)
	defer ticker.Stop()
	// 启动即先检查一次（窗口内补发场景），随后按固定间隔轮询。
	w.runCheck()
	for {
		select {
		case <-w.ctx.Done():
			return nil
		case <-ticker.C:
			w.runCheck()
		}
	}
}

func (w *ExchangeRateReminderWorker) runCheck() {
	now := time.Now().In(biz.ExchangeRateBusinessLocation())
	if !exchangeRateReminderCheckDue(now) {
		return
	}
	checkCtx, cancel := context.WithTimeout(w.ctx, 30*time.Second)
	defer cancel()
	notified, err := w.usecase.RunWeeklyReminderCheck(checkCtx, now)
	if err != nil {
		w.logger.Error("run exchange rate weekly reminder check", slog.Any("error", err))
		return
	}
	if notified > 0 {
		w.logger.Info("exchange rate weekly reminders enqueued", slog.Int("count", notified))
	}
}

// exchangeRateReminderCheckDue 判定当前时刻是否处于督办检查窗口：
// 周一 10:00 至周二 10:00（Asia/Shanghai）。
func exchangeRateReminderCheckDue(now time.Time) bool {
	weekFrom, _ := biz.ExchangeRateWeekWindow(now)
	deadline := weekFrom.AddDate(0, 0, 0).Add(time.Duration(exchangeRateReminderCheckStartHour) * time.Hour)
	windowEnd := deadline.Add(exchangeRateReminderCheckWindowHours * time.Hour)
	return !now.Before(deadline) && now.Before(windowEnd)
}

func (w *ExchangeRateReminderWorker) Stop(ctx context.Context) error {
	w.once.Do(w.cancel)
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
