package biz

import (
	"context"
	"github.com/google/uuid"
	"testing"
	"time"
)

type directoryGatewayStub struct {
	snapshot *DingTalkDirectorySnapshot
	err      error
}

func (s *directoryGatewayStub) Enabled() bool        { return true }
func (s *directoryGatewayStub) EnterpriseID() string { return "test-corp" }
func (s *directoryGatewayStub) ReadDirectory(context.Context) (*DingTalkDirectorySnapshot, error) {
	return s.snapshot, s.err
}

type directoryTaskRepoStub struct{ BackgroundTaskRepo }

func (s *directoryTaskRepoStub) ClaimAny(context.Context, []BackgroundTaskKind, time.Duration, time.Time) (*BackgroundTask, error) {
	token := "lease"
	expires := time.Now().Add(time.Minute)
	return &BackgroundTask{ID: uuid.New(), LeaseToken: &token, LeaseExpiresAt: &expires}, nil
}

type directoryRepoStub struct {
	DingTalkDirectoryRepo
	people   []*DingTalkDirectoryPerson
	finished *DingTalkDirectorySync
	message  string
	applyErr error
}

func (s *directoryRepoStub) ApplyPerson(_ context.Context, _ *BackgroundTask, p *DingTalkDirectoryPerson, _ []string) (string, error) {
	s.people = append(s.people, p)
	return "created", s.applyErr
}
func (s *directoryRepoStub) Finish(_ context.Context, _ *BackgroundTask, result *DingTalkDirectorySync, message string) error {
	s.finished = result
	s.message = message
	return nil
}
func TestDingTalkDirectoryDedupRejectsConflictingIdentityAndMergesDepartments(t *testing.T) {
	repo := &directoryRepoStub{}
	gateway := &directoryGatewayStub{snapshot: &DingTalkDirectorySnapshot{CorpID: "test-corp", People: []DingTalkDirectoryPerson{
		{UserID: "conflict", UnionID: "u1", Name: "冲突人员"},
		{UserID: "conflict", UnionID: "u2", Name: "冲突人员", Departments: []DingTalkDepartment{{ID: 2, Name: "部门乙"}}},
		{UserID: "conflict", UnionID: "u1", Name: "冲突人员"},
		{UserID: "good", UnionID: "u3", Name: "跨部门人员", Departments: []DingTalkDepartment{{ID: 1, Name: "部门甲"}}},
		{UserID: "good", UnionID: "u3", Name: "跨部门人员", Departments: []DingTalkDepartment{{ID: 2, Name: "部门乙"}}},
	}}}
	uc := NewDingTalkDirectoryUsecase(repo, gateway, NewBackgroundTaskUsecase(&directoryTaskRepoStub{}))
	if err := uc.ProcessNext(context.Background(), time.Minute); err != nil {
		t.Fatal(err)
	}
	if len(repo.people) != 1 || repo.people[0].UserID != "good" || len(repo.people[0].Departments) != 2 {
		t.Fatalf("身份去重错误: %+v", repo.people)
	}
	if repo.finished.TotalCount != 2 || repo.finished.CreatedCount != 1 || repo.finished.FailedCount != 1 || repo.finished.Failures[0].UserID != "conflict" {
		t.Fatalf("本轮人数/失败统计错误: %+v", repo.finished)
	}
	if repo.finished.Failures[0].UserName != "冲突人员" || repo.finished.Failures[0].DepartmentName != "部门乙" || repo.finished.Failures[0].DepartmentID != 2 {
		t.Fatal("身份冲突未保留失败时已知名称")
	}
}

func TestDingTalkDirectoryApplyFailureKeepsPersonSnapshot(t *testing.T) {
	for _, failure := range []error{ErrDingTalkIdentityConflict, ErrDingTalkDirectoryFailed} {
		repo := &directoryRepoStub{applyErr: failure}
		gateway := &directoryGatewayStub{snapshot: &DingTalkDirectorySnapshot{People: []DingTalkDirectoryPerson{{UserID: "failed", UnionID: "union", Name: "失败人员", Departments: []DingTalkDepartment{{ID: 20, Name: "部门乙"}, {ID: 10, Name: "部门甲"}}}}}}
		uc := NewDingTalkDirectoryUsecase(repo, gateway, NewBackgroundTaskUsecase(&directoryTaskRepoStub{}))
		if err := uc.ProcessNext(context.Background(), time.Minute); err != nil {
			t.Fatal(err)
		}
		if repo.finished.FailedCount != 1 || repo.finished.CreatedCount != 0 {
			t.Fatal("真实写入错误被误判成功")
		}
		f := repo.finished.Failures[0]
		if f.UserName != "失败人员" || f.UserID != "failed" || f.DepartmentID != 10 || f.DepartmentName != "部门甲" || f.Message == "" {
			t.Fatal("写入错误未携带确定性人员部门快照")
		}
	}
}
func TestDingTalkDirectoryGatewayFailureFinishesExplicitly(t *testing.T) {
	repo := &directoryRepoStub{}
	uc := NewDingTalkDirectoryUsecase(repo, &directoryGatewayStub{err: ErrDingTalkDirectoryFailed}, NewBackgroundTaskUsecase(&directoryTaskRepoStub{}))
	if err := uc.ProcessNext(context.Background(), time.Minute); err != nil {
		t.Fatal(err)
	}
	if repo.message == "" || len(repo.people) != 0 {
		t.Fatal("网关失败不能伪装为成功空同步")
	}
}
