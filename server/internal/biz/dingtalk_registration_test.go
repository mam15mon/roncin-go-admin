package biz

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
)

// ===== 手机号归一化与脱敏 =====

func TestNormalizeDingTalkMobileStripsPrefixesAndSeparators(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{raw: "13800138000", want: "13800138000"},
		{raw: " 138 0013-8000 ", want: "13800138000"},
		{raw: "+8613800138000", want: "13800138000"},
		{raw: "008613800138000", want: "13800138000"},
		{raw: "+86 138 0013 8000", want: "13800138000"},
		// 裸 86 开头的 11 位号码是合法手机号，不能被当国际前缀剥离。
		{raw: "8613813800", want: "8613813800"},
	}
	for _, test := range tests {
		if got := NormalizeDingTalkMobile(test.raw); got != test.want {
			t.Fatalf("NormalizeDingTalkMobile(%q) = %q, want %q", test.raw, got, test.want)
		}
	}
}

func TestMaskDingTalkMobileNeverLeaksFullNumber(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{raw: "13800138000", want: "138****8000"},
		{raw: "+8613800138000", want: "138****8000"},
		{raw: "12345", want: "12****"},
		{raw: "ab", want: "****"},
	}
	for _, test := range tests {
		if got := MaskDingTalkMobile(test.raw); got != test.want {
			t.Fatalf("MaskDingTalkMobile(%q) = %q, want %q", test.raw, got, test.want)
		}
		if got := MaskDingTalkMobile(test.raw); got == NormalizeDingTalkMobile(test.raw) && len(got) > 4 {
			t.Fatalf("MaskDingTalkMobile(%q) 泄漏完整号码: %q", test.raw, got)
		}
	}
}

func TestDingTalkInvitationEffectiveStatusTreatsExpiredPendingAsExpired(t *testing.T) {
	now := time.Now().UTC()
	invitation := &DingTalkInvitation{Status: DingTalkInvitationStatusPending, ExpiresAt: now.Add(-time.Minute)}
	if invitation.EffectiveStatus(now) != DingTalkInvitationStatusExpired {
		t.Fatalf("过期 PENDING 邀请应按 EXPIRED 展示，实际 %v", invitation.EffectiveStatus(now))
	}
	active := &DingTalkInvitation{Status: DingTalkInvitationStatusPending, ExpiresAt: now.Add(time.Minute)}
	if active.EffectiveStatus(now) != DingTalkInvitationStatusPending {
		t.Fatalf("未过期邀请应保持 PENDING，实际 %v", active.EffectiveStatus(now))
	}
}

// ===== 通道 A：扫码自动激活匹配链 =====

type dingTalkDirectoryStub struct {
	mobile string
	err    error
}

func (s *dingTalkDirectoryStub) LookupMobileByUserID(context.Context, string) (string, error) {
	return s.mobile, s.err
}

type dingTalkLoginRegistrationRepoStub struct {
	invitation         *DingTalkInvitation
	invitationErr      error
	consumeErr         error
	consumedInvitation uuid.UUID
	registrationOrgs   []OrganizationChoice
	organization       *Organization
	headquartersID     uuid.UUID
	recipients         []*DingTalkApproverRecipient
	requestedOrg       *uuid.UUID
	approverIDs        []uuid.UUID
	escalatedOrgID     uuid.UUID
	isEscalated        bool
}

func (s *dingTalkLoginRegistrationRepoStub) FindActiveInvitationByMobile(_ context.Context, mobile string) (*DingTalkInvitation, error) {
	if s.invitation == nil || s.invitation.Mobile == nil || *s.invitation.Mobile != mobile {
		return nil, ErrDingTalkInvitationNotFound
	}
	return s.invitation, s.invitationErr
}

func (s *dingTalkLoginRegistrationRepoStub) FindInvitationByToken(_ context.Context, token string) (*DingTalkInvitation, error) {
	if s.invitation == nil || s.invitation.Token != token {
		return nil, ErrDingTalkInvitationNotFound
	}
	return s.invitation, s.invitationErr
}

func (s *dingTalkLoginRegistrationRepoStub) ConsumeInvitationAndActivate(_ context.Context, _ *DingTalkIdentity, invitationID uuid.UUID) (*Credential, error) {
	if s.consumeErr != nil {
		return nil, s.consumeErr
	}
	s.consumedInvitation = invitationID
	return &Credential{UserID: uuid.New(), DisplayName: "激活员工", PrimaryOrganizationID: s.invitation.OrganizationID, Enabled: true}, nil
}

func (s *dingTalkLoginRegistrationRepoStub) ListRegistrationOrganizations(context.Context) ([]OrganizationChoice, error) {
	return s.registrationOrgs, nil
}

