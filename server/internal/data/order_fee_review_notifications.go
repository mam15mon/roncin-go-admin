package data

import (
	"context"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	taskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	noticeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/notificationdelivery"
)

func enqueueFeeReviewResultForUser(ctx context.Context, client *ent.Client, o *ent.Order, eventID, recipientID uuid.UUID, actorName, result string) error {
	return enqueueFeeReviewNoticeForUser(ctx, client, o, eventID, recipientID, actorName, result, noticeent.TemplateFEE_REVIEW_RESULT, "fee-review-result:")
}

func enqueueFeeReviewNoticeForUser(ctx context.Context, client *ent.Client, o *ent.Order, eventID, recipientID uuid.UUID, actorName, result string, template noticeent.Template, prefix string) error {
	taskID := uuid.NewSHA1(dingTalkNotificationNamespace, []byte(prefix+eventID.String()+":"+recipientID.String()))
	if err := client.BackgroundTask.Create().SetID(taskID).SetOrganizationID(o.OrganizationID).SetKind(taskent.KindDINGTALK_NOTIFICATION).SetIdempotencyKey("dingtalk-notice:" + taskID.String()).SetStatus(taskent.StatusPENDING).SetAttempts(0).SetMaxAttempts(5).SetNextRunAt(time.Now()).OnConflict(entsql.DoNothing()).Exec(ctx); err != nil {
		return err
	}
	parameter := actorName + "：" + result + feeReviewOrderEntry(o)
	return client.NotificationDelivery.Create().SetBackgroundTaskID(taskID).SetRecipientUserID(recipientID).SetChannel(noticeent.ChannelDINGTALK).SetTemplate(template).SetResourceType("ORDER").SetResourceID(o.ID).SetReferenceCode(clampNotificationBytes(o.OrderNo, 64)).SetParameter(clampNotificationBytes(parameter, 256)).OnConflict(entsql.DoNothing()).Exec(ctx)
}

// 仅为前端已注册的业务类型提供入口；其余类型保留订单编码，避免生成不可打开的链接。
func feeReviewOrderEntry(o *ent.Order) string {
	path := biz.OrderFeesPath(biz.OrderBusinessType(o.BusinessType), o.ID)
	if path == "" {
		return ""
	}
	return "\n订单入口：" + path
}
