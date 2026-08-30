package data

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	backgroundtaskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	masterdataent "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	notificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/notificationdelivery"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
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
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	dingUnionID := "union-" + suffix
	dingUserID := "ding-user-" + suffix
	data, cleanup, err := NewData(&conf.Data{Database: &conf.Data_Database{
		Driver:      "postgres",
		Source:      source,
		AutoMigrate: true,
	}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("初始化集成测试数据库: %v", err)
	}
	// 关库注册为最早的 t.Cleanup（LIFO 中最后执行），保证数据清理先于连接关闭。
	t.Cleanup(cleanup)

	headquarters, err := data.db.Organization.Create().
		SetCode("HQ-" + suffix).
		SetName("总部-" + suffix).
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建总部: %v", err)
	}
	t.Cleanup(func() { cleanupAdminLifecycleFixture(t, data, headquarters.ID, nil) })
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
		SetDisplayName("返聘员工-" + suffix).
		SetDingtalkUnionid(dingUnionID).
		SetDingtalkUserid(dingUserID).
		SetDingtalkName("返聘员工-" + suffix).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建员工: %v", err)
	}
	t.Cleanup(func() { cleanupAdminLifecycleFixture(t, data, headquarters.ID, &account.ID) })
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
		SetTokenHash("integration-session-" + suffix).
		SetExpiresAt(time.Now().Add(time.Hour)).
		Save(ctx); err != nil {
		t.Fatalf("创建会话: %v", err)
	}

	adminRepo := &adminRepo{data: data}
	if err := adminRepo.DeleteUserMembership(ctx, account.ID, membershipRecord.ID, adminLifecycleAudit("admin.user.membership.delete")); err != biz.ErrAdminUserLastMembership {
		t.Fatalf("移出最后组织 error = %v, want ErrAdminUserLastMembership", err)
	}
	if err := adminRepo.TerminateUser(ctx, headquarters.ID, account.ID, adminLifecycleAudit("admin.user.terminate")); err != nil {
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
	}, &biz.AuditEvent{Action: "auth.dingtalk.register", Result: "success"})
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
	if err := adminRepo.ResetUserPassword(ctx, headquarters.ID, account.ID, "pending-password-hash", &pendingUsername, adminLifecycleAudit("admin.user.password.reset")); err != biz.ErrAdminUserNotFound {
		t.Fatalf("待审批账号设置密码 error = %v, want ErrAdminUserNotFound", err)
	}
	if _, err := adminRepo.UpdateUser(ctx, headquarters.ID, account.ID, &biz.AdminUser{
		ID:          account.ID,
		DisplayName: "返聘员工",
		Enabled:     true,
	}, []uuid.UUID{role.ID}, adminLifecycleAudit("admin.user.update")); err != biz.ErrAdminUserAuthorizationRequired {
		t.Fatalf("普通编辑绕过外部身份授权 error = %v, want ErrAdminUserAuthorizationRequired", err)
	}

	notification := biz.NewDingTalkUserAuthorizedNotification(account.ID)
	authorized, err := adminRepo.AuthorizeDingTalkUser(ctx, headquarters.ID, headquarters.ID, &biz.AdminUser{
		ID:          account.ID,
		DisplayName: "返聘员工",
	}, []uuid.UUID{role.ID}, notification, adminLifecycleAudit("admin.user.dingtalk.authorize"))
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
	if err := adminRepo.ResetUserPassword(ctx, headquarters.ID, account.ID, "active-password-hash", &backupUsername, adminLifecycleAudit("admin.user.password.reset")); err != nil {
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
		SetCode("COMPANY-" + suffix).
		SetName("分公司-" + suffix).
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
	if err := adminRepo.DeleteUserMembership(ctx, account.ID, membershipRecord.ID, adminLifecycleAudit("admin.user.membership.delete")); err != nil {
		t.Fatalf("移出总部: %v", err)
	}
	replacementUsername := "replacement." + suffix
	if err := adminRepo.ResetUserPassword(ctx, headquarters.ID, account.ID, "replacement-password-hash", &replacementUsername, adminLifecycleAudit("admin.user.password.reset")); err != biz.ErrAdminUserNotFound {
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

func adminLifecycleAudit(action string) *biz.AuditEvent {
	return &biz.AuditEvent{Action: action, Result: "success", Details: map[string]string{}}
}

// cleanupAdminLifecycleFixture 按外键依赖顺序清理生命周期测试写入的全部数据；
// accountID 为空表示员工尚未创建，只清理组织级数据。
func cleanupAdminLifecycleFixture(t *testing.T, data *Data, organizationID uuid.UUID, accountID *uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	steps := []struct {
		name string
		run  func() error
	}{
		{name: "通知明细", run: func() error {
			_, err := data.db.NotificationDelivery.Delete().Where(notificationent.HasBackgroundTaskWith(backgroundtaskent.OrganizationIDEQ(organizationID))).Exec(ctx)
			return err
		}},
		{name: "后台任务", run: func() error {
			_, err := data.db.BackgroundTask.Delete().Where(backgroundtaskent.OrganizationIDEQ(organizationID)).Exec(ctx)
			return err
		}},
		{name: "审计日志", run: func() error {
			_, err := data.db.AuditLog.Delete().Where(auditlogent.OrganizationIDEQ(organizationID)).Exec(ctx)
			return err
		}},
	}
	if accountID != nil {
		steps = append(steps,
			struct {
				name string
				run  func() error
			}{name: "会话", run: func() error {
				_, err := data.db.Session.Delete().Where(sessionent.UserIDEQ(*accountID)).Exec(ctx)
				return err
			}},
			struct {
				name string
				run  func() error
			}{name: "角色分配", run: func() error {
				_, err := data.db.RoleAssignment.Delete().Where(roleassignment.HasMembershipWith(membership.UserIDEQ(*accountID))).Exec(ctx)
				return err
			}},
			struct {
				name string
				run  func() error
			}{name: "组织关系", run: func() error {
				_, err := data.db.Membership.Delete().Where(membership.UserIDEQ(*accountID)).Exec(ctx)
				return err
			}},
			struct {
				name string
				run  func() error
			}{name: "账号", run: func() error {
				return data.db.User.DeleteOneID(*accountID).Exec(ctx)
			}},
		)
	}
	// 子组织依赖成员关系先删，总部最后删；重复清理时组织已不存在视为成功。
	steps = append(steps,
		struct {
			name string
			run  func() error
		}{name: "角色", run: func() error {
			_, err := data.db.Role.Delete().Where(roleent.OrganizationIDEQ(organizationID)).Exec(ctx)
			return err
		}},
		struct {
			name string
			run  func() error
		}{name: "主数据", run: func() error {
			_, err := data.db.MasterDataItem.Delete().Where(masterdataent.OrganizationIDEQ(organizationID)).Exec(ctx)
			return err
		}},
		struct {
			name string
			run  func() error
		}{name: "子组织", run: func() error {
			_, err := data.db.Organization.Delete().Where(organizationent.ParentIDEQ(organizationID)).Exec(ctx)
			return err
		}},
		struct {
			name string
			run  func() error
		}{name: "总部", run: func() error {
			err := data.db.Organization.DeleteOneID(organizationID).Exec(ctx)
			if err != nil && !ent.IsNotFound(err) {
				return err
			}
			return nil
		}},
	)
	for _, step := range steps {
		if err := step.run(); err != nil {
			t.Errorf("清理%s失败: %v", step.name, err)
			return
		}
	}
}
