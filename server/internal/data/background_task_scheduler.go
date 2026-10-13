package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	task "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	inbox "github.com/roncin/roncin-go-admin/server/internal/data/ent/dingtalkapprovalinboxevent"
)

const taskScheduleChannel = "roncin_task_schedule_changed"

// taskScheduler 用一条专用连接服务本实例所有启用消费者；连接不参与业务事务。
type taskScheduler struct {
	data          *Data
	logger        *slog.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	done          chan struct{}
	changed       chan struct{}
	mu            sync.Mutex
	subscriptions map[chan struct{}]biz.TaskConsumer
	sessionCancel context.CancelFunc
	ready         bool
	listenerPID   uint32
}

func NewBackgroundTaskScheduler(data *Data, logger *slog.Logger) biz.BackgroundTaskScheduler {
	if logger == nil {
		logger = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &taskScheduler{data: data, logger: logger, ctx: ctx, cancel: cancel, done: make(chan struct{}), changed: make(chan struct{}, 1), subscriptions: make(map[chan struct{}]biz.TaskConsumer)}
}

func signalTaskSchedule(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

func (s *taskScheduler) Subscribe(consumer biz.TaskConsumer) (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	s.mu.Lock()
	if s.ctx.Err() == nil {
		s.subscriptions[ch] = consumer
		if s.ready {
			signalTaskSchedule(ch)
		}
		signalTaskSchedule(s.changed)
	} else {
		close(ch)
	}
	s.mu.Unlock()
	var once sync.Once
	return ch, func() {
		once.Do(func() {
			s.mu.Lock()
			delete(s.subscriptions, ch)
			if len(s.subscriptions) == 0 && s.sessionCancel != nil {
				s.sessionCancel()
			}
			signalTaskSchedule(s.changed)
			s.mu.Unlock()
		})
	}
}

func (s *taskScheduler) broadcast(consumer biz.TaskConsumer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch, target := range s.subscriptions {
		if consumer == "" || consumer == target {
			signalTaskSchedule(ch)
		}
	}
}

func (s *taskScheduler) Start(context.Context) error {
	defer close(s.done)
	for s.ctx.Err() == nil {
		s.mu.Lock()
		active := len(s.subscriptions) > 0
		var sessionCtx context.Context
		if active {
			sessionCtx, s.sessionCancel = context.WithCancel(s.ctx)
		}
		s.mu.Unlock()
		if !active {
			select {
			case <-s.ctx.Done():
				return nil
			case <-s.changed:
				continue
			}
		}
		s.listenSession(sessionCtx)
		s.mu.Lock()
		s.sessionCancel()
		s.sessionCancel = nil
		s.ready = false
		s.listenerPID = 0
		s.mu.Unlock()
	}
	return nil
}

func (s *taskScheduler) Stop(ctx context.Context) error {
	s.cancel()
	s.mu.Lock()
	for ch := range s.subscriptions {
		delete(s.subscriptions, ch)
		close(ch)
	}
	s.mu.Unlock()
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// 复用连接池已解析的 pgx 配置，也支持集成测试注册的 search_path 配置。
func (s *taskScheduler) connectionConfig(ctx context.Context) (*pgx.ConnConfig, error) {
	conn, err := s.data.sqlDB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	var config *pgx.ConnConfig
	err = conn.Raw(func(raw any) error {
		pgConn, ok := raw.(*stdlib.Conn)
		if !ok {
			return fmt.Errorf("后台调度监听要求 pgx 驱动")
		}
		config = pgConn.Conn().Config().Copy()
		// Config 来自已打开连接，OnNotification 绑定了原连接的通知缓冲；专用连接须重新绑定自身缓冲。
		config.OnNotification = nil
		config.RuntimeParams["application_name"] = "roncin_task_listener"
		return nil
	})
	return config, err
}

func (s *taskScheduler) listenSession(ctx context.Context) {
	backoff := time.Second
	disconnected := false
	for ctx.Err() == nil {
		setupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		config, err := s.connectionConfig(setupCtx)
		var conn *pgx.Conn
		var schema string
		if err == nil {
			conn, err = pgx.ConnectConfig(setupCtx, config)
		}
		if err == nil {
			err = conn.QueryRow(setupCtx, "SELECT current_schema()").Scan(&schema)
		}
		if err == nil {
			_, err = conn.Exec(setupCtx, "LISTEN "+taskScheduleChannel)
		}
		cancel()
		if err == nil {
			s.mu.Lock()
			s.ready = true
			s.listenerPID = conn.PgConn().PID()
			s.mu.Unlock()
			if disconnected {
				s.logger.Info("后台任务通知监听已恢复")
			} else {
				s.logger.Info("后台任务通知监听已启动")
			}
			disconnected = false
			backoff = time.Second
			// LISTEN 提交之后扫描，覆盖监听建立期间的写入和断线期间积压。
			s.broadcast("")
			for ctx.Err() == nil {
				notification, waitErr := conn.WaitForNotification(ctx)
				if waitErr != nil {
					err = waitErr
					break
				}
				s.routeNotification(schema, notification.Payload)
			}
		}
		s.mu.Lock()
		s.ready = false
		s.listenerPID = 0
		s.mu.Unlock()
		if conn != nil {
			closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = conn.Close(closeCtx)
			closeCancel()
		}
		if ctx.Err() != nil {
			return
		}
		if !disconnected {
			// pgx 连接错误可能携带连接目标，不输出原始错误或连接配置。
			s.logger.Warn("后台任务通知监听中断，等待重连")
			disconnected = true
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		backoff = min(backoff*2, 30*time.Second)
	}
}

func (s *taskScheduler) routeNotification(schema, payload string) {
	if len(payload) > 512 {
		return
	}
	var route struct {
		Schema   string           `json:"schema"`
		Consumer biz.TaskConsumer `json:"consumer"`
	}
	if json.Unmarshal([]byte(payload), &route) != nil || route.Schema != schema {
		return
	}
	switch route.Consumer {
	case biz.TaskConsumerNotification, biz.TaskConsumerObjectDeletion, biz.TaskConsumerDingTalkApproval, biz.TaskConsumerDingTalkDirectory:
		s.broadcast(route.Consumer)
	}
}

func taskConsumerKind(consumer biz.TaskConsumer) (task.Kind, error) {
	switch consumer {
	case biz.TaskConsumerNotification:
		return task.KindDINGTALK_NOTIFICATION, nil
	case biz.TaskConsumerObjectDeletion:
		return task.KindOBJECT_STORAGE_DELETION, nil
	case biz.TaskConsumerDingTalkApproval:
		return task.KindDINGTALK_APPROVAL_CREATE, nil
	case biz.TaskConsumerDingTalkDirectory:
		return task.KindDINGTALK_DIRECTORY_SYNC, nil
	default:
		return "", errors.New("未知后台任务消费者")
	}
}

// NextRun 与领取条件一致：未执行/失败看下次执行时间，执行中看租约到期；终态排除。
// 聚合 SQL 集中在 data，以免全量加载任务或将 PostgreSQL 细节暴露给 worker。
func (s *taskScheduler) NextRun(ctx context.Context, consumer biz.TaskConsumer, includeInbox bool) (*time.Time, error) {
	kind, err := taskConsumerKind(consumer)
	if err != nil {
		return nil, err
	}
	client, err := s.data.client(ctx)
	if err != nil {
		return nil, err
	}
	var times []struct {
		NextRunAt *time.Time `json:"next_run_at"`
	}
	err = client.BackgroundTask.Query().Where(task.KindEQ(kind), task.StatusIn(task.StatusPENDING, task.StatusFAILED, task.StatusRUNNING)).
		Aggregate(func(selector *entsql.Selector) string {
			return entsql.As("MIN(CASE WHEN "+selector.C(task.FieldStatus)+" = 'RUNNING' THEN "+selector.C(task.FieldLeaseExpiresAt)+" ELSE "+selector.C(task.FieldNextRunAt)+" END)", "next_run_at")
		}).Scan(ctx, &times)
	if err != nil {
		return nil, err
	}
	next := times[0].NextRunAt
	if consumer == biz.TaskConsumerDingTalkApproval && includeInbox {
		times = nil
		err = client.DingTalkApprovalInboxEvent.Query().Where(inbox.StatusIn(inbox.StatusRECEIVED, inbox.StatusFAILED, inbox.StatusPROCESSING)).
			Aggregate(func(selector *entsql.Selector) string {
				return entsql.As("MIN(CASE WHEN "+selector.C(inbox.FieldStatus)+" = 'PROCESSING' THEN "+selector.C(inbox.FieldProcessingExpiresAt)+" ELSE "+selector.C(inbox.FieldNextRunAt)+" END)", "next_run_at")
			}).Scan(ctx, &times)
		if err != nil {
			return nil, err
		}
		if inboxNext := times[0].NextRunAt; inboxNext != nil && (next == nil || inboxNext.Before(*next)) {
			next = inboxNext
		}
	}
	return next, nil
}

var _ biz.BackgroundTaskScheduler = (*taskScheduler)(nil)