func (s *dingTalkLoginRegistrationRepoStub) FindRegistrationOrganization(_ context.Context, organizationID uuid.UUID) (*Organization, error) {
	if s.organization == nil || s.organization.ID != organizationID {
		return nil, ErrDingTalkRegistrationOrgInvalid
	}
	return s.organization, nil
}

func (s *dingTalkLoginRegistrationRepoStub) FindHeadquartersOrganizationID(context.Context) (uuid.UUID, error) {
	return s.headquartersID, nil
}

func (s *dingTalkLoginRegistrationRepoStub) ListApproverRecipients(_ context.Context, organizationID uuid.UUID) ([]*DingTalkApproverRecipient, error) {
	if s.headquartersID != uuid.Nil && organizationID != s.headquartersID && s.organization != nil && organizationID != s.organization.ID {
		return nil, nil
	}
	return s.recipients, nil
}

func (s *dingTalkLoginRegistrationRepoStub) ListApproverRecipientsWithEscalation(_ context.Context, organizationID uuid.UUID) ([]*DingTalkApproverRecipient, uuid.UUID, bool, error) {
	if s.isEscalated {
		return s.recipients, s.escalatedOrgID, true, nil
	}
	return s.recipients, organizationID, false, nil
}

func (s *dingTalkLoginRegistrationRepoStub) GetParentOrganizationID(_ context.Context, _ uuid.UUID) (*uuid.UUID, bool, error) {
	return nil, false, nil
}

func newDingTalkLoginUsecase(repo AuthRepo, registrations DingTalkLoginRegistrationRepo, directory DingTalkDirectoryLookup, codec DingTalkRegistrationTokenCodec) *AuthUsecase {
	return NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, &dingTalkProviderStub{enabled: true, identity: &DingTalkIdentity{UnionID: "union-id", UserID: "user-id", CorpID: "ding-corp", Name: "张三"}}, codec, registrations, directory, nil)
}

func TestAuthUsecaseDingTalkInvitationAutoActivatesAndCreatesSession(t *testing.T) {
	targetOrganizationID := uuid.New()
	repo := &wecomAuthRepoStub{}
	mobile := "13800138000"
	invitations := &dingTalkLoginRegistrationRepoStub{
		invitation: &DingTalkInvitation{ID: uuid.New(), Kind: DingTalkInvitationKindTargeted, OrganizationID: targetOrganizationID, Mobile: &mobile, Status: DingTalkInvitationStatusPending, ExpiresAt: time.Now().Add(time.Hour)},
	}
	directory := &dingTalkDirectoryStub{mobile: "+86 138 0013 8000"}
	codec := &dingTalkRegistrationTokenCodecStub{}
	usecase := newDingTalkLoginUsecase(repo, invitations, directory, codec)

	result, err := usecase.LoginDingTalk(context.Background(), "code", "state", "state", "test")
	if err != nil {
		t.Fatalf("LoginDingTalk() error = %v", err)
	}
	if result.Status != DingTalkLoginStatusAuthenticated || result.Principal == nil {
		t.Fatalf("命中邀请应直接建立会话，实际 %#v", result)
	}
	if repo.createdSession == nil || repo.createdSession.OrganizationID != targetOrganizationID {
		t.Fatalf("会话应落在邀请目标组织: %#v", repo.createdSession)
	}
	if invitations.consumedInvitation != invitations.invitation.ID {
		t.Fatalf("应消费命中的邀请，实际 %v", invitations.consumedInvitation)
	}
}

func TestAuthUsecaseDingTalkInvitationDegradesOnDirectoryFailure(t *testing.T) {
	repo := &wecomAuthRepoStub{}
	invitations := &dingTalkLoginRegistrationRepoStub{
		registrationOrgs: []OrganizationChoice{{OrganizationID: uuid.New(), OrganizationName: "成都公司"}},
	}
	directory := &dingTalkDirectoryStub{err: errors.New("钉钉通讯录不可用")}
	usecase := newDingTalkLoginUsecase(repo, invitations, directory, &dingTalkRegistrationTokenCodecStub{})

	result, err := usecase.LoginDingTalk(context.Background(), "code", "state", "state", "test")
	if err != nil {
		t.Fatalf("钉钉接口失败不得阻断扫码登录: %v", err)
	}
	if result.Status != DingTalkLoginStatusRegistrationRequired {
		t.Fatalf("钉钉接口失败应降级通道 B，实际 %#v", result)
	}
	if len(result.RegistrationOrganizations) != 1 || result.RegistrationOrganizations[0].OrganizationName != "成都公司" {
		t.Fatalf("降级响应应携带可选目标公司: %#v", result.RegistrationOrganizations)
	}
	if repo.createdSession != nil {
		t.Fatal("降级路径不应创建会话")
	}
}

