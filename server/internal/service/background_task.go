package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	taskv1 "github.com/roncin/roncin-go-admin/server/api/task/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
)

// BackgroundTaskService 后台任务追踪服务，只做 DTO 转换、边界校验和用例调用。
type BackgroundTaskService struct {
	taskv1.UnimplementedBackgroundTaskServiceServer
	usecase *biz.BackgroundTaskUsecase
}

func NewBackgroundTaskService(usecase *biz.BackgroundTaskUsecase) *BackgroundTaskService {
	return &BackgroundTaskService{usecase: usecase}
}

func (s *BackgroundTaskService) ListBackgroundTasks(ctx context.Context, request *taskv1.ListBackgroundTasksRequest) (*taskv1.ListBackgroundTasksResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrBackgroundTaskInvalidArgument)
	if err != nil {
		return nil, err
	}
	options := biz.BackgroundTaskListOptions{Page: page, PageSize: pageSize}
	if request.GetStatus() != taskv1.BackgroundTaskStatus_BACKGROUND_TASK_STATUS_UNSPECIFIED {
		status, err := backgroundTaskStatusFromAPI(request.GetStatus())
		if err != nil {
			return nil, err
		}
		options.Status = &status
	}
	if request.GetPhase() != taskv1.BackgroundTaskPhase_BACKGROUND_TASK_PHASE_UNSPECIFIED {
		phase, err := backgroundTaskPhaseFromAPI(request.GetPhase())
		if err != nil {
			return nil, err
		}
		options.Phase = &phase
	}
	if request.GetKind() != taskv1.BackgroundTaskKind_BACKGROUND_TASK_KIND_UNSPECIFIED {
		kind, err := backgroundTaskKindFromAPI(request.GetKind())
		if err != nil {
			return nil, err
		}
		options.Kind = &kind
	}
	if request.GetStartTime() != "" {
		start, err := time.Parse(time.RFC3339, request.GetStartTime())
		if err != nil {
			return nil, biz.ErrBackgroundTaskInvalidArgument
		}
		options.StartTime = &start
	}
	if request.GetEndTime() != "" {
		end, err := time.Parse(time.RFC3339, request.GetEndTime())
		if err != nil {
			return nil, biz.ErrBackgroundTaskInvalidArgument
		}
		options.EndTime = &end
	}
	result, err := s.usecase.List(ctx, principal.Organization.ID, options)
	if err != nil {
		return nil, err
	}
	data := make([]*taskv1.BackgroundTask, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, backgroundTaskToAPI(item))
	}
	return &taskv1.ListBackgroundTasksResponse{
		Success:  true,
		Code:     0,
		Message:  "OK",
		Data:     data,
		Total:    int32(result.Total),
		Page:     int32(result.Page),
		PageSize: int32(result.PageSize),
		TraceId:  requestmeta.TraceID(ctx),
	}, nil
}

func (s *BackgroundTaskService) GetBackgroundTask(ctx context.Context, request *taskv1.GetBackgroundTaskRequest) (*taskv1.GetBackgroundTaskResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrBackgroundTaskInvalidArgument
	}
	task, err := s.usecase.Get(ctx, principal.Organization.ID, id)
	if err != nil {
		return nil, err
	}
	return &taskv1.GetBackgroundTaskResponse{Success: true, Code: 0, Message: "OK", Data: backgroundTaskToAPI(task), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *BackgroundTaskService) RequeueBackgroundTask(ctx context.Context, request *taskv1.RequeueBackgroundTaskRequest) (*taskv1.RequeueBackgroundTaskResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrBackgroundTaskInvalidArgument
	}
	task, err := s.usecase.Requeue(ctx, principal.Organization.ID, principal.UserID, id)
	if err != nil {
		return nil, err
	}
	return &taskv1.RequeueBackgroundTaskResponse{Success: true, Code: 0, Message: "OK", Data: backgroundTaskToAPI(task), TraceId: requestmeta.TraceID(ctx)}, nil
}

// backgroundTaskToAPI 刻意不暴露租约令牌与租约到期时间，它们是内部执行凭据。
func backgroundTaskToAPI(value *biz.BackgroundTask) *taskv1.BackgroundTask {
	var recipientUserID *string
	if value.RecipientUserID != nil {
		id := value.RecipientUserID.String()
		recipientUserID = &id
	}
	return &taskv1.BackgroundTask{
		Id:                   value.ID.String(),
		Kind:                 backgroundTaskKindToAPI(value.Kind),
		IdempotencyKey:       value.IdempotencyKey,
		Status:               backgroundTaskStatusToAPI(value.Status),
		Attempts:             int32(value.Attempts),
		MaxAttempts:          int32(value.MaxAttempts),
		NextRunAt:            value.NextRunAt.UTC().Format(time.RFC3339),
		LastError:            value.LastError,
		CreatedAt:            value.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:            value.UpdatedAt.UTC().Format(time.RFC3339),
		RecipientDisplayName: value.RecipientDisplayName,
		RecipientUserId:      recipientUserID,
	}
}

