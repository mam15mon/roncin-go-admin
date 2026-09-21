package data

import (
	"context"
	"fmt"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	backgroundtaskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	exchangerateent "github.com/roncin/roncin-go-admin/server/internal/data/ent/exchangeratesetting"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	notificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/notificationdelivery"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	roleassignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

type exchangeRateReminderRepo struct{ data *Data }

func NewExchangeRateReminderRepo(data *Data) biz.ExchangeRateReminderRepo {
	return &exchangeRateReminderRepo{data: data}
}

// ListUnsyncedWeeklyOrganizations 返回当周尚未同步本组织周汇率的启用分公司：
// kind=company、具备本币（自身或沿组织树继承），且在目标周窗口内没有任何启用的
// 组织汇率行。
func (r *exchangeRateReminderRepo) ListUnsyncedWeeklyOrganizations(ctx context.Context, weekFrom, weekTo time.Time) ([]*biz.ExchangeRateReminderOrganization, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	organizations, err := client.Organization.Query().
		Where(organizationent.KindEQ(organizationent.KindCompany), organizationent.EnabledEQ(true)).
		Order(organizationent.ByName()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.ExchangeRateReminderOrganization, 0, len(organizations))
	for _, organization := range organizations {
		baseCurrency, err := resolveOrganizationBaseCurrency(ctx, client.Organization, organization)
		if err != nil {
			// 无本币的分公司不参与汇率维护，跳过督办。
			continue
		}
		synced, err := client.ExchangeRateSetting.Query().
			Where(
				exchangerateent.OrganizationIDEQ(organization.ID),
				exchangerateent.IsActiveEQ(true),
				exchangerateent.EffectiveFromEQ(weekFrom),
				exchangerateent.EffectiveToEQ(weekTo),
			).
			Exist(ctx)
		if err != nil {
			return nil, err
		}
		if synced {
			continue
		}
		result = append(result, &biz.ExchangeRateReminderOrganization{OrganizationID: organization.ID, Name: organization.Name, BaseCurrency: baseCurrency})
	}
	return result, nil
}

// ListFinanceReminderRecipients 返回组织内持有汇率维护权限（创建或编辑）且绑定
// 钉钉的启用用户；用户按 ID 去重（一人多角色只提醒一次）。
func (r *exchangeRateReminderRepo) ListFinanceReminderRecipients(ctx context.Context, organizationID uuid.UUID) ([]*biz.ExchangeRateReminderRecipient, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	users, err := client.User.Query().
		Where(
			userent.EnabledEQ(true),
			userent.IsBootstrapAdminEQ(false),
			userent.DingtalkUseridNotNil(),
			userent.HasMembershipsWith(
				membershipent.EnabledEQ(true),
				membershipent.OrganizationIDEQ(organizationID),
				membershipent.HasRoleAssignmentsWith(
					roleassignmentent.HasRoleWith(
						roleent.EnabledEQ(true),
						roleent.DataScopeIn(roleent.DataScopeOrganization, roleent.DataScopeOrganizationTree, roleent.DataScopeAll),
						roleent.HasPermissionsWith(
							permissionent.KeyIn(access.FinanceExchangeRateCreate, access.FinanceExchangeRateUpdate),
						),
					),
				),
			),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	seen := make(map[uuid.UUID]struct{}, len(users))
	recipients := make([]*biz.ExchangeRateReminderRecipient, 0, len(users))
	for _, user := range users {
		if _, ok := seen[user.ID]; ok {
			continue
		}
		seen[user.ID] = struct{}{}
		recipients = append(recipients, &biz.ExchangeRateReminderRecipient{UserID: user.ID})
	}
	return recipients, nil
}

// EnqueueWeeklyReminders 幂等入队督办通知：任务与明细 ID 由 biz 层按
// （组织、收件人、目标周）确定性派生，OnConflict DoNothing 保证同周期重复检查
// 不重发。1 任务 = 1 明细 = 1 收件人，与既有通知发件箱模型一致。
func (r *exchangeRateReminderRepo) EnqueueWeeklyReminders(ctx context.Context, intents []*biz.ExchangeRateReminderIntent) error {
	if len(intents) == 0 {
		return nil
	}
	now := time.Now()
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		for _, intent := range intents {
			if intent == nil || intent.ID == uuid.Nil || intent.OrganizationID == uuid.Nil || intent.RecipientUserID == uuid.Nil {
				continue
			}
			if err := tx.BackgroundTask.Create().
				SetID(intent.ID).
				SetOrganizationID(intent.OrganizationID).
				SetKind(backgroundtaskent.KindDINGTALK_NOTIFICATION).
				SetIdempotencyKey("exchange-rate-weekly:" + intent.ID.String()).
				SetStatus(backgroundtaskent.StatusPENDING).
				SetAttempts(0).
				SetMaxAttempts(5).
				SetNextRunAt(now).
				OnConflict(entsql.DoNothing()).
				Exec(ctx); err != nil {
				return err
			}
			if err := tx.NotificationDelivery.Create().
				SetBackgroundTaskID(intent.ID).
				SetRecipientUserID(intent.RecipientUserID).
				SetChannel(notificationent.ChannelDINGTALK).
				SetTemplate(notificationent.TemplateEXCHANGE_RATE_WEEKLY_REMINDER).
				SetResourceType("ORGANIZATION").
				SetResourceID(intent.OrganizationID).
				SetReferenceCode(clampNotificationBytes(intent.OrganizationName, 64)).
				SetParameter(clampNotificationBytes(fmt.Sprintf("%s %s~%s", intent.BaseCurrency, intent.WeekFrom.Format("2006-01-02"), intent.WeekTo.Format("2006-01-02")), 256)).
				OnConflict(entsql.DoNothing()).
				Exec(ctx); err != nil {
				return err
			}
		}
		return nil
	})
}

var _ biz.ExchangeRateReminderRepo = (*exchangeRateReminderRepo)(nil)
