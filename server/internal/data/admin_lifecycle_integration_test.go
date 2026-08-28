package data

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	backgroundtaskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	notificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/notificationdelivery"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	sessionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/session"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"

	"github.com/google/uuid"
)

func TestAdminEmployeeLifecyclePostgres(t *testing.T) {
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}
	ctx := context.Background()
	data, cleanup, err := NewData(&conf.Data{Database: &conf.Data_Database{
		Driver:      "postgres",
		Source:      source,
		AutoMigrate: true,
	}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("初始化集成测试数据库: %v", err)
	}
	defer cleanup()

	headquarters, err := data.db.Organization.Create().
		SetCode("HQ").
		SetName("总部").
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建总部: %v", err)
	}
	role, err := data.db.Role.Create().
		SetOrganizationID(headquarters.ID).
		SetCode("operator").
		SetName("操作员").
		SetDataScope("organization").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建角色: %v", err)
	}
	account, err := data.db.User.Create().
		SetDisplayName("返聘员工").
		SetDingtalkUnionid("union-id").
		SetDingtalkUserid("ding-user-id").
		SetDingtalkName("返聘员工").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建员工: %v", err)
	}
	membershipRecord, err := data.db.Membership.Create().
		SetUserID(account.ID).
		SetOrganizationID(headquarters.ID).
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
		SetOrganizationID(headquarters.ID).
		SetTokenHash("integration-session-token-hash").
		SetExpiresAt(time.Now().Add(time.Hour)).
		Save(ctx); err != nil {
		t.Fatalf("创建会话: %v", err)
	}

	adminRepo := &adminRepo{data: data}
	if err := adminRepo.DeleteUserMembership(ctx, account.ID, membershipRecord.ID); err != biz.ErrAdminUserLastMembership {
		t.Fatalf("移出最后组织 error = %v, want ErrAdminUserLastMembership", err)
	}
	if err := adminRepo.TerminateUser(ctx, headquarters.ID, account.ID); err != nil {
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
		UnionID: "union-id",
		UserID:  "ding-user-id",
		Name:    "返聘员工",
	})
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
	pendingUsername := "pending.user"
	if err := adminRepo.ResetUserPassword(ctx, headquarters.ID, account.ID, "pending-password-hash", &pendingUsername); err != biz.ErrAdminUserNotFound {
		t.Fatalf("待审批账号设置密码 error = %v, want ErrAdminUserNotFound", err)
	}
	if _, err := adminRepo.UpdateUser(ctx, headquarters.ID, account.ID, &biz.AdminUser{
		ID:          account.ID,
		DisplayName: "返聘员工",
		Enabled:     true,
	}, []uuid.UUID{role.ID}); err != biz.ErrAdminUserAuthorizationRequired {
		t.Fatalf("普通编辑绕过外部身份授权 error = %v, want ErrAdminUserAuthorizationRequired", err)
	}

	notification := biz.NewDingTalkUserAuthorizedNotification(account.ID)
	authorized, err := adminRepo.AuthorizeDingTalkUser(ctx, headquarters.ID, headquarters.ID, &biz.AdminUser{
		ID:          account.ID,
		DisplayName: "返聘员工",
	}, []uuid.UUID{role.ID}, notification)
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
	if err != nil || delivery.RecipientUserID != account.ID || delivery.Template != notificationent.TemplateUSER_AUTHORIZED || delivery.ResourceType != "USER" || delivery.ResourceID != account.ID {
		t.Fatalf("授权通知明细 = %#v, error = %v", delivery, err)
	}

	backupUsername := "backup.user"
	if err := adminRepo.ResetUserPassword(ctx, headquarters.ID, account.ID, "active-password-hash", &backupUsername); err != nil {
		t.Fatalf("为在职账号设置备用账密: %v", err)
	}
	activeAccount, err := data.db.User.Get(ctx, account.ID)
	if err != nil || activeAccount.Username != backupUsername || activeAccount.PasswordHash == nil || *activeAccount.PasswordHash != "active-password-hash" {
		t.Fatalf("备用账密保存结果 = %#v, error = %v", activeAccount, err)
	}
	actorRoles, err := adminRepo.GetActorRolesPrivilegeProfiles(ctx, headquarters.ID, account.ID)
	if err != nil || len(actorRoles) != 1 || actorRoles[0].Code != "operator" {
		t.Fatalf("在职账号角色能力 = %#v, error = %v", actorRoles, err)
	}

	company, err := data.db.Organization.Create().
		SetCode("COMPANY").
		SetName("分公司").
		SetKind("company").
		SetParentID(headquarters.ID).
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
	if err := adminRepo.DeleteUserMembership(ctx, account.ID, membershipRecord.ID); err != nil {
		t.Fatalf("移出总部: %v", err)
	}
	replacementUsername := "replacement.user"
	if err := adminRepo.ResetUserPassword(ctx, headquarters.ID, account.ID, "replacement-password-hash", &replacementUsername); err != biz.ErrAdminUserNotFound {
		t.Fatalf("历史组织重置全局密码 error = %v, want ErrAdminUserNotFound", err)
	}
	retainedAccount, err := data.db.User.Get(ctx, account.ID)
	if err != nil || !retainedAccount.Enabled || retainedAccount.Username != backupUsername || retainedAccount.PasswordHash == nil || *retainedAccount.PasswordHash != "active-password-hash" {
		t.Fatalf("移出组织后的账号结果 = %#v, error = %v", retainedAccount, err)
	}
	if _, err := adminRepo.GetActorRolesPrivilegeProfiles(ctx, headquarters.ID, account.ID); err != biz.ErrAdminPrivilegeEscalation {
		t.Fatalf("历史组织读取操作者能力 error = %v, want ErrAdminPrivilegeEscalation", err)
	}
}
