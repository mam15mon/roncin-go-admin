package server

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/data"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

type acceptanceTaskRepo struct {
	biz.BackgroundTaskRepo
	claims  atomic.Int64
	claimed chan struct{}
}

func (r *acceptanceTaskRepo) ClaimAny(ctx context.Context, kinds []biz.BackgroundTaskKind, lease time.Duration, now time.Time) (*biz.BackgroundTask, error) {
	r.claims.Add(1)
	select {
	case r.claimed <- struct{}{}:
	default:
	}
	return r.BackgroundTaskRepo.ClaimAny(ctx, kinds, lease, now)
}

type acceptanceScheduler struct {
	biz.BackgroundTaskScheduler
	queries atomic.Int64
	queried chan struct{}
}

func (s *acceptanceScheduler) NextRun(ctx context.Context, consumer biz.TaskConsumer, inbox bool) (*time.Time, error) {
	s.queries.Add(1)
	next, err := s.BackgroundTaskScheduler.NextRun(ctx, consumer, inbox)
	select {
	case s.queried <- struct{}{}:
	default:
	}
	return next, err
}

type acceptanceNotificationSender struct{}

func (acceptanceNotificationSender) Enabled() bool { return true }
func (acceptanceNotificationSender) SendMessage(context.Context, string, biz.DingTalkMessage) error {
	return nil
}

type acceptanceListenerLog struct {
	slog.Handler
	ready chan struct{}
}

func (h acceptanceListenerLog) Handle(ctx context.Context, r slog.Record) error {
	if r.Message == "后台任务通知监听已启动" {
		select {
		case h.ready <- struct{}{}:
		default:
		}
	}
	return h.Handler.Handle(ctx, r)
}

// 本验收使用真实时钟与真实 PostgreSQL；只允许显式创建的临时数据库。
func TestWorkerScheduleRealTimeAcceptance(t *testing.T) {
	if os.Getenv("RONCIN_TASK_REALTIME_ACCEPTANCE") != "1" {
		t.Skip("未开启真实时间后台任务验收")
	}
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	u, err := url.Parse(source)
	if err != nil || !strings.HasPrefix(strings.TrimPrefix(u.Path, "/"), "roncin_notify_verify_") {
		t.Fatal("真实时间验收必须指向 roncin_notify_verify_ 临时数据库")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db, err := sql.Open("pgx", source)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migration.Apply(ctx, db, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	ready := make(chan struct{}, 1)
	logger := slog.New(acceptanceListenerLog{Handler: slog.NewJSONHandler(io.Discard, nil), ready: ready})
	d, cleanup, err := data.NewData(&conf.Data{Database: &conf.Data_Database{Driver: "postgres", Source: source}}, logger)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	scheduler := &acceptanceScheduler{BackgroundTaskScheduler: data.NewBackgroundTaskScheduler(d, logger), queried: make(chan struct{}, 16)}
	repo := &acceptanceTaskRepo{BackgroundTaskRepo: data.NewBackgroundTaskRepo(d), claimed: make(chan struct{}, 16)}
	worker := NewNotificationWorker(biz.NewNotificationUsecase(biz.NewBackgroundTaskUsecase(repo), data.NewNotificationRepo(d), acceptanceNotificationSender{}), logger, scheduler)
	go func() { _ = scheduler.Start(ctx) }()
	go func() { _ = worker.Start(ctx) }()
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		if err := worker.Stop(stopCtx); err != nil {
			t.Errorf("停止 worker: %v", err)
		}
		if err := scheduler.Stop(stopCtx); err != nil {
			t.Errorf("停止监听: %v", err)
		}
	}()
	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal("监听未就绪")
	}
	select {
	case <-scheduler.queried:
	case <-ctx.Done():
		t.Fatal("初始检查未完成")
	}
	// 初始 LISTEN 广播可以和首次扫描交错；等待它完成后再统计恢复周期。
	time.Sleep(200 * time.Millisecond)
	initialClaims, initialQueries := repo.claims.Load(), scheduler.queries.Load()
	start := time.Now()
	timer := time.NewTimer(65 * time.Second)
	select {
	case <-timer.C:
	case <-ctx.Done():
		timer.Stop()
		t.Fatal("验收超时")
	}
	claims, queries := repo.claims.Load()-initialClaims, scheduler.queries.Load()-initialQueries
	if claims != 1 || queries != 1 {
		t.Fatalf("65 秒内恢复检查应各一次，实际领取=%d，最近到期=%d", claims, queries)
	}
	t.Logf("真实 PostgreSQL 空闲观察 %s：领取查询 %d 次，最近到期查询 %d 次", time.Since(start).Round(time.Millisecond), claims, queries)
	for len(repo.claimed) > 0 {
		<-repo.claimed
	}
	commitStart := time.Now()
	if _, err := db.ExecContext(ctx, `SELECT pg_notify('roncin_task_schedule_changed', '{"schema":"public","consumer":"notification"}')`); err != nil {
		t.Fatal(err)
	}
	wakeTimeout := time.NewTimer(time.Second)
	defer wakeTimeout.Stop()
	select {
	case <-repo.claimed:
		t.Logf("提交调度通知至真实 worker 再次领取：%s", time.Since(commitStart))
	case <-wakeTimeout.C:
		t.Fatal("提交调度通知后一秒内未开始领取")
	}
}
