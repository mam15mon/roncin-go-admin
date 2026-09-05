package biz

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type dingTalkApprovalRepoStub struct {
	dispatch       *DingTalkApprovalDispatch
	dispatchResult *DingTalkApprovalDispatchOutcome
	inboxJob       *DingTalkApprovalInboxJob
	failMessage    string
	failNextRunAt  time.Time
	ignoredCode    string
}

func (s *dingTalkApprovalRepoStub) PrepareDispatch(context.Context, *BackgroundTask) (*DingTalkApprovalDispatch, error) {
	return s.dispatch, nil
}

func (s *dingTalkApprovalRepoStub) FinishDispatch(_ context.Context, _ *BackgroundTask, outcome *DingTalkApprovalDispatchOutcome, _ time.Time) error {
	s.dispatchResult = outcome
	return nil
}

func (*dingTalkApprovalRepoStub) StoreCallback(context.Context, *DingTalkApprovalCallbackEvent) error {
	return nil
}

func (s *dingTalkApprovalRepoStub) ClaimInbox(context.Context, time.Duration, time.Time) (*DingTalkApprovalInboxJob, error) {
	return s.inboxJob, nil
}

func (s *dingTalkApprovalRepoStub) FailInbox(_ context.Context, _ *DingTalkApprovalInboxJob, message string, nextRunAt time.Time) error {
	s.failMessage = message
	s.failNextRunAt = nextRunAt
	return nil
}

func (s *dingTalkApprovalRepoStub) IgnoreInbox(_ context.Context, _ *DingTalkApprovalInboxJob, resultCode string) error {
	s.ignoredCode = resultCode
	return nil
}

func (*dingTalkApprovalRepoStub) RecordRejected(context.Context, *DingTalkApprovalInboxJob, *DingTalkApprovalQueryResult, time.Time) error {
	return nil
}

func (*dingTalkApprovalRepoStub) PrepareApproved(context.Context, *DingTalkApprovalInboxJob, *DingTalkApprovalQueryResult, time.Time) (uuid.UUID, bool, error) {
	return uuid.Nil, false, nil
}

func (*dingTalkApprovalRepoStub) ApplyApproved(context.Context, *DingTalkApprovalInboxJob, uuid.UUID, *DingTalkApprovalQueryResult, time.Time) error {
	return nil
}

type dingTalkApprovalGatewayStub struct {
	createCalls   int
	createCommand *DingTalkApprovalCreateCommand
	createResult  *DingTalkApprovalCreateResult
	createErr     error
	queryResult   *DingTalkApprovalQueryResult
	queryErr      error
}

func (s *dingTalkApprovalGatewayStub) Create(_ context.Context, command *DingTalkApprovalCreateCommand) (*DingTalkApprovalCreateResult, error) {
	s.createCalls++
	s.createCommand = command
	if s.createErr != nil {
		return nil, s.createErr
	}
	if s.createResult != nil {
		return s.createResult, nil
	}
	return nil, errors.New("测试不应发起审批")
}

func (s *dingTalkApprovalGatewayStub) Query(context.Context, string) (*DingTalkApprovalQueryResult, error) {
	return s.queryResult, s.queryErr
}

type dingTalkApprovalCodecStub struct{}

func (dingTalkApprovalCodecStub) Enabled() bool { return false }

func (dingTalkApprovalCodecStub) Decode(string, string, string, string) (*DingTalkApprovalCallbackEvent, error) {
	return nil, errors.New("测试未启用回调")
}

func (dingTalkApprovalCodecStub) EncodeSuccess(string, string) (string, string, error) {
	return "", "", errors.New("测试未启用回调")
}

