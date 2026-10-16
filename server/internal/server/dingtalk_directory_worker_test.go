package server

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type idleDirectoryTaskRepo struct {
	biz.BackgroundTaskRepo
	polls chan time.Time
}

func (r *idleDirectoryTaskRepo) ClaimAny(context.Context, []biz.BackgroundTaskKind, time.Duration, time.Time) (*biz.BackgroundTask, error) {
	r.polls <- time.Now()
	return nil, biz.ErrBackgroundTaskNoTask
}

type enabledDirectoryGateway struct{ biz.DingTalkDirectoryGateway }

func (enabledDirectoryGateway) Enabled() bool { return true }

func TestDingTalkDirectoryWorkerWaitsForNotificationAndRecovery(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		repo := &idleDirectoryTaskRepo{polls: make(chan time.Time, 1)}
		tasks := biz.NewBackgroundTaskUsecase(repo)
		usecase := biz.NewDingTalkDirectoryUsecase(nil, enabledDirectoryGateway{}, tasks)
		scheduler := &fakeTaskScheduler{wake: make(chan struct{}, 1)}
		worker := NewDingTalkDirectoryWorker(usecase, slog.New(slog.NewTextHandler(io.Discard, nil)), scheduler)
		result := make(chan error, 1)
		go func() { result <- worker.Start(context.Background()) }()
		synctest.Wait()
		<-repo.polls
		time.Sleep(59 * time.Second)
		synctest.Wait()
		select {
		case <-repo.polls:
			t.Fatal("完全空闲期间发生高频查询")
		default:
		}
		time.Sleep(time.Second)
		synctest.Wait()
		<-repo.polls
		scheduler.wake <- struct{}{}
		synctest.Wait()
		<-repo.polls
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := worker.Stop(ctx); err != nil {
			t.Fatalf("停止 worker: %v", err)
		}
		if err := <-result; err != nil {
			t.Fatalf("worker 退出: %v", err)
		}
	})
}
