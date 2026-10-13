package data

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestTaskScheduleBroadcastRoutingAndCoalescing(t *testing.T) {
	s := NewBackgroundTaskScheduler(nil, slog.New(slog.NewTextHandler(io.Discard, nil))).(*taskScheduler)
	a, closeA := s.Subscribe(biz.TaskConsumerNotification)
	defer closeA()
	b, closeB := s.Subscribe(biz.TaskConsumerNotification)
	defer closeB()
	other, closeOther := s.Subscribe(biz.TaskConsumerDingTalkDirectory)
	defer closeOther()
	for index := 0; index < 10; index++ {
		s.routeNotification("test", `{"schema":"test","consumer":"notification"}`)
	}
	for _, ch := range []<-chan struct{}{a, b} {
		select {
		case <-ch:
		default:
			t.Fatal("未广播给全部相同类别订阅者")
		}
		select {
		case <-ch:
			t.Fatal("重复通知未合并")
		default:
		}
	}
	s.routeNotification("test", `{"schema":"other","consumer":"notification"}`)
	s.routeNotification("test", `{"schema":"test","consumer":"unknown"}`)
	s.routeNotification("test", `{"schema":"test","consumer":""}`)
	for _, ch := range []<-chan struct{}{a, b, other} {
		select {
		case <-ch:
			t.Fatal("其他 Schema 或未知类别不应唤醒")
		default:
		}
	}
	closeA()
	closeA()
	s.broadcast(biz.TaskConsumerNotification)
	select {
	case <-a:
		t.Fatal("已退订仍收到通知")
	default:
	}
}

func TestTaskSchedulerStartsAndStopsWithoutSubscribers(t *testing.T) {
	s := NewBackgroundTaskScheduler(nil, nil)
	go s.Start(context.Background())
	if err := s.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	ch, closeSubscription := s.Subscribe(biz.TaskConsumerNotification)
	defer closeSubscription()
	if _, open := <-ch; open {
		t.Fatal("停止后订阅应立即结束")
	}
}
