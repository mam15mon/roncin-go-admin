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

var ErrDingTalkNotificationSiteInvalid = errors.BadRequest("DINGTALK_NOTIFICATION_SITE_INVALID", "钉钉通知站点配置不合法，请检查回调地址配置")

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

// DingTalkMessage 是领域层渲染的消息；动作仅来自受控业务路径。
type DingTalkMessage struct {
	Title, Text                        string
	ActionTitle, ActionPath, ActionURL string
}

type DingTalkNotificationSender interface {
	Enabled() bool
	SendMessage(context.Context, string, DingTalkMessage) error
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
	if err := uc.sender.SendMessage(ctx, delivery.DingTalkUserID, message); err != nil {
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

func renderNotification(delivery *NotificationDelivery) (DingTalkMessage, error) {
	if delivery == nil || strings.TrimSpace(delivery.DingTalkUserID) == "" || delivery.ResourceID == uuid.Nil {
		return DingTalkMessage{}, fmt.Errorf("通知明细不完整")
	}
	if delivery.Channel != NotificationChannelDingTalk {
		return DingTalkMessage{}, fmt.Errorf("通知渠道或模板不受支持")
	}
	switch delivery.Template {
	case NotificationTemplateOrderPersonnelAssign:
		if delivery.ResourceType != "ORDER" {
			return DingTalkMessage{}, fmt.Errorf("通知渠道或模板不受支持")
		}
		if strings.TrimSpace(delivery.ReferenceCode) == "" {
			return DingTalkMessage{}, fmt.Errorf("通知明细不完整")
		}
		roleLabel, ok := orderPersonnelRoleLabel(OrderPersonnelRole(delivery.Parameter))
		if !ok {
			return DingTalkMessage{}, fmt.Errorf("订单人员角色不受支持")
		}
		return newDingTalkMessage("订单协作提醒", "🔵 新分配", fmt.Sprintf("**%s**\n\n你已被分配为该订单的%s。", escapeNotificationMarkdown(delivery.ReferenceCode), roleLabel), "查看订单列表", "/orders"), nil
	case NotificationTemplateUserAuthorized:
		displayName := strings.TrimSpace(delivery.RecipientDisplayName)
		if delivery.ResourceType != "USER" || displayName == "" {
			return DingTalkMessage{}, fmt.Errorf("通知明细不完整")
		}
		return newDingTalkMessage("账号授权完成", "🟢 已开通", fmt.Sprintf("**%s**\n\n你的组织和角色已完成授权，可以进入 Roncin 工作台。", escapeNotificationMarkdown(displayName)), "进入工作台", "/welcome"), nil
	case NotificationTemplateExchangeRateWeeklyReminder:
		organizationName := strings.TrimSpace(delivery.ReferenceCode)
		// Parameter 携带「本币 目标周起~目标周止」，如「CNY 2026-09-14~2026-09-20」。
		weekScope := strings.TrimSpace(delivery.Parameter)
		if delivery.ResourceType != "ORGANIZATION" || organizationName == "" || weekScope == "" {
			return DingTalkMessage{}, fmt.Errorf("通知明细不完整")
		}
		return newDingTalkMessage("周汇率待维护", "🟠 待处理", fmt.Sprintf("**%s**\n\n%s\n\n本周汇率尚未同步，缺少配置将阻止业务折算。", escapeNotificationMarkdown(organizationName), escapeNotificationMarkdown(weekScope)), "维护汇率", "/finance/exchange-rates"), nil
	case "FEE_REVIEW_PENDING", "FEE_REVIEW_RESULT":
		if delivery.ResourceType != "ORDER" || strings.TrimSpace(delivery.ReferenceCode) == "" || strings.TrimSpace(delivery.Parameter) == "" {
			return DingTalkMessage{}, fmt.Errorf("通知明细不完整")
		}
		return renderFeeReviewMessage(delivery, delivery.Template == "FEE_REVIEW_RESULT"), nil
	case NotificationTemplateFeeSupplementApprovalPending:
		// ResourceID 为补录申请 ID，列表入口避免误用为订单 ID。
		orderNo := strings.TrimSpace(delivery.ReferenceCode)
		summary := strings.TrimSpace(delivery.Parameter)
		if delivery.ResourceType != "FEE_SUPPLEMENT_REQUEST" || orderNo == "" || summary == "" {
			return DingTalkMessage{}, fmt.Errorf("通知明细不完整")
		}
		return newDingTalkMessage("费用补录待审批", "🟠 待审批", fmt.Sprintf("**%s**\n\n%s\n\n请从订单列表进入费用页审核补录申请。", escapeNotificationMarkdown(orderNo), escapeNotificationMarkdown(summary)), "查看订单列表", "/orders"), nil
	case NotificationTemplateCommissionDecreaseSuggested:
		// 只链接员工本人专属冲减来源详情；Parameter 携带经长度限制的金额摘要。
		orderNo := strings.TrimSpace(delivery.ReferenceCode)
		summary := strings.TrimSpace(delivery.Parameter)
		if delivery.ResourceType != "COMMISSION_ADJUSTMENT" || orderNo == "" || summary == "" {
			return DingTalkMessage{}, fmt.Errorf("通知明细不完整")
		}
		return newDingTalkMessage("提成冲减通知", "🟠 冲减已确认", fmt.Sprintf("**%s**\n\n%s\n\n本消息供你知情，实际少发或扣回由财务登记。", escapeNotificationMarkdown(orderNo), escapeNotificationMarkdown(summary)), "查看本人冲减详情", "/commission-adjustments/"+delivery.ResourceID.String()+"/my-supplement-source"), nil
	default:
		return DingTalkMessage{}, fmt.Errorf("通知渠道或模板不受支持")
	}
}

func newDingTalkMessage(title, status, body, actionTitle, actionPath string) DingTalkMessage {
	return DingTalkMessage{Title: title, Text: "### " + title + "\n" + status + "\n\n" + body, ActionTitle: actionTitle, ActionPath: actionPath}
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

func renderFeeReviewMessage(delivery *NotificationDelivery, result bool) DingTalkMessage {
	parameter := delivery.Parameter
	actionPath := ""
	// 入队路径由订单 UUID 生成；仅将与当前资源完全匹配的入口作为链接。
	if delivery.ResourceType == "ORDER" {
		for _, businessType := range []OrderBusinessType{OrderBusinessSE, OrderBusinessAE} {
			path := OrderFeesPath(businessType, delivery.ResourceID)
			entrySuffix := "\n订单入口：" + path
			if strings.HasSuffix(parameter, entrySuffix) {
				parameter = strings.TrimSuffix(parameter, entrySuffix)
				actionPath = path
				break
			}
		}
	}
	title, status := "订单费用待审核", "🟠 待审核"
	summary := fmt.Sprintf("%s已提交整票费用，请核对费用及利润后处理。", escapeNotificationMarkdown(parameter))
	if result {
		title, status = "订单费用审核结果", feeReviewResultStatus(parameter)
		summary = escapeNotificationMarkdown(parameter) + "\n\n请查看审批意见。"
	}
	message := newDingTalkMessage(title, status, "**"+escapeNotificationMarkdown(delivery.ReferenceCode)+"**\n\n"+summary, "", actionPath)
	if actionPath != "" {
		message.ActionTitle = "查看订单费用"
	}
	return message
}

// OrderFeesPath 只为已接通的订单品类生成费用入口。
func OrderFeesPath(businessType OrderBusinessType, id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	var kind string
	switch businessType {
	case OrderBusinessSE:
		kind = "sea-export"
	case OrderBusinessAE:
		kind = "air-export"
	default:
		return ""
	}
	return "/orders/" + kind + "/" + id.String() + "/fees"
}

// 结果仅识别入队端的固定后缀，姓名中出现状态词不会改变显示状态。
func feeReviewResultStatus(parameter string) string {
	for _, suffix := range []string{"：已批准", "：费用补录已批准"} {
		if strings.HasSuffix(parameter, suffix) {
			return "🟢 已批准"
		}
	}
	for _, suffix := range []string{"：已退回", "：费用补录已退回"} {
		if strings.HasSuffix(parameter, suffix) {
			return "🔴 已退回"
		}
	}
	return "🔵 处理结果"
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
