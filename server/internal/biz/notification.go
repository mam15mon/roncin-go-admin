package biz

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var ErrNotificationNotFound = errors.NotFound("NOTIFICATION_NOT_FOUND", "通知明细不存在")

const (
	NotificationChannelDingTalk              = "DINGTALK"
	NotificationTemplateOrderPersonnelAssign = "ORDER_PERSONNEL_ASSIGNED"
	NotificationTemplateUserAuthorized       = "USER_AUTHORIZED"
)

// NotificationIntent 是业务用例交给仓储、并与业务写入同事务落库的通知意图。
type NotificationIntent struct {
	ID              uuid.UUID
	RecipientUserID uuid.UUID
	Channel         string
	Template        string
}

// NotificationDelivery 是通知 Worker 执行一次发送所需的最小业务快照。
type NotificationDelivery struct {
	Task                 *BackgroundTask
	RecipientUserID      uuid.UUID
	RecipientDisplayName string
	DingTalkUserID       string
	Channel              string
	Template             string
	ResourceType         string
	ResourceID           uuid.UUID
	ReferenceCode        string
	Parameter            string
}

type NotificationRepo interface {
	FindByTaskID(context.Context, uuid.UUID) (*NotificationDelivery, error)
}

type DingTalkNotificationSender interface {
	Enabled() bool
	SendText(context.Context, string, string) error
}

type NotificationUsecase struct {
	tasks  *BackgroundTaskUsecase
	repo   NotificationRepo
	sender DingTalkNotificationSender
	now    func() time.Time
}

func NewNotificationUsecase(tasks *BackgroundTaskUsecase, repo NotificationRepo, sender DingTalkNotificationSender) *NotificationUsecase {
	return &NotificationUsecase{tasks: tasks, repo: repo, sender: sender, now: time.Now}
}

func NewOrderPersonnelNotification(recipientUserID uuid.UUID) *NotificationIntent {
	return &NotificationIntent{
		ID:              uuid.Must(uuid.NewV7()),
		RecipientUserID: recipientUserID,
		Channel:         NotificationChannelDingTalk,
		Template:        NotificationTemplateOrderPersonnelAssign,
	}
}

func NewDingTalkUserAuthorizedNotification(recipientUserID uuid.UUID) *NotificationIntent {
	return &NotificationIntent{
		ID:              uuid.Must(uuid.NewV7()),
		RecipientUserID: recipientUserID,
		Channel:         NotificationChannelDingTalk,
		Template:        NotificationTemplateUserAuthorized,
	}
}

func (uc *NotificationUsecase) Enabled() bool {
	return uc != nil && uc.sender != nil && uc.sender.Enabled()
}

// ProcessNext 领取并处理一条钉钉通知；发送失败会记录重试状态后返回错误。
func (uc *NotificationUsecase) ProcessNext(ctx context.Context, leaseDuration time.Duration) error {
	task, err := uc.tasks.ClaimAny(ctx, []BackgroundTaskKind{BackgroundTaskKindDingTalkNotice}, leaseDuration)
	if err != nil {
		return err
	}
	if task.LeaseToken == nil {
		return uc.fail(ctx, task, "通知任务缺少租约令牌")
	}
	delivery, err := uc.repo.FindByTaskID(ctx, task.ID)
	if err != nil {
		return uc.fail(ctx, task, fmt.Sprintf("读取通知明细失败: %v", err))
	}
	delivery.Task = task
	content, err := renderNotification(delivery)
	if err != nil {
		return uc.fail(ctx, task, err.Error())
	}
	if err := uc.sender.SendText(ctx, delivery.DingTalkUserID, content); err != nil {
		return uc.fail(ctx, task, fmt.Sprintf("发送钉钉通知失败: %v", err))
	}
	_, err = uc.tasks.Complete(ctx, task.OrganizationID, task.ID, *task.LeaseToken)
	return err
}

func (uc *NotificationUsecase) fail(ctx context.Context, task *BackgroundTask, message string) error {
	message = strings.TrimSpace(message)
	if utf8.RuneCountInString(message) > 2000 {
		message = string([]rune(message)[:2000])
	}
	if task == nil || task.LeaseToken == nil {
		return fmt.Errorf("%s", message)
	}
	backoff := 30 * time.Second * time.Duration(1<<min(task.Attempts, 5))
	if _, err := uc.tasks.Fail(ctx, task.OrganizationID, task.ID, *task.LeaseToken, message, uc.now().Add(backoff)); err != nil {
		return fmt.Errorf("%s；记录失败状态失败: %w", message, err)
	}
	return fmt.Errorf("%s", message)
}

func renderNotification(delivery *NotificationDelivery) (string, error) {
	if delivery == nil || strings.TrimSpace(delivery.DingTalkUserID) == "" || delivery.ResourceID == uuid.Nil || strings.TrimSpace(delivery.ReferenceCode) == "" {
		return "", fmt.Errorf("通知明细不完整")
	}
	if delivery.Channel != NotificationChannelDingTalk {
		return "", fmt.Errorf("通知渠道或模板不受支持")
	}
	switch delivery.Template {
	case NotificationTemplateOrderPersonnelAssign:
		if delivery.ResourceType != "ORDER" {
			return "", fmt.Errorf("通知渠道或模板不受支持")
		}
		roleLabel, ok := orderPersonnelRoleLabel(OrderPersonnelRole(delivery.Parameter))
		if !ok {
			return "", fmt.Errorf("订单人员角色不受支持")
		}
		return fmt.Sprintf("【海运出口订单协作提醒】\n订单：%s\n您已被分配为：%s\n请登录 Roncin 系统查看并处理。", delivery.ReferenceCode, roleLabel), nil
	case NotificationTemplateUserAuthorized:
		displayName := strings.TrimSpace(delivery.RecipientDisplayName)
		if delivery.ResourceType != "USER" || displayName == "" {
			return "", fmt.Errorf("通知明细不完整")
		}
		return fmt.Sprintf("【Roncin 账号授权完成】\n%s，您的所属组织和角色已完成授权。\n现在可以使用钉钉扫码登录 Roncin 系统。", displayName), nil
	default:
		return "", fmt.Errorf("通知渠道或模板不受支持")
	}
}

func orderPersonnelRoleLabel(role OrderPersonnelRole) (string, bool) {
	switch role {
	case OrderPersonnelRoleOperator:
		return "操作人员", true
	case OrderPersonnelRoleSales:
		return "业务人员", true
	case OrderPersonnelRoleCustomerService:
		return "客服人员", true
	case OrderPersonnelRoleDocument:
		return "单证人员", true
	case OrderPersonnelRoleCommercial:
		return "商务人员", true
	case OrderPersonnelRoleAssociate:
		return "协作人员", true
	case OrderPersonnelRoleAssociate2:
		return "协作人员 2", true
	default:
		return "", false
	}
}
