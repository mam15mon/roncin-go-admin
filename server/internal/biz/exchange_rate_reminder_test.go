package biz

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type reminderRepoStub struct {
	ExchangeRateReminderRepo
	unsynced    []*ExchangeRateReminderOrganization
	recipients  []*ExchangeRateReminderRecipient
	enqueued    []*ExchangeRateReminderIntent
	recipientBy map[uuid.UUID][]*ExchangeRateReminderRecipient
}

func (s *reminderRepoStub) ListUnsyncedWeeklyOrganizations(context.Context, time.Time, time.Time) ([]*ExchangeRateReminderOrganization, error) {
	return s.unsynced, nil
}

func (s *reminderRepoStub) ListFinanceReminderRecipients(_ context.Context, organizationID uuid.UUID) ([]*ExchangeRateReminderRecipient, error) {
	if s.recipientBy != nil {
		return s.recipientBy[organizationID], nil
	}
	return s.recipients, nil
}

func (s *reminderRepoStub) EnqueueWeeklyReminders(_ context.Context, intents []*ExchangeRateReminderIntent) error {
	s.enqueued = append(s.enqueued, intents...)
	return nil
}

// TestRunWeeklyReminderCheckEnqueuesOncePerOrgPerWeek 验证督办检测：当周未同步的
// 分公司向其汇率维护人员入队通知；重复检查按确定性 ID 幂等（同周期同收件人同 ID）。
func TestRunWeeklyReminderCheckEnqueuesOncePerOrgPerWeek(t *testing.T) {
	branch := uuid.Must(uuid.NewV7())
	finance := uuid.Must(uuid.NewV7())
	repo := &reminderRepoStub{
		unsynced: []*ExchangeRateReminderOrganization{{OrganizationID: branch, Name: "深圳分公司", BaseCurrency: "CNY"}},
		recipientBy: map[uuid.UUID][]*ExchangeRateReminderRecipient{
			branch: {{UserID: finance}},
		},
	}
	usecase := NewExchangeRateReminderUsecase(repo)
	// 2026-09-14 是周一。
	now := time.Date(2026, 9, 14, 10, 5, 0, 0, ExchangeRateBusinessLocation())
	notified, err := usecase.RunWeeklyReminderCheck(context.Background(), now)
	if err != nil || notified != 1 {
		t.Fatalf("首次检查应入队 1 条通知: notified=%d err=%v", notified, err)
	}
	if len(repo.enqueued) != 1 || repo.enqueued[0].RecipientUserID != finance || repo.enqueued[0].OrganizationID != branch {
		t.Fatalf("通知意图不符: %#v", repo.enqueued)
	}
	if repo.enqueued[0].WeekFrom.Format("2006-01-02") != "2026-09-14" || repo.enqueued[0].WeekTo.Format("2006-01-02") != "2026-09-20" {
		t.Fatalf("督办目标周不符: %s ~ %s", repo.enqueued[0].WeekFrom, repo.enqueued[0].WeekTo)
	}
	firstID := repo.enqueued[0].ID
	// 同周期重复检查派生相同 ID（配合仓储 OnConflict DoNothing 不重发）。
	if _, err := usecase.RunWeeklyReminderCheck(context.Background(), now.Add(30*time.Minute)); err != nil {
		t.Fatalf("重复检查失败: %v", err)
	}
	if len(repo.enqueued) != 2 || repo.enqueued[1].ID != firstID {
		t.Fatalf("同周期重复检查应派生相同任务 ID: %#v", repo.enqueued)
	}
}

func TestRenderExchangeRateWeeklyReminder(t *testing.T) {
	organizationID := uuid.Must(uuid.NewV7())
	delivery := &NotificationDelivery{
		RecipientUserID: uuid.Must(uuid.NewV7()), DingTalkUserID: "boc-finance",
		Channel: NotificationChannelDingTalk, Template: NotificationTemplateExchangeRateWeeklyReminder,
		ResourceType: "ORGANIZATION", ResourceID: organizationID,
		ReferenceCode: "深圳分公司", Parameter: "CNY 2026-09-14~2026-09-20",
	}
	content, err := renderNotification(delivery)
	if err != nil {
		t.Fatalf("渲染督办通知失败: %v", err)
	}
	for _, fragment := range []string{"深圳分公司", "CNY 2026-09-14~2026-09-20", "暂沿用上周汇率"} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("督办通知缺少内容 %q: %s", fragment, content)
		}
	}
}
