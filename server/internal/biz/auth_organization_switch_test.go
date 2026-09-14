package biz

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/security/password"
)

// organizationSwitchRepoStub 记录会话建立/轮转调用，并按组织返回可配置的主体。
type organizationSwitchRepoStub struct {
	wecomAuthRepoStub
	credential         *Credential
	choices            []OrganizationChoice
	allChoices         []OrganizationChoice
	principalsByOrg    map[uuid.UUID]*Principal
	createdSessions    []*Session
	createAuditActions []string
	rotatedFromHash    []string
	rotatedSessions    []*Session
	rotateAudits       []*AuditEvent
}

func (s *organizationSwitchRepoStub) FindCredential(context.Context, string) (*Credential, error) {
	if s.credential == nil {
		return nil, ErrInvalidCredentials
	}
	return s.credential, nil
}

func (s *organizationSwitchRepoStub) ListEnabledMembershipOrganizations(context.Context, uuid.UUID) ([]OrganizationChoice, error) {
	return s.choices, nil
}

func (s *organizationSwitchRepoStub) ListEnabledOrganizations(context.Context, uuid.UUID) ([]OrganizationChoice, error) {
	return s.allChoices, nil
}

func (s *organizationSwitchRepoStub) ResolvePrincipal(_ context.Context, userID, organizationID uuid.UUID) (*Principal, error) {
	if principal, ok := s.principalsByOrg[organizationID]; ok {
		return principal, nil
	}
	return &Principal{UserID: userID, Organization: Organization{ID: organizationID}}, nil
}

func (s *organizationSwitchRepoStub) CreateSession(_ context.Context, session *Session, _ string, audit *AuditEvent) error {
	s.createdSessions = append(s.createdSessions, session)
	s.createAuditActions = append(s.createAuditActions, audit.Action)
	return nil
}

func (s *organizationSwitchRepoStub) RotateSession(_ context.Context, fromHash string, next *Session, _ time.Time, audit *AuditEvent) error {
	s.rotatedFromHash = append(s.rotatedFromHash, fromHash)
	s.rotatedSessions = append(s.rotatedSessions, next)
	s.rotateAudits = append(s.rotateAudits, audit)
	return nil
}

func newOrganizationSwitchUsecase(repo *organizationSwitchRepoStub) *AuthUsecase {
	return NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, &dingTalkProviderStub{}, &dingTalkRegistrationTokenCodecStub{}, nil, nil, nil)
}

func organizationSwitchPasswordHash(t *testing.T) string {
	t.Helper()
	hash, err := password.Hash("correct-password")
	if err != nil {
		t.Fatalf("password.Hash() error = %v", err)
	}
	return hash
}

func TestAuthUsecaseLoginUsesPrimaryOrganizationAndReturnsSortedChoices(t *testing.T) {
	primaryID, branchID := uuid.New(), uuid.New()
	userID := uuid.New()
	passwordHash := organizationSwitchPasswordHash(t)
	repo := &organizationSwitchRepoStub{
		credential: &Credential{UserID: userID, PasswordHash: &passwordHash, Enabled: true, PrimaryOrganizationID: primaryID},
		// 故意乱序返回，排序必须由用例统一完成。
		choices: []OrganizationChoice{
			{OrganizationID: branchID, OrganizationName: "成都公司", OrganizationCode: "CD"},
			{OrganizationID: primaryID, OrganizationName: "北京公司", OrganizationCode: "BJ", IsDefault: true},
		},
		principalsByOrg: map[uuid.UUID]*Principal{primaryID: {UserID: userID, Organization: Organization{ID: primaryID, Name: "北京公司"}}},
	}

	result, err := newOrganizationSwitchUsecase(repo).Login(context.Background(), "admin", "correct-password", uuid.Nil, "test", "127.0.0.1")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.Token == "" || result.ExpiresAt.Before(time.Now()) {
		t.Fatalf("Login() 令牌/过期时间异常: %#v", result)
	}
	if result.Principal.Organization.ID != primaryID {
		t.Fatalf("未指定组织时应沿用默认组织，实际 %v", result.Principal.Organization.ID)
	}
	if len(repo.createdSessions) != 1 || repo.createdSessions[0].OrganizationID != primaryID {
		t.Fatalf("会话应建立在默认组织上: %#v", repo.createdSessions)
	}
	if len(repo.createAuditActions) != 1 || repo.createAuditActions[0] != "auth.login" {
		t.Fatalf("登录审计 = %v", repo.createAuditActions)
	}
	if len(result.OrganizationChoices) != 2 || result.OrganizationChoices[0].OrganizationID != primaryID || !result.OrganizationChoices[0].IsDefault || result.OrganizationChoices[1].OrganizationID != branchID {
		t.Fatalf("候选组织应默认置首并按名称排序: %#v", result.OrganizationChoices)
	}
}