func TestAuthUsecaseDingTalkInvitationDegradesOnConsumeConflict(t *testing.T) {
	repo := &wecomAuthRepoStub{}
	mobile := "13800138000"
	invitations := &dingTalkLoginRegistrationRepoStub{
		invitation: &DingTalkInvitation{ID: uuid.New(), Kind: DingTalkInvitationKindTargeted, OrganizationID: uuid.New(), Mobile: &mobile, Status: DingTalkInvitationStatusPending, ExpiresAt: time.Now().Add(time.Hour)},
		consumeErr: ErrDingTalkInvitationNotConsumable,
	}
	usecase := newDingTalkLoginUsecase(repo, invitations, &dingTalkDirectoryStub{mobile: "13800138000"}, &dingTalkRegistrationTokenCodecStub{})

	result, err := usecase.LoginDingTalk(context.Background(), "code", "state", "state", "test")
	if err != nil {
		t.Fatalf("邀请消费冲突不得阻断扫码登录: %v", err)
	}
	if result.Status != DingTalkLoginStatusRegistrationRequired {
		t.Fatalf("邀请消费冲突应降级通道 B，实际 %#v", result)
	}
}

func TestAuthUsecaseDingTalkInvitationInactiveWithoutDirectory(t *testing.T) {
	repo := &wecomAuthRepoStub{}
	usecase := newDingTalkLoginUsecase(repo, nil, nil, &dingTalkRegistrationTokenCodecStub{})

	result, err := usecase.LoginDingTalk(context.Background(), "code", "state", "state", "test")
	if err != nil {
		t.Fatalf("LoginDingTalk() error = %v", err)
	}
	if result.Status != DingTalkLoginStatusRegistrationRequired {
		t.Fatalf("无目录查询能力时保持通道 B，实际 %#v", result)
	}
}

// ===== 通道 B：注册确认与审批通知路由 =====

func TestAuthUsecaseConfirmDingTalkRegistrationRoutesRequestedOrganization(t *testing.T) {
	targetOrganizationID := uuid.New()
	approverID := uuid.New()
	repo := &wecomAuthRepoStub{
		credential: &Credential{UserID: uuid.New(), DisplayName: "张三", PrimaryOrganizationID: uuid.New(), Enabled: false},
		created:    true,
	}
	invitations := &dingTalkLoginRegistrationRepoStub{
		organization:   &Organization{ID: targetOrganizationID, Code: "CD", Name: "成都公司"},
		recipients:     []*DingTalkApproverRecipient{{UserID: approverID, DisplayName: "成都管理员"}},
		headquartersID: uuid.New(),
	}
	codec := &dingTalkRegistrationTokenCodecStub{identity: &DingTalkIdentity{UnionID: "union-id", UserID: "user-id", CorpID: "ding-corp", Name: "张三"}}
	usecase := newDingTalkLoginUsecase(repo, invitations, nil, codec)

	registration, err := usecase.ConfirmDingTalkRegistration(context.Background(), "registration-token", "", targetOrganizationID)
	if err != nil {
		t.Fatalf("ConfirmDingTalkRegistration() error = %v", err)
	}
	if registration.Status != "PENDING" || registration.DisplayName != "张三" {
		t.Fatalf("registration = %#v", registration)
	}
	if repo.registeredRequestedOrg == nil || *repo.registeredRequestedOrg != targetOrganizationID {
		t.Fatalf("应落库自选目标组织: %#v", repo.registeredRequestedOrg)
	}
	if len(repo.registeredApproverIDs) != 1 || repo.registeredApproverIDs[0] != approverID {
		t.Fatalf("审批通知应收件人 = %#v", repo.registeredApproverIDs)
	}
	if repo.registeredNoticeOrgName != "成都公司" {
		t.Fatalf("通知应展示自选目标组织名，实际 %q", repo.registeredNoticeOrgName)
	}
}

// TestAuthUsecaseConfirmDingTalkRegistrationAnnotatesEscalatedNotice 覆盖目标
// 组织无管理员向上追溯场景：通知收件人来自追溯结果，展示组织名保留目标组织
// 并追加「（上级代管）」后缀（与一键转派路径口径一致）。
func TestAuthUsecaseConfirmDingTalkRegistrationAnnotatesEscalatedNotice(t *testing.T) {
	targetOrganizationID := uuid.New()
	escalatedOrgID := uuid.New()
	hqApproverID := uuid.New()
	repo := &wecomAuthRepoStub{
		credential: &Credential{UserID: uuid.New(), DisplayName: "张三", PrimaryOrganizationID: uuid.New(), Enabled: false},
		created:    true,
	}
	invitations := &dingTalkLoginRegistrationRepoStub{
		organization:   &Organization{ID: targetOrganizationID, Code: "QD", Name: "青岛分公司"},
		recipients:     []*DingTalkApproverRecipient{{UserID: hqApproverID, DisplayName: "总部管理员"}},
		headquartersID: escalatedOrgID,
		escalatedOrgID: escalatedOrgID,
		isEscalated:    true,
	}
	codec := &dingTalkRegistrationTokenCodecStub{identity: &DingTalkIdentity{UnionID: "union-id", UserID: "user-id", CorpID: "ding-corp", Name: "张三"}}
	usecase := newDingTalkLoginUsecase(repo, invitations, nil, codec)

	if _, err := usecase.ConfirmDingTalkRegistration(context.Background(), "registration-token", "", targetOrganizationID); err != nil {
		t.Fatalf("ConfirmDingTalkRegistration() error = %v", err)
	}
	if len(repo.registeredApproverIDs) != 1 || repo.registeredApproverIDs[0] != hqApproverID {
		t.Fatalf("代管通知应收追溯后收件人 = %#v", repo.registeredApproverIDs)
	}
	if repo.registeredNoticeOrgName != "青岛分公司"+DingTalkEscalatedOrgSuffix {
		t.Fatalf("代管通知应展示目标组织名并追加后缀，实际 %q", repo.registeredNoticeOrgName)
	}
}

