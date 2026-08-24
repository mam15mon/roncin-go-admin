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
	usecase := NewAuthUsecase(&wecomAuthRepoStub{}, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, &dingTalkProviderStub{enabled: true})
	enabled, authorizeURL, state, expiresAt, err := usecase.StartDingTalkLogin()
	if err != nil {
		t.Fatalf("StartDingTalkLogin() error = %v", err)
	}
	if !enabled || state == "" || authorizeURL == "" || !expiresAt.After(time.Now()) {
		t.Fatalf("StartDingTalkLogin() = enabled %v, url %q, state %q, expires %v", enabled, authorizeURL, state, expiresAt)
	}
}

func TestAuthUsecaseDingTalkFirstLoginWaitsForAuthorization(t *testing.T) {
	organizationID := uuid.New()
	userID := uuid.New()
	repo := &wecomAuthRepoStub{
		credential: &Credential{UserID: userID, PrimaryOrganizationID: organizationID, Enabled: false},
		created:    true,
	}
	provider := &dingTalkProviderStub{enabled: true, identity: &DingTalkIdentity{UnionID: "union-id", Name: "张三"}}
	usecase := NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, provider)

	_, _, _, err := usecase.LoginDingTalk(context.Background(), "code", "state", "state", "test")
	if err != ErrDingTalkAuthorizationPending {
		t.Fatalf("LoginDingTalk() error = %v, want ErrDingTalkAuthorizationPending", err)
	}
	if repo.createdSession != nil {
		t.Fatal("pending user should not receive a session")
	}
	if len(repo.auditActions) != 1 || repo.auditActions[0] != "auth.dingtalk.register" {
		t.Fatalf("audit actions = %v", repo.auditActions)
	}
}

func TestAuthUsecaseDingTalkAuthorizedLoginCreatesSession(t *testing.T) {
	organizationID := uuid.New()
	userID := uuid.New()
	repo := &wecomAuthRepoStub{
		credential: &Credential{UserID: userID, PrimaryOrganizationID: organizationID, Enabled: true},
	}
	provider := &dingTalkProviderStub{enabled: true, identity: &DingTalkIdentity{UnionID: "union-id", Name: "张三"}}
	usecase := NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, provider)

	_, _, _, err := usecase.LoginDingTalk(context.Background(), "code", "state", "state", "test")
	if err != nil {
		t.Fatalf("LoginDingTalk() error = %v", err)
	}
	if repo.createdSession == nil || repo.createdSession.UserID != userID || repo.createdSession.OrganizationID != organizationID {
		t.Fatalf("created session = %#v", repo.createdSession)
	}
	if len(repo.auditActions) != 1 || repo.auditActions[0] != "auth.dingtalk.login" {
		t.Fatalf("audit actions = %v", repo.auditActions)
	}
}

var _ DingTalkIdentityProvider = (*dingTalkProviderStub)(nil)