func TestDingTalkApprovalPendingAuthorityResultIsRetried(t *testing.T) {
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	leaseToken := uuid.NewString()
	taskRepo := &backgroundTaskRepoStub{claimTask: &BackgroundTask{
		ID: uuid.New(), OrganizationID: uuid.New(), LeaseToken: &leaseToken,
	}}
	repo := &dingTalkApprovalRepoStub{inboxJob: &DingTalkApprovalInboxJob{
		ID: uuid.New(), LeaseToken: uuid.NewString(), ProcessInstanceID: "instance-1",
	}}
	gateway := &dingTalkApprovalGatewayStub{queryResult: &DingTalkApprovalQueryResult{Decision: DingTalkApprovalDecisionPending}}
	usecase := NewDingTalkApprovalUsecase(NewBackgroundTaskUsecase(taskRepo), repo, gateway, dingTalkApprovalCodecStub{})
	usecase.now = func() time.Time { return now }

	if err := usecase.ProcessNextInbox(context.Background(), time.Minute); err != nil {
		t.Fatalf("暂未终结的权威结果应进入退避重查: %v", err)
	}
	if repo.failMessage != "钉钉审批权威状态尚未终结" {
		t.Fatalf("未保存可重试状态: %q", repo.failMessage)
	}
	if want := now.Add(time.Minute); !repo.failNextRunAt.Equal(want) {
		t.Fatalf("next run at = %s, want %s", repo.failNextRunAt, want)
	}
	if repo.ignoredCode != "" {
		t.Fatalf("暂未终结事件不得置为 IGNORED: %s", repo.ignoredCode)
	}
}

func TestDingTalkApprovalSendingRecoveryStopsDuplicateCreate(t *testing.T) {
	leaseToken := uuid.NewString()
	task := &BackgroundTask{ID: uuid.New(), OrganizationID: uuid.New(), LeaseToken: &leaseToken}
	taskRepo := &backgroundTaskRepoStub{claimTask: task}
	repo := &dingTalkApprovalRepoStub{dispatch: &DingTalkApprovalDispatch{
		TaskID: task.ID, OrganizationID: task.OrganizationID, DispatchStatus: "SENDING", ShouldSend: false,
	}}
	gateway := &dingTalkApprovalGatewayStub{}
	usecase := NewDingTalkApprovalUsecase(NewBackgroundTaskUsecase(taskRepo), repo, gateway, dingTalkApprovalCodecStub{})

	if err := usecase.ProcessNextDispatch(context.Background(), time.Minute); err != nil {
		t.Fatalf("发送中崩溃恢复应安全终结为 UNKNOWN: %v", err)
	}
	if gateway.createCalls != 0 {
		t.Fatalf("发送中状态不得重建外部审批，create calls = %d", gateway.createCalls)
	}
	if repo.dispatchResult == nil || repo.dispatchResult.FailureKind != DingTalkApprovalDispatchFailureUnknown {
		t.Fatalf("dispatch outcome = %#v, want UNKNOWN", repo.dispatchResult)
	}
}

func TestDingTalkApprovalDispatchForwardsUniversalOrderContext(t *testing.T) {
	leaseToken := uuid.NewString()
	task := &BackgroundTask{ID: uuid.New(), OrganizationID: uuid.New(), LeaseToken: &leaseToken}
	taskRepo := &backgroundTaskRepoStub{claimTask: task}
	reason := "修改空运单证"
	repo := &dingTalkApprovalRepoStub{dispatch: &DingTalkApprovalDispatch{
		TaskID:               task.ID,
		OrganizationID:       task.OrganizationID,
		ProcessCode:          "PROC-ORDER-UNLOCK",
		ApplicantUserID:      "dt-applicant",
		ApproverUserIDs:      []string{"dt-approver"},
		BusinessType:         OrderBusinessAI,
		OrderNo:              "AI-001",
		ApplicantDisplayName: "申请人甲",
		LockGeneration:       3,
		Reason:               &reason,
		DispatchStatus:       "PENDING",
		ShouldSend:           true,
	}}
	gateway := &dingTalkApprovalGatewayStub{createResult: &DingTalkApprovalCreateResult{ProcessInstanceID: "instance-1"}}
	usecase := NewDingTalkApprovalUsecase(NewBackgroundTaskUsecase(taskRepo), repo, gateway, dingTalkApprovalCodecStub{})

	if err := usecase.ProcessNextDispatch(context.Background(), time.Minute); err != nil {
		t.Fatalf("派发通用订单审批失败: %v", err)
	}
	command := gateway.createCommand
	if command == nil || command.BusinessType != OrderBusinessAI || command.OrderNo != "AI-001" || command.ApplicantDisplayName != "申请人甲" || command.LockGeneration != 3 || command.Reason == nil || *command.Reason != reason {
		t.Fatalf("审批命令上下文未完整转发: %#v", command)
	}
}

var _ DingTalkApprovalRepo = (*dingTalkApprovalRepoStub)(nil)
var _ DingTalkApprovalGateway = (*dingTalkApprovalGatewayStub)(nil)
var _ DingTalkApprovalCallbackCodec = dingTalkApprovalCodecStub{}