func TestAuthUsecaseConfirmDingTalkRegistrationFallsBackToHeadquarters(t *testing.T) {
	headquartersID := uuid.New()
	hqApprover := uuid.New()
	repo := &wecomAuthRepoStub{
		credential: &Credential{UserID: uuid.New(), DisplayName: "张三", PrimaryOrganizationID: headquartersID, Enabled: false},
		created:    true,
	}
	invitations := &dingTalkLoginRegistrationRepoStub{
		headquartersID: headquartersID,
		recipients:     []*DingTalkApproverRecipient{{UserID: hqApprover, DisplayName: "总部管理员"}},
	}
	codec := &dingTalkRegistrationTokenCodecStub{identity: &DingTalkIdentity{UnionID: "union-id", UserID: "user-id", CorpID: "ding-corp", Name: "张三"}}
	usecase := newDingTalkLoginUsecase(repo, invitations, nil, codec)

	if _, err := usecase.ConfirmDingTalkRegistration(context.Background(), "registration-token", "", uuid.Nil); err != nil {
		t.Fatalf("ConfirmDingTalkRegistration() error = %v", err)
	}
	if repo.registeredRequestedOrg != nil {
		t.Fatalf("未自选组织不应落 requested org: %#v", repo.registeredRequestedOrg)
	}
	if len(repo.registeredApproverIDs) != 1 || repo.registeredApproverIDs[0] != hqApprover {
		t.Fatalf("未自选组织应通知总部管理员: %#v", repo.registeredApproverIDs)
	}
}

func TestAuthUsecaseConfirmDingTalkRegistrationRejectsInvalidOrganization(t *testing.T) {
	repo := &wecomAuthRepoStub{}
	invitations := &dingTalkLoginRegistrationRepoStub{}
	codec := &dingTalkRegistrationTokenCodecStub{identity: &DingTalkIdentity{UnionID: "union-id", UserID: "user-id", CorpID: "ding-corp", Name: "张三"}}
	usecase := newDingTalkLoginUsecase(repo, invitations, nil, codec)

	if _, err := usecase.ConfirmDingTalkRegistration(context.Background(), "registration-token", "", uuid.New()); err != ErrDingTalkRegistrationOrgInvalid {
		t.Fatalf("无效目标组织错误 = %v，期望 ErrDingTalkRegistrationOrgInvalid", err)
	}
	if repo.created || repo.registeredApproverIDs != nil {
		t.Fatal("组织校验失败不应落库注册")
	}
}

func TestAuthUsecaseConfirmDingTalkRegistrationLocksOrganizationFromToken(t *testing.T) {
	invitationOrgID := uuid.New()
	bogusClientOrgID := uuid.New()
	approverID := uuid.New()
	repo := &wecomAuthRepoStub{
		credential: &Credential{UserID: uuid.New(), DisplayName: "受邀员工", PrimaryOrganizationID: uuid.New(), Enabled: false},
		created:    true,
	}
	invitations := &dingTalkLoginRegistrationRepoStub{
		invitation: &DingTalkInvitation{
			ID:               uuid.New(),
			Token:            "valid-token-12345",
			Kind:             DingTalkInvitationKindGeneric,
			OrganizationID:   invitationOrgID,
			OrganizationName: "深圳分公司",
			Status:           DingTalkInvitationStatusPending,
			ExpiresAt:        time.Now().Add(time.Hour),
		},
		recipients: []*DingTalkApproverRecipient{{UserID: approverID, DisplayName: "分公司管理员"}},
	}
	codec := &dingTalkRegistrationTokenCodecStub{identity: &DingTalkIdentity{UnionID: "union-id", UserID: "user-id", CorpID: "ding-corp", Name: "受邀员工"}}
	usecase := newDingTalkLoginUsecase(repo, invitations, nil, codec)

	// 传入伪造的目标组织 bogusClientOrgID，断言服务端根据 Token 强绑定 invitationOrgID
	registration, err := usecase.ConfirmDingTalkRegistration(context.Background(), "registration-token", "valid-token-12345", bogusClientOrgID)
	if err != nil {
		t.Fatalf("ConfirmDingTalkRegistration() error = %v", err)
	}
	if registration.Status != "PENDING" {
		t.Fatalf("registration status = %v", registration.Status)
	}
	if repo.registeredRequestedOrg == nil || *repo.registeredRequestedOrg != invitationOrgID {
		t.Fatalf("应根据 Token 锁定组织 %v，实际落库 %v", invitationOrgID, repo.registeredRequestedOrg)
	}
	if repo.registeredNoticeOrgName != "深圳分公司" {
		t.Fatalf("通知应展示 Token 锁定组织名，实际 %q", repo.registeredNoticeOrgName)
	}
}

