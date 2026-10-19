package data

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	reset "github.com/roncin/roncin-go-admin/server/internal/data/ent/passwordresetchallenge"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/schema"
)

// 通过正式迁移初始化的隔离 PostgreSQL 验证三种来源可选资料空值、其他资料和失败快照。
func TestDingTalkDirectoryOptionalProfileAndFailureNamesPostgres(t *testing.T) {
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
		name                  string
		email, mobile, avatar *string
		withoutAuthorization  bool
	}{
		{"缺键或null", nil, nil, nil, false},
		{"空串", directoryTestString(""), directoryTestString(""), directoryTestString(""), false},
		{"非空", directoryTestString("current@example.test"), directoryTestString("13800000000"), directoryTestString("https://example.test/avatar"), false},
		{"无字段授权", directoryTestString("current@example.test"), directoryTestString("13800000000"), directoryTestString("https://example.test/avatar"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			account, err := data.db.User.Create().SetDisplayName("旧姓名").SetEmail("old@example.test").SetDingtalkMobile("13900000000").SetDingtalkMobileNormalized("13900000000").SetAvatarURL("https://example.test/old-avatar").SetDingtalkUserid(tc.name).SetDingtalkUnionid("union-" + tc.name).SetEnabled(false).SetPendingAssignment(true).Save(ctx)
			if err != nil {
				t.Fatal(err)
			}
			challenge, err := data.db.PasswordResetChallenge.Create().SetMobileKeyHash("mobile-" + tc.name).SetTokenHash("token-" + tc.name).SetCodeHash("sensitive-code-hash").SetUserID(account.ID).SetStatus(reset.StatusREADY).SetExpiresAt(time.Now().Add(5 * time.Minute)).SetResendAvailableAt(time.Now().Add(time.Minute)).Save(ctx)
			if err != nil {
				t.Fatal(err)
			}
			person := &biz.DingTalkDirectoryPerson{UserID: tc.name, UnionID: "union-" + tc.name, Name: "新姓名", Email: tc.email, Mobile: tc.mobile, AvatarURL: tc.avatar, Departments: []biz.DingTalkDepartment{{ID: 10, Name: "新部门"}}}
			authorizedFields := fields
			if tc.withoutAuthorization {
				authorizedFields = []string{"display_name", "departments"}
			}
			if outcome, err := repo.ApplyPerson(ctx, task, person, authorizedFields); err != nil || outcome != "updated" {
				t.Fatalf("正常资料未更新: outcome=%s err=%v", outcome, err)
			}
			reloaded, err := data.db.User.Get(ctx, account.ID)
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range []struct {
				name      string
				got, want *string
			}{
				{"email", reloaded.Email, tc.email},
				{"mobile", reloaded.DingtalkMobile, tc.mobile},
				{"avatar", reloaded.AvatarURL, tc.avatar},
			} {
				want := field.want
				if tc.withoutAuthorization || want != nil && *want == "" {
					want = nil
				}
				if !reflect.DeepEqual(field.got, want) {
					t.Fatalf("%s 未按来源更新或清空", field.name)
				}
			}
			var expectedNormalized *string
			if !tc.withoutAuthorization && tc.mobile != nil {
				normalized := biz.NormalizeDingTalkMobile(*tc.mobile)
				if biz.ValidDingTalkMobile(normalized) {
					expectedNormalized = &normalized
				}
			}
			if !reflect.DeepEqual(reloaded.DingtalkMobileNormalized, expectedNormalized) {
				t.Fatal("手机号检索投影未随来源更新或清空")
			}
			reloadedChallenge := data.db.PasswordResetChallenge.GetX(ctx, challenge.ID)
			if reloadedChallenge.Status != reset.StatusFAILED || reloadedChallenge.CodeHash != "" {
				t.Fatal("号码变化未废弃旧找回挑战")
			}
			if reloaded.DisplayName != "新姓名" || !reflect.DeepEqual(reloaded.DingtalkDepartments, []schema.DirectoryDepartment{{ID: 10, Name: "新部门"}}) {
				t.Fatal("可选资料空值阻断了姓名和部门更新")
			}
			if reloaded.Enabled || !reloaded.PendingAssignment || !reflect.DeepEqual(reloaded.DingtalkUserid, account.DingtalkUserid) || !reflect.DeepEqual(reloaded.DingtalkUnionid, account.DingtalkUnionid) {
				t.Fatal("可选资料空值修改了人员身份或开通状态")
			}
			if n, err := data.db.Membership.Query().Where(membership.UserIDEQ(account.ID)).Count(ctx); err != nil || n != 0 {
				t.Fatal("同步自动创建了成员关系")
			}
			if outcome, err := repo.ApplyPerson(ctx, task, person, authorizedFields); err != nil || outcome != "unchanged" {
				t.Fatalf("重复同步未保持幂等: outcome=%s err=%v", outcome, err)
			}
		})
	}
	for _, field := range []string{"UserID", "UnionID", "Name"} {
		t.Run("必需字段缺失/"+field, func(t *testing.T) {
			account, err := data.db.User.Create().SetDisplayName("保留姓名").SetDingtalkUserid("required-" + field).SetDingtalkUnionid("required-union-" + field).SetEmail("old@example.test").SetDingtalkMobile("13900000000").SetAvatarURL("https://example.test/old-avatar").SetEnabled(false).SetPendingAssignment(true).Save(ctx)
			if err != nil {
				t.Fatal(err)
			}
			for _, invalid := range []string{"", " \t "} {
				person := &biz.DingTalkDirectoryPerson{UserID: *account.DingtalkUserid, UnionID: *account.DingtalkUnionid, Name: "新姓名"}
				switch field {
				case "UserID":
					person.UserID = invalid
				case "UnionID":
					person.UnionID = invalid
				case "Name":
					person.Name = invalid
				}
				if _, err := repo.ApplyPerson(ctx, task, person, fields); err != biz.ErrDingTalkIdentityConflict {
					t.Fatalf("必需字段缺失应拒绝: err=%v", err)
				}
				reloaded, err := data.db.User.Get(ctx, account.ID)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(reloaded.Email, account.Email) || !reflect.DeepEqual(reloaded.DingtalkMobile, account.DingtalkMobile) || !reflect.DeepEqual(reloaded.AvatarURL, account.AvatarURL) || reloaded.DisplayName != account.DisplayName || !reflect.DeepEqual(reloaded.DingtalkUserid, account.DingtalkUserid) || !reflect.DeepEqual(reloaded.DingtalkUnionid, account.DingtalkUnionid) || !reflect.DeepEqual(reloaded.DingtalkLastSyncedAt, account.DingtalkLastSyncedAt) || reloaded.Enabled != account.Enabled || reloaded.PendingAssignment != account.PendingAssignment {
					t.Fatal("必需字段缺失改写了本地资料、身份或状态")
				}
			}
		})
	}
	failures := []biz.DingTalkDirectoryFailure{
		{UserID: "known-person", UserName: "失败人员", DepartmentID: 10, DepartmentName: "已知部门", Reason: "DINGTALK_PERSON_SYNC_FAILED", Message: "写入失败"},
		{DepartmentID: 20, DepartmentName: "接口失败部门", Reason: "DINGTALK_MEMBERS_READ_FAILED", Message: "部门人员接口失败"},
		{UserID: "unknown-person", Reason: "DINGTALK_PERSON_READ_FAILED", Message: "人员接口失败"},
	}
	if err := repo.Finish(ctx, task, &biz.DingTalkDirectorySync{TotalCount: 4, UpdatedCount: 4, FailedCount: 3, Failures: failures}, ""); err != nil {
		t.Fatal(err)
	}
	for i, expected := range failures {
		result, err := repo.Get(ctx, org.ID, record.TaskID, i+1, 1)
		if err != nil || result.FailureTotal != 3 || result.FailedCount != 3 || len(result.Failures) != 1 || !reflect.DeepEqual(result.Failures[0], expected) {
			t.Fatalf("失败名称或分页未完整保留: page=%d err=%v", i+1, err)
		}
	}
	t.Run("人员接口失败保留本地资料", func(t *testing.T) {
		account, err := data.db.User.Create().SetDisplayName("保留姓名").SetDingtalkUserid("failed-user").SetDingtalkUnionid("failed-union").SetEmail("old@example.test").SetDingtalkMobile("13900000000").SetAvatarURL("https://example.test/old-avatar").SetDingtalkDepartments([]schema.DirectoryDepartment{{ID: 30, Name: "保留部门"}}).SetEnabled(false).SetPendingAssignment(true).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		failedRecord, err := repo.Start(ctx, org.ID, actor, "feedback-corp")
		if err != nil {
			t.Fatal(err)
		}
		provider := &dingTalkIdentityProvider{enabled: true, corpID: "feedback-corp", appToken: "test-token", appTokenTill: time.Now().Add(time.Hour), client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			body := `{"errcode":88}`
			if r.URL.Path == "/auth/scopes" {
				body = `{"errcode":0,"auth_user_field":["email","mobile","avatar"],"auth_org_scopes":{"authed_user":["failed-user"]}}`
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		})}}
		uc := biz.NewDingTalkDirectoryUsecase(repo, provider, biz.NewBackgroundTaskUsecase(&backgroundTaskRepo{data: data}))
		if err := uc.ProcessNext(ctx, time.Minute); err != nil {
			t.Fatal(err)
		}
		reloaded, err := data.db.User.Get(ctx, account.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(reloaded.Email, account.Email) || !reflect.DeepEqual(reloaded.DingtalkMobile, account.DingtalkMobile) || !reflect.DeepEqual(reloaded.AvatarURL, account.AvatarURL) || reloaded.DisplayName != account.DisplayName || !reflect.DeepEqual(reloaded.DingtalkDepartments, account.DingtalkDepartments) || !reflect.DeepEqual(reloaded.DingtalkLastSyncedAt, account.DingtalkLastSyncedAt) || reloaded.Enabled != account.Enabled || reloaded.PendingAssignment != account.PendingAssignment {
			t.Fatal("真实人员接口失败清空或改写了本地资料")
		}
		result, err := repo.Get(ctx, org.ID, failedRecord.TaskID, 1, 20)
		if err != nil || result.FailedCount != 1 || len(result.Failures) != 1 || result.Failures[0].Reason != "DINGTALK_PERSON_READ_FAILED" || result.UpdatedCount != 0 {
			t.Fatalf("真实人员接口错误未明确保存: err=%v", err)
		}
	})

}
