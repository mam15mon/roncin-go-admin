package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/security/password"
)

// authServiceRepoStub 记录会话建立/轮转调用，支撑登录组织选择与切换器的传输层测试。
type authServiceRepoStub struct {
	credential      *biz.Credential
	choices         []biz.OrganizationChoice
	principalsByOrg map[uuid.UUID]*biz.Principal
	createdSessions []*biz.Session
	rotatedFromHash []string
	rotatedSessions []*biz.Session
	rotateAudits    []*biz.AuditEvent
}

func (s *authServiceRepoStub) FindCredential(context.Context, string) (*biz.Credential, error) {
	if s.credential == nil {
		return nil, biz.ErrInvalidCredentials
	}
	return s.credential, nil
}

func (*authServiceRepoStub) LoginRateLimitExceeded(context.Context, []string, time.Time, time.Duration, int) (bool, error) {
	return false, nil
}

func (*authServiceRepoStub) RecordLoginFailure(context.Context, []string, time.Time, time.Duration, int, *biz.AuditEvent) (bool, error) {
	return false, nil
}

func (*authServiceRepoStub) FindOrCreateWeComCredential(context.Context, *biz.WeComIdentity, *biz.AuditEvent) (*biz.Credential, bool, error) {
	return nil, false, biz.ErrWeComDisabled
}

func (*authServiceRepoStub) FindDingTalkCredential(context.Context, *biz.DingTalkIdentity) (*biz.Credential, error) {
	return nil, biz.ErrDingTalkNotRegistered
}

func (*authServiceRepoStub) RegisterDingTalkCredential(context.Context, *biz.DingTalkIdentity, *uuid.UUID, *biz.DingTalkApproverNotice, *biz.AuditEvent) (*biz.Credential, bool, error) {
	return nil, false, biz.ErrDingTalkDisabled
}

func (s *authServiceRepoStub) ListEnabledMembershipOrganizations(context.Context, uuid.UUID) ([]biz.OrganizationChoice, error) {
	return s.choices, nil
}

func (s *authServiceRepoStub) ResolvePrincipal(_ context.Context, userID, organizationID uuid.UUID) (*biz.Principal, error) {
	if principal, ok := s.principalsByOrg[organizationID]; ok {
		return principal, nil
	}
	return &biz.Principal{UserID: userID, Organization: biz.Organization{ID: organizationID}}, nil
}

func (s *authServiceRepoStub) CreateSession(_ context.Context, session *biz.Session, _ string, _ *biz.AuditEvent) error {
	s.createdSessions = append(s.createdSessions, session)
	return nil
}

func (*authServiceRepoStub) FindSession(context.Context, string, time.Time) (*biz.Session, error) {
	return nil, biz.ErrSessionExpired
}

func (s *authServiceRepoStub) RotateSession(_ context.Context, fromHash string, next *biz.Session, _ time.Time, audit *biz.AuditEvent) error {
	s.rotatedFromHash = append(s.rotatedFromHash, fromHash)
	s.rotatedSessions = append(s.rotatedSessions, next)
	s.rotateAudits = append(s.rotateAudits, audit)
	return nil
}

func (*authServiceRepoStub) RevokeSession(context.Context, string, time.Time, *biz.AuditEvent) error {
	return nil
}

func newAuthServiceForTest(repo *authServiceRepoStub) *AuthService {
	policy := &biz.SessionPolicy{TTL: time.Hour}
	return NewAuthService(biz.NewAuthUsecase(repo, policy, nil, nil, nil, nil, nil, nil), policy)
}

func authServicePasswordHash(t *testing.T) *string {
	t.Helper()
	hash, err := password.Hash("correct-password")
	if err != nil {
		t.Fatalf("password.Hash() error = %v", err)
	}
	return &hash
}

func TestAuthServiceLoginResponseIncludesOrganizationChoices(t *testing.T) {
	primaryID, branchID := uuid.New(), uuid.New()
	userID := uuid.New()
	repo := &authServiceRepoStub{
		credential: &biz.Credential{UserID: userID, PasswordHash: authServicePasswordHash(t), Enabled: true, PrimaryOrganizationID: primaryID},
		choices: []biz.OrganizationChoice{
			{OrganizationID: primaryID, OrganizationName: "北京公司", OrganizationCode: "BJ", IsDefault: true},
			{OrganizationID: branchID, OrganizationName: "成都公司", OrganizationCode: "CD"},
		},
		principalsByOrg: map[uuid.UUID]*biz.Principal{primaryID: {UserID: userID, Organization: biz.Organization{ID: primaryID, Name: "北京公司", Code: "BJ"}}},
	}

	response, err := newAuthServiceForTest(repo).Login(context.Background(), requestWithOrganization("admin", "correct-password", ""))
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if response.Data.CurrentOrganization.Id != primaryID.String() {
		t.Fatalf("默认登录当前组织 = %s，期望 %s", response.Data.CurrentOrganization.Id, primaryID)
	}
	if len(response.OrganizationChoices) != 2 {
		t.Fatalf("登录响应候选组织 = %#v", response.OrganizationChoices)
	}
	first, second := response.OrganizationChoices[0], response.OrganizationChoices[1]
	if first.OrganizationId != primaryID.String() || first.OrganizationName != "北京公司" || first.OrganizationCode != "BJ" || !first.IsDefault {
		t.Fatalf("默认候选映射不完整: %#v", first)
	}
	if second.OrganizationId != branchID.String() || second.IsDefault {
		t.Fatalf("非默认候选映射不完整: %#v", second)
	}
}

