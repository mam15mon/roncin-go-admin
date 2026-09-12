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
}

func (s *dingTalkLoginRegistrationRepoStub) FindActiveInvitationByMobile(_ context.Context, mobile string) (*DingTalkInvitation, error) {
	if s.invitation == nil || s.invitation.Mobile != mobile {
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

func newDingTalkLoginUsecase(repo AuthRepo, registrations DingTalkLoginRegistrationRepo, directory DingTalkDirectoryLookup, codec DingTalkRegistrationTokenCodec) *AuthUsecase {
	return NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, &dingTalkProviderStub{enabled: true, identity: &DingTalkIdentity{UnionID: "union-id", UserID: "user-id", CorpID: "ding-corp", Name: "张三"}}, codec, registrations, directory, nil)
}

func TestAuthUsecaseDingTalkInvitationAutoActivatesAndCreatesSession(t *testing.T) {
	targetOrganizationID := uuid.New()
	repo := &wecomAuthRepoStub{}
	invitations := &dingTalkLoginRegistrationRepoStub{
		invitation: &DingTalkInvitation{ID: uuid.New(), OrganizationID: targetOrganizationID, Mobile: "13800138000", Status: DingTalkInvitationStatusPending, ExpiresAt: time.Now().Add(time.Hour)},
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
	invitations := &dingTalkLoginRegistrationRepoStub{
		invitation: &DingTalkInvitation{ID: uuid.New(), OrganizationID: uuid.New(), Mobile: "13800138000", Status: DingTalkInvitationStatusPending, ExpiresAt: time.Now().Add(time.Hour)},
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

	registration, err := usecase.ConfirmDingTalkRegistration(context.Background(), "registration-token", targetOrganizationID)
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

	if _, err := usecase.ConfirmDingTalkRegistration(context.Background(), "registration-token", uuid.Nil); err != nil {
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

	if _, err := usecase.ConfirmDingTalkRegistration(context.Background(), "registration-token", uuid.New()); err != ErrDingTalkRegistrationOrgInvalid {
		t.Fatalf("无效目标组织错误 = %v，期望 ErrDingTalkRegistrationOrgInvalid", err)
	}
	if repo.created || repo.registeredApproverIDs != nil {
		t.Fatal("组织校验失败不应落库注册")
	}
}

// ===== 管理侧：邀请与审批用例 =====

type dingTalkRegistrationRepoStub struct {
	createdInvitation *DingTalkInvitation
	createErr         error
	revokedID         uuid.UUID
	revokedOrgIDs     []uuid.UUID
	registration      *DingTalkRegistration
	approved          *DingTalkRegistrationDecision
	rejected          *DingTalkRegistrationDecision
	actorRoles        []*AdminRoleProfile
	roleProfiles      []*AdminRoleProfile
	actorRolesErr     error
	roleProfilesErr   error
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

func (s *dingTalkRegistrationRepoStub) ListApproverRecipients(context.Context, uuid.UUID) ([]*DingTalkApproverRecipient, error) {
	return nil, nil
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

	created, err := usecase.CreateInvitation(context.Background(), dingTalkInvitationManagePrincipal(organizationID), "+86 138 0013-8000", " 备注姓名 ", organizationID, roleID, 0)
	if err != nil {
		t.Fatalf("CreateInvitation() error = %v", err)
	}
	if created.Mobile != "13800138000" || created.DisplayName != "备注姓名" {
		t.Fatalf("邀请应存归一化手机号与去空格备注: %#v", created)
	}
	if created.ExpiresAt.Before(time.Now().Add(DingTalkInvitationDefaultTTLHours*time.Hour - time.Minute)) {
		t.Fatalf("缺省有效期应为 %d 小时: %v", DingTalkInvitationDefaultTTLHours, created.ExpiresAt)
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
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := usecase.CreateInvitation(context.Background(), dingTalkInvitationManagePrincipal(organizationID), test.mobile, "", test.orgID, test.roleID, test.ttl); err != test.expectedErr {
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

var _ DingTalkLoginRegistrationRepo = (*dingTalkLoginRegistrationRepoStub)(nil)
var _ DingTalkDirectoryLookup = (*dingTalkDirectoryStub)(nil)
var _ DingTalkRegistrationRepo = (*dingTalkRegistrationRepoStub)(nil)
