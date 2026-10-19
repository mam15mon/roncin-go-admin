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
	NotificationChannelDingTalk                    = "DINGTALK"
	NotificationTemplateOrderPersonnelAssign       = "ORDER_PERSONNEL_ASSIGNED"
	NotificationTemplateUserAuthorized             = "USER_AUTHORIZED"
	NotificationTemplateExchangeRateWeeklyReminder = "EXCHANGE_RATE_WEEKLY_REMINDER"
	// NotificationTemplateFeeSupplementApprovalPending 提交补录申请时发送给
	// 审批资格快照内人员的待审批通知；resource_type = FEE_SUPPLEMENT_REQUEST。
	NotificationTemplateFeeSupplementApprovalPending = "FEE_SUPPLEMENT_APPROVAL_PENDING"
	// NotificationTemplateCommissionDecreaseSuggested 补录审批生成冲减建议后
	// 发送给被冲减员工的知情通知；resource_type = COMMISSION_ADJUSTMENT。
	NotificationTemplateCommissionDecreaseSuggested = "COMMISSION_DECREASE_SUGGESTED"
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

// DingTalkMarkdownMessage 是领域层渲染的消息；订单入口仅来自已验证的业务路径。
type DingTalkMarkdownMessage struct {
	Title, Text    string
	OrderEntryPath string
}