func TestAuthUsecaseLoginWithExplicitOrganizationEntersChosenOrganization(t *testing.T) {
	primaryID, branchID := uuid.New(), uuid.New()
	userID := uuid.New()
	passwordHash := organizationSwitchPasswordHash(t)
	repo := &organizationSwitchRepoStub{
		credential: &Credential{UserID: userID, PasswordHash: &passwordHash, Enabled: true, PrimaryOrganizationID: primaryID},
		choices: []OrganizationChoice{
			{OrganizationID: primaryID, OrganizationName: "北京公司", OrganizationCode: "BJ", IsDefault: true},
			{OrganizationID: branchID, OrganizationName: "成都公司", OrganizationCode: "CD"},
		},
		principalsByOrg: map[uuid.UUID]*Principal{branchID: {UserID: userID, Organization: Organization{ID: branchID, Name: "成都公司"}}},
	}

	result, err := newOrganizationSwitchUsecase(repo).Login(context.Background(), "admin", "correct-password", branchID, "test", "127.0.0.1")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.Principal.Organization.ID != branchID {
		t.Fatalf("显式组织登录主体组织 = %v，期望 %v", result.Principal.Organization.ID, branchID)
	}
	if len(repo.createdSessions) != 1 || repo.createdSessions[0].OrganizationID != branchID {
		t.Fatalf("会话应建立在所选组织上: %#v", repo.createdSessions)
	}
}

func TestAuthUsecaseLoginRejectsOrganizationOutsideCandidates(t *testing.T) {
	primaryID := uuid.New()
	passwordHash := organizationSwitchPasswordHash(t)
	repo := &organizationSwitchRepoStub{
		credential: &Credential{UserID: uuid.New(), PasswordHash: &passwordHash, Enabled: true, PrimaryOrganizationID: primaryID},
		choices:    []OrganizationChoice{{OrganizationID: primaryID, OrganizationName: "北京公司", OrganizationCode: "BJ", IsDefault: true}},
	}

	result, err := newOrganizationSwitchUsecase(repo).Login(context.Background(), "admin", "correct-password", uuid.New(), "test", "127.0.0.1")
	if err != ErrAuthOrganizationInvalid {
		t.Fatalf("候选集外组织登录错误 = %v，期望 ErrAuthOrganizationInvalid", err)
	}
	if result != nil {
		t.Fatalf("校验失败不应返回会话结果: %#v", result)
	}
	if len(repo.createdSessions) != 0 {
		t.Fatalf("校验失败不应创建会话: %#v", repo.createdSessions)
	}
}

func TestAuthUsecaseSwitchOrganizationRotatesCurrentSessionToken(t *testing.T) {
	sourceID, targetID := uuid.New(), uuid.New()
	userID := uuid.New()
	repo := &organizationSwitchRepoStub{
		choices: []OrganizationChoice{
			{OrganizationID: sourceID, OrganizationName: "北京公司", OrganizationCode: "BJ", IsDefault: true},
			{OrganizationID: targetID, OrganizationName: "成都公司", OrganizationCode: "CD"},
		},
		principalsByOrg: map[uuid.UUID]*Principal{targetID: {UserID: userID, Organization: Organization{ID: targetID, Name: "成都公司"}, RoleGrants: []RoleGrant{roleGrant("branch-operator", DataScopeOrganization, []string{"business.order.se.read"})}}},
	}
	current := &Principal{UserID: userID, Organization: Organization{ID: sourceID}, SessionTokenHash: "current-token-hash"}

	result, err := newOrganizationSwitchUsecase(repo).SwitchOrganization(context.Background(), current, targetID)
	if err != nil {
		t.Fatalf("SwitchOrganization() error = %v", err)
	}
	if result.Token == "" || result.ExpiresAt.Before(time.Now()) {
		t.Fatalf("切换结果令牌/过期时间异常: %#v", result)
	}
	if len(repo.rotatedFromHash) != 1 || repo.rotatedFromHash[0] != "current-token-hash" {
		t.Fatalf("轮转应失效当前令牌: %v", repo.rotatedFromHash)
	}
	if len(repo.rotatedSessions) != 1 {
		t.Fatalf("轮转会话记录 = %#v", repo.rotatedSessions)
	}
	rotated := repo.rotatedSessions[0]
	if rotated.UserID != userID || rotated.OrganizationID != targetID {
		t.Fatalf("新会话应指向目标组织: %#v", rotated)
	}
	if result.Principal.Organization.ID != targetID || result.Principal.SessionTokenHash != rotated.TokenHash {
		t.Fatalf("切换后主体应随目标组织重算并携带新令牌哈希: %#v", result.Principal)
	}
	if got := result.Principal.PermissionKeys(); !slices.Equal(got, []string{"business.order.se.read"}) {
		t.Fatalf("切换后权限集应来自目标组织角色: %v", got)
	}
	if len(repo.rotateAudits) != 1 {
		t.Fatalf("切换必须写审计: %#v", repo.rotateAudits)
	}
	audit := repo.rotateAudits[0]
	if audit.Action != "auth.organization.switch" || audit.OrganizationID == nil || *audit.OrganizationID != targetID {
		t.Fatalf("切换审计动作/目标组织异常: %#v", audit)
	}
	if audit.Details["source_organization.id"] != sourceID.String() || audit.Details["target_organization.id"] != targetID.String() {
		t.Fatalf("切换审计应包含来源与目标组织: %#v", audit.Details)
	}
	if len(result.OrganizationChoices) != 2 || result.OrganizationChoices[0].OrganizationID != sourceID {
		t.Fatalf("切换响应应携带候选组织列表: %#v", result.OrganizationChoices)
	}
}

