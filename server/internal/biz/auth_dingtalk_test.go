package biz

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type dingTalkProviderStub struct {
	enabled  bool
	identity *DingTalkIdentity
}

func (s *dingTalkProviderStub) Enabled() bool { return s.enabled }

func (s *dingTalkProviderStub) AuthorizeURL(state string) (string, error) {
	return "https://example.com/login?state=" + state, nil
}

func (s *dingTalkProviderStub) ResolveIdentity(context.Context, string) (*DingTalkIdentity, error) {
	return s.identity, nil
}

func TestAuthUsecaseStartDingTalkLogin(t *testing.T) {
	usecase := NewAuthUsecase(&wecomAuthRepoStub{}, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, &dingTalkProviderStub{enabled: true}, nil)
	enabled, authorizeURL, state, expiresAt, err := usecase.StartDingTalkLogin()
	if err != nil {
		t.Fatalf("StartDingTalkLogin() error = %v", err)
	}
	if !enabled || state == "" || authorizeURL == "" || !expiresAt.After(time.Now()) {
		t.Fatalf("StartDingTalkLogin() = enabled %v, url %q, state %q, expires %v", enabled, authorizeURL, state, expiresAt)
	}
}

func TestAuthUsecaseDingTalkNotOpenedDoesNotCreateSession(t *testing.T) {
	cases := []struct {
		name       string
		credential *Credential
		status     DingTalkLoginStatus
	}{
		{"未同步", nil, DingTalkLoginStatusNotSynced},
		{"待分配", &Credential{UserID: uuid.New(), PendingAssignment: true, Enabled: false}, DingTalkLoginStatusPendingAssignment},
		{"已停用", &Credential{UserID: uuid.New(), Enabled: false}, DingTalkLoginStatusDisabled},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			repo := &wecomAuthRepoStub{credential: item.credential}
			provider := &dingTalkProviderStub{enabled: true, identity: &DingTalkIdentity{UnionID: "union", UserID: "user"}}
			uc := NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, provider, nil)
			result, err := uc.LoginDingTalk(context.Background(), "code", "state", "state", "test")
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != item.status || result.SessionToken != "" || result.Message == "" || repo.createdSession != nil || repo.created {
				t.Fatalf("未开通状态误建会话: %+v", result)
			}
		})
	}
}

func TestAuthUsecaseDingTalkAuthorizedLoginCreatesSession(t *testing.T) {
	organizationID := uuid.New()
	userID := uuid.New()
	repo := &wecomAuthRepoStub{
		credential: &Credential{UserID: userID, PrimaryOrganizationID: organizationID, Enabled: true},
	}
	provider := &dingTalkProviderStub{enabled: true, identity: &DingTalkIdentity{UnionID: "union-id", UserID: "user-id", CorpID: "ding-corp", Name: "张三"}}
	usecase := NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, provider, nil)
	result, err := usecase.LoginDingTalk(context.Background(), "code", "state", "state", "test")
	if err != nil {
		t.Fatalf("LoginDingTalk() error = %v", err)
	}
	if result.Status != DingTalkLoginStatusAuthenticated || result.Principal == nil {
		t.Fatalf("LoginDingTalk() result = %#v", result)
	}
	if repo.createdSession == nil || repo.createdSession.UserID != userID || repo.createdSession.OrganizationID != organizationID {
		t.Fatalf("created session = %#v", repo.createdSession)
	}
	if len(repo.auditActions) != 1 || repo.auditActions[0] != "auth.dingtalk.login" {
		t.Fatalf("audit actions = %v", repo.auditActions)
	}
}

var _ DingTalkIdentityProvider = (*dingTalkProviderStub)(nil)