func TestAuthUsecaseGetDingTalkInvitationInfoAndRateLimiting(t *testing.T) {
	orgID := uuid.New()
	now := time.Now().UTC()
	invitations := &dingTalkLoginRegistrationRepoStub{
		invitation: &DingTalkInvitation{
			ID:               uuid.New(),
			Token:            "test-public-token",
			Kind:             DingTalkInvitationKindGeneric,
			OrganizationID:   orgID,
			OrganizationName: "上海分公司",
			InviterName:      "李经理",
			Status:           DingTalkInvitationStatusPending,
			ExpiresAt:        now.Add(24 * time.Hour),
		},
	}
	rateRepo := &loginRateLimitRepoStub{counts: make(map[string]int)}
	usecase := NewAuthUsecase(rateRepo, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, &dingTalkProviderStub{enabled: true}, nil, invitations, nil, nil)

	// 1. 正常查询：返回最小化公开信息（组织名、邀请人、有效期）
	info, err := usecase.GetDingTalkInvitationInfo(context.Background(), "test-public-token", "1.2.3.4")
	if err != nil {
		t.Fatalf("GetDingTalkInvitationInfo() error = %v", err)
	}
	if info.OrganizationName != "上海分公司" || info.InviterDisplayName != "李经理" {
		t.Fatalf("返回公开信息不符合预期: %#v", info)
	}

	// 2. 错误 Token：触发失败计数
	for i := 0; i < 5; i++ {
		_, err := usecase.GetDingTalkInvitationInfo(context.Background(), "non-existent-token", "1.2.3.4")
		if err != ErrDingTalkInvitationNotFound {
			t.Fatalf("查询不存在邀请应返回 ErrDingTalkInvitationNotFound，实际 %v", err)
		}
	}

	// 3. 第 6 次触发 IP 登录同款限流
	_, err = usecase.GetDingTalkInvitationInfo(context.Background(), "test-public-token", "1.2.3.4")
	if err != ErrLoginRateLimited {
		t.Fatalf("超限后应返回 ErrLoginRateLimited，实际 %v", err)
	}
}

// ===== 管理侧：邀请与审批用例 =====

type dingTalkRegistrationRepoStub struct {
	createdInvitation   *DingTalkInvitation
	createErr           error
	revokedID           uuid.UUID
	revokedOrgIDs       []uuid.UUID
	registration        *DingTalkRegistration
	approved            *DingTalkRegistrationDecision
	rejected            *DingTalkRegistrationDecision
	transferDecision    *DingTalkRegistrationDecision
	transferTargetOrg   uuid.UUID
	transferErr         error
	escalatedRecipients []*DingTalkApproverRecipient
	escalatedOrgID      uuid.UUID
	isEscalated         bool
	parentOrgs          map[uuid.UUID]*uuid.UUID
	actorRoles          []*AdminRoleProfile
	roleProfiles        []*AdminRoleProfile
	actorRolesErr       error
	roleProfilesErr     error
}

func (s *dingTalkRegistrationRepoStub) GetActorRolesPrivilegeProfiles(context.Context, uuid.UUID, uuid.UUID) ([]*AdminRoleProfile, error) {
	return s.actorRoles, s.actorRolesErr
}

func (s *dingTalkRegistrationRepoStub) GetRolesPrivilegeProfiles(_ context.Context, _ uuid.UUID, roleIDs []uuid.UUID) ([]*AdminRoleProfile, error) {
	return s.roleProfiles, s.roleProfilesErr
}

func (s *dingTalkRegistrationRepoStub) CreateInvitation(_ context.Context, input *DingTalkInvitation, _ *AuditEvent) (*DingTalkInvitation, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	s.createdInvitation = input
	return input, nil
}

func (s *dingTalkRegistrationRepoStub) ListInvitations(context.Context, []uuid.UUID, DingTalkInvitationListOptions) (*DingTalkInvitationList, error) {
	return &DingTalkInvitationList{}, nil
}