func TestAuthUsecaseSwitchOrganizationRejectsOutsideCandidates(t *testing.T) {
	sourceID := uuid.New()
	userID := uuid.New()
	repo := &organizationSwitchRepoStub{
		choices: []OrganizationChoice{{OrganizationID: sourceID, OrganizationName: "北京公司", OrganizationCode: "BJ", IsDefault: true}},
	}
	current := &Principal{UserID: userID, Organization: Organization{ID: sourceID}, SessionTokenHash: "current-token-hash"}

	result, err := newOrganizationSwitchUsecase(repo).SwitchOrganization(context.Background(), current, uuid.New())
	if err != ErrAuthOrganizationForbidden {
		t.Fatalf("候选集外切换错误 = %v，期望 ErrAuthOrganizationForbidden", err)
	}
	if result != nil {
		t.Fatalf("拒绝切换不应返回结果: %#v", result)
	}
	if len(repo.rotatedSessions) != 0 || len(repo.rotateAudits) != 0 {
		t.Fatalf("拒绝切换不得轮转会话或写审计: %#v", repo.rotatedSessions)
	}
}

func TestAuthUsecaseLoginBootstrapAdminEntersNonMembershipOrganization(t *testing.T) {
	primaryID, branchID := uuid.New(), uuid.New()
	userID := uuid.New()
	passwordHash := organizationSwitchPasswordHash(t)
	repo := &organizationSwitchRepoStub{
		// bootstrap 管理员可能只在总部有成员资格，但候选集应放宽为全部启用组织。
		credential: &Credential{UserID: userID, PasswordHash: &passwordHash, Enabled: true, IsBootstrapAdmin: true, PrimaryOrganizationID: primaryID},
		choices:    []OrganizationChoice{{OrganizationID: primaryID, OrganizationName: "总部集团", OrganizationCode: "HQ", IsDefault: true}},
		allChoices: []OrganizationChoice{
			{OrganizationID: primaryID, OrganizationName: "总部集团", OrganizationCode: "HQ", IsDefault: true},
			{OrganizationID: branchID, OrganizationName: "北京财务", OrganizationCode: "BJFD"},
		},
		principalsByOrg: map[uuid.UUID]*Principal{branchID: {UserID: userID, IsBootstrapAdmin: true, Organization: Organization{ID: branchID, Name: "北京财务"}}},
	}

	result, err := newOrganizationSwitchUsecase(repo).Login(context.Background(), "admin", "correct-password", branchID, "test", "127.0.0.1")
	if err != nil {
		t.Fatalf("bootstrap 管理员显式登录非成员组织失败: %v", err)
	}
	if result.Principal.Organization.ID != branchID {
		t.Fatalf("登录主体组织 = %v，期望 %v", result.Principal.Organization.ID, branchID)
	}
	if len(repo.createdSessions) != 1 || repo.createdSessions[0].OrganizationID != branchID {
		t.Fatalf("会话应建立在所选组织上: %#v", repo.createdSessions)
	}
	if len(result.OrganizationChoices) != 2 || result.OrganizationChoices[0].OrganizationID != primaryID {
		t.Fatalf("bootstrap 管理员候选应来自全部启用组织: %#v", result.OrganizationChoices)
	}
}

