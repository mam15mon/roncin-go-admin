package data

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	task "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	inbox "github.com/roncin/roncin-go-admin/server/internal/data/ent/dingtalkapprovalinboxevent"
	orgent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

func awaitTaskWake(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	timer := time.NewTimer(4 * time.Second)
	defer timer.Stop()
	select {
	case <-ch:
	case <-timer.C:
		t.Fatal("未收到调度唤醒")
	}
}
func assertNoTaskWake(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ch:
		t.Fatal("不应收到调度唤醒")
	case <-timer.C:
	}
}

func TestBackgroundTaskSchedulerPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := d.client(ctx)
	if err != nil {
		t.Fatal(err)
	}
	org, err := client.Organization.Create().SetCode("SCHEDULE").SetName("调度测试").SetKind(orgent.KindSystem).SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	actor, err := client.User.Create().SetDisplayName("调度人员").SetDingtalkUserid("schedule-user").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	scheduler := NewBackgroundTaskScheduler(d, slog.New(slog.NewTextHandler(io.Discard, nil))).(*taskScheduler)
	consumers := []biz.TaskConsumer{biz.TaskConsumerNotification, biz.TaskConsumerObjectDeletion, biz.TaskConsumerDingTalkApproval, biz.TaskConsumerDingTalkDirectory}
	wakes := make(map[biz.TaskConsumer]<-chan struct{})
	closes := make([]func(), 0)
	for _, consumer := range consumers {
		ch, closeSub := scheduler.Subscribe(consumer)
		wakes[consumer] = ch
		closes = append(closes, closeSub)
	}
	go scheduler.Start(context.Background())
	defer func() {
		for _, closeSub := range closes {
			closeSub()
		}
		stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second)
		defer stopCancel()
		if err := scheduler.Stop(stopCtx); err != nil {
			t.Error(err)
		}
	}()
	for _, ch := range wakes {
		awaitTaskWake(t, ch)
	}
	tasks := NewBackgroundTaskRepo(d)

	t.Run("提交与回滚及独立写入口", func(t *testing.T) {
		rollback := errors.New("主动回滚")
		err := d.WithinTransaction(ctx, func(txCtx context.Context) error {
			_, err := tasks.Enqueue(txCtx, org.ID, &biz.BackgroundTask{Kind: biz.BackgroundTaskKindDingTalkNotice, IdempotencyKey: "rollback", MaxAttempts: 3, NextRunAt: time.Now()})
			if err != nil {
				return err
			}
			assertNoTaskWake(t, wakes[biz.TaskConsumerNotification])
			return rollback
		})
		if !errors.Is(err, rollback) {
			t.Fatal(err)
		}
		assertNoTaskWake(t, wakes[biz.TaskConsumerNotification])
		started := time.Now()
		if err := d.WithTx(ctx, func(tx *ent.Tx) error {
			return enqueueDingTalkUserAuthorizedNotification(ctx, tx, org.ID, actor, biz.NewDingTalkUserAuthorizedNotification(actor.ID))
		}); err != nil {
			t.Fatal(err)
		}
		awaitTaskWake(t, wakes[biz.TaskConsumerNotification])
		if time.Since(started) > time.Second {
			t.Fatal("正常通知延迟超过 1 秒")
		}
		if err := d.WithTx(ctx, func(tx *ent.Tx) error { return enqueueObjectStorageDeletion(ctx, tx, org.ID, "schedule/object") }); err != nil {
			t.Fatal(err)
		}
		awaitTaskWake(t, wakes[biz.TaskConsumerObjectDeletion])
		if _, err := NewDingTalkDirectoryRepo(d).Start(ctx, org.ID, actor.ID, "schedule-corp"); err != nil {
			t.Fatal(err)
		}
		awaitTaskWake(t, wakes[biz.TaskConsumerDingTalkDirectory])
		if _, err := tasks.Enqueue(ctx, org.ID, &biz.BackgroundTask{Kind: biz.BackgroundTaskKindDingTalkApproval, IdempotencyKey: "dispatch", MaxAttempts: 3, NextRunAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
		awaitTaskWake(t, wakes[biz.TaskConsumerDingTalkApproval])
		event := &biz.DingTalkApprovalCallbackEvent{EventID: "schedule-event", CorpID: "corp", EventType: "bpms_instance_change", ProcessInstanceID: "schedule-instance", EncryptedPayloadHash: "hash"}
		if err := NewDingTalkApprovalRepo(d).StoreCallback(ctx, event); err != nil {
			t.Fatal(err)
		}
		awaitTaskWake(t, wakes[biz.TaskConsumerDingTalkApproval])
	})

	t.Run("最近时间终态与收件箱", func(t *testing.T) {
		if _, err := client.BackgroundTask.Update().SetStatus(task.StatusSUCCEEDED).ClearLeaseExpiresAt().Save(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := client.DingTalkApprovalInboxEvent.Update().SetStatus(inbox.StatusPROCESSED).Save(ctx); err != nil {
			t.Fatal(err)
		}
		due := time.Now().Add(300 * time.Millisecond)
		row, err := client.BackgroundTask.Create().SetOrganizationID(org.ID).SetKind(task.KindDINGTALK_APPROVAL_CREATE).SetIdempotencyKey("future").SetMaxAttempts(3).SetNextRunAt(due).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		next, err := scheduler.NextRun(ctx, biz.TaskConsumerDingTalkApproval, true)
		if err != nil || next == nil || next.Sub(due).Abs() > time.Microsecond {
			t.Fatalf("未来调度时间=%v error=%v", next, err)
		}
		if _, err := tasks.ClaimAny(ctx, []biz.BackgroundTaskKind{biz.BackgroundTaskKindDingTalkApproval}, time.Second, time.Now()); !errors.Is(err, biz.ErrBackgroundTaskNoTask) {
			t.Fatalf("未来任务不应提前领取: %v", err)
		}
		timer := time.NewTimer(time.Until(*next))
		defer timer.Stop()
		<-timer.C
		claimed, err := tasks.ClaimAny(ctx, []biz.BackgroundTaskKind{biz.BackgroundTaskKindDingTalkApproval}, time.Second, time.Now())
		if err != nil || claimed.ID != row.ID {
			t.Fatalf("到期领取失败: %v", err)
		}
		retryDue := time.Now().Add(200 * time.Millisecond)
		if _, err := tasks.Fail(ctx, org.ID, row.ID, *claimed.LeaseToken, "定向验证重试", retryDue); err != nil {
			t.Fatal(err)
		}
		next, err = scheduler.NextRun(ctx, biz.TaskConsumerDingTalkApproval, false)
		if err != nil || next == nil || next.Sub(retryDue).Abs() > time.Microsecond {
			t.Fatalf("失败重试调度时间=%v error=%v", next, err)
		}
		if _, err := tasks.ClaimAny(ctx, []biz.BackgroundTaskKind{biz.BackgroundTaskKindDingTalkApproval}, time.Second, time.Now()); !errors.Is(err, biz.ErrBackgroundTaskNoTask) {
			t.Fatalf("失败任务不应提前重试: %v", err)
		}
		timer.Reset(time.Until(*next))
		<-timer.C
		claimed, err = tasks.ClaimAny(ctx, []biz.BackgroundTaskKind{biz.BackgroundTaskKindDingTalkApproval}, time.Second, time.Now())
		if err != nil || claimed.ID != row.ID || claimed.Attempts != 1 {
			t.Fatalf("失败任务到期未重新领取: %v task=%v", err, claimed)
		}
		oldToken := *claimed.LeaseToken
		leaseDue := time.Now().Add(200 * time.Millisecond)
		if _, err := client.BackgroundTask.UpdateOneID(row.ID).SetLeaseExpiresAt(leaseDue).Save(ctx); err != nil {
			t.Fatal(err)
		}
		next, err = scheduler.NextRun(ctx, biz.TaskConsumerDingTalkApproval, false)
		if err != nil || next == nil || next.Sub(leaseDue).Abs() > time.Microsecond {
			t.Fatalf("租约到期查询=%v error=%v", next, err)
		}
		if _, err := tasks.ClaimAny(ctx, []biz.BackgroundTaskKind{biz.BackgroundTaskKindDingTalkApproval}, time.Second, time.Now()); !errors.Is(err, biz.ErrBackgroundTaskNoTask) {
			t.Fatalf("租约未过期不应再次领取: %v", err)
		}
		timer.Reset(time.Until(*next))
		<-timer.C
		claimed, err = tasks.ClaimAny(ctx, []biz.BackgroundTaskKind{biz.BackgroundTaskKindDingTalkApproval}, time.Second, time.Now())
		if err != nil || claimed.ID != row.ID || claimed.Attempts != 2 || *claimed.LeaseToken == oldToken {
			t.Fatalf("过期租约未重新领取并换 token: %v task=%v", err, claimed)
		}
		if _, err := client.BackgroundTask.UpdateOneID(row.ID).SetStatus(task.StatusSUCCEEDED).Save(ctx); err != nil {
			t.Fatal(err)
		}
		processingDue := time.Now().Add(100 * time.Millisecond)
		if _, err := client.DingTalkApprovalInboxEvent.Update().SetStatus(inbox.StatusPROCESSING).SetProcessingToken("old-lease").SetProcessingExpiresAt(processingDue).Save(ctx); err != nil {
			t.Fatal(err)
		}
		next, err = scheduler.NextRun(ctx, biz.TaskConsumerDingTalkApproval, false)
		if err != nil || next != nil {
			t.Fatalf("回调关闭时不得读取 inbox: %v %v", next, err)
		}
		next, err = scheduler.NextRun(ctx, biz.TaskConsumerDingTalkApproval, true)
		if err != nil || next == nil || next.Sub(processingDue).Abs() > time.Microsecond {
			t.Fatalf("收件箱租约查询=%v error=%v", next, err)
		}
		timer.Reset(time.Until(*next))
		<-timer.C
		inboxJob, err := NewDingTalkApprovalRepo(d).ClaimInbox(ctx, time.Second, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if inboxJob.LeaseToken == "old-lease" {
			t.Fatal("过期 inbox 领取未更换有效 token")
		}
	})

	t.Run("两个实例通知与领取竞争", func(t *testing.T) {
		second := NewBackgroundTaskScheduler(d, nil)
		ch, closeSub := second.Subscribe(biz.TaskConsumerObjectDeletion)
		go second.Start(context.Background())
		defer func() {
			closeSub()
			if err := second.Stop(ctx); err != nil {
				t.Error(err)
			}
		}()
		awaitTaskWake(t, ch)
		created, err := tasks.Enqueue(ctx, org.ID, &biz.BackgroundTask{Kind: biz.BackgroundTaskKindObjectStorageDelete, IdempotencyKey: "compete", MaxAttempts: 3, NextRunAt: time.Now()})
		if err != nil {
			t.Fatal(err)
		}
		awaitTaskWake(t, ch)
		var wg sync.WaitGroup
		results := make(chan *biz.BackgroundTask, 2)
		errorsCh := make(chan error, 2)
		for index := 0; index < 2; index++ {
			wg.Go(func() {
				value, err := tasks.ClaimAny(ctx, []biz.BackgroundTaskKind{biz.BackgroundTaskKindObjectStorageDelete}, time.Minute, time.Now())
				results <- value
				errorsCh <- err
			})
		}
		wg.Wait()
		close(results)
		close(errorsCh)
		successes := 0
		for value := range results {
			if value != nil {
				successes++
				if value.ID != created.ID {
					t.Fatal("领取了非预期任务")
				}
			}
		}
		for err := range errorsCh {
			if err != nil && !errors.Is(err, biz.ErrBackgroundTaskNoTask) {
				t.Fatal(err)
			}
		}
		if successes != 1 {
			t.Fatalf("有效租约应恰有一个，实际%d", successes)
		}
	})

	t.Run("监听断线重连扫描", func(t *testing.T) {
		// 消耗旧信号后终止精确本 Schema 的监听连接，恢复必须重新广播扫描。
		for _, ch := range wakes {
			select {
			case <-ch:
			default:
			}
		}
		scheduler.mu.Lock()
		pid := scheduler.listenerPID
		scheduler.mu.Unlock()
		if pid == 0 {
			t.Fatal("监听连接尚未就绪")
		}
		var terminated bool
		if err := d.sqlDB.QueryRowContext(ctx, "SELECT pg_terminate_backend($1)", pid).Scan(&terminated); err != nil || !terminated {
			t.Fatalf("终止监听失败: %v", err)
		}
		if _, err := tasks.Enqueue(ctx, org.ID, &biz.BackgroundTask{Kind: biz.BackgroundTaskKindDingTalkNotice, IdempotencyKey: uuid.NewString(), MaxAttempts: 3, NextRunAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
		// 无关类别同样获得恢复信号，证明重连扫描而非依赖断线期间通知。
		awaitTaskWake(t, wakes[biz.TaskConsumerDingTalkDirectory])
		next, err := scheduler.NextRun(ctx, biz.TaskConsumerNotification, false)
		if err != nil || next == nil {
			t.Fatalf("断线期间任务未恢复: %v", err)
		}
	})
}

func TestBackgroundTaskSchedulerMigrationUpgrade(t *testing.T) {
	d, cleanup := getIntegrationDataBeforeMigration(t, "20261016090000_background_task_notify.sql")
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var before int
	if err := d.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM pg_trigger WHERE tgrelid IN ('background_tasks'::regclass,'ding_talk_approval_inbox_events'::regclass) AND NOT tgisinternal`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if before != 0 {
		t.Fatalf("前态通知触发器=%d", before)
	}
	for attempt := 0; attempt < 2; attempt++ {
		if err := migration.Apply(ctx, d.sqlDB, "../../migrations"); err != nil {
			t.Fatal(err)
		}
	}
	var triggers, revisions int
	if err := d.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM pg_trigger WHERE tgrelid IN ('background_tasks'::regclass,'ding_talk_approval_inbox_events'::regclass) AND NOT tgisinternal`).Scan(&triggers); err != nil {
		t.Fatal(err)
	}
	if err := d.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations WHERE version='20261016090000_background_task_notify'`).Scan(&revisions); err != nil {
		t.Fatal(err)
	}
	if triggers != 2 || revisions != 1 {
		t.Fatalf("通知迁移应只有两触发器且一次记录，triggers=%d revisions=%d", triggers, revisions)
	}
}

func TestBackgroundTaskSchedulerResubscribeAndStopPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	scheduler := NewBackgroundTaskScheduler(d, nil).(*taskScheduler)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go scheduler.Start(context.Background())
	defer func() {
		if err := scheduler.Stop(ctx); err != nil {
			t.Error(err)
		}
	}()
	first, releaseFirst := scheduler.Subscribe(biz.TaskConsumerNotification)
	awaitTaskWake(t, first)
	scheduler.mu.Lock()
	oldPID := scheduler.listenerPID
	scheduler.mu.Unlock()
	releaseFirst()
	releaseFirst()
	second, releaseSecond := scheduler.Subscribe(biz.TaskConsumerDingTalkDirectory)
	defer releaseSecond()
	// 即刻重订阅可能先收到旧监听就绪信号，再收到新连接就绪广播。
	for {
		awaitTaskWake(t, second)
		scheduler.mu.Lock()
		newPID := scheduler.listenerPID
		scheduler.mu.Unlock()
		if newPID != 0 && newPID != oldPID {
			break
		}
	}
	if err := scheduler.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case _, open := <-second:
		if open { // Stop 前缓冲的扫描信号可合法残留，再读应关闭。
			if _, stillOpen := <-second; stillOpen {
				t.Fatal("Stop 后等待未关闭")
			}
		}
	case <-ctx.Done():
		t.Fatal("Stop 未解除订阅等待")
	}
}

func TestBackgroundTaskSchedulerReconnectCancellationPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	scheduler := NewBackgroundTaskScheduler(d, nil).(*taskScheduler)
	ch, release := scheduler.Subscribe(biz.TaskConsumerNotification)
	defer release()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go scheduler.Start(context.Background())
	awaitTaskWake(t, ch)
	scheduler.mu.Lock()
	pid := scheduler.listenerPID
	scheduler.mu.Unlock()
	var terminated bool
	if err := d.sqlDB.QueryRowContext(ctx, "SELECT pg_terminate_backend($1)", pid).Scan(&terminated); err != nil || !terminated {
		t.Fatalf("终止监听失败: %v", err)
	}
	// Stop 不依赖重连计时器到期或 worker 的停止先后顺序。
	if err := scheduler.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	var listeners int
	if err := d.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM pg_stat_activity WHERE pid=$1 AND application_name='roncin_task_listener'`, pid).Scan(&listeners); err != nil {
		t.Fatal(err)
	}
	if listeners != 0 {
		t.Fatal("停止后遗留监听连接")
	}
}

func TestBackgroundTaskSchedulerSchemaAndUpdatePostgres(t *testing.T) {
	firstData, firstCleanup := getIntegrationData(t)
	defer firstCleanup()
	secondData, secondCleanup := getIntegrationData(t)
	defer secondCleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	start := func(d *Data) (*taskScheduler, <-chan struct{}) {
		t.Helper()
		s := NewBackgroundTaskScheduler(d, slog.New(slog.NewTextHandler(io.Discard, nil))).(*taskScheduler)
		ch, release := s.Subscribe(biz.TaskConsumerNotification)
		go s.Start(context.Background())
		t.Cleanup(func() {
			release()
			stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second)
			defer stopCancel()
			if err := s.Stop(stopCtx); err != nil {
				t.Error(err)
			}
		})
		awaitTaskWake(t, ch)
		return s, ch
	}
	_, firstWake := start(firstData)
	_, secondWake := start(secondData)
	client, err := secondData.client(ctx)
	if err != nil {
		t.Fatal(err)
	}
	org, err := client.Organization.Create().SetCode("UPDATE").SetName("通知更新测试").SetKind(orgent.KindSystem).SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	row, err := client.BackgroundTask.Create().SetOrganizationID(org.ID).SetKind(task.KindDINGTALK_NOTIFICATION).SetIdempotencyKey("schedule-update").SetMaxAttempts(3).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	awaitTaskWake(t, secondWake)
	assertNoTaskWake(t, firstWake)
	// 无关字段和调度字段相同值更新都不能造成新的空闲扫描。
	if _, err := client.BackgroundTask.UpdateOneID(row.ID).SetLastError("验证非调度字段").Save(ctx); err != nil {
		t.Fatal(err)
	}
	assertNoTaskWake(t, secondWake)
	if _, err := client.BackgroundTask.UpdateOneID(row.ID).SetNextRunAt(row.NextRunAt).SetStatus(row.Status).Save(ctx); err != nil {
		t.Fatal(err)
	}
	assertNoTaskWake(t, secondWake)
	if _, err := client.BackgroundTask.UpdateOneID(row.ID).SetNextRunAt(time.Now().Add(time.Second)).Save(ctx); err != nil {
		t.Fatal(err)
	}
	awaitTaskWake(t, secondWake)
	assertNoTaskWake(t, firstWake)
	if _, err := client.BackgroundTask.UpdateOneID(row.ID).SetStatus(task.StatusRUNNING).SetLeaseToken("schedule-lease").SetLeaseExpiresAt(time.Now().Add(time.Second)).Save(ctx); err != nil {
		t.Fatal(err)
	}
	awaitTaskWake(t, secondWake)
	if _, err := client.BackgroundTask.UpdateOneID(row.ID).SetStatus(task.StatusSUCCEEDED).ClearLeaseToken().ClearLeaseExpiresAt().Save(ctx); err != nil {
		t.Fatal(err)
	}
	awaitTaskWake(t, secondWake)
	assertNoTaskWake(t, firstWake)
}
