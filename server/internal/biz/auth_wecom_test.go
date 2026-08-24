package biz

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type wecomProviderStub struct {
	enabled  bool
	identity *WeComIdentity
}

func (s *wecomProviderStub) Enabled() bool { return s.enabled }

func (s *wecomProviderStub) AuthorizeURL(state string) (string, error) {
	return "https://example.com/login?state=" + state, nil
}

func (s *wecomProviderStub) ResolveIdentity(context.Context, string) (*WeComIdentity, error) {
	return s.identity, nil
}

type wecomAuthRepoStub struct {
	credential     *Credential
	created        bool
	createdSession *Session
	auditActions   []string
}

func (s *wecomAuthRepoStub) FindCredential(context.Context, string) (*Credential, error) {
	return nil, ErrInvalidCredentials
}

func (s *wecomAuthRepoStub) FindOrCreateWeComCredential(context.Context, *WeComIdentity) (*Credential, bool, error) {
	return s.credential, s.created, nil
}

func (s *wecomAuthRepoStub) FindOrCreateDingTalkCredential(context.Context, *DingTalkIdentity) (*Credential, bool, error) {
	return s.credential, s.created, nil
}

func (s *wecomAuthRepoStub) ResolvePrincipal(_ context.Context, userID, organizationID uuid.UUID) (*Principal, error) {
	return &Principal{UserID: userID, Organization: Organization{ID: organizationID}}, nil
}

func (s *wecomAuthRepoStub) CreateSession(_ context.Context, session *Session) error {
	s.createdSession = session
	return nil
}

func (s *wecomAuthRepoStub) FindSession(context.Context, string, time.Time) (*Session, error) {
	return nil, ErrSessionExpired
}

func (s *wecomAuthRepoStub) SwitchSessionOrganization(context.Context, string, uuid.UUID, uuid.UUID, time.Time) error {
	return nil
}

func (s *wecomAuthRepoStub) RevokeSession(context.Context, string, time.Time) error { return nil }

func (s *wecomAuthRepoStub) WriteAudit(_ context.Context, event *AuditEvent) error {
	s.auditActions = append(s.auditActions, event.Action)
	return nil
}

func TestAuthUsecaseStartWeComLogin(t *testing.T) {
	usecase := NewAuthUsecase(&wecomAuthRepoStub{}, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{enabled: true}, &dingTalkProviderStub{})
	enabled, authorizeURL, state, expiresAt, err := usecase.StartWeComLogin()
	if err != nil {
		t.Fatalf("StartWeComLogin() error = %v", err)
	}
	if !enabled || state == "" || authorizeURL == "" || !expiresAt.After(time.Now()) {
		t.Fatalf("StartWeComLogin() = enabled %v, url %q, state %q, expires %v", enabled, authorizeURL, state, expiresAt)
	}
}

func TestAuthUsecaseWeComFirstLoginWaitsForAuthorization(t *testing.T) {
	organizationID := uuid.New()
	userID := uuid.New()
	repo := &wecomAuthRepoStub{
		credential: &Credential{UserID: userID, PrimaryOrganizationID: organizationID, Enabled: false},
		created:    true,
	}
	provider := &wecomProviderStub{enabled: true, identity: &WeComIdentity{UserID: "zhangsan", Name: "张三"}}
	usecase := NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, provider, &dingTalkProviderStub{})

	_, _, _, err := usecase.LoginWeCom(context.Background(), "code", "state", "state", "test")
	if err != ErrWeComAuthorizationPending {
		t.Fatalf("LoginWeCom() error = %v, want ErrWeComAuthorizationPending", err)
	}
	if repo.createdSession != nil {
		t.Fatal("pending user should not receive a session")
	}
	if len(repo.auditActions) != 1 || repo.auditActions[0] != "auth.wecom.register" {
		t.Fatalf("audit actions = %v", repo.auditActions)
	}
}

func TestAuthUsecaseWeComAuthorizedLoginCreatesSession(t *testing.T) {
	organizationID := uuid.New()
	userID := uuid.New()
	repo := &wecomAuthRepoStub{
		credential: &Credential{UserID: userID, PrimaryOrganizationID: organizationID, Enabled: true},
	}
	provider := &wecomProviderStub{enabled: true, identity: &WeComIdentity{UserID: "zhangsan", Name: "张三"}}
	usecase := NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, provider, &dingTalkProviderStub{})

	_, _, _, err := usecase.LoginWeCom(context.Background(), "code", "state", "state", "test")
	if err != nil {
		t.Fatalf("LoginWeCom() error = %v", err)
	}
	if repo.createdSession == nil || repo.createdSession.UserID != userID || repo.createdSession.OrganizationID != organizationID {
		t.Fatalf("created session = %#v", repo.createdSession)
	}
	if len(repo.auditActions) != 1 || repo.auditActions[0] != "auth.wecom.login" {
		t.Fatalf("audit actions = %v", repo.auditActions)
	}
}

var _ WeComIdentityProvider = (*wecomProviderStub)(nil)
var _ AuthRepo = (*wecomAuthRepoStub)(nil)
