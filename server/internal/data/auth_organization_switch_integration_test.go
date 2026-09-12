package data

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
)

type authOrganizationSwitchFixture struct {
	t             *testing.T
	data          *Data
	repo          *authRepo
	userID        uuid.UUID
	headquarters  uuid.UUID
	branch        uuid.UUID
	extra         uuid.UUID // 启用组织但成员资格停用
	disabledOrg   uuid.UUID // 组织停用但成员资格启用
	suffix        string
	sessionExpiry time.Time
}

func newAuthOrganizationSwitchFixture(t *testing.T) *authOrganizationSwitchFixture {
	t.Helper()
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	headquarters, err := data.db.Organization.Create().
		SetCode("HQ-" + suffix).
		SetName("总部集团-" + suffix).
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建总部组织失败: %v", err)
	}
	branch, err := data.db.Organization.Create().
		SetCode("CD-" + suffix).
		SetName("成都公司-" + suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建分公司组织失败: %v", err)
	}
	extra, err := data.db.Organization.Create().
		SetCode("SH-" + suffix).
		SetName("上海公司-" + suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建额外组织失败: %v", err)
	}
	disabledOrg, err := data.db.Organization.Create().
		SetCode("BJ-" + suffix).
		SetName("北京公司-" + suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		SetEnabled(false).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建停用组织失败: %v", err)
	}
	account, err := data.db.User.Create().
		SetUsername("switch-" + suffix).
		SetDisplayName("切换测试用户-" + suffix).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	memberships := []struct {
		organizationID uuid.UUID
		primary        bool
		enabled        bool
	}{
		{organizationID: headquarters.ID, primary: true, enabled: true},
		{organizationID: branch.ID, primary: false, enabled: true},
		{organizationID: extra.ID, primary: false, enabled: false},
		{organizationID: disabledOrg.ID, primary: false, enabled: true},
	}
	for _, membership := range memberships {
		if _, err := data.db.Membership.Create().
			SetUserID(account.ID).
			SetOrganizationID(membership.organizationID).
			SetPrimary(membership.primary).
			SetEnabled(membership.enabled).
			Save(ctx); err != nil {
			t.Fatalf("创建成员资格失败: %v", err)
		}
	}
	return &authOrganizationSwitchFixture{
		t:             t,
		data:          data,
		repo:          NewAuthRepo(data).(*authRepo),
		userID:        account.ID,
		headquarters:  headquarters.ID,
		branch:        branch.ID,
		extra:         extra.ID,
		disabledOrg:   disabledOrg.ID,
		suffix:        suffix,
		sessionExpiry: time.Now().Add(time.Hour),
	}
}

func (f *authOrganizationSwitchFixture) createSession(ctx context.Context, tokenHash string, organizationID uuid.UUID) {
	f.t.Helper()
	if err := f.repo.CreateSession(ctx, &biz.Session{
		TokenHash:      tokenHash,
		UserID:         f.userID,
		OrganizationID: organizationID,
		ExpiresAt:      f.sessionExpiry,
		UserAgent:      "integration-test",
	}, "", &biz.AuditEvent{OrganizationID: &organizationID, UserID: &f.userID, Action: "auth.login", Result: "success"}); err != nil {
		f.t.Fatalf("创建会话失败: %v", err)
	}
}

func TestAuthRepoListEnabledMembershipOrganizationsFiltersDisabled(t *testing.T) {
	fixture := newAuthOrganizationSwitchFixture(t)

	choices, err := fixture.repo.ListEnabledMembershipOrganizations(context.Background(), fixture.userID)
	if err != nil {
		t.Fatalf("查询成员资格候选组织失败: %v", err)
	}
	if len(choices) != 2 {
		t.Fatalf("候选组织 = %#v，期望仅总部与分公司", choices)
	}
	byID := make(map[uuid.UUID]biz.OrganizationChoice, len(choices))
	for _, choice := range choices {
		byID[choice.OrganizationID] = choice
	}
	headquartersChoice, ok := byID[fixture.headquarters]
	if !ok || !headquartersChoice.IsDefault || headquartersChoice.OrganizationCode != "HQ-"+fixture.suffix || headquartersChoice.OrganizationName != "总部集团-"+fixture.suffix {
		t.Fatalf("默认候选组织异常: %#v", headquartersChoice)
	}
	branchChoice, ok := byID[fixture.branch]
	if !ok || branchChoice.IsDefault {
		t.Fatalf("非默认候选组织异常: %#v", branchChoice)
	}
	if _, filtered := byID[fixture.extra]; filtered {
		t.Fatal("成员资格停用的组织不应进入候选列表")
	}
	if _, filtered := byID[fixture.disabledOrg]; filtered {
		t.Fatal("组织停用的成员资格不应进入候选列表")
	}
}

