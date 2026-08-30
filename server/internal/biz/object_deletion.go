package biz

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// ObjectDeletion 是对象删除 Worker 执行一次删除所需的最小业务快照。
type ObjectDeletion struct {
	Task      *BackgroundTask
	ObjectKey string
}

type ObjectDeletionRepo interface {
	FindByTaskID(context.Context, uuid.UUID) (*ObjectDeletion, error)
}

type ObjectDeletionUsecase struct {
	tasks   *BackgroundTaskUsecase
	repo    ObjectDeletionRepo
	storage EnterpriseImageStorage
	now     func() time.Time
}

func NewObjectDeletionUsecase(tasks *BackgroundTaskUsecase, repo ObjectDeletionRepo, storage EnterpriseImageStorage) *ObjectDeletionUsecase {
	return &ObjectDeletionUsecase{tasks: tasks, repo: repo, storage: storage, now: time.Now}
}

func (uc *ObjectDeletionUsecase) Enabled() bool {
	return uc != nil && uc.storage != nil && uc.storage.Enabled()
}

// ProcessNext 领取并处理一条对象存储删除任务；删除失败会记录重试状态后返回错误。
func (uc *ObjectDeletionUsecase) ProcessNext(ctx context.Context, leaseDuration time.Duration) error {
	task, err := uc.tasks.ClaimAny(ctx, []BackgroundTaskKind{BackgroundTaskKindObjectStorageDelete}, leaseDuration)
	if err != nil {
		return err
	}
	if task.LeaseToken == nil {
		return uc.fail(ctx, task, "对象删除任务缺少租约令牌")
	}
	deletion, err := uc.repo.FindByTaskID(ctx, task.ID)
	if err != nil {
		return uc.fail(ctx, task, fmt.Sprintf("读取对象删除明细失败: %v", err))
	}
	deletion.Task = task
	if strings.TrimSpace(deletion.ObjectKey) == "" {
		return uc.fail(ctx, task, "对象删除明细不完整")
	}
	if err := uc.storage.Delete(ctx, task.OrganizationID, deletion.ObjectKey); err != nil {
		return uc.fail(ctx, task, fmt.Sprintf("删除对象存储文件失败: %v", err))
	}
	_, err = uc.tasks.Complete(ctx, task.OrganizationID, task.ID, *task.LeaseToken)
	return err
}

func (uc *ObjectDeletionUsecase) fail(ctx context.Context, task *BackgroundTask, message string) error {
	message = strings.TrimSpace(message)
	if utf8.RuneCountInString(message) > 2000 {
		message = string([]rune(message)[:2000])
	}
	if task == nil || task.LeaseToken == nil {
		return fmt.Errorf("%s", message)
	}
	backoff := 30 * time.Second * time.Duration(1<<min(task.Attempts, 5))
	if _, err := uc.tasks.Fail(ctx, task.OrganizationID, task.ID, *task.LeaseToken, message, uc.now().Add(backoff)); err != nil {
		return fmt.Errorf("%s；记录失败状态失败: %w", message, err)
	}
	return fmt.Errorf("%s", message)
}
