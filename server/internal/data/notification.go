package data

import (
	"context"
	"fmt"
	"strings"
	"time"

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
