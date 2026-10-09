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

func TestDingTalkDirectoryWorkerKeepsPollingAndStops(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		repo := &idleDirectoryTaskRepo{polls: make(chan time.Time, 1)}
		tasks := biz.NewBackgroundTaskUsecase(repo)
		usecase := biz.NewDingTalkDirectoryUsecase(nil, enabledDirectoryGateway{}, tasks)
		worker := NewDingTalkDirectoryWorker(usecase, slog.New(slog.NewTextHandler(io.Discard, nil)))
		result := make(chan error, 1)
		go func() { result <- worker.Start(context.Background()) }()
		synctest.Wait()
		previous := <-repo.polls
		for index := 0; index < 8; index++ {
			time.Sleep(2 * time.Second)
			synctest.Wait()
			select {
			case current := <-repo.polls:
				if interval := current.Sub(previous); interval != 2*time.Second {
					t.Fatalf("空闲轮询间隔 = %s，期望固定 2 秒", interval)
				}
				previous = current
			default:
				t.Fatal("空闲 2 秒后未检查新任务")
			}
		}
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
