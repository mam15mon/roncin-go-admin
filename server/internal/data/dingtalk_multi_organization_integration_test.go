package data

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/notificationdelivery"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
)

// 正式迁移隔离 Schema 验证多公司开通、人员独立提交、返聘及任意写入失败的整人回滚。
func TestDingTalkMultiOrganizationAssignmentPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	sys, err := data.db.Organization.Create().SetCode("multi-system").SetName("系统管理").SetKind("system").SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	first, err := data.db.Organization.Create().SetCode("multi-a").SetName("北京公司").SetKind("company").SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	second, err := data.db.Organization.Create().SetCode("multi-b").SetName("上海公司").SetKind("company").SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	deptA, err := data.db.Organization.Create().SetCode("multi-a-1").SetName("北京业务部").SetKind("department").SetParentID(first.ID).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	deptB, err := data.db.Organization.Create().SetCode("multi-a-2").SetName("北京财务部").SetKind("department").SetParentID(first.ID).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	roleA, err := data.db.Role.Create().SetOrganizationID(first.ID).SetCode("staff").SetName("员工").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	roleB, err := data.db.Role.Create().SetOrganizationID(second.ID).SetCode("staff").SetName("员工").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ctx, actor := directoryAdministrator(t, data, sys.ID)
	uc := biz.NewAdminUsecase(&adminRepo{data: data}, data)
	matrix := []biz.AdminUserMembershipCreation{{OrganizationID: deptA.ID, RoleIDs: []uuid.UUID{roleA.ID}}, {OrganizationID: deptB.ID}, {OrganizationID: second.ID, RoleIDs: []uuid.UUID{roleB.ID}}}
	newPerson := func(t *testing.T) *ent.User {
		t.Helper()
		identity := uuid.NewString()
		p, err := data.db.User.Create().SetDisplayName("多组织员工").SetDingtalkUnionid("union-" + identity).SetDingtalkUserid("user-" + identity).SetDingtalkLastSyncedAt(time.Now()).SetPendingAssignment(true).SetEnabled(false).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	assertAssignments := func(t *testing.T, userID uuid.UUID) {
		t.Helper()
		user, err := data.db.User.Get(ctx, userID)
		if err != nil || !user.Enabled || user.PendingAssignment {
			t.Fatalf("账号未完整开通:%+v %v", user, err)
		}
		members, err := data.db.Membership.Query().Where(membership.UserIDEQ(userID), membership.EnabledEQ(true)).WithRoleAssignments().All(ctx)
		if err != nil || len(members) != len(matrix) {
			t.Fatalf("组织数量:%+v %v", members, err)
		}
		for _, item := range matrix {
			found := false
			for _, m := range members {
				if m.OrganizationID != item.OrganizationID {
					continue
				}
				found = true
				if len(m.Edges.RoleAssignments) != len(item.RoleIDs) {
					t.Fatalf("组织%s角色串用:%+v", item.OrganizationID, m.Edges.RoleAssignments)
				}
				for _, r := range m.Edges.RoleAssignments {
					if len(item.RoleIDs) != 1 || r.RoleID != item.RoleIDs[0] {
						t.Fatalf("公司角色不匹配:%+v", r)
					}
				}
			}
			if !found {
				t.Fatalf("缺少组织%s", item.OrganizationID)
			}
		}
	}
	assertPending := func(t *testing.T, userID uuid.UUID) {
		t.Helper()
		user, err := data.db.User.Get(ctx, userID)
		if err != nil || user.Enabled || !user.PendingAssignment {
			t.Fatalf("失败改变账号:%+v %v", user, err)
		}
		if n, err := data.db.Membership.Query().Where(membership.UserIDEQ(userID)).Count(ctx); err != nil || n != 0 {
			t.Fatalf("失败残留成员:%d %v", n, err)
		}
		if n, err := data.db.NotificationDelivery.Query().Where(notificationdelivery.RecipientUserIDEQ(userID)).Count(ctx); err != nil || n != 0 {
			t.Fatalf("失败残留通知:%d %v", n, err)
		}
		var audits int
		if err := data.sqlDB.QueryRowContext(ctx, "SELECT count(*) FROM audit_logs WHERE action = 'admin.user.dingtalk.authorize' AND details->>'resource_id' = $1", userID.String()).Scan(&audits); err != nil || audits != 0 {
			t.Fatalf("失败残留审计:%d %v", audits, err)
		}
	}
	t.Run("多公司同公司多部门与选填角色", func(t *testing.T) {
		p := newPerson(t)
		if _, err := uc.AuthorizeDingTalkUser(ctx, sys.ID, actor, &biz.AdminUser{ID: p.ID}, matrix); err != nil {
			t.Fatal(err)
		}
		assertAssignments(t, p.ID)
		var audits int
		if err := data.sqlDB.QueryRowContext(ctx, "SELECT count(*) FROM audit_logs WHERE action = 'admin.user.dingtalk.authorize' AND details->>'resource_id' = $1", p.ID.String()).Scan(&audits); err != nil || audits != 1 {
			t.Fatalf("成功必须有一条开通审计:%d %v", audits, err)
		}
		deliveries, err := data.db.NotificationDelivery.Query().Where(notificationdelivery.RecipientUserIDEQ(p.ID)).WithBackgroundTask().All(ctx)
		if err != nil || len(deliveries) != 1 || deliveries[0].Edges.BackgroundTask.OrganizationID != first.ID {
			t.Fatalf("一人一通知及任务归属:%+v %v", deliveries, err)
		}
	})
	t.Run("并发第二次不覆盖", func(t *testing.T) {
		p := newPerson(t)
		var wg sync.WaitGroup
		errs := make(chan error, 2)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := uc.AuthorizeDingTalkUser(ctx, sys.ID, actor, &biz.AdminUser{ID: p.ID}, matrix)
				errs <- err
			}()
		}
		wg.Wait()
		close(errs)
		successes, conflicts := 0, 0
		for err := range errs {
			if err == nil {
				successes++
			} else if err == biz.ErrDingTalkAssignmentConflict {
				conflicts++
			} else {
				t.Fatal(err)
			}
		}
		if successes != 1 || conflicts != 1 {
			t.Fatalf("并发结果%d成功%d冲突", successes, conflicts)
		}
		assertAssignments(t, p.ID)
		if n, err := data.db.NotificationDelivery.Query().Where(notificationdelivery.RecipientUserIDEQ(p.ID)).Count(ctx); err != nil || n != 1 {
			t.Fatalf("重复通知%d %v", n, err)
		}
	})
	t.Run("后续无效目标整人拒绝", func(t *testing.T) {
		cases := []struct {
			name     string
			later    biz.AdminUserMembershipCreation
			expected error
		}{
			{"不存在组织", biz.AdminUserMembershipCreation{OrganizationID: uuid.New()}, biz.ErrAdminOrganizationNotFound},
			{"系统工作台", biz.AdminUserMembershipCreation{OrganizationID: sys.ID}, biz.ErrAdminOrganizationNotFound},
			{"跨公司同名角色", biz.AdminUserMembershipCreation{OrganizationID: second.ID, RoleIDs: []uuid.UUID{roleA.ID}}, biz.ErrAdminRoleNotFound},
			{"不存在角色", biz.AdminUserMembershipCreation{OrganizationID: second.ID, RoleIDs: []uuid.UUID{uuid.New()}}, biz.ErrAdminRoleNotFound},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				p := newPerson(t)
				_, err := uc.AuthorizeDingTalkUser(ctx, sys.ID, actor, &biz.AdminUser{ID: p.ID}, []biz.AdminUserMembershipCreation{matrix[0], tc.later})
				if err != tc.expected {
					t.Fatalf("预期%v得到%v", tc.expected, err)
				}
				assertPending(t, p.ID)
			})
		}
		p := newPerson(t)
		if _, err := data.db.Organization.UpdateOne(first).SetEnabled(false).Save(ctx); err != nil {
			t.Fatal(err)
		}
		_, err := uc.AuthorizeDingTalkUser(ctx, sys.ID, actor, &biz.AdminUser{ID: p.ID}, []biz.AdminUserMembershipCreation{matrix[2], matrix[1]})
		if err != biz.ErrAdminOrganizationNotFound {
			t.Fatalf("停用父公司未拒绝:%v", err)
		}
		assertPending(t, p.ID)
		if _, err := data.db.Organization.UpdateOne(first).SetEnabled(true).Save(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := data.db.Role.UpdateOne(roleB).SetEnabled(false).Save(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := uc.AuthorizeDingTalkUser(ctx, sys.ID, actor, &biz.AdminUser{ID: p.ID}, matrix); err != biz.ErrAdminRoleNotFound {
			t.Fatalf("停用后续角色:%v", err)
		}
		assertPending(t, p.ID)
		if _, err := data.db.Role.UpdateOne(roleB).SetEnabled(true).Save(ctx); err != nil {
			t.Fatal(err)
		}
	})
	// 用真实数据库触发器在后续落库点抛错，验证先前写入不是独立提交。
	t.Run("中途写入审计通知失败回滚", func(t *testing.T) {
		for _, tc := range []struct{ name, table, condition string }{
			{"第二成员", "memberships", fmt.Sprintf("NEW.organization_id = '%s'::uuid", deptB.ID)},
			{"第二角色", "role_assignments", fmt.Sprintf("NEW.role_id = '%s'::uuid", roleB.ID)},
			{"通知任务", "background_tasks", "NEW.kind = 'DINGTALK_NOTIFICATION'"},
			{"通知投递", "notification_deliveries", "NEW.template = 'USER_AUTHORIZED'"},
			{"审计", "audit_logs", "NEW.action = 'admin.user.dingtalk.authorize'"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				p := newPerson(t)
				function := `CREATE FUNCTION fail_multi_assignment() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF ` + tc.condition + ` THEN RAISE EXCEPTION 'test multi assignment failure'; END IF; RETURN NEW; END $$`
				if _, err := data.sqlDB.ExecContext(ctx, function); err != nil {
					t.Fatal(err)
				}
				if _, err := data.sqlDB.ExecContext(ctx, "CREATE TRIGGER fail_multi_assignment BEFORE INSERT ON "+tc.table+" FOR EACH ROW EXECUTE FUNCTION fail_multi_assignment()"); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if _, err := data.sqlDB.ExecContext(ctx, "DROP TRIGGER fail_multi_assignment ON "+tc.table); err != nil {
						t.Error(err)
					}
					if _, err := data.sqlDB.ExecContext(ctx, "DROP FUNCTION fail_multi_assignment()"); err != nil {
						t.Error(err)
					}
				})
				before, err := data.db.BackgroundTask.Query().Where(backgroundtask.KindEQ(backgroundtask.KindDINGTALK_NOTIFICATION)).Count(ctx)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := uc.AuthorizeDingTalkUser(ctx, sys.ID, actor, &biz.AdminUser{ID: p.ID}, matrix); err == nil {
					t.Fatal("失败注入未生效")
				}
				assertPending(t, p.ID)
				if after, err := data.db.BackgroundTask.Query().Where(backgroundtask.KindEQ(backgroundtask.KindDINGTALK_NOTIFICATION)).Count(ctx); err != nil || after != before {
					t.Fatalf("失败残留任务:之前%d之后%d %v", before, after, err)
				}
			})
		}
	})
	t.Run("批量保留成功只重试失败", func(t *testing.T) {
		success, failure := newPerson(t), newPerson(t)
		if _, err := data.db.User.UpdateOne(failure).SetEnabled(true).SetPendingAssignment(false).Save(ctx); err != nil {
			t.Fatal(err)
		}
		results, err := uc.BatchAuthorizeDingTalkUsers(ctx, sys.ID, actor, []uuid.UUID{success.ID, failure.ID}, matrix)
		if err != nil || len(results) != 2 || !results[0].Success || results[1].Success {
			t.Fatalf("人员独立结果:%+v %v", results, err)
		}
		assertAssignments(t, success.ID)
		if _, err := data.db.User.UpdateOne(failure).SetEnabled(false).SetPendingAssignment(true).Save(ctx); err != nil {
			t.Fatal(err)
		}
		results, err = uc.BatchAuthorizeDingTalkUsers(ctx, sys.ID, actor, []uuid.UUID{failure.ID}, matrix)
		if err != nil || len(results) != 1 || !results[0].Success {
			t.Fatalf("只重试失败:%+v %v", results, err)
		}
		assertAssignments(t, failure.ID)
		if n, err := data.db.NotificationDelivery.Query().Where(notificationdelivery.RecipientUserIDIn(success.ID, failure.ID)).Count(ctx); err != nil || n != 2 {
			t.Fatalf("重试重复通知:%d %v", n, err)
		}
	})
	t.Run("返聘失败保留旧关系成功只恢复所选", func(t *testing.T) {
		p := newPerson(t)
		if _, err := data.db.User.UpdateOne(p).SetPendingAssignment(false).Save(ctx); err != nil {
			t.Fatal(err)
		}
		old, err := data.db.Membership.Create().SetUserID(p.ID).SetOrganizationID(first.ID).SetEnabled(false).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		selected, err := data.db.Membership.Create().SetUserID(p.ID).SetOrganizationID(deptA.ID).SetEnabled(false).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		binding, err := data.db.RoleAssignment.Create().SetMembershipID(old.ID).SetRoleID(roleA.ID).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		badAudit := &biz.AuditEvent{OrganizationID: &sys.ID, UserID: &actor, Action: "", Result: "success"}
		if _, err := (&adminRepo{data: data}).AuthorizeDingTalkUser(ctx, sys.ID, &biz.AdminUser{ID: p.ID}, matrix, biz.NewDingTalkUserAuthorizedNotification(p.ID), badAudit); err == nil {
			t.Fatal("审计失败未生效")
		}
		if user, err := data.db.User.Get(ctx, p.ID); err != nil || user.Enabled || user.PendingAssignment {
			t.Fatalf("返聘失败改变状态:%+v %v", user, err)
		}
		if n, err := data.db.Membership.Query().Where(membership.UserIDEQ(p.ID)).Count(ctx); err != nil || n != 2 {
			t.Fatalf("返聘失败残留新关系:%d %v", n, err)
		}
		for _, id := range []uuid.UUID{old.ID, selected.ID} {
			if m, err := data.db.Membership.Get(ctx, id); err != nil || m.Enabled {
				t.Fatalf("旧成员状态改变:%+v %v", m, err)
			}
		}
		if exists, err := data.db.RoleAssignment.Query().Where(roleassignment.IDEQ(binding.ID)).Exist(ctx); err != nil || !exists {
			t.Fatalf("返聘失败清了旧授权:%v %v", exists, err)
		}
		if n, err := data.db.NotificationDelivery.Query().Where(notificationdelivery.RecipientUserIDEQ(p.ID)).Count(ctx); err != nil || n != 0 {
			t.Fatalf("返聘失败残留通知:%d %v", n, err)
		}
		if _, err := uc.AuthorizeDingTalkUser(ctx, sys.ID, actor, &biz.AdminUser{ID: p.ID}, matrix); err != nil {
			t.Fatal(err)
		}
		assertAssignments(t, p.ID)
		if m, err := data.db.Membership.Get(ctx, selected.ID); err != nil || !m.Enabled {
			t.Fatalf("未恢复原所选关系:%+v %v", m, err)
		}
		if m, err := data.db.Membership.Get(ctx, old.ID); err != nil || m.Enabled {
			t.Fatalf("自动恢复未选组织:%+v %v", m, err)
		}
		if n, err := data.db.RoleAssignment.Query().Where(roleassignment.MembershipIDEQ(old.ID)).Count(ctx); err != nil || n != 0 {
			t.Fatalf("未清旧组织授权:%d %v", n, err)
		}
	})
	t.Run("撤销操作者权限后陈旧Principal不能开通", func(t *testing.T) {
		p := newPerson(t)
		permission, err := data.db.Permission.Query().Where(permission.KeyEQ(access.UserAuthorizeDingTalk)).Only(ctx)
		if err != nil {
			t.Fatal(err)
		}
		actorRole, err := data.db.Role.Query().Where(role.OrganizationIDEQ(sys.ID), role.CodeEQ("administrator")).Only(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := data.db.Role.UpdateOne(actorRole).RemovePermissionIDs(permission.ID).Save(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := uc.AuthorizeDingTalkUser(ctx, sys.ID, actor, &biz.AdminUser{ID: p.ID}, matrix); err != biz.ErrPermissionDenied {
			t.Fatalf("锁内未复核撤权:%v", err)
		}
		assertPending(t, p.ID)
	})
}