func TestAuthRepoRotateSessionSwitchesTokenAndKeepsOtherSessions(t *testing.T) {
	fixture := newAuthOrganizationSwitchFixture(t)
	ctx := context.Background()
	currentHash, nextHash, otherDeviceHash := "rotate-current-"+fixture.suffix, "rotate-next-"+fixture.suffix, "rotate-other-"+fixture.suffix
	fixture.createSession(ctx, currentHash, fixture.headquarters)
	fixture.createSession(ctx, otherDeviceHash, fixture.headquarters)
	now := time.Now().UTC()

	err := fixture.repo.RotateSession(ctx, currentHash, &biz.Session{
		TokenHash:      nextHash,
		UserID:         fixture.userID,
		OrganizationID: fixture.branch,
		ExpiresAt:      fixture.sessionExpiry,
		UserAgent:      "ignored",
	}, now, &biz.AuditEvent{OrganizationID: &fixture.branch, UserID: &fixture.userID, Action: "auth.organization.switch", Result: "success", Details: map[string]string{"source_organization.id": fixture.headquarters.String(), "target_organization.id": fixture.branch.String()}})
	if err != nil {
		t.Fatalf("轮转会话失败: %v", err)
	}

	switched, err := fixture.repo.FindSession(ctx, nextHash, time.Now().UTC())
	if err != nil {
		t.Fatalf("新令牌应立即可用: %v", err)
	}
	if switched.OrganizationID != fixture.branch {
		t.Fatalf("新会话组织 = %v，期望分公司", switched.OrganizationID)
	}
	if _, err := fixture.repo.FindSession(ctx, currentHash, time.Now().UTC()); err != biz.ErrSessionExpired {
		t.Fatalf("旧令牌应失效，错误 = %v", err)
	}
	otherDevice, err := fixture.repo.FindSession(ctx, otherDeviceHash, time.Now().UTC())
	if err != nil || otherDevice.OrganizationID != fixture.headquarters {
		t.Fatalf("其他设备会话不应受轮转影响: session=%#v error=%v", otherDevice, err)
	}
}

func TestAuthRepoRotateSessionRejectsInvalidTargets(t *testing.T) {
	fixture := newAuthOrganizationSwitchFixture(t)
	ctx := context.Background()
	currentHash := "rotate-reject-" + fixture.suffix
	fixture.createSession(ctx, currentHash, fixture.headquarters)
	now := time.Now().UTC()

	tests := []struct {
		name           string
		organizationID uuid.UUID
	}{
		{name: "非本人成员资格", organizationID: uuid.New()},
		{name: "成员资格已停用", organizationID: fixture.extra},
		{name: "组织已停用", organizationID: fixture.disabledOrg},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nextHash := "rotate-reject-next-" + uuid.NewString()
			err := fixture.repo.RotateSession(ctx, currentHash, &biz.Session{
				TokenHash:      nextHash,
				UserID:         fixture.userID,
				OrganizationID: test.organizationID,
				ExpiresAt:      fixture.sessionExpiry,
			}, now, &biz.AuditEvent{OrganizationID: &test.organizationID, UserID: &fixture.userID, Action: "auth.organization.switch", Result: "success"})
			if err != biz.ErrAuthOrganizationForbidden {
				t.Fatalf("轮转错误 = %v，期望 ErrAuthOrganizationForbidden", err)
			}
			if _, findErr := fixture.repo.FindSession(ctx, nextHash, time.Now().UTC()); findErr != biz.ErrSessionExpired {
				t.Fatalf("被拒绝的轮转不应留下新会话: %v", findErr)
			}
		})
	}
	if _, err := fixture.repo.FindSession(ctx, currentHash, time.Now().UTC()); err != nil {
		t.Fatalf("被拒绝的轮转不应影响当前会话: %v", err)
	}
}

func TestAuthRepoRotateSessionWritesAuditWithSourceAndTarget(t *testing.T) {
	fixture := newAuthOrganizationSwitchFixture(t)
	ctx := context.Background()
	currentHash := "rotate-audit-" + fixture.suffix
	fixture.createSession(ctx, currentHash, fixture.headquarters)

	err := fixture.repo.RotateSession(ctx, currentHash, &biz.Session{
		TokenHash:      "rotate-audit-next-" + fixture.suffix,
		UserID:         fixture.userID,
		OrganizationID: fixture.branch,
		ExpiresAt:      fixture.sessionExpiry,
	}, time.Now().UTC(), &biz.AuditEvent{OrganizationID: &fixture.branch, UserID: &fixture.userID, Action: "auth.organization.switch", Result: "success", Details: map[string]string{"source_organization.id": fixture.headquarters.String(), "target_organization.id": fixture.branch.String()}})
	if err != nil {
		t.Fatalf("轮转会话失败: %v", err)
	}

	audit, err := fixture.data.db.AuditLog.Query().
		Where(auditlogent.ActionEQ("auth.organization.switch"), auditlogent.UserIDEQ(fixture.userID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("读取切换审计失败: %v", err)
	}
	if audit.OrganizationID == nil || *audit.OrganizationID != fixture.branch {
		t.Fatalf("切换审计组织应为目标组织: %#v", audit)
	}
	var details map[string]string
	if err := json.Unmarshal(audit.Details, &details); err != nil {
		t.Fatalf("解析切换审计详情失败: %v", err)
	}
	if details["source_organization.id"] != fixture.headquarters.String() || details["target_organization.id"] != fixture.branch.String() {
		t.Fatalf("切换审计应包含来源与目标组织: %#v", details)
	}
}
