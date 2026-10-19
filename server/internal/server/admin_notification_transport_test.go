package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/uuid"
	adminv1 "github.com/roncin/roncin-go-admin/server/api/admin/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/service"
)

type notificationHTTPAdminRepo struct {
	biz.AdminRepo
	user      *biz.AdminUser
	requested uuid.UUID
}

func (r *notificationHTTPAdminRepo) GetUser(_ context.Context, _, id uuid.UUID) (*biz.AdminUser, error) {
	r.requested = id
	return r.user, nil
}

type notificationHTTPSender struct {
	enabled    bool
	recipients []string
	fail       error
}

func (s *notificationHTTPSender) Enabled() bool { return s.enabled }
func (s *notificationHTTPSender) SendMarkdown(_ context.Context, to string, _ biz.DingTalkMarkdownMessage) error {
	s.recipients = append(s.recipients, to)
	return s.fail
}
func (s *notificationHTTPSender) PrepareMarkdown(m biz.DingTalkMarkdownMessage) (biz.DingTalkMarkdownMessage, string, error) {
	return m, "", nil
}
func newNotificationHTTPFixture() (*kratoshttp.Server, *biz.Principal, *notificationHTTPAdminRepo, *notificationHTTPSender) {
	org, user := uuid.New(), uuid.New()
	ding := "current-user-ding"
	repo := &notificationHTTPAdminRepo{user: &biz.AdminUser{ID: user, Enabled: true, DisplayName: "管理员", DingTalkUserID: &ding}}
	sender := &notificationHTTPSender{enabled: true}
	p := &biz.Principal{UserID: user, Organization: biz.Organization{ID: org, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: org, Kind: biz.OrganizationKindCompany}}, RoleGrants: []biz.RoleGrant{serverRoleGrant("notification-test", biz.DataScopeOrganization, []string{access.PlatformAccess, access.NotificationPreview, access.NotificationTestSend})}}
	policy := &biz.SessionPolicy{CookieName: "sid", TTL: time.Hour, SameSite: "lax"}
	authUC := biz.NewAuthUsecase(&middlewareAuthRepoStub{session: &biz.Session{UserID: user, OrganizationID: org, ExpiresAt: time.Now().Add(time.Hour)}, principal: p}, policy, nil, nil)
	srv := kratoshttp.NewServer(kratoshttp.ResponseEncoder(encodeResponse), kratoshttp.ErrorEncoder(encodeError), kratoshttp.Middleware(Authorization(authUC, policy, nil, nil), RequiredFieldsValidator()))
	adminv1.RegisterAdminServiceHTTPServer(srv, service.NewAdminService(nil, nil, biz.NewNotificationTestUsecase(repo, nil, sender, sender)))
	return srv, p, repo, sender
}
func TestNotificationTestHTTPActualRoutes(t *testing.T) {
	srv, p, repo, sender := newNotificationHTTPFixture()
	const cookie = "sid=test-session"
	res := embeddedRequest(srv, http.MethodGet, "/api/v1/admin/notification-tests/types", "", cookie)
	if res.Code != 200 || !strings.Contains(res.Body.String(), `"recipientName":"管理员"`) || !strings.Contains(res.Body.String(), `"notificationType":"FEE_REVIEW_PENDING"`) {
		t.Fatalf("目录camelCase:%d %s", res.Code, res.Body.String())
	}
	res = embeddedRequest(srv, http.MethodGet, "/api/v1/admin/notification-tests/preview?notificationType=FEE_REVIEW_PENDING", "", cookie)
	if res.Code != 200 || !strings.Contains(res.Body.String(), `"markdown":`) || !strings.Contains(res.Body.String(), "测试通知") || len(sender.recipients) != 0 {
		t.Fatalf("GET预览:%d %s", res.Code, res.Body.String())
	}
	res = embeddedRequest(srv, http.MethodPost, "/api/v1/admin/notification-tests/send", `{"notificationType":"PASSWORD_RESET_CODE","recipientUserId":"someone-else","dingtalkUserId":"attacker","title":"任意内容"}`, cookie)
	var accepted struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			Accepted bool `json:"accepted"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if res.Code != 200 || !accepted.Success || !accepted.Data.Accepted || accepted.Message != "钉钉已受理，请在客户端确认" || len(sender.recipients) != 1 || sender.recipients[0] != "current-user-ding" || repo.requested != p.UserID {
		t.Fatalf("自发目标/受理协议:%d %s %#v", res.Code, res.Body.String(), sender.recipients)
	}
}
func TestNotificationTestHTTPValidationAndPermissions(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, body, cookie string
		status                           int
	}{
		{"未登录", "GET", "/api/v1/admin/notification-tests/types", "", "", 401},
		{"预览缺类型", "GET", "/api/v1/admin/notification-tests/preview", "", "sid=test", 400},
		{"发送缺类型", "POST", "/api/v1/admin/notification-tests/send", `{}`, "sid=test", 400},
		{"未知类型", "GET", "/api/v1/admin/notification-tests/preview?notificationType=unknown", "", "sid=test", 400},
		{"非法订单", "GET", "/api/v1/admin/notification-tests/preview?notificationType=FEE_REVIEW_PENDING&orderId=bad", "", "sid=test", 400},
		{"零订单", "POST", "/api/v1/admin/notification-tests/send", `{"notificationType":"FEE_REVIEW_PENDING","orderId":"00000000-0000-0000-0000-000000000000"}`, "sid=test", 400},
		{"越权订单", "POST", "/api/v1/admin/notification-tests/send", `{"notificationType":"FEE_REVIEW_PENDING","orderId":"00000000-0000-4000-8000-000000000001"}`, "sid=test", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, _, _, sender := newNotificationHTTPFixture()
			res := embeddedRequest(srv, tc.method, tc.path, tc.body, tc.cookie)
			if res.Code != tc.status || len(sender.recipients) != 0 {
				t.Fatalf("%d %s", res.Code, res.Body.String())
			}
		})
	}
	srv, p, repo, sender := newNotificationHTTPFixture()
	const cookie = "sid=test"
	delete(p.RoleGrants[0].Permissions, access.NotificationTestSend)
	res := embeddedRequest(srv, "POST", "/api/v1/admin/notification-tests/send", `{"notificationType":"USER_AUTHORIZED"}`, cookie)
	if res.Code != 403 || len(sender.recipients) != 0 {
		t.Fatal("发送权限未隔离")
	}
	p.RoleGrants[0].Permissions[access.NotificationTestSend] = struct{}{}
	repo.user.DingTalkUserID = nil
	res = embeddedRequest(srv, "POST", "/api/v1/admin/notification-tests/send", `{"notificationType":"USER_AUTHORIZED"}`, cookie)
	if res.Code != 400 || len(sender.recipients) != 0 {
		t.Fatalf("无绑定:%d %s", res.Code, res.Body.String())
	}
	ding := "current-user-ding"
	repo.user.DingTalkUserID = &ding
	sender.enabled = false
	res = embeddedRequest(srv, "POST", "/api/v1/admin/notification-tests/send", `{"notificationType":"USER_AUTHORIZED"}`, cookie)
	if res.Code < 400 || len(sender.recipients) != 0 {
		t.Fatal("禁用仍发送")
	}
	sender.enabled = true
	sender.fail = errors.New("上游失败")
	res = embeddedRequest(srv, "POST", "/api/v1/admin/notification-tests/send", `{"notificationType":"USER_AUTHORIZED"}`, cookie)
	if res.Code != 502 || len(sender.recipients) != 1 {
		t.Fatalf("上游失败伪成功:%d %s", res.Code, res.Body.String())
	}
	delete(p.RoleGrants[0].Permissions, access.NotificationPreview)
	for _, path := range []string{"/api/v1/admin/notification-tests/types", "/api/v1/admin/notification-tests/preview?notificationType=USER_AUTHORIZED"} {
		res = embeddedRequest(srv, "GET", path, "", cookie)
		if res.Code != 403 {
			t.Fatalf("预览权限未校验:%d %s", res.Code, res.Body.String())
		}
	}
}
