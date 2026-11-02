package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"google.golang.org/protobuf/encoding/protojson"
)

type transportQRProvider struct {
	bootstrapDingTalkProvider
	calls int
}

func (p *transportQRProvider) PollQRCode(context.Context, string) (*biz.DingTalkQRCodePollResult, error) {
	p.calls++
	return &biz.DingTalkQRCodePollResult{Status: biz.DingTalkQRCodeConfirmed, AuthCode: "confirmed-code"}, nil
}

func TestDingTalkQRPollHTTPStateCookieAndNoStore(t *testing.T) {
	for _, tc := range []struct {
		name, body, cookie string
		status             int
	}{
		{"匹配", `{"state":"current"}`, "roncin_session_dingtalk_state=current", 200},
		{"刷新旧请求", `{"state":"old"}`, "roncin_session_dingtalk_state=current", 401},
		{"无Cookie", `{"state":"current"}`, "", 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := &transportQRProvider{bootstrapDingTalkProvider: bootstrapDingTalkProvider{enabled: true}}
			policy := &biz.SessionPolicy{CookieName: "roncin_session", TTL: time.Hour}
			s := NewAuthService(biz.NewAuthUsecase(nil, policy, provider, nil), policy, nil)
			server := kratoshttp.NewServer()
			v1.RegisterAuthServiceHTTPServer(server, s)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/dingtalk/qr/poll", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Cookie", tc.cookie)
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, req)
			if recorder.Code != tc.status || recorder.Header().Get("Cache-Control") != "no-store" || recorder.Header().Get("Pragma") != "no-cache" || len(recorder.Result().Cookies()) != 0 {
				t.Fatalf("轮询传输契约错误: %d %v", recorder.Code, recorder.Header())
			}
			if tc.status == 200 {
				var reply v1.PollDingTalkQRCodeResponse
				if err := protojson.Unmarshal(recorder.Body.Bytes(), &reply); err != nil {
					t.Fatal(err)
				}
				if !reply.Success || reply.Data.Status != v1.DingTalkQRCodeStatus_DING_TALK_QR_CODE_STATUS_CONFIRMED || reply.Data.GetAuthCode() != "confirmed-code" || provider.calls != 1 {
					t.Fatal("确认授权码没有按契约返回")
				}
			} else if provider.calls != 0 {
				t.Fatalf("不匹配state仍访问上游%d次", provider.calls)
			}
		})
	}
}