func (s *dingTalkRegistrationRepoStub) RevokeInvitation(_ context.Context, _ uuid.UUID, id uuid.UUID, organizationIDs []uuid.UUID, _ *AuditEvent) (*DingTalkInvitation, error) {
	s.revokedID = id
	s.revokedOrgIDs = organizationIDs
	return nil, nil
}

func (s *dingTalkRegistrationRepoStub) FindInvitationByToken(_ context.Context, token string) (*DingTalkInvitation, error) {
	if s.createdInvitation != nil && s.createdInvitation.Token == token {
		return s.createdInvitation, nil
	}
	return nil, ErrDingTalkInvitationNotFound
}

func (s *dingTalkRegistrationRepoStub) GetPendingRegistration(_ context.Context, userID uuid.UUID) (*DingTalkRegistration, error) {
	if s.registration == nil || s.registration.UserID != userID {
		return nil, ErrDingTalkRegistrationNotFound
	}
	return s.registration, nil
}

func (s *dingTalkRegistrationRepoStub) ListPendingRegistrations(context.Context, []uuid.UUID, DingTalkRegistrationListOptions) (*DingTalkRegistrationList, error) {
	return &DingTalkRegistrationList{}, nil
}

func (s *dingTalkRegistrationRepoStub) ApproveRegistration(_ context.Context, decision *DingTalkRegistrationDecision) (*DingTalkRegistration, error) {
	s.approved = decision
	return s.registration, nil
}

func (s *dingTalkRegistrationRepoStub) RejectRegistration(_ context.Context, decision *DingTalkRegistrationDecision) error {
	s.rejected = decision
	return nil
}

func (s *dingTalkRegistrationRepoStub) TransferRegistration(_ context.Context, decision *DingTalkRegistrationDecision, targetOrgID uuid.UUID) error {
	s.transferDecision = decision
	s.transferTargetOrg = targetOrgID
	return s.transferErr
}

func (s *dingTalkRegistrationRepoStub) ListApproverRecipients(context.Context, uuid.UUID) ([]*DingTalkApproverRecipient, error) {
	return nil, nil
}

func (s *dingTalkRegistrationRepoStub) ListApproverRecipientsWithEscalation(_ context.Context, orgID uuid.UUID) ([]*DingTalkApproverRecipient, uuid.UUID, bool, error) {
	if s.escalatedRecipients != nil {
		return s.escalatedRecipients, s.escalatedOrgID, s.isEscalated, nil
	}
	return nil, orgID, false, nil
}

func (s *dingTalkRegistrationRepoStub) GetParentOrganizationID(_ context.Context, orgID uuid.UUID) (*uuid.UUID, bool, error) {
	if s.parentOrgs != nil {
		if parentID, ok := s.parentOrgs[orgID]; ok {
			return parentID, true, nil
		}
	}
	return nil, false, nil
}

func dingTalkInvitationManagePrincipal(organizationID uuid.UUID) *Principal {
	return &Principal{
		UserID:            uuid.New(),
		Organization:      Organization{ID: organizationID},
		OrganizationNodes: scopeNodes(organizationID),
		RoleGrants:        []RoleGrant{roleGrant("inviter", DataScopeOrganization, []string{access.UserDingTalkInvitationManage}, nil)},
	}
}

func TestDingTalkRegistrationUsecaseCreateInvitationValidatesAndNormalizes(t *testing.T) {
	organizationID := uuid.New()
	roleID := uuid.New()
	repo := &dingTalkRegistrationRepoStub{
		actorRoles:   []*AdminRoleProfile{{Code: "inviter", DataScope: DataScopeOrganization, PermissionKeys: []string{access.UserDingTalkInvitationManage}}},
		roleProfiles: []*AdminRoleProfile{{Code: "staff", DataScope: DataScopeSelf}},
	}
	usecase := NewDingTalkRegistrationUsecase(repo)

	created, err := usecase.CreateInvitation(context.Background(), dingTalkInvitationManagePrincipal(organizationID), DingTalkInvitationKindTargeted, "+86 138 0013-8000", " 备注姓名 ", organizationID, &roleID, 0)
	if err != nil {
		t.Fatalf("CreateInvitation() error = %v", err)
	}
	if created.Mobile == nil || *created.Mobile != "13800138000" || created.DisplayName != "备注姓名" {
		t.Fatalf("邀请应存归一化手机号与去空格备注: %#v", created)
	}
	if created.Token == "" || len(created.Token) != 32 {
		t.Fatalf("应生成 128-bit (32 字符 Hex) Token: %q", created.Token)
	}
	if created.Kind != DingTalkInvitationKindTargeted {
		t.Fatalf("邀请类型应为 TARGETED: %v", created.Kind)
	}
	if created.ExpiresAt.Before(time.Now().Add(DingTalkInvitationDefaultTTLHours*time.Hour - time.Minute)) {
		t.Fatalf("缺省有效期应为 %d 小时: %v", DingTalkInvitationDefaultTTLHours, created.ExpiresAt)
	}
}

