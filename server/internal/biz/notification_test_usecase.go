package biz

import (
	"context"
	"strings"

	"github.com/roncin/roncin-go-admin/server/internal/access"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

// NotificationTestUsecase 为 Admin 提供只读预览及向当前登录人发送示例通知。
type NotificationTestUsecase struct {
	admin    AdminRepo
	order    *OrderUsecase
	sender   DingTalkNotificationSender
	preparer DingTalkMessagePreparer
}

func NewNotificationTestUsecase(admin AdminRepo, order *OrderUsecase, sender DingTalkNotificationSender, preparer DingTalkMessagePreparer) *NotificationTestUsecase {
	return &NotificationTestUsecase{admin: admin, order: order, sender: sender, preparer: preparer}
}

type NotificationTestType struct {
	Type, Label   string
	SupportsOrder bool
}
type NotificationTestCatalog struct {
	Types                                       []NotificationTestType
	RecipientName                               string
	RecipientBound, IntegrationEnabled, CanSend bool
	SendUnavailableReason                       string
	CanSelectOrder                              bool
	OrderSelectionUnavailableReason             string
}
type NotificationTestPreview struct{ Type, Title, Markdown, ActionTitle, ActionURL string }

// DingTalkMessagePreparer 复用实际发送的纯内容准备，不申请令牌或发送消息。
type DingTalkMessagePreparer interface {
	PrepareMessage(DingTalkMessage) (DingTalkMessage, error)
}

func notificationTestOrderScopes(p *Principal) []OrderOrganizationScope {
	if p == nil {
		return nil
	}
	var scopes []OrderOrganizationScope
	for _, businessType := range []OrderBusinessType{OrderBusinessSE, OrderBusinessAE} {
		read, err := p.ResolvePermissionOrganizationScope(access.OrderPermission(access.OrderBusinessType(businessType), access.OrderRead))
		if err != nil {
			continue
		}
		fee, err := p.ResolvePermissionOrganizationScope(access.OrderPermission(access.OrderBusinessType(businessType), access.OrderFeeRead))
		if err != nil {
			continue
		}
		var ids []uuid.UUID
		for _, id := range read.ReadableOrganizationIDs {
			for _, feeID := range fee.ReadableOrganizationIDs {
				if id == feeID {
					ids = append(ids, id)
					break
				}
			}
		}
		if len(ids) > 0 {
			scopes = append(scopes, OrderOrganizationScope{BusinessType: businessType, OrganizationIDs: ids})
		}
	}
	return scopes
}

var notificationTestTypes = []NotificationTestType{
	{NotificationTemplateOrderPersonnelAssign, "订单人员分配", false},
	{NotificationTemplateUserAuthorized, "账号授权完成", false},
	{NotificationTemplateExchangeRateWeeklyReminder, "周汇率提醒", false},
	{"FEE_REVIEW_PENDING", "费用待审核", true},
	{"FEE_REVIEW_RESULT", "费用审核结果", true},
	{NotificationTemplateFeeSupplementApprovalPending, "费用补录待审批", false},
	{NotificationTemplateCommissionDecreaseSuggested, "提成冲减通知", false},
	{"PASSWORD_RESET_CODE", "密码验证码", false},
}

func (uc *NotificationTestUsecase) Catalog(ctx context.Context, principal *Principal) (*NotificationTestCatalog, error) {
	if principal == nil || principal.UserID == uuid.Nil || !principal.HasPermission(access.NotificationPreview) {
		return nil, ErrPermissionDenied
	}
	user, err := uc.admin.GetUser(ctx, principal.Organization.ID, principal.UserID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(user.DisplayName)
	if name == "" {
		name = strings.TrimSpace(user.Username)
	}
	bound := user.DingTalkUserID != nil && strings.TrimSpace(*user.DingTalkUserID) != ""
	c := &NotificationTestCatalog{Types: notificationTestTypes, RecipientName: name, RecipientBound: bound, IntegrationEnabled: uc.sender != nil && uc.sender.Enabled(), CanSend: bound && uc.sender != nil && uc.sender.Enabled()}
	if !bound {
		c.SendUnavailableReason = "当前账号未绑定钉钉"
	} else if uc.sender == nil || !uc.sender.Enabled() {
		c.SendUnavailableReason = "钉钉集成未启用"
	}
	c.CanSelectOrder = len(notificationTestOrderScopes(principal)) > 0
	if !c.CanSelectOrder {
		c.OrderSelectionUnavailableReason = "系统管理工作台无经营数据读取权限，本页仅使用示例测试"
	}
	if !principal.HasPermission(access.NotificationTestSend) {
		c.CanSend = false
		c.SendUnavailableReason = "没有发送测试通知权限"
	}
	if !user.Enabled {
		c.CanSend = false
		c.SendUnavailableReason = "当前账号已停用"
	}
	return c, nil
}

func notificationTestDelivery(typ string, selectedOrder *Order) (*NotificationDelivery, error) {

	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	d := &NotificationDelivery{RecipientDisplayName: "示例员工", DingTalkUserID: "preview-only", Channel: NotificationChannelDingTalk, Template: typ, ResourceID: id}
	switch typ {
	case NotificationTemplateOrderPersonnelAssign:
		d.ResourceType, d.ReferenceCode, d.Parameter = "ORDER", "TEST-ORDER-001", string(OrderPersonnelRoleOperator)
	case NotificationTemplateUserAuthorized:
		d.ResourceType = "USER"
	case NotificationTemplateExchangeRateWeeklyReminder:
		d.ResourceType, d.ReferenceCode, d.Parameter = "ORGANIZATION", "测试公司", "CNY 2026-10-19~2026-10-25"
	case "FEE_REVIEW_PENDING", "FEE_REVIEW_RESULT":
		d.ResourceType, d.ReferenceCode = "ORDER", "TEST-ORDER-001"
		d.Parameter = "示例提交人"
		if typ == "FEE_REVIEW_RESULT" {
			d.Parameter = "示例审批人：已退回"
		}
		if selectedOrder != nil {
			d.ResourceID = selectedOrder.ID
			if selectedOrder.OrderNo != "" {
				d.ReferenceCode = selectedOrder.OrderNo
			}
			if path := OrderFeesPath(selectedOrder.BusinessType, d.ResourceID); path != "" {
				d.Parameter += "\n订单入口：" + path
			}
		}
	case NotificationTemplateFeeSupplementApprovalPending:
		d.ResourceType, d.ReferenceCode, d.Parameter = "FEE_SUPPLEMENT_REQUEST", "TEST-ORDER-001", "测试补录摘要"
	case NotificationTemplateCommissionDecreaseSuggested:
		d.ResourceType, d.ReferenceCode, d.Parameter = "COMMISSION_ADJUSTMENT", "TEST-ORDER-001", "测试冲减摘要"
	case "PASSWORD_RESET_CODE":
		d.ResourceType, d.Parameter, d.ResourceID = "USER", "012345", id
	default:
		return nil, errors.BadRequest("NOTIFICATION_TEST_TYPE_INVALID", "通知测试类型不受支持")
	}
	return d, nil
}

func addNotificationTestMark(m DingTalkMessage) DingTalkMessage {
	m.Title = "[测试] " + m.Title
	m.Text = "[测试通知]\n\n" + m.Text
	return m
}
func (uc *NotificationTestUsecase) build(ctx context.Context, principal *Principal, typ string, orderID *uuid.UUID) (DingTalkMessage, *NotificationTestPreview, error) {
	if principal == nil || !principal.HasPermission(access.NotificationPreview) {
		return DingTalkMessage{}, nil, ErrPermissionDenied
	}
	user, err := uc.admin.GetUser(ctx, principal.Organization.ID, principal.UserID)
	if err != nil {
		return DingTalkMessage{}, nil, err
	}
	if !user.Enabled {
		return DingTalkMessage{}, nil, ErrPermissionDenied
	}
	var selectedOrder *Order
	if orderID != nil {
		if typ != "FEE_REVIEW_PENDING" && typ != "FEE_REVIEW_RESULT" {
			return DingTalkMessage{}, nil, ErrAdminInvalidArgument
		}
		scopes := notificationTestOrderScopes(principal)
		if len(scopes) == 0 {
			return DingTalkMessage{}, nil, ErrPermissionDenied
		}
		if uc.order == nil {
			return DingTalkMessage{}, nil, ErrOrderNotFound
		}
		order, e := uc.order.FindAuthorized(ctx, *orderID, scopes)
		if e != nil {
			return DingTalkMessage{}, nil, e
		}
		selectedOrder = order
	}
	d, err := notificationTestDelivery(typ, selectedOrder)
	if err != nil {
		return DingTalkMessage{}, nil, err
	}
	var m DingTalkMessage
	if typ == "PASSWORD_RESET_CODE" {
		m = renderPasswordResetCode("012345")
	} else {
		m, err = renderNotification(d)
	}
	if err != nil {
		return DingTalkMessage{}, nil, err
	}
	m = addNotificationTestMark(m)
	if typ == "PASSWORD_RESET_CODE" {
		m.Text += "\n\n仅为无效示例，不能用于登录或重置密码；本次不会创建验证码挑战。"
		m.ActionTitle, m.ActionPath, m.ActionURL = "", "", ""
	} else {
		m.Text += "\n\n示例测试，不指向真实订单；按钮仅打开系统首页。"
		m.ActionTitle, m.ActionPath, m.ActionURL = "打开系统首页", "/welcome", ""
	}
	m, err = uc.preparer.PrepareMessage(m)
	if err != nil {
		return DingTalkMessage{}, nil, err
	}
	return m, &NotificationTestPreview{Type: typ, Title: m.Title, Markdown: m.Text, ActionTitle: m.ActionTitle, ActionURL: m.ActionURL}, nil
}
func (uc *NotificationTestUsecase) Preview(ctx context.Context, principal *Principal, typ string, orderID *uuid.UUID) (*NotificationTestPreview, error) {
	_, p, err := uc.build(ctx, principal, typ, orderID)
	return p, err
}
func (uc *NotificationTestUsecase) Send(ctx context.Context, principal *Principal, typ string, orderID *uuid.UUID) error {
	if principal == nil || !principal.HasPermission(access.NotificationTestSend) {
		return ErrPermissionDenied
	}
	if uc.sender == nil || !uc.sender.Enabled() {
		return ErrDingTalkDisabled
	}
	m, _, err := uc.build(ctx, principal, typ, orderID)
	if err != nil {
		return err
	}
	user, err := uc.admin.GetUser(ctx, principal.Organization.ID, principal.UserID)
	if err != nil {
		return err
	}
	if !user.Enabled {
		return ErrPermissionDenied
	}
	if user.DingTalkUserID == nil || strings.TrimSpace(*user.DingTalkUserID) == "" {
		return errors.BadRequest("NOTIFICATION_TEST_RECIPIENT_UNBOUND", "当前账号未绑定钉钉")
	}
	if err := uc.sender.SendMessage(ctx, strings.TrimSpace(*user.DingTalkUserID), m); err != nil {
		return errors.New(502, "NOTIFICATION_TEST_SEND_FAILED", "未确认钉钉受理测试通知，请先在客户端确认，再决定是否重试")
	}
	return nil
}
