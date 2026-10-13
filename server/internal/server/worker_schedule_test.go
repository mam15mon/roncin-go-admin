package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type fakeTaskScheduler struct {
	biz.BackgroundTaskScheduler
	wake         chan struct{}
	next         *time.Time
	err          error
	query        func()
	includeInbox bool
	closed       bool
}

func (s *fakeTaskScheduler) Subscribe(biz.TaskConsumer) (<-chan struct{}, func()) {
	return s.wake, func() { s.closed = true }
}
func (s *fakeTaskScheduler) NextRun(_ context.Context, _ biz.TaskConsumer, includeInbox bool) (*time.Time, error) {
	s.includeInbox = includeInbox
	if s.query != nil {
		s.query()
	}
	return s.next, s.err
}

func TestWorkerScheduleDeadlineWithoutNotification(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		due := time.Now().Add(7 * time.Second)
		scheduler := &fakeTaskScheduler{wake: make(chan struct{}, 1), next: &due}
		waiter := newWorkerScheduleWait(scheduler, biz.TaskConsumerDingTalkApproval, false, slog.New(slog.NewTextHandler(io.Discard, nil)))
		defer waiter.close()
		finished := make(chan bool, 1)
		go func() { finished <- waiter.wait(ctx, false) }()
		synctest.Wait()
		time.Sleep(6 * time.Second)
		synctest.Wait()
		select {
		case <-finished:
			t.Fatal("到期前不应重新领取")
		default:
		}
		time.Sleep(time.Second)
		synctest.Wait()
		if !<-finished {
			t.Fatal("到期未唤醒")
		}
		if scheduler.includeInbox {
			t.Fatal("回调关闭时不应查询收件箱")
		}
	})
}

func TestWorkerScheduleRetainsNotificationDuringQuery(t *testing.T) {
	scheduler := &fakeTaskScheduler{wake: make(chan struct{}, 1)}
	scheduler.query = func() { scheduler.wake <- struct{}{} }
	waiter := newWorkerScheduleWait(scheduler, biz.TaskConsumerNotification, false, slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer waiter.close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if !waiter.wait(ctx, false) {
		t.Fatal("查询与等待之间的通知丢失")
	}
}

func TestWorkerSchedulePastDeadlineAndErrorsBackOff(t *testing.T) {
	for _, queryError := range []bool{false, true} {
		t.Run(map[bool]string{false: "past_deadline", true: "query_error"}[queryError], func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				due := time.Now().Add(-time.Second)
				scheduler := &fakeTaskScheduler{wake: make(chan struct{}, 1), next: &due}
				if queryError {
					scheduler.err = errors.New("查询失败")
				}
				waiter := newWorkerScheduleWait(scheduler, biz.TaskConsumerNotification, false, slog.New(slog.NewTextHandler(io.Discard, nil)))
				defer waiter.close()
				finished := make(chan time.Time, 1)
				go func() {
					for index := 0; index < 4; index++ {
						waiter.wait(ctx, false)
						finished <- time.Now()
					}
				}()
				previous := time.Now()
				for _, delay := range []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second} {
					time.Sleep(delay)
					synctest.Wait()
					actual := <-finished
					if actual.Sub(previous) != delay {
						t.Fatalf("退避间隔=%s，期望%s", actual.Sub(previous), delay)
					}
					previous = actual
				}
			})
		})
	}
}

type forbiddenScheduler struct {
	biz.BackgroundTaskScheduler
	accesses atomic.Int64
}

func (s *forbiddenScheduler) Subscribe(biz.TaskConsumer) (<-chan struct{}, func()) {
	s.accesses.Add(1)
	panic("禁用消费者不得订阅")
}
func (s *forbiddenScheduler) NextRun(context.Context, biz.TaskConsumer, bool) (*time.Time, error) {
	s.accesses.Add(1)
	panic("禁用消费者不得查询")
}

type disabledDirectoryGateway struct{ biz.DingTalkDirectoryGateway }

func (disabledDirectoryGateway) Enabled() bool { return false }