func TestDingTalkRegistrationUsecaseCreateGenericInvitation(t *testing.T) {
	organizationID := uuid.New()
	repo := &dingTalkRegistrationRepoStub{}
	usecase := NewDingTalkRegistrationUsecase(repo)

	created, err := usecase.CreateInvitation(context.Background(), dingTalkInvitationManagePrincipal(organizationID), DingTalkInvitationKindGeneric, "", "通用入职码", organizationID, nil, 24)
	if err != nil {
		t.Fatalf("CreateInvitation() error = %v", err)
	}
	if created.Mobile != nil {
		t.Fatalf("通用码手机号必须为 nil: %v", created.Mobile)
	}
	if created.Kind != DingTalkInvitationKindGeneric {
		t.Fatalf("类型应为 GENERIC: %v", created.Kind)
	}
	if created.RoleID != nil {
		t.Fatalf("未传角色应为 nil: %v", created.RoleID)
	}
	if created.Token == "" || len(created.Token) != 32 {
		t.Fatalf("应生成 128-bit Token: %q", created.Token)
	}
}

func TestDingTalkRegistrationUsecaseCreateInvitationRejectsBadInput(t *testing.T) {
	organizationID := uuid.New()
	roleID := uuid.New()
	usecase := NewDingTalkRegistrationUsecase(&dingTalkRegistrationRepoStub{})

	tests := []struct {
		name        string
		mobile      string
		orgID       uuid.UUID
		roleID      uuid.UUID
		ttl         int
		expectedErr error
	}{
		{name: "手机号非法", mobile: "12345", orgID: organizationID, roleID: roleID, ttl: 0, expectedErr: ErrAdminInvalidArgument},
		{name: "有效期超上限", mobile: "13800138000", orgID: organizationID, roleID: roleID, ttl: DingTalkInvitationMaxTTLHours + 1, expectedErr: ErrAdminInvalidArgument},
		{name: "有效期低于下限", mobile: "13800138000", orgID: organizationID, roleID: roleID, ttl: 0 - 1, expectedErr: ErrAdminInvalidArgument},
		{name: "越权目标组织", mobile: "13800138000", orgID: uuid.New(), roleID: roleID, ttl: 0, expectedErr: ErrPermissionDenied},
		{name: "定向邀请缺初始角色", mobile: "13800138000", orgID: organizationID, roleID: uuid.Nil, ttl: 0, expectedErr: ErrAdminInvalidArgument},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testRoleID := test.roleID
			var roleIDPtr *uuid.UUID
			if testRoleID != uuid.Nil {
				roleIDPtr = &testRoleID
			}
			if _, err := usecase.CreateInvitation(context.Background(), dingTalkInvitationManagePrincipal(organizationID), DingTalkInvitationKindTargeted, test.mobile, "", test.orgID, roleIDPtr, test.ttl); err != test.expectedErr {
				t.Fatalf("CreateInvitation() error = %v, want %v", err, test.expectedErr)
			}
		})
	}
}

func TestDingTalkRegistrationUsecaseApproveEnforcesRoutingScope(t *testing.T) {
	organizationID := uuid.New()
	targetOrganizationID := uuid.New()
	userID := uuid.New()
	repo := &dingTalkRegistrationRepoStub{
		registration: &DingTalkRegistration{UserID: userID, IntakeOrganizationID: organizationID, RequestedOrganizationID: &targetOrganizationID},
	}
	usecase := NewDingTalkRegistrationUsecase(repo)

	if _, err := usecase.ApproveRegistration(context.Background(), dingTalkInvitationManagePrincipal(organizationID), userID, []uuid.UUID{uuid.New()}); err != ErrPermissionDenied {
		t.Fatalf("审批目标组织越权错误 = %v，期望 ErrPermissionDenied", err)
	}
	if repo.approved != nil {
		t.Fatal("越权审批不应触达仓储")
	}
}

func TestDingTalkRegistrationUsecaseApproveBuildsDecision(t *testing.T) {
	organizationID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	repo := &dingTalkRegistrationRepoStub{
		registration: &DingTalkRegistration{UserID: userID, IntakeOrganizationID: organizationID},
		actorRoles:   []*AdminRoleProfile{{Code: "inviter", DataScope: DataScopeOrganization, PermissionKeys: []string{access.UserDingTalkInvitationManage}}},
		roleProfiles: []*AdminRoleProfile{{Code: "staff", DataScope: DataScopeSelf}},
	}
	usecase := NewDingTalkRegistrationUsecase(repo)

	if _, err := usecase.ApproveRegistration(context.Background(), dingTalkInvitationManagePrincipal(organizationID), userID, []uuid.UUID{roleID}); err != nil {
		t.Fatalf("ApproveRegistration() error = %v", err)
	}
	if repo.approved == nil || repo.approved.UserID != userID {
		t.Fatalf("审批决策 = %#v", repo.approved)
	}
	if repo.approved.Notification == nil || repo.approved.Notification.RecipientUserID != userID || repo.approved.Notification.Template != NotificationTemplateUserAuthorized {
		t.Fatalf("审批应携带本人授权完成通知: %#v", repo.approved.Notification)
	}
	if repo.approved.Audit == nil || repo.approved.Audit.Action != "admin.dingtalk.registration.approve" {
		t.Fatalf("审批应写审计: %#v", repo.approved.Audit)
	}
}

