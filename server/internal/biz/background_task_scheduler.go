package biz

import (
	"context"
	"time"
)

// TaskConsumer 表示已有后台消费者，不携带任务或人员资料。
type TaskConsumer string

const (
	TaskConsumerNotification      TaskConsumer = "notification"
	TaskConsumerObjectDeletion    TaskConsumer = "object_deletion"
	TaskConsumerDingTalkApproval  TaskConsumer = "dingtalk_approval"
	TaskConsumerDingTalkDirectory TaskConsumer = "dingtalk_directory"
)

// BackgroundTaskScheduler 只负责唤醒与下次检查时间，任务领取仍由原用例负责。
// Subscribe 必须在首次领取前调用；释放函数可重复调用，信号允许合并。
type BackgroundTaskScheduler interface {
	Start(context.Context) error
	Stop(context.Context) error
	Subscribe(TaskConsumer) (<-chan struct{}, func())
	NextRun(context.Context, TaskConsumer, bool) (*time.Time, error)
}
