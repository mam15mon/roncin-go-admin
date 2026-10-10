package data

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/schema"
)

// 通过正式迁移初始化的隔离 PostgreSQL 验证来源邮箱空值、其他资料和失败快照。
func TestDingTalkDirectoryOptionalEmailAndFailureNamesPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	org, err := data.db.Organization.Create().SetCode("feedback-system").SetName("系统管理").SetKind("system").SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ctx, actor := directoryAdministrator(t, data, org.ID)
	repo := &dingTalkDirectoryRepo{data: data}
	record, err := repo.Start(ctx, org.ID, actor, "feedback-corp")
	if err != nil {
		t.Fatal(err)
	}
	task, err := (&backgroundTaskRepo{data: data}).ClaimAny(ctx, []biz.BackgroundTaskKind{biz.BackgroundTaskKindDingTalkDirectorySync}, time.Minute, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	fields := []string{"display_name", "departments", "email", "mobile", "avatar_url"}
	for _, tc := range []struct {
		name  string
		email *string
	}{
		{"缺键或null", nil},
		{"空串", directoryTestString("")},
		{"非空", directoryTestString("current@example.test")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			account, err := data.db.User.Create().SetDisplayName("旧姓名").SetEmail("old@example.test").SetDingtalkUserid(tc.name).SetDingtalkUnionid("union-" + tc.name).SetEnabled(false).SetPendingAssignment(true).Save(ctx)
			if err != nil {
				t.Fatal(err)
			}
			person := &biz.DingTalkDirectoryPerson{UserID: tc.name, UnionID: "union-" + tc.name, Name: "新姓名", Email: tc.email, Mobile: directoryTestString("13800000000"), AvatarURL: directoryTestString("https://example.test/avatar"), Departments: []biz.DingTalkDepartment{{ID: 10, Name: "新部门"}}}
			if outcome, err := repo.ApplyPerson(ctx, task, person, fields); err != nil || outcome != "updated" {
				t.Fatalf("正常资料未更新: outcome=%s err=%v", outcome, err)
			}
			reloaded, err := data.db.User.Get(ctx, account.ID)
			if err != nil {
				t.Fatal(err)
			}
			wantEmail := tc.email
			if wantEmail != nil && *wantEmail == "" {
				wantEmail = nil
			}
			if !reflect.DeepEqual(reloaded.Email, wantEmail) || reloaded.DisplayName != "新姓名" || !reflect.DeepEqual(reloaded.DingtalkMobile, person.Mobile) || !reflect.DeepEqual(reloaded.AvatarURL, person.AvatarURL) || !reflect.DeepEqual(reloaded.DingtalkDepartments, []schema.DirectoryDepartment{{ID: 10, Name: "新部门"}}) {
				t.Fatal("邮箱或相邻资料未按来源更新")
			}
			if reloaded.Enabled || !reloaded.PendingAssignment || !reflect.DeepEqual(reloaded.DingtalkUserid, account.DingtalkUserid) || !reflect.DeepEqual(reloaded.DingtalkUnionid, account.DingtalkUnionid) {
				t.Fatal("邮箱空值修改了人员身份或开通状态")
			}
			if n, err := data.db.Membership.Query().Where(membership.UserIDEQ(account.ID)).Count(ctx); err != nil || n != 0 {
				t.Fatal("同步自动创建了成员关系")
			}
			if outcome, err := repo.ApplyPerson(ctx, task, person, fields); err != nil || outcome != "unchanged" {
				t.Fatalf("重复同步未保持幂等: outcome=%s err=%v", outcome, err)
			}
		})
	}
	failures := []biz.DingTalkDirectoryFailure{
		{UserID: "known-person", UserName: "失败人员", DepartmentID: 10, DepartmentName: "已知部门", Reason: "DINGTALK_PERSON_SYNC_FAILED", Message: "写入失败"},
		{DepartmentID: 20, DepartmentName: "接口失败部门", Reason: "DINGTALK_MEMBERS_READ_FAILED", Message: "部门人员接口失败"},
		{UserID: "unknown-person", Reason: "DINGTALK_PERSON_READ_FAILED", Message: "人员接口失败"},
	}
	if err := repo.Finish(ctx, task, &biz.DingTalkDirectorySync{TotalCount: 3, UpdatedCount: 3, FailedCount: 3, Failures: failures}, ""); err != nil {
		t.Fatal(err)
	}
	for i, expected := range failures {
		result, err := repo.Get(ctx, org.ID, record.TaskID, i+1, 1)
		if err != nil || result.FailureTotal != 3 || result.FailedCount != 3 || len(result.Failures) != 1 || !reflect.DeepEqual(result.Failures[0], expected) {
			t.Fatalf("失败名称或分页未完整保留: page=%d err=%v", i+1, err)
		}
	}
}
