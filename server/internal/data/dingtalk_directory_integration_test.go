package data

import (
	"context"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/dingtalkdirectorysync"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	"sync"
	"testing"
	"time"
)

func directoryAdministrator(t *testing.T, data *Data, sys uuid.UUID) (context.Context, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	actor, err := data.db.User.Create().SetDisplayName("通讯录管理员").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	member, err := data.db.Membership.Create().SetUserID(actor.ID).SetOrganizationID(sys).SetEnabled(true).SetPrimary(true).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	permission, err := data.db.Permission.Create().SetKey(access.UserAuthorizeDingTalk).SetName("人员开通").SetGroup("系统").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	syncPermission, err := data.db.Permission.Create().SetKey(access.UserDingTalkDirectorySync).SetName("人员同步").SetGroup("系统").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	r, err := data.db.Role.Create().SetOrganizationID(sys).SetCode("administrator").SetName("系统管理员").SetDataScope(role.DataScopeAll).AddPermissionIDs(permission.ID, syncPermission.ID).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = data.db.RoleAssignment.Create().SetMembershipID(member.ID).SetRoleID(r.ID).Save(ctx); err != nil {
		t.Fatal(err)
	}
	principal, err := (&authRepo{data: data}).ResolvePrincipal(ctx, actor.ID, sys)
	if err != nil {
		t.Fatal(err)
	}
	return biz.WithPrincipal(ctx, principal), actor.ID
}
func TestDingTalkDirectorySyncPostgresIdentityLeaseAndAvailability(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	org, err := data.db.Organization.Create().SetCode("directory-system").SetName("系统管理").SetKind("system").SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ctx, actor := directoryAdministrator(t, data, org.ID)
	repo := &dingTalkDirectoryRepo{data: data}
	syncRecord, err := repo.Start(ctx, org.ID, uuid.New(), "test-corp")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Start(ctx, org.ID, uuid.New(), "test-corp"); err != biz.ErrDingTalkDirectoryBusy {
		t.Fatalf("并发重复触发应冲突: %v", err)
	}
	tasks := &backgroundTaskRepo{data: data}
	task, err := tasks.ClaimAny(ctx, []biz.BackgroundTaskKind{biz.BackgroundTaskKindDingTalkDirectorySync}, time.Minute, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	mobile, email, job := "13800000000", "private@example.test", "001"
	person := &biz.DingTalkDirectoryPerson{UserID: "u1", UnionID: "union1", Name: "员工甲", Mobile: &mobile, Email: &email, JobNumber: &job, Departments: []biz.DingTalkDepartment{{ID: 1, Name: "企业"}}}
	if outcome, err := repo.ApplyPerson(ctx, task, person, []string{"display_name", "departments", "job_number"}); err != nil || outcome != "created" {
		t.Fatalf("新人员: %s %v", outcome, err)
	}
	pending, err := (&adminRepo{data: data}).ListUsers(ctx, org.ID, biz.AdminUserListOptions{Page: 1, PageSize: 20, Status: biz.AdminUserStatusPendingAssignment})
	if err != nil || len(pending.Items) != 1 {
		t.Fatalf("无组织用户查询: %+v %v", pending, err)
	}
	account, err := data.db.User.Get(ctx, pending.Items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if account.Enabled || !account.PendingAssignment || account.Email != nil || account.DingtalkMobile != nil || account.DingtalkJobNumber == nil {
		t.Fatal("资料授权与待分配状态错误")
	}
	if count, err := data.db.Membership.Query().Where(membership.UserIDEQ(account.ID)).Count(ctx); err != nil || count != 0 {
		t.Fatal("同步不能建立组织成员关系")
	}
	if outcome, err := repo.ApplyPerson(ctx, task, person, []string{"display_name", "departments", "job_number"}); err != nil || outcome != "unchanged" {
		t.Fatalf("重复同步应幂等: %s %v", outcome, err)
	}
	other, err := data.db.User.Create().SetDisplayName("冲突账号").SetDingtalkUnionid("union2").SetDingtalkUserid("u2").SetEnabled(false).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_ = other
	conflict := *person
	conflict.UnionID = "union2"
	if _, err := repo.ApplyPerson(ctx, task, &conflict, []string{"display_name"}); err != biz.ErrDingTalkIdentityConflict {
		t.Fatalf("两个身份对应不同人员必须冲突: %v", err)
	}
	if err := repo.Finish(ctx, task, &biz.DingTalkDirectorySync{TotalCount: 1, CreatedCount: 1}, ""); err != nil {
		t.Fatal(err)
	}
	stale := *task
	if _, err := repo.ApplyPerson(ctx, &stale, person, []string{"display_name"}); err != biz.ErrBackgroundTaskLeaseMismatch {
		t.Fatalf("终态旧租约仍可写入: %v", err)
	}
	latest, err := repo.Get(ctx, org.ID, syncRecord.TaskID, 1, 20)
	if err != nil || latest.Status != biz.BackgroundTaskStatusSucceeded || latest.CreatedCount != 1 {
		t.Fatalf("同步调度/结果未原子完成: %+v %v", latest, err)
	}
	next, err := repo.Start(ctx, org.ID, uuid.New(), "test-corp")
	if err != nil {
		t.Fatal(err)
	}
	expired, err := tasks.ClaimAny(ctx, []biz.BackgroundTaskKind{biz.BackgroundTaskKindDingTalkDirectorySync}, time.Second, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := data.db.BackgroundTask.UpdateOneID(next.TaskID).SetLeaseExpiresAt(time.Now().Add(-time.Second)).Save(ctx); err != nil {
		t.Fatal(err)
	}
	if expired.ID != next.TaskID {
		t.Fatal("错误任务")
	}
	if _, err = tasks.ClaimAny(ctx, []biz.BackgroundTaskKind{biz.BackgroundTaskKindDingTalkDirectorySync}, time.Minute, time.Now()); err != biz.ErrBackgroundTaskNoTask {
		t.Fatalf("过期转终态: %v", err)
	}
	detail, err := data.db.DingTalkDirectorySync.Query().Where(dingtalkdirectorysync.BackgroundTaskIDEQ(next.TaskID)).Only(ctx)
	if err != nil || detail.Active {
		t.Fatal("过期终态未释放企业活动资格")
	}
	if _, err = repo.Start(ctx, org.ID, uuid.New(), "test-corp"); err != nil {
		t.Fatal("过期任务永久阻塞后续同步", err)
	}
	if _, err = tasks.Requeue(ctx, org.ID, next.TaskID, time.Now(), &biz.AuditEvent{UserID: &actor, Action: "test.requeue", Result: "success"}); err != biz.ErrDingTalkDirectoryBusy {
		t.Fatalf("人工重试必须重新竞争活动唯一资格: %v", err)
	}
}
func TestDingTalkDirectoryAssignmentPostgresConcurrencyAndPartialSuccess(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	sys, err := data.db.Organization.Create().SetCode("assign-sys").SetName("系统管理").SetKind("system").SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	company, err := data.db.Organization.Create().SetCode("assign-company").SetName("公司").SetKind("company").SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	department, err := data.db.Organization.Create().SetCode("assign-dept").SetName("业务部门").SetKind("department").SetParentID(company.ID).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	r, err := data.db.Role.Create().SetOrganizationID(company.ID).SetCode("staff").SetName("员工").SetDataScope(role.DataScopeOrganization).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ctx, actor := directoryAdministrator(t, data, sys.ID)
	person, err := data.db.User.Create().SetDisplayName("待分配人员").SetDingtalkUnionid("union-a").SetDingtalkUserid("user-a").SetPendingAssignment(true).SetEnabled(false).SetDingtalkLastSyncedAt(time.Now()).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	uc := biz.NewAdminUsecase(&adminRepo{data: data})
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := uc.AuthorizeDingTalkUser(ctx, sys.ID, department.ID, actor, &biz.AdminUser{ID: person.ID}, []uuid.UUID{r.ID})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	success, conflicts := 0, 0
	for err := range errs {
		if err == nil {
			success++
		} else if err == biz.ErrDingTalkAssignmentConflict {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatalf("两管理员开通 %d成功 %d冲突", success, conflicts)
	}
	if credential, err := (&authRepo{data: data}).FindDingTalkCredential(ctx, &biz.DingTalkIdentity{UnionID: "union-a", UserID: "user-a", Name: "扫码昵称"}); err != nil || credential.PrimaryOrganizationID != company.ID || credential.DisplayName != "待分配人员" {
		t.Fatalf("默认公司和姓名来源: %+v %v", credential, err)
	}

	// 9名待分配 + 1名已开通，合法批量保留9成功并明确1冲突。
	userIDs := []uuid.UUID{person.ID}
	for i := 0; i < 9; i++ {
		identity := uuid.NewString()
		pending, err := data.db.User.Create().SetDisplayName("批量人员").SetDingtalkUnionid("union-" + identity).SetDingtalkUserid("user-" + identity).SetPendingAssignment(true).SetEnabled(false).SetDingtalkLastSyncedAt(time.Now()).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		userIDs = append(userIDs, pending.ID)
	}
	results, err := uc.BatchAuthorizeDingTalkUsers(ctx, sys.ID, department.ID, actor, userIDs, []uuid.UUID{r.ID})
	if err != nil || len(results) != 10 {
		t.Fatalf("批量: %+v %v", results, err)
	}
	successes := 0
	for _, item := range results {
		if item.Success {
			successes++
		}
	}
	if successes != 9 || results[0].Success {
		t.Fatalf("批量应9成功1失败: %+v", results)
	}
	// 目标组织停用与角色停用都在事务内拒绝，账号状态及成员关系原子保留。
	invalidPerson, err := data.db.User.Create().SetDisplayName("回滚人员").SetDingtalkUnionid("rollback-union").SetDingtalkUserid("rollback-user").SetPendingAssignment(true).SetEnabled(false).SetDingtalkLastSyncedAt(time.Now()).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = data.db.Role.UpdateOne(r).SetEnabled(false).Save(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.AuthorizeDingTalkUser(ctx, sys.ID, department.ID, actor, &biz.AdminUser{ID: invalidPerson.ID}, []uuid.UUID{r.ID}); err != biz.ErrAdminRoleNotFound {
		t.Fatalf("停用角色未拒绝: %v", err)
	}
	if _, err = data.db.Role.UpdateOne(r).SetEnabled(true).Save(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = data.db.Organization.UpdateOne(department).SetEnabled(false).Save(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.AuthorizeDingTalkUser(ctx, sys.ID, department.ID, actor, &biz.AdminUser{ID: invalidPerson.ID}, []uuid.UUID{r.ID}); err != biz.ErrAdminOrganizationNotFound {
		t.Fatalf("停用组织未拒绝: %v", err)
	}
	if _, err = data.db.Organization.UpdateOne(department).SetEnabled(true).Save(ctx); err != nil {
		t.Fatal(err)
	}
	reloaded, err := data.db.User.Get(ctx, invalidPerson.ID)
	if err != nil || reloaded.Enabled || !reloaded.PendingAssignment {
		t.Fatalf("失败开通修改了账号: %v", err)
	}
	if n, err := data.db.Membership.Query().Where(membership.UserIDEQ(invalidPerson.ID)).Count(ctx); err != nil || n != 0 {
		t.Fatalf("失败开通残留成员关系: %d %v", n, err)
	}
	if _, err := uc.UpdateUser(ctx, sys.ID, actor, person.ID, &biz.AdminUser{ID: person.ID, DisplayName: "擅自改名", Enabled: true}, []uuid.UUID{r.ID}); err != biz.ErrDingTalkProfileReadOnly {
		t.Fatalf("资料只读未生效: %v", err)
	}
	if _, err := uc.UpdateUser(ctx, sys.ID, actor, person.ID, &biz.AdminUser{ID: person.ID, Enabled: true}, []uuid.UUID{r.ID}); err != nil {
		t.Fatalf("基础资料只读误伤角色编辑: %v", err)
	}
	if err := uc.TerminateUser(ctx, sys.ID, actor, person.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.AuthorizeDingTalkUser(ctx, sys.ID, department.ID, actor, &biz.AdminUser{ID: person.ID}, []uuid.UUID{r.ID}); err != nil {
		t.Fatalf("人工返聘: %v", err)
	}
	if count, err := data.db.RoleAssignment.Query().Where(roleassignment.HasMembershipWith(membership.UserIDEQ(person.ID))).Count(ctx); err != nil || count != 1 {
		t.Fatalf("原账号返聘未收敛角色: %d %v", count, err)
	}
	if n, err := data.db.BackgroundTask.Query().Where(backgroundtask.KindEQ(backgroundtask.KindDINGTALK_NOTIFICATION)).Count(ctx); err != nil || n != 11 {
		t.Fatalf("每次成功开通仅一通知: %d %v", n, err)
	}
}

// 人工重试会重新发放租约；旧执行器迟到的写入和完成均不能覆盖新一轮结果。
func TestDingTalkDirectoryOldLeaseCannotWriteAfterManualRetryPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	org, err := data.db.Organization.Create().SetCode("retry-system").SetName("系统管理").SetKind("system").SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ctx, actor := directoryAdministrator(t, data, org.ID)
	repo := &dingTalkDirectoryRepo{data: data}
	tasks := &backgroundTaskRepo{data: data}
	record, err := repo.Start(ctx, org.ID, actor, "retry-corp")
	if err != nil {
		t.Fatal(err)
	}
	old, err := tasks.ClaimAny(ctx, []biz.BackgroundTaskKind{biz.BackgroundTaskKindDingTalkDirectorySync}, time.Minute, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := data.db.BackgroundTask.UpdateOneID(record.TaskID).SetLeaseExpiresAt(time.Now().Add(-time.Second)).Save(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := tasks.ClaimAny(ctx, []biz.BackgroundTaskKind{biz.BackgroundTaskKindDingTalkDirectorySync}, time.Minute, time.Now()); err != biz.ErrBackgroundTaskNoTask {
		t.Fatalf("过期任务应先进入终态: %v", err)
	}
	if _, err := tasks.Requeue(ctx, org.ID, record.TaskID, time.Now(), &biz.AuditEvent{UserID: &actor, Action: "test.directory.retry", Result: "success"}); err != nil {
		t.Fatal(err)
	}
	fresh, err := tasks.ClaimAny(ctx, []biz.BackgroundTaskKind{biz.BackgroundTaskKindDingTalkDirectorySync}, time.Minute, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if old.LeaseToken == nil || fresh.LeaseToken == nil || *old.LeaseToken == *fresh.LeaseToken {
		t.Fatal("重试必须重新发放租约")
	}
	person := &biz.DingTalkDirectoryPerson{UserID: "retry-user", UnionID: "retry-union", Name: "重试员工"}
	if _, err := repo.ApplyPerson(ctx, old, person, []string{"display_name"}); err != biz.ErrBackgroundTaskLeaseMismatch {
		t.Fatalf("旧租约迟到写入应拒绝: %v", err)
	}
	if err := repo.Finish(ctx, old, &biz.DingTalkDirectorySync{TotalCount: 99}, ""); err != biz.ErrBackgroundTaskLeaseMismatch {
		t.Fatalf("旧租约迟到完成应拒绝: %v", err)
	}
	if _, err := repo.ApplyPerson(ctx, fresh, person, []string{"display_name"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Finish(ctx, fresh, &biz.DingTalkDirectorySync{TotalCount: 1, CreatedCount: 1}, ""); err != nil {
		t.Fatal(err)
	}
	result, err := repo.Get(ctx, org.ID, record.TaskID, 1, 20)
	if err != nil || result.TotalCount != 1 || result.CreatedCount != 1 || result.Status != biz.BackgroundTaskStatusSucceeded {
		t.Fatalf("重试结果被旧执行器覆盖: %+v %v", result, err)
	}
}
