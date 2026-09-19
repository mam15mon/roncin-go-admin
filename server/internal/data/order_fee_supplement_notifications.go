package data

import (
	"context"
	"fmt"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	backgroundtaskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	notificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/notificationdelivery"
)

// EnqueueApprovalPendingNotifications 在创建申请的同一事务内为审批人逐人入队
// 待审批通知：任务 ID 与幂等键按「申请 + 审批人」确定性生成，BackgroundTask 与
// NotificationDelivery 均以唯一约束幂等收敛；未绑定钉钉的审批人跳过投递明细，
// 审批资格仍以接口实时校验为准。
func (r *orderFeeSupplementRepo) EnqueueApprovalPendingNotifications(ctx context.Context, organizationID, requestID uuid.UUID, orderNo string, totalAmount decimal.Decimal, currency string, approvers []biz.OrderFeeSupplementApprover) error {
	if len(approvers) == 0 {
		return nil
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	summary := clampNotificationBytes(fmt.Sprintf("订单 %s 补录应付 %s %s", orderNo, totalAmount.StringFixed(8), currency), 256)
	referenceCode := clampNotificationBytes(orderNo, 64)
	if summary == "" || referenceCode == "" {
		return nil
	}
	now := time.Now()
	for _, approver := range approvers {
		if strings.TrimSpace(approver.DingTalkUserID) == "" {
			continue
		}
		taskID := uuid.NewSHA1(dingTalkNotificationNamespace, []byte("fee-supplement-pending:"+requestID.String()+":"+approver.UserID.String()))
		if err := client.BackgroundTask.Create().
			SetID(taskID).
			SetOrganizationID(organizationID).
			SetKind(backgroundtaskent.KindDINGTALK_NOTIFICATION).
			SetIdempotencyKey("dingtalk-notice:" + taskID.String()).
			SetStatus(backgroundtaskent.StatusPENDING).
			SetAttempts(0).
			SetMaxAttempts(5).
			SetNextRunAt(now).
			OnConflict(entsql.DoNothing()).
			Exec(ctx); err != nil {
			return err
		}
		if err := client.NotificationDelivery.Create().
			SetBackgroundTaskID(taskID).
			SetRecipientUserID(approver.UserID).
			SetChannel(notificationent.ChannelDINGTALK).
			SetTemplate(notificationent.TemplateFEE_SUPPLEMENT_APPROVAL_PENDING).
			SetResourceType("FEE_SUPPLEMENT_REQUEST").
			SetResourceID(requestID).
			SetReferenceCode(referenceCode).
			SetParameter(summary).
			OnConflict(entsql.DoNothing()).
			Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

// EnqueueDecreaseSuggestedNotifications 在审批事务内为实际生成冲减建议的员工
// 逐人入队知情通知：任务 ID 与幂等键按「调整 + 员工」确定性生成；未产生建议
// 的员工不通知。
func (r *orderFeeSupplementRepo) EnqueueDecreaseSuggestedNotifications(ctx context.Context, organizationID uuid.UUID, orderNo string, suggestions []biz.OrderFeeSupplementDecreaseSuggestion) error {
	if len(suggestions) == 0 {
		return nil
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	referenceCode := clampNotificationBytes(orderNo, 64)
	if referenceCode == "" {
		return nil
	}
	now := time.Now()
	for _, suggestion := range suggestions {
		summary := clampNotificationBytes(fmt.Sprintf("冲减建议 %s %s", suggestion.Amount.StringFixed(8), suggestion.BaseCurrency), 256)
		taskID := uuid.NewSHA1(dingTalkNotificationNamespace, []byte("commission-decrease-suggested:"+suggestion.AdjustmentID.String()+":"+suggestion.EmployeeID.String()))
		if err := client.BackgroundTask.Create().
			SetID(taskID).
			SetOrganizationID(organizationID).
			SetKind(backgroundtaskent.KindDINGTALK_NOTIFICATION).
			SetIdempotencyKey("dingtalk-notice:" + taskID.String()).
			SetStatus(backgroundtaskent.StatusPENDING).
			SetAttempts(0).
			SetMaxAttempts(5).
			SetNextRunAt(now).
			OnConflict(entsql.DoNothing()).
			Exec(ctx); err != nil {
			return err
		}
		if err := client.NotificationDelivery.Create().
			SetBackgroundTaskID(taskID).
			SetRecipientUserID(suggestion.EmployeeID).
			SetChannel(notificationent.ChannelDINGTALK).
			SetTemplate(notificationent.TemplateCOMMISSION_DECREASE_SUGGESTED).
			SetResourceType("COMMISSION_ADJUSTMENT").
			SetResourceID(suggestion.AdjustmentID).
			SetReferenceCode(referenceCode).
			SetParameter(summary).
			OnConflict(entsql.DoNothing()).
			Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}
