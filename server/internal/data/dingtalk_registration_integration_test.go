package data

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	invitationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/dingtalkinvitation"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	notificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/notificationdelivery"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	roleassignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

type dingTalkRegistrationFixture struct {
	t               *testing.T
	data            *Data
	repo            *dingTalkRegistrationRepo
	authRepo        *authRepo
	ctx             context.Context
	suffix          string
	systemWorkspace *ent.Organization
	branch          *ent.Organization
	staffRole       *ent.Role
	inviter         *ent.User
}

func newDingTalkRegistrationFixture(t *testing.T) *dingTalkRegistrationFixture {
	t.Helper()
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:10]
	systemWorkspace, err := data.db.Organization.Create().
		SetCode("DTR-HQ-" + suffix).
		SetName("钉钉注册系统管理-" + suffix).
		SetKind("system").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建系统管理组织失败: %v", err)
	}
	branch, err := data.db.Organization.Create().
		SetCode("DTR-CD-" + suffix).
		SetName("钉钉注册成都公司-" + suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建分公司组织失败: %v", err)
	}

	ensurePermission := func(key, name string) *ent.Permission {
		permission, queryErr := data.db.Permission.Query().Where(permissionent.KeyEQ(key)).First(ctx)
		if ent.IsNotFound(queryErr) {
			permission, queryErr = data.db.Permission.Create().SetKey(key).SetName(name).SetDescription("钉钉注册集成测试").SetGroup("system").Save(ctx)
		}
		if queryErr != nil {
			t.Fatalf("准备权限 %s 失败: %v", key, queryErr)
		}
		return permission
	}
	managePermission := ensurePermission(access.UserDingTalkInvitationManage, "管理钉钉邀请与注册审批")
	staffRole, err := data.db.Role.Create().
		SetOrganizationID(branch.ID).
		SetCode("dtr_staff_" + suffix).
		SetName("初始员工角色-" + suffix).
		SetDataScope(roleent.DataScopeSelf).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建初始角色失败: %v", err)
	}
	inviter, err := data.db.User.Create().
		SetDisplayName("邀请人-" + suffix).
		SetDingtalkUserid("dtr-inviter-" + suffix).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建邀请人失败: %v", err)
	}
	inviterRole, err := data.db.Role.Create().
		SetOrganizationID(branch.ID).
		SetCode("dtr_inviter_" + suffix).
		SetName("邀请管理员-" + suffix).
		SetDataScope(roleent.DataScopeOrganization).
		AddPermissions(managePermission).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建邀请管理角色失败: %v", err)
	}
	inviterMembership, err := data.db.Membership.Create().
		SetUserID(inviter.ID).
		SetOrganizationID(branch.ID).
		SetPrimary(true).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建邀请人成员资格失败: %v", err)
	}
	if _, err := data.db.RoleAssignment.Create().SetMembershipID(inviterMembership.ID).SetRoleID(inviterRole.ID).Save(ctx); err != nil {
		t.Fatalf("分配邀请管理角色失败: %v", err)
	}

	return &dingTalkRegistrationFixture{
		t:               t,
		data:            data,
		repo:            NewDingTalkRegistrationRepo(data),
		authRepo:        NewAuthRepo(data).(*authRepo),
		ctx:             ctx,
		suffix:          suffix,
		systemWorkspace: systemWorkspace,
		branch:          branch,
		staffRole:       staffRole,
		inviter:         inviter,
	}
}

func (f *dingTalkRegistrationFixture) createInvitationInput(mobile string, organizationID uuid.UUID, expiresAt time.Time) *biz.DingTalkInvitation {
	token, _ := biz.GenerateInvitationToken()
	roleID := f.staffRole.ID
	m := mobile
	return &biz.DingTalkInvitation{
		Token:          token,
		Kind:           biz.DingTalkInvitationKindTargeted,
		OrganizationID: organizationID,
		RoleID:         &roleID,
		Mobile:         &m,
		DisplayName:    "备注-" + mobile,
		InvitedBy:      f.inviter.ID,
		Status:         biz.DingTalkInvitationStatusPending,
		ExpiresAt:      expiresAt,
	}
}

