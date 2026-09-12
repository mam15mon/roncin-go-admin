package data

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	backgroundtaskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	notificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/notificationdelivery"
)

type notificationRepo struct {
	data *Data
}

func NewNotificationRepo(data *Data) biz.NotificationRepo {
	return &notificationRepo{data: data}
}

func enqueueOrderPersonnelNotification(ctx context.Context, tx *ent.Tx, organizationID, orderID uuid.UUID, orderNo string, role biz.OrderPersonnelRole, recipient *ent.User, intent *biz.NotificationIntent) error {
	if intent == nil || recipient == nil || recipient.DingtalkUserid == nil || strings.TrimSpace(*recipient.DingtalkUserid) == "" {
		return nil
	}
	if intent.ID == uuid.Nil || intent.RecipientUserID != recipient.ID || intent.Channel != biz.NotificationChannelDingTalk || intent.Template != biz.NotificationTemplateOrderPersonnelAssign {
		return fmt.Errorf("订单人员通知意图不合法")
	}
	now := time.Now()
	if _, err := tx.BackgroundTask.Create().
		SetID(intent.ID).
		SetOrganizationID(organizationID).
		SetKind(backgroundtaskent.KindDINGTALK_NOTIFICATION).
		SetIdempotencyKey("order-personnel:" + intent.ID.String()).
		SetStatus(backgroundtaskent.StatusPENDING).
		SetAttempts(0).
		SetMaxAttempts(5).
		SetNextRunAt(now).
		Save(ctx); err != nil {
		return err
	}
	if _, err := tx.NotificationDelivery.Create().
		SetBackgroundTaskID(intent.ID).
		SetRecipientUserID(recipient.ID).
		SetChannel(notificationent.ChannelDINGTALK).
		SetTemplate(notificationent.TemplateORDER_PERSONNEL_ASSIGNED).
		SetResourceType("ORDER").
		SetResourceID(orderID).
		SetReferenceCode(orderNo).
		SetParameter(string(role)).
		Save(ctx); err != nil {
		return err
	}
	return nil
}

func enqueueDingTalkUserAuthorizedNotification(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, recipient *ent.User, intent *biz.NotificationIntent) error {
	if recipient == nil || recipient.DingtalkUserid == nil || strings.TrimSpace(*recipient.DingtalkUserid) == "" {
		return fmt.Errorf("钉钉授权完成通知缺少收件人")
	}
	if intent == nil || intent.ID == uuid.Nil || intent.RecipientUserID != recipient.ID || intent.Channel != biz.NotificationChannelDingTalk || intent.Template != biz.NotificationTemplateUserAuthorized {
		return fmt.Errorf("钉钉授权完成通知意图不合法")
	}
	now := time.Now()
	if _, err := tx.BackgroundTask.Create().
		SetID(intent.ID).
		SetOrganizationID(organizationID).
		SetKind(backgroundtaskent.KindDINGTALK_NOTIFICATION).
		SetIdempotencyKey("user-authorized:" + intent.ID.String()).
		SetStatus(backgroundtaskent.StatusPENDING).
		SetAttempts(0).
		SetMaxAttempts(5).
		SetNextRunAt(now).
		Save(ctx); err != nil {
		return err
	}
	if _, err := tx.NotificationDelivery.Create().
		SetBackgroundTaskID(intent.ID).
		SetRecipientUserID(recipient.ID).
		SetChannel(notificationent.ChannelDINGTALK).
		SetTemplate(notificationent.TemplateUSER_AUTHORIZED).
		SetResourceType("USER").
		SetResourceID(recipient.ID).
		Save(ctx); err != nil {
		return err
	}
	return nil
}