func TestDingTalkRegistrationUsecaseRejectRequiresReason(t *testing.T) {
	organizationID := uuid.New()
	userID := uuid.New()
	usecase := NewDingTalkRegistrationUsecase(&dingTalkRegistrationRepoStub{registration: &DingTalkRegistration{UserID: userID, IntakeOrganizationID: organizationID}})

	if err := usecase.RejectRegistration(context.Background(), dingTalkInvitationManagePrincipal(organizationID), userID, "   "); err != ErrDingTalkRegistrationReasonMissing {
		t.Fatalf("空拒绝原因错误 = %v，期望 ErrDingTalkRegistrationReasonMissing", err)
	}
}

func TestDingTalkRegistrationUsecaseRejectBuildsNotification(t *testing.T) {
	organizationID := uuid.New()
	userID := uuid.New()
	repo := &dingTalkRegistrationRepoStub{registration: &DingTalkRegistration{UserID: userID, IntakeOrganizationID: organizationID}}
	usecase := NewDingTalkRegistrationUsecase(repo)

	if err := usecase.RejectRegistration(context.Background(), dingTalkInvitationManagePrincipal(organizationID), userID, "身份信息不完整"); err != nil {
		t.Fatalf("RejectRegistration() error = %v", err)
	}
	if repo.rejected == nil || repo.rejected.Reason != "身份信息不完整" {
		t.Fatalf("拒绝决策 = %#v", repo.rejected)
	}
	if repo.rejected.Notification == nil || repo.rejected.Notification.RecipientUserID != userID || repo.rejected.Notification.Template != NotificationTemplateDingTalkRegistrationRejected {
		t.Fatalf("拒绝应携带本人通知: %#v", repo.rejected.Notification)
	}
}

func TestDingTalkRegistrationUsecaseTransferRegistration(t *testing.T) {
	orgID := uuid.New()
	targetOrgID := uuid.New()
	userID := uuid.New()
	repo := &dingTalkRegistrationRepoStub{
		registration: &DingTalkRegistration{UserID: userID, IntakeOrganizationID: orgID},
	}
	usecase := NewDingTalkRegistrationUsecase(repo)

	// 1. 同组织转派拦截
	if err := usecase.TransferRegistration(context.Background(), dingTalkInvitationManagePrincipal(orgID), userID, orgID, "转派理由"); err != ErrDingTalkRegistrationTransferSame {
		t.Fatalf("同组织转派错误 = %v，期望 ErrDingTalkRegistrationTransferSame", err)
	}

	// 2. 理由为空拦截
	if err := usecase.TransferRegistration(context.Background(), dingTalkInvitationManagePrincipal(orgID), userID, targetOrgID, "  "); err != ErrDingTalkRegistrationReasonMissing {
		t.Fatalf("空理由错误 = %v，期望 ErrDingTalkRegistrationReasonMissing", err)
	}

	// 3. 越权转派拦截
	otherOrgID := uuid.New()
	if err := usecase.TransferRegistration(context.Background(), dingTalkInvitationManagePrincipal(otherOrgID), userID, targetOrgID, "转派理由"); err != ErrPermissionDenied {
		t.Fatalf("越权转派错误 = %v，期望 ErrPermissionDenied", err)
	}

	// 4. 正常转派成功
	if err := usecase.TransferRegistration(context.Background(), dingTalkInvitationManagePrincipal(orgID), userID, targetOrgID, "员工实际归属上海分公司"); err != nil {
		t.Fatalf("正常转派失败: %v", err)
	}
	if repo.transferTargetOrg != targetOrgID {
		t.Fatalf("转派目标组织 = %v, 期望 %v", repo.transferTargetOrg, targetOrgID)
	}
	if repo.transferDecision == nil || repo.transferDecision.Reason != "员工实际归属上海分公司" {
		t.Fatalf("转派决策 = %#v", repo.transferDecision)
	}
}

var _ DingTalkLoginRegistrationRepo = (*dingTalkLoginRegistrationRepoStub)(nil)
var _ DingTalkDirectoryLookup = (*dingTalkDirectoryStub)(nil)
var _ DingTalkRegistrationRepo = (*dingTalkRegistrationRepoStub)(nil)
