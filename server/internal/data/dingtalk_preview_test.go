package data

import (
	"context"
	"encoding/json"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestDingTalkPrepareMarkdownIsOfflineAndMatchesSend(t *testing.T) {
	calls := 0
	var sent struct {
		Title string `json:"title"`
		Text  string `json:"text"`
	}
	p := &dingTalkIdentityProvider{redirectURI: "https://admin.example.com/user/login/dingtalk/callback?state=ignored", appToken: "cached", appTokenTill: time.Now().Add(time.Hour), client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		var body struct {
			MsgParam string `json:"msgParam"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(body.MsgParam), &sent); err != nil {
			t.Fatal(err)
		}
		return jsonResponse(200, `{"processQueryKey":"accepted"}`), nil
	})}}
	m := biz.DingTalkMarkdownMessage{Title: "[测试] 订单费用", Text: "[测试通知]\n\n### 订单费用", OrderEntryPath: "/orders/sea-export/00000000-0000-4000-8000-000000000001/fees"}
	preview, u, err := p.PrepareMarkdown(m)
	if err != nil || calls != 0 || preview.OrderEntryPath != "" || !strings.HasPrefix(u, "https://admin.example.com/orders/") {
		t.Fatalf("纯准备路径:%+v %s %v", preview, u, err)
	}
	p.enabled = true
	if err := p.SendMarkdown(context.Background(), "self", preview); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || sent.Title != preview.Title || sent.Text != preview.Text || strings.Count(sent.Text, "查看订单费用") != 1 {
		t.Fatalf("预览发送不一致:%+v", sent)
	}
	for _, origin := range []string{"", "://invalid", "ftp://admin.example.com/callback"} {
		p.redirectURI = origin
		if _, _, err := p.PrepareMarkdown(m); err == nil {
			t.Fatalf("站点无效未拒绝:%s", origin)
		}
	}
}

func TestDingTalkDisabledProviderCanPrepareConfiguredOrderLink(t *testing.T) {
	p, err := NewDingTalkIdentityProvider(&conf.Security{Dingtalk: &conf.Security_DingTalk{Enabled: false, RedirectUri: "https://admin.example.com/user/login/dingtalk/callback"}})
	if err != nil || p.Enabled() {
		t.Fatalf("禁用配置:%+v %v", p, err)
	}
	_, u, err := p.PrepareMarkdown(biz.DingTalkMarkdownMessage{Title: "标题", Text: "正文", OrderEntryPath: "/orders/sea-export/00000000-0000-4000-8000-000000000001/fees"})
	if err != nil || !strings.HasPrefix(u, "https://admin.example.com/orders/") {
		t.Fatalf("禁用集成丢失离线预览站点:%s %v", u, err)
	}
}