type DingTalkNotificationSender interface {
	Enabled() bool
	SendMarkdown(context.Context, string, DingTalkMarkdownMessage) error
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
	message, err := renderNotification(delivery)
	if err != nil {
		return uc.fail(ctx, task, err.Error())
	}
	if err := uc.sender.SendMarkdown(ctx, delivery.DingTalkUserID, message); err != nil {
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

func renderNotification(delivery *NotificationDelivery) (DingTalkMarkdownMessage, error) {
	if delivery == nil || strings.TrimSpace(delivery.DingTalkUserID) == "" || delivery.ResourceID == uuid.Nil {
		return DingTalkMarkdownMessage{}, fmt.Errorf("通知明细不完整")
	}
	if delivery.Channel != NotificationChannelDingTalk {
		return DingTalkMarkdownMessage{}, fmt.Errorf("通知渠道或模板不受支持")
	}
	switch delivery.Template {
	case NotificationTemplateOrderPersonnelAssign:
		if delivery.ResourceType != "ORDER" {
			return DingTalkMarkdownMessage{}, fmt.Errorf("通知渠道或模板不受支持")
		}
		if strings.TrimSpace(delivery.ReferenceCode) == "" {
			return DingTalkMarkdownMessage{}, fmt.Errorf("通知明细不完整")
		}
		roleLabel, ok := orderPersonnelRoleLabel(OrderPersonnelRole(delivery.Parameter))
		if !ok {
			return DingTalkMarkdownMessage{}, fmt.Errorf("订单人员角色不受支持")
		}
		return newDingTalkMarkdownMessage("海运出口订单协作提醒", fmt.Sprintf("**订单：** %s\n\n**您已被分配为：** %s\n\n请登录 Roncin 系统查看并处理。", escapeNotificationMarkdown(delivery.ReferenceCode), roleLabel)), nil
	case NotificationTemplateUserAuthorized:
		displayName := strings.TrimSpace(delivery.RecipientDisplayName)
		if delivery.ResourceType != "USER" || displayName == "" {
			return DingTalkMarkdownMessage{}, fmt.Errorf("通知明细不完整")
		}
		return newDingTalkMarkdownMessage("Roncin 账号授权完成", fmt.Sprintf("**%s**，您的所属组织和角色已完成授权。\n\n现在可以使用**钉钉扫码登录** Roncin 系统。", escapeNotificationMarkdown(displayName))), nil
	case NotificationTemplateExchangeRateWeeklyReminder:
		organizationName := strings.TrimSpace(delivery.ReferenceCode)
		// Parameter 携带「本币 目标周起~目标周止」，如「CNY 2026-09-14~2026-09-20」。
		weekScope := strings.TrimSpace(delivery.Parameter)
		if delivery.ResourceType != "ORGANIZATION" || organizationName == "" || weekScope == "" {
			return DingTalkMarkdownMessage{}, fmt.Errorf("通知明细不完整")
		}
		return newDingTalkMarkdownMessage("周汇率同步督办", fmt.Sprintf("**组织：** %s\n\n**本币与目标周：** %s\n\n本周汇率尚未同步，缺少本周配置的业务折算将被阻止。\n\n请先在「财务 · 汇率」页维护汇率。", escapeNotificationMarkdown(organizationName), escapeNotificationMarkdown(weekScope))), nil
	case "FEE_REVIEW_PENDING":
		return renderFeeReviewMarkdown(delivery, "订单费用待审核", "提交人", "整票费用已完成录入并提交审核，请进入 Roncin 查看费用及利润后处理。"), nil
	case "FEE_REVIEW_RESULT":
		return renderFeeReviewMarkdown(delivery, "订单费用审核结果", "处理结果", "请进入 Roncin 查看审批意见。"), nil
	case NotificationTemplateFeeSupplementApprovalPending:
		// 只链接目标申请的审批最小详情；Parameter 携带经长度限制的订单号与金额摘要。
		orderNo := strings.TrimSpace(delivery.ReferenceCode)
		summary := strings.TrimSpace(delivery.Parameter)
		if delivery.ResourceType != "FEE_SUPPLEMENT_REQUEST" || orderNo == "" || summary == "" {
			return DingTalkMarkdownMessage{}, fmt.Errorf("通知明细不完整")
		}
		return newDingTalkMarkdownMessage("费用补录待审批", fmt.Sprintf("**订单：** %s\n\n**补录摘要：** %s\n\n您具备该订单的费用审核权限，请登录 Roncin 系统在订单费用页处理补录申请。", escapeNotificationMarkdown(orderNo), escapeNotificationMarkdown(summary))), nil
	case NotificationTemplateCommissionDecreaseSuggested:
		// 只链接员工本人专属冲减来源详情；Parameter 携带经长度限制的金额摘要。
		orderNo := strings.TrimSpace(delivery.ReferenceCode)
		summary := strings.TrimSpace(delivery.Parameter)
		if delivery.ResourceType != "COMMISSION_ADJUSTMENT" || orderNo == "" || summary == "" {
			return DingTalkMarkdownMessage{}, fmt.Errorf("通知明细不完整")
		}
		return newDingTalkMarkdownMessage("提成冲减通知", fmt.Sprintf("**订单：** %s\n\n**冲减摘要：** %s\n\n系统已为您的提成生成已确认的冲减。\n\n本消息供您知情，实际少发或扣回由财务登记；请登录 Roncin 系统查看本人专属详情。", escapeNotificationMarkdown(orderNo), escapeNotificationMarkdown(summary))), nil
	default:
		return DingTalkMarkdownMessage{}, fmt.Errorf("通知渠道或模板不受支持")
	}
}

func newDingTalkMarkdownMessage(title, body string) DingTalkMarkdownMessage {
	return DingTalkMarkdownMessage{Title: title, Text: "### " + title + "\n\n" + body}
}

// 动态值只占一行，避免姓名或摘要中的换行与 Markdown/HTML 控制符改变布局。
func escapeNotificationMarkdown(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	var escaped strings.Builder
	for _, r := range value {
		switch r {
		case '&':
			escaped.WriteString("&amp;")
		case '<':
			escaped.WriteString("&lt;")
		case '>':
			escaped.WriteString("&gt;")
		default:
			if strings.ContainsRune("\\`*_{}[]()#+-.!|~", r) {
				escaped.WriteByte('\\')
			}
			escaped.WriteRune(r)
		}
	}
	return escaped.String()
}

func renderFeeReviewMarkdown(delivery *NotificationDelivery, title, label, instruction string) DingTalkMarkdownMessage {
	parameter := delivery.Parameter
	path := "/orders/sea-export/" + delivery.ResourceID.String() + "/fees"
	entrySuffix := "\n订单入口：" + path
	orderEntryPath := ""
	// 入队路径由订单 UUID 生成；仅将与当前资源完全匹配的入口作为链接。
	if delivery.ResourceType == "ORDER" && strings.HasSuffix(parameter, entrySuffix) {
		parameter = strings.TrimSuffix(parameter, entrySuffix)
		orderEntryPath = path
	}
	message := newDingTalkMarkdownMessage(title, fmt.Sprintf("**订单号：** %s\n\n**%s：** %s\n\n%s", escapeNotificationMarkdown(delivery.ReferenceCode), label, escapeNotificationMarkdown(parameter), instruction))
	message.OrderEntryPath = orderEntryPath
	return message
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