// enqueueDingTalkRegistrationPendingNotifications 在注册同事务内向审批人入队
// 「注册待审批」通知。通知模型是 1 任务 = 1 明细 = 1 收件人，因此任务与明细
// ID 按 (注册人, 路由组织, 收件人) 三元组确定性生成并 OnConflict DoNothing：
// 每位收件人各得一条独立任务与明细，重复确认注册时同一收件人不重复提醒。
// 取舍：被拒绝后重新注册的同组织注册不会再次提醒（旧任务已存在），
// 审批队列实时查询兜底，审批人仍能在队列页发现该注册。
func enqueueDingTalkRegistrationPendingNotifications(ctx context.Context, tx *ent.Tx, routingOrganizationID uuid.UUID, registrantUserID uuid.UUID, registrantName, organizationName string, recipientUserIDs []uuid.UUID) error {
	if len(recipientUserIDs) == 0 {
		return nil
	}
	referenceCode, parameter := clampNotificationBytes(registrantName, 64), clampNotificationBytes(organizationName, 256)
	if referenceCode == "" || parameter == "" {
		return nil
	}
	now := time.Now()
	for _, recipientUserID := range recipientUserIDs {
		intentID := uuid.NewSHA1(dingTalkNotificationNamespace, []byte(fmt.Sprintf("registration-pending:%s:%s:%s", registrantUserID, routingOrganizationID, recipientUserID)))
		if err := tx.BackgroundTask.Create().
			SetID(intentID).
			SetOrganizationID(routingOrganizationID).
			SetKind(backgroundtaskent.KindDINGTALK_NOTIFICATION).
			SetIdempotencyKey("registration-pending:" + intentID.String()).
			SetStatus(backgroundtaskent.StatusPENDING).
			SetAttempts(0).
			SetMaxAttempts(5).
			SetNextRunAt(now).
			OnConflict(entsql.DoNothing()).
			Exec(ctx); err != nil {
			return err
		}
		if err := tx.NotificationDelivery.Create().
			SetBackgroundTaskID(intentID).
			SetRecipientUserID(recipientUserID).
			SetChannel(notificationent.ChannelDINGTALK).
			SetTemplate(notificationent.TemplateDINGTALK_REGISTRATION_PENDING).
			SetResourceType("USER").
			SetResourceID(registrantUserID).
			SetReferenceCode(referenceCode).
			SetParameter(parameter).
			OnConflict(entsql.DoNothing()).
			Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

// enqueueDingTalkRegistrationRejectedNotification 通知注册本人审批被拒绝；
// 拒绝原因截断后随通知明细投递。
func enqueueDingTalkRegistrationRejectedNotification(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, recipient *ent.User, reason string, intent *biz.NotificationIntent) error {
	if intent == nil || intent.ID == uuid.Nil || intent.RecipientUserID != recipient.ID || intent.Channel != biz.NotificationChannelDingTalk || intent.Template != biz.NotificationTemplateDingTalkRegistrationRejected {
		return fmt.Errorf("注册拒绝通知意图不合法")
	}
	return createDingTalkNotificationDelivery(ctx, tx, organizationID, recipient.ID, notificationent.TemplateDINGTALK_REGISTRATION_REJECTED, "USER", recipient.ID, "", clampNotificationBytes(reason, 256), intent.ID)
}

// enqueueDingTalkInvitationActivatedNotification 通知邀请人其邀请的员工已自动激活。
func enqueueDingTalkInvitationActivatedNotification(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, recipient *ent.User, activatedName, organizationName string, intent *biz.NotificationIntent) error {
	if intent == nil || intent.ID == uuid.Nil || intent.RecipientUserID != recipient.ID || intent.Channel != biz.NotificationChannelDingTalk || intent.Template != biz.NotificationTemplateDingTalkInvitationActivated {
		return fmt.Errorf("邀请激活通知意图不合法")
	}
	referenceCode, parameter := clampNotificationBytes(activatedName, 64), clampNotificationBytes(organizationName, 256)
	if referenceCode == "" || parameter == "" {
		return nil
	}
	return createDingTalkNotificationDelivery(ctx, tx, organizationID, recipient.ID, notificationent.TemplateDINGTALK_INVITATION_ACTIVATED, "USER", recipient.ID, referenceCode, parameter, intent.ID)
}

// createDingTalkNotificationDelivery 建立通知任务与明细的通用封装。
func createDingTalkNotificationDelivery(ctx context.Context, tx *ent.Tx, organizationID, recipientUserID uuid.UUID, template notificationent.Template, resourceType string, resourceID uuid.UUID, referenceCode, parameter string, taskID uuid.UUID) error {
	now := time.Now()
	if _, err := tx.BackgroundTask.Create().
		SetID(taskID).
		SetOrganizationID(organizationID).
		SetKind(backgroundtaskent.KindDINGTALK_NOTIFICATION).
		SetIdempotencyKey("dingtalk-notice:" + taskID.String()).
		SetStatus(backgroundtaskent.StatusPENDING).
		SetAttempts(0).
		SetMaxAttempts(5).
		SetNextRunAt(now).
		Save(ctx); err != nil {
		return err
	}
	create := tx.NotificationDelivery.Create().
		SetBackgroundTaskID(taskID).
		SetRecipientUserID(recipientUserID).
		SetChannel(notificationent.ChannelDINGTALK).
		SetTemplate(template).
		SetResourceType(resourceType).
		SetResourceID(resourceID)
	if referenceCode != "" {
		create.SetReferenceCode(referenceCode)
	}
	if parameter != "" {
		create.SetParameter(parameter)
	}
	_, err := create.Save(ctx)
	return err
}

// clampNotificationBytes 把通知明细文本按字节上限安全截断，在 UTF-8 字符边界对齐，
// 避免多字节字符截断损坏以及超出 Ent MaxLen(字节数) 导致的入库失败。
func clampNotificationBytes(value string, maxBytes int) string {
	value = strings.TrimSpace(value)
	if maxBytes <= 0 {
		return ""
	}
	if len(value) <= maxBytes {
		return value
	}
	idx := maxBytes
	for idx > 0 && !utf8.RuneStart(value[idx]) {
		idx--
	}
	return strings.TrimSpace(value[:idx])
}

// dingTalkNotificationNamespace 用于派生确定性通知任务 ID（注册审批提醒按
// 注册人 + 组织去重），随机生成的固定命名空间，无业务含义。
var dingTalkNotificationNamespace = uuid.NewSHA1(uuid.NameSpaceURL, []byte("roncin-dingtalk-notification"))

func (r *notificationRepo) FindByTaskID(ctx context.Context, taskID uuid.UUID) (*biz.NotificationDelivery, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.NotificationDelivery.Query().
		Where(notificationent.BackgroundTaskIDEQ(taskID)).
		WithRecipientUser().
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrNotificationNotFound, nil)
	}
	recipient, err := item.Edges.RecipientUserOrErr()
	if err != nil {
		return nil, err
	}
	dingTalkUserID := ""
	if recipient.DingtalkUserid != nil {
		dingTalkUserID = strings.TrimSpace(*recipient.DingtalkUserid)
	}
	return &biz.NotificationDelivery{
		RecipientUserID:      item.RecipientUserID,
		RecipientDisplayName: recipient.DisplayName,
		DingTalkUserID:       dingTalkUserID,
		Channel:              string(item.Channel),
		Template:             string(item.Template),
		ResourceType:         item.ResourceType,
		ResourceID:           item.ResourceID,
		ReferenceCode:        item.ReferenceCode,
		Parameter:            item.Parameter,
	}, nil
}

var _ biz.NotificationRepo = (*notificationRepo)(nil)
