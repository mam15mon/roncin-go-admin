package biz

import (
	"context"
	stderr "errors"
	"strings"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/access"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

type notificationPreviewAdminRepo struct {
	AdminRepo
	user      *AdminUser
	requested []uuid.UUID
}

func (r *notificationPreviewAdminRepo) GetUser(_ context.Context, _, id uuid.UUID) (*AdminUser, error) {
	r.requested = append(r.requested, id)
	return r.user, nil
}

type notificationPreviewSender struct {
	enabled    bool
	messages   []DingTalkMarkdownMessage
	recipients []string
	fail       error
}

func (s *notificationPreviewSender) Enabled() bool { return s.enabled }
func (s *notificationPreviewSender) SendMarkdown(_ context.Context, recipient string, m DingTalkMarkdownMessage) error {
	s.recipients = append(s.recipients, recipient)
	s.messages = append(s.messages, m)
	return s.fail
}
func (*notificationPreviewSender) PrepareMarkdown(m DingTalkMarkdownMessage) (DingTalkMarkdownMessage, string, error) {
	u := ""
	if m.OrderEntryPath != "" {
		u = "https://roncin.example" + m.OrderEntryPath
		m.Text += "\n\n[查看订单费用](" + u + ")"
		m.OrderEntryPath = ""
	}
	return m, u, nil
}

type notificationPreviewOrderRepo struct {
	OrderRepo
	order     *Order
	requested []OrderOrganizationScope
}

func (r *notificationPreviewOrderRepo) FindAuthorized(_ context.Context, id uuid.UUID, scopes []OrderOrganizationScope) (*Order, error) {
	r.requested = scopes
	for _, s := range scopes {
		for _, org := range s.OrganizationIDs {
			if r.order != nil && r.order.ID == id && r.order.OrganizationID == org && r.order.BusinessType == s.BusinessType {
				return r.order, nil
			}
		}
	}
	return nil, ErrOrderNotFound
}
func newNotificationPreviewFixture() (*NotificationTestUsecase, *notificationPreviewAdminRepo, *notificationPreviewSender, *Principal, *notificationPreviewOrderRepo) {
	org, user := uuid.New(), uuid.New()
	ding := "self-dingtalk"
	repo := &notificationPreviewAdminRepo{user: &AdminUser{ID: user, DisplayName: "当前管理员", Enabled: true, DingTalkUserID: &ding}}
	sender := &notificationPreviewSender{enabled: true}
	orders := &notificationPreviewOrderRepo{order: &Order{ID: uuid.New(), OrganizationID: org, BusinessType: OrderBusinessSE, OrderNo: "SE-[001]"}}
	permissions := map[string]struct{}{}
	for _, k := range []string{access.PlatformAccess, access.NotificationPreview, access.NotificationTestSend, access.OrderPermission(access.OrderBusinessSE, access.OrderRead), access.OrderPermission(access.OrderBusinessSE, access.OrderFeeRead)} {
		permissions[k] = struct{}{}
	}
	p := &Principal{UserID: user, Organization: Organization{ID: org, Kind: OrganizationKindSystem}, OrganizationNodes: []OrganizationScopeNode{{ID: org, Kind: OrganizationKindSystem}}, RoleGrants: []RoleGrant{{RoleCode: "administrator", DataScope: DataScopeOrganization, Permissions: permissions}}}
	return NewNotificationTestUsecase(repo, NewOrderUsecase(orders, nil, nil, nil, nil, nil), sender, sender), repo, sender, p, orders
}
func TestNotificationTestAllTypesPreviewAndSelfSend(t *testing.T) {
	uc, repo, sender, p, _ := newNotificationPreviewFixture()
	ctx := context.Background()
	catalog, err := uc.Catalog(ctx, p)
	if err != nil || len(catalog.Types) != 8 || !catalog.CanSend || catalog.CanSelectOrder {
		t.Fatalf("目录:%+v %v", catalog, err)
	}
	for _, typ := range catalog.Types {
		t.Run(typ.Type, func(t *testing.T) {
			before := len(sender.messages)
			preview, err := uc.Preview(ctx, p, typ.Type, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(sender.messages) != before || !strings.HasPrefix(preview.Title, "[测试]") || !strings.Contains(preview.Markdown, "[测试通知]") || preview.OrderURL != "" || strings.Contains(preview.Markdown, "https://") {
				t.Fatalf("预览出现副作用或伪造链接:%+v", preview)
			}
			if typ.Type == "PASSWORD_RESET_CODE" && (!strings.Contains(preview.Markdown, "**012345**") || !strings.Contains(preview.Markdown, "不会创建验证码挑战")) {
				t.Fatal("验证码示例不完整")
			}
			if err := uc.Send(ctx, p, typ.Type, nil); err != nil {
				t.Fatal(err)
			}
			sent := sender.messages[len(sender.messages)-1]
			if sent.Title != preview.Title || sent.Text != preview.Markdown || sender.recipients[len(sender.recipients)-1] != "self-dingtalk" {
				t.Fatal("预览与自发载荷不同")
			}
		})
	}
	for _, id := range repo.requested {
		if id != p.UserID {
			t.Fatal("读取了其他用户绑定")
		}
	}
}
func TestNotificationTestRejectsRealOrderInSystemWorkspace(t *testing.T) {
	uc, _, sender, p, orders := newNotificationPreviewFixture()
	id := orders.order.ID
	for _, typ := range []string{"FEE_REVIEW_PENDING", "FEE_REVIEW_RESULT"} {
		if _, err := uc.Preview(t.Context(), p, typ, &id); err != ErrPermissionDenied {
			t.Fatalf("系统管理员预览真实订单:%v", err)
		}
		if err := uc.Send(t.Context(), p, typ, &id); err != ErrPermissionDenied {
			t.Fatalf("系统管理员发送真实订单:%v", err)
		}
	}
	if len(orders.requested) != 0 || len(sender.messages) != 0 {
		t.Fatal("拒绝后仍读取订单或发送消息")
	}
	if _, err := uc.Preview(t.Context(), p, "USER_AUTHORIZED", &id); err != ErrAdminInvalidArgument {
		t.Fatalf("非费用类型接受订单:%v", err)
	}
	c, err := uc.Catalog(t.Context(), p)
	if err != nil || c.CanSelectOrder || c.OrderSelectionUnavailableReason == "" {
		t.Fatalf("目录遗漏订单不可用说明:%+v %v", c, err)
	}
}

func TestNotificationTestOrderScopesKeepCompanyReadBoundary(t *testing.T) {
	uc, _, _, p, orders := newNotificationPreviewFixture()
	p.Organization.Kind = OrganizationKindCompany
	p.OrganizationNodes[0].Kind = OrganizationKindCompany
	p.RoleGrants[0].DataScope = DataScopeAll
	scopes := notificationTestOrderScopes(p)
	if len(scopes) == 0 {
		t.Fatal("公司订单读取范围丢失")
	}
	if _, err := uc.order.FindAuthorized(t.Context(), orders.order.ID, scopes); err != nil {
		t.Fatal(err)
	}
	orders.order.OrganizationID = uuid.New()
	if _, err := uc.order.FindAuthorized(t.Context(), orders.order.ID, scopes); err != ErrOrderNotFound {
		t.Fatalf("跨公司读取:%v", err)
	}
	for _, key := range []string{access.OrderPermission(access.OrderBusinessSE, access.OrderRead), access.OrderPermission(access.OrderBusinessSE, access.OrderFeeRead)} {
		t.Run(key, func(t *testing.T) {
			delete(p.RoleGrants[0].Permissions, key)
			if len(notificationTestOrderScopes(p)) != 0 {
				t.Fatal("缺少读取权限仍产生范围")
			}
			p.RoleGrants[0].Permissions[key] = struct{}{}
		})
	}
}

func TestNotificationTestRejectionsAndDisabledPreview(t *testing.T) {
	ctx := context.Background()
	uc, repo, sender, p, _ := newNotificationPreviewFixture()
	if _, err := uc.Preview(ctx, p, "UNKNOWN", nil); err == nil {
		t.Fatal("未知类型未拒绝")
	}
	repo.user.DingTalkUserID = nil
	c, _ := uc.Catalog(ctx, p)
	if c.CanSend || c.RecipientBound || c.SendUnavailableReason == "" {
		t.Fatal("无绑定仍可发送")
	}
	if _, err := uc.Preview(ctx, p, "USER_AUTHORIZED", nil); err != nil {
		t.Fatal("无绑定仍应可预览", err)
	}
	if err := uc.Send(ctx, p, "USER_AUTHORIZED", nil); err == nil {
		t.Fatal("无绑定发送未拒绝")
	}
	ding := "new-binding"
	repo.user.DingTalkUserID = &ding
	sender.enabled = false
	if _, err := uc.Preview(ctx, p, "PASSWORD_RESET_CODE", nil); err != nil {
		t.Fatal("集成禁用不能预览", err)
	}
	if err := uc.Send(ctx, p, "USER_AUTHORIZED", nil); err != ErrDingTalkDisabled {
		t.Fatalf("禁用:%v", err)
	}
	sender.enabled = true
	sender.fail = stderr.New("上游失败")
	if err := uc.Send(ctx, p, "USER_AUTHORIZED", nil); kratoserrors.FromError(err).Reason != "NOTIFICATION_TEST_SEND_FAILED" {
		t.Fatalf("上游失败被吞:%v", err)
	}
	if sender.recipients[len(sender.recipients)-1] != "new-binding" {
		t.Fatal("未重新读取真实绑定")
	}
	delete(p.RoleGrants[0].Permissions, access.NotificationTestSend)
	before := len(sender.messages)
	if err := uc.Send(ctx, p, "USER_AUTHORIZED", nil); err != ErrPermissionDenied || len(sender.messages) != before {
		t.Fatal("发送越权")
	}
	delete(p.RoleGrants[0].Permissions, access.NotificationPreview)
	if _, err := uc.Preview(ctx, p, "USER_AUTHORIZED", nil); err != ErrPermissionDenied {
		t.Fatal("预览越权")
	}
}

func TestNotificationTestSystemWorkspaceDoesNotGainOrderScope(t *testing.T) {
	uc, _, _, p, _ := newNotificationPreviewFixture()
	p.IsBootstrapAdmin = true
	p.Organization.Kind = OrganizationKindSystem
	p.OrganizationNodes[0].Kind = OrganizationKindSystem
	catalog, err := uc.Catalog(context.Background(), p)
	if err != nil || catalog.CanSelectOrder {
		t.Fatalf("系统管理扩大经营权限:%+v %v", catalog, err)
	}
	preview, err := uc.Preview(context.Background(), p, "FEE_REVIEW_PENDING", nil)
	if err != nil || preview.OrderURL != "" {
		t.Fatalf("无订单仍应可预览:%+v %v", preview, err)
	}
}
