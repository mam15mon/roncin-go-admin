package biz

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type notificationRepoStub struct {
	delivery *NotificationDelivery
	err      error
}

func (s *notificationRepoStub) FindByTaskID(context.Context, uuid.UUID) (*NotificationDelivery, error) {
	return s.delivery, s.err
}

type dingTalkNotificationSenderStub struct {
	enabled bool
	userID  string
	content string
	err     error
}

func (s *dingTalkNotificationSenderStub) Enabled() bool { return s.enabled }

func (s *dingTalkNotificationSenderStub) SendText(_ context.Context, userID, content string) error {
	s.userID = userID
	s.content = content
	return s.err
}

func TestNotificationProcessNextCompletesDingTalkTask(t *testing.T) {
	organizationID := uuid.New()
	taskID := uuid.New()
	leaseToken := "lease-token"
	taskRepo := &backgroundTaskRepoStub{claimTask: &BackgroundTask{
		ID: taskID, OrganizationID: organizationID, Kind: BackgroundTaskKindDingTalkNotice,
		Status: BackgroundTaskStatusRunning, LeaseToken: &leaseToken,
	}}
	sender := &dingTalkNotificationSenderStub{enabled: true}
	usecase := NewNotificationUsecase(NewBackgroundTaskUsecase(taskRepo), &notificationRepoStub{delivery: &NotificationDelivery{
		RecipientUserID: uuid.New(), DingTalkUserID: "ding-user-id", Channel: NotificationChannelDingTalk,
		Template: NotificationTemplateOrderPersonnelAssign, ResourceType: "ORDER", ResourceID: uuid.New(),
		ReferenceCode: "SE202608280001", Parameter: string(OrderPersonnelRoleDocument),
	}}, sender)

	if err := usecase.ProcessNext(context.Background(), 30*time.Second); err != nil {
		t.Fatalf("ProcessNext() error = %v", err)
	}
	if sender.userID != "ding-user-id" || !strings.Contains(sender.content, "SE202608280001") || !strings.Contains(sender.content, "单证人员") {
		t.Fatalf("发送内容错误: userID=%q content=%q", sender.userID, sender.content)
	}
	if taskRepo.completeOrgID != organizationID || taskRepo.completeID != taskID || taskRepo.completeLeaseToken != leaseToken {
		t.Fatalf("任务完成参数错误: %#v", taskRepo)
	}
}

func TestNotificationProcessNextRecordsRetryOnSendFailure(t *testing.T) {
	organizationID := uuid.New()
	taskID := uuid.New()
	leaseToken := "lease-token"
	taskRepo := &backgroundTaskRepoStub{claimTask: &BackgroundTask{
		ID: taskID, OrganizationID: organizationID, Kind: BackgroundTaskKindDingTalkNotice,
		Status: BackgroundTaskStatusRunning, LeaseToken: &leaseToken, Attempts: 1,
	}}
	sender := &dingTalkNotificationSenderStub{enabled: true, err: context.DeadlineExceeded}
	usecase := NewNotificationUsecase(NewBackgroundTaskUsecase(taskRepo), &notificationRepoStub{delivery: &NotificationDelivery{
		RecipientUserID: uuid.New(), DingTalkUserID: "ding-user-id", Channel: NotificationChannelDingTalk,
		Template: NotificationTemplateOrderPersonnelAssign, ResourceType: "ORDER", ResourceID: uuid.New(),
		ReferenceCode: "SE202608280002", Parameter: string(OrderPersonnelRoleOperator),
	}}, sender)
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	usecase.now = func() time.Time { return now }

	if err := usecase.ProcessNext(context.Background(), 30*time.Second); err == nil {
		t.Fatal("ProcessNext() expected send error")
	}
	if taskRepo.failID != taskID || taskRepo.failLeaseToken != leaseToken || !strings.Contains(taskRepo.failErrorMessage, "context deadline exceeded") {
		t.Fatalf("任务失败参数错误: %#v", taskRepo)
	}
	if want := now.Add(time.Minute); !taskRepo.failNextRunAt.Equal(want) {
		t.Fatalf("next run = %s, want %s", taskRepo.failNextRunAt, want)
	}
}

var _ NotificationRepo = (*notificationRepoStub)(nil)
var _ DingTalkNotificationSender = (*dingTalkNotificationSenderStub)(nil)