func TestAuthUsecaseSwitchOrganizationBootstrapAdminEntersNonMembershipOrganization(t *testing.T) {
	sourceID, targetID := uuid.New(), uuid.New()
	userID := uuid.New()
	repo := &organizationSwitchRepoStub{
		// 普通用户候选集只含总部；bootstrap 管理员候选集含全部启用组织。
		choices: []OrganizationChoice{{OrganizationID: sourceID, OrganizationName: "总部集团", OrganizationCode: "HQ", IsDefault: true}},
		allChoices: []OrganizationChoice{
			{OrganizationID: sourceID, OrganizationName: "总部集团", OrganizationCode: "HQ", IsDefault: true},
			{OrganizationID: targetID, OrganizationName: "北京财务", OrganizationCode: "BJFD"},
		},
		principalsByOrg: map[uuid.UUID]*Principal{targetID: {UserID: userID, IsBootstrapAdmin: true, Organization: Organization{ID: targetID, Name: "北京财务"}}},
	}
	current := &Principal{UserID: userID, IsBootstrapAdmin: true, Organization: Organization{ID: sourceID}, SessionTokenHash: "current-token-hash"}

	result, err := newOrganizationSwitchUsecase(repo).SwitchOrganization(context.Background(), current, targetID)
	if err != nil {
		t.Fatalf("bootstrap 管理员切换进入非成员组织失败: %v", err)
	}
	if result.Principal.Organization.ID != targetID {
		t.Fatalf("切换后主体组织 = %v，期望 %v", result.Principal.Organization.ID, targetID)
	}
	if len(repo.rotatedSessions) != 1 || repo.rotatedSessions[0].OrganizationID != targetID {
		t.Fatalf("新会话应指向目标组织: %#v", repo.rotatedSessions)
	}
	if len(repo.rotateAudits) != 1 || repo.rotateAudits[0].Action != "auth.organization.switch" {
		t.Fatalf("切换必须写审计: %#v", repo.rotateAudits)
	}
}

func TestAuthUsecaseSwitchOrganizationBootstrapAdminStillRejectsDisabledOrganization(t *testing.T) {
	sourceID := uuid.New()
	repo := &organizationSwitchRepoStub{
		allChoices: []OrganizationChoice{{OrganizationID: sourceID, OrganizationName: "总部集团", OrganizationCode: "HQ", IsDefault: true}},
	}
	current := &Principal{UserID: uuid.New(), IsBootstrapAdmin: true, Organization: Organization{ID: sourceID}, SessionTokenHash: "current-token-hash"}

	// 候选集只含启用中组织，停用组织不可进入（全组织穿透不含停用组织）。
	if _, err := newOrganizationSwitchUsecase(repo).SwitchOrganization(context.Background(), current, uuid.New()); err != ErrAuthOrganizationForbidden {
		t.Fatalf("bootstrap 管理员切换候选集外组织错误 = %v，期望 ErrAuthOrganizationForbidden", err)
	}
}

func TestSortOrganizationChoicesDefaultFirstThenNameStable(t *testing.T) {
	defaultID, alphaID, betaID, gammaID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	choices := []OrganizationChoice{
		{OrganizationID: betaID, OrganizationName: "成都公司"},
		{OrganizationID: alphaID, OrganizationName: "北京公司"},
		{OrganizationID: defaultID, OrganizationName: "总部集团", IsDefault: true},
		{OrganizationID: gammaID, OrganizationName: "成都公司"},
	}
	if gammaID.String() < betaID.String() {
		choices[0], choices[3] = choices[3], choices[0]
	}

	sortOrganizationChoices(choices)
	want := []uuid.UUID{defaultID, alphaID, minUUID(betaID, gammaID), maxUUID(betaID, gammaID)}
	for index, organizationID := range want {
		if choices[index].OrganizationID != organizationID {
			t.Fatalf("候选排序第 %d 位 = %v，期望 %v（整体 %#v）", index, choices[index].OrganizationID, organizationID, choices)
		}
	}
	if !choices[0].IsDefault {
		t.Fatal("默认组织应置首并保留标记")
	}
}

func minUUID(left, right uuid.UUID) uuid.UUID {
	if left.String() < right.String() {
		return left
	}
	return right
}

func maxUUID(left, right uuid.UUID) uuid.UUID {
	if left.String() > right.String() {
		return left
	}
	return right
}