func TestDingTalkInvitationConsumeActivatesAccountEndToEnd(t *testing.T) {
	fixture := newDingTalkRegistrationFixture(t)
	mobile := "13800138000"
	created, err := fixture.repo.CreateInvitation(fixture.ctx, fixture.createInvitationInput(mobile, fixture.branch.ID, time.Now().Add(time.Hour)), &biz.AuditEvent{UserID: &fixture.inviter.ID, Action: "admin.dingtalk.invitation.create", Result: "success", Details: map[string]string{}})
	if err != nil {
		t.Fatalf("创建邀请失败: %v", err)
	}

	matched, err := fixture.repo.FindActiveInvitationByMobile(fixture.ctx, mobile)
	if err != nil {
		t.Fatalf("按手机号匹配邀请失败: %v", err)
	}
	if matched.ID != created.ID || matched.OrganizationID != fixture.branch.ID {
		t.Fatalf("匹配邀请 = %#v", matched)
	}

	identity := &biz.DingTalkIdentity{UnionID: "dtr-union-" + fixture.suffix, UserID: "dtr-user-" + fixture.suffix, CorpID: "ding-corp", Name: "邀请员工"}
	credential, err := fixture.repo.ConsumeInvitationAndActivate(fixture.ctx, identity, matched.ID)
	if err != nil {
		t.Fatalf("消费邀请自动激活失败: %v", err)
	}
	if !credential.Enabled || credential.PrimaryOrganizationID != fixture.branch.ID {
		t.Fatalf("激活后凭据 = %#v", credential)
	}

	account, err := fixture.data.db.User.Query().Where(user.DingtalkUnionidEQ(identity.UnionID)).Only(fixture.ctx)
	if err != nil || !account.Enabled {
		t.Fatalf("激活账号应为启用状态: %#v err=%v", account, err)
	}
	targetMembership, err := fixture.data.db.Membership.Query().
		Where(membershipent.UserIDEQ(account.ID), membershipent.OrganizationIDEQ(fixture.branch.ID)).Only(fixture.ctx)
	if err != nil || !targetMembership.Primary || !targetMembership.Enabled {
		t.Fatalf("目标组织成员资格 = %#v err=%v", targetMembership, err)
	}
	assignments, err := fixture.data.db.RoleAssignment.Query().Where(roleassignmentent.MembershipIDEQ(targetMembership.ID)).All(fixture.ctx)
	if err != nil || len(assignments) != 1 || assignments[0].RoleID != fixture.staffRole.ID {
		t.Fatalf("初始角色分配 = %#v err=%v", assignments, err)
	}

	invitation, err := fixture.data.db.DingTalkInvitation.Query().Where(invitationent.IDEQ(matched.ID)).Only(fixture.ctx)
	if err != nil || invitation.Status != invitationent.StatusCONSUMED || invitation.ConsumedBy == nil || *invitation.ConsumedBy != account.ID {
		t.Fatalf("邀请应标记 CONSUMED: %#v err=%v", invitation, err)
	}
	if _, err := fixture.repo.FindActiveInvitationByMobile(fixture.ctx, mobile); err != biz.ErrDingTalkInvitationNotFound {
		t.Fatalf("消费后同手机号应无活跃邀请: %v", err)
	}

	// 通知任务：本人授权完成 + 邀请人已激活。
	userAuthorized, err := fixture.data.db.NotificationDelivery.Query().
		Where(notificationent.RecipientUserIDEQ(account.ID), notificationent.TemplateEQ(notificationent.TemplateUSER_AUTHORIZED)).
		All(fixture.ctx)
	if err != nil || len(userAuthorized) != 1 {
		t.Fatalf("本人授权完成通知 = %#v err=%v", userAuthorized, err)
	}
	activated, err := fixture.data.db.NotificationDelivery.Query().
		Where(notificationent.RecipientUserIDEQ(fixture.inviter.ID), notificationent.TemplateEQ(notificationent.TemplateDINGTALK_INVITATION_ACTIVATED)).
		All(fixture.ctx)
	if err != nil || len(activated) != 1 || activated[0].ReferenceCode != identity.Name || activated[0].Parameter != fixture.branch.Name {
		t.Fatalf("邀请人激活通知 = %#v err=%v", activated, err)
	}

	audit, err := fixture.data.db.AuditLog.Query().
		Where(auditlogent.ActionEQ("auth.dingtalk.invitation.consume"), auditlogent.UserIDEQ(account.ID)).
		Only(fixture.ctx)
	if err != nil {
		t.Fatalf("消费审计缺失: %v", err)
	}
	var details map[string]string
	if err := json.Unmarshal(audit.Details, &details); err != nil {
		t.Fatalf("解析消费审计详情失败: %v", err)
	}
	if details["mobile"] != "138****8000" {
		t.Fatalf("审计手机号必须脱敏: %#v", details)
	}
}

