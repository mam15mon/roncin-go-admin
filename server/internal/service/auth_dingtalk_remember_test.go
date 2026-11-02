package service

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type rememberDingTalkProvider struct{ bootstrapDingTalkProvider }

func (*rememberDingTalkProvider) ResolveIdentity(context.Context, string) (*biz.DingTalkIdentity, error) {
	return &biz.DingTalkIdentity{UnionID: "union", UserID: "ding-user", CorpID: "ding-corp", Name: "成员"}, nil
}

type rememberDingTalkRepo struct{ authServiceRepoStub }

func (r *rememberDingTalkRepo) FindDingTalkCredential(context.Context, *biz.DingTalkIdentity) (*biz.Credential, error) {
	return r.credential, nil
}

func TestDingTalkLoginRememberMeUsesRequestPreference(t *testing.T) {
	for _, tc := range []struct {
		name     string
		remember *bool
		want     bool
	}{{"缺省", nil, true}, {"自动登录", boolPointer(true), true}, {"当前会话", boolPointer(false), false}} {
		t.Run(tc.name, func(t *testing.T) {
			userID, orgID := uuid.New(), uuid.New()
			repo := &rememberDingTalkRepo{authServiceRepoStub: authServiceRepoStub{credential: &biz.Credential{UserID: userID, Enabled: true}, choices: []biz.OrganizationChoice{{OrganizationID: orgID}}}}
			policy := &biz.SessionPolicy{CookieName: "roncin_session", TTL: 30 * 24 * time.Hour, Secure: true, SameSite: "lax"}
			provider := &rememberDingTalkProvider{bootstrapDingTalkProvider: bootstrapDingTalkProvider{enabled: true}}
			s := NewAuthService(biz.NewAuthUsecase(repo, policy, provider, nil), policy, nil)
			server := kratoshttp.NewServer(kratoshttp.RequestDecoder(authCookieProtoDecoder))
			v1.RegisterAuthServiceHTTPServer(server, s)
			payload := map[string]any{"authCode": "oauth-code", "state": "current"}
			if tc.remember != nil {
				payload["rememberMe"] = *tc.remember
			}
			body, _ := json.Marshal(payload)
			response := postAuthCookieRequest(server, "/api/v1/auth/dingtalk/login", string(body), "roncin_session_dingtalk_state=current")
			if response.Code != 200 || len(repo.createdSessions) != 1 {
				t.Fatalf("扫码登录失败: %d %s", response.Code, response.Body.String())
			}
			cookies := response.Result().Cookies()
			if len(cookies) != 3 {
				t.Fatal("扫码state清理与登录Cookie数量错误")
			}
			sessionCookies := []*http.Cookie{}
			token := ""
			clearedState := false
			for _, c := range cookies {
				if c.Name == "roncin_session_dingtalk_state" {
					if clearedState || c.Value != "" || c.MaxAge != -1 || !c.HttpOnly || !c.Secure || c.Path != "/" || c.SameSite != http.SameSiteLaxMode {
						t.Fatal("扫码state未正确清理")
					}
					clearedState = true
				} else {
					sessionCookies = append(sessionCookies, c)
					if c.Name == "roncin_session" {
						token = c.Value
					}
				}
			}
			if !clearedState || token == "" {
				t.Fatal("state清理或会话凭证缺失")
			}
			assertSessionCookiePair(t, sessionCookies, token, tc.want, false)
		})
	}
}
