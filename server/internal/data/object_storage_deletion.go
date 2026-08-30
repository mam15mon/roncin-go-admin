package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	backgroundtaskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	objectdeletionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/objectstoragedeletion"
)

type objectDeletionRepo struct{ data *Data }

func NewObjectDeletionRepo(data *Data) biz.ObjectDeletionRepo { return &objectDeletionRepo{data: data} }

func (r *objectDeletionRepo) FindByTaskID(ctx context.Context, taskID uuid.UUID) (*biz.ObjectDeletion, error) {
	item, err := r.data.db.ObjectStorageDeletion.Query().
		Where(objectdeletionent.BackgroundTaskIDEQ(taskID)).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrBackgroundTaskNotFound, nil)
	}
	return &biz.ObjectDeletion{ObjectKey: item.ObjectKey}, nil
}

// enqueueObjectStorageDeletion 在业务删除事务内登记类型化对象删除任务。
// 对象键按上传规则全局唯一，直接作为幂等键，防止同一对象重复登记。
func enqueueObjectStorageDeletion(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, objectKey string) error {
	now := time.Now()
	taskID := uuid.Must(uuid.NewV7())
	if _, err := tx.BackgroundTask.Create().
		SetID(taskID).
		SetOrganizationID(organizationID).
		SetKind(backgroundtaskent.KindOBJECT_STORAGE_DELETION).
		SetIdempotencyKey(objectKey).
		SetStatus(backgroundtaskent.StatusPENDING).
		SetAttempts(0).
		SetMaxAttempts(5).
		SetNextRunAt(now).
		Save(ctx); err != nil {
		return err
	}
	_, err := tx.ObjectStorageDeletion.Create().
		SetID(uuid.Must(uuid.NewV7())).
		SetBackgroundTaskID(taskID).
		SetObjectKey(objectKey).
		Save(ctx)
	return err
}
