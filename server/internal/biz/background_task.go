package biz

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrBackgroundTaskNotFound        = errors.NotFound("BACKGROUND_TASK_NOT_FOUND", "后台任务不存在")
	ErrBackgroundTaskInvalidArgument = errors.BadRequest("BACKGROUND_TASK_INVALID_ARGUMENT", "后台任务参数不合法")
	ErrBackgroundTaskExists          = errors.Conflict("BACKGROUND_TASK_EXISTS", "后台任务已存在")
	ErrBackgroundTaskLeaseMismatch   = errors.Conflict("BACKGROUND_TASK_LEASE_MISMATCH", "后台任务租约不匹配或已失效")
	ErrBackgroundTaskNoTask          = errors.NotFound("BACKGROUND_TASK_NO_TASK", "没有可执行的后台任务")
	ErrBackgroundTaskInvalidStatus   = errors.BadRequest("BACKGROUND_TASK_INVALID_STATUS", "后台任务状态不合法")
)

type BackgroundTaskKind string

const (
	BackgroundTaskKindMasterDataImport BackgroundTaskKind = "MASTER_DATA_IMPORT"
	BackgroundTaskKindUnlocodeImport   BackgroundTaskKind = "UNLOCODE_IMPORT"
	BackgroundTaskKindOrderReminder    BackgroundTaskKind = "ORDER_REMINDER"
	BackgroundTaskKindIntegration      BackgroundTaskKind = "INTEGRATION"
)

func (k BackgroundTaskKind) Valid() bool {
	switch k {
	case BackgroundTaskKindMasterDataImport,
		BackgroundTaskKindUnlocodeImport,
		BackgroundTaskKindOrderReminder,
		BackgroundTaskKindIntegration:
		return true
	default:
		return false
	}
}

type BackgroundTaskStatus string

const (
	BackgroundTaskStatusPending    BackgroundTaskStatus = "PENDING"
	BackgroundTaskStatusRunning    BackgroundTaskStatus = "RUNNING"
	BackgroundTaskStatusSucceeded  BackgroundTaskStatus = "SUCCEEDED"
	BackgroundTaskStatusFailed     BackgroundTaskStatus = "FAILED"
	BackgroundTaskStatusDeadLetter BackgroundTaskStatus = "DEAD_LETTER"
)

func (s BackgroundTaskStatus) Valid() bool {
	switch s {
	case BackgroundTaskStatusPending,
		BackgroundTaskStatusRunning,
		BackgroundTaskStatusSucceeded,
		BackgroundTaskStatusFailed,
		BackgroundTaskStatusDeadLetter:
		return true
	default:
		return false
	}
}

type BackgroundTask struct {
	ID             uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
	OrganizationID uuid.UUID
	Kind           BackgroundTaskKind
	IdempotencyKey string
	Status         BackgroundTaskStatus
	Attempts       int
	MaxAttempts    int
	NextRunAt      time.Time
	LeaseToken     *string
	LeaseExpiresAt *time.Time
	LastError      *string
}

type BackgroundTaskRepo interface {
	Enqueue(context.Context, uuid.UUID, *BackgroundTask) (*BackgroundTask, error)
	Claim(context.Context, uuid.UUID, []BackgroundTaskKind, time.Duration, time.Time) (*BackgroundTask, error)
	Complete(context.Context, uuid.UUID, uuid.UUID, string) (*BackgroundTask, error)
	Fail(context.Context, uuid.UUID, uuid.UUID, string, string, time.Time) (*BackgroundTask, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (*BackgroundTask, error)
}

type BackgroundTaskUsecase struct {
	repo BackgroundTaskRepo
	now  func() time.Time
}

func NewBackgroundTaskUsecase(repo BackgroundTaskRepo) *BackgroundTaskUsecase {
	return &BackgroundTaskUsecase{repo: repo, now: time.Now}
}

func (uc *BackgroundTaskUsecase) Enqueue(ctx context.Context, organizationID uuid.UUID, input *BackgroundTask) (*BackgroundTask, error) {
	if organizationID == uuid.Nil || input == nil {
		return nil, ErrBackgroundTaskInvalidArgument
	}
	if !input.Kind.Valid() {
		return nil, ErrBackgroundTaskInvalidArgument
	}
	idempotencyKey := strings.TrimSpace(input.IdempotencyKey)
	if idempotencyKey == "" || utf8.RuneCountInString(idempotencyKey) > 128 {
		return nil, ErrBackgroundTaskInvalidArgument
	}
	maxAttempts := input.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 3
	}
	if maxAttempts < 1 || maxAttempts > 10 {
		return nil, ErrBackgroundTaskInvalidArgument
	}
	nextRunAt := input.NextRunAt
	if nextRunAt.IsZero() {
		nextRunAt = uc.now()
	}

	task := &BackgroundTask{
		OrganizationID: organizationID,
		Kind:           input.Kind,
		IdempotencyKey: idempotencyKey,
		Status:         BackgroundTaskStatusPending,
		Attempts:       0,
		MaxAttempts:    maxAttempts,
		NextRunAt:      nextRunAt,
	}
	return uc.repo.Enqueue(ctx, organizationID, task)
}

func (uc *BackgroundTaskUsecase) Claim(ctx context.Context, organizationID uuid.UUID, kinds []BackgroundTaskKind, leaseDuration time.Duration) (*BackgroundTask, error) {
	if organizationID == uuid.Nil || leaseDuration <= 0 {
		return nil, ErrBackgroundTaskInvalidArgument
	}
	for _, kind := range kinds {
		if !kind.Valid() {
			return nil, ErrBackgroundTaskInvalidArgument
		}
	}
	now := uc.now()
	return uc.repo.Claim(ctx, organizationID, kinds, leaseDuration, now)
}

func (uc *BackgroundTaskUsecase) Complete(ctx context.Context, organizationID, id uuid.UUID, leaseToken string) (*BackgroundTask, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrBackgroundTaskInvalidArgument
	}
	leaseToken = strings.TrimSpace(leaseToken)
	if leaseToken == "" || utf8.RuneCountInString(leaseToken) > 128 {
		return nil, ErrBackgroundTaskInvalidArgument
	}
	return uc.repo.Complete(ctx, organizationID, id, leaseToken)
}

func (uc *BackgroundTaskUsecase) Fail(ctx context.Context, organizationID, id uuid.UUID, leaseToken, errorMessage string, nextRunAt time.Time) (*BackgroundTask, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrBackgroundTaskInvalidArgument
	}
	leaseToken = strings.TrimSpace(leaseToken)
	if leaseToken == "" || utf8.RuneCountInString(leaseToken) > 128 {
		return nil, ErrBackgroundTaskInvalidArgument
	}
	errorMessage = strings.TrimSpace(errorMessage)
	if utf8.RuneCountInString(errorMessage) > 2000 {
		return nil, ErrBackgroundTaskInvalidArgument
	}
	if nextRunAt.IsZero() {
		return nil, ErrBackgroundTaskInvalidArgument
	}
	return uc.repo.Fail(ctx, organizationID, id, leaseToken, errorMessage, nextRunAt)
}

func (uc *BackgroundTaskUsecase) Get(ctx context.Context, organizationID, id uuid.UUID) (*BackgroundTask, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrBackgroundTaskInvalidArgument
	}
	return uc.repo.Get(ctx, organizationID, id)
}

var _ BackgroundTaskRepo = (BackgroundTaskRepo)(nil)