func TestDingTalkInvitationLifecycleConstraints(t *testing.T) {
	fixture := newDingTalkRegistrationFixture(t)
	mobile := "13900139000"
	audit := &biz.AuditEvent{UserID: &fixture.inviter.ID, Action: "admin.dingtalk.invitation.create", Result: "success", Details: map[string]string{}}

	if _, err := fixture.repo.CreateInvitation(fixture.ctx, fixture.createInvitationInput(mobile, fixture.branch.ID, time.Now().Add(time.Hour)), audit); err != nil {
		t.Fatalf("创建首条邀请失败: %v", err)
	}
	if _, err := fixture.repo.CreateInvitation(fixture.ctx, fixture.createInvitationInput(mobile, fixture.branch.ID, time.Now().Add(time.Hour)), audit); err != biz.ErrDingTalkInvitationExists {
		t.Fatalf("同组织同手机号活跃邀请应冲突: %v", err)
	}
	// 撤销后可重建。
	if _, err := fixture.repo.RevokeInvitation(fixture.ctx, fixture.inviter.ID, mustInvitationIDByMobile(t, fixture, mobile), []uuid.UUID{fixture.branch.ID}, &biz.AuditEvent{UserID: &fixture.inviter.ID, Action: "admin.dingtalk.invitation.revoke", Result: "success", Details: map[string]string{}}); err != nil {
		t.Fatalf("撤销邀请失败: %v", err)
	}
	recreated, err := fixture.repo.CreateInvitation(fixture.ctx, fixture.createInvitationInput(mobile, fixture.branch.ID, time.Now().Add(-time.Minute)), audit)
	if err != nil {
		t.Fatalf("撤销后重建邀请失败: %v", err)
	}
	// 已过期的邀请不进入匹配链，消费按不可用处理。
	if _, err := fixture.repo.FindActiveInvitationByMobile(fixture.ctx, mobile); err != biz.ErrDingTalkInvitationNotFound {
		t.Fatalf("过期邀请不应匹配: %v", err)
	}
	identity := &biz.DingTalkIdentity{UnionID: "dtr-union-exp-" + fixture.suffix, UserID: "dtr-user-exp-" + fixture.suffix, CorpID: "ding-corp", Name: "过期邀请员工"}
	if _, err := fixture.repo.ConsumeInvitationAndActivate(fixture.ctx, identity, recreated.ID); err != biz.ErrDingTalkInvitationNotConsumable {
		t.Fatalf("过期邀请消费错误 = %v，期望 ErrDingTalkInvitationNotConsumable", err)
	}
	// 撤销范围越权。
	again, err := fixture.repo.CreateInvitation(fixture.ctx, fixture.createInvitationInput(mobile, fixture.branch.ID, time.Now().Add(time.Hour)), audit)
	if err != nil {
		t.Fatalf("重建有效邀请失败: %v", err)
	}
	if _, err := fixture.repo.RevokeInvitation(fixture.ctx, fixture.inviter.ID, again.ID, []uuid.UUID{fixture.systemWorkspace.ID}, &biz.AuditEvent{UserID: &fixture.inviter.ID, Action: "admin.dingtalk.invitation.revoke", Result: "success", Details: map[string]string{}}); err != biz.ErrPermissionDenied {
		t.Fatalf("越权撤销错误 = %v，期望 ErrPermissionDenied", err)
	}
}

func mustInvitationIDByMobile(t *testing.T, fixture *dingTalkRegistrationFixture, mobile string) uuid.UUID {
	t.Helper()
	item, err := fixture.data.db.DingTalkInvitation.Query().Where(invitationent.MobileEQ(mobile), invitationent.StatusEQ(invitationent.StatusPENDING)).Only(fixture.ctx)
	if err != nil {
		t.Fatalf("查询活跃邀请失败: %v", err)
	}
	return item.ID
}

