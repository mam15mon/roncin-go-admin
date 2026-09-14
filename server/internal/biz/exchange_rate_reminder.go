package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ExchangeRateReminderRepo 提供周汇率漏配督办的查询与通知入队。
type ExchangeRateReminderRepo interface {
	// ListUnsyncedWeeklyOrganizations 返回当周尚未同步本组织周汇率的启用分公司
	// （kind=company 且具备本币）：判定依据为该组织在目标周窗口内没有任何启用的
	// 组织汇率行。
	ListUnsyncedWeeklyOrganizations(ctx context.Context, weekFrom, weekTo time.Time) ([]*ExchangeRateReminderOrganization, error)
	// ListFinanceReminderRecipients 返回组织内持有汇率维护权限且绑定钉钉的启用用户。
	ListFinanceReminderRecipients(ctx context.Context, organizationID uuid.UUID) ([]*ExchangeRateReminderRecipient, error)
	// EnqueueWeeklyReminders 幂等入队督办通知（确定性任务 ID，重复检查不重发）。
	EnqueueWeeklyReminders(ctx context.Context, intents []*ExchangeRateReminderIntent) error
}

// ExchangeRateReminderOrganization 是待督办的分公司档案。
type ExchangeRateReminderOrganization struct {
	OrganizationID uuid.UUID
	Name           string
	BaseCurrency   string
}

// ExchangeRateReminderRecipient 是督办通知收件人。
type ExchangeRateReminderRecipient struct {
	UserID uuid.UUID
}

// ExchangeRateReminderIntent 是一条督办通知意图（1 任务 = 1 明细 = 1 收件人）。
type ExchangeRateReminderIntent struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	RecipientUserID  uuid.UUID
	OrganizationName string
	BaseCurrency     string
	WeekFrom         time.Time
	WeekTo           time.Time
}

type ExchangeRateReminderUsecase struct {
	repo ExchangeRateReminderRepo
	now  func() time.Time
}

func NewExchangeRateReminderUsecase(repo ExchangeRateReminderRepo) *ExchangeRateReminderUsecase {
	return &ExchangeRateReminderUsecase{repo: repo, now: time.Now}
}

// RunWeeklyReminderCheck 执行一次当周汇率漏配督办：周一 10:00（Asia/Shanghai）后
// 检测当周仍未同步的分公司，向其汇率维护人员推送财务待办；同周期已同步或通知已
// 入队（确定性 ID 幂等）则不再发送。返回本次新入队的通知条数。
func (uc *ExchangeRateReminderUsecase) RunWeeklyReminderCheck(ctx context.Context, now time.Time) (int, error) {
	weekFrom, weekTo := ExchangeRateWeekWindow(now)
	organizations, err := uc.repo.ListUnsyncedWeeklyOrganizations(ctx, weekFrom, weekTo)
	if err != nil {
		return 0, err
	}
	if len(organizations) == 0 {
		return 0, nil
	}
	intents := make([]*ExchangeRateReminderIntent, 0, len(organizations))
	for _, organization := range organizations {
		recipients, err := uc.repo.ListFinanceReminderRecipients(ctx, organization.OrganizationID)
		if err != nil {
			return 0, err
		}
		for _, recipient := range recipients {
			intents = append(intents, &ExchangeRateReminderIntent{
				ID:               newExchangeRateReminderIntentID(organization.OrganizationID, recipient.UserID, weekFrom),
				OrganizationID:   organization.OrganizationID,
				RecipientUserID:  recipient.UserID,
				OrganizationName: organization.Name,
				BaseCurrency:     organization.BaseCurrency,
				WeekFrom:         weekFrom,
				WeekTo:           weekTo,
			})
		}
	}
	if len(intents) == 0 {
		return 0, nil
	}
	if err := uc.repo.EnqueueWeeklyReminders(ctx, intents); err != nil {
		return 0, err
	}
	return len(intents), nil
}

// newExchangeRateReminderIntentID 按（组织、收件人、目标周）确定性派生任务 ID，
// 保证同周期重复检查最多投递一条。
func newExchangeRateReminderIntentID(organizationID, recipientUserID uuid.UUID, weekFrom time.Time) uuid.UUID {
	namespace := uuid.NewSHA1(uuid.NameSpaceURL, []byte("roncin-exchange-rate-reminder"))
	return uuid.NewSHA1(namespace, []byte(fmt.Sprintf("weekly:%s:%s:%s", organizationID, recipientUserID, weekFrom.Format("2006-01-02"))))
}
