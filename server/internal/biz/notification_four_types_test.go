package biz

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
)

func TestNotificationFourTypePathsAndNoNetworkPreview(t *testing.T) {
	id := uuid.New()
	for _, tc := range []struct {
		business OrderBusinessType
		kind     string
	}{{OrderBusinessSE, "sea-export"}, {OrderBusinessSI, "sea-import"}, {OrderBusinessAE, "air-export"}, {OrderBusinessAI, "air-import"}} {
		t.Run(string(tc.business), func(t *testing.T) {
			want := "/orders/" + tc.kind + "/" + id.String() + "/fees"
			if OrderFeesPath(tc.business, id) != want {
				t.Fatal("四类费用导航映射错误")
			}
			for _, result := range []bool{false, true} {
				d := &NotificationDelivery{ResourceType: "ORDER", ResourceID: id, ReferenceCode: string(tc.business) + "-TEST", Parameter: "测试用户提交\n订单入口：" + want}
				message := renderFeeReviewMessage(d, result)
				if message.ActionPath != want || message.ActionTitle != "查看订单费用" || strings.Contains(message.Text, "订单入口：") {
					t.Fatalf("四类受控动作应真实匹配并从正文移除URL: %+v", message)
				}
				d.ResourceID = uuid.New()
				if invalid := renderFeeReviewMessage(d, result); invalid.ActionPath != "" {
					t.Fatal("跨资源链接被采纳")
				}
			}
			uc, _, sender, p, orders := newNotificationPreviewFixture()
			p.Organization.Kind = OrganizationKindCompany
			p.OrganizationNodes[0].Kind = OrganizationKindCompany
			orders.order.BusinessType = tc.business
			read := access.OrderPermission(access.OrderBusinessType(tc.business), access.OrderRead)
			fee := access.OrderPermission(access.OrderBusinessType(tc.business), access.OrderFeeRead)
			p.RoleGrants[0].Permissions = map[string]struct{}{read: {}, fee: {}}
			scopes := notificationTestOrderScopes(p)
			if len(scopes) != 1 || scopes[0].BusinessType != tc.business {
				t.Fatalf("四类读取交集错误: %+v", scopes)
			}
			if _, err := uc.order.FindAuthorized(t.Context(), orders.order.ID, scopes); err != nil {
				t.Fatal(err)
			}
			delete(p.RoleGrants[0].Permissions, fee)
			if len(notificationTestOrderScopes(p)) != 0 {
				t.Fatal("无费用读取不应产生范围")
			}
			p.Organization.Kind = OrganizationKindSystem
			p.OrganizationNodes[0].Kind = OrganizationKindSystem
			p.RoleGrants[0].Permissions[fee] = struct{}{}
			if len(notificationTestOrderScopes(p)) != 0 {
				t.Fatal("系统工作台不能读取实际四类订单")
			}
			if len(sender.messages) != 0 {
				t.Fatal("纯内容预览意外调用真实发送")
			}
		})
	}
	if OrderFeesPath(OrderBusinessLand, id) != "" || OrderFeesPath(OrderBusinessSI, uuid.Nil) != "" {
		t.Fatal("未接通类型/空ID不应生成链接")
	}
}