func TestAuthServiceLoginPassesThroughRequestedOrganization(t *testing.T) {
	primaryID, branchID := uuid.New(), uuid.New()
	userID := uuid.New()
	repo := &authServiceRepoStub{
		credential: &biz.Credential{UserID: userID, PasswordHash: authServicePasswordHash(t), Enabled: true, PrimaryOrganizationID: primaryID},
		choices: []biz.OrganizationChoice{
			{OrganizationID: primaryID, OrganizationName: "北京公司", OrganizationCode: "BJ", IsDefault: true},
			{OrganizationID: branchID, OrganizationName: "成都公司", OrganizationCode: "CD"},
		},
		principalsByOrg: map[uuid.UUID]*biz.Principal{branchID: {UserID: userID, Organization: biz.Organization{ID: branchID, Name: "成都公司", Code: "CD"}}},
	}

	response, err := newAuthServiceForTest(repo).Login(context.Background(), requestWithOrganization("admin", "correct-password", branchID.String()))
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if response.Data.CurrentOrganization.Id != branchID.String() {
		t.Fatalf("所选组织登录当前组织 = %s，期望 %s", response.Data.CurrentOrganization.Id, branchID)
	}
	if len(repo.createdSessions) != 1 || repo.createdSessions[0].OrganizationID != branchID {
		t.Fatalf("会话应建立在所选组织上: %#v", repo.createdSessions)
	}
}

func TestAuthServiceLoginRejectsInvalidOrganizationID(t *testing.T) {
	repo := &authServiceRepoStub{credential: &biz.Credential{UserID: uuid.New(), PasswordHash: authServicePasswordHash(t), Enabled: true, PrimaryOrganizationID: uuid.New()}}

	_, err := newAuthServiceForTest(repo).Login(context.Background(), requestWithOrganization("admin", "correct-password", "not-a-uuid"))
	if err != biz.ErrAuthOrganizationInvalid {
		t.Fatalf("非法组织标识登录错误 = %v，期望 ErrAuthOrganizationInvalid", err)
	}
	if len(repo.createdSessions) != 0 {
		t.Fatalf("非法组织标识不得创建会话: %#v", repo.createdSessions)
	}
}

func TestAuthServiceSwitchOrganizationResponseSameShapeAsLogin(t *testing.T) {
	sourceID, targetID := uuid.New(), uuid.New()
	userID := uuid.New()
	repo := &authServiceRepoStub{
		choices: []biz.OrganizationChoice{
			{OrganizationID: sourceID, OrganizationName: "北京公司", OrganizationCode: "BJ", IsDefault: true},
			{OrganizationID: targetID, OrganizationName: "成都公司", OrganizationCode: "CD"},
		},
		principalsByOrg: map[uuid.UUID]*biz.Principal{targetID: {UserID: userID, Organization: biz.Organization{ID: targetID, Name: "成都公司", Code: "CD"}}},
	}
	ctx := biz.WithPrincipal(context.Background(), &biz.Principal{UserID: userID, Organization: biz.Organization{ID: sourceID}, SessionTokenHash: "current-token-hash"})

	response, err := newAuthServiceForTest(repo).SwitchOrganization(ctx, switchRequest(targetID.String()))
	if err != nil {
		t.Fatalf("SwitchOrganization() error = %v", err)
	}
	if response.Data.CurrentOrganization.Id != targetID.String() {
		t.Fatalf("切换后当前组织 = %s，期望 %s", response.Data.CurrentOrganization.Id, targetID)
	}
	if len(response.OrganizationChoices) != 2 || response.OrganizationChoices[0].OrganizationId != sourceID.String() {
		t.Fatalf("切换响应应与登录同构携带候选列表: %#v", response.OrganizationChoices)
	}
	if len(repo.rotatedFromHash) != 1 || repo.rotatedFromHash[0] != "current-token-hash" {
		t.Fatalf("切换应轮转当前令牌: %v", repo.rotatedFromHash)
	}
	if len(repo.rotateAudits) != 1 || repo.rotateAudits[0].Action != "auth.organization.switch" {
		t.Fatalf("切换审计异常: %#v", repo.rotateAudits)
	}
}

func TestAuthServiceSwitchOrganizationForbidden(t *testing.T) {
	sourceID := uuid.New()
	repo := &authServiceRepoStub{
		choices: []biz.OrganizationChoice{{OrganizationID: sourceID, OrganizationName: "北京公司", OrganizationCode: "BJ", IsDefault: true}},
	}
	ctx := biz.WithPrincipal(context.Background(), &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: sourceID}, SessionTokenHash: "current-token-hash"})

	_, err := newAuthServiceForTest(repo).SwitchOrganization(ctx, switchRequest(uuid.New().String()))
	if err != biz.ErrAuthOrganizationForbidden {
		t.Fatalf("候选集外切换错误 = %v，期望 ErrAuthOrganizationForbidden", err)
	}
	if len(repo.rotatedSessions) != 0 {
		t.Fatalf("拒绝切换不得轮转会话: %#v", repo.rotatedSessions)
	}
}

func requestWithOrganization(username, plainPassword, organizationID string) *v1.LoginRequest {
	request := &v1.LoginRequest{Username: username, Password: plainPassword}
	if organizationID != "" {
		request.OrganizationId = &organizationID
	}
	return request
}

func switchRequest(organizationID string) *v1.SwitchOrganizationRequest {
	return &v1.SwitchOrganizationRequest{OrganizationId: organizationID}
}
