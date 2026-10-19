package data

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
)

func TestDingTalkPrepareMessageIsOfflineAndMatchesSend(t *testing.T) {
	calls := 0
	var sent map[string]string
	p := &dingTalkIdentityProvider{redirectURI: "https://admin.example.com/user/login/dingtalk/callback?state=ignored", appToken: "cached", appTokenTill: time.Now().Add(time.Hour), client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		var body struct {
			MsgKey   string `json:"msgKey"`
			MsgParam string `json:"msgParam"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.MsgKey != "sampleActionCard" {
			t.Fatalf("消息不是ActionCard:%s", body.MsgKey)
		}
		if err := json.Unmarshal([]byte(body.MsgParam), &sent); err != nil {
			t.Fatal(err)
		}
		return jsonResponse(200, `{"processQueryKey":"accepted"}`), nil
	})}}
	m := biz.DingTalkMessage{Title: "[测试] 订单费用", Text: "[测试通知]\n\n### 订单费用", ActionTitle: "查看订单费用", ActionPath: "/orders/sea-export/00000000-0000-4000-8000-000000000001/fees"}
	preview, err := p.PrepareMessage(m)
	if err != nil || calls != 0 || preview.ActionPath != "" || preview.ActionTitle != m.ActionTitle || !strings.HasPrefix(preview.ActionURL, "https://admin.example.com/orders/") || preview.Text != m.Text {
		t.Fatalf("纯准备路径:%+v %v", preview, err)
	}
	again, err := p.PrepareMessage(preview)
	if err != nil || again != preview || calls != 0 {
		t.Fatalf("准备不幂等:%+v %v", again, err)
	}
	p.enabled = true
	if err := p.SendMessage(context.Background(), "self", preview); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || !reflect.DeepEqual(sent, map[string]string{"title": preview.Title, "text": preview.Text, "singleTitle": preview.ActionTitle, "singleURL": preview.ActionURL}) {
		t.Fatalf("预览发送不一致:%+v", sent)
	}
	for _, origin := range []string{"", "://invalid", "ftp://admin.example.com/callback", "https://user@admin.example.com/callback", "https:///callback"} {
		p.redirectURI = origin
		if _, err := p.PrepareMessage(m); err != biz.ErrDingTalkNotificationSiteInvalid {
			t.Fatalf("站点无效未拒绝:%s %v", origin, err)
		}
	}
}

func TestDingTalkPrepareMessageAllowsOnlyControlledSiteActions(t *testing.T) {
	const orderID = "00000000-0000-4000-8000-000000000001"
	const origin = "https://admin.example.com"
	for _, path := range []string{"/welcome", "/orders", "/finance/exchange-rates", "/orders/sea-export/" + orderID + "/fees", "/commission-adjustments/" + orderID + "/my-supplement-source"} {
		t.Run(path, func(t *testing.T) {
			p := &dingTalkIdentityProvider{redirectURI: origin + "/callback"}
			m, err := p.PrepareMessage(biz.DingTalkMessage{Title: "标题", Text: "正文", ActionTitle: "查看详情", ActionPath: path})
			if err != nil || m.ActionURL != origin+path || m.Text != "正文" {
				t.Fatalf("固定动作被拒绝:%+v %v", m, err)
			}
		})
	}
	for _, tc := range []struct{ name, title, path, target string }{
		{"只有标题", "查看详情", "", ""},
		{"只有路径", "", "/welcome", ""},
		{"只有绝对目标", "", "", origin + "/welcome"},
		{"双重目标", "查看详情", "/welcome", origin + "/welcome"},
		{"空白标题", " ", "/welcome", ""},
		{"多行标题", "查看\n详情", "/welcome", ""},
		{"过长标题", strings.Repeat("查", 21), "/welcome", ""},
		{"外部路径", "查看详情", "https://evil.example/welcome", ""},
		{"协议相对", "查看详情", "//evil.example/welcome", ""},
		{"动态查询", "查看详情", "/welcome?redirect=https://evil.example", ""},
		{"未知页面", "查看详情", "/admin/users", ""},
		{"错误UUID", "查看详情", "/orders/sea-export/not-uuid/fees", ""},
		{"零UUID", "查看详情", "/orders/sea-export/00000000-0000-0000-0000-000000000000/fees", ""},
		{"路径穿越", "查看详情", "/orders/../welcome", ""},
		{"伪造站点", "查看详情", "", "https://evil.example/welcome"},
		{"伪造协议", "查看详情", "", "http://admin.example.com/welcome"},
		{"带凭据", "查看详情", "", "https://user@admin.example.com/welcome"},
		{"绝对目标查询", "查看详情", "", origin + "/welcome?q=1"},
		{"绝对目标片段", "查看详情", "", origin + "/welcome#part"},
		{"绝对目标编码", "查看详情", "", origin + "/%77elcome"},
		{"绝对目标空查询", "查看详情", "", origin + "/welcome?"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			p := &dingTalkIdentityProvider{enabled: true, redirectURI: origin + "/callback", client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return jsonResponse(200, `{}`), nil })}}
			err := p.SendMessage(context.Background(), "self", biz.DingTalkMessage{Title: "标题", Text: "正文", ActionTitle: tc.title, ActionPath: tc.path, ActionURL: tc.target})
			if err == nil || calls != 0 {
				t.Fatalf("非法动作请求了网络:%d %v", calls, err)
			}
		})
	}
}

func TestDingTalkDisabledProviderCanPrepareConfiguredAction(t *testing.T) {
	p, err := NewDingTalkIdentityProvider(&conf.Security{Dingtalk: &conf.Security_DingTalk{Enabled: false, RedirectUri: "https://admin.example.com/user/login/dingtalk/callback"}})
	if err != nil || p.Enabled() {
		t.Fatalf("禁用配置:%+v %v", p, err)
	}
	m, err := p.PrepareMessage(biz.DingTalkMessage{Title: "标题", Text: "正文", ActionTitle: "打开系统首页", ActionPath: "/welcome"})
	if err != nil || m.ActionURL != "https://admin.example.com/welcome" {
		t.Fatalf("禁用集成丢失离线预览站点:%+v %v", m, err)
	}
}
