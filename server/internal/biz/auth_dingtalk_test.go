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

func TestAuthUsecaseDingTalkUnregisteredLoginIsRejected(t *testing.T) {
	repo := &wecomAuthRepoStub{}
	provider := &dingTalkProviderStub{enabled: true, identity: &DingTalkIdentity{UnionID: "union-id", CorpID: "ding-corp", Name: "张三"}}
	usecase := NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, provider)

	_, _, _, err := usecase.LoginDingTalk(context.Background(), "code", "state", "state", "test")
	if err != ErrDingTalkNotRegistered {
		t.Fatalf("LoginDingTalk() error = %v, want ErrDingTalkNotRegistered", err)
	}
}

func TestAuthUsecaseDingTalkRegistrationWaitsForAuthorization(t *testing.T) {
	organizationID := uuid.New()
	userID := uuid.New()
	repo := &wecomAuthRepoStub{
		credential: &Credential{UserID: userID, DisplayName: "张三", PrimaryOrganizationID: organizationID, Enabled: false},
		created:    true,
	}
	provider := &dingTalkProviderStub{enabled: true, identity: &DingTalkIdentity{UnionID: "union-id", CorpID: "ding-corp", Name: "张三"}}
	usecase := NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, provider)

	registration, err := usecase.RegisterDingTalk(context.Background(), "code", "state", "state")
	if err != nil {
		t.Fatalf("RegisterDingTalk() error = %v", err)
	}
	if registration.DisplayName != "张三" || registration.Status != "PENDING" {
		t.Fatalf("registration = %#v", registration)
	}
	if len(repo.auditActions) != 1 || repo.auditActions[0] != "auth.dingtalk.register" {
		t.Fatalf("audit actions = %v", repo.auditActions)
	}
}

func TestAuthUsecaseDingTalkPendingLoginDoesNotCreateSession(t *testing.T) {
	organizationID := uuid.New()
	userID := uuid.New()
	repo := &wecomAuthRepoStub{
		credential: &Credential{UserID: userID, DisplayName: "张三", PrimaryOrganizationID: organizationID, Enabled: false},
	}
	provider := &dingTalkProviderStub{enabled: true, identity: &DingTalkIdentity{UnionID: "union-id", CorpID: "ding-corp", Name: "张三"}}
	usecase := NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, provider)

	_, _, _, err := usecase.LoginDingTalk(context.Background(), "code", "state", "state", "test")
	if err != ErrDingTalkAuthorizationPending {
		t.Fatalf("LoginDingTalk() error = %v, want ErrDingTalkAuthorizationPending", err)
	}
	if repo.createdSession != nil {
		t.Fatal("待授权账号不应创建登录会话")
	}
}

func TestAuthUsecaseDingTalkRegistrationRejectsEnabledAccount(t *testing.T) {
	repo := &wecomAuthRepoStub{
		credential: &Credential{UserID: uuid.New(), DisplayName: "张三", PrimaryOrganizationID: uuid.New(), Enabled: true},
	}
	provider := &dingTalkProviderStub{enabled: true, identity: &DingTalkIdentity{UnionID: "union-id", CorpID: "ding-corp", Name: "张三"}}
	usecase := NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, provider)

	if _, err := usecase.RegisterDingTalk(context.Background(), "code", "state", "state"); err != ErrDingTalkAlreadyRegistered {
		t.Fatalf("RegisterDingTalk() error = %v, want ErrDingTalkAlreadyRegistered", err)
	}
}

func TestAuthUsecaseDingTalkAuthorizedLoginCreatesSession(t *testing.T) {
	organizationID := uuid.New()
	userID := uuid.New()
	repo := &wecomAuthRepoStub{
		credential: &Credential{UserID: userID, PrimaryOrganizationID: organizationID, Enabled: true},
	}
	provider := &dingTalkProviderStub{enabled: true, identity: &DingTalkIdentity{UnionID: "union-id", CorpID: "ding-corp", Name: "张三"}}
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
