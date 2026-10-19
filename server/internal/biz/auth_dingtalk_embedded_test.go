package biz

import (
	"context"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestAuthUsecaseDingTalkEmbeddedConfigAndState(t *testing.T) {
	provider := &dingTalkProviderStub{enabled: true}
	uc := NewAuthUsecase(&authRepoStub{}, &SessionPolicy{TTL: time.Hour}, provider, nil)
	_, corp, state, expires, err := uc.StartDingTalkEmbeddedLogin()
	if err != nil || corp != "ding-corp" || state == "" || time.Until(expires) > 5*time.Minute || time.Until(expires) < 4*time.Minute {
		t.Fatalf("配置错误: %q %v", corp, err)
	}
	_, _, next, _, _ := uc.StartDingTalkEmbeddedLogin()
	if next == state {
		t.Fatal("不同配置共享 state")
	}
	for _, item := range []struct{ code, state, cookie string }{{"", "s", "s"}, {"c", "", "s"}, {"c", "s", ""}, {"c", "s", "other"}} {
		if _, err := uc.LoginDingTalkEmbedded(context.Background(), item.code, item.state, item.cookie, "ua"); err != ErrDingTalkStateInvalid {
			t.Fatalf("未拒绝无效 state: %v", err)
		}
	}
	if provider.embeddedCalls != 0 || provider.oauthCalls != 0 {
		t.Fatal("state 校验前调用了上游")
	}
	provider.enabled = false
	enabled, corp, state, expires, err := uc.StartDingTalkEmbeddedLogin()
	if err != nil || enabled || corp != "" || state != "" || !expires.IsZero() {
		t.Fatal("禁用泄漏配置")
	}
	if _, err := uc.LoginDingTalkEmbedded(context.Background(), "c", "s", "s", ""); err != ErrDingTalkDisabled {
		t.Fatalf("禁用未拒绝: %v", err)
	}
}

func TestAuthUsecaseDingTalkEmbeddedWorkspaces(t *testing.T) {
	for _, count := range []int{0, 1, 2} {
		t.Run(string(rune('0'+count)), func(t *testing.T) {
			first, second := uuid.New(), uuid.New()
			repo := &authRepoStub{credential: &Credential{UserID: uuid.New(), Enabled: true}}
			if count == 1 {
				repo.choices = []OrganizationChoice{{OrganizationID: first, OrganizationName: "A"}}
			}
			if count == 2 {
				repo.choices = []OrganizationChoice{{OrganizationID: second, OrganizationName: "B"}, {OrganizationID: first, OrganizationName: "A"}}
			}
			provider := &dingTalkProviderStub{enabled: true, identity: &DingTalkIdentity{UnionID: "u", UserID: "i", Name: "n"}}
			uc := NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, provider, nil)
			result, err := uc.LoginDingTalkEmbedded(context.Background(), "c", "s", "s", "ua")
			if count == 0 {
				if err != ErrOrganizationForbidden || repo.createdSession != nil {
					t.Fatalf("无工作台误建会话: %v", err)
				}
				return
			}
			if err != nil || result.Status != DingTalkLoginStatusAuthenticated || repo.createdSession.OrganizationID != first || repo.createdSession.UserAgent != "ua" || result.SessionToken == repo.createdSession.TokenHash || provider.oauthCalls != 0 || provider.embeddedCalls != 1 {
				t.Fatalf("工作台/通道/会话错误: %+v %v", result, err)
			}
		})
	}
}
