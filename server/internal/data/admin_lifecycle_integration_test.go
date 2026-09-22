package data

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	backgroundtaskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	notificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/notificationdelivery"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	sessionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/session"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"

	"github.com/google/uuid"
)

// TestAdminEmployeeLifecyclePostgres 在隔离 Schema + 完整版本化迁移链上验证员工
// 生命周期（离职 → 返聘审批 → 重新授权 → 多组织调出）的数据一致性与权限边界；
// 不依赖 public schema 的开发数据，夹具随 Schema 一并清理。
func TestAdminEmployeeLifecyclePostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	dingUnionID := "union-" + suffix
	dingUserID := "ding-user-" + suffix

	systemWorkspace, err := data.db.Organization.Create().
		SetCode("HQ-" + suffix).
		SetName("系统管理-" + suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建系统管理: %v", err)
	}
	intakeWorkspace, err := data.db.Organization.Create().SetCode("SYS-" + suffix).SetName("系统管理").SetKind("system").SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	role, err := data.db.Role.Create().
		SetOrganizationID(systemWorkspace.ID).
		SetCode("operator").
		SetName("操作员").
		SetDataScope("organization").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建角色: %v", err)
	}
	account, err := data.db.User.Create().
		SetDisplayName("返聘员工-" + suffix).
		SetDingtalkUnionid(dingUnionID).
		SetDingtalkUserid(dingUserID).
		SetDingtalkName("返聘员工-" + suffix).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建员工: %v", err)
	}
	membershipRecord, err := data.db.Membership.Create().
		SetUserID(account.ID).
		SetOrganizationID(systemWorkspace.ID).
		SetPrimary(true).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建组织关系: %v", err)
	}
	if _, err := data.db.RoleAssignment.Create().SetMembershipID(membershipRecord.ID).SetRoleID(role.ID).Save(ctx); err != nil {
		t.Fatalf("分配角色: %v", err)
	}
	if _, err := data.db.Session.Create().
		SetUserID(account.ID).
		SetOrganizationID(systemWorkspace.ID).
		SetTokenHash("integration-session-" + suffix).
		SetExpiresAt(time.Now().Add(time.Hour)).
		Save(ctx); err != nil {
		t.Fatalf("创建会话: %v", err)
	}

	adminRepo := &adminRepo{data: data}
	if err := adminRepo.DeleteUserMembership(ctx, account.ID, membershipRecord.ID, adminLifecycleAudit(systemWorkspace.ID, "admin.user.membership.delete")); err != biz.ErrAdminUserLastMembership {
		t.Fatalf("移出最后组织 error = %v, want ErrAdminUserLastMembership", err)
	}
	if err := adminRepo.TerminateUser(ctx, systemWorkspace.ID, account.ID, adminLifecycleAudit(systemWorkspace.ID, "admin.user.terminate")); err != nil {
		t.Fatalf("办理离职: %v", err)
	}
	terminated, err := data.db.User.Get(ctx, account.ID)
	if err != nil || terminated.Enabled {
		t.Fatalf("离职账号状态 = %#v, error = %v", terminated, err)
	}
	activeMemberships, err := data.db.Membership.Query().Where(membership.UserIDEQ(account.ID), membership.EnabledEQ(true)).Count(ctx)
	if err != nil || activeMemberships != 0 {
		t.Fatalf("离职后有效组织数 = %d, error = %v", activeMemberships, err)
	}
	assignments, err := data.db.RoleAssignment.Query().Where(roleassignment.MembershipIDEQ(membershipRecord.ID)).Count(ctx)
	if err != nil || assignments != 0 {
		t.Fatalf("离职后角色数 = %d, error = %v", assignments, err)
	}
	activeSessions, err := data.db.Session.Query().Where(sessionent.UserIDEQ(account.ID), sessionent.RevokedAtIsNil()).Count(ctx)
	if err != nil || activeSessions != 0 {
		t.Fatalf("离职后有效会话数 = %d, error = %v", activeSessions, err)
	}

	authRepo := &authRepo{data: data}
	credential, submitted, err := authRepo.RegisterDingTalkCredential(ctx, &biz.DingTalkIdentity{
		UnionID: dingUnionID,
		UserID:  dingUserID,
		Name:    "返聘员工-" + suffix,
	}, nil, nil, &biz.AuditEvent{Action: "auth.dingtalk.register", Result: "success"})
	if err != nil {
		t.Fatalf("提交返聘登记: %v", err)
	}
	if !submitted || credential.Enabled {
		t.Fatalf("返聘登记结果 submitted=%v credential=%#v", submitted, credential)
	}
	activeMemberships, err = data.db.Membership.Query().Where(membership.UserIDEQ(account.ID), membership.EnabledEQ(true)).Count(ctx)
	if err != nil || activeMemberships != 1 {
		t.Fatalf("返聘待审批组织数 = %d, error = %v", activeMemberships, err)
	}
	assignments, err = data.db.RoleAssignment.Query().Where(roleassignment.MembershipIDEQ(membershipRecord.ID)).Count(ctx)
	if err != nil || assignments != 0 {
		t.Fatalf("返聘待审批不应恢复旧角色，角色数 = %d, error = %v", assignments, err)
	}
	pendingUsername := "pending." + suffix
	if err := adminRepo.ResetUserPassword(ctx, systemWorkspace.ID, account.ID, "pending-password-hash", &pendingUsername, adminLifecycleAudit(systemWorkspace.ID, "admin.user.password.reset")); err != biz.ErrAdminUserNotFound {
		t.Fatalf("待审批账号设置密码 error = %v, want ErrAdminUserNotFound", err)
	}
	if _, err := adminRepo.UpdateUser(ctx, systemWorkspace.ID, account.ID, &biz.AdminUser{
		ID:          account.ID,
		DisplayName: "返聘员工",
		Enabled:     true,
	}, []uuid.UUID{role.ID}, adminLifecycleAudit(systemWorkspace.ID, "admin.user.update")); err != biz.ErrAdminUserAuthorizationRequired {
		t.Fatalf("普通编辑绕过外部身份授权 error = %v, want ErrAdminUserAuthorizationRequired", err)
	}

	notification := biz.NewDingTalkUserAuthorizedNotification(account.ID)
	authorized, err := adminRepo.AuthorizeDingTalkUser(ctx, intakeWorkspace.ID, systemWorkspace.ID, &biz.AdminUser{
		ID:          account.ID,
		DisplayName: "返聘员工",
	}, []uuid.UUID{role.ID}, notification, adminLifecycleAudit(systemWorkspace.ID, "admin.user.dingtalk.authorize"))
	if err != nil {
		t.Fatalf("返聘重新授权: %v", err)
	}
	if !authorized.Enabled || authorized.Status != biz.AdminUserStatusActive {
		t.Fatalf("返聘授权结果 = %#v", authorized)
	}
	if exists, err := data.db.User.Query().Where(userent.IDEQ(account.ID), userent.EnabledEQ(true)).Exist(ctx); err != nil || !exists {
		t.Fatalf("返聘账号未恢复在职，exists=%v error=%v", exists, err)
	}
	task, err := data.db.BackgroundTask.Query().Where(backgroundtaskent.IDEQ(notification.ID)).Only(ctx)
	if err != nil || task.Kind != backgroundtaskent.KindDINGTALK_NOTIFICATION || task.Status != backgroundtaskent.StatusPENDING {
		t.Fatalf("授权通知任务 = %#v, error = %v", task, err)
	}
	delivery, err := data.db.NotificationDelivery.Query().Where(notificationent.BackgroundTaskIDEQ(notification.ID)).Only(ctx)
	if err != nil || delivery.RecipientUserID != account.ID || delivery.Template != notificationent.TemplateUSER_AUTHORIZED || delivery.ResourceType != "USER" || delivery.ResourceID != account.ID || delivery.ReferenceCode != "" {
		t.Fatalf("授权通知明细 = %#v, error = %v", delivery, err)
	}

	backupUsername := "backup." + suffix
	if err := adminRepo.ResetUserPassword(ctx, systemWorkspace.ID, account.ID, "active-password-hash", &backupUsername, adminLifecycleAudit(systemWorkspace.ID, "admin.user.password.reset")); err != nil {
		t.Fatalf("为在职账号设置备用账密: %v", err)
	}
	activeAccount, err := data.db.User.Get(ctx, account.ID)
	if err != nil || activeAccount.Username != backupUsername || activeAccount.PasswordHash == nil || *activeAccount.PasswordHash != "active-password-hash" {
		t.Fatalf("备用账密保存结果 = %#v, error = %v", activeAccount, err)
	}
	actorRoles, err := adminRepo.GetActorRolesPrivilegeProfiles(ctx, systemWorkspace.ID, account.ID)
	if err != nil || len(actorRoles) != 1 || actorRoles[0].Code != "operator" {
		t.Fatalf("在职账号角色能力 = %#v, error = %v", actorRoles, err)
	}

	company, err := data.db.Organization.Create().
		SetCode("COMPANY-" + suffix).
		SetName("分公司-" + suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建分公司: %v", err)
	}
	if _, err := data.db.Membership.Create().
		SetUserID(account.ID).
		SetOrganizationID(company.ID).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("加入分公司: %v", err)
	}
	if err := adminRepo.DeleteUserMembership(ctx, account.ID, membershipRecord.ID, adminLifecycleAudit(systemWorkspace.ID, "admin.user.membership.delete")); err != nil {
		t.Fatalf("移出系统管理: %v", err)
	}
	// 工作台范围口径变更：移出系统管理后用户仍归属公司（系统管理子树），系统管理工作台仍可为其
	// 重置密码；精确组织口径下的「历史组织不可重置」不再成立。
	replacementUsername := "replacement." + suffix
	if err := adminRepo.ResetUserPassword(ctx, intakeWorkspace.ID, account.ID, "replacement-password-hash", &replacementUsername, adminLifecycleAudit(systemWorkspace.ID, "admin.user.password.reset")); err != nil {
		t.Fatalf("系统管理工作台为公司子树成员重置密码: %v", err)
	}
	retainedAccount, err := data.db.User.Get(ctx, account.ID)
	if err != nil || !retainedAccount.Enabled || retainedAccount.Username != replacementUsername || retainedAccount.PasswordHash == nil || *retainedAccount.PasswordHash != "replacement-password-hash" {
		t.Fatalf("重置密码后的账号结果 = %#v, error = %v", retainedAccount, err)
	}
	if _, err := adminRepo.GetActorRolesPrivilegeProfiles(ctx, systemWorkspace.ID, account.ID); err != biz.ErrAdminPrivilegeEscalation {
		t.Fatalf("历史组织读取操作者能力 error = %v, want ErrAdminPrivilegeEscalation", err)
	}
}