// TestDingTalkInvitationMatchPrefersEarliestAcrossOrganizations 验证同一手机号在
// 多个组织各持一条活跃邀请时按「先建先得」匹配，而不是因多行命中报错降级。
func TestDingTalkInvitationMatchPrefersEarliestAcrossOrganizations(t *testing.T) {
	fixture := newDingTalkRegistrationFixture(t)
	otherOrganization, err := fixture.data.db.Organization.Create().
		SetCode("DTR-SH-" + fixture.suffix).
		SetName("钉钉注册上海公司-" + fixture.suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(fixture.ctx)
	if err != nil {
		t.Fatalf("创建第二公司组织失败: %v", err)
	}
	otherRole, err := fixture.data.db.Role.Create().
		SetOrganizationID(otherOrganization.ID).
		SetCode("dtr_staff_sh_" + fixture.suffix).
		SetName("上海初始角色-" + fixture.suffix).
		SetDataScope(roleent.DataScopeSelf).
		Save(fixture.ctx)
	if err != nil {
		t.Fatalf("创建第二组织角色失败: %v", err)
	}

	mobile := "13700137000"
	audit := &biz.AuditEvent{UserID: &fixture.inviter.ID, Action: "admin.dingtalk.invitation.create", Result: "success", Details: map[string]string{}}
	token1, _ := biz.GenerateInvitationToken()
	first, err := fixture.repo.CreateInvitation(fixture.ctx, &biz.DingTalkInvitation{
		Token:          token1,
		Kind:           biz.DingTalkInvitationKindTargeted,
		OrganizationID: fixture.branch.ID,
		RoleID:         &fixture.staffRole.ID,
		Mobile:         &mobile,
		InvitedBy:      fixture.inviter.ID,
		Status:         biz.DingTalkInvitationStatusPending,
		ExpiresAt:      time.Now().Add(time.Hour),
	}, audit)
	if err != nil {
		t.Fatalf("创建成都邀请失败: %v", err)
	}
	// 确保两条邀请 created_at 严格递增，避免同毫秒建行退化为 ID 排序。
	time.Sleep(10 * time.Millisecond)
	token2, _ := biz.GenerateInvitationToken()
	second, err := fixture.repo.CreateInvitation(fixture.ctx, &biz.DingTalkInvitation{
		Token:          token2,
		Kind:           biz.DingTalkInvitationKindTargeted,
		OrganizationID: otherOrganization.ID,
		RoleID:         &otherRole.ID,
		Mobile:         &mobile,
		InvitedBy:      fixture.inviter.ID,
		Status:         biz.DingTalkInvitationStatusPending,
		ExpiresAt:      time.Now().Add(time.Hour),
	}, audit)
	if err != nil {
		t.Fatalf("创建上海邀请失败: %v", err)
	}

	matched, err := fixture.repo.FindActiveInvitationByMobile(fixture.ctx, mobile)
	if err != nil {
		t.Fatalf("多组织活跃邀请不应因多行命中报错: %v", err)
	}
	if matched.ID != first.ID || matched.OrganizationID != fixture.branch.ID {
		t.Fatalf("匹配应先建先得（成都）: matched=%#v first=%#v second=%#v", matched, first, second)
	}
}

// TestDingTalkRegistrationQueueAndApproval 覆盖通道 B：注册落库路由、队列过滤、
// 一站式同意（幂等冲突）、拒绝出队与通知。
func TestDingTalkRegistrationQueueAndApproval(t *testing.T) {
	fixture := newDingTalkRegistrationFixture(t)

	// 分公司管理员（收件人路由目标）；两位审批人验证多收件人各自成单。
	branchApprover, err := fixture.data.db.User.Create().
		SetDisplayName("成都审批人-" + fixture.suffix).
		SetDingtalkUserid("dtr-approver-" + fixture.suffix).
		SetEnabled(true).
		Save(fixture.ctx)
	if err != nil {
		t.Fatalf("创建审批人失败: %v", err)
	}
	branchApprover2, err := fixture.data.db.User.Create().
		SetDisplayName("成都审批人2-" + fixture.suffix).
		SetDingtalkUserid("dtr-approver2-" + fixture.suffix).
		SetEnabled(true).
		Save(fixture.ctx)
	if err != nil {
		t.Fatalf("创建第二审批人失败: %v", err)
	}
	managePermission, queryErr := fixture.data.db.Permission.Query().Where(permissionent.KeyEQ(access.UserDingTalkInvitationManage)).Only(fixture.ctx)
	if queryErr != nil {
		t.Fatalf("查询权限失败: %v", queryErr)
	}
	approverRole, err := fixture.data.db.Role.Create().
		SetOrganizationID(fixture.branch.ID).
		SetCode("dtr_approver_" + fixture.suffix).
		SetName("成都审批角色-" + fixture.suffix).
		SetDataScope(roleent.DataScopeOrganization).
		AddPermissions(managePermission).
		Save(fixture.ctx)
	if err != nil {
		t.Fatalf("创建审批角色失败: %v", err)
	}
	for _, approver := range []*ent.User{branchApprover, branchApprover2} {
		approverMembership, err := fixture.data.db.Membership.Create().
			SetUserID(approver.ID).
			SetOrganizationID(fixture.branch.ID).
			SetPrimary(true).
			SetEnabled(true).
			Save(fixture.ctx)
		if err != nil {
			t.Fatalf("创建审批人成员资格失败: %v", err)
		}
		if _, err := fixture.data.db.RoleAssignment.Create().SetMembershipID(approverMembership.ID).SetRoleID(approverRole.ID).Save(fixture.ctx); err != nil {
			t.Fatalf("分配审批角色失败: %v", err)
		}
	}

	// 收件人路由：目标组织内有权限且启用的人入选；系统管理兜底收件人不含分公司注册。
	recipients, err := fixture.repo.ListApproverRecipients(fixture.ctx, fixture.branch.ID)
	if err != nil {
		t.Fatalf("查询审批收件人失败: %v", err)
	}
	found := false
	for _, recipient := range recipients {
		if recipient.UserID == branchApprover.ID {
			found = true
		}
		if recipient.UserID == fixture.inviter.ID && fixture.inviter.ID != branchApprover.ID {
			// 邀请人也在目标组织持权限，属于合法收件人；不视为异常。
			_ = recipient
		}
	}
	if !found {
		t.Fatalf("成都审批人应在收件人列表: %#v", recipients)
	}
	hqRecipients, err := fixture.repo.ListApproverRecipients(fixture.ctx, fixture.systemWorkspace.ID)
	if err != nil {
		t.Fatalf("查询系统管理收件人失败: %v", err)
	}
	for _, recipient := range hqRecipients {
		if recipient.UserID == branchApprover.ID {
			t.Fatal("成都审批人不应出现在系统管理兜底收件人中")
		}
	}

	// 通道 B 注册：自选成都公司，通知两位审批人。
	identity := &biz.DingTalkIdentity{UnionID: "dtr-union-b-" + fixture.suffix, UserID: "dtr-user-b-" + fixture.suffix, CorpID: "ding-corp", Name: "认领员工"}
	requestedOrg := fixture.branch.ID
	credential, created, err := fixture.authRepo.RegisterDingTalkCredential(fixture.ctx, identity, &requestedOrg, &biz.DingTalkApproverNotice{ApproverUserIDs: approverUserIDsOf(branchApprover.ID, branchApprover2.ID), OrganizationName: fixture.branch.Name}, &biz.AuditEvent{Action: "auth.dingtalk.register", Result: "success"})
	if err != nil || !created || credential.Enabled {
		t.Fatalf("注册落库结果 = (%#v, %v, %v)", credential, created, err)
	}

	// 多收件人各自成单：每位审批人都有自己的任务与明细，不因确定性幂等键
	// 坍缩为单人（1 任务 = 1 明细 = 1 收件人）。
	pendingDeliveries, err := fixture.data.db.NotificationDelivery.Query().
		Where(notificationent.TemplateEQ(notificationent.TemplateDINGTALK_REGISTRATION_PENDING), notificationent.ResourceIDEQ(credential.UserID)).
		All(fixture.ctx)
	if err != nil || len(pendingDeliveries) != 2 {
		t.Fatalf("两位审批人应各有通知明细 = %#v err=%v", pendingDeliveries, err)
	}
	// 通知卡片必须展示自选目标组织名，而不是固定系统管理名。
	for _, delivery := range pendingDeliveries {
		if delivery.Parameter != fixture.branch.Name {
			t.Fatalf("通知应展示目标组织名 %q，实际 %q", fixture.branch.Name, delivery.Parameter)
		}
	}
	taskIDs := make(map[uuid.UUID]struct{}, len(pendingDeliveries))
	recipientSeen := make(map[uuid.UUID]struct{}, len(pendingDeliveries))
	for _, delivery := range pendingDeliveries {
		taskIDs[delivery.BackgroundTaskID] = struct{}{}
		recipientSeen[delivery.RecipientUserID] = struct{}{}
	}
	if len(taskIDs) != 2 {
		t.Fatalf("两位审批人应各挂独立任务（坍缩为单人）: %#v", pendingDeliveries)
	}
	if _, ok := recipientSeen[branchApprover.ID]; !ok {
		t.Fatalf("第一位审批人缺少通知: %#v", recipientSeen)
	}
	if _, ok := recipientSeen[branchApprover2.ID]; !ok {
		t.Fatalf("第二位审批人缺少通知: %#v", recipientSeen)
	}
	// 重复确认：按 (注册人, 组织, 收件人) 幂等键去重，不重复提醒。
	if _, _, err := fixture.authRepo.RegisterDingTalkCredential(fixture.ctx, identity, &requestedOrg, &biz.DingTalkApproverNotice{ApproverUserIDs: approverUserIDsOf(branchApprover.ID, branchApprover2.ID), OrganizationName: fixture.branch.Name}, &biz.AuditEvent{Action: "auth.dingtalk.register", Result: "success"}); err != nil {
		t.Fatalf("重复确认注册失败: %v", err)
	}
	pendingDeliveries, err = fixture.data.db.NotificationDelivery.Query().
		Where(notificationent.TemplateEQ(notificationent.TemplateDINGTALK_REGISTRATION_PENDING), notificationent.ResourceIDEQ(credential.UserID)).
		All(fixture.ctx)
	if err != nil || len(pendingDeliveries) != 2 {
		t.Fatalf("重复确认不应重复入队通知: %#v err=%v", pendingDeliveries, err)
	}

	// 队列路由：成都组织能看到，系统管理组织看不到该注册。
	branchQueue, err := fixture.repo.ListPendingRegistrations(fixture.ctx, []uuid.UUID{fixture.branch.ID}, biz.DingTalkRegistrationListOptions{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("查询成都审批队列失败: %v", err)
	}
	if len(branchQueue.Items) != 1 || branchQueue.Items[0].UserID != credential.UserID || branchQueue.Items[0].RequestedOrganizationID == nil || *branchQueue.Items[0].RequestedOrganizationID != fixture.branch.ID {
		t.Fatalf("成都审批队列 = %#v", branchQueue.Items)
	}
	hqQueue, err := fixture.repo.ListPendingRegistrations(fixture.ctx, []uuid.UUID{fixture.systemWorkspace.ID}, biz.DingTalkRegistrationListOptions{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("查询系统管理审批队列失败: %v", err)
	}
	if len(hqQueue.Items) != 0 {
		t.Fatalf("成都注册不应出现在系统管理队列: %#v", hqQueue.Items)
	}

	// 越权审批：系统管理范围处理成都注册 → 403。
	if _, err := fixture.repo.ApproveRegistration(fixture.ctx, &biz.DingTalkRegistrationDecision{
		ActorID:         branchApprover.ID,
		UserID:          credential.UserID,
		OrganizationIDs: []uuid.UUID{fixture.systemWorkspace.ID},
		RoleIDs:         []uuid.UUID{fixture.staffRole.ID},
		Notification:    biz.NewDingTalkUserAuthorizedNotification(credential.UserID),
		Audit:           &biz.AuditEvent{UserID: &branchApprover.ID, Action: "admin.dingtalk.registration.approve", Result: "success", Details: map[string]string{}},
	}); err != biz.ErrPermissionDenied {
		t.Fatalf("越权审批错误 = %v，期望 ErrPermissionDenied", err)
	}

	// 一站式同意。
	approved, err := fixture.repo.ApproveRegistration(fixture.ctx, &biz.DingTalkRegistrationDecision{
		ActorID:         branchApprover.ID,
		UserID:          credential.UserID,
		OrganizationIDs: []uuid.UUID{fixture.branch.ID},
		RoleIDs:         []uuid.UUID{fixture.staffRole.ID},
		Notification:    biz.NewDingTalkUserAuthorizedNotification(credential.UserID),
		Audit:           &biz.AuditEvent{UserID: &branchApprover.ID, OrganizationID: &fixture.branch.ID, Action: "admin.dingtalk.registration.approve", Result: "success"},
	})
	if err != nil {
		t.Fatalf("审批同意失败: %v", err)
	}
	if approved.RequestedOrganizationID == nil || *approved.RequestedOrganizationID != fixture.branch.ID {
		t.Fatalf("审批结果组织 = %#v", approved)
	}
	account, err := fixture.data.db.User.Query().Where(user.IDEQ(credential.UserID)).Only(fixture.ctx)
	if err != nil || !account.Enabled {
		t.Fatalf("审批后账号应启用: %#v err=%v", account, err)
	}
	enabledMembership, err := fixture.data.db.Membership.Query().Where(membershipent.UserIDEQ(account.ID), membershipent.EnabledEQ(true)).Only(fixture.ctx)
	if err != nil || enabledMembership.OrganizationID != fixture.branch.ID || !enabledMembership.Primary {
		t.Fatalf("审批后成员资格 = %#v err=%v", enabledMembership, err)
	}
	// 幂等：二次审批返回冲突。
	if _, err := fixture.repo.ApproveRegistration(fixture.ctx, &biz.DingTalkRegistrationDecision{
		ActorID:         branchApprover.ID,
		UserID:          credential.UserID,
		OrganizationIDs: []uuid.UUID{fixture.branch.ID},
		RoleIDs:         []uuid.UUID{fixture.staffRole.ID},
		Notification:    biz.NewDingTalkUserAuthorizedNotification(credential.UserID),
		Audit:           &biz.AuditEvent{UserID: &branchApprover.ID, Action: "admin.dingtalk.registration.approve", Result: "success", Details: map[string]string{}},
	}); err != biz.ErrDingTalkRegistrationProcessed {
		t.Fatalf("重复审批错误 = %v，期望 ErrDingTalkRegistrationProcessed", err)
	}

	// 兜底注册（未自选组织）应出现在系统管理队列。
	fallbackIdentity := &biz.DingTalkIdentity{UnionID: "dtr-union-hq-" + fixture.suffix, UserID: "dtr-user-hq-" + fixture.suffix, CorpID: "ding-corp", Name: "系统管理兜底员工"}
	fallbackCredential, created, err := fixture.authRepo.RegisterDingTalkCredential(fixture.ctx, fallbackIdentity, nil, nil, &biz.AuditEvent{Action: "auth.dingtalk.register", Result: "success"})
	if err != nil || !created {
		t.Fatalf("兜底注册失败: (%#v, %v, %v)", fallbackCredential, created, err)
	}
	hqQueue, err = fixture.repo.ListPendingRegistrations(fixture.ctx, []uuid.UUID{fixture.systemWorkspace.ID}, biz.DingTalkRegistrationListOptions{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("查询系统管理兜底队列失败: %v", err)
	}
	if len(hqQueue.Items) != 1 || hqQueue.Items[0].UserID != fallbackCredential.UserID {
		t.Fatalf("系统管理兜底队列 = %#v", hqQueue.Items)
	}

	// 拒绝：停用收口成员资格、通知本人、出队。
	reason := "身份信息与通讯录不符"
	if err := fixture.repo.RejectRegistration(fixture.ctx, &biz.DingTalkRegistrationDecision{
		ActorID:         branchApprover.ID,
		UserID:          fallbackCredential.UserID,
		OrganizationIDs: []uuid.UUID{fixture.systemWorkspace.ID},
		Reason:          reason,
		Notification:    biz.NewDingTalkRegistrationRejectedNotification(fallbackCredential.UserID),
		Audit:           &biz.AuditEvent{UserID: &branchApprover.ID, OrganizationID: &fixture.systemWorkspace.ID, Action: "admin.dingtalk.registration.reject", Result: "success", Details: map[string]string{"value": reason}},
	}); err != nil {
		t.Fatalf("拒绝注册失败: %v", err)
	}
	remaining, err := fixture.data.db.Membership.Query().Where(membershipent.UserIDEQ(fallbackCredential.UserID), membershipent.EnabledEQ(true)).Count(fixture.ctx)
	if err != nil || remaining != 0 {
		t.Fatalf("拒绝后启用成员资格应清零: %d err=%v", remaining, err)
	}
	hqQueue, err = fixture.repo.ListPendingRegistrations(fixture.ctx, []uuid.UUID{fixture.systemWorkspace.ID}, biz.DingTalkRegistrationListOptions{Page: 1, PageSize: 20})
	if err != nil || len(hqQueue.Items) != 0 {
		t.Fatalf("拒绝后应出队: %#v err=%v", hqQueue.Items, err)
	}
	rejectedDeliveries, err := fixture.data.db.NotificationDelivery.Query().
		Where(notificationent.RecipientUserIDEQ(fallbackCredential.UserID), notificationent.TemplateEQ(notificationent.TemplateDINGTALK_REGISTRATION_REJECTED)).
		All(fixture.ctx)
	if err != nil || len(rejectedDeliveries) != 1 || rejectedDeliveries[0].Parameter != reason {
		t.Fatalf("拒绝通知 = %#v err=%v", rejectedDeliveries, err)
	}
	rejectAudit, err := fixture.data.db.AuditLog.Query().
		Where(auditlogent.ActionEQ("admin.dingtalk.registration.reject"), auditlogent.UserIDEQ(branchApprover.ID)).Only(fixture.ctx)
	if err != nil {
		t.Fatalf("拒绝审计缺失: %v", err)
	}
	var rejectDetails map[string]string
	if err := json.Unmarshal(rejectAudit.Details, &rejectDetails); err != nil || rejectDetails["value"] != reason {
		t.Fatalf("拒绝审计详情 = %v err=%v", rejectDetails, err)
	}
}

func approverUserIDsOf(ids ...uuid.UUID) []uuid.UUID {
	return ids
}

// TestDingTalkApproverRecipientsLimitByActivity 验证收件人上限保护：超过 10 人时
// 按最近会话活跃截断。
func TestDingTalkApproverRecipientsLimitByActivity(t *testing.T) {
	fixture := newDingTalkRegistrationFixture(t)
	managePermission, queryErr := fixture.data.db.Permission.Query().Where(permissionent.KeyEQ(access.UserDingTalkInvitationManage)).Only(fixture.ctx)
	if queryErr != nil {
		t.Fatalf("查询权限失败: %v", queryErr)
	}
	role, err := fixture.data.db.Role.Create().
		SetOrganizationID(fixture.branch.ID).
		SetCode("dtr_bulk_" + fixture.suffix).
		SetName("批量审批角色-" + fixture.suffix).
		SetDataScope(roleent.DataScopeOrganization).
		AddPermissions(managePermission).
		Save(fixture.ctx)
	if err != nil {
		t.Fatalf("创建批量角色失败: %v", err)
	}
	mostActive := uuid.UUID{}
	for index := 0; index < 12; index++ {
		member, err := fixture.data.db.User.Create().
			SetDisplayName("批量审批人-" + fixture.suffix + "-" + string(rune('a'+index))).
			SetDingtalkUserid("dtr-bulk-" + fixture.suffix + "-" + string(rune('a'+index))).
			SetEnabled(true).
			Save(fixture.ctx)
		if err != nil {
			t.Fatalf("创建批量审批人失败: %v", err)
		}
		membership, err := fixture.data.db.Membership.Create().
			SetUserID(member.ID).
			SetOrganizationID(fixture.branch.ID).
			SetEnabled(true).
			Save(fixture.ctx)
		if err != nil {
			t.Fatalf("创建批量成员资格失败: %v", err)
		}
		if _, err := fixture.data.db.RoleAssignment.Create().SetMembershipID(membership.ID).SetRoleID(role.ID).Save(fixture.ctx); err != nil {
			t.Fatalf("分配批量角色失败: %v", err)
		}
		// 只给第 1 位建立会话：最近活跃者必须进入截断名单。
		if index == 0 {
			mostActive = member.ID
			if _, err := fixture.data.db.Session.Create().
				SetTokenHash("dtr-activity-" + fixture.suffix).
				SetUserID(member.ID).
				SetOrganizationID(fixture.branch.ID).
				SetExpiresAt(time.Now().Add(time.Hour)).
				Save(fixture.ctx); err != nil {
				t.Fatalf("创建会话失败: %v", err)
			}
		}
	}
	recipients, err := fixture.repo.ListApproverRecipients(fixture.ctx, fixture.branch.ID)
	if err != nil {
		t.Fatalf("查询收件人失败: %v", err)
	}
	if len(recipients) > 10 {
		t.Fatalf("收件人应截断到 10 人，实际 %d", len(recipients))
	}
	first := recipients[0]
	if first.UserID != mostActive {
		t.Fatalf("最近活跃者应排首: %#v", recipients)
	}
}

func TestDingTalkGenericInvitationAndTransferAndEscalation(t *testing.T) {
	fixture := newDingTalkRegistrationFixture(t)
	// 1. 创建 GENERIC 邀请（手机号为空，Token 128-bit）
	token, err := biz.GenerateInvitationToken()
	if err != nil {
		t.Fatalf("生成 Token 失败: %v", err)
	}
	genericInvitation, err := fixture.repo.CreateInvitation(fixture.ctx, &biz.DingTalkInvitation{
		Token:          token,
		Kind:           biz.DingTalkInvitationKindGeneric,
		OrganizationID: fixture.branch.ID,
		DisplayName:    "分公司通用扩招码",
		InvitedBy:      fixture.inviter.ID,
		Status:         biz.DingTalkInvitationStatusPending,
		ExpiresAt:      time.Now().Add(72 * time.Hour),
	}, &biz.AuditEvent{UserID: &fixture.inviter.ID, Action: "admin.dingtalk.invitation.create", Result: "success", Details: map[string]string{}})
	if err != nil {
		t.Fatalf("创建通用邀请失败: %v", err)
	}
	if genericInvitation.Kind != biz.DingTalkInvitationKindGeneric || genericInvitation.Mobile != nil {
		t.Fatalf("通用邀请属性异常: %#v", genericInvitation)
	}

	// 2. 通过 Token 查询邀请公开信息
	found, err := fixture.repo.FindInvitationByToken(fixture.ctx, token)
	if err != nil {
		t.Fatalf("FindInvitationByToken 失败: %v", err)
	}
	if found.ID != genericInvitation.ID {
		t.Fatalf("Token 查询结果不一致: %v vs %v", found.ID, genericInvitation.ID)
	}

	// 3. 向上追溯：创建新分公司（无管理员），断言向上追溯命中系统管理
	subBranch, err := fixture.data.db.Organization.Create().
		SetCode("SUB-" + fixture.suffix).
		SetName("二级办事处-" + fixture.suffix).
		SetKind(organizationent.KindCompany).
		SetBaseCurrency("CNY").
		SetEnabled(true).
		Save(fixture.ctx)
	if err != nil {
		t.Fatalf("创建二级办事处失败: %v", err)
	}
	// 在系统管理配置审批人
	managePermission, _ := fixture.data.db.Permission.Query().Where(permissionent.KeyEQ(access.UserDingTalkInvitationManage)).Only(fixture.ctx)
	hqRole, _ := fixture.data.db.Role.Create().
		SetOrganizationID(fixture.systemWorkspace.ID).
		SetName("系统管理管理员角色-" + fixture.suffix).
		SetCode("hq_admin_" + fixture.suffix).
		SetEnabled(true).
		AddPermissions(managePermission).
		Save(fixture.ctx)
	hqAdmin, _ := fixture.data.db.User.Create().
		SetDisplayName("系统管理审批人").
		SetDingtalkUnionid("hq-union-" + fixture.suffix).
		SetDingtalkUserid("hq-user-" + fixture.suffix).
		SetDingtalkName("系统管理审批人").
		SetEnabled(true).
		Save(fixture.ctx)
	hqMembership, _ := fixture.data.db.Membership.Create().
		SetUserID(hqAdmin.ID).
		SetOrganizationID(fixture.systemWorkspace.ID).
		SetPrimary(true).
		SetEnabled(true).
		Save(fixture.ctx)
	fixture.data.db.RoleAssignment.Create().
		SetMembershipID(hqMembership.ID).
		SetRoleID(hqRole.ID).
		Save(fixture.ctx)

	// 追溯决策已上移 biz 层（组合仓储原语），此处经 biz 函数驱动同一仓储原语验证真实库行为。
	recipients, escalatedOrgID, isEscalated, err := biz.ListApproverRecipientsWithEscalation(fixture.ctx, fixture.repo, subBranch.ID)
	if err != nil {
		t.Fatalf("向上追溯失败: %v", err)
	}
	if !isEscalated || escalatedOrgID != fixture.systemWorkspace.ID || len(recipients) == 0 {
		t.Fatalf("办事处无审批人应向上追溯至系统管理: isEscalated=%v, escalatedOrgID=%v, recipients=%v", isEscalated, escalatedOrgID, recipients)
	}

	// 4. 转派测试：注册员工初建在 branch，转派至 subBranch
	employeeIdentity := &biz.DingTalkIdentity{UnionID: "transfer-union-" + fixture.suffix, UserID: "transfer-user-" + fixture.suffix, CorpID: "ding-corp", Name: "转派员工"}
	cred, created, err := fixture.authRepo.RegisterDingTalkCredential(fixture.ctx, employeeIdentity, &fixture.branch.ID, nil, &biz.AuditEvent{Action: "auth.dingtalk.register", Result: "success"})
	if err != nil || !created {
		t.Fatalf("注册失败: %v", err)
	}

	// 尝试转派到同一组织：应拦截
	decision := &biz.DingTalkRegistrationDecision{
		ActorID:         hqAdmin.ID,
		UserID:          cred.UserID,
		OrganizationIDs: []uuid.UUID{fixture.branch.ID, fixture.systemWorkspace.ID, subBranch.ID},
		Reason:          "转派至办事处",
		Audit:           &biz.AuditEvent{Action: "admin.dingtalk.registration.transfer", Result: "success", Details: map[string]string{}},
	}
	if err := fixture.repo.TransferRegistration(fixture.ctx, decision, fixture.branch.ID); err != biz.ErrDingTalkRegistrationTransferSame {
		t.Fatalf("转派至同组织应拦截，实际: %v", err)
	}

	// 正常转派至 subBranch
	if err := fixture.repo.TransferRegistration(fixture.ctx, decision, subBranch.ID); err != nil {
		t.Fatalf("转派至办事处失败: %v", err)
	}
	updatedUser, err := fixture.data.db.User.Get(fixture.ctx, cred.UserID)
	if err != nil || updatedUser.DingtalkRequestedOrganizationID == nil || *updatedUser.DingtalkRequestedOrganizationID != subBranch.ID {
		t.Fatalf("转派后目标组织未更新: %v", updatedUser.DingtalkRequestedOrganizationID)
	}
}