func TestAllDisabledWorkersAvoidScheduler(t *testing.T) {
	scheduler := &forbiddenScheduler{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	workers := []interface {
		Start(context.Context) error
		Stop(context.Context) error
	}{
		NewNotificationWorker(biz.NewNotificationUsecase(nil, nil, nil), logger, scheduler),
		NewObjectDeletionWorker(biz.NewObjectDeletionUsecase(nil, nil, nil), logger, scheduler),
		NewDingTalkApprovalWorker(biz.NewDingTalkApprovalUsecase(nil, nil, nil, nil), logger, scheduler),
		NewDingTalkDirectoryWorker(biz.NewDingTalkDirectoryUsecase(nil, disabledDirectoryGateway{}, nil), logger, scheduler),
	}
	for _, worker := range workers {
		result := make(chan error, 1)
		go func() { result <- worker.Start(context.Background()) }()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		if err := worker.Stop(ctx); err != nil {
			t.Fatal(err)
		}
		cancel()
		if err := <-result; err != nil {
			t.Fatal(err)
		}
	}
	if scheduler.accesses.Load() != 0 {
		t.Fatal("关闭功能仍访问调度器")
	}
}

func TestWorkerScheduleClosedSubscriptionStops(t *testing.T) {
	scheduler := &fakeTaskScheduler{wake: make(chan struct{}, 1)}
	close(scheduler.wake)
	waiter := newWorkerScheduleWait(scheduler, biz.TaskConsumerNotification, false, slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer waiter.close()
	if waiter.wait(context.Background(), false) {
		t.Fatal("监听生命周期停止后 worker 不应继续循环")
	}
}

func TestWorkerScheduleErrorsRetainMinimumBackoffUnderNotifications(t *testing.T) {
	for _, scenario := range []string{"processing_error", "query_error", "past_deadline"} {
		t.Run(scenario, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				scheduler := &fakeTaskScheduler{wake: make(chan struct{}, 1)}
				if scenario == "query_error" {
					scheduler.err = errors.New("查询失败")
				}
				if scenario == "past_deadline" {
					past := time.Now().Add(-time.Second)
					scheduler.next = &past
				}
				// 模拟领取更新/查询期间收到自身通知，不能立即再次领取。
				scheduler.query = func() { scheduler.wake <- struct{}{} }
				waiter := newWorkerScheduleWait(scheduler, biz.TaskConsumerNotification, false, slog.New(slog.NewTextHandler(io.Discard, nil)))
				defer waiter.close()
				for attempt := 0; attempt < 3; attempt++ {
					finished := make(chan bool, 1)
					go func() { finished <- waiter.wait(context.Background(), scenario == "processing_error") }()
					synctest.Wait()
					for pulse := 0; pulse < 9; pulse++ {
						time.Sleep(100 * time.Millisecond)
						scheduler.wake <- struct{}{}
						synctest.Wait()
						select {
						case <-finished:
							t.Fatal("异常通知绕过了一秒最短退避")
						default:
						}
					}
					time.Sleep(100 * time.Millisecond)
					synctest.Wait()
					if !<-finished {
						t.Fatal("最短退避后未保留通知唤醒")
					}
				}
			})
		})
	}
}

func TestWorkerScheduleErrorBackoffCanStopImmediately(t *testing.T) {
	for _, stop := range []string{"cancel", "closed_subscription"} {
		t.Run(stop, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				scheduler := &fakeTaskScheduler{wake: make(chan struct{}, 1)}
				waiter := newWorkerScheduleWait(scheduler, biz.TaskConsumerNotification, false, slog.New(slog.NewTextHandler(io.Discard, nil)))
				defer waiter.close()
				finished := make(chan bool, 1)
				go func() { finished <- waiter.wait(ctx, true) }()
				synctest.Wait()
				start := time.Now()
				if stop == "cancel" {
					cancel()
				} else {
					close(scheduler.wake)
				}
				synctest.Wait()
				if <-finished || time.Now() != start {
					t.Fatal("异常退避停止不应等待计时器")
				}
			})
		})
	}
}
