package biz

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DingTalkApprovalDispatchFailureKind 描述审批创建调用能否安全重试。
type DingTalkApprovalDispatchFailureKind string

const (
	DingTalkApprovalDispatchFailureRetryable DingTalkApprovalDispatchFailureKind = "RETRYABLE"
	DingTalkApprovalDispatchFailureRejected  DingTalkApprovalDispatchFailureKind = "REJECTED"
	DingTalkApprovalDispatchFailureUnknown   DingTalkApprovalDispatchFailureKind = "UNKNOWN"
)

// DingTalkApprovalDispatchError 是钉钉审批创建的分类错误，不携带完整外部响应。
type DingTalkApprovalDispatchError struct {
	Kind           DingTalkApprovalDispatchFailureKind
	Code           string
	Message        string
	ResponseDigest string
	Cause          error
}

func (e *DingTalkApprovalDispatchError) Error() string {
	if e == nil {
		return "钉钉审批创建失败"
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	return "钉钉审批创建失败"
}

func (e *DingTalkApprovalDispatchError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// DingTalkApprovalCreateCommand 是平台无关的审批创建命令。
type DingTalkApprovalCreateCommand struct {
	ProcessCode          string
	ApplicantUserID      string
	ApproverUserIDs      []string
	BusinessType         OrderBusinessType
	OrderNo              string
	ApplicantDisplayName string
	LockGeneration       uint64
	Reason               *string
}

// DingTalkApprovalCreateResult 是审批创建成功后的最小结果。
type DingTalkApprovalCreateResult struct {
	ProcessInstanceID string
	ResponseDigest    string
}

// DingTalkApprovalDecision 是权威 OA 查询返回的终态；非终态统一为 PENDING。
type DingTalkApprovalDecision string

const (
	DingTalkApprovalDecisionPending  DingTalkApprovalDecision = "PENDING"
	DingTalkApprovalDecisionApproved DingTalkApprovalDecision = "APPROVED"
	DingTalkApprovalDecisionRejected DingTalkApprovalDecision = "REJECTED"
)

// DingTalkApprovalQueryResult 仅保留本地决策所需的权威结果。
type DingTalkApprovalQueryResult struct {
	Decision       DingTalkApprovalDecision
	ApproverUserID string
	DecidedAt      *time.Time
}

// DingTalkApprovalGateway 隔离原生 OA API，不与机器人通知接口混用。
type DingTalkApprovalGateway interface {
	Create(context.Context, *DingTalkApprovalCreateCommand) (*DingTalkApprovalCreateResult, error)
	Query(context.Context, string) (*DingTalkApprovalQueryResult, error)
}

// DingTalkApprovalDispatch 保存 Worker 创建审批所需的不可变快照。
type DingTalkApprovalDispatch struct {
	TaskID               uuid.UUID
	OrganizationID       uuid.UUID
	UnlockRequestID      uuid.UUID
	ProcessCode          string
	ApplicantUserID      string
	ApproverUserIDs      []string
	BusinessType         OrderBusinessType
	OrderNo              string
	ApplicantDisplayName string
	LockGeneration       uint64
	Reason               *string
	DispatchStatus       string
	ShouldSend           bool
	ProcessInstanceID    string
	ErrorCategory        string
}

// DingTalkApprovalDispatchOutcome 是外部调用完成后写回的最小结果。
type DingTalkApprovalDispatchOutcome struct {
	FailureKind       DingTalkApprovalDispatchFailureKind
	ProcessInstanceID string
	ResponseDigest    string
	ErrorCode         string
	ErrorMessage      string
}

// DingTalkApprovalRepo 持有派发状态与后台任务租约的原子写入契约。
type DingTalkApprovalRepo interface {
	PrepareDispatch(context.Context, *BackgroundTask) (*DingTalkApprovalDispatch, error)
	FinishDispatch(context.Context, *BackgroundTask, *DingTalkApprovalDispatchOutcome, time.Time) error
	StoreCallback(context.Context, *DingTalkApprovalCallbackEvent) error
	ClaimInbox(context.Context, time.Duration, time.Time) (*DingTalkApprovalInboxJob, error)
	FailInbox(context.Context, *DingTalkApprovalInboxJob, string, time.Time) error
	IgnoreInbox(context.Context, *DingTalkApprovalInboxJob, string) error
	RecordRejected(context.Context, *DingTalkApprovalInboxJob, *DingTalkApprovalQueryResult, time.Time) error
	PrepareApproved(context.Context, *DingTalkApprovalInboxJob, *DingTalkApprovalQueryResult, time.Time) (uuid.UUID, bool, error)
	ApplyApproved(context.Context, *DingTalkApprovalInboxJob, uuid.UUID, *DingTalkApprovalQueryResult, time.Time) error
}

// DingTalkApprovalCallbackEvent 是验签、解密和企业校验后的最小事件。
type DingTalkApprovalCallbackEvent struct {
	EventID              string
	CorpID               string
	EventType            string
	ProcessInstanceID    string
	EncryptedPayloadHash string
}

// DingTalkApprovalInboxJob 是 Inbox Worker 的租约快照。
type DingTalkApprovalInboxJob struct {
	ID                uuid.UUID
	LeaseToken        string
	ProcessInstanceID string
}

// DingTalkApprovalCallbackCodec 将钉钉回调协议隔离在 integration 层。
type DingTalkApprovalCallbackCodec interface {
	Enabled() bool
	Decode(signature, timestamp, nonce, encrypted string) (*DingTalkApprovalCallbackEvent, error)
	EncodeSuccess(timestamp, nonce string) (encrypted, signature string, err error)
}

// DingTalkApprovalUsecase 编排事务外 OA 网络调用与事务内可靠状态写回。
type DingTalkApprovalUsecase struct {
	tasks   *BackgroundTaskUsecase
	repo    DingTalkApprovalRepo
	gateway DingTalkApprovalGateway
	now     func() time.Time
	codec   DingTalkApprovalCallbackCodec
}

func NewDingTalkApprovalUsecase(tasks *BackgroundTaskUsecase, repo DingTalkApprovalRepo, gateway DingTalkApprovalGateway, codec DingTalkApprovalCallbackCodec) *DingTalkApprovalUsecase {
	return &DingTalkApprovalUsecase{tasks: tasks, repo: repo, gateway: gateway, codec: codec, now: time.Now}
}

func (uc *DingTalkApprovalUsecase) CallbackEnabled() bool { return uc.codec.Enabled() }

// ReceiveCallback 验证入站协议并幂等写入 Inbox；不在 HTTP 请求内查询 OA 或生效解锁。
func (uc *DingTalkApprovalUsecase) ReceiveCallback(ctx context.Context, signature, timestamp, nonce, encrypted string) (string, string, error) {
	event, err := uc.codec.Decode(signature, timestamp, nonce, encrypted)
	if err != nil {
		return "", "", err
	}
	if event.EventType == "bpms_instance_change" {
		if err := uc.repo.StoreCallback(ctx, event); err != nil {
			return "", "", err
		}
	}
	return uc.codec.EncodeSuccess(timestamp, nonce)
}

// ProcessNextDispatch 领取一条审批创建任务；外部网络调用始终发生在数据库事务外。
func (uc *DingTalkApprovalUsecase) ProcessNextDispatch(ctx context.Context, leaseDuration time.Duration) error {
	task, err := uc.tasks.ClaimAny(ctx, []BackgroundTaskKind{BackgroundTaskKindDingTalkApproval}, leaseDuration)
	if err != nil {
		return err
	}
	dispatch, err := uc.repo.PrepareDispatch(ctx, task)
	if err != nil {
		return fmt.Errorf("准备钉钉审批派发: %w", err)
	}
	if !dispatch.ShouldSend {
		outcome := &DingTalkApprovalDispatchOutcome{}
		switch dispatch.DispatchStatus {
		case "DISPATCHED":
			outcome.ProcessInstanceID = dispatch.ProcessInstanceID
		case "FAILED":
			outcome.FailureKind = DingTalkApprovalDispatchFailureRejected
			outcome.ErrorCode = dispatch.ErrorCategory
			outcome.ErrorMessage = "钉钉审批发起已明确失败"
		case "UNKNOWN", "SENDING":
			outcome.FailureKind = DingTalkApprovalDispatchFailureUnknown
			outcome.ErrorCode = "DINGTALK_APPROVAL_CREATE_UNKNOWN"
			outcome.ErrorMessage = "上次钉钉审批发起可能已送达，已停止自动重试，请人工核对"
		default:
			outcome.FailureKind = DingTalkApprovalDispatchFailureUnknown
			outcome.ErrorCode = "DINGTALK_APPROVAL_DISPATCH_STATE_INVALID"
			outcome.ErrorMessage = "钉钉审批派发状态异常，已停止自动重试"
		}
		return uc.repo.FinishDispatch(ctx, task, outcome, uc.now().UTC())
	}

	result, createErr := uc.gateway.Create(ctx, &DingTalkApprovalCreateCommand{
		ProcessCode:          dispatch.ProcessCode,
		ApplicantUserID:      dispatch.ApplicantUserID,
		ApproverUserIDs:      append([]string(nil), dispatch.ApproverUserIDs...),
		BusinessType:         dispatch.BusinessType,
		OrderNo:              dispatch.OrderNo,
		ApplicantDisplayName: dispatch.ApplicantDisplayName,
		LockGeneration:       dispatch.LockGeneration,
		Reason:               dispatch.Reason,
	})
	outcome := &DingTalkApprovalDispatchOutcome{}
	if createErr == nil && result != nil {
		outcome.ProcessInstanceID = result.ProcessInstanceID
		outcome.ResponseDigest = result.ResponseDigest
	} else if createErr == nil {
		createErr = &DingTalkApprovalDispatchError{
			Kind:    DingTalkApprovalDispatchFailureUnknown,
			Code:    "DINGTALK_APPROVAL_CREATE_RESPONSE_INCOMPLETE",
			Message: "钉钉审批发起结果未知，请人工核对",
		}
		outcome.FailureKind = DingTalkApprovalDispatchFailureUnknown
		outcome.ErrorCode = "DINGTALK_APPROVAL_CREATE_RESPONSE_INCOMPLETE"
		outcome.ErrorMessage = "钉钉审批发起结果未知，请人工核对"
	} else if classified, ok := createErr.(*DingTalkApprovalDispatchError); ok {
		outcome.FailureKind = classified.Kind
		outcome.ErrorCode = classified.Code
		outcome.ErrorMessage = classified.Error()
		outcome.ResponseDigest = classified.ResponseDigest
	} else {
		// 未分类错误不能证明外部实例未创建，按结果未知处理且禁止重发。
		outcome.FailureKind = DingTalkApprovalDispatchFailureUnknown
		outcome.ErrorCode = "DINGTALK_APPROVAL_CREATE_UNKNOWN"
		outcome.ErrorMessage = "钉钉审批发起结果未知，请人工核对"
	}
	if err := uc.repo.FinishDispatch(ctx, task, outcome, uc.now().UTC()); err != nil {
		return fmt.Errorf("保存钉钉审批派发结果: %w", err)
	}
	if createErr != nil {
		return createErr
	}
	return nil
}

// ProcessNextInbox 先在事务外查询 OA 权威状态，再由仓储按固定锁序落本地终态。
func (uc *DingTalkApprovalUsecase) ProcessNextInbox(ctx context.Context, leaseDuration time.Duration) error {
	now := uc.now().UTC()
	job, err := uc.repo.ClaimInbox(ctx, leaseDuration, now)
	if err != nil {
		return err
	}
	result, err := uc.gateway.Query(ctx, job.ProcessInstanceID)
	if err != nil {
		if failErr := uc.repo.FailInbox(ctx, job, "查询钉钉审批权威状态失败", now.Add(time.Minute)); failErr != nil {
			return fmt.Errorf("查询失败且保存 Inbox 重试状态失败: %v: %w", err, failErr)
		}
		return err
	}
	switch result.Decision {
	case DingTalkApprovalDecisionPending:
		// 回调可能先于 OA 权威查询结果可见。此时不能把 Inbox 事件终结为
		// IGNORED，否则同 event_id 重投只会命中幂等记录，最终的批准或拒绝
		// 可能永久丢失。保留事件并退避重查，不依据回调正文猜测结果。
		return uc.repo.FailInbox(ctx, job, "钉钉审批权威状态尚未终结", now.Add(time.Minute))
	case DingTalkApprovalDecisionRejected:
		return uc.repo.RecordRejected(ctx, job, result, now)
	case DingTalkApprovalDecisionApproved:
		requestID, shouldApply, err := uc.repo.PrepareApproved(ctx, job, result, now)
		if err != nil {
			_ = uc.repo.FailInbox(ctx, job, "保存钉钉审批通过状态失败", now.Add(time.Minute))
			return err
		}
		if !shouldApply {
			return nil
		}
		if err := uc.repo.ApplyApproved(ctx, job, requestID, result, now); err != nil {
			_ = uc.repo.FailInbox(ctx, job, "钉钉审批已通过但本地解锁失败", now.Add(time.Minute))
			return err
		}
		return nil
	default:
		return uc.repo.FailInbox(ctx, job, "钉钉审批权威状态无法识别", now.Add(time.Minute))
	}
}
