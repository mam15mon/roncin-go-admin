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

type dingTalkRegistrationTokenCodecStub struct {
	identity *DingTalkIdentity
}

func (s *dingTalkRegistrationTokenCodecStub) Seal(identity *DingTalkIdentity, _ time.Time) (string, error) {
	s.identity = identity
	return "registration-token", nil
}

func (s *dingTalkRegistrationTokenCodecStub) Open(token string, _ time.Time) (*DingTalkIdentity, error) {
	if token != "registration-token" || s.identity == nil {
		return nil, ErrDingTalkRegistrationExpired
	}
	return s.identity, nil
}

func (s *dingTalkProviderStub) Enabled() bool { return s.enabled }

func (s *dingTalkProviderStub) AuthorizeURL(state string) (string, error) {
	return "https://example.com/login?state=" + state, nil
}

func (s *dingTalkProviderStub) ResolveIdentity(context.Context, string) (*DingTalkIdentity, error) {
	return s.identity, nil
}

func TestAuthUsecaseStartDingTalkLogin(t *testing.T) {
	usecase := NewAuthUsecase(&wecomAuthRepoStub{}, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, &dingTalkProviderStub{enabled: true}, &dingTalkRegistrationTokenCodecStub{})
	enabled, authorizeURL, state, expiresAt, err := usecase.StartDingTalkLogin()
	if err != nil {
		t.Fatalf("StartDingTalkLogin() error = %v", err)
	}
	if !enabled || state == "" || authorizeURL == "" || !expiresAt.After(time.Now()) {
		t.Fatalf("StartDingTalkLogin() = enabled %v, url %q, state %q, expires %v", enabled, authorizeURL, state, expiresAt)
	}
}

func TestAuthUsecaseDingTalkUnregisteredLoginRequiresConfirmation(t *testing.T) {
	repo := &wecomAuthRepoStub{}
	provider := &dingTalkProviderStub{enabled: true, identity: &DingTalkIdentity{UnionID: "union-id", UserID: "user-id", CorpID: "ding-corp", Name: "张三"}}
	codec := &dingTalkRegistrationTokenCodecStub{}
	usecase := NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, provider, codec)

	result, err := usecase.LoginDingTalk(context.Background(), "code", "state", "state", "test")
	if err != nil {
		t.Fatalf("LoginDingTalk() error = %v", err)
	}
	if result.Status != DingTalkLoginStatusRegistrationRequired || result.RegistrationToken != "registration-token" || result.DisplayName != "张三" {
		t.Fatalf("LoginDingTalk() result = %#v", result)
	}
	if repo.created {
		t.Fatal("登录发现未登记人员时不应自动创建账号")
	}
}

func TestAuthUsecaseDingTalkRegistrationWaitsForAuthorization(t *testing.T) {
	organizationID := uuid.New()
	userID := uuid.New()
	repo := &wecomAuthRepoStub{
		credential: &Credential{UserID: userID, DisplayName: "张三", PrimaryOrganizationID: organizationID, Enabled: false},
		created:    true,
	}
	provider := &dingTalkProviderStub{enabled: true, identity: &DingTalkIdentity{UnionID: "union-id", UserID: "user-id", CorpID: "ding-corp", Name: "张三"}}
	codec := &dingTalkRegistrationTokenCodecStub{identity: provider.identity}
	usecase := NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, provider, codec)

	registration, err := usecase.ConfirmDingTalkRegistration(context.Background(), "registration-token")
	if err != nil {
		t.Fatalf("ConfirmDingTalkRegistration() error = %v", err)
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
	provider := &dingTalkProviderStub{enabled: true, identity: &DingTalkIdentity{UnionID: "union-id", UserID: "user-id", CorpID: "ding-corp", Name: "张三"}}
	usecase := NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, provider, &dingTalkRegistrationTokenCodecStub{})

	_, err := usecase.LoginDingTalk(context.Background(), "code", "state", "state", "test")
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
	provider := &dingTalkProviderStub{enabled: true, identity: &DingTalkIdentity{UnionID: "union-id", UserID: "user-id", CorpID: "ding-corp", Name: "张三"}}
	codec := &dingTalkRegistrationTokenCodecStub{identity: provider.identity}
	usecase := NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, provider, codec)

	if _, err := usecase.ConfirmDingTalkRegistration(context.Background(), "registration-token"); err != ErrDingTalkAlreadyRegistered {
		t.Fatalf("ConfirmDingTalkRegistration() error = %v, want ErrDingTalkAlreadyRegistered", err)
	}
}

func TestAuthUsecaseDingTalkAuthorizedLoginCreatesSession(t *testing.T) {
	organizationID := uuid.New()
	userID := uuid.New()
	repo := &wecomAuthRepoStub{
		credential: &Credential{UserID: userID, PrimaryOrganizationID: organizationID, Enabled: true},
	}
	provider := &dingTalkProviderStub{enabled: true, identity: &DingTalkIdentity{UnionID: "union-id", UserID: "user-id", CorpID: "ding-corp", Name: "张三"}}
	usecase := NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, provider, &dingTalkRegistrationTokenCodecStub{})

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
var _ DingTalkRegistrationTokenCodec = (*dingTalkRegistrationTokenCodecStub)(nil)