// adminLifecycleAudit 携带组织上下文构造审计事件。
func adminLifecycleAudit(organizationID uuid.UUID, action string) *biz.AuditEvent {
	return &biz.AuditEvent{OrganizationID: &organizationID, Action: action, Result: "success", Details: map[string]string{}}
}

// TestAdminDepartmentInheritsCompanyRolesPostgres 验证部门/团队公用所属公司的角色与工作区：
// 1. 查询部门角色（ListRoles）自动返回所属公司的角色库；
// 2. 为部门成员分配角色（CreateUserMembership）及提权校验通过；
// 3. 用户在部门下的成员关系在所属公司工作台下能正确收集角色画像（actorRolesPrivilegeProfiles）。
func TestAdminDepartmentInheritsCompanyRolesPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]

	company, err := data.db.Organization.Create().
		SetCode("CO-" + suffix).
		SetName("分公司-" + suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建分公司: %v", err)
	}

	financeRole, err := data.db.Role.Create().
		SetOrganizationID(company.ID).
		SetCode("finance_" + suffix).
		SetName("财务角色-" + suffix).
		SetDataScope("organization").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建财务角色: %v", err)
	}

	dept, err := data.db.Organization.Create().
		SetCode("DEPT-" + suffix).
		SetName("财务部-" + suffix).
		SetKind("department").
		SetParentID(company.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建财务部: %v", err)
	}

	adminRepo := NewAdminRepo(data)

	// 1. 部门查询角色，应返回所属公司的角色
	deptRoles, err := adminRepo.ListRoles(ctx, dept.ID)
	if err != nil {
		t.Fatalf("部门查询角色失败: %v", err)
	}
	foundRole := false
	for _, r := range deptRoles {
		if r.ID == financeRole.ID {
			foundRole = true
			break
		}
	}
	if !foundRole {
		t.Fatalf("部门角色列表中未包含所属公司的财务角色")
	}

	// 2. 查询单个角色详情（GetRole）在部门上下文中亦可读取
	singleRole, err := adminRepo.GetRole(ctx, dept.ID, financeRole.ID)
	if err != nil || singleRole.ID != financeRole.ID {
		t.Fatalf("部门上下文读取单个角色失败: %v", err)
	}

	// 3. 提权画像校验（GetRolesPrivilegeProfiles）针对部门节点应成功返回所属公司角色画像
	profiles, err := adminRepo.GetRolesPrivilegeProfiles(ctx, dept.ID, []uuid.UUID{financeRole.ID})
	if err != nil || len(profiles) != 1 || profiles[0].ID != financeRole.ID {
		t.Fatalf("部门上下文获取角色画像失败: %v, profiles = %v", err, profiles)
	}

	// 4. 创建用户并将成员关系加入部门，分配财务角色
	userAccount, err := data.db.User.Create().
		SetDisplayName("财务专员-" + suffix).
		SetUsername("fin_user_" + suffix).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	membershipBiz, err := adminRepo.CreateUserMembership(ctx, &biz.AdminUserMembership{
		UserID:         userAccount.ID,
		OrganizationID: dept.ID,
		Primary:        true,
	}, []uuid.UUID{financeRole.ID}, adminLifecycleAudit(company.ID, "admin.user.membership.create"))
	if err != nil {
		t.Fatalf("在部门下创建成员关系并分配角色失败: %v", err)
	}
	if membershipBiz.OrganizationID != dept.ID {
		t.Fatalf("成员关系组织 ID 不匹配: got %v, want %v", membershipBiz.OrganizationID, dept.ID)
	}

	// 5. 验证操作人在公司工作台下能提取到该用户在部门内的有效角色画像
	actorProfiles, err := adminRepo.GetActorRolesPrivilegeProfiles(ctx, company.ID, userAccount.ID)
	if err != nil {
		t.Fatalf("在公司工作台提取部门成员角色画像失败: %v", err)
	}
	if len(actorProfiles) != 1 || actorProfiles[0].ID != financeRole.ID {
		t.Fatalf("提取的角色画像不符合预期: got %v, want roleID %v", actorProfiles, financeRole.ID)
	}
}