func backgroundTaskKindFromAPI(kind taskv1.BackgroundTaskKind) (biz.BackgroundTaskKind, error) {
	switch kind {
	case taskv1.BackgroundTaskKind_BACKGROUND_TASK_KIND_MASTER_DATA_IMPORT:
		return biz.BackgroundTaskKindMasterDataImport, nil
	case taskv1.BackgroundTaskKind_BACKGROUND_TASK_KIND_UNLOCODE_IMPORT:
		return biz.BackgroundTaskKindUnlocodeImport, nil
	case taskv1.BackgroundTaskKind_BACKGROUND_TASK_KIND_ORDER_REMINDER:
		return biz.BackgroundTaskKindOrderReminder, nil
	case taskv1.BackgroundTaskKind_BACKGROUND_TASK_KIND_INTEGRATION:
		return biz.BackgroundTaskKindIntegration, nil
	case taskv1.BackgroundTaskKind_BACKGROUND_TASK_KIND_DINGTALK_NOTIFICATION:
		return biz.BackgroundTaskKindDingTalkNotice, nil
	default:
		return "", biz.ErrBackgroundTaskInvalidArgument
	}
}

func backgroundTaskKindToAPI(kind biz.BackgroundTaskKind) taskv1.BackgroundTaskKind {
	switch kind {
	case biz.BackgroundTaskKindMasterDataImport:
		return taskv1.BackgroundTaskKind_BACKGROUND_TASK_KIND_MASTER_DATA_IMPORT
	case biz.BackgroundTaskKindUnlocodeImport:
		return taskv1.BackgroundTaskKind_BACKGROUND_TASK_KIND_UNLOCODE_IMPORT
	case biz.BackgroundTaskKindOrderReminder:
		return taskv1.BackgroundTaskKind_BACKGROUND_TASK_KIND_ORDER_REMINDER
	case biz.BackgroundTaskKindIntegration:
		return taskv1.BackgroundTaskKind_BACKGROUND_TASK_KIND_INTEGRATION
	case biz.BackgroundTaskKindDingTalkNotice:
		return taskv1.BackgroundTaskKind_BACKGROUND_TASK_KIND_DINGTALK_NOTIFICATION
	default:
		return taskv1.BackgroundTaskKind_BACKGROUND_TASK_KIND_UNSPECIFIED
	}
}

func backgroundTaskStatusFromAPI(status taskv1.BackgroundTaskStatus) (biz.BackgroundTaskStatus, error) {
	switch status {
	case taskv1.BackgroundTaskStatus_BACKGROUND_TASK_STATUS_PENDING:
		return biz.BackgroundTaskStatusPending, nil
	case taskv1.BackgroundTaskStatus_BACKGROUND_TASK_STATUS_RUNNING:
		return biz.BackgroundTaskStatusRunning, nil
	case taskv1.BackgroundTaskStatus_BACKGROUND_TASK_STATUS_SUCCEEDED:
		return biz.BackgroundTaskStatusSucceeded, nil
	case taskv1.BackgroundTaskStatus_BACKGROUND_TASK_STATUS_FAILED:
		return biz.BackgroundTaskStatusFailed, nil
	case taskv1.BackgroundTaskStatus_BACKGROUND_TASK_STATUS_DEAD_LETTER:
		return biz.BackgroundTaskStatusDeadLetter, nil
	default:
		return "", biz.ErrBackgroundTaskInvalidArgument
	}
}

func backgroundTaskStatusToAPI(status biz.BackgroundTaskStatus) taskv1.BackgroundTaskStatus {
	switch status {
	case biz.BackgroundTaskStatusPending:
		return taskv1.BackgroundTaskStatus_BACKGROUND_TASK_STATUS_PENDING
	case biz.BackgroundTaskStatusRunning:
		return taskv1.BackgroundTaskStatus_BACKGROUND_TASK_STATUS_RUNNING
	case biz.BackgroundTaskStatusSucceeded:
		return taskv1.BackgroundTaskStatus_BACKGROUND_TASK_STATUS_SUCCEEDED
	case biz.BackgroundTaskStatusFailed:
		return taskv1.BackgroundTaskStatus_BACKGROUND_TASK_STATUS_FAILED
	case biz.BackgroundTaskStatusDeadLetter:
		return taskv1.BackgroundTaskStatus_BACKGROUND_TASK_STATUS_DEAD_LETTER
	default:
		return taskv1.BackgroundTaskStatus_BACKGROUND_TASK_STATUS_UNSPECIFIED
	}
}

func backgroundTaskPhaseFromAPI(phase taskv1.BackgroundTaskPhase) (biz.BackgroundTaskPhase, error) {
	switch phase {
	case taskv1.BackgroundTaskPhase_BACKGROUND_TASK_PHASE_ACTIVE:
		return biz.BackgroundTaskPhaseActive, nil
	case taskv1.BackgroundTaskPhase_BACKGROUND_TASK_PHASE_HISTORY:
		return biz.BackgroundTaskPhaseHistory, nil
	default:
		return "", biz.ErrBackgroundTaskInvalidArgument
	}
}

var _ taskv1.BackgroundTaskServiceServer = (*BackgroundTaskService)(nil)
